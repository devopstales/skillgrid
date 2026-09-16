package service

import (
	"context"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/webcache"
)

// TestOpenEmptyProjectIDAborts guards the handle seam: an empty project id
// must abort open with an error and never yield a partial handle.
func TestOpenEmptyProjectIDAborts(t *testing.T) {
	dataDir := t.TempDir()
	svc := New(dataDir)

	for _, id := range []string{"", "   "} {
		h, cleanup, err := svc.Open(id)
		if err == nil {
			t.Fatalf("Open(%q): expected abort, got nil error", id)
		}
		if h != nil {
			t.Fatalf("Open(%q): expected nil handle on abort, got non-nil", id)
		}
		if cleanup != nil {
			cleanup()
		}
	}
}

// TestOpenInvalidProjectIDAborts covers ids store.Open rejects ("..").
func TestOpenInvalidProjectIDAborts(t *testing.T) {
	dataDir := t.TempDir()
	svc := New(dataDir)

	h, cleanup, err := svc.Open("a/../b")
	if err == nil {
		t.Fatal("Open(a/../b): expected abort, got nil error")
	}
	if h != nil {
		t.Fatal("Open(a/../b): expected nil handle on abort, got non-nil")
	}
	if cleanup != nil {
		cleanup()
	}
}

// TestProjectHandleExposesMemoryAndWeb is the single-project seam contract:
// one open returns an exported *ProjectHandle reaching memory and webcache
// without the wide facade surface.
func TestProjectHandleExposesMemoryAndWeb(t *testing.T) {
	dataDir := t.TempDir()
	svc := New(dataDir)
	const pid = "handle-seam"

	h, cleanup, err := svc.Open(pid)
	if err != nil {
		t.Fatalf("Open(%q): %v", pid, err)
	}
	defer cleanup()
	if h == nil {
		t.Fatal("Open returned nil handle")
	}

	if got := h.ProjectID(); got != pid {
		t.Fatalf("ProjectID()=%q want %q", got, pid)
	}
	if h.Memory() == nil {
		t.Fatal("Memory() returned nil")
	}
	var mem *memory.Service = h.Memory()
	if mem == nil {
		t.Fatal("handle memory service is nil")
	}
	if h.Web() == nil {
		t.Fatal("Web() returned nil")
	}
	if h.Store() == nil {
		t.Fatal("Store() returned nil")
	}

	// Reach memory end-to-end through the handle.
	sessionID, err := h.Memory().SessionStart(context.Background(), ".", "handle seam session")
	if err != nil {
		t.Fatalf("SessionStart via handle: %v", err)
	}
	id, err := h.Memory().Save(context.Background(), memory.SaveInput{
		Title:     "handle seam save",
		Type:      "pattern",
		Content:   "opened once, reached memory and web",
		Scope:     "project",
		SessionID: sessionID,
	})
	if err != nil {
		t.Fatalf("Save via handle: %v", err)
	}
	got, err := h.Memory().Get(context.Background(), id)
	if err != nil {
		t.Fatalf("Get via handle: %v", err)
	}
	if got.Title != "handle seam save" {
		t.Fatalf("title=%q", got.Title)
	}

	// Web through the handle: seed a snapshot via Web().Save, then prove the
	// lookup reaches the same store and actually finds it (hit + id), not just
	// a nil-error miss.
	const exaQuery = "handle seam"
	webID, err := h.Web().Save(context.Background(), webcache.SaveWebInput{
		Source:  "exa",
		Query:   exaQuery,
		Content: "seeded snapshot for handle seam web reachability",
	})
	if err != nil {
		t.Fatalf("Save via handle: %v", err)
	}
	if webID == 0 {
		t.Fatal("Save via handle returned id 0")
	}
	lookup, err := h.Web().Lookup(context.Background(), webcache.LookupInput{
		Source: "exa",
		Query:  exaQuery,
	})
	if err != nil {
		t.Fatalf("Lookup via handle: %v", err)
	}
	if lookup.Status != "hit" {
		t.Fatalf("Lookup status=%q want hit (seeded snapshot not found)", lookup.Status)
	}
	if lookup.ID != webID {
		t.Fatalf("Lookup id=%d want %d (seeded snapshot)", lookup.ID, webID)
	}
}
