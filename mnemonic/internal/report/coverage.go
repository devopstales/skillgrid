package report

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Kind is the detected project language of a scanned root (R9).
type Kind string

const (
	KindGo      Kind = "go"
	KindTS      Kind = "ts"
	KindPy      Kind = "py"
	KindRust    Kind = "rust"
	KindJava    Kind = "java"
	KindUnknown Kind = ""
)

// DetectKind returns the kind of the first manifest present in root:
// go.mod, package.json, pyproject.toml, Cargo.toml, pom.xml, or
// build.gradle. No manifest → KindUnknown.
func DetectKind(root string) Kind {
	for _, f := range []struct {
		kind     Kind
		manifest string
	}{
		{KindGo, "go.mod"},
		{KindTS, "package.json"},
		{KindPy, "pyproject.toml"},
		{KindRust, "Cargo.toml"},
		{KindJava, "pom.xml"},
		{KindJava, "build.gradle"},
	} {
		if _, err := os.Stat(filepath.Join(root, f.manifest)); err == nil {
			return f.kind
		}
	}
	return KindUnknown
}

// CmdFor is the coverage seam: the native coverage command per kind. A new
// language is a new switch arm, not a new collector. KindUnknown has no
// native tool → error (the source reports "unavailable", not an error).
func CmdFor(k Kind) (bin string, args []string, err error) {
	switch k {
	case KindGo:
		return "go", []string{"test", "-coverprofile=cover.out", "./..."}, nil
	case KindTS:
		return "nyc", []string{"report", "--reporter=json"}, nil
	case KindPy:
		return "pytest", []string{"--cov", "--cov-report=json"}, nil
	case KindRust:
		return "cargo-llvm-cov", []string{"--json"}, nil
	case KindJava:
		return "jacoco", []string{"report"}, nil
	default:
		return "", nil, fmt.Errorf("no coverage tool for kind %q", k)
	}
}

