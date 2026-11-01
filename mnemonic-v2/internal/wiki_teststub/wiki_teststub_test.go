// Package wiki_teststub is the dependency-direction proof (spec
// 000-architecture §3): a consumer that lives OUTSIDE internal/memory,
// imports only the event-bus package + event type (never
// internal/memory's pipeline types), and receives a well-formed SaveEvent
// after a save driven through memory.Service. The later llm-wiki consumer
// plugs into this exact shape; the import-check guard
// (internal/memory/importcheck_test.go) keeps memory from importing it.
package wiki_teststub

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/memory"
)

// newStubService builds a memory.Service over a real SQLite store wired to a
// fresh bus (the stub consumer is the only external subscriber).
func newStubService(t *testing.T) (*memory.Service, *memory.EventBus) {
	t.Helper()
	c := config.Default()
	c.DataDir = filepath.Join(t.TempDir(), "stub-data")
	c.Project.Detect = false
	st, err := store.Open(context.Background(), c.DataDir, "p", store.SQLite, "")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	bus := memory.NewEventBus()
	return memory.NewService(st, c, bus, nil, nil), bus
}

// TestWikiStubConsumes: a consumer outside internal/memory subscribes to the
// bus and receives the save's event — the handoff boundary works for a
// consumer package that does not live in internal/memory.
func TestWikiStubConsumes(t *testing.T) {
	svc, bus := newStubService(t)
	got := make(chan memory.SaveEvent, 4)
	// The stub consumer body: a stand-in for the later llm-wiki's handler.
	// It only reads the event — the wiki-facing payload (Entity, Importance,
	// routing fields) must be well-formed for the handoff to be usable.
	bus.Subscribe(func(e memory.SaveEvent) { got <- e })

	res, err := svc.Save(context.Background(), memory.SaveRequest{
		SessionID: "s", Type: "architecture", Title: "bus shape",
		Content: "the stub consumes across the package boundary",
		Project: "p", Scope: "project", TopicKey: "memory/bus",
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	select {
	case e := <-got:
		// well-formed handoff payload (spec 004-transport-mcp §5)
		if e.SyncID != res.SyncID {
			t.Fatalf("event sync id = %q, want %q", e.SyncID, res.SyncID)
		}
		if e.Entity == "" {
			t.Fatalf("stub consumer got an event with no Entity (handoff payload missing)")
		}
		if e.Importance <= 0 {
			t.Fatalf("stub consumer got an event with Importance %v (want > 0)", e.Importance)
		}
		if e.Project != "p" || e.TopicKey != "memory/bus" {
			t.Fatalf("stub consumer got wrong routing fields: %+v", e)
		}
	default:
		t.Fatal("stub consumer never received the event")
	}
}
