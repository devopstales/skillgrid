package mcp

// writequeue.go — the serializing write queue (spec 004-transport-mcp §4,
// the Engram internal/mcp/write_queue.go pattern). Every write tool
// (mem_save, mem_update, mem_delete, mem_judge, mem_judge_many,
// mem_consolidate, mem_save_prompt, mem_capture_passive, mem_pin,
// mem_unpin) enqueues its call and waits for the result; a single worker
// goroutine runs them one at a time. Reads bypass the queue (they are
// safe to run concurrently — the store's read path is single-connection,
// Task 1).
//
// One queued write = one transaction (the store is single-connection;
// the queue guarantees no two transactions interleave). A failed write
// does not wedge the queue: the worker recovers a panic, answers the
// caller's channel with the error, and moves to the next item.

import (
	"context"
	"fmt"
	"sync"
)

type writeItem struct {
	fn     func()
	done   chan struct{}
	result error
}

type writeQueue struct {
	ch         chan writeItem
	mu         sync.Mutex
	serialized bool
	writes     int
	dead       bool
	workerDone chan struct{}
	started    bool
}

func newWriteQueue() *writeQueue {
	q := &writeQueue{
		ch:         make(chan writeItem, 64),
		workerDone: make(chan struct{}),
		started:    true,
	}
	go q.worker()
	return q
}

func (q *writeQueue) worker() {
	for item := range q.ch {
		q.mu.Lock()
		if !q.dead {
			// A panic in a write fn must not kill the worker (a failed
			// write must not wedge the queue — the next write still runs).
			func() {
				defer func() {
					if r := recover(); r != nil {
						item.result = fmt.Errorf("write queue: panic: %v", r)
					}
				}()
				item.fn()
				q.serialized = true
				q.writes++
			}()
		}
		q.mu.Unlock()
		close(item.done)
	}
	close(q.workerDone)
}

// Submit enqueues fn and waits for it to run (the write tool's call).
// Returns the write's error (nil when the fn completed without
// panicking; the fn's own result is captured by the caller's closure).
// A cancelled ctx is returned (a dead queue — a test's stopped worker —
// is answered, not hung: the item is still enqueued and the worker
// answers its done channel).
func (q *writeQueue) Submit(ctx context.Context, fn func()) error {
	item := writeItem{fn: fn, done: make(chan struct{})}
	select {
	case q.ch <- item:
	case <-ctx.Done():
		return ctx.Err()
	}
	<-item.done
	return item.result
}

// WritesSerialized reports whether at least one write ran through the
// queue (the test hook: a write that bypassed the queue would leave
// this false after a concurrent-submit test).
func (q *writeQueue) WritesSerialized() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.serialized
}

// WritesSinceLastReset returns the total write count (the test hook for
// the judge-many-once test).
func (q *writeQueue) WritesSinceLastReset() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.writes
}

// SetDeadlockForTest stops the worker from running writes (a test hook:
// a write that submits is answered immediately without running — a
// read that enqueued would hang, proving reads bypass the queue).
func (q *writeQueue) SetDeadlockForTest() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.dead = true
}

// ClearDeadlockForTest reopens the queue (the test cleanup for
// SetDeadlockForTest).
func (q *writeQueue) ClearDeadlockForTest() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.dead = false
}

// Close stops the worker (the test cleanup; production closes the queue
// when the server shuts down).
func (q *writeQueue) Close() {
	q.mu.Lock()
	if !q.started {
		q.mu.Unlock()
		return
	}
	q.started = false
	q.mu.Unlock()
	close(q.ch)
	<-q.workerDone
}