// collectCoverage runs the detected kind's native coverage tool and upserts
// the parsed per-file line-% rows into the coverage table (R9, line % only).
// Fail-open (ADR-0016): an undetectable kind, a missing binary, a runner
// error, an unparseable output, or a store error all map to that source's
// "unavailable"/"error" status with no rows and no propagated error.
func (s *Service) collectCoverage(ctx context.Context) (SourceResult, []CoverageRow) {
	unavailable := SourceResult{Tool: "coverage", Status: "unavailable"}
	kind := DetectKind(s.root)
	bin, args, err := CmdFor(kind)
	if err != nil {
		return unavailable, nil // unknown kind: no native coverage tool
	}
	if _, err := exec.LookPath(bin); err != nil {
		return unavailable, nil // kind detected but the tool is not on PATH
	}
	out, err := s.run(ctx, bin, args...)
	if err != nil {
		return SourceResult{Tool: "coverage", Status: "error"}, nil
	}
	rows, err := parseCover(kind, out)
	if err != nil {
		return SourceResult{Tool: "coverage", Status: "error"}, nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if err := StoreCoverage(ctx, s.store.DB, s.root, rows, now); err != nil {
		return SourceResult{Tool: "coverage", Status: "error"}, nil
	}
	return SourceResult{Tool: "coverage", Status: "ok"}, rows
}

// StoreCoverage upserts per-file line-coverage rows into the coverage table,
// one row per (project, file); a re-scan updates pct_line and scanned_at
// instead of duplicating. now is the scanned_at stamp.
func StoreCoverage(ctx context.Context, db *sql.DB, project string, rows []CoverageRow, now string) error {
	for _, r := range rows {
		if _, err := db.ExecContext(ctx,
			`INSERT INTO coverage (project, file, package, pct_line, scanned_at) VALUES (?, ?, ?, ?, ?)
			 ON CONFLICT(project, file) DO UPDATE SET
			   pct_line = excluded.pct_line,
			   scanned_at = excluded.scanned_at`,
			project, r.File, r.Pkg, r.PctLine, now); err != nil {
			return err
		}
	}
	return nil
}

// parseCover is the per-kind coverage-output dispatcher: each kind emits a
// different JSON/text shape (Go cover profile, Istanbul, coverage.py,
// llvm-cov, JaCoCo).
func parseCover(k Kind, data []byte) ([]CoverageRow, error) {
	switch k {
	case KindGo:
		return parseCoverGo(data)
	case KindTS:
		return parseCoverTS(data)
	case KindPy:
		return parseCoverPy(data)
	case KindRust:
		return parseCoverRust(data)
	case KindJava:
		return parseCoverJava(data)
	default:
		return nil, fmt.Errorf("no coverage parser for kind %q", k)
	}
}

// covFileStats accumulates per-file covered/total line counts while parsing a
// coverage report; fileCoverageRows converts them to rows, sorted by file so
// parsers with map-based fixtures produce deterministic output.
type covFileStats struct{ covered, total int }

func fileCoverageRows(files map[string]*covFileStats) []CoverageRow {
	files_ := make([]string, 0, len(files))
	for f := range files {
		files_ = append(files_, f)
	}
	sort.Strings(files_)
	rows := make([]CoverageRow, 0, len(files))
	for _, f := range files_ {
		fs := files[f]
		pct := 0.0
		if fs.total > 0 {
			pct = 100.0 * float64(fs.covered) / float64(fs.total)
		}
		rows = append(rows, CoverageRow{File: f, Pkg: f, PctLine: pct})
	}
	return rows
}

// parseCoverTS maps Istanbul (nyc) coverage-final.json — a map keyed by file
// path whose values carry per-file "statement"/"line" summary counters
// (total lines and uncovered lines) — to per-file line-% rows. Pkg is the
// file path (Istanbul carries no package).
func parseCoverTS(data []byte) ([]CoverageRow, error) {
	var perFile map[string]struct {
		Summary struct {
			Line struct {
				Total     int `json:"total"`
				Uncovered int `json:"uncovered"`
			} `json:"line"`
		} `json:"s"`
	}
	if err := json.Unmarshal(data, &perFile); err != nil {
		return nil, fmt.Errorf("parse istanbul coverage: %w", err)
	}
	files := map[string]*covFileStats{}
	for file, f := range perFile {
		total := f.Summary.Line.Total
		files[file] = &covFileStats{covered: total - f.Summary.Line.Uncovered, total: total}
	}
	rows := fileCoverageRows(files)
	for i := range rows {
		rows[i].Pkg = ""
	}
	return rows, nil
}

// parseCoverPy maps coverage.py's --cov-report=json — the per-file entries of
// "files" keyed by relative path, each carrying "executed_lines" and/or
// "missing" line-number arrays (or a "summary" statements/missed pair) and
// "excluded_lines" — to per-file line-% rows. Pkg is the file path
// (coverage.py carries no package).
func parseCoverPy(data []byte) ([]CoverageRow, error) {
	var report struct {
		Files map[string]struct {
			ExecutedLines []int `json:"executed_lines"`
			Missing       []int `json:"missing"`
			ExcludedLines []int `json:"excluded_lines"`
			Summary       struct {
				Statements int `json:"statements"`
				Missed     int `json:"missed"`
			} `json:"summary"`
		} `json:"files"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("parse coverage.py report: %w", err)
	}
	files := map[string]*covFileStats{}
	for file, f := range report.Files {
		excluded := map[int]bool{}
		for _, l := range f.ExcludedLines {
			excluded[l] = true
		}
		executed := map[int]bool{}
		for _, l := range f.ExecutedLines {
			if !excluded[l] {
				executed[l] = true
			}
		}
		for _, l := range f.Missing {
			if !excluded[l] {
				executed[l] = false
			}
		}
		fs := &covFileStats{}
		for line, hit := range executed {
			_ = line
			fs.total++
			if hit {
				fs.covered++
			}
		}
		if f.Summary.Statements > 0 {
			// The summary is the authoritative statement count when present.
			fs = &covFileStats{covered: f.Summary.Statements - f.Summary.Missed, total: f.Summary.Statements}
		}
		files[file] = fs
	}
	rows := fileCoverageRows(files)
	for i := range rows {
		rows[i].Pkg = ""
	}
	return rows, nil
}

// parseCoverRust maps cargo-llvm-cov --json — the "data" array of per-file
// entries (name, regions: covered/total) — to per-file rows. Line % is
// derived from the region counts (the report carries no separate line
// counts); Pkg is the crate name from the entry path.
func parseCoverRust(data []byte) ([]CoverageRow, error) {
	var report struct {
		Data []struct {
			Name    string `json:"name"`
			Regions struct {
				Covered int `json:"covered"`
				Total   int `json:"total"`
			} `json:"regions"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("parse llvm-cov report: %w", err)
	}
	files := map[string]*covFileStats{}
	for _, d := range report.Data {
		fs := files[d.Name]
		if fs == nil {
			fs = &covFileStats{}
			files[d.Name] = fs
		}
		fs.covered += d.Regions.Covered
		fs.total += d.Regions.Total
	}
	rows := make([]CoverageRow, 0, len(files))
	for name, fs := range files {
		pkg := name
		if i := strings.LastIndex(name, "/"); i >= 0 {
			pkg = name[:i]
		}
		pct := 0.0
		if fs.total > 0 {
			pct = 100.0 * float64(fs.covered) / float64(fs.total)
		}
		rows = append(rows, CoverageRow{File: name, Pkg: pkg, PctLine: pct})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].File < rows[j].File })
	return rows, nil
}

