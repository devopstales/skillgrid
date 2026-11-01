package memory

import (
	"log"
	"strings"
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
// in Task 7; Task 12 completes the SaveEvent payload (Entity + the
// wiki-handoff contract) and proves the boundary for a consumer that does
// not live in internal/memory (internal/wiki_teststub).

// SaveEvent is the committed-save handoff (spec 004-transport-mcp §5):
// enough for the wiki to do entity-targeted regeneration. SessionID is the
// provenance anchor (INV-1) the trail consumer appends under; it is not
// wiki-facing.
type SaveEvent struct {
	SyncID     string
	SessionID  string
	Project    string
	Scope      string
	Type       string
	TopicKey   string
	Entity     string
	Importance float64
	Degraded   bool
	// SupersededSyncID is the target the save superseded (the dedup-delete
	// verdict): non-empty when the save soft-deleted an existing row, and
	// the consumer records the supersedes relation (source = SyncID,
	// target = SupersededSyncID).
	SupersededSyncID string
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

// Publish fans ev out to every subscriber, in order. A consumer that panics
// is recovered and logged (with the event's sync id for traceability) and
// the fan-out continues — the publisher never sees a failure (spec
// 01-memory/001 §5: failure isolation). The consumer signature is a bare
// func (an error-returning consumer cannot fail the publisher either way);
// a consumer with write side effects that must survive a partial failure
// reports it by panicking (the trail consumer does — trails.go).
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

// EntityFor derives the SaveEvent's Entity from a committed save (spec
// 004-transport-mcp §5 "from topic_key family / title"): the topic_key's
// family head (first path segment) when a topic_key is set, else the
// title's first significant term (best-effort: the first word — a title
// that is only punctuation falls through to the whole trimmed title). The
// pipeline calls this at step 10 so the event carries the same value the
// wiki handoff targets; it is pure so the derivation is unit-testable
// without a store.
func EntityFor(topicKey, title string) string {
	if i := strings.IndexByte(topicKey, '/'); i >= 0 {
		if head := topicKey[:i]; head != "" {
			return head
		}
	}
	if topicKey != "" {
		return topicKey
	}
	for _, w := range strings.Fields(title) {
		if term := strings.Trim(w, ".,;:!?()[]{}\"'`-"); term != "" {
			return term
		}
	}
	return strings.TrimSpace(title)
}
