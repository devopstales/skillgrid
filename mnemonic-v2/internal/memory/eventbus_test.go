package memory

import (
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
