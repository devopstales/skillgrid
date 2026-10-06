package memory

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Default budget caps (change 013, step 03) — the "Context-window flood" guard.
// A 20-hit read must never drown the context window: the item cap bounds how
// many observations are ever in-list, the char budget bounds how much text each
// in-list snippet may carry (with an explicit "N chars omitted" marker), and
// the context timeout bounds how long a read may run (a slow read is cut and
// returned as a truncated partial, never hung). All are tunable via config
// (mnemonic.retrieval_budget); the values below are the defaults.
const (
	defaultBudgetItems  = 10
	defaultBudgetChars  = 1200
	defaultBudgetTimeout = 3 * time.Second
)

// BudgetConfig is the tunable budget (change 013, step 03). Every field is
// additive and optional: a zero value falls back to the matching default. The
// timeout is expressed in nanoseconds so it is a plain JSON/YAML scalar.
type BudgetConfig struct {
	// Items caps how many in-list results a read returns. <= 0 → default.
	Items int
	// Chars caps how many characters an in-list snippet may carry. The snippet
	// is truncated with an explicit "N chars omitted" marker. <= 0 → default.
	Chars int
	// TimeoutNs is the context-timeout budget in nanoseconds. A read that
	// exceeds it is cut and returned as a truncated:true partial with a reason.
	// <= 0 → default.
	TimeoutNs int64
}

// DefaultBudget returns the default budget (item + char + timeout).
func DefaultBudget() BudgetConfig {
	return BudgetConfig{Items: defaultBudgetItems, Chars: defaultBudgetChars, TimeoutNs: int64(defaultBudgetTimeout)}
}

// normalize replaces zero/negative fields with their defaults so a partial
// config (only one knob set) still enforces the rest.
func (c BudgetConfig) normalize() BudgetConfig {
	if c.Items <= 0 {
		c.Items = defaultBudgetItems
	}
	if c.Chars <= 0 {
		c.Chars = defaultBudgetChars
	}
	if c.TimeoutNs <= 0 {
		c.TimeoutNs = int64(defaultBudgetTimeout)
	}
	return c
}

// Budget applies the item + char + timeout caps to a set of in-list reads.
// It is the uniform wrapper every mem_* read path runs through (change 013,
// step 03) so memory can never overwhelm the context window.
//
// The timeout is enforced (not just reported) by Bound: the budgeted read path
// calls Bound before the underlying query so a slow read is cut at the
// configured deadline (its QueryContext returns context.DeadlineExceeded /
// DeadlineExceeded) and Apply then surfaces that lapsed deadline as a
// truncated:true partial with reason "timeout". A read that finishes inside the
// deadline is not cut. The bound context is injectable (Bound takes the caller
// ctx) so tests can inject a tiny deadline without a real wall-clock wait.
type Budget struct {
	cfg BudgetConfig
}

// NewBudget builds a budget from cfg (zero fields → defaults).
func NewBudget(cfg BudgetConfig) *Budget {
	return &Budget{cfg: cfg.normalize()}
}

// Default returns a budget with the default caps.
func Default() *Budget {
	return NewBudget(BudgetConfig{})
}

// Config returns the effective (normalized) budget config.
func (b *Budget) Config() BudgetConfig {
	if b == nil {
		return DefaultBudget()
	}
	return b.cfg.normalize()
}

// Bound returns a context whose deadline enforces the budget's context timeout.
// The budgeted read path derives the read context from this BEFORE the query
// (so a genuinely slow read is cut, not run to completion); Apply then marks
// the result truncated:true / reason "timeout" when that deadline has lapsed.
//
// If ctx already carries a deadline that falls within the budget timeout, the
// existing tighter deadline is preserved (the smaller bound wins), so a caller
// deadline is never loosened. The returned context is a child of ctx — when the
// query (QueryContext) completes it releases its hold on the parent cancel, and
// the caller's ctx (or the handler's) is the source of truth, so no leak.
// Returns ctx unchanged when no timeout is set.
func (b *Budget) Bound(ctx context.Context) context.Context {
	if b == nil {
		return ctx
	}
	timeout := time.Duration(b.Config().TimeoutNs)
	if timeout <= 0 {
		return ctx
	}
	if dl, ok := ctx.Deadline(); ok && time.Now().Add(timeout).After(dl) {
		// The caller's deadline is tighter than the budget timeout — keep it.
		return ctx
	}
	bound, _ := context.WithTimeout(ctx, timeout)
	return bound
}

// TruncationMarker is the explicit "N chars omitted" suffix appended to a
// char-budgeted snippet. The truncation is never silent: a reader always sees
// how much was dropped and that a full-content fetch (mem_get_observation) is
// available.
const TruncationMarker = "… (%d chars omitted)"

