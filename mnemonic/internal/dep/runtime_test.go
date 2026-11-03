package dep

import (
	"context"
	"strings"
	"testing"
)

// [happy path dep_runtime flags declared-only and import-only]
func TestRuntimeFlagsDeclaredAndImportOnly(t *testing.T) {
	st := openTestStore(t)
	s := New(st.DB)
	ctx := context.Background()

	// files row: the source file to resolve against.
	var fileID int64
	if err := st.DB.QueryRow(`
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES ('src/app.py', 1, 256, 'h-app', '2026-01-01T00:00:00Z')
		RETURNING id`).Scan(&fileID); err != nil {
		t.Fatalf("insert files row: %v", err)
	}
	// symbols row: the importing symbol (from_id must reference a real symbol).
	var fromID int64
	if err := st.DB.QueryRow(`
		INSERT INTO symbols (file_id, name, kind, start_line, end_line, content_hash, uid)
		VALUES (?, 'app', 'module', 1, 30, 'c-app', 'uid-app')
		RETURNING id`, fileID).Scan(&fromID); err != nil {
		t.Fatalf("insert symbols row: %v", err)
	}

	// dependencies rows (retired=0): the declared manifest set.
	for _, purl := range []string{"pkg:pypi/flask@3.0.2", "pkg:pypi/werkzeug@3.0.6"} {
		if _, err := st.DB.Exec(`
			INSERT INTO dependencies (purl, name, version, ecosystem, manifest, retired)
			VALUES (?, ?, ?, 'pypi', 'requirements.txt', 0)`,
			purl, purlToName(purl), purlToVersion(purl)); err != nil {
			t.Fatalf("insert dependency %s: %v", purl, err)
		}
	}

	// edges rows (kind='imports', file_id = the files row): the imported set.
	// flask is imported; os is imported; werkzeug is NOT (declared but unused).
	for _, toName := range []string{"flask", "os"} {
		if _, err := st.DB.Exec(`
			INSERT INTO edges (kind, from_id, file_id, to_name, target_path, confidence, line)
			VALUES ('imports', ?, ?, ?, ?, 'EXTRACTED', 5)`,
			fromID, fileID, toName, toName); err != nil {
			t.Fatalf("insert import edge %s: %v", toName, err)
		}
	}

	res, err := s.Runtime(ctx, "src/app.py")
	if err != nil {
		t.Fatalf("runtime: %v", err)
	}
	if len(res.Shared) != 1 || res.Shared[0] != "flask" {
		t.Errorf("flask should be shared, got %v", res.Shared)
	}
	if len(res.DeclaredOnly) != 1 || res.DeclaredOnly[0] != "werkzeug" {
		t.Errorf("werkzeug should be declared-only, got %v", res.DeclaredOnly)
	}
	if len(res.ImportOnly) != 1 || res.ImportOnly[0] != "os" {
		t.Errorf("os should be import-only, got %v", res.ImportOnly)
	}
}

// purlToName extracts the name segment from a purl (pkg:pypi/flask@3.0.2 -> flask).
func purlToName(purl string) string {
	seg := purl
	if i := strings.LastIndex(seg, "/"); i >= 0 {
		seg = seg[i+1:]
	}
	if i := strings.LastIndex(seg, "@"); i >= 0 {
		seg = seg[:i]
	}
	return seg
}

func purlToVersion(purl string) string {
	if i := strings.LastIndex(purl, "@"); i >= 0 {
		return purl[i+1:]
	}
	return "0.0.0"
}

func TestRuntimeMissingFile(t *testing.T) {
	st := openTestStore(t)
	s := New(st.DB)
	ctx := context.Background()

	if _, err := s.Runtime(ctx, "nope/absent.py"); err == nil {
		t.Errorf("runtime on missing file = nil error, want not-found error")
	}
}
