package skills

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

func openTestStore(t *testing.T) (*store.Store, string) {
	t.Helper()
	root := t.TempDir()
	st, err := store.Open(filepath.Join(root, "data"), "skilltest")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st, root
}

// TestWriteCreatesFSFileSQLRowAndFTS covers @step-03 (write portion):
// Write creates the FS file under .skillgrid/files/skills/{name}.{ext},
// inserts the SQL metadata row, and the FTS index mirrors it.
func TestWriteCreatesFSFileSQLRowAndFTS(t *testing.T) {
	st, root := openTestStore(t)

	s := New(st.DB, root)
	id, err := s.Write(context.Background(), "deploy-check", "sh", "pre-deploy checklist", "echo ok\n", false)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if id <= 0 {
		t.Fatalf("Write returned id %d, want > 0", id)
	}

	// FS: .skillgrid/files/skills/deploy-check.sh exists with the code.
	abs := filepath.Join(root, ".skillgrid", "files", "skills", "deploy-check.sh")
	data, err := os.ReadFile(abs)
	if err != nil {
		t.Fatalf("FS file missing: %v", err)
	}
	if string(data) != "echo ok\n" {
		t.Errorf("FS content = %q, want the code as written", data)
	}

	// SQL: metadata row with code_path under files/skills/.
	var name, language, description, codePath string
	var deletedAt sql.NullString
	if err := st.DB.QueryRow(`
		SELECT name, language, description, code_path, deleted_at
		FROM skills WHERE id = ?`, id).
		Scan(&name, &language, &description, &codePath, &deletedAt); err != nil {
		t.Fatalf("read skill: %v", err)
	}
	if name != "deploy-check" || language != "sh" || description != "pre-deploy checklist" {
		t.Errorf("skill row = %q/%q/%q", name, language, description)
	}
	if !strings.Contains(codePath, "files/skills/deploy-check.sh") {
		t.Errorf("code_path = %q, want it under files/skills/", codePath)
	}
	if deletedAt.Valid {
		t.Errorf("deleted_at should be NULL for a fresh skill")
	}

	// FTS: the name and description are matchable.
	var ftsCount int
	if err := st.DB.QueryRow(`
		SELECT COUNT(*) FROM skills_fts WHERE skills_fts MATCH 'checklist'`).Scan(&ftsCount); err != nil {
		t.Fatalf("skills_fts match: %v", err)
	}
	if ftsCount != 1 {
		t.Errorf("skills_fts matches = %d, want 1", ftsCount)
	}
}

// TestWriteRejectsNameCollisionWithoutOverwrite covers @step-03 (edge:
// Overwrite false rejects name collision): a second Write with the same name
// and overwrite=false is a clear error, writes nothing, and touches no FS file.
func TestWriteRejectsNameCollisionWithoutOverwrite(t *testing.T) {
	st, root := openTestStore(t)
	s := New(st.DB, root)

	if _, err := s.Write(context.Background(), "dupe", "sh", "first", "echo 1\n", false); err != nil {
		t.Fatalf("Write first: %v", err)
	}
	if _, err := s.Write(context.Background(), "dupe", "sh", "second", "echo 2\n", false); err == nil {
		t.Fatal("second Write with overwrite=false should fail")
	}

	// No extra row and the FS file keeps the first content.
	var n int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM skills WHERE name = 'dupe'`).Scan(&n); err != nil {
		t.Fatalf("count skills: %v", err)
	}
	if n != 1 {
		t.Errorf("skills rows for dupe = %d, want 1", n)
	}
	data, err := os.ReadFile(filepath.Join(root, ".skillgrid", "files", "skills", "dupe.sh"))
	if err != nil {
		t.Fatalf("read dupe.sh: %v", err)
	}
	if string(data) != "echo 1\n" {
		t.Errorf("dupe.sh = %q, want the first write's content", data)
	}
}

// TestWriteOverwriteReplaces: overwrite=true replaces the FS file, the SQL
// metadata, and (via the update trigger) the FTS index.
func TestWriteOverwriteReplaces(t *testing.T) {
	st, root := openTestStore(t)
	s := New(st.DB, root)

	id1, err := s.Write(context.Background(), "rev", "sh", "v1", "echo 1\n", false)
	if err != nil {
		t.Fatalf("Write v1: %v", err)
	}
	id2, err := s.Write(context.Background(), "rev", "sh", "v2", "echo 2\n", true)
	if err != nil {
		t.Fatalf("Write v2: %v", err)
	}

	// The row is updated in place (name is UNIQUE: one live row).
	var n int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM skills WHERE name = 'rev'`).Scan(&n); err != nil {
		t.Fatalf("count skills: %v", err)
	}
	if n != 1 {
		t.Errorf("skills rows for rev = %d, want 1", n)
	}
	var description string
	if err := st.DB.QueryRow(`SELECT description FROM skills WHERE id = ?`, id2).Scan(&description); err != nil {
		t.Fatalf("read rev: %v", err)
	}
	if description != "v2" {
		t.Errorf("description = %q, want v2", description)
	}
	if id1 == id2 {
		// Overwrite reuses the row id (UPDATE in place).
		t.Logf("overwrite reused id %d", id1)
	}

	data, err := os.ReadFile(filepath.Join(root, ".skillgrid", "files", "skills", "rev.sh"))
	if err != nil {
		t.Fatalf("read rev.sh: %v", err)
	}
	if string(data) != "echo 2\n" {
		t.Errorf("rev.sh = %q, want the overwritten content", data)
	}

	// FTS mirrors the new description only.
	var oldHits, newHits int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM skills_fts WHERE skills_fts MATCH '"v1"'`).Scan(&oldHits); err != nil {
		t.Fatalf("fts v1: %v", err)
	}
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM skills_fts WHERE skills_fts MATCH '"v2"'`).Scan(&newHits); err != nil {
		t.Fatalf("fts v2: %v", err)
	}
	if oldHits != 0 || newHits != 1 {
		t.Errorf("FTS v1=%d v2=%d, want 0/1", oldHits, newHits)
	}
}

