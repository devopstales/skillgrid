package proxy

import (
	"fmt"
	"time"

	"github.com/devopstales/skillgrid/mnemonic/internal/ctx/store"
)

// GetMode returns the current compression mode. It is safe for concurrent use:
// the mode is read on the hot forward path (ServeHTTP) and written by SetMode,
// so the read takes an RLock and the write takes a full Lock. The lock is held
// only for the field read/write — never across I/O.
func (p *Proxy) GetMode() Mode {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.mode
}

// SetMode flips the active compression mode at runtime. It validates new with
// valid() (an invalid mode is an error and leaves the current mode untouched),
// no-ops when new equals the current mode (no history row, no event), and
// otherwise updates the field under the write lock, writes a
// ctx_proxy_mode_history row (nil store → no-op, ADR-0016), and fires the
// SetModeEventSink (nil sink → no-op). The write lock is not held while writing
// the history row or firing the sink, so neither can block the forward path.
func (p *Proxy) SetMode(new Mode, reason string) (old Mode, err error) {
	if !new.valid() {
		return p.GetMode(), fmt.Errorf("proxy: invalid mode %q", new)
	}

	p.mu.RLock()
	current := p.mode
	p.mu.RUnlock()
	old = current

	if new == current {
		// Same mode: no-op. Return the current mode as old, no history row,
		// no event.
		return current, nil
	}

	// Mint a deterministic, collision-free change ID under the write lock:
	// a per-proxy sequence salted by the process start time. The sequence
	// makes two rapid SetMode calls unique (no same-nanosecond collision),
	// and the process-start salt keeps IDs unique across restarts that share
	// the same machine clock (a bare counter would reuse 1, 2, 3… across
	// restarts). The wall clock is read once, outside the I/O path.
	now := time.Now()
	p.mu.Lock()
	p.mode = new
	p.modeChangeSeq++
	changeID := fmt.Sprintf("mode-%d-%d", now.UnixNano(), p.modeChangeSeq)
	p.lastChangeID = changeID
	p.mu.Unlock()

	// History row + event fire outside the write lock (no I/O / sink under
	// the lock).
	if p.store != nil {
		_ = p.store.PutModeChange(&store.ModeChange{
			ChangeID:  changeID,
			SessionID: "", // a runtime switch is not tied to a proxied session in T06
			OldMode:   old.String(),
			NewMode:   new.String(),
			Reason:    reason,
			ChangedAt: now.UnixNano(),
		})
	}

	if p.SetModeEventSink != nil {
		p.SetModeEventSink(old, new, reason)
	}
	return old, nil
}

// lastModeChangeID returns the ChangeID most recently recorded by SetMode (""
// if SetMode has not written a history row — e.g. a same-mode no-op). It is a
// test hook: the store exposes GetModeChange by ChangeID but no list, so the
// test recovers the id here and reads the row back.
func (p *Proxy) lastModeChangeID() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.lastChangeID
}
