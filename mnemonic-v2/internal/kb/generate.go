package kb

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// generate.go — the pipeline (003 §1, 011 deliverable 7): resolve → graduate
// → target → synthesize → reindex → no-op check. The graduate-rule is
// deterministic (importance gate + type map); synthesis is the LLM seam with
// an append-only degraded floor (INV-3). A clean incremental run is a PROVEN
// no-op: all four layers (003 §3) must pass — processed-set clean, read-only
// pin re-check, baseline coverage, double source-fingerprint re-check.

// Fact is the minimal source-fact record the pipeline consumes (a mapping of
// the memory SaveEvent + its stored row). Keepers of the memory bus map their
// SaveEvent into this.
type Fact struct {
	SyncID           string
	Type             string
	TopicKey         string
	Entity           string
	Importance       float64
	Content          string
	NormalizedHash   string
	SupersededSyncID string
}

// AllFacts reads the project's live facts (CLI-driven runs).
func AllFacts(ctx context.Context, st store.Store, project string) ([]Fact, error) {
	rows, err := st.Query(ctx,
		`SELECT sync_id, type, COALESCE(topic_key, ''), COALESCE(content, ''),
		        COALESCE(normalized_hash, ''), importance, COALESCE(superseded_by_sync_id, '')
		 FROM observations WHERE project = ? AND deleted_at IS NULL ORDER BY sync_id`, project)
	if err != nil {
		return nil, fmt.Errorf("kb: facts: %w", err)
	}
	out := make([]Fact, 0, len(rows))
	for _, r := range rows {
		f := Fact{
			SyncID:           s0(r, 0),
			Type:             s0(r, 1),
			TopicKey:         s0(r, 2),
			Content:          s0(r, 3),
			NormalizedHash:   s0(r, 4),
			SupersededSyncID: s0(r, 6),
		}
		if v, ok := r[5].(float64); ok {
			f.Importance = v
		}
		out = append(out, f)
	}
	return out, nil
}

// Synthesizer is the LLM merge seam (003 §4). nil = absent, error = failing —
// both fall to the append-only floor for that page (INV-3).
type Synthesizer interface {
	Synthesize(ctx context.Context, in SynthInput) (string, error)
}

// SynthInput is one page merge request.
type SynthInput struct {
	PageRelPath string
	Page        string // the current full page bytes (frontmatter + body)
	Fact        Fact
	Relations   []string // judged relations to weigh (conflict/supersede)
	CodeSpans   []string // live grounded snippets
}

// GraduateDecision is the deterministic graduate-rule output (003 §2).
type GraduateDecision struct {
	Graduate bool
	Type     string // the target page-type folder
	Reason   string
}

// typeMap maps a memory fact type to its page-type folder (003 §2).
var typeMap = map[string]string{
	"decision":     "decisions",
	"architecture": "architecture",
	"bugfix":       "gotchas",
	"convention":   "invariants",
	"config":       "invariants",
	"discovery":    "gotchas",
	"learning":     "glossary",
}

// importanceFloor is the graduate-rule gate (003 §8: the one tunable; the
// default lives here until 008-transport config lands).
const importanceFloor = 0.5

// Graduate runs the deterministic graduate-rule for one fact (003 §2):
// importance gate + type map. Novelty (update vs create) is resolved by the
// targeting step, not here.
func Graduate(f Fact) GraduateDecision {
	d := GraduateDecision{Reason: fmt.Sprintf("type %q not page-mapped", f.Type)}
	folder, known := typeMap[f.Type]
	if !known {
		return d
	}
	if f.Importance < importanceFloor {
		return GraduateDecision{Reason: fmt.Sprintf("importance %.2f below floor %.2f", f.Importance, importanceFloor)}
	}
	d.Graduate, d.Type = true, folder
	d.Reason = "graduated"
	return d
}

