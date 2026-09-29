package wiki

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Warning is a non-fatal notice raised by a Pillar-1 adapter (e.g. an
// unknown top-level key in state.yaml, R4.5). Warnings are returned by the
// caller and surfaced in compile output; they never abort a parse.
type Warning struct {
	Key     string // the offending key or input
	Message string // human-readable explanation
}

// knownStateKeys are the recognized top-level keys of state.yaml. Any other
// top-level key produces a Warning (R4.5), never an error.
var knownStateKeys = map[string]bool{
	"schema":          true,
	"pipeline":        true,
	"progress":        true,
	"constraints_ref": true,
	"notes":           true,
}

// adrHeadingRE matches "### ADR-NNNN — Title" headings.
var adrHeadingRE = regexp.MustCompile(`^###\s+ADR-(\d+)\s*[—–-]\s*(.+?)\s*$`)

// wikilinkRE matches "[[Target]]" or "[[Target|alias]]" Obsidian links.
var wikilinkRE = regexp.MustCompile(`\[\[([^\]|]+?)(?:\|[^\]]+?)?\]\]`)

// ParseAssumptions parses a .skillgrid/ASSUMPTIONS.md file into ADR and
// Constraint concepts plus Supersedes/Amends edges.
//
// Sections recognized:
//
//   - "### In-force set" — the ADR table. A row's "In force" cell decides the
//     ADR status: yes → stable, no → deprecated. The row's Supersedes and
//     Amends cells produce EXTRACTED edges.
//   - "### ADR-NNNN — Title" entries — the full ADR text (title + body).
//   - "### Locked constraints" — one bullet → one Constraint concept
//     (status stable, no stale_after).
//
// Parsing is tolerant: a missing file or a missing section yields no
// concepts and no error. Only an unreadable existing file is an error.
func ParseAssumptions(path string) ([]Concept, []Edge, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	lines := strings.Split(string(data), "\n")

	var concepts []Concept
	var edges []Edge
	tableStatuses := map[string]string{}

	// In-force table: #, Title, Status, Supersedes, Amends, Date, In force.
	// The table ends at the first non-table, non-blank line after the header.
	if i, ok := findHeading(lines, "### In-force set"); ok {
		i++
		for i < len(lines) {
			line := strings.TrimSpace(lines[i])
			if strings.HasPrefix(line, "###") {
				break
			}
			if line == "" {
				i++
				continue
			}
			cells := splitTableRow(line)
			if len(cells) < 7 {
				i++
				continue
			}
			if cells[0] == "#" || strings.HasPrefix(cells[0], "---") {
				i++
				continue
			}
			num := strings.TrimSpace(cells[0])
			if !isDigits(num) {
				i++
				continue
			}
			id := "adr-" + pad4(num)
			title := strings.TrimSpace(cells[1])
			inForce := strings.HasPrefix(strings.ToLower(cells[6]), "yes")
			status := "stable"
			if !inForce {
				status = "deprecated"
			}
			tableStatuses[id] = status
			concepts = append(concepts, Concept{
				ID:         id,
				Type:       "ADR",
				Title:      title,
				Status:     status,
				Tags:       []string{"adr"},
				Body:       "### ADR-" + pad4(num) + " — " + title,
				SourcePath: path,
			})
			for _, ref := range strings.Split(cells[3], ",") {
				if t := adrRef(ref); t != "" {
					edges = append(edges, Edge{From: id, To: t, Kind: "supersedes", Confidence: ConfidenceExtracted})
				}
			}
			for _, ref := range strings.Split(cells[4], ",") {
				if t := adrRef(ref); t != "" {
					edges = append(edges, Edge{From: id, To: t, Kind: "amends", Confidence: ConfidenceExtracted})
				}
			}
			i++
		}
	}

	// ADR entries: "### ADR-NNNN — Title" followed by the full body text.
	if i, ok := findAnyADRHeading(lines); ok {
		for i < len(lines) {
			line := lines[i]
			if m := adrHeadingRE.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
				num, title := m[1], m[2]
				body := []string{"### ADR-" + pad4(num) + " — " + title, ""}
				j := i + 1
				for j < len(lines) {
					if strings.HasPrefix(strings.TrimSpace(lines[j]), "### ") {
						break
					}
					body = append(body, lines[j])
					j++
				}
				// Drop trailing blank lines.
				for len(body) > 0 && strings.TrimSpace(body[len(body)-1]) == "" {
					body = body[:len(body)-1]
				}
				existing := findConcept(concepts, "adr-"+pad4(num))
				if existing != nil {
					// The in-force table already created this ADR (with its
					// status); enrich it with the full body.
					existing.Body = strings.Join(body, "\n")
					if existing.Title == "" {
						existing.Title = title
					}
				} else if tableStatus, ok := tableStatuses["adr-"+pad4(num)]; ok {
					// No "### ADR-NNNN" entry but present in the in-force
					// table: use the table's status.
					concepts = append(concepts, Concept{
						ID:         "adr-" + pad4(num),
						Type:       "ADR",
						Title:      title,
						Status:     tableStatus,
						Tags:       []string{"adr"},
						Body:       strings.Join(body, "\n"),
						SourcePath: path,
					})
				} else {
					concepts = append(concepts, Concept{
						ID:         "adr-" + pad4(num),
						Type:       "ADR",
						Title:      title,
						Status:     "stable",
						Tags:       []string{"adr"},
						Body:       strings.Join(body, "\n"),
						SourcePath: path,
					})
				}
				i = j
				continue
			}
			i++
		}
	}

	// Locked constraints: top-level bullets inside "### Locked constraints".
	if i, ok := findHeading(lines, "### Locked constraints"); ok {
		i++
		n := 0
		for i < len(lines) {
			line := strings.TrimSpace(lines[i])
			if strings.HasPrefix(line, "###") {
				break
			}
			if !strings.HasPrefix(line, "- ") {
				i++
				continue
			}
			text := strings.TrimSpace(strings.TrimPrefix(line, "- "))
			if text == "" {
				i++
				continue
			}
			n++
			concepts = append(concepts, Concept{
				ID:         fmt.Sprintf("locked-constraint-%d", n),
				Type:       "Constraint",
				Title:      text,
				Status:     "stable",
				Tags:       []string{"locked-constraint"},
				Body:       text,
				SourcePath: path,
			})
			i++
		}
	}

	// R4.1/4.2: every Supersedes/Amends edge must appear as a [[adr-NNNN]]
	// wikilink in BOTH related ADR bodies (superseder links the superseded,
	// superseded links the superseder; amender links the amended and vice
	// versa). Idempotent: an ADR body that already contains the wikilink
	// (written in the source markdown) is left untouched.
	for _, e := range edges {
		switch e.Kind {
		case "supersedes", "amends":
			addWikilink(&concepts, e.From, e.To)
			addWikilink(&concepts, e.To, e.From)
		}
	}

	return concepts, edges, nil
}

