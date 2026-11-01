package report

import (
	"bufio"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic_back/internal/scan"
)

// debtMarker is a TODO/FIXME/XXX marker at the start of a comment (or a
// standalone marker line): the colon or whitespace after the marker keeps
// plain identifiers like "TODO" in a symbol name out of scope.
var debtMarker = regexp.MustCompile(`(TODO|FIXME|XXX)\s*[:\s]`)

// debtSkipDirs are the directories collectDebt never descends into: generated
// or vendored trees are not project source (and can be huge).
var debtSkipDirs = map[string]bool{
	"node_modules": true,
	".git":         true,
	"vendor":       true,
}

// debtScanLimit caps the debt source at this many findings so a marker-flooded
// tree cannot bloat the report or the findings store.
// ponytail: hard cap on total findings; a per-file cap + "N more" note would
// be a better UX but needs a UI leg — the report stays correct either way.
const debtScanLimit = 1000

// collectDebt greps s.root for TODO/FIXME/XXX markers and records each as a
// scan.Finding (rule_id debt:<marker>, file + line, severity LOW). It walks
// the whole project root skipping node_modules/.git/vendor, so the findings
// are project-wide debt, not only what a scanner saw. Fail-open (ADR-0016):
// an unreadable root (or missing root) is a zero-finding ok source, never a
// propagated error.
func (s *Service) collectDebt(ctx context.Context) SourceResult {
	res := SourceResult{Tool: "debt", Status: "ok"}
	walkErr := filepath.WalkDir(s.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // unreadable file/dir: skip, never fail the report
		}
		if d.IsDir() {
			if debtSkipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !isDebtSourceFile(d.Name()) {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		if err := s.scanDebtFile(f, path, &res); err != nil {
			// Malformed lines are skipped inside scanDebtFile; only IO errors
			// surface, and they are dropped fail-open.
			return nil
		}
		return nil
	})
	if walkErr != nil {
		// The root itself was unreadable: zero findings, still ok.
		return res
	}
	return res
}

// scanDebtFile reads one source file line by line and appends a debt finding
// per marker hit to res. Findings are capped at debtScanLimit; relative paths
// (from s.root) are recorded so the report is stable across machines.
func (s *Service) scanDebtFile(f *os.File, path string, res *SourceResult) error {
	rel, err := filepath.Rel(s.root, path)
	if err != nil {
		rel = path
	}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	line := 0
	for sc.Scan() {
		line++
		sub := debtMarker.FindStringSubmatch(sc.Text())
		if sub == nil {
			continue
		}
		if len(res.Findings) >= debtScanLimit {
			return nil
		}
		marker := sub[1]
		res.Findings = append(res.Findings, scan.Finding{
			Tool:     "debt",
			RuleID:   "debt:" + marker,
			Severity: "LOW",
			Title:    debtMarkerText(sc.Text()),
			File:     rel,
			Line:     line,
		})
	}
	return sc.Err()
}

// debtMarkerText builds the finding title: "<marker>: <rest of the line
// after the marker>".
func debtMarkerText(line string) string {
	m := debtMarker.FindStringSubmatchIndex(line)
	if m == nil {
		return line
	}
	rest := strings.TrimSpace(line[m[3]:])
	if rest == "" {
		return line[m[2]:m[3]]
	}
	return line[m[2]:m[3]] + ": " + rest
}

// isDebtSourceFile keeps text files with a source-like extension and skips
// binaries, build artifacts, and fixtures.
func isDebtSourceFile(name string) bool {
	if strings.HasPrefix(name, ".") {
		return false
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs", ".py", ".rs",
		".java", ".kt", ".rb", ".c", ".cc", ".cpp", ".h", ".hpp", ".cs",
		".swift", ".scala", ".php", ".sh", ".sql", ".vue", ".svelte":
		return true
	default:
		return false
	}
}
