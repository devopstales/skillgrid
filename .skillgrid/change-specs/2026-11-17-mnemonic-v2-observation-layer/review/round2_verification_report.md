# Round 2 Verification Report — 2026-11-17-mnemonic-v2-observation-layer

Fresh-context verification pass after the round-1 revision. Scope: (a) rule
RESOLVED/RE-OPENED on each round-1 finding F-1..F-13 against the revised
`blueprint.md` / `acceptance.feature` / `briefing.md`; (b) fresh-eyes pass on
the revised deltas (envelope mapping table, MatchFile, install
source-resolution, exit-code contract, sessions derivations, detection rules)
for NEW Blocker/Major issues; (c) code verification wherever a fix makes a
code claim. Read-only toward the change docs; no git/go test run.

## Code verification record (claims checked against `mnemonic/` + this host)

| Claim | Where checked | Result |
|---|---|---|
| `MNEMONIC_DATA_DIR` is the v2 config env | `internal/core/config/config.go` (`EnvDataDir = "MNEMONIC_DATA_DIR"` const block) | ✓ holds |
| `sessions` DDL: `id, project, directory NOT NULL, title, started_at, ended_at, summary`; no agent column | `internal/core/store/migrations/001_schema.sql:11-24` | ✓ holds (basis of F-6/F-9 fixes) |
| `store.Store.Begin(ctx) (Txn, error)` at `store.go:139` | `internal/core/store/store.go:139` (exact line) | ✓ seam exists — but see N-1 on the `Txn` shape |
| `Txn` interface shape | `store.go:146-153`: UpsertFile/UpsertSymbols/ReplaceEdgesForFile/ReplaceUnresolvedForFile/ReplaceChunksForFile/Commit/Rollback only | ✗ no generic Exec/Query — N-1 |
| generic `Exec` is single-statement, non-transactional | `store.go:134`; `sqlite.go:2596` (`db.ExecContext` direct) | ✓ |
| `ExecDDL` is NOT a transaction (per-statement autocommit) | `sqlite.go:2581-2593` | ✓ |
| `busy_timeout(5000)` DSN + `foreign_keys(1)` (FK probe valid) | `sqlite.go:308`, `concurrency_test.go:61` | ✓ holds (F-11 fix) |
| real `~/.cursor/hooks.json` = exactly 20 hook names | live file, `hooks` object | ✓ all 20 covered by the pinned mapping table (verified name-by-name) |
| `path.Match("src/**","src/a/b.go")` == false (basis of MatchFile) | 6-case `go run` script | ✓ false, and `*.env`~`.env` basename fallback true, all pinned rows consistent |
| migration number 300 free | `migrations/` listing: 001-004, 100, 200-202 | ✓ |
| `project.Detector.Resolve(cwd, explicit, gitToplevel)` | `internal/core/project/project.go:36` | ✓ holds |
| `storeFor` + `run_test_helpers_test.go` swap `gitToplevel` | `cmd/mnemonic/run_test_helpers_test.go:31,45` | ✓ holds |
| `raw.Probe` test helper + `TestMigrationReopenSkipsAppliedFiles` pattern | `store/migrate_test.go:24,66` | ✓ hold |
| DDL `action_type` vocabulary includes `session_error`, excludes `session_shutdown` | `briefing.md` DDL comment; blueprint T-2; acceptance `obs-live-omp-session-produces-events` | see F-1 below |

## F-1..F-13 disposition

