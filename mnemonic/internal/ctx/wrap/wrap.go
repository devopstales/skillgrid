package wrap

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"

	ctxconfig "github.com/devopstales/skillgrid/mnemonic/internal/ctx/config"
	ctxproxy "github.com/devopstales/skillgrid/mnemonic/internal/ctx/proxy"
)

// Options configures one Wrap run.
type Options struct {
	// Cfg is the resolved CTX config (wrap.agents / proxy bind+port).
	Cfg *ctxconfig.CtxConfig
	// AgentName is the agent to wrap ("claude", "aider", ...).
	AgentName string
	// Binary is the executable to launch as the agent (tests inject a
	// stub; the CLI resolves the real binary name from the agent table).
	Binary string
	// Args are passed after Binary.
	Args []string
	// Env is the parent environment for the subprocess (tests inject a
	// controlled env; the CLI passes os.Environ()).
	Env []string
	// BackupDir overrides the default backup dir (tests use t.TempDir()).
	BackupDir string
	// Port overrides the proxy port; 0 means "use the configured port
	// (cfg.Proxy.Port, default 8787)". Tests that need an ephemeral port
	// pass EphemeralPort.
	Port int
	// EphemeralPort forces an ephemeral loopback port (bound as :0)
	// regardless of Port; the actual port comes back in Result.Port.
	EphemeralPort bool
	// Bind is the proxy bind address (default cfg.Proxy.Bind, 127.0.0.1).
	Bind string
	// Log receives the launcher's log lines; nil discards them.
	Log *slog.Logger

	// Stdout/Stderr for the launched agent; nil → os.Stdout/os.Stderr.
	Stdout io.Writer
	Stderr io.Writer
}

// Result is the outcome of a Wrap run.
type Result struct {
	// Agent is the resolved agent.
	Agent Agent
	// ProxyBaseURL is the loopback base URL the agent was pointed at.
	ProxyBaseURL string
	// Port is the port the proxy is serving on (useful for ephemeral).
	Port int
	// Injected is true when this run wrote the injection (first wrap);
	// false on an idempotent re-run (already wrapped).
	Injected bool
	// BackedUp is true when this run wrote the backup file (first wrap).
	BackedUp bool
	// BackupPath is where the pre-inject state lives ("", if
	// backup_config is disabled).
	BackupPath string
	// Command is the launched subprocess; the caller (or the CLI) waits
	// on it. Tests use it to capture the subprocess env.
	Command *exec.Cmd
}

// StartProxy is a test/CLI seam for launching the proxy server: it is
// called with the resolved cfg, bind, and port (0 = ephemeral) and must
// return the serving base URL, the listener's actual address, and a stop
// func. The default implementation reuses the T04 build: proxy.New +
// net/http on a loopback listener.
type StartProxy func(cfg *ctxconfig.CtxConfig, bind string, port int) (baseURL, addr string, stop func(), err error)

// DefaultStartProxy builds and serves the loopback proxy (same build the
// `ctx proxy` CLI uses: ctxproxy.New + http.Server). With port 0 it binds
// an ephemeral loopback port and returns the actual address. Telemetry is
// a no-op (nil store): wrap owns the lifecycle, not the per-turn history
// (the proxy's own serve path records it when a store is wired).
func DefaultStartProxy(cfg *ctxconfig.CtxConfig, bind string, port int) (string, string, func(), error) {
	if bind == "" {
		bind = cfg.Proxy.Bind
	}
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", bind, port))
	if err != nil {
		return "", "", nil, fmt.Errorf("wrap: listening on %s:%d: %w", bind, port, err)
	}
	mode := ctxproxy.Mode(cfg.Proxy.Mode)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	p := ctxproxy.New(cfg, nil, log, mode)
	srv := &http.Server{Handler: p.Handler()}
	go func() { _ = srv.Serve(ln) }()
	stop := func() {
		_ = srv.Close()
	}
	addr := ln.Addr().String()
	base := "http://" + addr
	return base, addr, stop, nil
}

