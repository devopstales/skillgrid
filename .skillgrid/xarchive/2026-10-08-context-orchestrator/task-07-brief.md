# T07 — wrap: `ctx wrap <agent>` launcher (core-4 + opt-in)

**Ticket:** TASK-095 (spec T07 / TICKET-07) · Wave 5 · PR3 · Blocks: T08, T14 · Blocked by: T04 (done), T05 (done)
**SATISFIES:** `wrap-core4-launches`, `wrap-optin-agent` (acceptance.feature @step-09, @step-10)

## Goal
Add a tier-2 launcher `mnemonic ctx wrap <agent>` that: starts the proxy, injects the agent's config to point at the proxy, backs up the previous config before writing, and launches the agent as a subprocess. **The agent's code is never modified — only its config.** Re-running is idempotent (no double-inject, no double-backup).

## What already exists (DO NOT rebuild)
- `internal/ctx/config/config.go` — `Wrap` struct ALREADY exists (T03): `Enabled bool`, `Agents []string` (default `["opencode","claude","codex","cursor"]` = **core-4**), `InjectEnv bool` (default true), `BackupConfig bool` (default true). Reuse it; do NOT redefine.
- `internal/ctx/proxy/` — the proxy (`New(cfg, store, log, mode)`, `Handler()`, `GetMode()`, `SetMode()`). The wrap launcher must START the proxy (reuse the same build the `ctx proxy` CLI uses in `cmd/mnemonic/ctx_proxy.go`) on a loopback port.
- `cmd/mnemonic/ctx.go` — the `ctx` subcommand group with `proxy` dispatch. Add a SIBLING `ctx_wrap.go` (do NOT edit ctx.go unless the dispatch line is required — prefer a new file that registers itself).
- The proxy base URL the agent must point at: `http://127.0.0.1:<port>` (the loopback bind). For Anthropic-compatible agents set `ANTHROPIC_BASE_URL`; for OpenAI-compatible agents set `OPENAI_BASE_URL` (plan 16.8 Tier 1 contract).

## Core-4 vs opt-in
- **Core-4** (always available, no config needed): `claude`, `codex`, `cursor`, `opencode`. These are the defaults in `Wrap.Agents`.
- **Opt-in agents** (e.g. `aider`, `cline`, `continue`, `goose`, `copilot`, `openclaw`): available ONLY if listed in the `wrap.agents` config. `wrap aider` works when aider is in config; a non-opt-in agent (not core-4 and not in config) is **rejected with a clear error** (e.g. `unknown agent "foo" (add it to wrap.agents to enable it)`).
- The agent → config mapping (which env var / config file each agent uses) lives in the wrap package. For this ticket, model each agent as: name, provider (anthropic|openai), the base-URL env var to set, and (optionally) a config-file path. Claude = anthropic (`ANTHROPIC_BASE_URL`); codex/cursor/opencode/aider = openai (`OPENAI_BASE_URL`) — keep it simple, the exact per-agent detail can be a small table; the ACs only require claude (anthropic) + aider (openai) to work.

## Acceptance (must be green)
`go test ./internal/ctx/wrap/...` exits 0, with tests proving:
1. **`wrap claude`**: starts the proxy (assert the proxy is serving on the loopback port), injects claude's config to point at the proxy (assert `ANTHROPIC_BASE_URL` = `http://127.0.0.1:<port>` is set in the launched subprocess env, OR the config file is rewritten — pick ONE mechanism and document it; env-inject via `inject_env` is the simplest and testable), **writes a backup of the previous config BEFORE writing the new one** (assert the backup file exists and contains the pre-inject content), and launches the agent (a test stub subprocess).
2. **Idempotent re-run**: running `wrap` a second time does NOT double-inject (config/env already points at the proxy → detected, no second write) and does NOT write a second backup (assert only one backup exists after two runs).
3. **Opt-in `wrap aider`**: when aider is in the `wrap.agents` config, `wrap aider` starts the proxy + injects + launches.
4. **Non-opt-in rejected**: `wrap <agent>` for an agent that is neither core-4 nor in config returns a clear error (assert the error message + that the proxy did NOT start / agent did NOT launch).
5. **Agent code never modified**: the launched agent's source tree is byte-identical before/after (only its config/env changes).

