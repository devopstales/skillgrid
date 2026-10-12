package proxy

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// [mode-switch-emitted] SetMode cache->token: fires the SetModeEventSink with
// (old, new, reason) and flips GetMode to the new value.
func TestSetModeEmitsEventAndFlipsMode(t *testing.T) {
	cfg := testConfig("cache", 0, 0)
	p := newTestProxy(t, cfg, nil, ModeCache, newUpstreamServer(t, http.StatusOK, `{}`, nil))

	var gotOld, gotNew Mode
	var gotReason string
	var calls int
	p.SetModeEventSink = func(old, new Mode, reason string) {
		gotOld, gotNew, gotReason = old, new, reason
		calls++
	}

	old, err := p.SetMode(ModeToken, "runtime flip")
	if err != nil {
		t.Fatalf("SetMode: %v", err)
	}
	if old != ModeCache {
		t.Fatalf("SetMode returned old = %q, want cache", old)
	}
	if m := p.GetMode(); m != ModeToken {
		t.Fatalf("GetMode() = %q, want token after switch", m)
	}
	if calls != 1 {
		t.Fatalf("event sink fired %d times, want 1", calls)
	}
	if gotOld != ModeCache || gotNew != ModeToken || gotReason != "runtime flip" {
		t.Fatalf("sink = (%q, %q, %q), want (cache, token, runtime flip)", gotOld, gotNew, gotReason)
	}
}

// [mode-switch-emitted] SetMode with a real store writes a
// ctx_proxy_mode_history row (old_mode=cache, new_mode=token) that is
// retrievable via store.GetModeChange using the ChangeID SetMode mints.
func TestSetModeWritesHistoryRow(t *testing.T) {
	cs := newSharedStore(t, "ctxmodehistory")
	cfg := testConfig("cache", 0, 0)
	p := newTestProxy(t, cfg, cs, ModeCache, newUpstreamServer(t, http.StatusOK, `{}`, nil))

	if _, err := p.SetMode(ModeToken, "record history"); err != nil {
		t.Fatalf("SetMode: %v", err)
	}

	changeID := p.lastModeChangeID()
	if changeID == "" {
		t.Fatalf("SetMode did not record a ChangeID")
	}
	row, err := cs.GetModeChange(changeID)
	if err != nil {
		t.Fatalf("GetModeChange: %v", err)
	}
	if row == nil {
		t.Fatalf("history row not found for %q", changeID)
	}
	if row.OldMode != "cache" || row.NewMode != "token" {
		t.Errorf("history row = (%q -> %q), want (cache -> token)", row.OldMode, row.NewMode)
	}
	if row.Reason != "record history" {
		t.Errorf("history reason = %q, want record history", row.Reason)
	}
	if row.ChangedAt == 0 {
		t.Errorf("history ChangedAt = 0, want a non-zero timestamp")
	}
}

// [mode-switch-emitted] SetMode to the same mode is a no-op: no history row,
// no event emission, current mode unchanged.
func TestSetModeSameModeNoop(t *testing.T) {
	cs := newSharedStore(t, "ctxmodesame")
	cfg := testConfig("token", 0, 0)
	p := newTestProxy(t, cfg, cs, ModeToken, newUpstreamServer(t, http.StatusOK, `{}`, nil))

	var sinkCalls int
	p.SetModeEventSink = func(old, new Mode, reason string) { sinkCalls++ }

	old, err := p.SetMode(ModeToken, "same mode")
	if err != nil {
		t.Fatalf("SetMode: %v", err)
	}
	if old != ModeToken {
		t.Fatalf("SetMode returned old = %q, want token", old)
	}
	if m := p.GetMode(); m != ModeToken {
		t.Fatalf("GetMode() = %q, want token (unchanged)", m)
	}
	if sinkCalls != 0 {
		t.Fatalf("event sink fired %d times, want 0 for same-mode no-op", sinkCalls)
	}
	if id := p.lastModeChangeID(); id != "" {
		t.Fatalf("lastModeChangeID = %q, want empty (no-op must not record)", id)
	}
}

// [mode-switch-emitted] SetMode with an invalid (unknown) mode returns an
// error and leaves the current mode untouched — no history row, no event.
func TestSetModeInvalidModeErrors(t *testing.T) {
	cfg := testConfig("cache", 0, 0)
	p := newTestProxy(t, cfg, nil, ModeCache, newUpstreamServer(t, http.StatusOK, `{}`, nil))

	var sinkCalls int
	p.SetModeEventSink = func(old, new Mode, reason string) { sinkCalls++ }

	_, err := p.SetMode(Mode("bogus"), "bad mode")
	if err == nil {
		t.Fatalf("SetMode(bogus) = nil error, want an error for an invalid mode")
	}
	if m := p.GetMode(); m != ModeCache {
		t.Fatalf("GetMode() = %q, want cache (unchanged after invalid switch)", m)
	}
	if sinkCalls != 0 {
		t.Fatalf("event sink fired %d times, want 0 for invalid mode", sinkCalls)
	}
}

// [mode-switch-emitted] Nil store: SetMode still flips the mode and emits the
// event; the history write is a no-op (no 500 / panic).
func TestSetModeNilStoreNoop(t *testing.T) {
	cfg := testConfig("cache", 0, 0)
	p := newTestProxy(t, cfg, nil, ModeCache, newUpstreamServer(t, http.StatusOK, `{}`, nil))

	var gotOld, gotNew Mode
	p.SetModeEventSink = func(old, new Mode, reason string) { gotOld, gotNew = old, new }

	old, err := p.SetMode(ModeToken, "nil store switch")
	if err != nil {
		t.Fatalf("SetMode with nil store: %v", err)
	}
	if old != ModeCache {
		t.Fatalf("SetMode returned old = %q, want cache", old)
	}
	if m := p.GetMode(); m != ModeToken {
		t.Fatalf("GetMode() = %q, want token (nil store must still switch)", m)
	}
	if gotOld != ModeCache || gotNew != ModeToken {
		t.Fatalf("sink = (%q, %q), want (cache, token)", gotOld, gotNew)
	}
}

// [mode-switch-emitted] Restart-to-mode: a FRESH proxy whose mode is resolved
// from the CTX_PROXY_MODE env var (token) while the config says cache starts
// in token mode — the env var wins on a clean construction (AC 4).
func TestRestartToModeFromEnv(t *testing.T) {
	t.Setenv("CTX_PROXY_MODE", "token")
	cfg := testConfig("cache", 0, 0) // config says cache, but the env must win
	// Build the proxy exactly as a fresh `ctx proxy serve` would: the mode is
	// resolved from the real environment, not hard-coded.
	p := New(cfg, nil, testLogger(), ResolveMode("", "cache", os.Getenv))

	if m := p.GetMode(); m != ModeToken {
		t.Fatalf("GetMode() = %q, want token (CTX_PROXY_MODE must win over config)", m)
	}
}

// httptest is referenced so the import is used when the upstream server is
// constructed in tests above.
var _ = httptest.NewServer
