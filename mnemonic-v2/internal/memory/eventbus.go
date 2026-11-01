package memory

import (
	"fmt"
	"log"
	"sync"
)

// eventbus.go — the in-process save-event bus (spec 01-memory/001 §5).
//
// The bus is the memory→wiki boundary (dependency inversion: memory emits,
// the wiki consumes — the memory package does not import the wiki).
// Publish runs subscribers synchronously and failure-isolated: a consumer
// error or panic is logged and never fails the publisher or the other
// consumers (a wiki failure never fails a save).
//
// Task 5 creates the bus; the pipeline's step-10 Publish call site is wired
// in Task 7.

// SaveEvent is the committed-save handoff (spec 004-transport-mcp §5):
// enough for the wiki to do entity-targeted regeneration.
type SaveEvent struct {
	SyncID     string
	Project    string
	Scope      string
	Type       string
	TopicKey   string
	Entity     string
	Importance float64
	Degraded   bool
}

// EventBus is the in-process, fan-out save-event bus.
type EventBus struct {
	mu   sync.RWMutex
	subs []func(SaveEvent)
}

// NewEventBus returns an empty bus.
func NewEventBus() *EventBus { return &EventBus{} }

// Subscribe registers a consumer. Order is preserved (subscription order).
func (b *EventBus) Subscribe(fn func(SaveEvent)) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs = append(b.subs, fn)
}

// Publish fans ev out to every subscriber, in order. A consumer that returns
// an error or panics is logged (with the event's sync id for traceability)
// and the fan-out continues — the publisher never sees a failure (spec
// 01-memory/001 §5: failure isolation).
func (b *EventBus) Publish(ev SaveEvent) {
	if b == nil {
		return
	}
	b.mu.RLock()
	subs := make([]func(SaveEvent), len(b.subs))
	copy(subs, b.subs)
	b.mu.RUnlock()
	for _, fn := range subs {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("memory: save-event consumer panic (sync_id=%s): %v", ev.SyncID, r)
				}
			}()
			fn(ev)
		}()
	}
}

var _ = fmt.Sprintf // reserved for later consumer-error formatting (Task 7)
