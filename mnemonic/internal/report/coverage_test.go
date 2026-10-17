package report

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// SATISFIES: coverage-dispatch-go

func mkProject(t *testing.T, files ...string) string {
	t.Helper()
	d := t.TempDir()
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(d, f), []byte("{}"), 0o644); err != nil {
			t.Fatalf("write %s: %v", f, err)
		}
	}
	return d
}

func TestDetectKind(t *testing.T) {
	cases := []struct {
		files []string
		want  Kind
	}{
		{[]string{"go.mod"}, KindGo},
		{[]string{"package.json"}, KindTS},
		{[]string{"pyproject.toml"}, KindPy},
		{[]string{"Cargo.toml"}, KindRust},
		{[]string{"pom.xml"}, KindJava},
		{[]string{"build.gradle"}, KindJava},
		{[]string{"README.md"}, KindUnknown},
	}
	for _, c := range cases {
		got := DetectKind(mkProject(t, c.files...))
		if got != c.want {
			t.Errorf("DetectKind(%v) = %q, want %q", c.files, got, c.want)
		}
	}
}

// TestCmdFor_AllFive triangulates DetectKind: every detected kind except
// unknown has a native coverage command, and unknown does not.
func TestCmdFor_AllFive(t *testing.T) {
	if bin, _, err := CmdFor(KindGo); err != nil || bin != "go" {
		t.Errorf("CmdFor(KindGo) = %q, %v; want bin \"go\", no error", bin, err)
	}
	for _, k := range []Kind{KindTS, KindPy, KindRust, KindJava} {
		bin, args, err := CmdFor(k)
		if err != nil {
			t.Errorf("CmdFor(%q) error: %v", k, err)
			continue
		}
		if bin == "" || len(args) == 0 {
			t.Errorf("CmdFor(%q) = %q %v; want a non-empty command", k, bin, args)
		}
	}
	if _, _, err := CmdFor(KindUnknown); err == nil {
		t.Error("CmdFor(KindUnknown) = no error; want an error (no tool)")
	}
}

func TestStoreCoverage_Upsert(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	now := time.Now().UTC().Format(time.RFC3339)

	if err := StoreCoverage(ctx, st.DB, "p", []CoverageRow{{File: "a.go", Pkg: "x", PctLine: 85}}, now); err != nil {
		t.Fatalf("StoreCoverage insert: %v", err)
	}
	if err := StoreCoverage(ctx, st.DB, "p", []CoverageRow{{File: "a.go", Pkg: "x", PctLine: 90}}, now); err != nil {
		t.Fatalf("StoreCoverage upsert: %v", err)
	}

	var n int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM coverage WHERE project='p' AND file='a.go'`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Errorf("rows for (p, a.go) = %d, want 1 (upsert, not duplicate)", n)
	}
	var pct float64
	if err := st.DB.QueryRow(`SELECT pct_line FROM coverage WHERE project='p'`).Scan(&pct); err != nil {
		t.Fatalf("pct_line: %v", err)
	}
	if pct != 90 {
		t.Errorf("pct_line = %v, want 90 (latest scan wins)", pct)
	}
	var scannedAt string
	if err := st.DB.QueryRow(`SELECT scanned_at FROM coverage WHERE project='p'`).Scan(&scannedAt); err != nil {
		t.Fatalf("scanned_at: %v", err)
	}
	if scannedAt != now {
		t.Errorf("scanned_at = %q, want %q", scannedAt, now)
	}
}

// TestParseCoverGo is fixture-driven over two distinct cover profiles so a
// trivial implementation (e.g. single-file or constant pct) cannot pass.
func TestParseCoverGo(t *testing.T) {
	// Real `go test -coverprofile` format: file:startline.col,endline.col count numStmt
	twoFile := []byte(`mode: set
./x.go:1.1,4.2 1 2
./x.go:6.1,8.2 0 2
./y.go:2.1,9.3 1 5
`)
	rows, err := parseCoverGo(twoFile)
	if err != nil {
		t.Fatalf("parseCoverGo: %v", err)
	}
	got := map[string]float64{}
	for _, r := range rows {
		got[r.File] = r.PctLine
	}
	if len(got) != 2 {
		t.Fatalf("files = %d, want 2: %v", len(got), got)
	}
	if got["./x.go"] != 50 {
		t.Errorf("./x.go pct = %v, want 50", got["./x.go"])
	}
	if got["./y.go"] != 100 {
		t.Errorf("./y.go pct = %v, want 100", got["./y.go"])
	}

	// Triangulate: a second, differently-shaped profile (zero-covered file).
	zeroFile := []byte(`mode: set
./z.go:1.1,3.2 0 2
./z.go:5.1,7.2 0 1
`)
	rows, err = parseCoverGo(zeroFile)
	if err != nil {
		t.Fatalf("parseCoverGo (zero): %v", err)
	}
	if len(rows) != 1 || rows[0].File != "./z.go" || rows[0].PctLine != 0 {
		t.Errorf("zero profile rows = %+v, want one row for ./z.go at 0", rows)
	}
}

// TestParseCoverDispatcher: the Go arm routes; unknown kinds error.
func TestParseCoverDispatcher(t *testing.T) {
	if _, err := parseCover(KindGo, []byte("mode: set\n")); err != nil {
		t.Errorf("parseCover(KindGo): %v", err)
	}
	if _, err := parseCover(KindUnknown, nil); err == nil {
		t.Error("parseCover(KindUnknown) = no error; want an error")
	}
}
