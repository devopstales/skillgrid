# Mnemonic-v2 Observation Layer — Implementation Blueprint (steps, files, verification)

> **Role:** this is the *how* — implementation plan: concrete steps, files, code,
> tests, verification. Companion `briefing.md` is the *what*. Assume zero codebase
> context. DRY. YAGNI. TDD. Frequent commits. No placeholders.

> **For agentic workers:** REQUIRED SUB-SKILL: subagent-execution (recommended) or
> simple-execution, task-by-task. Steps use `- [ ]` checkboxes.

**Status:** PROPOSED

**Tier:** T2 (change class: **risky** — new trust boundary: untrusted agent
processes exec the binary with a stdin contract + migration; floor L2+L3)

**Build shape:** Tracer thread — omp end-to-end first (migration → ingest →
`obs logs` → live omp session), then host fan-out, then the read verbs.

**Goal:** gryph-style no-daemon event capture for 5 agents (pi, omp, opencode,
kilo, cursor) into the v2 per-project SQLite store, with a `mnemonic obs`
command family (install/uninstall/status/logs/query/sessions/session/
stats/export/prune + hidden `_ingest`).

**Architecture:** Per-host adapters (2 TS sources serve 4 hosts; cursor is a
config merge) exec `mnemonic obs _ingest <agent> <event>` with a JSON envelope
on stdin — gryph's model (briefing D2/D6, 2026-11-17-confirmed against the
live gryph install). Go owns all normalization (D5): tool-type mapping,
`<private>` strip, truncation, sensitive-path content gate, SHA-256. Events
land in `obs_events` (migration 300, per-project store, local-only like
`session_trails`); the ingest tx auto-registers the `sessions` row (the v2
SaveEvent pattern). No daemon, no MCP tools, no new Go dependencies.

**Tech Stack:** Go 1.22+ (stdlib + existing `modernc.org/sqlite` store); build-free
TypeScript adapters (node builtins only; opencode's bundled `@opencode/plugin`
import pattern per ADR-0045); no new dependencies (AGENTS.md — ADR-0008 rule).

**Spec:** `briefing.md` (same folder). In-force constraints cited: ADR-0031
(Go floor), ADR-0008 (dependency policy — satisfied: zero new deps), ADR-0032
(opencode+kilo native TS, kilo copy-at-install), ADR-0043 (pi extension folder +
manifest), ADR-0045 (opencode V1+V2 dual entrypoint), ADR-0013 (repo is truth).
No ADR Review Manifest (`adr.md`) exists for this change — the briefing's
Context section is the research record (no `findings.md` → no Findings line).

**Decision note (R6 reconciliation):** the briefing's R6 says "reuse the
memory-plugins setup machinery"; that installer is in flight (this change is
queued behind it) and its API is not pinned in this blueprint. The blueprint
therefore ships a **self-contained installer** (`obs/install.go`, the v1
`stagedPluginPath` pattern); if the memory-plugins installer exposes a
reusable host-registry API by execution time, Task 4 migrates onto it —
recorded as a follow-up inside Task 4, not a scope change.

## Terms

- **observation layer / obs events** — agent-activity audit trail
  (`obs_events`). New term; glossary row added in Task 8 (disambiguated from
  the memory-domain *observation*, `.skillgrid/projectwiki/glossary/02-technical-terms.md`).
- **ingest envelope** — the hook→binary JSON contract (below, R3).

## Must-Haves (goal-backward verification)

**Truths:**
- T-1: piping a valid envelope to `mnemonic obs _ingest omp tool_call` inserts
  one `obs_events` row in the project store and (if absent) a `sessions` row,
  in one transaction.
- T-2 (backstop — live host): an omp session with tool calls produces
  `session_start`, `tool_call`+`tool_result` pairs, `session_end` rows, visible
  in `mnemonic obs logs`.
- T-3: `obs query --action tool_call --command "git *"` returns exactly the
  matching shell events; `--file "src/**"` glob-matches `path`.
- T-4: `<private>x</private>` in input/prompt → absent from all stored text;
  sensitive path (`.env`, `~/.ssh/id_ed25519`, `*.pem`) → row has `path`, empty
  `input_preview`/`output_preview`; `content_hash` = sha256 of the full
  pre-truncation value.
- T-5 (backstop — live host): `mnemonic` absent from PATH → the agent's tool
  call proceeds; hook exits 0 with a stderr note.
- T-6: `obs install all` on a fixture HOME with gryph/orca/herdr entries in
  `~/.cursor/hooks.json` → foreign entries byte-identical after install;
  re-run is a no-op; `obs uninstall` removes exactly the 20 entries + files it
  created; backups exist under `~/.skillgrid/mnemonic/backups/`.
- T-7: `obs prune --retention 90d` deletes only `obs_events` rows older than
  the window; `sessions` and all other tables untouched; `--dry-run` reports
  the count without deleting.
- T-8: `obs export --since 1d` emits one parseable JSON object per line;
  `obs stats` counters equal direct SQL counts for the same window.
- T-9: migration 300 applies clean on a fresh store and on an existing
  store (existing tables/rows untouched); re-open is a no-op.