// TargetPages returns the existing pages a fact affects (003 §3): pages whose
// frontmatter sources: cite the fact's sync_id (or the fact it superseded),
// or whose title/aliases/slug match the fact's entity or topic_key.
func TargetPages(ctx context.Context, st store.Store, project string, f Fact) ([]string, error) {
	rows, err := st.Query(ctx,
		`SELECT rel_path, body FROM kb_pages WHERE project = ? AND deleted_at IS NULL`, project)
	if err != nil {
		return nil, fmt.Errorf("kb: target: %w", err)
	}
	wantIDs := map[string]bool{f.SyncID: true}
	if f.SupersededSyncID != "" {
		wantIDs[f.SupersededSyncID] = true
	}
	entity := strings.ToLower(strings.TrimSpace(f.Entity))
	topic := slugOf(f.TopicKey)

	seen := map[string]bool{}
	var out []string
	for _, r := range rows {
		rel, body := s0(r, 0), s0(r, 1)
		fm, _, _ := SplitFrontmatter(body)

		cited := false
		for _, s := range fm.Sources {
			if s.Inline != "" && strings.HasPrefix(s.Inline, "memory:") && wantIDs[s.MemorySyncID()] {
				cited = true
				break
			}
		}
		match := cited
		if !match && entity != "" {
			if strings.EqualFold(fm.Title, f.Entity) || aliasMatch(fm.Aliases, entity) ||
				strings.Contains(slugOf(rel), entity) {
				match = true
			}
		}
		if !match && topic != "" && slugOf(rel) == topic {
			match = true
		}
		if match && !seen[rel] {
			seen[rel] = true
			out = append(out, rel)
		}
	}
	sort.Strings(out)
	return out, nil
}

func aliasMatch(aliases []string, entity string) bool {
	for _, a := range aliases {
		if strings.EqualFold(strings.TrimSpace(a), entity) {
			return true
		}
	}
	return false
}

// GenerateResult is the pipeline outcome (003 §1 step 6).
type GenerateResult struct {
	Noop     bool
	Changed  int
	Degraded bool
	RelPaths []string // pages written
	RunID    string
	Reason   string
}

// Generate runs the pipeline for the given facts (003 §1). full=true re-derives
// the vault from all facts (ignoring the processed set); the default run
// honors it (layer 1). A clean run returns a proven no-op (changed 0, Noop
// true) — the four layers of 003 §3 all passed.
func Generate(ctx context.Context, st store.Store, root, project string, full bool, syn Synthesizer, facts []Fact) (*GenerateResult, error) {
	fp, err := SourceFingerprint(ctx, st, project, root)
	if err != nil {
		return nil, err
	}
	mode := "update"
	if full {
		mode = "full"
	}
	rs, err := Begin(ctx, st, root, project, mode, fp)
	if err != nil {
		return nil, err
	}

	// Layer 4, first re-take: the run's own metadata writes (the Begin
	// projection) must not have perturbed the source identity.
	fpRe, err := SourceFingerprint(ctx, st, project, root)
	if err != nil {
		return nil, err
	}
	fingerprintStable := fpRe == fp

	// Resolve targets: per graduating fact, the affected page(s) or a new page.
	targets, _, err := planTargets(ctx, st, project, facts)
	if err != nil {
		return nil, err
	}

	// Layer 1: processed-set clean (incremental only — a full run re-derives).
	layer1 := true
	if !full {
		for _, f := range facts {
			if rs.Processed[f.SyncID] != f.NormalizedHash {
				layer1 = false
				break
			}
		}
	}

	// Layer 2: read-only pin re-check on the affected pages (no verified
	// stamp — that is kb_validate's job, 006 §4).
	layer2 := true
	for _, rel := range targets {
		body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			layer2 = false
			break
		}
		fm, _, _ := SplitFrontmatter(string(body))
		if !IsGrounded(fm.Type) {
			continue
		}
		pc, err := ValidatePage(ctx, st, root, rel, fm)
		if err != nil {
			layer2 = false
			break
		}
		if pc.Status != "fresh" {
			layer2 = false
			break
		}
	}

	// Layer 3: baseline coverage — every affected grounded page has a baseline
	// current for this fingerprint (a page with no/stale baseline is forced to
	// re-derive, never a no-op).
	layer3 := true
	baselines, err := Baselines(ctx, st, project)
	if err != nil {
		return nil, err
	}
	for _, rel := range targets {
		body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			layer3 = false
			break
		}
		fm, _, _ := SplitFrontmatter(string(body))
		if !IsGrounded(fm.Type) {
			continue
		}
		b, ok := baselines[rel]
		if !ok || !b.Current(fp, hashBody(string(body))) {
			layer3 = false
			break
		}
	}

	// Proven no-op: all four layers hold.
	if layer1 && layer2 && layer3 && fingerprintStable {
		if len(facts) > 0 {
			for _, f := range facts {
				rs.Processed[f.SyncID] = f.NormalizedHash
			}
			if err := saveRun(ctx, st, rs); err != nil {
				return nil, err
			}
		}
		if err := Finish(ctx, st, root, project, true, full); err != nil {
			return nil, err
		}
		return &GenerateResult{Noop: true, RunID: rs.RunID, Reason: "proven no-op: 4 layers passed"}, nil
	}

	// Synthesis path: plan the jobs, then work them.
	jobs := make([]PageJob, 0, len(targets))
	for _, rel := range targets {
		jobs = append(jobs, PageJob{RelPath: rel, Title: rel})
	}
	if len(jobs) > 0 {
		if err := SetPlan(ctx, st, root, project, jobs); err != nil {
			return nil, err
		}
	}

	now := time.Now().UTC()
	res := &GenerateResult{RunID: rs.RunID}
	for _, job := range jobs {
		written, degraded, changed, err := generatePage(ctx, st, root, project, job.RelPath, fp, now, syn, facts)
		if err != nil {
			return nil, err
		}
		res.Degraded = res.Degraded || degraded
		if written {
			res.RelPaths = append(res.RelPaths, job.RelPath)
		}
		if changed {
			res.Changed++
		}
		if err := MarkJob(ctx, st, root, project, job.RelPath, JobComplete, "kb-generate"); err != nil {
			return nil, err
		}
	}

	// REINDEX: advance the derived index over the written pages.
	if _, err := Index(ctx, st, root, project); err != nil {
		return nil, err
	}

	// Record processed (hash-keyed; upserts with new content re-graduate).
	rs2, err := LoadRun(ctx, st, project)
	if err != nil {
		return nil, err
	}
	for _, f := range facts {
		rs2.Processed[f.SyncID] = f.NormalizedHash
	}
	if err := saveRun(ctx, st, rs2); err != nil {
		return nil, err
	}

	if err := Finish(ctx, st, root, project, true, full); err != nil {
		return nil, err
	}
	return res, nil
}

