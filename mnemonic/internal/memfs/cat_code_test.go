package memfs

import (
	"context"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

func TestCatSymbolSource(t *testing.T) {
	st, _ := store.Open(t.TempDir(), "code-cat")
	t.Cleanup(func() { st.Close() })
	seedCodeIndex(t, st)
	fs := New(st, "code-cat")
	src, err := fs.Cat(context.Background(), "src/auth/login.go::Handler")
	if err != nil {
		t.Fatalf("Cat(symbol): %v", err)
	}
	if !strings.Contains(src, "func Handler() error") {
		t.Errorf("Cat(symbol) = %q, want symbol source", src)
	}
}

func TestCatFileFullText(t *testing.T) {
	st, _ := store.Open(t.TempDir(), "code-cat2")
	t.Cleanup(func() { st.Close() })
	seedCodeIndex(t, st)
	fs := New(st, "code-cat2")
	src, err := fs.Cat(context.Background(), "src/auth/login.go")
	if err != nil {
		t.Fatalf("Cat(file): %v", err)
	}
	if !strings.Contains(src, "func Handler() error") {
		t.Errorf("Cat(file) = %q", src)
	}
}

func TestCatUnknownSymbolErrors(t *testing.T) {
	st, _ := store.Open(t.TempDir(), "code-cat3")
	t.Cleanup(func() { st.Close() })
	seedCodeIndex(t, st)
	fs := New(st, "code-cat3")
	if _, err := fs.Cat(context.Background(), "src/auth/login.go::Nope"); err == nil {
		t.Errorf("Cat(unknown symbol) should error")
	}
}

func TestCatEmptyStoreErrors(t *testing.T) {
	st, _ := store.Open(t.TempDir(), "code-cat4")
	t.Cleanup(func() { st.Close() })
	fs := New(st, "code-cat4")
	if _, err := fs.Cat(context.Background(), "src/auth/login.go"); err == nil {
		t.Errorf("Cat on unindexed store should error")
	}
}
