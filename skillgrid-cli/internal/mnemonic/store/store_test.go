package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"skillgrid-cli/internal/mnemonic/store"
)

func TestStoreMigrations(t *testing.T) {
	dir := t.TempDir()
	s, err := store.Open(dir, "test-project")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	var v int
	if err := s.DB.QueryRow("SELECT schema_version FROM index_meta WHERE key='schema_version'").Scan(&v); err != nil {
		t.Fatal(err)
	}
	if v != 1 {
		t.Fatalf("schema_version=%d", v)
	}
}

func TestStoreOpenProjectIDWithSlashes(t *testing.T) {
	dir := t.TempDir()
	projectID := "org/custom-name"
	s, err := store.Open(dir, projectID)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	want := filepath.Join(dir, projectID+".sqlite")
	if s.DB == nil {
		t.Fatal("expected open database")
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("expected db at %s: %v", want, err)
	}
}
