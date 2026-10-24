# T07 Report — `ctx wrap <agent>` tier-2 launcher (core-4 + opt-in)

**Ticket:** TASK-095 (spec T07 / TICKET-07) · Wave 5 · PR3 · Blocks: T08, T14
**Commit:** `ba1e4ab1`
**Status:** DONE — all 5 ACs green, all gates exit 0.

## What I built

### `internal/ctx/wrap/agents.go` (new)
- `Agent` struct: `{Name, Provider, EnvVar}` — the launcher's agent → config mapping.
- `agentTable` — the small table the brief asked for: claude (anthropic, `ANTHROPIC_BASE_URL`); codex/cursor/opencode/aider (openai, `OPENAI_BASE_URL`).
- `core4` — always-available set: claude, codex, cursor, opencode (matches `config.Wrap.Agents` defaults).
- `resolveAgent(name, enabledAgents)` — core-4 always resolves; a known-but-not-core-4 agent (aider, …) resolves only when listed in `enabledAgents`; an unknown name returns the clear error `unknown agent "foo" (add it to wrap.agents to enable it)`; a known-but-not-enabled agent returns `agent "aider" is not enabled (add it to wrap.agents to enable it)`.

### `internal/ctx/wrap/config-inject.go` (new)
- `InjectEnv(env, agent, proxyBaseURL)` — returns a COPY of env with the agent's base-URL var pointed at the proxy (replaces in place, or appends). Never mutates the caller's slice.
- `alreadyWrapped(env, agent)` — the idempotency predicate: true when the agent's base-URL env var already holds a loopback proxy URL.
- `isLoopbackBaseURL(base)` — heuristic loopback detection (127.0.0.1 / ::1 / localhost), handling scheme, path, `host:port`, and bracketed IPv6 (`[::1]:8787/v1`). The launcher only ever writes loopback URLs, so "loopback base URL present" == "we wrapped this".
- `configPathFor(agent)` — returns `""` for all agents: **env-only injection, no config-file rewriting in v1** (documented in the doc comment; the backup/inject seam already supports file rewriting when an agent needs it).

### `internal/ctx/wrap/backup.go` (new)
- `WriteBackup(agent, configPath, env)` — writes the pre-inject state **before** the injection takes effect. For env-only agents: a snapshot of `ENVVAR=<pre-inject value>` into `.skillgrid/ctx/wrap-backups/<agent>.env.bak` (dir overridable via `Options.BackupDir` in tests). **Idempotent:** an existing backup is left untouched and `(false, nil)` is returned so the caller skips re-injection. I/O failure → `(false, err)`, fail-open-abort (ADR-0016).
- `RestoreBackup(agent, configPath)` — inverse of the injection (the reversibility seam; the CLI verb is a T14/ops concern, not built here).

