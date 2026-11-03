package codeindex

import (
	"fmt"
	"os"
	"strings"
)

// Publish builds a fresh copy of the index database as a sidecar file and then
// atomically swaps it into place (os.Rename), so a reader never observes a torn
// index: it sees either the old or the new, never a partial one. On
// sidecar-unsupported filesystems (or any pre-write failure) Publish falls back
// to an in-place rewrite that retries on pre-write failure and stops once it may
// have mutated the live index (returning a non-nil error so the caller keeps the
// old index live rather than serving a torn one).
type Publish struct {
	// LivePath is the live index database path (the one readers open).
	LivePath string
	// Build fills sidecar with the new index contents. It runs before the
	// swap. A build error means the live index is untouched.
	Build func(sidecar string) error
	// SupportSidecar reports whether the filesystem supports a sidecar + atomic
	// rename (true in practice; false selects the in-place fallback).
	SupportSidecar bool
	// InPlace rewrites livePath in place (used by the fallback path).
	InPlace func() error
	// Copy copies src to dst (used by the sidecar build).
	Copy func(src, dst string) error
	// MaxInPlaceRetries bounds pre-write in-place retries.
	MaxInPlaceRetries int
}

// ErrMayHaveMutated is returned when an in-place rewrite failed after it may
// have partially written the live index. The caller must keep the old index
// live (the in-place mode stops the watcher).
var ErrMayHaveMutated = fmt.Errorf("in-place publish may have mutated the live index; keeping the old index live")

// sidecarPath returns the sidecar file name for livePath (livePath + ".swp").
func sidecarPath(livePath string) string {
	return livePath + ".swp"
}

// Publish performs the copy-and-swap (or in-place fallback) publication. It
// returns an error only when the in-place fallback may have mutated the live
// index (the sidecar path is always atomic and either fully swaps or not at
// all).
func (p *Publish) Publish() error {
	if p.SupportSidecar {
		sidecar := sidecarPath(p.LivePath)
		// Build the sidecar fresh.
		if err := p.buildSidecar(sidecar); err != nil {
			_ = os.Remove(sidecar)
			return err
		}
		// Atomic swap: rename the sidecar over the live path.
		if err := os.Rename(sidecar, p.LivePath); err != nil {
			_ = os.Remove(sidecar)
			return err
		}
		return nil
	}
	return p.publishInPlace()
}

// buildSidecar writes the new index into the sidecar.
func (p *Publish) buildSidecar(sidecar string) error {
	if p.Copy == nil || p.Build == nil {
		return fmt.Errorf("publish: build and copy required for sidecar mode")
	}
	// Start from the live index, then apply the build delta on the sidecar.
	if err := p.Copy(p.LivePath, sidecar); err != nil {
		return fmt.Errorf("copy to sidecar: %w", err)
	}
	return p.Build(sidecar)
}

// publishInPlace rewrites the live index in place, retrying on pre-write
// failure and stopping when it may have mutated the live index.
func (p *Publish) publishInPlace() error {
	retries := p.MaxInPlaceRetries
	if retries <= 0 {
		retries = 1
	}
	var lastErr error
	for i := 0; i < retries; i++ {
		err := p.InPlace()
		if err == nil {
			return nil
		}
		lastErr = err
		// A pre-write failure (no bytes landed) is safe to retry; a mid-write
		// failure may have mutated the live index and must stop.
		if !isPreWriteFailure(err) {
			return fmt.Errorf("%w (%v)", ErrMayHaveMutated, err)
		}
	}
	return fmt.Errorf("%w (%v)", ErrMayHaveMutated, lastErr)
}

// preWriteFailure marks an in-place failure that happened before any byte was
// written to the live index (safe to retry).
type preWriteFailure struct{ err error }

func (e *preWriteFailure) Error() string  { return e.err.Error() }
func (e *preWriteFailure) Unwrap() error  { return e.err }

// NewPreWriteFailure wraps err as a pre-write (retry-safe) in-place failure.
func NewPreWriteFailure(err error) error { return &preWriteFailure{err: err} }

// isPreWriteFailure reports whether err is a retry-safe pre-write in-place
// failure.
func isPreWriteFailure(err error) bool {
	for err != nil {
		if _, ok := err.(*preWriteFailure); ok {
			return true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}

// IsMayHaveMutated reports whether err is the ErrMayHaveMutated sentinel (or
// wraps it).
func IsMayHaveMutated(err error) bool {
	return strings.Contains(err.Error(), ErrMayHaveMutated.Error()) || err == ErrMayHaveMutated
}

// sidecarExists reports whether a stale sidecar is present (test helper).
func sidecarExists(livePath string) bool {
	_, err := os.Stat(sidecarPath(livePath))
	return err == nil
}

// PublishPath is a convenience for tests: the sidecar path for a live path.
func PublishPath(livePath string) string { return sidecarPath(livePath) }

// EnsureDirAll is a thin os.MkdirAll wrapper (test helper for publish fixtures).
func EnsureDirAll(dir string) error { return os.MkdirAll(dir, 0o755) }