// Wrap runs the tier-2 launcher: resolve the agent (core-4 always, others
// opt-in via cfg.Wrap.Agents), start the proxy, back up the pre-inject
// state (first wrap only), inject the base-URL env, and launch the agent
// subprocess. It never modifies the agent's code.
//
// Idempotency: if the subprocess env already targets the loopback proxy
// (alreadyWrapped), the injection and backup are skipped (no double-inject,
// no double-backup) but the agent still launches.
//
// Fail-open (ADR-0016): a proxy-start failure aborts BEFORE launch with a
// clear error (never point the agent at a dead proxy); a launch failure
// returns the error from Command.Start (no panic). The proxy is stopped
// automatically when the returned stopProxy func is called; Wrap returns
// stopProxy via the result so the caller owns the lifecycle (the CLI keeps
// it alive for the subprocess lifetime).
func Wrap(opts Options, startProxy StartProxy) (Result, func(), error) {
	if opts.Cfg == nil {
		return Result{}, nil, fmt.Errorf("wrap: nil config")
	}
	if opts.AgentName == "" {
		return Result{}, nil, fmt.Errorf("wrap: agent name required")
	}
	if startProxy == nil {
		startProxy = DefaultStartProxy
	}
	log := opts.Log
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	// 1. Resolve the agent: core-4 always; opt-in via cfg.Wrap.Agents.
	agent, err := resolveAgent(opts.AgentName, opts.Cfg.Wrap.Agents)
	if err != nil {
		return Result{}, nil, err
	}
	res := Result{Agent: *agent}

	// 2. Start the proxy BEFORE anything else (a dead proxy must never be
	//    pointed at). Ephemeral port (0) is allowed for tests.
	bind := opts.Bind
	if bind == "" {
		bind = opts.Cfg.Proxy.Bind
	}
	port := opts.Cfg.Proxy.Port
	if opts.Port != 0 {
		port = opts.Port
	}
	if opts.EphemeralPort {
		port = 0
	}
	base, addr, stop, err := startProxy(opts.Cfg, bind, port)
	if err != nil {
		if stop != nil {
			stop()
		}
		return Result{}, nil, fmt.Errorf("wrap: starting proxy: %w", err)
	}
	res.ProxyBaseURL = base
	res.Port = portFromAddr(addr)

	// The launcher stops the proxy on any error path from here on; on
	// success the caller takes ownership via the returned stop func.
	stopped := false
	defer func() {
		if !stopped {
			stop()
		}
	}()

	// Confirm the proxy actually serves (loopback liveness) before the
	// agent is pointed at it.
	if err := ping(base); err != nil {
		return res, nil, fmt.Errorf("wrap: proxy not serving at %s: %w", base, err)
	}

	// 3. Idempotency: already wrapped? Skip inject + backup, still launch.
	if alreadyWrapped(opts.Env, *agent) {
		log.Info("wrap: already wrapped, skipping inject+backup", "agent", agent.Name)
		res.Injected = false
		res.BackedUp = false
		cmd, err := launch(opts, *agent, base)
		if err != nil {
			return res, nil, err
		}
		res.Command = cmd
		stopped = true
		return res, stop, nil
	}

	// 4. Backup the pre-inject state BEFORE it takes effect (first wrap
	//    only; a pre-existing backup is left untouched).
	if opts.Cfg.Wrap.BackupConfig {
		backedUp, err := writeBackupWithDir(opts, *agent)
		if err != nil {
			return res, nil, err
		}
		res.BackedUp = backedUp
		res.BackupPath = backupPath(*agent, configPathFor(*agent))
		if !backedUp {
			// A backup already exists from a prior wrap: treat as
			// already wrapped (no double-inject either).
			log.Info("wrap: backup exists, skipping inject", "agent", agent.Name)
			cmd, err := launch(opts, *agent, base)
			if err != nil {
				return res, nil, err
			}
			res.Command = cmd
			stopped = true
			return res, stop, nil
		}
	}

	// 5. Inject: the subprocess env gets the base-URL var pointed at the
	//    proxy (config files are untouched — env injection only, T07).
	res.Injected = true

	// 6. Launch the agent subprocess with the injected env.
	cmd, err := launch(opts, *agent, base)
	if err != nil {
		return res, nil, err
	}
	res.Command = cmd
	stopped = true
	return res, stop, nil
}

// launch builds (and starts) the agent subprocess with the injected env.
// Fail-open: Start errors are returned, never panicked.
func launch(opts Options, agent Agent, base string) (*exec.Cmd, error) {
	binary := opts.Binary
	if binary == "" {
		binary = agent.Name
	}
	cmd := exec.Command(binary, opts.Args...)
	cmd.Env = InjectEnv(opts.Env, agent, base)
	if opts.Stdout != nil {
		cmd.Stdout = opts.Stdout
	}
	if opts.Stderr != nil {
		cmd.Stderr = opts.Stderr
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("wrap: launching %s: %w", binary, err)
	}
	return cmd, nil
}

// writeBackupWithDir runs the backup with an optional dir override.
func writeBackupWithDir(opts Options, agent Agent) (bool, error) {
	if opts.BackupDir == "" {
		return WriteBackup(agent, configPathFor(agent), envGetter(opts.Env))
	}
	// Mkdir the override dir and redirect the default dir for this call.
	if err := os.MkdirAll(opts.BackupDir, 0o755); err != nil {
		return false, fmt.Errorf("wrap: creating backup dir %s: %w", opts.BackupDir, err)
	}
	prev := defaultBackupDir
	defaultBackupDir = opts.BackupDir
	defer func() { defaultBackupDir = prev }()
	return WriteBackup(agent, configPathFor(agent), envGetter(opts.Env))
}

// envGetter adapts a []string env to the env func the backup uses.
func envGetter(env []string) func(string) string {
	return func(key string) string {
		prefix := key + "="
		for _, kv := range env {
			if len(kv) > len(prefix) && kv[:len(prefix)] == prefix {
				return kv[len(prefix):]
			}
		}
		return ""
	}
}

// ping confirms the proxy answers on loopback (liveness before launch).
func ping(base string) error {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(base + "/stats")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

// portFromAddr extracts the port from a "host:port" address.
func portFromAddr(addr string) int {
	_, p, err := net.SplitHostPort(addr)
	if err != nil {
		return 0
	}
	var port int
	_, _ = fmt.Sscanf(p, "%d", &port)
	return port
}