### `internal/ctx/wrap/wrap.go` (new)
- `Options` — `Cfg` (the T03 `ctxconfig.CtxConfig`, reused not redefined), `AgentName`, `Binary`/`Args`/`Env` (tests inject a stub + controlled env), `BackupDir`, `Port`/`EphemeralPort`/`Bind` (test port injection), `Log`, `Stdout`/`Stderr`.
- `Result` — `Agent`, `ProxyBaseURL`, `Port`, `Injected`, `BackedUp`, `BackupPath`, `Command` (the launched `*exec.Cmd`).
- `StartProxy` seam + `DefaultStartProxy` — reuses the T04 build (`ctxproxy.New` + `http.Server` on a loopback listener, nil store = no per-turn telemetry; wrap owns the lifecycle, not the history). Port 0 → ephemeral, actual port returned.
- `Wrap(opts, startProxy)` — the orchestrator:
  1. resolve the agent (core-4 / opt-in / unknown — errors BEFORE any side effect, so AC 4's "proxy did not start" holds);
  2. start the proxy and **ping** `/stats` to confirm it serves before pointing the agent at it (proxy-start failure aborts with a clear error, no launch);
  3. idempotency check — already wrapped → skip inject + backup, still launch;
  4. backup (first wrap only; `backup_config: false` skips it entirely, injection still happens);
  5. inject + launch via `exec.Command(binary, args...)` with `InjectEnv`'d env — `cmd.Start()` errors are returned, never panicked (ADR-0016).
- The proxy stop func is returned to the caller on success (the CLI keeps it alive for the subprocess lifetime); on any error path the launcher stops the proxy itself.

### `cmd/mnemonic/ctx_wrap.go` (new) + `cmd/mnemonic/ctx.go` (dispatch only)
- `runCtxWrap` — the `ctx wrap <agent> [args...]` entry: loads the ctx config (config.d precedence, `-dir`), resolves `-bind`/`-port`/`-bin`, calls `Wrap` with `os.Environ()`, logs the backup/inject/launch state, keeps the proxy alive for the subprocess lifetime (SIGINT/SIGTERM stop it; agent exit stops it), and exits with the agent's exit code.
- `ctx.go` — 3-line sibling dispatch (`case "wrap": runCtxWrap(rest)`) + one usage line, per the brief's "dispatch line is required".

## Injection mechanism: env injection (chosen, per the brief)

The launched subprocess gets `ANTHROPIC_BASE_URL` / `OPENAI_BASE_URL` = the proxy's loopback base URL. **Why:** it is the most testable (the stub subprocess reads its own env back out), it needs no per-agent config-file format to parse, and it matches the plan 16.8 Tier-1 contract (the proxy speaks both protocols; the provider field only selects the env var). Config-file rewriting is parked as an upgrade path behind the same `configPathFor` seam.

## AC → test map

| AC | Test(s) | Result |
|----|---------|--------|
| 1. `wrap claude`: proxy serving on loopback, `ANTHROPIC_BASE_URL` injected into subprocess env, backup written BEFORE injection (exists, contains pre-inject content), agent stub launched | `TestWrapClaudeProxyInjectBackupLaunch` (real T04 proxy on an ephemeral loopback port; stub echoes its env; backup file asserted present with pre-inject value) | PASS |
| 2. Idempotent re-run: no double-inject, no second backup, agent still launches | `TestWrapIdempotentReRun` (two Wrap calls, same BackupDir: exactly one backup file; second run `Injected`/`BackedUp` false; subprocess env still points at the proxy) + `TestAlreadyWrapped` (predicate unit test incl. `[::1]` bracketed IPv6) | PASS |
| 3. Opt-in `wrap aider` (aider in `wrap.agents`): starts proxy + injects + launches | `TestWrapOptInAider` (`OPENAI_BASE_URL` asserted in stub env) | PASS |
| 4. Non-opt-in rejected: clear error, proxy did NOT start, agent did NOT launch | `TestWrapNonOptInRejected` (aider NOT in config: error message asserted, `proxyRecorder.called == false`), `TestWrapUnknownAgent` (`"foo"`: `unknown agent` message), `TestWrapKnownButNotEnabled` (message asserted) | PASS |
| 5. Agent code never modified: source tree byte-identical before/after | `agentSourceTree` helper (checksum over a fake agent tree) asserted equal after the Wrap in the AC-1 and AC-3 tests | PASS |

Extra coverage: `TestWrapProxyStartFailureFailsOpen` (listener bind fails → clear error, no launch), `TestWrapLaunchFailureFailsOpen` (missing binary → `cmd.Start` error returned, no panic), `TestWrapNoBackupWhenDisabled` (`backup_config: false` → no backup file, injection still happens).

## Gates (all exit 0, run from `mnemonic/`)

| Gate | Command | Exit |
|------|---------|------|
| G1 | `go build ./...` | 0 |
| G2 | `go test -count=1 ./internal/ctx/wrap/...` | 0 |
| G3 | `go test -count=1 ./internal/ctx/proxy/... ./internal/ctx/config/...` | 0 |
| G4 | `go vet ./internal/ctx/wrap/... ./cmd/...` | 0 |
| fmt | `gofmt -l internal/ctx/wrap/ cmd/mnemonic/ctx_wrap.go cmd/mnemonic/ctx.go` | empty |

Full `./internal/ctx/proxy/...` suite (T04/T05/T06) and `./internal/ctx/config/...` (T03) green — no regressions.

## Staging
Staged only the seven owned files (never `git add -A`):
`internal/ctx/wrap/{wrap,agents,config-inject,backup,wrap_test}.go`, `cmd/mnemonic/ctx_wrap.go`, `cmd/mnemonic/ctx.go` (dispatch + usage line only).
The `mnemonic/mnemonic` build artifact and unrelated modified files were NOT staged.

## Concerns / parked items
- **Backup content is an env snapshot, not a file** (`<agent>.env.bak` holding `ENVVAR=<prior value>`), because v1 injection is env-only. The file-rewriting upgrade path (a config file that must be backed up as raw bytes) is already supported by `WriteBackup`/`RestoreBackup` via `configPathFor` — flip `configPathFor` and the backup format follows.
- **`RestoreBackup` is implemented but not surfaced** — no CLI verb in this ticket (the brief scopes the CLI to `ctx wrap`; T14/ops owns the restore verb).
- **The proxy runs with a nil store** in `DefaultStartProxy`: wrap owns the lifecycle, not per-turn telemetry; when the agent talks through the proxy, the proxy's own serve path records history once a store is wired (T08's E2E exercises the full turn).
- **`writeBackupWithDir` swaps a package-level `defaultBackupDir`** for the test override — fine for serial tests (Go test files in one package run sequentially without `-parallel`), but a per-call dir parameter is the cleaner shape if the test suite ever parallizes.
- **`isLoopbackBaseURL` is a heuristic, not a URL grammar** — the launcher only ever writes loopback URLs, so "looks like our proxy" is the correct bar; a real provider base URL is never mistaken for wrapped.
