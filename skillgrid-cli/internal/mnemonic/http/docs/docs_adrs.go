package docs

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// adrFilesGlob is where locked decisions live (ADR-0019). ASSUMPTIONS.md
// stores the path; these files store the body. Relative to docsCwd.
const adrFilesGlob = ".skillgrid/artifacts/04-adr-*.md"

// assumptionsPath is the index. A repo with this file and no ADR files is an
// empty list, not a 404. A repo with neither is a 404.
const assumptionsPath = ".skillgrid/ASSUMPTIONS.md"

var adrFileRE = regexp.MustCompile(`^04-adr-(\d+)-.+\.md$`)

// ADR is one architectural decision record parsed from an artifacts ADR file.
type ADR struct {
	ID       string `json:"id"`
	Number   int    `json:"number"`
	Title    string `json:"title"`
	Removed  bool   `json:"removed"`
	Decision string `json:"decision,omitempty"`
	Why      string `json:"why,omitempty"`
	Conseq   string `json:"consequences,omitempty"`
	Source   string `json:"source"`
}

type adrSummary struct {
	Total   int `json:"total"`
	InForce int `json:"in_force"`
	Removed int `json:"removed"`
	Next    int `json:"next_number"`
}

// NewADRs returns the GET /docs/adrs handler bound to cwd. It parses
// `.skillgrid/artifacts/04-adr-*.md` and returns them as JSON so the Decisions
// panel can surface the repo's ADR files.
func NewADRs(cwd string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		adrs, err := ParseADRs(r.Context(), cwd)
		if err != nil {
			code := http.StatusInternalServerError
			if os.IsNotExist(err) {
				code = http.StatusNotFound
			}
			writeJSON(w, code, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"source":  ".skillgrid/artifacts",
			"summary": adrSummaryFor(adrs),
			"adrs":    adrs,
		})
	})
}

// ParseADRs reads `.skillgrid/artifacts/04-adr-*.md`. Missing artifacts and a
// missing ASSUMPTIONS.md return os.ErrNotExist (→ 404). An index with no ADR
// files returns an empty slice.
func ParseADRs(ctx context.Context, cwd string) ([]ADR, error) {
	matches, err := filepath.Glob(filepath.Join(cwd, filepath.FromSlash(adrFilesGlob)))
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		if _, statErr := os.Stat(filepath.Join(cwd, filepath.FromSlash(assumptionsPath))); statErr != nil {
			return nil, statErr
		}
		return []ADR{}, nil
	}
	sort.Strings(matches)
	var out []ADR
	for _, full := range matches {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		name := filepath.Base(full)
		m := adrFileRE.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		data, err := os.ReadFile(full)
		if err != nil {
			return nil, err
		}
		num, _ := strconv.Atoi(m[1])
		rel := filepath.ToSlash(filepath.Join(".skillgrid", "artifacts", name))
		out = append(out, adrFromFile(string(data), num, m[1], rel))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Number < out[j].Number })
	return out, nil
}

func adrFromFile(content string, num int, digits, source string) ADR {
	title := titleFrom(content)
	adr := ADR{
		ID:      "ADR-" + digits,
		Number:  num,
		Title:   title,
		Source:  source,
		Removed: removedRE.MatchString(content) || statusRemoved(content),
	}
	if fm := decisionRE.FindStringSubmatch(content); fm != nil {
		adr.Decision = strings.TrimSpace(fm[1])
	} else {
		adr.Decision = sectionText(content, "Decision Outcome")
		if adr.Decision == "" {
			adr.Decision = sectionText(content, "Decision")
		}
	}
	if fm := whyRE.FindStringSubmatch(content); fm != nil {
		adr.Why = strings.TrimSpace(fm[1])
	} else {
		adr.Why = sectionText(content, "Context and Problem Statement")
		if adr.Why == "" {
			adr.Why = sectionText(content, "Context")
		}
	}
	if fm := conseqRE.FindStringSubmatch(content); fm != nil {
		adr.Conseq = strings.TrimSpace(fm[1])
	} else {
		adr.Conseq = sectionText(content, "Consequences")
	}
	if adr.Removed {
		adr.Decision = ""
		adr.Why = ""
		adr.Conseq = ""
	}
	return adr
}

func titleFrom(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}

func statusRemoved(content string) bool {
	re := regexp.MustCompile(`(?m)^status:\s*"?(superseded|removed)"?`)
	return re.MatchString(content)
}

// sectionText returns the body under a markdown heading of that name, up to
// the next heading.
func sectionText(content, heading string) string {
	lines := strings.Split(content, "\n")
	capturing := false
	var b strings.Builder
	for _, line := range lines {
		if !capturing {
			t := strings.TrimSpace(line)
			if t == "## "+heading || t == "### "+heading || t == "#### "+heading {
				capturing = true
			}
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			break
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return strings.TrimSpace(b.String())
}

var (
	decisionRE = regexp.MustCompile(`(?m)^\*{2}Decision\.\*{2}\s*(.+)$`)
	whyRE      = regexp.MustCompile(`(?m)^\*{2}Why\.\*{2}\s*(.+)$`)
	conseqRE   = regexp.MustCompile(`(?m)^\*{2}Consequences\.\*{2}\s*(.+)$`)
	removedRE  = regexp.MustCompile(`(?mi)removed on \d{4}-\d{2}-\d{2}|This ADR was removed`)
)

func adrSummaryFor(adrs []ADR) adrSummary {
	s := adrSummary{Total: len(adrs), Next: 1}
	max := 0
	for _, a := range adrs {
		if a.Removed {
			s.Removed++
		} else {
			s.InForce++
		}
		if a.Number > max {
			max = a.Number
		}
	}
	s.Next = max + 1
	return s
}