// planTargets resolves the page-job set: per graduating fact, the affected
// existing page(s) (003 §3 targeting) or a NEW page (novelty: no existing
// target → create <type-folder>/<slug>.md). Facts already processed at their
// current hash stay candidates — a stale pin or missing baseline (layers
// 2/3) still forces a re-derive of their pages.
func planTargets(ctx context.Context, st store.Store, project string, facts []Fact) (targets []string, newPages int, err error) {
	seen := map[string]bool{}
	add := func(rel string) {
		if !seen[rel] {
			seen[rel] = true
			targets = append(targets, rel)
		}
	}
	for _, f := range facts {
		d := Graduate(f)
		if !d.Graduate {
			continue
		}
		rel, err := TargetPages(ctx, st, project, f)
		if err != nil {
			return nil, 0, err
		}
		if len(rel) > 0 {
			for _, r := range rel {
				add(r)
			}
			continue
		}
		// Novelty: no existing page → a new page for this fact's entity.
		slug := slugOf(f.TopicKey)
		if slug == "" {
			slug = slugOf(f.Entity)
		}
		if slug == "" {
			slug = slugOf(f.SyncID)
		}
		add(filepath.ToSlash(d.Type + "/" + slug + ".md"))
		newPages++
	}
	sort.Strings(targets)
	return targets, newPages, nil
}

// generatePage works one planned job: synthesize (LLM or the append-only
// floor), write, pin-check, record the baseline (only when consistent),
// emit the page-write event. Returns (written, degraded, changed, err).
func generatePage(ctx context.Context, st store.Store, root, project, relPath, fp string, now time.Time, syn Synthesizer, facts []Fact) (bool, bool, bool, error) {
	fullPath := filepath.Join(root, filepath.FromSlash(relPath))
	oldBody, _ := os.ReadFile(fullPath)
	page := string(oldBody)

	// Fresh page (a new-page job): seed the skeleton.
	if page == "" {
		fm := Frontmatter{
			Title:   pageTitleFromSlug(relPath),
			Type:    shortTypeFor(filepath.Base(filepath.Dir(relPath))),
			Project: project,
			Status:  "draft",
		}
		page = fm.Render("\n")
	}

	// Synthesize per fact.
	degraded := false
	changed := false
	for _, f := range facts {
		if !Graduate(f).Graduate {
			continue
		}
		out, deg, err := synthesizePage(ctx, syn, SynthInput{PageRelPath: relPath, Page: page, Fact: f}, now)
		if err != nil {
			return false, false, false, err
		}
		degraded = degraded || deg
		if out != page {
			page = out
			changed = true
		}
	}

	// NO-OP CHECK (003 §1 step 6): content unchanged → no write.
	if !changed {
		return false, degraded, false, nil
	}
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return false, false, false, err
	}
	if err := os.WriteFile(fullPath, []byte(page), 0o644); err != nil {
		return false, false, false, err
	}
	if err := emitPageEvent(ctx, st, project, relPath, hashBody(page)); err != nil {
		return false, false, false, err
	}

	// Pin-check the written page; record a baseline only when mutually
	// consistent (003 §5.2: a page earns a baseline only when it has proven).
	fm, _, _ := SplitFrontmatter(page)
	if IsGrounded(fm.Type) {
		pc, err := ValidatePage(ctx, st, root, relPath, fm)
		if err != nil {
			return true, degraded, true, err
		}
		if pc.Status == "fresh" {
			if err := RecordBaseline(ctx, st, root, project, relPath, fp, hashBody(page), "kb-generate:"+now.Format(time.RFC3339), "kb-generate"); err != nil {
				return true, degraded, true, err
			}
		}
	}
	return true, degraded, true, nil
}

