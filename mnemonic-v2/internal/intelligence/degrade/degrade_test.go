package degrade_test

import (
	"context"
	"errors"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/intelligence/degrade"
)

// testSeam is a Seam for the degrade package's own tests.
type testSeam interface {
	Call(ctx context.Context, a, b any) (any, error)
}

type stub struct {
	f func(ctx context.Context, a, b any) (any, error)
}

func (s stub) Call(ctx context.Context, a, b any) (any, error) { return s.f(ctx, a, b) }

// [degraded-explicit] — absent seam (nil interface value) → floor runs,
// degraded=true, no retry.
func TestRunNilSeam(t *testing.T) {
	var s testSeam // nil
	calls := 0
	out, degraded := degrade.Run(s, func() any {
		calls++
		return "floor"
	}, "op")
	_ = out
	if out != "floor" || !degraded || calls != 1 {
		t.Fatalf("got out=%v degraded=%v calls=%d", out, degraded, calls)
	}
}

// [degraded-bounded-retry] — failing seam → exactly 2 calls (one retry), then
// floor + degraded.
func TestRunFailingSeamRetriesOnce(t *testing.T) {
	calls := 0
	s := stub{f: func(ctx context.Context, a, b any) (any, error) {
		calls++
		return nil, errors.New("boom")
	}}
	outAny, degraded := degrade.Run(s, func() any { return "floor" }, "op")
	out, ok := outAny.(string)
	if !ok {
		t.Fatalf("out type = %T, want string", outAny)
	}
	if out != "floor" || !degraded {
		t.Fatalf("got out=%v degraded=%v", out, degraded)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2 (initial + one bounded retry)", calls)
	}
}

// Success → the seam's result is returned, floor never runs, not degraded.
func TestRunSuccessNotDegraded(t *testing.T) {
	s := stub{f: func(ctx context.Context, a, b any) (any, error) { return "seam-result", nil }}
	floorRan := false
	outAny, degraded := degrade.Run(s, func() any {
		floorRan = true
		return "floor"
	}, "op")
	out, ok := outAny.(string)
	if !ok {
		t.Fatalf("out type = %T, want string", outAny)
	}
	if out != "seam-result" || degraded || floorRan {
		t.Fatalf("got out=%v degraded=%v floorRan=%v", out, degraded, floorRan)
	}
}

// [degraded-bounded-retry] — a seam that fails once then recovers on the
// retry returns the success after exactly 2 calls, and is NOT degraded (the
// seam recovered within its bounded budget).
func TestRunSeamRecoversOnRetry(t *testing.T) {
	calls := 0
	s := stub{f: func(ctx context.Context, a, b any) (any, error) {
		calls++
		if calls == 1 {
			return nil, errors.New("transient")
		}
		return "recovered", nil
	}}
	outAny, degraded := degrade.Run(s, func() any { return "floor" }, "op")
	out, ok := outAny.(string)
	if !ok {
		t.Fatalf("out type = %T, want string", outAny)
	}
	if out != "recovered" || degraded {
		t.Fatalf("got out=%v degraded=%v, want recovered + not degraded", out, degraded)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}
