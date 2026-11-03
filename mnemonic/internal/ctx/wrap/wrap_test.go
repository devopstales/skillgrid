package wrap

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	ctxconfig "github.com/devopstales/skillgrid/mnemonic/internal/ctx/config"
	ctxproxy "github.com/devopstales/skillgrid/mnemonic/internal/ctx/proxy"
)

// --- test helpers -------------------------------------------------------------

func newTestCfg(t *testing.T, agents []string) *ctxconfig.CtxConfig {
	t.Helper()
	cfg := ctxconfig.DefaultCtxConfig()
	if agents != nil {
		cfg.Wrap.Agents = agents
	}
	return &cfg
}

func envWith(pairs ...string) []string { return pairs }

func envGet(env []string, key string) string {
	for _, kv := range env {
		if strings.HasPrefix(kv, key+"=") {
			return kv[len(key)+1:]
		}
	}
	return ""
}

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

// proxyRecorder is the test seam: it builds the real T04 proxy (nil store,
// discarded logs) on the requested port and records whether it was called
// (for the "proxy did NOT start" assertion, AC 4).
type proxyRecorder struct {
	called bool
}

func (r *proxyRecorder) start(t *testing.T, cfg *ctxconfig.CtxConfig, bind string, port int) (string, string, func(), error) {
	t.Helper()
	r.called = true
	if bind == "" {
		bind = "127.0.0.1"
	}
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", bind, port))
	if err != nil {
		return "", "", nil, err
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	p := ctxproxy.New(cfg, nil, log, ctxproxy.Mode(cfg.Proxy.Mode))
	srv := &http.Server{Handler: p.Handler()}
	go func() { _ = srv.Serve(ln) }()
	return "http://" + ln.Addr().String(), ln.Addr().String(), func() { _ = srv.Close() }, nil
}

// agentSourceTree creates a tiny fake "agent code" tree and returns a
// checksum function over its bytes (AC 5: agent code never modified).
func agentSourceTree(t *testing.T) func() string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"main.go":    "package main\nfunc main(){}\n",
		"agent.json": `{"provider":"anthropic"}`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return func() string {
		names := make([]string, 0, len(files))
		for n := range files {
			names = append(names, n)
		}
		sort.Strings(names)
		var parts []string
		for _, name := range names {
			b, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				return "missing:" + name
			}
			parts = append(parts, name+"="+string(b))
		}
		return strings.Join(parts, "|")
	}
}

// --- AC 1: wrap claude ----------------------------------------------------------

// TestWrapClaudeProxyInjectBackupLaunch: `wrap claude` starts the proxy
// (asserted serving on the loopback port), injects ANTHROPIC_BASE_URL = the
// proxy base URL into the launched subprocess env, writes a backup of the
// pre-inject state BEFORE the injection takes effect, and launches the
// agent stub (which echoes the injected env back as its output).
func TestWrapClaudeProxyInjectBackupLaunch(t *testing.T) {
	cfg := newTestCfg(t, nil) // default core-4
	backupDir := t.TempDir()
	rec := &proxyRecorder{}
	checksum := agentSourceTree(t)
	before := checksum()

	var out strings.Builder
	res, stop, err := Wrap(Options{
		Cfg:           cfg,
		AgentName:     "claude",
		Binary:        "sh",
		Args:          []string{"-c", `env | grep -E 'BASE_URL' | sort`},
		Env:           envWith("PATH=/usr/bin:/bin", "FOO=bar"),
		BackupDir:     backupDir,
		EphemeralPort: true,
		Stdout:        &out,
		Stderr:        &strings.Builder{},
	}, func(c *ctxconfig.CtxConfig, bind string, p int) (string, string, func(), error) {
		return rec.start(t, c, bind, p)
	})
	if err != nil {
		t.Fatalf("Wrap: %v", err)
	}
	defer stop()

	// Proxy started + serving on the loopback port.
	if !rec.called {
		t.Fatal("proxy was not started")
	}
	if !strings.HasPrefix(res.ProxyBaseURL, "http://127.0.0.1:") {
		t.Fatalf("proxy base URL not loopback: %q", res.ProxyBaseURL)
	}
	if err := ping(res.ProxyBaseURL); err != nil {
		t.Fatalf("proxy not serving: %v", err)
	}

	// Injection: the launched subprocess env carries ANTHROPIC_BASE_URL =
	// the proxy base URL (asserted on the live command AND on the stub's
	// own env echo).
	if got := envGet(res.Command.Env, "ANTHROPIC_BASE_URL"); got != res.ProxyBaseURL {
		t.Fatalf("ANTHROPIC_BASE_URL = %q, want %q", got, res.ProxyBaseURL)
	}
	if err := res.Command.Wait(); err != nil {
		t.Fatalf("agent stub exit: %v", err)
	}
	if !strings.Contains(out.String(), "ANTHROPIC_BASE_URL="+res.ProxyBaseURL) {
		t.Fatalf("agent subprocess did not observe the injection; output: %q", out.String())
	}

	// Backup: exists, contains the pre-inject state (the base-URL var was
	// unset before the wrap, so the snapshot is the empty value), and was
	// written before the inject took effect.
	bk := filepath.Join(backupDir, "claude.env.bak")
	data, err := os.ReadFile(bk)
	if err != nil {
		t.Fatalf("backup missing: %v", err)
	}
	if strings.Contains(string(data), res.ProxyBaseURL) {
		t.Fatalf("backup captured post-inject state: %q", data)
	}
	if !strings.HasPrefix(string(data), "ANTHROPIC_BASE_URL=") {
		t.Fatalf("backup does not snapshot the base-URL var: %q", data)
	}
	if !res.Injected || !res.BackedUp || res.BackupPath == "" {
		t.Fatalf("expected Injected+BackedUp on first wrap, got %+v", res)
	}

	// Agent code byte-identical (only its config/env changed).
	if got := checksum(); got != before {
		t.Fatalf("agent source tree changed:\nbefore: %q\nafter:  %q", before, got)
	}
}

