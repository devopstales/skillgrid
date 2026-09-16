package relay

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Watchdog env vars (change 006, step 05). The watchdog is OFF BY DEFAULT:
// with no enable gate set, Check is a no-op and never auto-hands-off. It is
// never always-on — an explicit opt-in is required.
//
//	EnvWatchdog           — enable gate. ""/0/off/false/no (case-insensitive,
//	                       whitespace-trimmed) disable; any other value enables.
//	EnvWatchdogThreshold  — the context-usage fraction (0.0–1.0) at/above
//	                       which an enabled watchdog hands off.
const (
	EnvWatchdog          = "SKILLGRID_HANDOFF_WATCHDOG"
	EnvWatchdogThreshold = "SKILLGRID_HANDOFF_WATCHDOG_THRESHOLD"
)

// isOff reports whether an enable-gate value means "off" (unset or falsy).
func isOff(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "0", "off", "false", "no":
		return true
	}
	return false
}

// WatchdogResult is the outcome of a watchdog Check. HandedOff is true only
// when the watchdog actually invoked the SAME Relay.Handoff path (a row + the
// cleave bundle were written). HandoffID/Paths mirror what Handoff returned.
type WatchdogResult struct {
	HandedOff bool
	HandoffID string
	Paths     []string
}

// Check is the optional, flag/env-gated context-limit watchdog (step 05). It
// reads the enable gate (EnvWatchdog) + threshold (EnvWatchdogThreshold) and,
// when enabled AND the current context-usage fraction is at/past the
// threshold, triggers the SAME Relay.Handoff path (this is not a
// re-implementation — it calls Handoff, so a session_handoffs row + the
// .cleave/ bundle appear exactly as they would for an operator handoff).
//
// Usage-signal decision (05.1): Check takes the usage as a caller-supplied
// fraction (usage, 0.0–1.0) rather than computing a token estimate internally.
// The fraction is a client context-`%` / token-ratio supplied by the caller —
// the same interface shape works whether the caller reports a context
// percentage or a token ratio. This keeps the watchdog dependency-free and
// hermetic: the fraction is an input, not something computed from a model, so
// tests can drive it directly and the watchdog never needs a tokenizer.
//
// Semantics:
//   - disabled (gate unset / off)  → no-op (no row, no bundle, no error).
//   - below threshold              → no-op (no row, no bundle, no error).
//   - enabled + at/past threshold  → SAME Relay.Handoff path.
//   - invalid config (threshold not a number in [0,1]) → fail closed: a clear
//     config error is returned and NO auto-handoff occurs.
func Check(ctx context.Context, db Store, projectID, projectRoot string, usage float64, b Bundle) (WatchdogResult, error) {
	// Off by default: no enable gate → no-op, never always-on.
	if isOff(os.Getenv(EnvWatchdog)) {
		return WatchdogResult{}, nil
	}

	thresholdStr := os.Getenv(EnvWatchdogThreshold)
	if strings.TrimSpace(thresholdStr) == "" {
		// Enabled but no threshold configured → invalid config, fail closed
		// (we know the operator opted in, but not at what limit).
		return WatchdogResult{}, fmt.Errorf("relay watchdog: %s is set but %s is empty; provide a threshold in [0,1]", EnvWatchdog, EnvWatchdogThreshold)
	}
	threshold, perr := strconv.ParseFloat(strings.TrimSpace(thresholdStr), 64)
	if perr != nil || threshold < 0 || threshold > 1 {
		// Malformed threshold (not a number, or out of [0,1]) → fail closed:
		// clear config error, no auto-handoff.
		return WatchdogResult{}, fmt.Errorf("relay watchdog: invalid %s %q; must be a number in [0,1]", EnvWatchdogThreshold, thresholdStr)
	}

	// Below threshold → no-op (never auto-handoff).
	if usage < threshold {
		return WatchdogResult{}, nil
	}

	// Enabled + at/past threshold → the SAME Relay.Handoff path.
	handoffID, paths, herr := Handoff(ctx, db, projectID, "", projectRoot, b)
	if herr != nil {
		return WatchdogResult{}, herr
	}
	return WatchdogResult{HandedOff: true, HandoffID: handoffID, Paths: paths}, nil
}
