package pdg

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func contextBackground() context.Context { return context.Background() }

func osMkdirAll(t *testing.T, root, name string) string {
	t.Helper()
	dir := filepath.Join(root, filepath.Dir(name))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	return dir
}

func writeFile2(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

func filepathJoin(root, name string) string {
	return filepath.Join(root, name)
}
