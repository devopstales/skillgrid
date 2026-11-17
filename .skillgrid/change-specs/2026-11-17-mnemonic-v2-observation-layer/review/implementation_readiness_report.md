# Stage 9 — Implementation Readiness (target: `blueprint.md` task steps + codebase)

Personas merged: `implementation-readiness.md` + `codebase-details.md` +
`style.md` (fresh passes, findings tagged). Mode: REVIEW.
Harness isolation: part 1 enforced by fresh subagent dispatch; part 2 enforced
(writes confined to `review/`); part 3 partial — per-project instruction files
not loaded as instructions (stated, not silent); part 4 enforced (local files +
local binaries only: `go run` for the stdlib-semantics probe, `gryph query
--help`, `node --version`; no network).

## Findings

### codebase-details

Severity: Major
Affected scope: Global Constraints, data-dir line (line 126)
Description: The data-dir override env var is pinned as `SKILLGRID_MNEMONIC_
DATA_DIR`; v2's config reads `MNEMONIC_DATA_DIR` (verified:
`mnemonic/internal/core/config/config.go:35` `EnvDataDir = "MNEMONIC_DATA_DIR"`,
applied at `applyEnv` line 411). `SKILLGRID_MNEMONIC_DATA_DIR` matches nothing
in the v2 module (grep over `mnemonic/` returns zero hits).
Rationale: Path/symbol-accuracy vector: a wrong env var name is a wrong
contract — Task 8's user-guide section and any test that redirects the data dir
would propagate it silently (the var simply has no effect). Fix: rename to
`MNEMONIC_DATA_DIR` (one line).

Severity: Major
Affected scope: Task 2 Interfaces — Produces (line 257)
Description: (Carried from Stage 7 design-critic, verified against code here.)
`path.Match` cannot serve the pinned `--file "src/**"` recursive semantics:
`path.Match("src/**", "src/a/b.go")` == false and
`path.Match("src/**", "src/b.go")` == true — empirically run on this host
(Go stdlib, `path.Match`'s `*` does not cross `/`; `**` is two single-segment
wildcards, so `src/**` ≡ `src/*`). gryph's `--file` is a recursive glob
(verified: `gryph query --help` example `--file "src/**/*.ts"`), which is the
parity reference for T-3.
Rationale: Behavioral-accuracy vector: the pinned stdlib function does not
behave the way the pinned acceptance requires, and the mismatch is silent for
shallow fixture paths. Fix: pin a `**`-aware string matcher in `internal/obs`
with nested-path table cases, or re-pin T-3/acceptance to single-segment
`src/*`.

Severity: Major
Affected scope: Task 4 Step 3 (line 395)
Description: (Carried from Stage 7, verified against code here.) "the built-in
copy" — the non-checkout source of the four adapter source files — is undefined.
The v1 pattern it cites, `stagedPluginPath`
(`skillgrid-cli/internal/setup/setup.go:196`), resolves
`~/.skillgrid/plugins/<rel>` staged by v1's `skillgrid install` hook-tree
staging; no such staging owner exists for `plugins/obs/` in v2, and no task's
Files list contains an `//go:embed` directive or a staging step.
Rationale: Dependency-claim + Zero-Decision vectors: every host installer reads
its source through this path; a globally-installed `mnemonic` binary with no
checkout must have a built-in copy, and the only stdlib mechanism for that is
`go:embed`, which is not pinned. Fix: pin the mechanism (embed as the built-in
copy; `--repo`/checkout re-stages `$DATA_DIR/obs-plugins/` for dev).

Severity: Minor
Affected scope: Task 1 Step 1 (lines 212-219)
Description: (Carried from Stage 7, verified against the v1 file here.) The
attribution "table tests porting the v1 `tool-call-capture.js` contract (the
file is the spec; port, don't re-derive)" is wrong for three of the listed
items:
- `sensitivePath` — the JS function is only
  `/(^|\/)\.(git|ssh|aws|config)(\/|$)/i` (verified:
  `hooks/tool-call-capture.js:72-74`); the `.env`/`*.pem`/`*.key`/`*secret*`
  members come from the gryph/briefing-R5 union, not the JS file.
- `content_hash` — the JS `contentHash` is a 32-bit djb2-style hash
  (`"h" + hex8`, verified lines 34-40), not sha256; the sha256-hex contract
  comes from gryph/briefing R5 and is what acceptance pins
  (`sha256("SECRET=1")`).