// addWikilink appends "[[targetID]]" to the body of the ADR with ID
// fromID, unless the body already contains the wikilink (idempotent) or the
// ADR concept is absent.
func addWikilink(concepts *[]Concept, fromID, targetID string) {
	link := "[[" + targetID + "]]"
	c := findConcept(*concepts, fromID)
	if c == nil || c.Type != "ADR" {
		return
	}
	if strings.Contains(c.Body, link) {
		return
	}
	c.Body = strings.TrimRight(c.Body, "\n") + "\n\n" + link
}

// ParseState parses .skillgrid/state.yaml into one State concept (status
// draft). Unknown top-level keys produce Warnings, not errors (R4.5). A
// missing file yields a nil concept and no error.
func ParseState(path string) (*Concept, []Warning, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	lines := strings.Split(string(data), "\n")

	var warnings []Warning
	var body []string

	i := 0
	for i < len(lines) {
		line := lines[i]
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			body = append(body, line)
			i++
			continue
		}
		if isIndented(line) {
			body = append(body, line)
			i++
			continue
		}
		key, _, ok := splitKeyValue(line)
		if !ok {
			body = append(body, line)
			i++
			continue
		}
		if !knownStateKeys[key] {
			warnings = append(warnings, Warning{Key: key, Message: "unknown top-level key in state.yaml"})
		}
		i++
	}

	bodyText := strings.Join(body, "\n")
	for len(bodyText) > 0 && (bodyText[len(bodyText)-1] == '\n') {
		bodyText = bodyText[:len(bodyText)-1]
	}

	c := &Concept{
		ID:         "project-state",
		Type:       "State",
		Title:      "Project State",
		Status:     "draft",
		Tags:       []string{"state"},
		Body:       bodyText,
		SourcePath: path,
	}
	return c, warnings, nil
}