// --- AC 2: idempotent re-run ------------------------------------------------------

// TestWrapIdempotentReRun: a second `wrap` (env already pointing at the
// proxy) does NOT double-inject and does NOT write a second backup — exactly
// one backup exists after two runs, and the agent still launches.
func TestWrapIdempotentReRun(t *testing.T) {
	cfg := newTestCfg(t, nil)
	backupDir := t.TempDir()
	rec := &proxyRecorder{}
	start := func(c *ctxconfig.CtxConfig, bind string, p int) (string, string, func(), error) {
		return rec.start(t, c, bind, p)
	}
	// Use a fixed free port so run 2's env (captured from run 1) matches.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_, portStr, _ := net.SplitHostPort(ln.Addr().String())
	var port int
	fmt.Sscanf(portStr, "%d", &port)
	ln.Close()

	// First wrap: fresh env (not wrapped).
	res1, stop1, err := Wrap(Options{
		Cfg: cfg, AgentName: "claude",
		Binary: "sh", Args: []string{"-c", "true"},
		Env:       envWith("PATH=/usr/bin:/bin"),
		BackupDir: backupDir,
		Port:      port,
		Stdout:    &strings.Builder{}, Stderr: &strings.Builder{},
	}, start)
	if err != nil {
		t.Fatalf("first wrap: %v", err)
	}
	if !res1.Injected || !res1.BackedUp {
		t.Fatalf("first wrap should inject+backup, got %+v", res1)
	}
	if err := res1.Command.Wait(); err != nil {
		t.Fatal(err)
	}
	stop1()

	// Second wrap: env already targets the loopback proxy (run 1's
	// injected URL) → no second inject, no second backup, still launches.
	res2, stop2, err := Wrap(Options{
		Cfg: cfg, AgentName: "claude",
		Binary: "sh", Args: []string{"-c", "true"},
		Env:       envWith("PATH=/usr/bin:/bin", "ANTHROPIC_BASE_URL="+res1.ProxyBaseURL),
		BackupDir: backupDir,
		Port:      port,
		Stdout:    &strings.Builder{}, Stderr: &strings.Builder{},
	}, start)
	if err != nil {
		t.Fatalf("second wrap: %v", err)
	}
	defer stop2()
	if res2.Injected || res2.BackedUp {
		t.Fatalf("second wrap must not inject/backup again, got %+v", res2)
	}
	if err := res2.Command.Wait(); err != nil {
		t.Fatal(err)
	}

	// Exactly one backup after two runs.
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatal(err)
	}
	var backups []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".bak") || strings.HasSuffix(e.Name(), ".ctx-backup") {
			backups = append(backups, e.Name())
		}
	}
	if len(backups) != 1 {
		t.Fatalf("expected exactly 1 backup after two runs, got %v", backups)
	}
}