- truncation marker — the JS is `str.slice(0, max) + "\n[truncated]"`
  (verified lines 92-95); the pinned "497 + `…[+N]`" format is new
  (gryph-style), not the JS format.
Rationale: Reuse-claim accuracy vector: the per-item pinned values in Task 1
(sha256, 497+…[+N], the R5 set) are the correct ones for the acceptance, so a
porter following the item list lands right — but a porter who trusts the
attribution line and "ports the file" will regress all three to the JS values.
Fix: attribute each sub-contract to its real source (JS: `mapToolType`,
`stripPrivateTags`, the `SKILLGRID_MNEMONIC_PRIVATE_TOOLS` env; briefing R5 /
gryph: sensitive set, sha256, truncation marker, 90d default).
Verified-clean items in the same contract: `MAX_PREVIEW = 500` (JS line 25)
matches the pinned 500 cap; the `mapToolType` mapping (bash/shell→shell;
read/glob/grep→file_read; write/edit/multiedit/strreplace/patch/delete→
file_write; default other) matches the JS exactly; `stripPrivateTags` is a
loop-to-fixpoint over `/<private>[\s\S]*?(<\/private>|$)/gi` with whitespace
collapse — "nested/partial loops to fixpoint" is accurate; the private-tools
allowlist env name matches the JS.

Severity: Minor
Affected scope: Task 4 Files + SATISFIES (lines 333-378)
Description: Task 4 claims `obs-fail-open-missing-binary` "(unit half)" but its
Files list contains no unit test for the pi/omp adapter (only Go
`install_test.go`, which covers install mechanics); the pi/omp adapter's
fail-open behavior (ENOENT/timeout swallowed → exit-0 semantics, exit 2 →
`{block: true}`, `session_id` fallback `"ephemeral"`) is covered only by the
live backstops (T-2/T-9, EVIDENCE-at-QA). Task 5, by contrast, pins a node
test for the opencode adapter.
Rationale: Test-plan vector: the "unit half" tag names a verification the task
does not contain; the asymmetry means the trust-boundary contract on the two
pi-family hosts is only live-verified. Fix: add `plugins/obs/pi/obs.test.ts`
(fake `MNEMONIC_BIN` binary: missing → handler resolves without throwing;
exit 2 → `{block: true, reason}`; exit 1 → swallowed) and run it in Task 4
Step 4 next to the opencode one.

### implementation-readiness

Verified actionable (no findings): every task has exact create/modify paths;
steps are RED→verify→implement→GREEN→commit with commands; the Task 1 inline Go
test is runnable as written (it matches the `store.Open`/`raw.Probe` test
pattern, verified in `migrate_test.go`); Task 3's fixture approach matches
`run_test_helpers_test.go` (verified: `installRunEnv` swaps the store/handler
seams and `gitToplevel`, `captureRun` captures stdout); Task 4's
`t.Setenv("HOME")` + `SKILL_PI_AGENT_DIR` approach matches the v1 pi installer's
env (verified: `setup/pi.go:116-124`); Task 5's `node plugins/obs/opencode/
obs.test.ts` works on this host (node v24.19.0 type-strips `.ts`; the existing
`plugins/opencode/test/memory-prefetch.test.ts` uses the same plain-node,
stub-`@opencode/plugin` pattern); the codebase is buildable at every task
checkpoint (each task adds a self-contained package/file pair; no task leaves a
dangling symbol — the only cross-task dependency is Task 2 ← Task 1's `Event`,
which is created in Task 1 Step 3).

The carried findings from Stage 7 that remain implementation-readiness
defects after verification (not re-listed here, see `critic_report.md`): the
`Event`/`Ingest` directory gap (Task 1 line 232 / Task 2 line 252), the
`obs sessions` agent/last-active derivation (Task 7 line 505), and
`install all` host-absence semantics (Task 4 line 339).

### style

Verified clean on this pass: re-scan of the full document against the banned
lists (`_shared/craft/review-discipline.md`) returns the single hit already
recorded in Stage 7 (line 568 "correct agent label") — no new hits, no apology
hedges, no glazing directed at the plan.

## Codebase-claims verification record (checks a–h + supporting claims)

Checked against the code; clean items are listed so the parent does not
re-verify.

- **(a) `Store.Begin(ctx) (Txn, error)` at `store.go:139`** — CLEAN. Exact line
  match (`mnemonic/internal/core/store/store.go:139`); `Txn` is the K1 tx
  surface with Commit/Rollback.
- **(b) migrations auto-pickup via `//go:embed migrations/*.sql`** — CLEAN.
  `sqlite.go:34` embeds `migrations/*.sql`; `migrateSQLiteUpTo`
  (`sqlite.go:350-407`) walks `migrationsFS.ReadDir("migrations")`, skips files
  recorded in `index_meta` (`migration:<name>`), applies the rest, records
  `schema_version`. A new `300_obs_events.sql` is picked up with zero runner
  changes. Migration numbers 001-004, 100, 200-202 verified in `migrations/` —
  300 is the correct next number. The test patterns cited in Task 1 exist:
  `TestMigrationReopenSkipsAppliedFiles` (`migrate_test.go:66`) and the
  `st.(*sqliteStore)` + `raw.Probe` idiom (`migrate_test.go:24-31`).
