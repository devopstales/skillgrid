package wrap

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	ctxconfig "github.com/devopstales/skillgrid/mnemonic/internal/ctx/config"
	ctxproxy "github.com/devopstales/skillgrid/mnemonic/internal/ctx/proxy"
	ctxstore "github.com/devopstales/skillgrid/mnemonic/internal/ctx/store"
	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// --- T08 E2E: launcher + proxy + store end-to-end --------------------------------
//
// TestWrapE2ETurnThroughProxy is the acceptance end-to-end proof (plan 16.10):
// a wrapped agent completes a FULL turn through the real loopback proxy, the
// proxy's savings header rides back on the response, the turn lands a
// telemetry row on the real shared store, and the agent's source tree is
// byte-identical before and after. Nothing is stubbed except the agent
// binary (a shell script) and the LLM provider (an httptest upstream) — the
// proxy, the store, the launcher, the listener, and the subprocess are all
// real.

// stubAgentEnvAgent is the fake agent: a one-shot shell command that reads
// the injected base-URL env var, POSTs one chat-completion turn to the
// proxy, and prints the proxy's status code and savings header so the test
// can assert on what the agent actually saw.
func stubAgentEnvAgent(baseVar string) []string {
	return []string{
		"-c",
		`set -e; hdrs=$(mktemp); trap 'rm -f "$hdrs"' EXIT; ` +
			`curl -sS -o /dev/null -D "$hdrs" ` +
			`-H 'Content-Type: application/json' ` +
			`-H 'x-ctx-session: e2e-sess' ` +
			`-d '{"messages":[{"role":"user","content":"the quick brown"},{"role":"assistant","content":"fox jumps over"},{"role":"user","content":"the lazy dog"}]}' ` +
			`"$` + baseVar + `"/v1/messages; ` +
			`code=$(head -n1 "$hdrs" | awk '{print $2}'); ` +
			`sav=$(grep -i '^x-ctx-savings:' "$hdrs" | awk '{print $2}' | tr -d '\r'); ` +
			`echo "$code $sav"`,
	}
}