// parseCoverJava maps JaCoCo's JSON report — "classes[]", each with a package
// path (e.g. "com/x"), source filename, and "coverage" (0) counters with
// "missed" and "covered" — to per-file rows; the package is the class's
// package path.
func parseCoverJava(data []byte) ([]CoverageRow, error) {
	var report struct {
		Classes []struct {
			SourceFilename string `json:"sourcefilename"`
			Package        string `json:"package"`
			Coverage       map[string]struct {
				Missed  int `json:"missed"`
				Covered int `json:"covered"`
			} `json:"coverage"`
		} `json:"classes"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("parse jacoco report: %w", err)
	}
	files := map[string]*covFileStats{}
	pkgs := map[string]string{}
	for _, c := range report.Classes {
		counter, ok := c.Coverage["LINE"]
		if !ok {
			continue
		}
		fs := files[c.SourceFilename]
		if fs == nil {
			fs = &covFileStats{}
			files[c.SourceFilename] = fs
		}
		fs.covered += counter.Covered
		fs.total += counter.Missed + counter.Covered
		if _, seen := pkgs[c.SourceFilename]; !seen {
			pkgs[c.SourceFilename] = c.Package
		}
	}
	files_ := make([]string, 0, len(files))
	for f := range files {
		files_ = append(files_, f)
	}
	sort.Strings(files_)
	rows := make([]CoverageRow, 0, len(files))
	for _, f := range files_ {
		fs := files[f]
		pct := 0.0
		if fs.total > 0 {
			pct = 100.0 * float64(fs.covered) / float64(fs.total)
		}
		rows = append(rows, CoverageRow{File: f, Pkg: pkgs[f], PctLine: pct})
	}
	return rows, nil
}

// parseCoverGo extracts per-file line-coverage rows from a Go cover profile:
// a "mode: ..." header line followed by "file:startline.col,endline.col count numStmt"
// block lines (the exact format `go test -coverprofile` writes). Per-file pct
// is covered stmts / total stmts * 100 (stmt-weighted — the cover profile's
// granularity; branch/function % is out of scope, R9).
func parseCoverGo(data []byte) ([]CoverageRow, error) {
	type fileStats struct{ covered, total int }
	files := map[string]*fileStats{}

	for _, line := range strings.Split(string(data), "\n") {
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 3 {
			return nil, fmt.Errorf("parseCoverGo: unparseable line %q", line)
		}
		count, err := strconv.Atoi(fields[1])
		if err != nil {
			return nil, fmt.Errorf("parseCoverGo: bad count in %q", line)
		}
		numStmt, err := strconv.Atoi(fields[2])
		if err != nil {
			return nil, fmt.Errorf("parseCoverGo: bad stmt count in %q", line)
		}
		block, i := fields[0], strings.Index(fields[0], ":")
		if i <= 0 {
			return nil, fmt.Errorf("parseCoverGo: bad block %q", fields[0])
		}
		file := block[:i]
		fs := files[file]
		if fs == nil {
			fs = &fileStats{}
			files[file] = fs
		}
		fs.total += numStmt
		if count > 0 {
			fs.covered += numStmt
		}
	}

	rows := make([]CoverageRow, 0, len(files))
	for file, fs := range files {
		pct := 0.0
		if fs.total > 0 {
			pct = 100.0 * float64(fs.covered) / float64(fs.total)
		}
		rows = append(rows, CoverageRow{File: file, Pkg: "", PctLine: pct})
	}
	return rows, nil
}