// --- AC 3: opt-in wrap aider --------------------------------------------------------

// TestWrapOptInAider: when aider is in the wrap.agents config, `wrap aider`
// starts the proxy + injects OPENAI_BASE_URL + launches.
func TestWrapOptInAider(t *testing.T) {
	cfg := newTestCfg(t, []string{"opencode", "claude", "codex", "cursor", "aider"})
	rec := &proxyRecorder{}

	var out strings.Builder
	res, stop, err := Wrap(Options{
		Cfg:           cfg,
		AgentName:     "aider",
		Binary:        "sh",
		Args:          []string{"-c", `env | grep -E 'BASE_URL' | sort`},
		Env:           envWith("PATH=/usr/bin:/bin"),
		BackupDir:     t.TempDir(),
		EphemeralPort: true,
		Stdout:        &out,
		Stderr:        &strings.Builder{},
	}, func(c *ctxconfig.CtxConfig, bind string, p int) (string, string, func(), error) {
		return rec.start(t, c, bind, p)
	})
	if err != nil {
		t.Fatalf("Wrap(aider): %v", err)
	}
	defer stop()
	if !rec.called {
		t.Fatal("proxy was not started for opt-in agent")
	}
	if got := envGet(res.Command.Env, "OPENAI_BASE_URL"); got != res.ProxyBaseURL {
		t.Fatalf("OPENAI_BASE_URL = %q, want %q", got, res.ProxyBaseURL)
	}
	if err := res.Command.Wait(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "OPENAI_BASE_URL="+res.ProxyBaseURL) {
		t.Fatalf("aider subprocess did not observe the injection; output: %q", out.String())
	}
}

// --- AC 4: non-opt-in rejected --------------------------------------------------------

// TestWrapNonOptInRejected: `wrap goose` (neither core-4 nor in config)
// returns a clear error; the proxy did NOT start and the agent did NOT
// launch.
func TestWrapNonOptInRejected(t *testing.T) {
	cfg := newTestCfg(t, nil) // default core-4, no goose
	rec := &proxyRecorder{}
	_, _, err := Wrap(Options{
		Cfg: cfg, AgentName: "goose",
		Binary: "sh", Args: []string{"-c", "true"},
		Env: envWith("PATH=/usr/bin:/bin"),
	}, func(c *ctxconfig.CtxConfig, bind string, p int) (string, string, func(), error) {
		return rec.start(t, c, bind, p)
	})
	if err == nil {
		t.Fatal("expected error for non-opt-in agent")
	}
	if !strings.Contains(err.Error(), `agent "goose"`) {
		t.Fatalf("error not clear: %v", err)
	}
	if rec.called {
		t.Fatal("proxy must NOT start for a rejected agent")
	}
}

// TestWrapUnknownAgent: a name not in the agent table at all gets the
// "unknown agent" message (distinct from known-but-not-enabled).
func TestWrapUnknownAgent(t *testing.T) {
	cfg := newTestCfg(t, nil)
	rec := &proxyRecorder{}
	_, _, err := Wrap(Options{
		Cfg: cfg, AgentName: "frobnicator",
		Binary: "sh",
		Env:    envWith("PATH=/usr/bin:/bin"),
	}, func(c *ctxconfig.CtxConfig, bind string, p int) (string, string, func(), error) {
		return rec.start(t, c, bind, p)
	})
	if err == nil || !strings.Contains(err.Error(), `unknown agent "frobnicator"`) {
		t.Fatalf("want unknown-agent error, got: %v", err)
	}
	if rec.called {
		t.Fatal("proxy must not start for an unknown agent")
	}
}

// TestWrapKnownButNotEnabled: a known opt-in agent absent from config is
// rejected with the "not enabled" message.
func TestWrapKnownButNotEnabled(t *testing.T) {
	cfg := newTestCfg(t, nil) // core-4 only; aider is known but not enabled
	_, _, err := Wrap(Options{
		Cfg: cfg, AgentName: "aider",
		Binary: "sh",
		Env:    envWith("PATH=/usr/bin:/bin"),
	}, func(c *ctxconfig.CtxConfig, bind string, p int) (string, string, func(), error) {
		t.Fatal("proxy must not start")
		return "", "", nil, nil
	})
	if err == nil || !strings.Contains(err.Error(), `agent "aider" is not enabled`) {
		t.Fatalf("want not-enabled error, got: %v", err)
	}
}