// synthesizePage merges one fact into the page (003 §4): the LLM seam with a
// deterministic output guard; nil/failing/malformed → the append-only floor
// (a dated, cited entry — idempotent per fact: the citation is appended once,
// the entry keyed on the sync_id).
func synthesizePage(ctx context.Context, syn Synthesizer, in SynthInput, now time.Time) (string, bool, error) {
	if syn != nil {
		if out, err := syn.Synthesize(ctx, in); err == nil && pageOutputOK(in.Page, out) {
			return out, false, nil
		}
	}
	// Append-only floor.
	fm, body, _ := SplitFrontmatter(in.Page)
	cited := false
	for _, s := range fm.Sources {
		if s.MemorySyncID() == in.Fact.SyncID {
			cited = true
			break
		}
	}
	// Idempotency: a fact already merged (cited) leaves the page unchanged.
	if cited {
		return in.Page, false, nil
	}
	fm.Sources = append(fm.Sources, Source{Inline: "memory:" + in.Fact.SyncID})
	if fm.Generated != nil {
		fm.Generated.At = now.UTC().Format(time.RFC3339)
	} else {
		fm.Generated = &Stamp{By: "kb-generate", At: now.UTC().Format(time.RFC3339)}
	}
	entry := fmt.Sprintf("## %s — %s\n\n%s\n\n> source: memory:%s\n",
		now.UTC().Format("2006-01-02"), in.Fact.Type, in.Fact.Content, in.Fact.SyncID)
	return fm.Render(body + "\n" + entry), true, nil
}

// pageOutputOK is the deterministic guard (003 §4): frontmatter present, the
// page type preserved, a non-empty body. A malformed LLM output is rejected
// (the page degrades; a corrupt page is never written).
func pageOutputOK(orig, out string) bool {
	if strings.TrimSpace(out) == "" {
		return false
	}
	ofm, _, _ := SplitFrontmatter(orig)
	nfm, nbody, _ := SplitFrontmatter(out)
	if nfm.Type == "" {
		return false
	}
	if !strings.EqualFold(nfm.Type, ofm.Type) {
		return false
	}
	return strings.TrimSpace(nbody) != ""
}

// emitPageEvent appends the INV-5 page-write trail (idempotent per content).
func emitPageEvent(ctx context.Context, st store.Store, project, relPath, bodyHash string) error {
	key := relPath + ":" + bodyHash
	return st.Exec(ctx,
		`INSERT INTO kb_events (project, event_type, subject_kind, rel_path, event_key, occurred_at)
		 VALUES (?, 'page_written', 'kb_page', ?, ?, ?)
		 ON CONFLICT(project, event_key) DO NOTHING`,
		project, relPath, key, time.Now().UTC().Format(time.RFC3339))
}

func shortTypeFor(folder string) string {
	switch folder {
	case "decisions":
		return "Decision"
	case "invariants":
		return "InvariantContract"
	case "workflows":
		return "BusinessWorkflow"
	case "architecture":
		return "Architecture"
	case "gotchas":
		return "Gotcha"
	case "glossary":
		return "GlossaryTerm"
	}
	return "Decision"
}

func pageTitleFromSlug(relPath string) string {
	base := filepath.Base(relPath)
	base = strings.TrimSuffix(base, ".md")
	base = strings.ReplaceAll(base, "-", " ")
	return strings.TrimSpace(base)
}
