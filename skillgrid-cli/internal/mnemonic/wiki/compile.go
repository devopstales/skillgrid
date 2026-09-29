package wiki

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// generatorBy is the `generated.by` actor recorded on every emitted page.
const generatorBy = "skillgrid-wiki"

// manifestFile is the name of the prior-manifest file inside the out dir.
const manifestFile = ".wiki-manifest.json"

// logRel is the bundle-relative path of the compile log.
const logRel = "wiki/log.md"

// CompileInput is the full input to one compile. All paths are absolute (the
// CLI resolves them from the flags); Now is the injected compile clock.
//
// WebRows are the fresh web_cache rows read by the CLI (the wiki package stays
// pure: no database import). The CLI fetches only fresh rows of the indexed
// sources; Compile re-filters them defensively via ParseWebCache.
type CompileInput struct {
	ProjectDir string    // root of the project (where .skillgrid/ lives)
	OutDir     string    // where to write .wiki/ (default: ProjectDir/.wiki)
	Now        time.Time
	WebRows    []WebRow  // fresh web_cache rows (CLI-populated)
	RawRoot    string    // root of user-dropped raw/ (default: ProjectDir/raw)
}

// CompileResult reports what a compile wrote.
type CompileResult struct {
	Written   int      // number of files written
	Unchanged int      // number of files skipped (content-hash match)
	Paths     []string // absolute paths of written files
}

// manifestEntry records the content hash (of the page minus generated.at) and
// the preserved generated.at for one emitted file (design: Determinism).
type manifestEntry struct {
	ContentHash string `json:"content_hash"`
	GeneratedAt string `json:"generated_at"`
}

// manifest is the persisted map of bundle-relative path → manifestEntry.
type manifest struct {
	GeneratedAt string            `json:"generated_at"`
	Files       map[string]string `json:"files"`
}

// priorState is the loaded prior-manifest state plus the prior log content
// (log.md lives in the bundle, so it is read from disk directly).
type priorState struct {
	Hashes      map[string]string           // relPath → content hash (minus generated.at)
	GeneratedAt map[string]manifestGenerated // relPath → preserved generated.at
	Log         string
}