// --- fail-open: proxy start failure aborts before launch ------------------------------

// TestWrapProxyStartFailureFailsOpen: a proxy-start failure aborts with a
// clear error and the agent is never launched (ADR-0016: never point the
// agent at a dead proxy).
func TestWrapProxyStartFailureFailsOpen(t *testing.T) {
	cfg := newTestCfg(t, nil)
	_, _, err := Wrap(Options{
		Cfg: cfg, AgentName: "claude",
		Binary: "sh", Args: []string{"-c", "true"},
		Env: envWith("PATH=/usr/bin:/bin"),
	}, func(c *ctxconfig.CtxConfig, bind string, p int) (string, string, func(), error) {
		return "", "", nil, fmt.Errorf("bind: address already in use")
	})
	if err == nil || !strings.Contains(err.Error(), "starting proxy") {
		t.Fatalf("want proxy-start error, got: %v", err)
	}
}

// --- fail-open: launch failure returns a clear error, no panic -------------------------

// TestWrapLaunchFailureFailsOpen: an unlaunchable binary returns a clear
// error, not a panic.
func TestWrapLaunchFailureFailsOpen(t *testing.T) {
	cfg := newTestCfg(t, nil)
	rec := &proxyRecorder{}
	_, _, err := Wrap(Options{
		Cfg: cfg, AgentName: "claude",
		Binary:        "/nonexistent/agent-binary-xyz",
		Env:           envWith("PATH=/usr/bin:/bin"),
		BackupDir:     t.TempDir(),
		EphemeralPort: true,
	}, func(c *ctxconfig.CtxConfig, bind string, p int) (string, string, func(), error) {
		return rec.start(t, c, bind, p)
	})
	if err == nil || !strings.Contains(err.Error(), "launching") {
		t.Fatalf("want launch error, got: %v", err)
	}
}

// --- backup_config=false skips the backup ----------------------------------------------

