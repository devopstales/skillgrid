package report

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
// llvm-cov, JaCoCo). Only the Go arm exists this task; the other four land
// in Task 6 as new switch arms.
func parseCover(k Kind, data []byte) ([]CoverageRow, error) {
	switch k {
	case KindGo:
		return parseCoverGo(data)
	default:
		return nil, fmt.Errorf("no coverage parser for kind %q", k)
	}
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