// Compile reads the project's pillar-1 sources (ADRs, State, Terms, Specs,
// Spikes, Architecture), the indexed web_cache rows, and the user-dropped
// raw/ tree; renders every concept to an OKF v0.2 page; and writes the .wiki/
// tree with a content-hash gate: unchanged files are not rewritten and their
// generated.at is preserved (R8.1–R8.3).
func Compile(in CompileInput) (CompileResult, error) {
	if in.ProjectDir == "" {
		return CompileResult{}, fmt.Errorf("wiki: Compile: ProjectDir is required")
	}
	outDir := in.OutDir
	if outDir == "" {
		outDir = filepath.Join(in.ProjectDir, ".wiki")
	}
	rawRoot := in.RawRoot
	if rawRoot == "" {
		rawRoot = filepath.Join(in.ProjectDir, "raw")
	}
	now := in.Now.UTC()
	var result CompileResult

	skillgridDir := filepath.Join(in.ProjectDir, ".skillgrid")

	// 1. Read all sources.
	assumptionsPath := filepath.Join(skillgridDir, "ASSUMPTIONS.md")
	concepts, _, err := ParseAssumptions(assumptionsPath)
	if err != nil {
		return CompileResult{}, fmt.Errorf("wiki: parse ASSUMPTIONS.md: %w", err)
	}
	if sc, _, err := ParseState(filepath.Join(skillgridDir, "state.yaml")); err != nil {
		return CompileResult{}, fmt.Errorf("wiki: parse state.yaml: %w", err)
	} else if sc != nil {
		concepts = append(concepts, *sc)
	}
	concepts = append(concepts, ParseTerms(
		filepath.Join(skillgridDir, "artifacts", "01-business-terms.md"),
		filepath.Join(skillgridDir, "artifacts", "02-technical-terms.md"),
	)...)
	concepts = append(concepts, ParseSpecs(filepath.Join(skillgridDir, "specs"))...)
	concepts = append(concepts, ParseSpikes(filepath.Join(skillgridDir, "spikes"))...)
	if arch := ParseArchitecture(filepath.Join(skillgridDir, "ARCHITECTURE.md")); arch != nil {
		concepts = append(concepts, *arch)
	}
	concepts = append(concepts, ParseWebCache(in.WebRows)...)
	rawConcepts, err := ParseRaw(rawRoot)
	if err != nil {
		return CompileResult{}, fmt.Errorf("wiki: parse raw/: %w", err)
	}
	concepts = append(concepts, rawConcepts...)

	// 2. Stable sort by (TypeDir, Slug) — deterministic order (R8.3).
	sort.Slice(concepts, func(i, j int) bool {
		di, dj := TypeDir(concepts[i].Type), TypeDir(concepts[j].Type)
		if di != dj {
			return di < dj
		}
		return conceptSlug(concepts[i]) < conceptSlug(concepts[j])
	})

	// 4. Load the prior manifest (absent → empty).
	manifestPath := filepath.Join(outDir, manifestFile)
	prior, err := loadManifest(manifestPath)
	if err != nil {
		return CompileResult{}, fmt.Errorf("wiki: load manifest: %w", err)
	}

	// 5. Split web findings (R5.1/R5.2): a row is cited when its url appears
	// as sources[].resource in any other emitted concept; uncited rows are
	// demoted to draft and route to wiki/research/.
	cited := citedResources(concepts)
	splitWebFindings(concepts, cited)

	// 6. Render every concept. The gate hash is the content hash minus
	// generated.at; the preserved generated.at is carried from the prior
	// manifest when the hash is unchanged (R8.2).
	var plans []filePlan
	for _, c := range concepts {
		slug := conceptSlug(c)
		if slug == "" {
			continue
		}
		dir := TypeDir(c.Type)
		if c.Type == "Finding" && c.Status == "draft" && len(c.Sources) > 0 && webSourced(c.Sources) {
			// Uncited web_cache research → wiki/research/ (R5.2).
			dir = "research"
		}
		relPath := filepath.ToSlash(filepath.Join("wiki", dir, slug+".md"))
		h := conceptContentHash(c, now)
		ga := manifestGenerated{At: now.Format(time.RFC3339)}
		if old, ok := prior.Hashes[relPath]; ok && old == h {
			ga = prior.GeneratedAt[relPath] // unchanged: preserve generated.at
		}
		pages, perr := renderConcept(c, ga)
		if perr != nil {
			return CompileResult{}, fmt.Errorf("wiki: render %s: %w", relPath, perr)
		}
		plans = append(plans, filePlan{RelPath: relPath, Content: pages, Hash: h})
	}

	// Special bundle files (AGENTS.md, index.md, log.md).
	agentsContent := RenderAgents()
	indexContent := RenderIndex(concepts)
	plans = append(plans, filePlan{
		RelPath: "AGENTS.md",
		Content: agentsContent,
		Hash:    hashContent(agentsContent),
	})
	plans = append(plans, filePlan{
		RelPath: "wiki/index.md",
		Content: indexContent,
		Hash:    hashContent(indexContent),
	})

	// 7. First pass: decide written vs unchanged for every file (excluding
	// the log, which depends on the set of changed pages).
	type decision struct {
		plan   filePlan
		write  bool
		unchg  bool // unchanged vs prior manifest (content-hash match)
		changed bool // changed vs prior manifest
	}
	var decisions []decision
	changedSet := map[string]bool{}
	for _, p := range plans {
		old, hadOld := prior.Hashes[p.RelPath]
		write := !hadOld || old != p.Hash
		// The log is not in plans; it depends on this changed set.
		if hadOld && write {
			changedSet[p.RelPath] = true
		}
		decisions = append(decisions, decision{
			plan:    p,
			write:   write,
			unchg:   hadOld && old == p.Hash,
			changed: hadOld && old != p.Hash,
		})
	}

	// 8. Log: the bundle always carries a log.md (R: "log.md — change log").
	// On the first compile (no prior log) seed the header so the file exists;
	// an empty change set otherwise leaves the log untouched (no-op compile →
	// no log churn, R8).
	newLog := prior.Log
	if newLog == "" {
		newLog = logFrontmatter + "# Log\n"
	}
	logChanged := false
	if len(changedSet) > 0 {
		var changes []string
		for rel := range changedSet {
			changes = append(changes, "change "+rel)
		}
		sort.Strings(changes)
		newLog = AppendLog(prior.Log, now, changes)
		logChanged = true
	}
	// The log is written when its bytes differ from the prior stored bytes —
	// this also creates the initial "# Log" on the first compile (prior.Log
	// is "" then). A no-op recompile has newLog == prior.Log, so no write.
	logWrite := newLog != prior.Log

	// 9. Write phase: every plan file, then the log when it changed.
	for _, d := range decisions {
		if !d.write {
			if d.unchg {
				result.Unchanged++
			}
			continue
		}
		if err := writeBundleFile(outDir, d.plan.RelPath, d.plan.Content); err != nil {
			return result, err
		}
		result.Written++
		result.Paths = append(result.Paths, filepath.Join(outDir, filepath.FromSlash(d.plan.RelPath)))
	}
	if logWrite {
		if err := writeBundleFile(outDir, logRel, newLog); err != nil {
			return result, err
		}
		result.Written++
		result.Paths = append(result.Paths, filepath.Join(outDir, logRel))
	}

	// 10. Save the manifest (last, so a crash never loses a prior manifest).
	// Rebuild the new hash/generated-at maps from the freshly rendered plans,
	// dropping files that no longer exist and updating the ones we just
	// (re)wrote.
	newHashes := map[string]string{}
	newGA := map[string]manifestGenerated{}
	for _, d := range decisions {
		newHashes[d.plan.RelPath] = d.plan.Hash
		newGA[d.plan.RelPath] = extractGeneratedAt(d.plan.Content)
	}
	if logChanged {
		newHashes[logRel] = hashContent(newLog)
		newGA[logRel] = manifestGenerated{} // the log has no frontmatter
	}
	m := manifest{
		GeneratedAt: now.Format(time.RFC3339),
		Files:       newHashes,
	}
	if err := saveManifest(manifestPath, m, newGA); err != nil {
		return result, err
	}
	return result, nil
}

