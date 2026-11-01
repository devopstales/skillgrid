package files

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestContentPlaneWriteRollbackOnSQLFail — @step-01 Scenario: SQL fail after FS write rolls back
func TestContentPlaneWriteRollbackOnSQLFail(t *testing.T) {
	root := t.TempDir()
	cp := NewContentPlane(root)

	_, err := cp.Write(KindTasks, "task-1", "brief.md", []byte("# brief\n"), func(relPath string) error {
		return errors.New("sql insert failed")
	})
	if err == nil {
		t.Fatal("expected error when commit fails")
	}
	if !strings.Contains(err.Error(), "sql insert failed") {
		t.Errorf("error should wrap commit failure, got %v", err)
	}

	abs := filepath.Join(root, ".skillgrid", "files", "tasks", "task-1", "brief.md")
	if _, statErr := os.Stat(abs); !os.IsNotExist(statErr) {
		t.Fatalf("expected orphan file removed, stat=%v", statErr)
	}
}

// TestContentPlaneWriteStoresMarkdownPathsOnly — @step-01 Scenario: Markdown on disk SQL paths only
func TestContentPlaneWriteStoresMarkdownPathsOnly(t *testing.T) {
	root := t.TempDir()
	var seamCalled bool
	cp := NewContentPlane(root)
	cp.AfterWrite = func(relPath string, content []byte) error {
		seamCalled = true
		return nil // no-op seam (no L0/L1/L2)
	}

	var committedRel string
	rel, err := cp.Write(KindTasks, "task-2", "brief.md", []byte("# hello\n"), func(relPath string) error {
		committedRel = relPath
		// Simulate SQL storing path/status only — no content blob.
		if strings.Contains(relPath, "# hello") {
			t.Errorf("relPath must not embed markdown body")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if committedRel != rel {
		t.Errorf("commit path %q != returned %q", committedRel, rel)
	}
	if !strings.HasPrefix(rel, "files/tasks/") || !strings.HasSuffix(rel, "brief.md") {
		t.Errorf("unexpected rel path %q", rel)
	}
	if !seamCalled {
		t.Error("expected no-op AfterWrite seam to be invoked")
	}

	abs := filepath.Join(root, ".skillgrid", rel)
	body, err := os.ReadFile(abs)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(body) != "# hello\n" {
		t.Errorf("disk content = %q", body)
	}

	got, err := cp.Read(rel)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if string(got) != "# hello\n" {
		t.Errorf("Read = %q", got)
	}
}