- **(c) CLI dispatch + `openRuntime`/`storeFor`/`gitToplevel` wiring** — CLEAN.
  `openRuntime` (`main.go:101-122`) is exactly
  `config.Load → project.Detector{EnvVar}.Resolve(cwd, explicit, gitToplevel) →
  storeFor → handlerFor`; the light path drops only `handlerFor` (the LLM/
  handler build), so `obsOpenStore` is feasible without it. `gitToplevel` and
  `storeFor` are `var` seams swappable in tests; the top-level `run()` switch
  (`main.go:192-208`) has `memory|code|projectwiki|sync|trail|doctor|setup` —
  no `obs` collision; `cli_family_test.go`/`stale_refs_test.go` do not pin a
  closed verb set, so adding `case "obs"` breaks nothing. `finish` /
  `dispatchResult` / `printHuman` exist for the `--format json` path. `serve`
  is in `renamedCommands` (did-you-mean, not a live verb) — the briefing's
  "deprecated no-op" is accurate in effect.
- **(d) v1 normalization contract in `hooks/tool-call-capture.js`** — CLEAN for
  `mapToolType`, `stripPrivateTags`, `MAX_PREVIEW = 500`, and the
  `SKILLGRID_MNEMONIC_PRIVATE_TOOLS` env; NOT the source of the sensitive-set /
  sha256 / truncation-marker values (see the Minor finding above — the pinned
  values are right, the attribution is not).
- **(e) live gryph shapes** — CLEAN. `~/.omp/agent/extensions/gryph-hooks.ts`:
  4 hooks (`session_start`/`session_shutdown` fire-and-forget `spawn`;
  `tool_call` `spawnSync` with `timeoutMs = 30000`; `tool_result`
  fire-and-forget), `session_id: ctx.sessionManager.getSessionFile() ??
  "ephemeral"`, `cwd: ctx.cwd`, payload `tool_name/tool_call_id/input`, exit 2
  → `return {block: true, …}` — every Task 4 claim matches. `~/.config/
  opencode/plugins/gryph.js`: `execFileSync("gryph", ["_hook","opencode",
  hookType], {timeout: 5000})`; exit 2 re-thrown only from
  `tool.execute.before` (after + event handlers swallow); event handler fires
  for exactly `session.created`/`session.idle`/`session.error`; before-payload
  `session_id/tool/args/cwd: directory`, after-payload
  `result.{title,output,metadata}` — every Task 5 claim matches. `~/.cursor/
  hooks.json`: shape `{version: 1, hooks: {<hookType>: [{command: …}]}}`,
  exactly 20 gryph entries spanning the pinned 20-hook set, entry shape plain
  `{"command": "gryph _hook cursor <hookType>"}`, foreign entries present as
  described (orca `if [ -f … ]` wrappers on 8 hook types + herdr
  `bash '…/herdr-agent-state.sh' session` on sessionStart). Task 6's fixture
  references are real.
- **(f) pi manifest convention** — CLEAN. `plugins/pi/mnemonic/package.json`
  carries `{"pi": {"extensions": ["./index.ts"]}}` under name
  `@devopstales/pi-mnemonic`; `setup/pi.go` copies the folder to
  `<agent-dir>/extensions/<name>/` (default `~/.skill-pi/agent/extensions/`),
  honors `SKILL_PI_AGENT_DIR`, and the launcher resolves `pi.extensions`
  manifest else `index.ts`/`index.js`. The Task 4 manifest
  `{"name":"@devopstales/pi-mnemonic-obs","pi":{"extensions":["./mnemonic-obs.ts"]}}`
  follows the convention; the `mnemonic-obs` folder name coexists with the
  in-flight memory plugin's `mnemonic` folder without collision.
