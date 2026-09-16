package relay

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWatchdog (05.2, RED) — enabled watchdog past threshold triggers the SAME
// Handoff path. The watchdog's Check must actually invoke Relay.Handoff: a
// session_handoffs row AND the three .cleave/ files must appear (not just a
// bool). The enable gate (SKILLGRID_HANDOFF_WATCHDOG) + threshold
// (SKILLGRID_HANDOFF_WATCHDOG_THRESHOLD) come from env.
func TestWatchdog(t *testing.T) {
	t.Setenv(EnvWatchdog, "1")
	t.Setenv(EnvWatchdogThreshold, "0.8")

	st, root := openStore(t, "relayproj")
	seedSession(t, st, "s-wd")

	res, err := Check(context.Background(), st.DB, "relayproj", root, 0.95, Bundle{
		Progress:      "watchdog handoff",
		NextPrompt:    "resume after context limit",
		SourceSession: "s-wd",
	})
	if err != nil {
		t.Fatalf("watchdog check past threshold: %v", err)
	}
	if !res.HandedOff {
		t.Fatalf("expected watchdog to hand off when enabled and past threshold")
	}
	if res.HandoffID == "" {
		t.Errorf("expected a handoff_id to be returned")
	}
	if len(res.Paths) != 3 {
		t.Fatalf("expected 3 cleave paths from the same Handoff path, got %d", len(res.Paths))
	}
	for _, p := range res.Paths {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected cleave file on disk (same Handoff path): %v", err)
		}
	}
	// The SAME Handoff path was invoked → a session_handoffs row exists.
	if n := handoffRowCount(t, st, "relayproj"); n != 1 {
		t.Fatalf("expected 1 session_handoffs row (same Handoff path), got %d", n)
	}
}

// TestWatchdogDisabled (05.3) — no enable gate set (default) → no-op. NO
// handoff row, NO cleave bundle, no error.
func TestWatchdogDisabled(t *testing.T) {
	// Ensure the gate is unset (default state = off).
	t.Setenv(EnvWatchdog, "")
	t.Setenv(EnvWatchdogThreshold, "0.8")

	st, root := openStore(t, "relayproj")
	seedSession(t, st, "s-wd-disabled")

	res, err := Check(context.Background(), st.DB, "relayproj", root, 0.99, Bundle{
		Progress:      "should not hand off",
		NextPrompt:    "never reached",
		SourceSession: "s-wd-disabled",
	})
	if err != nil {
		t.Fatalf("disabled watchdog must be a no-op, got: %v", err)
	}
	if res.HandedOff {
		t.Fatalf("disabled watchdog must NOT hand off (never auto-handoff)")
	}
	if n := handoffRowCount(t, st, "relayproj"); n != 0 {
		t.Fatalf("expected 0 session_handoffs rows when disabled, got %d", n)
	}
	if _, err := os.Stat(filepath.Join(root, ".skillgrid", ".cleave", FileNextPrompt)); !os.IsNotExist(err) {
		t.Errorf("expected no cleave bundle when disabled (stat err=%v)", err)
	}
}

// TestWatchdogDefault (05.3) — no env at all (the default state) → no-op, even
// at a usage fraction far past any sensible threshold. Proves it is never
// always-on.
func TestWatchdogDefault(t *testing.T) {
	// Belt-and-suspenders: clear both env vars to the default (unset) state.
	os.Unsetenv(EnvWatchdog)
	os.Unsetenv(EnvWatchdogThreshold)
	t.Setenv(EnvWatchdog, "")
	t.Setenv(EnvWatchdogThreshold, "")

	st, root := openStore(t, "relayproj")
	seedSession(t, st, "s-wd-default")

	res, err := Check(context.Background(), st.DB, "relayproj", root, 0.999, Bundle{
		Progress:   "default no-op",
		NextPrompt: "never reached",
	})
	if err != nil {
		t.Fatalf("default (no env) watchdog must be a no-op, got: %v", err)
	}
	if res.HandedOff {
		t.Fatalf("default watchdog must NOT hand off — never always-on")
	}
	if n := handoffRowCount(t, st, "relayproj"); n != 0 {
		t.Fatalf("expected 0 rows in default state, got %d", n)
	}
	if _, err := os.Stat(filepath.Join(root, ".skillgrid", ".cleave", FileNextPrompt)); !os.IsNotExist(err) {
		t.Errorf("expected no cleave bundle in default state (stat err=%v)", err)
	}
}

// TestWatchdogBelow (05.3) — enabled but the usage fraction is below the
// threshold → no-op. NO handoff row, NO cleave bundle, no error.
func TestWatchdogBelow(t *testing.T) {
	t.Setenv(EnvWatchdog, "1")
	t.Setenv(EnvWatchdogThreshold, "0.8")

	st, root := openStore(t, "relayproj")
	seedSession(t, st, "s-wd-below")

	res, err := Check(context.Background(), st.DB, "relayproj", root, 0.5, Bundle{
		Progress:      "below threshold",
		NextPrompt:    "never reached",
		SourceSession: "s-wd-below",
	})
	if err != nil {
		t.Fatalf("below-threshold watchdog must be a no-op, got: %v", err)
	}
	if res.HandedOff {
		t.Fatalf("below-threshold watchdog must NOT hand off")
	}
	if n := handoffRowCount(t, st, "relayproj"); n != 0 {
		t.Fatalf("expected 0 rows when below threshold, got %d", n)
	}
	if _, err := os.Stat(filepath.Join(root, ".skillgrid", ".cleave", FileNextPrompt)); !os.IsNotExist(err) {
		t.Errorf("expected no cleave bundle when below threshold (stat err=%v)", err)
	}
}

// TestWatchdogInvalidConfig (05.4) — a malformed threshold fails closed: a
// clear config error, and NO auto-handoff (no row, no bundle).
func TestWatchdogInvalidConfig(t *testing.T) {
	t.Setenv(EnvWatchdog, "1")
	t.Setenv(EnvWatchdogThreshold, "not-a-number")

	st, root := openStore(t, "relayproj")
	seedSession(t, st, "s-wd-invalid")

	res, err := Check(context.Background(), st.DB, "relayproj", root, 0.99, Bundle{
		Progress:      "invalid config",
		NextPrompt:    "never reached",
		SourceSession: "s-wd-invalid",
	})
	if err == nil {
		t.Fatalf("expected a config error for a malformed threshold")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "threshold") {
		t.Errorf("config error should name the threshold, got: %v", err)
	}
	if res.HandedOff {
		t.Fatalf("invalid config must fail closed — no auto-handoff")
	}
	if n := handoffRowCount(t, st, "relayproj"); n != 0 {
		t.Fatalf("expected 0 rows when config is invalid (fail closed), got %d", n)
	}
	if _, err := os.Stat(filepath.Join(root, ".skillgrid", ".cleave", FileNextPrompt)); !os.IsNotExist(err) {
		t.Errorf("expected no cleave bundle when config is invalid (stat err=%v)", err)
	}
}