// TruncateWithMarker truncates s to at most n characters and appends the
// explicit "N chars omitted" marker when anything was dropped. s returned
// unmodified when it fits.
func TruncateWithMarker(s string, n int) string {
	if n <= 0 {
		n = defaultBudgetChars
	}
	if len(s) <= n {
		return s
	}
	kept := len(s) - n
	return s[:n] + fmt.Sprintf(TruncationMarker, kept)
}

// BudgetResult is the output of Budget.Apply: the budgeted hits plus the
// truncated flag + reason when the read hit the char budget or the context
// timeout. Truncated is true whenever the read was cut (either way); Reason is
// human-readable and non-empty only when Truncated is true.
type BudgetResult struct {
	Hits      []Observation `json:"hits"`
	Truncated bool          `json:"truncated,omitempty"`
	Reason    string        `json:"reason,omitempty"`
	// CharsOmitted is the total characters dropped across all in-list snippets
	// by the char budget. 0 when nothing was char-truncated.
	CharsOmitted int `json:"chars_omitted,omitempty"`
}

// Apply enforces the item cap, the char budget (in-list snippets truncated with
// an explicit "N chars omitted"), and the context timeout (returns a
// truncated:true partial with a reason, never hangs).
//
// Timeout semantics: the read path bounds the query with Bound(ctx) BEFORE the
// read, so a genuinely slow read is cut at the deadline (its QueryContext
// returns context.DeadlineExceeded) and returns whatever partial it gathered.
// Apply observes that the bound deadline has lapsed and marks the result
// truncated=true / reason="timeout", returning the partial. If the read
// finished inside the deadline (deadline not yet lapsed) Apply does not cut it.
// Apply never blocks: it reads the (already-derived) deadline rather than
// waiting, so "never hangs" is enforced by the deadline-bound query, with Apply
// surfacing the outcome.
func (b *Budget) Apply(ctx context.Context, hits []Observation) BudgetResult {
	cfg := b.Config()

	// Item cap: never return more than the item budget in-list.
	out := hits
	if len(out) > cfg.Items {
		out = out[:cfg.Items]
	}

	// Char budget: truncate each in-list snippet with an explicit marker.
	truncated := false
	reason := ""
	charsOmitted := 0
	for i := range out {
		if len(out[i].Content) > cfg.Chars {
			omitted := len(out[i].Content) - cfg.Chars
			out[i].Content = TruncateWithMarker(out[i].Content, cfg.Chars)
			charsOmitted += omitted
			truncated = true
			if reason == "" {
				reason = "char-budget"
			}
		}
	}

	// Context timeout: the read was bounded by Bound(ctx) before the query. If
	// that deadline has lapsed (the read ran up against the budget and was cut
	// — e.g. a slow QueryContext returning DeadlineExceeded), mark the result a
	// truncated partial with reason "timeout". We read the deadline rather than
	// wait, so Apply itself never blocks.
	if dl, ok := ctx.Deadline(); ok && !time.Now().Before(dl) {
		truncated = true
		reason = "timeout"
	}

	// Copy so the caller's slice is not mutated in place (in-list truncation is
	// a projection, not a store rewrite — the full content stays fetchable via
	// mem_get_observation).
	res := make([]Observation, len(out))
	copy(res, out)
	return BudgetResult{Hits: res, Truncated: truncated, Reason: reason, CharsOmitted: charsOmitted}
}

// ApplyRead is the uniform budgeted-read wrapper: it runs read under the
// budget's deadline-bound context (Bound) and applies the full uniform budget
// (item cap + char budget + context timeout) to the result. A read cut by the
// deadline (its own error is a context deadline error) is surfaced as a
// truncated partial with reason "timeout" rather than a hard failure — this is
// what makes "never hangs" real: the slow read is cut and a partial returned.
// A read that fails for any other reason returns that error unchanged.
func (b *Budget) ApplyRead(ctx context.Context, read func(context.Context) ([]Observation, error)) (BudgetResult, error) {
	bctx := b.Bound(ctx)
	hits, rerr := read(bctx)
	if rerr != nil {
		if isDeadlineErr(rerr) {
			// Cut by the budget's context timeout: a truncated partial, not a
			// failure. (The partial may be empty if the read gathered nothing.)
			return b.Apply(bctx, hits), nil
		}
		return BudgetResult{}, rerr
	}
	return b.Apply(bctx, hits), nil
}

// isDeadlineErr reports whether err is a context deadline/cancellation error
// (a read cut by a deadline-bound context).
func isDeadlineErr(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)
}
