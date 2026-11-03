package affected

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCodeRenameApply covers @step-02 (Scenario: code_rename applies only
// listed files without committing): a non-dry rename edits exactly the files
// in its plan (no other file touched) and performs no git side effect (no
// commit, no push).
func TestCodeRenameApply(t *testing.T) {
	st := renameFixture(t)
	ctx := context.Background()

	// Build a real on-disk repo whose working tree matches the indexed graph:
	// base (src/base.go) is called by mid (src/mid.go); src/touch_me.go is
	// OUTSIDE the plan and must not be edited.
	root := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	write("src/base.go", "package p\n\nfunc base() int {\n\treturn 1\n}\n")
	write("src/mid.go", "package p\n\nfunc mid() int {\n\treturn base()\n}\n")
	write("src/touch_me.go", "package p\n\n// base is fine\nfunc other() {}\n")

	// dry_run first: the plan must not write anything.
	dry, err := Rename(ctx, st.DB, RenameOptions{Old: "base", New: "verifyBase"})
	if err != nil {
		t.Fatalf("rename(dry): %v", err)
	}
	if !dry.DryRun {
		t.Fatalf("expected dry_run=true, got %v", dry)
	}
	b, _ := os.ReadFile(filepath.Join(root, "src/base.go"))
	if !strings.Contains(string(b), "func base()") {
		t.Fatalf("dry_run must not write: base.go changed to %q", b)
	}

	oldDir, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(oldDir) }()

	// Non-dry apply: edits exactly the planned files.
	applied, err := Rename(ctx, st.DB, RenameOptions{Old: "base", New: "verifyBase", Apply: true})
	if err != nil {
		t.Fatalf("rename(apply): %v", err)
	}
	if applied.DryRun {
		t.Fatalf("apply must be non-dry, got %v", applied)
	}
	if !applied.Applied {
		t.Fatalf("expected applied=true, got %v", applied)
	}
	// base.go: the definition was renamed.
	b1, _ := os.ReadFile(filepath.Join(root, "src/base.go"))
	if !strings.Contains(string(b1), "func verifyBase()") {
		t.Errorf("base.go definition must be renamed, got %q", b1)
	}
	// mid.go: the typed caller was renamed.
	b2, _ := os.ReadFile(filepath.Join(root, "src/mid.go"))
	if !strings.Contains(string(b2), "verifyBase()") {
		t.Errorf("mid.go caller must be renamed, got %q", b2)
	}
	// touch_me.go: OUTSIDE the plan — untouched.
	b3, _ := os.ReadFile(filepath.Join(root, "src/touch_me.go"))
	if strings.Contains(string(b3), "verifyBase") {
		t.Errorf("touch_me.go is outside the plan and must be untouched, got %q", b3)
	}
	// No git side effect: a fresh repo (no commits) must not have gained one.
	// The apply path does no git work at all, so the .git dir does not exist.
	if _, err := os.Stat(filepath.Join(root, ".git")); !os.IsNotExist(err) {
		t.Errorf("apply must not perform git side effects (a .git dir appeared), err=%v", err)
	}
}