## Design / constraints
- **Files (Owns):** `mnemonic/internal/ctx/wrap/wrap.go` (orchestration: start proxy → backup → inject → launch → the idempotency check), `agents.go` (the agent table: name/provider/env-var/config-path + core-4 set + opt-in resolution), `config-inject.go` (read current config/env, write the proxy-pointing one), `backup.go` (write the backup BEFORE the config write; idempotent — skip if already wrapped), `mnemonic/internal/ctx/wrap/*_test.go`, `mnemonic/cmd/mnemonic/ctx_wrap.go` (the `ctx wrap <agent>` CLI subcommand).
- **Injection mechanism:** prefer **env injection** (`inject_env: true`) — launch the agent subprocess with `ANTHROPIC_BASE_URL`/`OPENAI_BASE_URL` = proxy base URL in its env. This is the most testable (the test stub can read its own env) and it matches "config-injecting launcher" loosely. If you use a config-file rewrite instead, the backup MUST capture the prior file bytes first. Document whichever you pick. The backup is the key AC — it must exist before the new config/env takes effect, and only for the FIRST wrap (idempotent).
- **Backup:** write to a predictable path (e.g. `<config path>.ctx-backup` or a `.skillgrid/ctx/wrap-backups/<agent>.bak`) containing the pre-inject bytes. If `backup_config` is false in config, skip (but the AC assumes the default true). Idempotency: if the agent is already wrapped (config/env already points at the proxy), do NOT overwrite the backup and do NOT re-inject.
- **Idempotency check:** detect "already wrapped" by checking whether the current config/env already targets the CTX proxy (e.g. env already == a `127.0.0.1:8787`-style proxy URL, or a marker). On re-run: skip inject + skip backup, still launch.
- **Start the proxy:** reuse the proxy build from `ctx proxy`. The wrap launcher owns the proxy lifecycle for the launched agent (start before launch; you may keep it running for the subprocess lifetime — for the ticket, starting + confirming it serves is enough; the E2E in T08 exercises the full turn). Use a loopback port (default 8787, but allow an ephemeral/injected port for tests).
- **Launch:** `exec.Command(agentBinary, args...)` with the injected env; the test uses a stub binary/script (e.g. a tiny Go test helper or a `sh -c` that echoes its env) as the "agent". Do NOT require a real claude/codex install — the test stubs the agent.
- **ADR-0016 fail-open / robustness:** a launch failure returns a clear error (non-zero exit), not a panic. A proxy-start failure aborts with a clear error (don't launch the agent pointed at a dead proxy).
- **No new 3rd-party deps** (stdlib only: `os/exec`, `os`, `path/filepath`, `net/http` for the proxy liveness check, etc.).
- **Do NOT modify** the proxy package, the config package, or the agent's source. `ctx wrap` is a setup layer only (plan 16.10: no per-agent proxy code, no control-plane).
- **Do NOT build** the E2E turn-through-proxy test (that's T08, `e2e_test.go`), the CLI `runCtx` rework (T14), or the proxy itself (T04).

## Gates (ALL exit 0, run from the mnemonic/ dir)
- G1: `go build ./...`
- G2: `go test -count=1 ./internal/ctx/wrap/...`
- G3: `go test -count=1 ./internal/ctx/proxy/... ./internal/ctx/config/...` (no regressions)
- G4: `go vet ./internal/ctx/wrap/... ./cmd/...`
- fmt: `gofmt -l internal/ctx/wrap/ cmd/mnemonic/ctx_wrap.go` empty.

## Commit
Conventional subject (feat(ctx): ...). Body block exactly:
```
[skillgrid-context]
change: 2026-10-08-context-orchestrator
ticket: TASK-095
wave: 5
pr: PR3
```
Stage ONLY your files. NEVER `git add -A`. Do NOT re-stage the `mnemonic/mnemonic` build artifact.

## Report
Write the FULL report to `/Users/paladm/git/ai-test/skillgrid-v2/.skillgrid/sdd/2026-10-08-context-orchestrator/task-07-report.md`: what you built (files + responsibilities), the injection mechanism you chose + why, each AC mapped to its test, gates with exit codes, commit SHA, and any concerns/parked items.