// realProxyStoreStartProxy builds the REAL T04 proxy wired to the REAL
// shared store (telemetry on) and serves it on an ephemeral loopback
// listener — the same shape DefaultStartProxy uses, plus the store handle
// the assertions read from. It returns the StartProxy seam closure.
//
// The proxy's upstream is resolved by detectUpstream from the PROCESS env
// (ANTHROPIC_BASE_URL) at construction time, so we point the process env at
// the httptest upstream before building the proxy and restore it on cleanup.
// This is process-global, so the E2E test runs without -parallel (the test
// binary runs serially by default; other wrap tests never read
// ANTHROPIC_BASE_URL from the process env, only from the env slice they pass
// to Wrap, so there is no cross-test interference).
func realProxyStoreStartProxy(
	t *testing.T,
	cfg *ctxconfig.CtxConfig,
	cs *ctxstore.Store,
	upstream *httptest.Server,
) func(c *ctxconfig.CtxConfig, bind string, port int) (string, string, func(), error) {
	t.Helper()
	prevBase, hadBase := os.LookupEnv("ANTHROPIC_BASE_URL")
	_ = os.Setenv("ANTHROPIC_BASE_URL", upstream.URL)
	t.Cleanup(func() {
		if hadBase {
			_ = os.Setenv("ANTHROPIC_BASE_URL", prevBase)
		} else {
			_ = os.Unsetenv("ANTHROPIC_BASE_URL")
		}
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("e2e: listening on loopback: %v", err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	p := ctxproxy.New(cfg, cs, log, ctxproxy.Mode("token"))
	srv := &http.Server{Handler: p.Handler()}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	return func(c *ctxconfig.CtxConfig, bind string, port int) (string, string, func(), error) {
		addr := ln.Addr().String()
		base := "http://" + addr
		return base, addr, func() { _ = srv.Close() }, nil
	}
}

// TestWrapE2ETurnThroughProxy: wrap claude → the stub agent POSTs one turn
// to the injected proxy base URL → the proxy (token mode, savings headers
// on, real shared store) transforms, forwards to the httptest upstream,
// writes the savings headers + a telemetry row, and the agent's source tree
// is byte-identical before and after the whole thing.
func TestWrapE2ETurnThroughProxy(t *testing.T) {
	// The shared store (real, migrated) the proxy writes telemetry into.
	shared, err := store.Open(t.TempDir(), "ctxwrap-e2e")
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = shared.Close() })
	cs := ctxstore.New(shared)
	if cs == nil {
		t.Fatal("ctxstore.New returned nil for a valid shared store")
	}

	// The fake LLM provider: a valid chat-completion response, 200, and a
	// record of the bodies it saw (to prove the transform reached it).
	var upstreamBodies []string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		upstreamBodies = append(upstreamBodies, string(b))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"type":"message","role":"assistant","content":"ok"}`))
	}))
	t.Cleanup(upstream.Close)

	cfg := newTestCfg(t, nil)
	cfg.Proxy.Mode = "token"
	cfg.Proxy.SavingsHeaders = true
	cfg.Proxy.ProviderRetries = 0
	cfg.Proxy.ProviderRetryBackoffMs = 0

	start := realProxyStoreStartProxy(t, cfg, cs, upstream)

	checksum := agentSourceTree(t)
	before := checksum()

	var out strings.Builder
	var agentErr strings.Builder
	res, stop, err := Wrap(Options{
		Cfg:           cfg,
		AgentName:     "claude",
		Binary:        "/bin/bash",
		Args:          stubAgentEnvAgent("ANTHROPIC_BASE_URL"),
		Env:           envWith("PATH=" + os.Getenv("PATH")),
		BackupDir:     t.TempDir(),
		StateDir:      t.TempDir(),
		EphemeralPort: true,
		Stdout:        &out,
		Stderr:        &agentErr,
	}, start)
	if err != nil {
		t.Fatalf("Wrap: %v", err)
	}
	defer stop()

	if got := envGet(res.Command.Env, "ANTHROPIC_BASE_URL"); got != res.ProxyBaseURL {
		t.Fatalf("injected ANTHROPIC_BASE_URL = %q, want proxy base %q", got, res.ProxyBaseURL)
	}

	if err := res.Command.Wait(); err != nil {
		t.Fatalf("agent stub exit: %v (stderr: %s)", err, agentErr.String())
	}

	// 1. The agent's turn completed through the proxy: it saw a 2xx from the
	//    upstream and the savings header on the response.
	fields := strings.Fields(out.String())
	if len(fields) != 2 {
		t.Fatalf("agent output = %q, want '<status> <savings>'", out.String())
	}
	status, savings := fields[0], fields[1]
	if status != "200" {
		t.Fatalf("proxy status = %s, want 200", status)
	}
	if savings == "" || savings == "0" {
		t.Fatalf("X-CTX-Savings = %q, want a non-zero token-mode delta", savings)
	}

	// 2. The upstream received the TRANSFORMED body (token mode compresses
	//    every message's content to the deterministic summary marker).
	if len(upstreamBodies) != 1 {
		t.Fatalf("upstream saw %d request(s), want exactly 1", len(upstreamBodies))
	}
	if !strings.Contains(upstreamBodies[0], "[ctx-summarized") {
		t.Fatalf("upstream body not summarized by the proxy: %q", upstreamBodies[0])
	}

	// 3. Telemetry landed on the real shared store for the session the stub
	//    tagged its request with.
	if err := waitForStore(t, cs, "e2e-sess"); err != nil {
		t.Fatalf("telemetry: %v", err)
	}
	savingsRows, err := cs.ListProxySavings("e2e-sess")
	if err != nil {
		t.Fatalf("ListProxySavings: %v", err)
	}
	if len(savingsRows) != 1 {
		t.Fatalf("savings rows = %d, want 1", len(savingsRows))
	}
	if savingsRows[0].TokensAfter >= savingsRows[0].TokensBefore {
		t.Errorf("token mode must compress: before=%d after=%d", savingsRows[0].TokensBefore, savingsRows[0].TokensAfter)
	}
	sess, err := cs.GetProxySession("e2e-sess")
	if err != nil {
		t.Fatalf("GetProxySession: %v", err)
	}
	if sess == nil {
		t.Fatal("no session row for e2e-sess")
	}
	if sess.Requests != 1 {
		t.Errorf("session requests = %d, want 1", sess.Requests)
	}

	// 4. The agent's source tree is byte-identical: the wrap only changed the
	//    agent's env, never its code.
	if got := checksum(); got != before {
		t.Fatalf("agent source tree changed:\nbefore: %q\nafter:  %q", before, got)
	}
}

// waitForStore polls the store until the session's savings row is visible
// (recordTelemetry runs after the response is written, so the write can
// lag the client's read by a few ms).
func waitForStore(t *testing.T, cs *ctxstore.Store, sessionID string) error {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		rows, err := cs.ListProxySavings(sessionID)
		if err != nil {
			return err
		}
		if len(rows) >= 1 {
			return nil
		}
		_, _ = cs.GetProxySession(sessionID)
		time.Sleep(25 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for telemetry row for session %q", sessionID)
}
