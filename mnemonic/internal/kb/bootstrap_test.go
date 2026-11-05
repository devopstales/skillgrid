package kb

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestBootstrapCreatesScaffoldWhenVaultAbsent(t *testing.T) {
	root := t.TempDir()

	sc, err := Bootstrap(context.Background(), root, Options{Project: "skillgrid"})
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	for _, f := range []string{VaultConfigFile, VaultSchemaFile, VaultIgnoreFile} {
		if _, err := os.Stat(filepath.Join(root, f)); err != nil {
			t.Errorf("expected scaffold file %s: %v", f, err)
		}
	}
	for _, d := range append([]string{VaultIndexDir}, PageTypeDirs...) {
		if _, err := os.Stat(filepath.Join(root, d)); err != nil {
			t.Errorf("expected scaffold dir %s: %v", d, err)
		}
	}
	if len(sc.Created) == 0 {
		t.Errorf("expected Bootstrap to report created scaffold entries, got none")
	}
}

func TestBootstrapIdempotentWhenVaultPresent(t *testing.T) {
	root := t.TempDir()
	sentinel := `{"schema_version":1,"project":"skillgrid"}`
	if err := os.WriteFile(filepath.Join(root, VaultConfigFile), []byte(sentinel), 0o644); err != nil {
		t.Fatalf("seed sentinel: %v", err)
	}

	sc, err := Bootstrap(context.Background(), root, Options{Project: "skillgrid"})
	if err != nil {
		t.Fatalf("Bootstrap on existing vault: %v", err)
	}
	if len(sc.Created) != 0 {
		t.Errorf("expected no created entries on existing vault, got %v", sc.Created)
	}
	if len(sc.Skipped) == 0 {
		t.Errorf("expected sentinel to be skipped on existing vault, got none")
	}
}
