package memory

import (
	"context"
	"strings"
	"testing"
)

func TestToolNameProvenanceStoredOnSave(t *testing.T) {
	_, svc := newTestStore(t, "toolproj")
	sid := newSession(t, svc)
	ctx := context.Background()
	id, err := svc.Save(ctx, SaveInput{
		Title: "tool provenance", Type: "decision", Content: "from mem_save path",
		SessionID: sid, Scope: "project", ToolName: "mem_save",
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	obs, err := svc.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if obs.ToolName != "mem_save" {
		t.Fatalf("tool_name=%q want mem_save", obs.ToolName)
	}
}

func TestInvalidPinRejected(t *testing.T) {
	_, svc := newTestStore(t, "badpin")
	err := svc.Pin(context.Background(), 999999)
	if err == nil {
		t.Fatal("expected error for missing pin id")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want structured not-found error, got %v", err)
	}
}

func TestMalformedExpiresAtRejected(t *testing.T) {
	_, svc := newTestStore(t, "badexp")
	sid := newSession(t, svc)
	ctx := context.Background()
	id, err := svc.Save(ctx, SaveInput{
		Title: "ttl row", Type: "decision", Content: "body", SessionID: sid, Scope: "project",
	})
	if err != nil {
		t.Fatal(err)
	}
	err = svc.SetExpiresAt(ctx, id, "not-a-timestamp")
	if err == nil {
		t.Fatal("expected invalid expires_at rejection")
	}
	if !strings.Contains(err.Error(), "invalid expires_at") {
		t.Fatalf("want invalid expires_at, got %v", err)
	}
}

func TestPinReordersSearchContext(t *testing.T) {
	_, svc := newTestStore(t, "pinorder")
	sid := newSession(t, svc)
	ctx := context.Background()
	older, err := svc.Save(ctx, SaveInput{
		Title: "shared pinorder topic alpha", Type: "decision",
		Content: "alpha body unique", SessionID: sid, Scope: "project",
	})
	if err != nil {
		t.Fatal(err)
	}
	newer, err := svc.Save(ctx, SaveInput{
		Title: "shared pinorder topic beta", Type: "decision",
		Content: "beta body unique", SessionID: sid, Scope: "project",
	})
	if err != nil {
		t.Fatal(err)
	}
	hits, err := svc.Search(ctx, "shared pinorder topic", "any", 10)
	if err != nil || len(hits) < 2 {
		t.Fatalf("baseline search: hits=%d err=%v", len(hits), err)
	}
	if err := svc.Pin(ctx, older); err != nil {
		t.Fatal(err)
	}
	hits, err = svc.Search(ctx, "shared pinorder topic", "any", 10)
	if err != nil || len(hits) < 2 {
		t.Fatalf("pinned search: hits=%d err=%v", len(hits), err)
	}
	if hits[0].ID != older {
		t.Fatalf("pinned obs should sort first: got id=%d want %d (newer=%d)", hits[0].ID, older, newer)
	}
	if err := svc.Unpin(ctx, older); err != nil {
		t.Fatal(err)
	}
}