// ParseTerms parses the glossary markdown files (01-business-terms.md,
// 02-technical-terms.md) into Term concepts, one per table row. Explicit
// "[[…]]" cross-refs stay in the term body as Obsidian wikilinks. A missing
// file yields no concepts and no error.
func ParseTerms(paths ...string) []Concept {
	var concepts []Concept
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			continue // tolerant: unreadable → no concepts
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, "|") {
				continue
			}
			cells := splitTableRow(trimmed)
			if len(cells) < 2 {
				continue
			}
			term := strings.TrimSpace(cells[0])
			if term == "" || term == "Term" || strings.HasPrefix(term, "---") {
				continue
			}
			def := strings.TrimSpace(cells[1])
			// Render the definition with any [[…]] cross-refs preserved.
			body := def
			if len(cells) > 2 {
				if u := strings.TrimSpace(cells[2]); u != "" {
					body += "\n\nUse when: " + u
				}
			}
			concepts = append(concepts, Concept{
				ID:          Slugify(term),
				Type:        "Term",
				Title:       term,
				Description: def,
				Tags:        []string{"term"},
				Body:        body,
				SourcePath:  p,
			})
		}
	}
	return concepts
}

// ParseSpecs walks specs/<change>/ directories and emits one Spec concept per
// directory. The title comes from briefing.md's first heading or "**Goal:**"
// line; the status from the briefing's "STATUS" line. A missing briefing.md
// is tolerated (title falls back to the directory name, status empty). An
// empty specs directory yields no concepts.
func ParseSpecs(dir string) []Concept {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var concepts []Concept
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		changeDir := filepath.Join(dir, e.Name())
		title := e.Name()
		status := ""
		var bodyParts []string
		if b, err := os.ReadFile(filepath.Join(changeDir, "briefing.md")); err == nil {
			text := string(b)
			if h := firstHeading(text); h != "" {
				title = h
			}
			if g := firstGoal(text); g != "" {
				bodyParts = append(bodyParts, g)
			}
			status = briefingStatus(text)
			if len(bodyParts) > 0 {
				bodyParts = append(bodyParts, "---", strings.TrimSpace(text))
			} else {
				bodyParts = append(bodyParts, strings.TrimSpace(text))
			}
		}
		concepts = append(concepts, Concept{
			ID:         Slugify(title),
			Type:       "Spec",
			Title:      title,
			Status:     status,
			Tags:       []string{"spec"},
			Body:       strings.Join(bodyParts, "\n\n"),
			SourcePath: changeDir,
		})
	}
	return concepts
}

// ParseSpikes walks spikes/<NNN-name>/ directories and emits one Spike
// concept per directory. The verdict (VALIDATED / INVALIDATED / PARTIAL) is
// scanned from the directory's markdown files; the first found wins. An
// empty spikes directory yields no concepts.
func ParseSpikes(dir string) []Concept {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var concepts []Concept
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		spikeDir := filepath.Join(dir, e.Name())
		concepts = append(concepts, Concept{
			ID:         Slugify(e.Name()),
			Type:       "Spike",
			Title:      e.Name(),
			Status:     spikeVerdict(spikeDir),
			Tags:       []string{"spike"},
			Body:       e.Name(),
			SourcePath: spikeDir,
		})
	}
	return concepts
}

// ParseArchitecture reads .skillgrid/ARCHITECTURE.md into one Architecture
// concept. Existence-gated (R4.4): a missing file returns nil, no error.
func ParseArchitecture(path string) *Concept {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	c := &Concept{
		ID:         "architecture",
		Type:       "Architecture",
		Title:      "Architecture",
		Tags:       []string{"architecture"},
		Body:       strings.TrimRight(string(data), "\n"),
		SourcePath: path,
	}
	return c
}

// ---------- helpers ----------

