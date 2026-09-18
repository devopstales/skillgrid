package memfs

import (
	"context"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

func TestTreeRepoTreeWithSymbolCounts(t *testing.T) {
	st, _ := store.Open(t.TempDir(), "code-tree")
	t.Cleanup(func() { st.Close() })
	seedCodeIndex(t, st)
	fs := New(st, "code-tree")

	tree, err := fs.Tree(context.Background(), "")
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}
	if !strings.Contains(tree, "src") {
		t.Errorf("tree missing 'src': %q", tree)
	}
	if !strings.Contains(tree, "login.go") || !strings.Contains(tree, "1 symbols") {
		t.Errorf("tree missing login.go with symbol count: %q", tree)
	}
	if !strings.Contains(tree, "util.go") || !strings.Contains(tree, "0 symbols") {
		t.Errorf("tree missing util.go 0 symbols: %q", tree)
	}
}

func TestTreeSubdirScope(t *testing.T) {
	st, _ := store.Open(t.TempDir(), "code-tree2")
	t.Cleanup(func() { st.Close() })
	seedCodeIndex(t, st)
	fs := New(st, "code-tree2")

	// Scope to src/auth/ → only login.go, not util.go.
	tree, err := fs.Tree(context.Background(), "src/auth/")
	if err != nil {
		t.Fatalf("Tree(src/auth/): %v", err)
	}
	if !strings.Contains(tree, "login.go") {
		t.Errorf("tree src/auth/ missing login.go: %q", tree)
	}
	if strings.Contains(tree, "util.go") {
		t.Errorf("tree src/auth/ should not show util.go: %q", tree)
	}
}