// TestListReturnsNonDeletedSkills covers @step-03 (list portion): List
// returns live skills with metadata and omits soft-deleted ones.
func TestListReturnsNonDeletedSkills(t *testing.T) {
	st, root := openTestStore(t)
	s := New(st.DB, root)

	ctx := context.Background()
	if _, err := s.Write(ctx, "keep", "sh", "kept skill", "echo keep\n", false); err != nil {
		t.Fatalf("Write keep: %v", err)
	}
	goneID, err := s.Write(ctx, "gone", "sh", "gone skill", "echo gone\n", false)
	if err != nil {
		t.Fatalf("Write gone: %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := st.DB.Exec(`UPDATE skills SET deleted_at = ? WHERE id = ?`, now, goneID); err != nil {
		t.Fatalf("soft-delete gone: %v", err)
	}

	got, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("List returned %d skills, want 1 (soft-deleted omitted)", len(got))
	}
	if got[0].Name != "keep" || got[0].Language != "sh" || got[0].Description != "kept skill" {
		t.Errorf("List[0] = %+v, want the keep skill", got[0])
	}
}

// TestSearchReturnsMatchesExcludingSoftDeleted covers @step-03 (search
// portion): Search is lexical FTS over skills_fts, ranked by bm25, and
// soft-deleted skills stay absent from the default search.
func TestSearchReturnsMatchesExcludingSoftDeleted(t *testing.T) {
	st, root := openTestStore(t)
	s := New(st.DB, root)
	ctx := context.Background()

	keptID, err := s.Write(ctx, "cache-warm", "sh", "warm the registry cache", "echo warm\n", false)
	if err != nil {
		t.Fatalf("Write kept: %v", err)
	}
	goneID, err := s.Write(ctx, "cache-cool", "sh", "cool the registry cache", "echo cool\n", false)
	if err != nil {
		t.Fatalf("Write gone: %v", err)
	}
	if _, err := s.Write(ctx, "unrelated", "sh", "a cooking recipe", "echo soup\n", false); err != nil {
		t.Fatalf("Write unrelated: %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := st.DB.Exec(`UPDATE skills SET deleted_at = ? WHERE id = ?`, now, goneID); err != nil {
		t.Fatalf("soft-delete gone: %v", err)
	}

	got, err := s.Search(ctx, "cache", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Search returned %d skills, want 1 (soft-deleted excluded)", len(got))
	}
	if got[0].ID != keptID {
		t.Errorf("Search returned skill %d, want %d (the live one)", got[0].ID, keptID)
	}

	// includeDeleted=true is the explicit opt-in escape hatch.
	all, err := s.SearchWith(ctx, "cache", 10, true)
	if err != nil {
		t.Fatalf("SearchWith includeDeleted: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("SearchWith includeDeleted returned %d skills, want 2", len(all))
	}
}

// TestSearchBlankQueryReturnsNoMatches guards the empty-query boundary.
func TestSearchBlankQueryReturnsNoMatches(t *testing.T) {
	st, root := openTestStore(t)
	s := New(st.DB, root)
	if _, err := s.Write(context.Background(), "s", "sh", "searchable", "echo s\n", false); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := s.Search(context.Background(), "   ", 10)
	if err != nil {
		t.Fatalf("Search blank: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Search blank query returned %d skills, want 0", len(got))
	}
}

// TestWriteRejectsUnknownLanguage guards the registry boundary: a language
// outside the known set is rejected before any FS write or SQL row exists.
func TestWriteRejectsUnknownLanguage(t *testing.T) {
	st, root := openTestStore(t)
	s := New(st.DB, root)
	if _, err := s.Write(context.Background(), "badlang", "cobol", "desc", "x\n", false); err == nil {
		t.Fatal("Write with unknown language should fail")
	}
	var n int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM skills`).Scan(&n); err != nil {
		t.Fatalf("count skills: %v", err)
	}
	if n != 0 {
		t.Errorf("skills rows = %d, want 0", n)
	}
	if _, err := os.Stat(filepath.Join(root, ".skillgrid", "files", "skills", "badlang.cobol")); !os.IsNotExist(err) {
		t.Errorf("FS file should not exist for a rejected language, stat err = %v", err)
	}
}

// TestWriteRejectsNamePathEscape guards the path-escape boundary: a name that
// could escape the skills directory is rejected with no FS or SQL side effect.
func TestWriteRejectsNamePathEscape(t *testing.T) {
	st, root := openTestStore(t)
	s := New(st.DB, root)
	for _, name := range []string{"../evil", "a/b", "x\\y", ""} {
		if _, err := s.Write(context.Background(), name, "sh", "desc", "x\n", false); err == nil {
			t.Errorf("Write name %q should fail (path escape / blank)", name)
		}
	}
	var n int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM skills`).Scan(&n); err != nil {
		t.Fatalf("count skills: %v", err)
	}
	if n != 0 {
		t.Errorf("skills rows = %d, want 0", n)
	}
}