- **(g) opencode V1+V2 dual entry** — CLEAN. `plugins/opencode/skillgrid-
  events.ts` has the V1 named export plus `importPlugin()` (`try import
  "@opencode/plugin"` catch → local `Plugin.define` stub) and `export default
  Plugin.define({…})` — the exact reference Task 5 names; ADR-0045 (accepted)
  pins the dual-entrypoint requirement. The plain-node test pattern in
  `plugins/opencode/test/memory-prefetch.test.ts` (no framework, no
  `@opencode/plugin` dependency, run via `node`/`--experimental-strip-types`)
  is the reference Task 5's node test copies.
- **(h) kilo config registration reference** — CLEAN.
  `skillgrid-cli/internal/setup/opencode.go` `installNativePlugins` copies
  `plugins/opencode/<base>.ts` → `~/.config/<harness>/plugin/` and registers
  `./plugin/<name>.ts` in the harness config via `upsertPluginKey` (setup.go:413,
  JSONC `plugin` array upsert); `AgentConfigPath(home, "kilo")` resolves
  `kilo.jsonc`/`opencode.json`/`opencode.jsonc`; v1 pins the agent label via
  `env.SKILLGRID_AGENT` (the blueprint instead substitutes the `AGENT` source
  constant — equivalent effect, no env dependency). The v1 test
  (`kilocode_test.go:82`) shows the foreign-entry-preservation fixture shape
  Task 5's kilo case copies.
- **`sessions` schema** — CLEAN for the Ingest upsert: `id TEXT PRIMARY KEY,
  project TEXT NOT NULL, directory TEXT NOT NULL, ownership_mode TEXT NOT NULL
  DEFAULT 'shared', title, started_at DEFAULT (datetime('now')), ended_at,
  summary` (001_schema.sql). `INSERT INTO sessions (id, project, directory) …
  ON CONFLICT(id) DO NOTHING` is valid; see the Stage 7 structural finding for
  where the `directory` value comes from.
- **v2 SaveEvent auto-register precedent** — CLEAN: `ensureSessionTx`
  (sqlite.go:623) / `ensureSessionTxn` (postgres.go:708) auto-register missing
  session rows with shared ownership inside the save tx
  (`TestSaveAutoRegistersMissingSession` pins it).
- **`project.Detector`** — CLEAN: `Resolve(cwd, explicit, git)` chain is
  explicit > env (`MNEMONIC_PROJECT`, configurable) > git toplevel > basename;
  a git lookup error falls through to basename (a lookup error is a signal, not
  a failure) — matches the threat-matrix "Git repository selection" row and
  the worktree/basename test cases.
- **DRY/dep claims** — CLEAN: `DefaultRetentionDays = 90`
  (`skillgrid-cli/internal/harnessconfig/load.go:277`); v1 pi prompt constants
  `MIN_PROMPT_LENGTH = 11` / `MAX_PROMPT_LENGTH = 2000` with the
  `messageType === "user"` filter and the `reason !== "quit"` shutdown guard
  (`plugins/pi/mnemonic/skillgrid-events.ts` and the live install at
  `~/.skill-pi/agent/extensions/mnemonic/skillgrid-events.ts:44-51`); ADR-0008
  is the `yaml`-package dependency precedent (AGENTS.md: "No new dependencies
  without an ADR … ADR-0008 is the yaml-package precedent"); ADRs 0013/0021/
  0031/0032/0043/0045 exist with the cited semantics; `glossary/02-technical-
  terms.md` and `docs/user-guide/04-hooks.md` exist; the "observation"
  memory-domain term the glossary disambiguation targets is heavily used there
  (the D1 collision is real and Task 8's rows land in the right file).
- **WAL/concurrency substrate** — CLEAN: the DSN applies
  `busy_timeout(5000)`, `journal_mode(WAL)`, `synchronous(NORMAL)`,
  `foreign_keys(1)`, `_txlock=immediate`, single-connection pool
  (sqlite.go:301-315) — concurrent ingests serialize within 5s; the
  worst-case store wait sits inside the pi/omp 30s cap and at the opencode 5s
  cap boundary (timeout kills the child → swallowed error → fail-open).
- **Placeholder scan** — CLEAN: no TBD/TODO/FIXME/"similar to Task N"/unpinned
  values in the document.