| ID | Round-1 sev | Status | Evidence (revised text / code) |
|---|---|---|---|
| F-1 | Major | **RE-OPENED** (residual) | Fix largely landed — envelope now says "`event` = the **raw per-host hook name** … `Normalize` (Task 1) maps `(agent, event) → action_type` via this pinned table"; Task 1 Step 1 pins "every row of the envelope table above, per host, + unknown `(agent, event)` → error"; mapping covers every raw hook the adapters emit (pi/omp 5, opencode/kilo 5, cursor 20 = verified against the live `~/.cursor/hooks.json`). **Residual contradiction:** the pi/omp row reads "`session_start` / `session_shutdown` (quit) / `tool_call` / `tool_result` | **same names**" — literally, `session_shutdown` emits `session_shutdown`, which is NOT in the pinned DDL vocabulary (`session_start\|session_end\|session_error\|tool_call\|tool_result\|prompt\|turn_end\|compact\|agent_response\|agent_thought\|subagent_start\|subagent_stop`) and contradicts blueprint T-2 ("produces `session_start`, `tool_call`+`tool_result` pairs, `session_end` rows") and acceptance `obs-live-omp-session-produces-events` ("shows … `session_end`") and briefing R1 ("`session_end` (shutdown-quit)") — round 1's own disposition prescribed `session_shutdown→session_end`. One-line fix: make the row explicit (`session_shutdown (quit) → session_end`). |
| F-2 | Major | RESOLVED | Envelope: "Exit codes: 0 ok · 1 error (usage, parse, store, dialect, unknown event — always allow) · 2 reserved (block, phase 2 — **never** a usage error)"; Task 3 Step 1: "missing args → exit 1 (a usage error does NOT take exit 2 — that code is reserved for block (phase 2); R8: a misconfigured adapter must allow, never block; deliberate deviation from the CLI's exit-2 usage convention)". Adapter pins unchanged (exit 2 → `{block:true}`), acceptance `obs-ingest-exit-codes-fail-open` consistent (1 for malformed/pg/unknown-agent). |
| F-3 | Major | RESOLVED | Task 4 Step 3: "**Adapter source resolution — pinned, no Go embed** (in order, first hit wins): (1) explicit `--repo <checkout>` containing `plugins/obs/`; (2) the staged dir `$DATA_DIR/obs-plugins/` when present — written by `obs install --sync <repo>` … (3) a checkout with `plugins/obs/` found by walking up from the cwd. None present → error: …" — `--sync` added to the Task 4 verb list; `plugins/obs/pi/*` listed as source truth in Files/Consumes. |
| F-4 | Major | RESOLVED | Task 2: "Pinned matcher `MatchFile(pattern, path) bool` (`match.go` — `path.Match` alone cannot recurse: `path.Match("src/**", "src/a/b.go")` is false …): split pattern and path on `/`; a `**` segment matches zero-or-more whole segments … pattern containing no `/` is also matched against the basename"; pinned test rows incl. `src/**`~`src/a/b.go` true; `match.go` in Task 2 Files; Step 1 tests the rows. Independently verified with a Go script: `path.Match("src/**","src/a/b.go")`=false, `*.env`~`.env`=true — all pinned rows hold under the stated semantics. Acceptance `obs-logs-and-query-read-stored-events` (`--file "src/**"` glob-matches) is now satisfiable for deep paths. |
| F-5 | Minor | RESOLVED | Global Constraints: "Data dir: `~/.skillgrid/mnemonic` (`MNEMONIC_DATA_DIR` override — the v2 config env, `config.go:35`)" — verified `EnvDataDir = "MNEMONIC_DATA_DIR"` in `config.go`. |
| F-6 | Minor | RESOLVED | Task 1 Produces: "`obs.Event` (the row struct; carries `Directory` = raw `env.cwd`, never redacted — the carrier for the sessions upsert)"; Task 2: "the `directory` column = `ev.Directory`, the raw cwd" — `sessions.directory` is `NOT NULL` in the 001 DDL, satisfied by the raw-cwd value. |
| F-7 | Minor | RESOLVED | Task 1 Step 1: "**values are pinned per item by the briefing R5 union, NOT the v1 file** — the v1 file lacks `.env`/`*.pem`/`*.key`/`*secret*` (that is the gryph set), uses 32-bit djb2 (NOT carried over — sha256, gryph parity), and truncates with `\n[truncated]` (NOT carried over — `…[+N]`)" — all three misattributions corrected in place. |
| F-8 | Minor | RESOLVED | Task 4 Step 1: "**pi/omp adapter unit (the fail-open unit half):** `plugins/obs/pi/obs.test.ts` (node, no deps — stub the `spawn`/`spawnSync` globals): missing binary (ENOENT) → no throw, exit-0 semantics; `tool_call` with exit 2 → returns `{block: true, reason: stderr}`; exit 1 → swallowed. Run: `node plugins/obs/pi/obs.test.ts`" — file in Task 4 Files. |
| F-9 | Minor | RESOLVED | Task 7 Produces: "derivations pinned — the `sessions` table has no agent column: agent = the most frequent `obs_events.agent` for that session, tie → most recent; title = `sessions.title` (null → `-`); events = row count; last-active = `MAX(obs_events.occurred_at)`, fallback `sessions.started_at`" — derivations composable against the verified 001 DDL (no agent column; title/started_at present). |
| F-10 | Minor | RESOLVED | Task 4 Step 1: "**Detection (pinned, drives `install all` + `status`):** omp → `~/.omp/agent` exists; pi → `$SKILL_PI_AGENT_DIR` or `~/.skill-pi/agent` exists; opencode → `~/.config/opencode` exists; kilo → `~/.config/kilo` exists; cursor → `~/.cursor` exists. Tests: `install all` installs only detected fixture hosts, undetected ones reported as skipped (exit 0); `--force` installs all five regardless." — matches acceptance `obs-live-hosts` ("all detected hosts"). |
| F-11 | Nit | RESOLVED | Global Constraints: "cursor's command runs under the harness's own limit (no cap in our command string) — the store side is bounded by the existing `busy_timeout(5000)` DSN, and project detection's `git rev-parse` shell-out is the CLI's existing behavior (unchanged)" — DSN verified in code. |
| F-12 | Nit | RESOLVED | `K1`/`K6` no longer appear anywhere in blueprint.md (grep: no matches); the CLI/MCP seam is now described by substance ("the CLI/MCP byte-identical output seam does not apply"). |
| F-13 | Nit | RESOLVED | "correct" no longer appears in blueprint.md (grep: no matches); Task 9 Step 1 reads "agent label equal to the host that ran the session". |

## New findings (fresh-eyes pass on revised deltas)

### N-1 — MAJOR — the pinned ingest tx seam cannot carry raw SQL through `store.Store` as written

Task 2 pins: "Consumes: `store.Store` (`st.Query`, `st.Begin(ctx) (Txn, error)`
— `store.go:139` …)" and Step 3: "Implement via the store's tx seam
`st.Begin(ctx) (Txn, error)` (`store.go:139` — the code-index single-per-file-tx
surface): inside one Txn — session upsert (`ON CONFLICT(id) DO NOTHING`),
event insert, commit." T-1 requires "in one transaction".

But the `Txn` interface (`store.go:146-153`) exposes only the five code-index
writes + Commit/Rollback — **no generic `Exec`/`Query`**. So the two raw
`INSERT`s cannot be composed inside the pinned seam. Alternatives checked:
the generic `st.Exec` (`store.go:134`) is a single statement with no
transaction (sqlite impl: `db.ExecContext` direct, `sqlite.go:2596`);
`ExecDDL` (`sqlite.go:2581`) runs each sub-statement in its own
`db.ExecContext` — also not one transaction. Consequence: as written, the
implementer either violates T-1's atomicity (two autocommit `Exec`s) or must
add an interface method that Task 2's Files list does not record (Files
currently: only `obs/store.go`, `obs/match.go`, `obs/store_test.go`).

**Minimal fix (agreed lean):** add `Exec(ctx context.Context, query string,
args ...any) error` to the `Txn` interface in
`mnemonic/internal/core/store/store.go`; implement in the two adapters that
already hold the open `*sql.Tx` — `sqliteTxn` (`sqlite.go:2639`) and
`pgCodeTxn` (`postgres_code.go:671`) via `tx.ExecContext`; record in Task 2
Files as Modify: `store.go`, `sqlite.go`, `postgres_code.go`. Both
implementations are one-liners; additive to the interface (S-2-compliant,
same shape as the existing `Exec`/`Query` seams).

### Cross-consistency checks (no material issues found)

- **Mapping table vs DDL vocabulary:** every produced `action_type`
  (session_start, session_end, session_error, tool_call, tool_result, prompt,
  turn_end, compact, agent_response, agent_thought, subagent_start,
  subagent_stop) is in the briefing DDL vocabulary — except the F-1 residual
  above. `result_status=error` for `postToolUseFailure` is in the
  `success|error|blocked` set.
- **Mapping table vs live adapters:** all 20 raw hook names in this host's
  `~/.cursor/hooks.json` have mapping rows; pi/omp (5) and opencode/kilo (5)
  raw events all covered; unknown-pair → exit 1 consistent across envelope
  section, Task 1 (Normalize error), Task 3 Step 1.
- **Exit-code contract:** envelope section / Task 3 / adapters / acceptance
  `obs-ingest-exit-codes-fail-open` mutually consistent (0 allow, 1 allow-on-error,
  2 reserved-block, never usage).
- **briefing vs blueprint DDL:** both carry `session_error` in the
  `action_type` vocabulary; 4 indexes in `300_obs_events.sql` match the
  briefing schema verbatim.
- **MatchFile vs acceptance:** `--file "src/**"` glob-matching deep paths is
  pinned true (`src/**`~`src/a/b.go`) — scenario satisfiable.
- **sessions derivations (Task 7)** composable against the real 001 DDL
  (FK enforced via `foreign_keys(1)` DSN, so the FK probe in Task 2 tests is
  meaningful).
- **Detection rules vs `obs-live-hosts`** ("detected hosts") — consistent.
- **Source-resolution order vs `--sync` verb pin** — consistent; error
  message pins the recovery path.
- Minor (not reported, below threshold): `obs status` "store event counts"
  sub-clause of briefing R9 is not carried into the blueprint's status
  output pin (no acceptance scenario tests it; Task 9 only checks install
  state).

## Verdict

**NOT CLEAN** — 0 Blocker, 2 Major outstanding (F-1 RE-OPENED residual:
pi/omp mapping row must explicitly pin `session_shutdown (quit) →
session_end`; N-1: additive `Txn.Exec` seam + Task 2 Files entry). All other
round-1 findings RESOLVED; no other new Blocker/Major.