**Artifacts:**
- `mnemonic/internal/core/store/migrations/300_obs_events.sql` — table + 4 indexes (briefing schema).
- `mnemonic/internal/obs/model.go` — Event, envelope, action/tool-type/result constants, agent validation.
- `mnemonic/internal/obs/normalize.go` — the single normalization contract (v1 JS logic ported to Go).
- `mnemonic/internal/obs/store.go` — Ingest (tx), Query, ListSessions, SessionDetail, Stats, Export, Prune.
- `mnemonic/internal/obs/install.go` — per-host installers + manifest + status.
- `mnemonic/cmd/mnemonic/obs.go` — all 11 verbs.
- `plugins/obs/pi/mnemonic-obs.ts` + `package.json` — pi+omp source (single file; `__AGENT__` substitution at install).
- `plugins/obs/opencode/mnemonic-obs.ts` — opencode+kilo source (ADR-0045 dual entry).
- `plugins/obs/cursor/hooks-obs-cursor.json` — the 20-entry template (binary placeholder).
- Tests: `mnemonic/internal/obs/{normalize,store,install}_test.go`, `mnemonic/cmd/mnemonic/obs_test.go`.

**Key links:**
- `_ingest` must call `obs.Normalize` before insert (no raw passthrough) and
  must resolve the project via `project.Detector` on the envelope's `cwd`
  (same chain the CLI uses — explicit > env > git > basename).
- The installed adapters must exec exactly
  `<bin> obs _ingest <agent> <event>` with the envelope on stdin — pinned by
  install tests (manifest command string) and the live backstops (T-2/T-5).
