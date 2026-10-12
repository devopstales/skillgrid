package traffic

import (
	"testing"

	ctxstore "github.com/devopstales/skillgrid/mnemonic/internal/ctx/store"
	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// newTestStore opens a real migrated shared project store and wraps it in the
// ctx accessor (the T06/T08 harness shape). The store is closed via t.Cleanup.
func newTestStore(t *testing.T, project string) *ctxstore.Store {
	t.Helper()
	shared, err := store.Open(t.TempDir(), project)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = shared.Close() })
	cs := ctxstore.New(shared)
	if cs == nil {
		t.Fatal("ctxstore.New returned nil for a valid shared store")
	}
	return cs
}