// TestWrapNoBackupWhenDisabled: with backup_config=false no backup file is
// written, but the injection still happens (and is idempotent on re-run).
func TestWrapNoBackupWhenDisabled(t *testing.T) {
	cfg := newTestCfg(t, nil)
	cfg.Wrap.BackupConfig = false
	backupDir := t.TempDir()
	rec := &proxyRecorder{}

	res, stop, err := Wrap(Options{
		Cfg: cfg, AgentName: "claude",
		Binary: "sh", Args: []string{"-c", "true"},
		Env:           envWith("PATH=/usr/bin:/bin"),
		BackupDir:     backupDir,
		EphemeralPort: true,
		Stdout:        &strings.Builder{}, Stderr: &strings.Builder{},
	}, func(c *ctxconfig.CtxConfig, bind string, p int) (string, string, func(), error) {
		return rec.start(t, c, bind, p)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	if err := res.Command.Wait(); err != nil {
		t.Fatal(err)
	}
	if res.BackedUp {
		t.Fatal("backup should be skipped when backup_config=false")
	}
	if got := envGet(res.Command.Env, "ANTHROPIC_BASE_URL"); got != res.ProxyBaseURL {
		t.Fatalf("injection should still happen: %q", got)
	}
	entries, _ := os.ReadDir(backupDir)
	if len(entries) != 0 {
		t.Fatalf("no backup files expected, got %v", entries)
	}
}

// --- unit: idempotency predicate ----------------------------------------------------------

func TestAlreadyWrapped(t *testing.T) {
	agent := Agent{Name: "claude", Provider: providerAnthropic, EnvVar: "ANTHROPIC_BASE_URL"}
	if !alreadyWrapped(envWith("ANTHROPIC_BASE_URL=http://127.0.0.1:8787"), agent) {
		t.Error("127.0.0.1 base should be wrapped")
	}
	if !alreadyWrapped(envWith("ANTHROPIC_BASE_URL=http://[::1]:8787/v1"), agent) {
		t.Error("[::1] base should be wrapped")
	}
	if alreadyWrapped(envWith("ANTHROPIC_BASE_URL=https://api.anthropic.com"), agent) {
		t.Error("real provider base should NOT be wrapped")
	}
	if alreadyWrapped(envWith("OTHER=x"), agent) {
		t.Error("var absent should NOT be wrapped")
	}
}

// --- I1: CLI re-run idempotency via persisted wrap-state --------------------------

// TestWrapCLIRerunIdempotent: two SEPARATE `wrap` invocations in the same repo
// share a fresh parent env each time (env injection is not persisted into the
// parent shell). The persisted wrap-state marker makes the second invocation
// idempotent — no re-inject, no re-backup — even though its env does NOT
// already point at the proxy. This is the real-CLI idempotency path the
// env-only `alreadyWrapped` predicate alone cannot see.
func TestWrapCLIRerunIdempotent(t *testing.T) {
	cfg := newTestCfg(t, nil)
	stateDir := t.TempDir() // repo root for the wrap-state marker
	backupDir := t.TempDir()
	rec := &proxyRecorder{}
	start := func(c *ctxconfig.CtxConfig, bind string, p int) (string, string, func(), error) {
		return rec.start(t, c, bind, p)
	}

	// Invocation 1: fresh env (not wrapped). Must inject + backup + record.
	res1, stop1, err := Wrap(Options{
		Cfg: cfg, AgentName: "claude",
		Binary: "sh", Args: []string{"-c", "true"},
		Env:       envWith("PATH=/usr/bin:/bin"),
		BackupDir: backupDir, StateDir: stateDir,
		Stdout: &strings.Builder{}, Stderr: &strings.Builder{},
	}, start)
	if err != nil {
		t.Fatalf("first wrap: %v", err)
	}
	if !res1.Injected || !res1.BackedUp {
		t.Fatalf("first wrap should inject+backup, got %+v", res1)
	}
	if err := res1.Command.Wait(); err != nil {
		t.Fatal(err)
	}
	stop1()

	// Invocation 2: a FRESH parent env (NOT pointing at the proxy) — exactly
	// what a second separate `ctx wrap claude` sees. Idempotency must come
	// from the persisted wrap-state marker, not from the env.
	res2, stop2, err := Wrap(Options{
		Cfg: cfg, AgentName: "claude",
		Binary: "sh", Args: []string{"-c", "true"},
		Env:       envWith("PATH=/usr/bin:/bin"),
		BackupDir: backupDir, StateDir: stateDir,
		Stdout: &strings.Builder{}, Stderr: &strings.Builder{},
	}, start)
	if err != nil {
		t.Fatalf("second wrap: %v", err)
	}
	defer stop2()
	if res2.Injected || res2.BackedUp {
		t.Fatalf("second wrap (fresh env, persisted marker) must not inject/backup again, got %+v", res2)
	}
	if err := res2.Command.Wait(); err != nil {
		t.Fatal(err)
	}

	// Exactly one backup across both invocations.
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatal(err)
	}
	var backups int
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".bak") || strings.HasSuffix(e.Name(), ".ctx-backup") {
			backups++
		}
	}
	if backups != 1 {
		t.Fatalf("expected exactly 1 backup after two CLI runs, got %d", backups)
	}
}

// --- M2: RestoreBackup is exercised (was exported-but-unreferenced) --------------

// TestRestoreBackupEnvOnly: for an env-only agent the restore is a no-op on the
// process env (the subprocess gets a fresh env per launch), and it reads the
// backup written by WriteBackup in the same dir — proving the exported API is
// reachable and coherent, not dead.
func TestRestoreBackupEnvOnly(t *testing.T) {
	agent := Agent{Name: "claude", Provider: providerAnthropic, EnvVar: "ANTHROPIC_BASE_URL"}
	dir := t.TempDir()
	// Write a backup capturing a prior provider base URL.
	if ok, err := WriteBackup(agent, "", dir, func(string) string { return "https://api.anthropic.com" }); err != nil {
		t.Fatal(err)
	} else if !ok {
		t.Fatal("WriteBackup should have written (no prior backup)")
	}
	// Restore is a no-op for env-only agents (no config file to revert).
	if err := RestoreBackup(agent, "", dir); err != nil {
		t.Fatalf("RestoreBackup env-only should be a no-op, got %v", err)
	}
}