// findHeading returns the index of the first line that starts with the given
// heading prefix. ok is false when the heading is absent.
func findHeading(lines []string, prefix string) (int, bool) {
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			return i, true
		}
	}
	return 0, false
}

// findAnyADRHeading returns the index of the first "### ADR-NNNN" heading.
func findAnyADRHeading(lines []string) (int, bool) {
	for i, line := range lines {
		if adrHeadingRE.MatchString(strings.TrimSpace(line)) {
			return i, true
		}
	}
	return 0, false
}

// splitTableRow splits a markdown table row "| a | b | c |" into trimmed
// cells ["a", "b", "c"].
func splitTableRow(line string) []string {
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "|") {
		line = line[1:]
	}
	if strings.HasSuffix(line, "|") {
		line = line[:len(line)-1]
	}
	var cells []string
	for _, c := range strings.Split(line, "|") {
		cells = append(cells, strings.TrimSpace(c))
	}
	return cells
}

// isDigits reports whether s is a non-empty string of ASCII digits.
func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// pad4 left-pads a numeric ADR sequence to 4 digits.
func pad4(num string) string {
	if n, err := strconv.Atoi(num); err == nil {
		return fmt.Sprintf("%04d", n)
	}
	return num
}

// adrRef extracts a 4-digit ADR reference ("0006") from a table cell; it
// returns "" for "—" or empty cells.
func adrRef(cell string) string {
	cell = strings.TrimSpace(cell)
	if cell == "" || cell == "—" || cell == "-" {
		return ""
	}
	if isDigits(cell) {
		return "adr-" + pad4(cell)
	}
	return ""
}

// findConcept returns the concept with the given ID, or nil.
func findConcept(concepts []Concept, id string) *Concept {
	for i := range concepts {
		if concepts[i].ID == id {
			return &concepts[i]
		}
	}
	return nil
}

// firstHeading returns the text of the first markdown heading in text.
func firstHeading(text string) string {
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			return strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
		}
	}
	return ""
}

// firstGoal returns the text after a "**Goal:**" line, or "".
func firstGoal(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if g, ok := strings.CutPrefix(strings.TrimSpace(line), "**Goal:**"); ok {
			return strings.TrimSpace(g)
		}
	}
	return ""
}

// briefingStatus extracts the status from a briefing's "> **STATUS:**" line,
// e.g. "`draft` (2026-09-29)" → "draft". Returns "" when absent.
func briefingStatus(text string) string {
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, ">") {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(trimmed, ">"))
		if s, ok := strings.CutPrefix(rest, "**STATUS:**"); ok {
			s = strings.TrimSpace(s)
			s = strings.TrimPrefix(s, "`")
			if i := strings.IndexAny(s, "` ("); i >= 0 {
				s = s[:i]
			}
			return s
		}
	}
	return ""
}

// spikeVerdict scans the markdown files in dir (sorted for determinism) for
// a VALIDATED / INVALIDATED / PARTIAL verdict. A "## Verdict" section wins
// when present; otherwise the first verdict keyword in file order wins.
func spikeVerdict(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	var fallback string
	for _, name := range files {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		text := string(data)
		if idx := strings.Index(text, "\n## Verdict"); idx >= 0 {
			section := text[idx+len("\n## Verdict"):]
			if end := strings.Index(section, "\n## "); end >= 0 {
				section = section[:end]
			}
			// The first non-blank line of the Verdict section carries the
			// bolded verdict (e.g. **PARTIAL**). Prefer it over any verdict
			// word mentioned later in the prose.
			for _, line := range strings.Split(section, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				for _, v := range []string{"VALIDATED", "INVALIDATED", "PARTIAL"} {
					if strings.Contains(line, "**"+v+"**") {
						return v
					}
				}
				break
			}
			// No bolded verdict on the first line; scan the whole section.
			for _, v := range []string{"VALIDATED", "INVALIDATED", "PARTIAL"} {
				if strings.Contains(section, v) {
					return v
				}
			}
			continue
		}
		if fallback == "" {
			for _, v := range []string{"VALIDATED", "INVALIDATED", "PARTIAL"} {
				if strings.Contains(text, v) {
					fallback = v
					break
				}
			}
		}
	}
	return fallback
}

