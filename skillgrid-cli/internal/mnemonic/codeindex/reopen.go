package codeindex

import (
	"os"
	"sync"
	"time"
)

// DefaultReopenWindow is how long a running process waits before it considers a
// published index "old enough" to reopen on its next tool call (~5s; no
// restart needed to see a just-made edit). Injectable in tests.
const DefaultReopenWindow = 5 * time.Second

// ReopenTracker lets a running process detect that a published index is newer
// than the one it currently holds, and reopen it on its next tool call (no
// restart). It compares the live index file's modification time against the
// timestamp recorded when the process last opened it.
type ReopenTracker struct {
	mu             sync.Mutex
	livePath       string
	openedAt       time.Time
	reopenWindow   time.Duration
	now            func() time.Time
	// OnReopen is invoked when a reopen is due; it receives the live path so
	// the caller can open the fresh index. A nil OnReopen means "just mark
	// reopened".
	OnReopen func(livePath string)
}

// NewReopenTracker returns a tracker for livePath. reopenWindow <= 0 uses
// DefaultReopenWindow.
func NewReopenTracker(livePath string, reopenWindow time.Duration) *ReopenTracker {
	if reopenWindow <= 0 {
		reopenWindow = DefaultReopenWindow
	}
	return &ReopenTracker{
		livePath:     livePath,
		openedAt:     time.Now(),
		reopenWindow: reopenWindow,
		now:          time.Now,
	}
}

// Check inspects the live index file. If it has been modified since the tracker
// last opened it (and the reopen window has elapsed since that open), it fires
// OnReopen, re-records the open time, and returns true. Otherwise it returns
// false. A missing file is treated as "no reopen needed".
func (r *ReopenTracker) Check() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	info, err := os.Stat(r.livePath)
	if err != nil {
		return false
	}
	// The published index is newer than what we opened, and the window since
	// our open has elapsed → reopen.
	if info.ModTime().After(r.openedAt) && r.now().Sub(r.openedAt) >= r.reopenWindow {
		r.openedAt = r.now()
		if r.OnReopen != nil {
			r.OnReopen(r.livePath)
		}
		return true
	}
	return false
}

// OpenedAt returns the time the tracker last opened (or last reopened) the
// index.
func (r *ReopenTracker) OpenedAt() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.openedAt
}

// ReopenWindow returns the effective reopen window.
func (r *ReopenTracker) ReopenWindow() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.reopenWindow
}
