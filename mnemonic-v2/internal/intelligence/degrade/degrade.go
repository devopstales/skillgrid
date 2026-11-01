// Package degrade owns the explicit-degradation policy (INV-3, spec
// 003-intelligence §4). The policy is per-call, never global: a failing
// Embedder does not degrade Classify.
//
// Absent and failing are different states:
//   - an ABSENT seam (a nil interface value — no provider configured) runs
//     the deterministic floor immediately, with NO retry, and flags the result
//     degraded;
//   - a FAILING seam (present but erroring) gets exactly one bounded retry,
//     then the floor, and flags the result degraded.
//
// A degraded call always logs one line; there is never a silent floor.
package degrade

import (
	"context"
	"log"
)

// Seam is the call unit degrade.Run degrades. The call takes the caller's
// context plus two caller-owned arguments (nil for seams whose method takes
// no extra inputs) and returns the result as any; the caller's floor produces
// a value of the same concrete type.
//
// ponytail: the brief's sketch `Run[T any](seam T, floor func() T, name
// string)` compiles fine for a nil interface seam (with T constrained to `any`
// a nil llm.LLM is assignable, and `seam == nil` works) — the real tradeoff is
// the RESULT: this any-typed form returns `any`, so every call site must
// type-assert the result back to the floor's concrete type, whereas the
// generic form returns T directly and the assert is impossible. The controller
// kept the any form (T08+ standardizes on it). Upgrade path: a per-seam typed
// helper (RunClassify(ctx, llm.LLM, func() llm.DedupDecision, name)) that
// returns T directly, if the call-site type-asserts ever bother.
type Seam interface {
	Call(ctx context.Context, a, b any) (any, error)
}

// Run executes one seam call with the INV-3 degradation policy:
//
//   - seam == nil (absent)        → floor(), degraded=true, no retry
//   - seam errors twice (failing) → floor(), degraded=true, one bounded retry
//   - seam recovers on the retry  → recovered result, degraded=false
//   - seam succeeds on the first  → result, degraded=false
//
// ctx is the caller's context and is passed through to the seam call(s), so a
// cancelled or deadline-bound ctx propagates to the provider. A degraded call
// logs exactly one line: `degraded <name>: <reason>`. name is the call site
// label (e.g. "dedup-classify", "embed"). floor must be the deterministic
// fallback for that call (hash dedup, no vector, ...). floor returns any, but
// the concrete value it returns must be the same type the seam returns (the
// caller types both, so a mismatch is a compile-time error at the call site —
// write the floor as `func() any { return llm.DedupDecision{...} }`). The
// result is that same concrete value, wrapped in any; the caller
// type-asserts it back.
func Run(ctx context.Context, seam any, floor func() any, name string) (result any, degraded bool) {
	if seam == nil {
		result = floor()
		log.Printf("degraded %s: seam absent (no provider configured)", name)
		return result, true
	}
	s, ok := seam.(Seam)
	if !ok {
		// A present value that is not a Seam is a caller bug; treat it as
		// failing so the floor (the only safe result) runs and it is flagged.
		result = floor()
		log.Printf("degraded %s: seam value of type %T does not implement degrade.Seam", name, seam)
		return result, true
	}
	// First attempt + one bounded retry. The retry is the whole budget —
	// never a loop (spec 003-intelligence §4: bounded retry, ≤1).
	res, err := s.Call(ctx, nil, nil)
	if err != nil {
		res, err = s.Call(ctx, nil, nil)
	}
	if err != nil {
		result = floor()
		log.Printf("degraded %s: %v", name, err)
		return result, true
	}
	// res is the floor's concrete type (the seam's Call returns it); the
	// caller type-asserts it back. A mismatch is a caller bug — the floor's
	// return type is what the caller expects.
	return res, false
}
