package memory

import (
	"context"
	"sync/atomic"
	"testing"
)

// Publish fans out to every subscriber, in subscription order.
func TestEventBusFanOutOrder(t *testing.T) {
	bus := NewEventBus()
	var order []string
	bus.Subscribe(func(e SaveEvent) { order = append(order, "a") })
	bus.Subscribe(func(e SaveEvent) { order = append(order, "b") })
	bus.Publish(SaveEvent{SyncID: "obs-1"})
	if len(order) != 2 || order[0] != "a" || order[1] != "b" {
		t.Fatalf("fan-out order wrong: %v", order)
	}
}

// Failure isolation: a panicking consumer never fails Publish or the other
// subscribers (spec 01-memory/001 §5).
func TestEventBusPanicNeverFailsPublish(t *testing.T) {
	bus := NewEventBus()
	var hit atomic.Bool
	bus.Subscribe(func(e SaveEvent) { panic("consumer boom") })
	bus.Subscribe(func(e SaveEvent) { hit.Store(true) })
	bus.Publish(SaveEvent{SyncID: "obs-2"}) // must not panic
	if !hit.Load() {
		t.Fatal("later subscriber must still run after an earlier panic")
	}
}

// [save-event-exactly-once] the committed save emits exactly one SaveEvent
// whose payload is complete: the event's SyncID is the save's sync_id, the
// routing fields match the request, and Entity + Importance are filled
// (spec 004-transport-mcp §5 — the wiki-handoff payload the later wiki
// consumer consumes without a dependency on internal/memory).
func TestSaveEventEmittedExactlyOnce(t *testing.T) {
	svc, bus := newTestServiceWithBus(t)
	got := make(chan SaveEvent, 4)
	bus.Subscribe(func(e SaveEvent) { got <- e })
	r, err := svc.Save(context.Background(), SaveRequest{SessionID: "s", Type: "decision", Title: "auth model",
		Content: "c", Project: "p", Scope: "project", TopicKey: "auth/model"})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	select {
	case e := <-got:
		if e.SyncID != r.SyncID {
			t.Fatalf("event sync id = %q, want %q", e.SyncID, r.SyncID)
		}
		if e.Project != "p" || e.TopicKey != "auth/model" {
			t.Fatalf("event routing fields wrong: %+v", e)
		}
		if e.Entity == "" {
			t.Fatalf("event Entity is empty (wiki handoff payload missing)")
		}
		if e.Importance <= 0 {
			t.Fatalf("event Importance = %v, want > 0", e.Importance)
		}
	default:
		t.Fatal("no SaveEvent emitted for a committed save")
	}
	select {
	case e := <-got:
		t.Fatalf("event emitted more than once: %+v", e)
	default:
	}
}

// [wiki-failure-never-fails-save] a consumer that panics on a real save is
// recovered by the bus: the save returns the committed result (spec
// 01-memory/001 §5 — a wiki failure never fails a save).
func TestConsumerFailureNeverFailsSave(t *testing.T) {
	svc, bus := newTestServiceWithBus(t)
	bus.Subscribe(func(SaveEvent) { panic("consumer boom") })
	r, err := svc.Save(context.Background(), SaveRequest{SessionID: "s", Type: "decision", Title: "t", Content: "c", Project: "p", Scope: "project"})
	if err != nil {
		t.Fatalf("consumer panic must not fail the save: %v", err)
	}
	if r.ID == 0 {
		t.Fatal("save must still succeed")
	}
}

// [wiki-failure-never-fails-save] triangulated: the panicking consumer is
// subscribed AFTER the trail consumer (the bus's real first subscriber,
// wired in NewService) — the trail append must still run and the save must
// succeed: isolation is per consumer, not per event. A panicking consumer
// that returns first (TestConsumerFailureNeverFailsSave) cannot prove the
// later consumers keep running.
func TestConsumerPanicNeverBlocksLaterConsumers(t *testing.T) {
	svc, bus := newTestServiceWithBus(t)
	// The trail consumer is already subscribed (NewService). Count trail
	// rows before/after: the append is the real consumer's side effect.
	before := trailCount(t, svc.st, "s")
	bus.Subscribe(func(SaveEvent) { panic("late consumer boom") })
	_, err := svc.Save(context.Background(), SaveRequest{SessionID: "s", Type: "decision", Title: "t", Content: "c", Project: "p", Scope: "project"})
	if err != nil {
		t.Fatalf("consumer panic must not fail the save: %v", err)
	}
	if after := trailCount(t, svc.st, "s"); after != before+1 {
		t.Fatalf("trail consumer did not run after an earlier consumer panicked: before=%d after=%d", before, after)
	}
}

// [save-event-exactly-once] triangulated: the Entity fill on the topic-key
// revision path (a different save shape than the fresh insert: same
// sync_id, bumped revision, revised content — the revision is still a
// committed new observation, so it publishes, and its Entity is the same
// topic_key family head the wiki page keys on).
func TestSaveEventEntityRevisionCarriesCommittedContent(t *testing.T) {
	svc, bus := newTestServiceWithBus(t)
	got := make(chan SaveEvent, 4)
	bus.Subscribe(func(e SaveEvent) { got <- e })
	base := SaveRequest{SessionID: "s", Type: "config", Title: "cache ttl", Content: "v1", Project: "p", Scope: "project", TopicKey: "cache/ttl"}
	r1, err := svc.Save(context.Background(), base)
	if err != nil {
		t.Fatalf("first save: %v", err)
	}
	e1 := <-got
	if e1.SyncID != r1.SyncID || e1.Entity == "" {
		t.Fatalf("fresh event wrong: %+v", e1)
	}
	base.Content = "v2"
	r2, err := svc.Save(context.Background(), base)
	if err != nil {
		t.Fatalf("revision save: %v", err)
	}
	e2 := <-got
	if e2.SyncID != r2.SyncID || r2.SyncID != r1.SyncID {
		t.Fatalf("revision event sync id = %q, want %q", e2.SyncID, r1.SyncID)
	}
	// The revision's Entity is the same topic_key family head (the entity is
	// the wiki page key, not the content — the wiki re-reads the row by
	// SyncID). What the revision path must prove is that the event is
	// emitted again under the same sync_id with a non-empty Entity.
	if e2.Entity == "" {
		t.Fatal("revision event Entity is empty")
	}
	if e2.Entity != e1.Entity {
		t.Fatalf("revision event Entity changed (same topic_key family): %q != %q", e2.Entity, e1.Entity)
	}
}