- `obs install`'s cursor merge must preserve foreign `hooks.json` entries —
  pinned by T-6 (fixture built from this host's real gryph/orca/herdr entries).

**One-way-door decisions:**
- ⚠ **Migration 300** (Task 1): new table in the per-project store; additive,
  but a migration is one-way by definition — requires explicit user approval
  before Task 1 lands.
- (Assessed, not one-way: installed-file ownership — manifest-tracked and
  reversible via `obs uninstall`; `_ingest` envelope — carries `v: 1`,
  additive fields allowed; CLI surface — additive top-level family.)

## Global Constraints

- Go 1.22+ to build (ADR-0031); `go build ./...` / `go test ./...` /
  `go vet ./...` clean from the `mnemonic/` module root.
- No new Go dependencies (AGENTS.md; ADR-0008 precedent) — stdlib + existing
  `modernc.org/sqlite`. TS adapters: node builtins only (no npm deps).
- Fail-open is a contract (R8): every adapter and the ingest path — a
  capture failure must never block the agent. Spawn caps where our adapter
  controls the spawn: 30s pi/omp `tool_call` (spawnSync), 5s opencode
  `tool.execute.before` (execFileSync); cursor's command runs under the
  harness's own limit (no cap in our command string) — the store side is
  bounded by the existing `busy_timeout(5000)` DSN, and project detection's
  `git rev-parse` shell-out is the CLI's existing behavior (unchanged).
- Data dir: `~/.skillgrid/mnemonic` (`MNEMONIC_DATA_DIR` override — the v2
  config env, `config.go:35`); per-project store `<project>.sqlite`. Obs is
  SQLite-dialect only (local-only audit, `session_trails` scope);
  PG-dialect ingest returns a clean error (exit 1 → fail-open), not a crash.
- Spec-zone commits before code-zone (pre-commit zone guard); conventional
  commits, no AI trailers.
- Truncation caps: tool previews 500, prompt 2000 (v1 `MAX_PREVIEW` /
  `MaxPromptLength` values). Retention default 90d (v1 `DefaultRetentionDays`).

## File Structure

- `mnemonic/internal/obs/` — new package: model.go (types + constants),
  normalize.go (pure functions, table-tested), store.go (SQL over
  `store.Store` — the projectwiki-package precedent: raw `st.Exec`/`st.Query`
  with the store's tx seam), install.go (per-host writers + manifest).
- `mnemonic/internal/obs/*_test.go` — table tests for normalize (the v1
  contract), store (fresh + existing store), install (fixture HOMEs).
- `mnemonic/cmd/mnemonic/obs.go` — verb dispatch (the trail.go/mem.go pattern:
  flag set per verb → openRuntime-light store open → obs pkg → dispatchResult).
- `mnemonic/cmd/mnemonic/main.go` — add `case "obs":` to the `run()` switch +
  usage line.
- `mnemonic/cmd/mnemonic/obs_test.go` — verb wiring (arg parse, exit codes,
  `--format json` envelope shape).
- `mnemonic/internal/core/store/migrations/300_obs_events.sql` — the table.
- `plugins/obs/pi/{mnemonic-obs.ts,package.json}` — pi+omp source.
- `plugins/obs/opencode/mnemonic-obs.ts` — opencode+kilo source.
- `plugins/obs/cursor/hooks-obs-cursor.json` — cursor entry template.
- `.skillgrid/projectwiki/glossary/02-technical-terms.md` — obs terms (Task 8).
- `docs/user-guide/04-hooks.md` — obs install section (Task 8).

**Install targets (live):** pi → `<pi-agent-dir>/extensions/mnemonic-obs/`
(dir + manifest, ADR-0043; `SKILL_PI_AGENT_DIR` honored, default
`~/.skill-pi/agent`); omp → `~/.omp/agent/extensions/mnemonic-obs.ts`
(auto-discovered); opencode → `~/.config/opencode/plugins/mnemonic-obs.ts`
(ADR-0032/0045 discovery); kilo → `~/.config/kilo/plugin/mnemonic-obs.ts` +
registration in kilo's config (v1 `installNativePlugins` configPath
resolution is the reference); cursor → merged entries in `~/.cursor/hooks.json`.
Manifest: `~/.skillgrid/mnemonic/obs-install.json` (per host: paths, content
hashes, exact hook entries, binary path, version, installed_at).

**Ingest envelope (v:1):**
```json
{"v":1,"session_id":"…","cwd":"/repo","agent":"omp","event":"tool_call",
 "tool_name":"bash","tool_call_id":"…","input":{"command":"git status"},
 "output":"…","status":"success","error":null,"duration_ms":42,
 "prompt":"…","project":"override?"}
```
`event` = the **raw per-host hook name** (what the adapter was triggered
by). `Normalize` (Task 1) maps `(agent, event) → action_type` via this
pinned table — the single source of truth; an unknown `(agent, event)`
pair → ingest error (exit 1, fail-open):

| agent | raw `event` | → `action_type` |
|---|---|---|
| pi, omp | `session_start` / `tool_call` / `tool_result` | same names |
| pi, omp | `session_shutdown` (quit only) | `session_end` |
| pi, omp | `before_agent_start` (user msgs only) | `prompt` |
| opencode, kilo | `tool.execute.before` / `tool.execute.after` | `tool_call` / `tool_result` |
| opencode, kilo | `session.created` / `session.idle` / `session.error` | `session_start` / `turn_end` / `session_error` |
| cursor | `preToolUse`, `beforeShellExecution`, `beforeMCPExecution`, `beforeReadFile`, `beforeTabFileRead` | `tool_call` |
| cursor | `postToolUse`, `afterShellExecution`, `afterMCPExecution`, `afterFileEdit`, `afterTabFileEdit`, `postToolUseFailure` | `tool_result` (failure → `result_status=error`) |
| cursor | `beforeSubmitPrompt` / `sessionStart` / `sessionEnd` / `stop` / `preCompact` | `prompt` / `session_start` / `session_end` / `turn_end` / `compact` |
| cursor | `afterAgentResponse` / `afterAgentThought` / `subagentStart` / `subagentStop` | `agent_response` / `agent_thought` / `subagent_start` / `subagent_stop` |

The raw `event` is NOT stored (the `action_type` + `tool_name` +
`result_status` columns carry the same information; future policy phase 2
can extend the table additively). Exit codes: 0 ok · 1 error (usage,
parse, store, dialect, unknown event — always allow) · 2 reserved (block,
phase 2 — **never** a usage error).

---

### Task 1: Migration 300 + obs model + normalization

> ⚠ one-way: migration 300 (additive table in the per-project store) — STOP
> for explicit user approval before landing.

**Files:**
- Create: `mnemonic/internal/core/store/migrations/300_obs_events.sql`
- Create: `mnemonic/internal/obs/model.go`, `mnemonic/internal/obs/normalize.go`
- Test: `mnemonic/internal/obs/normalize_test.go`,
  `mnemonic/internal/core/store/migrate_obs_test.go`

**Interfaces:**
- Consumes: the store's `//go:embed migrations/*.sql` runner
  (`migrateSQLiteUpTo` records each file in `index_meta` — a new file is
  picked up with zero runner changes).
- Produces: `obs.Event` (the row struct; carries `Directory` = raw
  `env.cwd`, never redacted — the carrier for the sessions upsert),
  `obs.Envelope` (the R3 JSON contract),
  `obs.Normalize(env Envelope, privateTools []string) (Event, error)` (pure,
  deterministic — the seam for the whole privacy contract; errors on an
  unknown `(agent, event)` pair), constants `Action*`, `ToolType*`,
  `Result*`, `Agent*` (valid set: pi, omp, opencode, kilo, cursor), and the
  pinned `(agent, event) → action_type` mapping table (above).
- Seam: `Normalize` is the unit under test for R5; adapters never normalize.
- Deletion test: without it, every host adapter + the CLI re-derives
  truncation/redaction (the v1 JS duplication this change kills).
- Adapters: prod (ingest path) + table tests (≥2 — justified).

**SATISFIES:** `obs-ingest-normalizes`, `obs-privacy-contract`

- [ ] **Step 1: Write the failing tests.**
  `migrate_obs_test.go`: fresh `store.Open` → `raw.Probe`
  `SELECT COUNT(*)>0 FROM sqlite_master WHERE name='obs_events'` true;
  re-open no-op (the `TestMigrationReopenSkipsAppliedFiles` pattern); existing
  store (open, add an observation, re-open after migration) → observation
  rows untouched. `normalize_test.go` — table tests. Structure ports from
  v1 `tool-call-capture.js` (`mapToolType` verbatim: bash/shell→shell;
  read/glob/grep→file_read; write/edit/multiedit/strreplace/patch/delete→
  file_write; default other); **values are pinned per item by the briefing
  R5 union, NOT the v1 file** — the v1 file lacks `.env`/`*.pem`/`*.key`/
  `*secret*` (that is the gryph set), uses 32-bit djb2 (NOT carried over —
  sha256, gryph parity), and truncates with `\n[truncated]` (NOT carried
  over — `…[+N]`): `sensitivePath` (the v1 regex + `.env`, `*.pem`, `*.key`,
  `*secret*`); `stripPrivateTags` (nested/partial `<private>` loops to
  fixpoint); truncation (500 → 497 + `…[+N]`; 2000 prompt); `content_hash`
  (sha256 of the FULL pre-truncation string, hex); private-tools allowlist
  (listed tool → previews nil); **event mapping** (every row of the envelope
  table above, per host, + unknown `(agent, event)` → error).

```go
func TestNormalizeSensitivePathDropsContent(t *testing.T) {
	ev, err := obs.Normalize(obs.Envelope{Agent: "omp", Event: "tool_result",
		ToolName: "read", Input: map[string]any{"file_path": "/repo/.env"},
		Output: "SECRET=1"}, nil)
	if err != nil || ev.Path != "/repo/.env" ||
		ev.InputPreview != nil || ev.OutputPreview != nil {
		t.Fatalf("sensitive path must keep path, drop previews: %+v %v", ev, err)
	}
	if ev.ContentHash == "" { t.Fatal("hash must cover the full value") }
}
```

- [ ] **Step 2: Run tests to verify they fail** (package doesn't exist).
- [ ] **Step 3: Implement** the migration (briefing schema, verbatim) +
  `model.go` + `normalize.go` (pure functions; `sha256.Sum256` hex; the
  v1 regexes as `regexp.MustCompile`).
- [ ] **Step 4: Run tests to verify they pass**
  (`go test ./internal/obs/... ./internal/core/store/ -run 'Obs|Normalize' -count=1`).
- [ ] **Step 5: Commit** — `feat(obs): migration 300 + normalization contract`
  (migration + Go together = code zone, one commit).

### Task 2: obs store ops — Ingest, Query, Prune

**Files:**
- Create: `mnemonic/internal/obs/store.go`, `mnemonic/internal/obs/match.go`
- Modify: `mnemonic/internal/core/store/store.go` (add
  `Exec(ctx context.Context, query string, args ...any) error` to the
  `Txn` interface), `sqlite.go` (`sqliteTxn`), `postgres_code.go`
  (`pgCodeTxn`) — additive interface extension (the ADR-0058
  Store-extension precedent — the Code-index `Store` interface extension
  that established the additive Begin/Txn surface; both adapters hold
  `*sql.Tx`, one-line impls). Find and update any test fakes implementing `Txn` (grep
  `ReplaceChunksForFile`).
- Test: `mnemonic/internal/obs/store_test.go`

**Interfaces:**
- Consumes: `store.Store` (`st.Query`, `st.Begin(ctx) (Txn, error)` —
  `store.go:139` — + the projectwiki-package raw-SQL-over-Store
  precedent). The `Txn` surface has no generic statement method today —
  this task adds `Txn.Exec` (Files above); the ingest tx then composes
  two raw statements inside one Txn.
- Produces: `Ingest(ctx, st, ev Event) (id int64, err)` — the tx:
  `INSERT INTO sessions (id, project, directory) VALUES (?,?,?)
  `ON CONFLICT(id) DO NOTHING` (only when `session_id` ≠ ""; the
  `directory` column = `ev.Directory`, the raw cwd) then
  `INSERT INTO obs_events (…) VALUES (…)`; `Query(ctx, st, F) []Event`
  `F{Project, Session, Agent, Action, File glob, Command glob/prefix, Since,
  Limit}` — empty = no filter; `--file` uses the pinned
  `obs.MatchFile(pattern, path)` helper (below), Go-side, not SQL);
  `ListSessions`, `SessionDetail`, `Stats`
  (GROUP BY agent/tool_type/action_type), `Export` (rows, same order as
  logs), `Prune(ctx, st, retention) (n int64, err)`
  (`DELETE FROM obs_events WHERE occurred_at < ?` only).
- Pinned matcher `MatchFile(pattern, path) bool` (`match.go` —
  `path.Match` alone cannot recurse: `path.Match("src/**", "src/a/b.go")`
  is false; gryph's `--file "src/**/*.ts"` is recursive): split pattern and
  path on `/`; a `**` segment matches zero-or-more whole segments; other
  segments use `path.Match`; a pattern containing no `/` is also matched
  against the basename (so `*.env` hits `/repo/.env`). Tests (pinned rows):
  `src/**`~`src/a/b.go` true; `src/*`~`src/a/b.go` false;
  `src/*.go`~`src/a.go` true / ~`src/a/b.go` false; `*.env`~`repo/.env`
  true.
- Seam: store.go is the SQL surface; verbs in Task 3/7 call these.
- Deletion test: every verb re-derives SQL; the privacy-adjacent invariants
  (auto-register, delete-only-obs_events) lose their single home.
- Adapters: prod verbs + tests (≥2).

**SATISFIES:** `obs-ingest-atomically-registers-session`,
`obs-prune-scoped-to-obs-events`

- [ ] **Step 1: Failing tests** — ingest with unknown session → both rows
  exist (FK resolves); ingest again same session → one session row; unknown
  `agent` → error before insert; Query filters (action/file/command/agent/
  since, combined); Prune with 2 old + 1 new row → 2 deleted, `sessions`
  row count unchanged; Export order = `(occurred_at, id)`; MatchFile the
  four pinned rows (recursive `src/**`, non-recursive `src/*`, basename
  `*.env`).
- [ ] **Step 2: Verify RED.**
- [ ] **Step 3: Implement** via the store's tx seam `st.Begin(ctx) (Txn,
  error)` (`store.go:139`): inside one Txn, using the `Txn.Exec`
  statement method added in this task's Files step — session upsert
  (`ON CONFLICT(id) DO NOTHING`), event insert, commit. (The generic
  `st.Exec`/`ExecDDL` are single-statement non-transactional — NOT the
  seam.) Atomicity pinned by the FK probe + a concurrent-ingest test on
  the same store (WAL).
- [ ] **Step 4: Verify GREEN** (`go test ./internal/obs/ -count=1 -race`).
- [ ] **Step 5: Commit** — `feat(obs): store ops (ingest tx, query, prune)`.

### Task 3: CLI — `_ingest`, `logs`, `query` (+ door check)

**Files:**
- Modify: `mnemonic/cmd/mnemonic/main.go` (`case "obs"` + usage line)
- Create: `mnemonic/cmd/mnemonic/obs.go` (runObs + the 3 verbs; the
  `obsOpenStore(cwd, fl, explicit)` helper = `config.Load` →
  `project.Detector.Resolve(cwd, explicit, gitToplevel)` → `storeFor` —
  the openRuntime light path WITHOUT the LLM/handler build)
- Test: `mnemonic/cmd/mnemonic/obs_test.go` (the run_test_helpers_test.go
  fixture pattern: swap `gitToplevel`, capture stdout)

**Interfaces:**
- Consumes: Task 1/2 (Normalize, Ingest, Query), `config.Load`,
  `project.Detector`, `storeFor`.
- Produces: the hidden verb contract: `mnemonic obs _ingest <agent> <event>`
  reads ALL of stdin (cap 1 MiB — beyond = exit 1, fail-open), parses the
  envelope (unknown `event`/`agent` → exit 1), normalizes, ingests; prints
  nothing on success. `logs` (default 24h, human table: time · agent ·
  action · tool · path/command preview; `--since 24h|--today|--7d`,
  `--agent/--session/--action/--limit`, `--format json` = the envelope
  array) and `query` (gryph-parity flags + `--count`).
- Seam: the stdin contract — pinned by the envelope fixtures in
  `obs_test.go` (one per host shape: omp `tool_call`, cursor `postToolUse`
  with `tool_input`/`tool_output`, pi `tool_result`).
- Deletion test: hooks have nowhere to deliver events; logs/query lose their
  only data path.
- Adapters: live adapters (Task 4-6) + fixtures (≥2).

**SATISFIES:** `obs-logs-and-query-read-stored-events`,
`obs-ingest-exit-codes-fail-open`

- [ ] **Step 1: Failing tests** — `run([]string{"obs","_ingest","omp","tool_call"})`
  with stdin envelope → exit 0, row present; malformed JSON → exit 1 (stderr
  note, no row, no panic); missing args → exit 1 (a usage error does NOT
  take exit 2 — that code is reserved for block (phase 2); R8: a
  misconfigured adapter must allow, never block; deliberate deviation from
  the CLI's exit-2 usage convention); PG dialect → exit 1 with
  "sqlite dialect only"; unknown `(agent, event)` → exit 1; `logs` on the
  seeded store prints the rows;
  `logs --format json` = parseable envelope array; `query --command "git *"`
  filtering.
- [ ] **Step 2: Verify RED.**
- [ ] **Step 3: Implement** (verb dispatch in `runObs`; `--json` reuses
  `finish`/`dispatchResult` — obs results print via `printHuman`-style
  lines, `--format json` = raw envelope array, NOT the mem-family envelope:
  obs is CLI-only — the CLI/MCP byte-identical output seam does not apply).
- [ ] **Step 4: Verify GREEN** (`go test ./cmd/mnemonic/ -run Obs -count=1`).
- [ ] **Step 5: Door check (live, this host)** — `go build -o /tmp/mnemonic ./cmd/mnemonic`;
  pipe a real-shaped omp envelope → `obs logs` shows the row in
  `~/.skillgrid/mnemonic/<project>.sqlite`. If the row doesn't appear, STOP —
  the tracer is dead.
- [ ] **Step 6: Commit** — `feat(obs): CLI _ingest + logs + query`.

### Task 4: pi/omp adapter + install (omp, pi) + manifest

**Files:**
- Create: `plugins/obs/pi/mnemonic-obs.ts`, `plugins/obs/pi/package.json`,
  `plugins/obs/pi/obs.test.ts` (build-free node test, no deps)
- Create: `mnemonic/internal/obs/install.go` (agent registry, file copyers,
  manifest read/write, Status)
- Modify: `mnemonic/cmd/mnemonic/obs.go` (`install`/`uninstall`/`status`
  verbs — host args `pi|omp|opencode|kilo|cursor|all`, `--dry-run`,
  `--sync <repo>` (stage adapter sources), `--force` (install undetected))
- Test: `mnemonic/internal/obs/install_test.go` (fixture HOME via
  `t.Setenv("HOME", ...)`; the pi agent dir via `SKILL_PI_AGENT_DIR`)

**Interfaces:**
- Consumes: `plugins/obs/pi/*` (source truth), `os.UserHomeDir`,
  `SKILL_PI_AGENT_DIR`, manifest path under the data dir.
- Produces: `obs.Install(ctx, opts Opts) (Manifest, error)` — Opts
  `{Home, Agents []string, DryRun, BinPath, RepoOrStagedDir}`;
  `obs.Uninstall(ctx, opts)`, `obs.Status(ctx) (per-host report)`;
  `Manifest {Version, BinPath, InstalledAt, Hosts map[string]HostRecord}`
  where `HostRecord {Kind file|dir|hooks-json, Path, Files map[string]string
  (sha256), Entries []string}`.
- Seam: the host registry — adding a host = one registry entry + one adapter
  file; the verb surface and manifest never change.
- Deletion test: install/status/uninstall each re-derive path + merge logic
  per verb (the idempotency + no-clobber invariants lose their home).
- Adapters: prod installers (omp, pi now; opencode/kilo/cursor in Tasks 5-6)
  + fixture tests (≥2).

**Adapter source** (`plugins/obs/pi/mnemonic-obs.ts` — the
`gryph-hooks.ts` shape, verified 2026-11-17): `export default function
(pi: ExtensionAPI)` with `const AGENT = "__AGENT__"` (installer substitutes
pi|omp; the substitution is pinned by test — installed file contains
`const AGENT = "omp"`, never `__AGENT__`); `pi.on("session_start")` →
fire-and-forget `send("session_start")`; `pi.on("before_agent_start")` →
prompt capture (only `event.messageType === "user"`, min length 11, max
2000 — the pi plugin's v1 constants); `pi.on("tool_call")` →
`sendSync("tool_call")` (30s cap; exit 2 → return `{block: true,
reason: stderr}`); `pi.on("tool_result")` → fire-and-forget;
`pi.on("session_shutdown")` → end only on `reason === "quit"`. `send`/
`sendSync` = `spawn(SpawnBin, ["obs","_ingest",AGENT,hook])`, payload JSON
on stdin, `MNEMONIC_BIN` env override else `mnemonic`; every error swallowed
(exit 0 semantics); session id from `ctx.sessionManager.getSessionFile() ??
"ephemeral"`, cwd from `ctx.cwd`. `package.json`: `{"name":"@devopstales/pi-mnemonic-obs",
"pi":{"extensions":["./mnemonic-obs.ts"]}}` (the ADR-0043 manifest shape,
`@devopstales/pi-mnemonic` as the naming reference).

**SATISFIES:** `obs-install-omp-and-pi`, `obs-fail-open-missing-binary`
(unit half), `obs-live-omp-session-produces-events` (backstop half)

**Follow-up (recorded, per the header decision note):** if the
memory-plugins change's installer exposes a host-registry API by execution
time, migrate this task's installers onto it and delete the duplicate.

- [ ] **Step 1: Failing tests** — install omp to fixture HOME →
  `~/.omp/agent/extensions/mnemonic-obs.ts` exists, contains `const AGENT =
  "omp"`, not `__AGENT__`; manifest record present; re-install = no-op
  (mtime/hash unchanged); install pi with `SKILL_PI_AGENT_DIR=<tmp>` →
  `<tmp>/extensions/mnemonic-obs/{mnemonic-obs.ts,package.json}`; uninstall
  removes exactly those (a foreign `other-extension.ts` survives); status
  reports installed/stale (hash mismatch) per host + that host's store
  event count (the briefing R9 sub-clause). **Detection (pinned,
  drives `install all` + `status`):** omp → `~/.omp/agent` exists; pi →
  `$SKILL_PI_AGENT_DIR` or `~/.skill-pi/agent` exists; opencode →
  `~/.config/opencode` exists; kilo → `~/.config/kilo` exists; cursor →
  `~/.cursor` exists. Tests: `install all` installs only detected fixture
  hosts, undetected ones reported as skipped (exit 0); `--force` installs
  all five regardless. **pi/omp adapter unit (the fail-open unit half):**
  `plugins/obs/pi/obs.test.ts` (node, no deps — stub the `spawn`/
  `spawnSync` globals): missing binary (ENOENT) → no throw, exit-0
  semantics; `tool_call` with exit 2 → returns `{block: true, reason:
  stderr}`; exit 1 → swallowed. Run: `node plugins/obs/pi/obs.test.ts`.
- [ ] **Step 2: Verify RED.**
- [ ] **Step 3: Implement** (file copyers = read→write→hash; the omp target
  is a bare file; the pi target is the folder). **Adapter source
  resolution — pinned, no Go embed** (in order, first hit wins): (1)
  explicit `--repo <checkout>` containing `plugins/obs/`; (2) the staged
  dir `$DATA_DIR/obs-plugins/` when present — written by
  `obs install --sync <repo>` (copies `plugins/obs/*` + a version stamp;
  the v1 `stagedPluginPath` pattern, now owned by this change); (3) a
  checkout with `plugins/obs/` found by walking up from the cwd. None
  present → error: "adapter sources not found: pass --repo, run from a
  skillgrid checkout, or sync first (`mnemonic obs install --sync <repo>`)".
- [ ] **Step 4: Verify GREEN** (`go test ./internal/obs/ -run Install -count=1`).
- [ ] **Step 5: Live (backstop T-2)** — real install on this host
  (`obs install omp`), run this omp session's tool calls, then
  `mnemonic obs logs` → rows present. Record evidence.
- [ ] **Step 6: Commit** — `feat(obs): pi/omp adapter + install + manifest`.

### Task 5: opencode/kilo adapter + install

**Files:**
- Create: `plugins/obs/opencode/mnemonic-obs.ts`
- Modify: `mnemonic/internal/obs/install.go` (opencode + kilo registry
  entries), `mnemonic/cmd/mnemonic/obs.go` (verbs now cover 4 hosts)
- Test: `mnemonic/internal/obs/install_test.go` (+opencode/kilo cases),
  `plugins/obs/opencode/obs.test.ts` (the `plugins/opencode/test/` pattern —
  a build-free node test with a stubbed `@opencode/plugin` import)

**Interfaces:**
- Consumes: ADR-0045 dual-entry pattern (the existing
  `plugins/opencode/skillgrid-events.ts` `importPlugin()` fallback is the
  reference file — read it before writing); the kilo config path resolution
  from `skillgrid-cli/internal/setup/opencode.go installNativePlugins`
  (reference implementation — the kilo config file + registration key).
- Produces: the opencode adapter: `"tool.execute.before"` (5s
  `execFileSync`, exit 2 → `throw new Error(stderr)`),
  `"tool.execute.after"` (swallow), `event` handler for exactly
  `session.created` (→ `session_start`), `session.idle` (→ `turn_end`),
  `session.error` (gryph parity — no others); payload carries `sessionID`,
  `tool`, `args`/`result.{title,output,metadata}`, `cwd: directory`;
  `AGENT = "opencode"` source constant, installer substitutes `kilo` for the
  kilo copy.
- Seam: same host registry as Task 4.
- Deletion test / Adapters: as Task 4 (registry entries + fixture tests).

**SATISFIES:** `obs-install-opencode-kilo`

- [ ] **Step 1: Failing tests** — install opencode →
  `~/.config/opencode/plugins/mnemonic-obs.ts` (contains `AGENT = "opencode"`);
  install kilo → `~/.config/kilo/plugin/mnemonic-obs.ts` (contains
  `AGENT = "kilo"`) + the registration entry in kilo's config (fixture
  config file, pre-existing foreign entries preserved); uninstall reverses
  both; the node test: stub `@opencode/plugin`, import the plugin, fire
  `tool.execute.before` with a stubbed `execFileSync` → args
  `["obs","_ingest","opencode","tool.execute.before"]` + JSON stdin; exit 2
  → throws; exit 1 → swallowed.
- [ ] **Step 2: Verify RED.**
- [ ] **Step 3: Implement.**
- [ ] **Step 4: Verify GREEN** (`go test ./internal/obs/ -run Install` +
  `node plugins/obs/opencode/obs.test.ts` from the repo root).
- [ ] **Step 5: Commit** — `feat(obs): opencode/kilo adapter + install`.

### Task 6: cursor installer (hooks.json merge, 20 entries)

**Files:**
- Create: `plugins/obs/cursor/hooks-obs-cursor.json` (the 20-entry template,
  `"command": "__BIN__ obs _ingest cursor <hookType>"` — `__BIN__`
  substituted at install with the absolute `os.Executable` path, falling
  back to bare `mnemonic`)
- Modify: `mnemonic/internal/obs/install.go` (cursor entry: JSON merge +
  backup + uninstall-by-exact-command-match)
- Test: `mnemonic/internal/obs/install_test.go` (+cursor cases — fixture
  `hooks.json` built from THIS host's real gryph/orca/herdr entries:
  `"gryph _hook cursor postToolUse"`, the orca `if [ -f … ]` wrapper,
  `"bash '/Users/…/herdr-agent-state.sh' session"`)

**Interfaces:**
- Consumes: the template (20 hook names pinned — the D6 list), the manifest.
- Produces: cursor install = read `~/.cursor/hooks.json` → for each of the
  20 hook types, append `{command}` if no entry with the identical command
  string exists (idempotency by exact match) → atomic write (tmp+rename) →
  backup the pre-image to `$DATA_DIR/backups/hooks.json.<UTCstamp>`;
  uninstall removes exactly the manifest-recorded command strings, leaves
  all foreign entries, deletes the file only if it became empty-and-ours
  (never: a foreign entry exists).
- Seam: the merge function `mergeCursorHooks(existing map[string][]any,
  entries []string) (map[string][]any, added []string)` — pure, table-tested.
- Deletion test: the no-clobber + idempotency + backup invariants
  (T-6/T-7 of the Must-Haves) lose their single home.
- Adapters: prod + fixture (≥2).

**SATISFIES:** `obs-install-cursor-preserves-foreign-entries` (the
idempotency + re-install cases live inside this scenario)

- [ ] **Step 1: Failing tests** — fixture merge: foreign entries
  byte-identical after install (deep-equal on the non-obs slices); the 20
  obs entries present, each exactly once; re-install → zero additions;
  uninstall → only the 20 removed; malformed existing JSON → install fails
  with a clear error and the file untouched (no partial write — the
  tmp+rename proves it); backup file exists after first install.
- [ ] **Step 2: Verify RED.**
- [ ] **Step 3: Implement.**
- [ ] **Step 4: Verify GREEN.**
- [ ] **Step 5: Live (backstop)** — real install on this host: verify
  `~/.cursor/hooks.json` keeps the gryph/orca/herdr entries, adds the 20
  obs entries; `obs status` reports cursor installed. (Cursor session
  evidence is Task 9.)
- [ ] **Step 6: Commit** — `feat(obs): cursor hooks.json installer`.

### Task 7: CLI read verbs — sessions, session, stats, export, prune

**Files:**
- Modify: `mnemonic/cmd/mnemonic/obs.go` (5 verbs)
- Test: `mnemonic/cmd/mnemonic/obs_test.go` (+cases)

**Interfaces:**
- Consumes: Task 2 store ops; the `sessions` table (id, title,
  started_at, ended_at — no agent column) joined with `obs_events` for
  agent/event-count (derivations pinned in Produces) and `obs session
  <id>` (ordered detail, id-prefix match like `--resume`).
- Produces: `sessions` (table: id · agent · title · events · last-active;
  derivations pinned — the `sessions` table has no agent column: agent =
  the most frequent `obs_events.agent` for that session, tie → most
  recent; title = `sessions.title` (null → `-`); events = row count;
  last-active = `MAX(obs_events.occurred_at)`, fallback
  `sessions.started_at`),
  `session <id>` (time-ordered rows), `stats` (`--since`/`--agent`;
  GROUP BY agent, tool_type, action_type counts — table, no TUI),
  `export [--since] [--agent] [-o file]` (JSONL, one envelope-shaped object
  per line, stdout when no `-o`), `prune --retention 90d [--dry-run]`.
- Seam: none new (store ops from Task 2).
- Deletion test: read surface re-enters ad-hoc SQL in the verb layer.
- Adapters: verbs + fixtures (≥2).

**SATISFIES:** `obs-sessions-session-stats-export`,
`obs-prune-scoped-to-obs-events` (CLI half)

- [ ] **Step 1: Failing tests** — seeded multi-session store: sessions
  listing counts; session detail ordering (id, not mtime); stats equal a
  direct SQL GROUP BY; export parses as JSONL with one object per line and
  matches query output for the same window; prune --dry-run prints the
  count and deletes nothing; prune deletes only the old rows.
- [ ] **Step 2: Verify RED.**
- [ ] **Step 3: Implement.**
- [ ] **Step 4: Verify GREEN** (`go test ./cmd/mnemonic/ -run Obs -count=1`).
- [ ] **Step 5: Commit** — `feat(obs): sessions/session/stats/export/prune`.

### Task 8: Docs + glossary + full suite

**Files:**
- Modify: `docs/user-guide/04-hooks.md` (an "Observation layer (mnemonic v2)"
  section: install/uninstall/status, the 5-host table, fail-open + privacy
  contract, the 90d retention, phase-2 policy pointer),
  `.skillgrid/projectwiki/glossary/02-technical-terms.md` (rows: observation
  layer, obs event, ingest envelope — each cross-linking the memory-domain
  *observation* row to kill the D1 collision in the glossary itself),
  `mnemonic/cmd/mnemonic` usage output (the obs line in `printUsage` —
  verify it shows the verb list)
- Test: none new (doc change); full-suite run below.

**Interfaces:** Consumes: shipped behavior (Tasks 1-7). Produces: the
user-facing contract + the term disambiguation (D1's resolution lands here).
- Seam / Deletion test / Adapters: docs — "none (prose)".

**SATISFIES:** — (docs task; no scenario)

- [ ] **Step 1: Write the sections** (from the shipped flags — no
  hand-typed flag lists).
- [ ] **Step 2: Full verification** — `cd mnemonic && go build ./... && go vet
  ./... && go test ./... -count=1` green; `gofmt -l .` empty.
- [ ] **Step 3: Commit** — `docs(obs): user guide + glossary terms`.

### Task 9: Live 5-host validation (backstop evidence)

**Files:** — (evidence only; recorded in the change's report at QA)

**Interfaces:** Consumes: real installs on this host (pi, omp, opencode,
kilo, cursor — whichever the user has active). Produces: the T-2/T-5/T-6
backstop evidence (session id + `obs logs` output per host; the fail-open
probe: `PATH` stripped of mnemonic → one tool call proceeds).

**SATISFIES:** `obs-live-hosts` (backstop), `obs-fail-open-missing-binary`
(live half)

- [ ] **Step 1: Per host** (omp = this session; the others = one short real
  session each): trigger ≥3 tool calls, then `mnemonic obs logs --agent
  <h>` → rows present, agent label equal to the host that ran the session,
  + expected action sequence.
- [ ] **Step 2: Fail-open probe** — run one tool call with the mnemonic
  binary temporarily renamed (restore immediately): the call succeeds,
  stderr note only.
- [ ] **Step 3: Record** evidence lines into the QA report (EVIDENCE
  comments in acceptance.feature, the repo pattern).

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| Documentation-like paths | N/A: installed adapters are `.ts` code files + JSON config (no executable Markdown/MDX, no `README.sh`-class files) | — | — |
| Git repository selection | Applicable: ingest resolves the project via `git rev-parse --show-toplevel` on the envelope's `cwd` (worktree vs main checkout, relative cwd, cwd outside any repo → basename fallback) | Reuse `project.Detector` (the CLI's chain: explicit > env > git > basename) — never a hand-rolled `git -C`; envelope `project` override wins | `TestIngestProjectResolution`: cwd in a worktree resolves the worktree's project; cwd outside git → basename; `project` override beats the detector |
| Commit state | N/A: obs v1 stores no commit state (gryph's from/to_commit is a phase-2 addition with the policy plane) | — | — |
| Push state | N/A: no push interaction | — | — |
| PR commands | N/A: no PR automation | — | — |
| Mnemonic tool surface (`mem_*`/`code_*`/`web_cache_*`) | N/A: zero MCP changes — obs is CLI-only in v1 (no `obs_*` tools registered; the D3 door keeps them for phase 2); the `_ingest` envelope is a hook-facing CLI contract, pinned by Task 3 fixture tests, not an MCP contract | — | — |
| Shared-convention drift | N/A: no `_shared/{planning,craft,verification,knowledge}/` or `agent-config/` edits; the glossary + user-guide edits (Task 8) are vault/docs, not shared skill rules | — | — |

Subprocess/shell exposure (hooks exec the binary; cursor's command strings
are shell-interpreted by the harness) is covered by: envelope stdin cap
(1 MiB, Task 3), the spawn caps where our adapter controls the spawn
(Global Constraints — cursor's command runs under the harness's own limit;
the store side is bounded by the existing `busy_timeout(5000)` DSN), and
the
cursor command string built ONLY from the installed absolute binary path +
fixed verb/event words (no interpolation of hook payload data into the
command — Task 6 test pins the exact 20 strings).

## Review Summary

[The goldfish pipeline writes its verdict here — do not hand-edit after the
pipeline runs.

**Verdict:** READY FOR EXECUTION (converged — rounds 3-4 both
material-clean: 0 Blocker / 0 Major; all findings resolved in-place, none
parked)

**Rounds run:** 4 (round 1 = 5-stage consolidated review; rounds 2-4 =
verification passes of the prior round's fixes + fresh-eyes on the
revised deltas)

**Findings by severity:** round 1: 0 Blocker / 4 Major / 9 Suggestion —
all 13 fixed in-place (event mapping table, exit-2 collision, adapter
source resolution, MatchFile, env name, Directory carrier, port
attribution, pi/omp unit test, sessions derivations, detection rules,
timeout claim, K1/K6 ids, validating adjective). round 2: 0 Blocker / 2
Major — F-1 residual (`session_shutdown` row) + N-1 (additive `Txn.Exec`
seam, recorded in Task 2) both fixed; 1 sub-threshold minor (R9 status
event counts) fixed. round 3: CLEAN (0B/0M); 1 sub-threshold citation
minor F-14 (ADR-0045→0058) fixed. round 4: CLEAN (0B/0M, full ADR-citation
sweep of both spec files); 1 sub-threshold citation minor F-15
(briefing ADR-0004-transport-mcp→0061, both sites) fixed post-round-4
against the ADR-0061 LOCKED title the round-4 pass already read.

**Parked findings:** — (none)

**Reviewed:** 2026-11-17 (goldfish 5-stage pipeline, fresh-context
subagents; reports in `review/`)
]