// filePlan is one rendered file ready to gate and write.
type filePlan struct {
	RelPath string
	Content string
	Hash    string // content hash (concept pages: minus generated.at; special files: full)
}

// renderConcept stamps the generated metadata (generated, stale_after) onto a
// copy of c and emits the page. The generated.at is the preserved value passed
// in (R8.2) — a fresh compile passes the current compile clock.
func renderConcept(c Concept, generatedAt manifestGenerated) (string, error) {
	stamped := c
	now := parseGeneratedAt(generatedAt.At)
	if stamped.Generated.By == "" || stamped.Generated.At.IsZero() {
		stamped.Generated = Verifier{By: generatorBy, At: now}
	}
	if stamped.StaleAfter.IsZero() && (stamped.Type == "Finding" || stamped.Type == "State") {
		// Horizon per ADR-0014 pillar 1: 90 days from generation.
		stamped.StaleAfter = now.Add(90 * 24 * time.Hour)
	}
	return Emit(stamped)
}

// conceptSlug returns the page slug for a concept: its ID, or the slugified
// title when the ID is empty.
func conceptSlug(c Concept) string {
	if c.ID != "" {
		return c.ID
	}
	return Slugify(c.Title)
}

// citedResources collects every sources[].resource referenced by any concept
// except the web-sourced findings themselves. A web_cache row is "cited" (R5.1)
// when its url appears here — i.e. some other emitted page already references
// the same upstream resource.
func citedResources(concepts []Concept) map[string]bool {
	cited := map[string]bool{}
	for _, c := range concepts {
		if c.Type != "Finding" || !webSourced(c.Sources) {
			for _, s := range c.Sources {
				if s.Resource != "" {
					cited[s.Resource] = true
				}
			}
		}
	}
	return cited
}

// splitWebFindings marks web-sourced findings as uncited (status draft → they
// route to wiki/research/) when none of their source resources are cited by
// another page (R5.2). Cited findings keep status stable (R5.1).
func splitWebFindings(concepts []Concept, cited map[string]bool) {
	for i := range concepts {
		c := &concepts[i]
		if c.Type != "Finding" || !webSourced(c.Sources) {
			continue
		}
		webCited := false
		for _, s := range c.Sources {
			if s.Resource != "" && cited[s.Resource] {
				webCited = true
				break
			}
		}
		if !webCited {
			c.Status = "draft"
		}
	}
}

// webSourced reports whether any source carries a web-process actor
// ("process:<source>"), i.e. the concept was produced from a web_cache row.
func webSourced(sources []SourceRef) bool {
	for _, s := range sources {
		if strings.HasPrefix(s.Author, "process:") {
			return true
		}
	}
	return false
}

// conceptContentHash hashes the concept's source-derived output minus every
// time-derived stamp (generated.at and stale_after), so an unchanged concept
// hashes identically across compiles even though its generated.at and
// now-derived stale_after may differ. This is the design's content-hash gate
// (design: Determinism): the hash tracks source content, not the compile clock.
func conceptContentHash(c Concept, now time.Time) string {
	stamped := c
	if stamped.Generated.By == "" || stamped.Generated.At.IsZero() {
		stamped.Generated = Verifier{By: generatorBy, At: now}
	}
	if stamped.StaleAfter.IsZero() && (stamped.Type == "Finding" || stamped.Type == "State") {
		stamped.StaleAfter = now.Add(90 * 24 * time.Hour)
	}
	out, _ := Emit(stamped)
	lines := strings.Split(out, "\n")
	var kept []string
	inGenerated := false
	for _, l := range lines {
		if l == "generated:" {
			inGenerated = true
			continue
		}
		if inGenerated {
			if strings.HasPrefix(l, "  ") || strings.TrimSpace(l) == "" {
				continue
			}
			inGenerated = false
		}
		// stale_after is a top-level "stale_after: <rfc3339>" line (design: it
		// is now + 90d, a time-derived value) — exclude it from the gate hash.
		if strings.HasPrefix(l, "stale_after: ") {
			continue
		}
		kept = append(kept, l)
	}
	return hashContent(strings.Join(kept, "\n"))
}

// manifestGenerated is the per-file generated marker stored in the manifest.
// It serializes to {"generated.at": "<rfc3339>"}: the key is the dotted
// "generated.at" (the exact frontmatter path, so the manifest literally
// contains the page's generated.at), which also makes the round-trip trivial.
type manifestGenerated struct {
	At string `json:"generated.at"`
}

// extractGeneratedAt pulls the generated.at value out of an emitted page's
// frontmatter (zero value when absent).
func extractGeneratedAt(content string) manifestGenerated {
	inFront := false
	for _, l := range strings.Split(content, "\n") {
		if l == "---" {
			if !inFront {
				inFront = true
				continue
			}
			break // closing frontmatter delimiter
		}
		if inFront && strings.HasPrefix(l, "  at: ") {
			return manifestGenerated{At: strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(l), "at: "))}
		}
	}
	return manifestGenerated{}
}

func hashContent(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func parseGeneratedAt(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// writeBundleFile writes content to outDir/relPath, creating parent dirs.
func writeBundleFile(outDir, relPath, content string) error {
	abs := filepath.Join(outDir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	return os.WriteFile(abs, []byte(content), 0o644)
}

func loadManifest(path string) (priorState, error) {
	ps := priorState{
		Hashes:      map[string]string{},
		GeneratedAt: map[string]manifestGenerated{},
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ps, nil
		}
		return ps, err
	}
	var m struct {
		Files map[string]struct {
			ContentHash string             `json:"content_hash"`
			Generated   manifestGenerated  `json:"generated"`
		} `json:"files"`
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return ps, err
	}
	for k, v := range m.Files {
		ps.Hashes[k] = v.ContentHash
		ps.GeneratedAt[k] = v.Generated
	}
	// Prior log content, if any.
	logPath := filepath.Join(filepath.Dir(path), "wiki", "log.md")
	if b, err := os.ReadFile(logPath); err == nil {
		ps.Log = string(b)
	}
	return ps, nil
}

func saveManifest(path string, m manifest, generatedAt map[string]manifestGenerated) error {
	type fileOut struct {
		ContentHash string            `json:"content_hash"`
		Generated   manifestGenerated `json:"generated"`
	}
	out := struct {
		Files map[string]fileOut `json:"files"`
	}{}
	out.Files = map[string]fileOut{}
	for k, v := range m.Files {
		out.Files[k] = fileOut{ContentHash: v, Generated: generatedAt[k]}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

