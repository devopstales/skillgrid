# skill-pi — Implementation Blueprint (steps, files, verification)

> **Role:** this is the *how* — the implementation plan: concrete steps,
> files to touch, code, tests, and verification. The companion `briefing.md`
> is the *what* (requirements & intent). Do not rename either file.
> Assume the engineer has zero context for the codebase. DRY. YAGNI. TDD.
> Frequent commits. No placeholders.

> **For agentic workers:** REQUIRED SUB-SKILL: Use subagent-execution (recommended) or simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** PROPOSED

**Tier:** T2

**Build shape:** Two tracers, in sequence. **Tracer A (this change's core):** the full `skill-pi` module — a launcher CLI that relocates Pi's config to skill-pi's own paths (global `~/.skill-pi/` + a per-project cwd shim resolving to `<project>/.skill-pi/`), the five opencode-behavior extension ports, the permission + ticket + context + MCP-bridge + chronos extensions, the `@earendil-works/pi-tui` TUI, and the `/memory-sync` command — proven end-to-end (a `skill-pi` launch in a skillgrid repo loads memory + skills + tools from skill-pi paths, leaving `~/.pi/` untouched) before the swarm slice is layered on. **Tracer B (the swarm slice, T11–T20):** the main pi lead runs a goal-loop swarm over the existing mnemonic team plane, spawning Pi SDK subagent sessions as worker/reviewer, enforcing the two-review done gate, with opt-in per-task worktree isolation and a `goals` container so the dashboard shows goal progress. Integration risk (does the launcher actually relocate pi, and does the swarm loop actually close against the live plane) is the unknown, so it is proven first.

**Goal:** Ship `skill-pi` — an opinionated, non-fork distribution of Pi that runs with the skillgrid standards, skills, memory (mnemonic MCP), code index, events, squad, compaction, a permission extension, ticketing, a context extension, a custom TUI, and the `/memory-sync` command — all from skill-pi's own config paths, never touching a user's plain `pi` install. On top of that foundation, give skill-pi a team/swarm extension: the main pi lead runs a goal-loop swarm over the existing mnemonic team plane, spawning Pi SDK subagent sessions as worker/reviewer, enforcing the two-review done gate, with opt-in per-task worktree isolation and a `goals` container so the dashboard shows goal progress.

**Architecture:** A launcher CLI (`bin/skill-pi.js`) sets `PI_CODING_AGENT_DIR=~/.skill-pi` (global) and builds a per-project cwd shim under `~/.skill-pi/cwd/<project>/` whose `package.json` carries `piConfig.configDir = ".skill-pi"` (project), so pi resolves its config from skill-pi paths; it owns `init`/`sync`/`version` subcommands, passes everything else through verbatim, spawns (not execs) pi, and re-spawns on exit 75. The Go installer (`skillgrid-cli/`) is extended to copy `skill-pi/` → `~/.skill-pi/`, `npm ci` the pinned pi, materialize `<project>/.skill-pi/` + the shim, and put `skill-pi` on PATH. Nine TS extensions under `skill-pi/extensions/` port the opencode plugins (memory, codeindex, events, squad, compaction) to Pi's `ExtensionAPI` and add the new permission (ADR-0037), tickets (ADR-0038), context, MCP-bridge, and chronos extensions. The TUI is built on `@earendil-works/pi-tui` primitives. The swarm slice (Tracer B) is a thin pi extension (`skillgrid-squad.ts`) — the HTTP client over the team plane (mirroring `plugins/opencode/skillgrid-squad.ts`) — plus role discovery (`roles.ts`), a subagent spawner (`spawn.ts`), worktree management (`worktree.ts`), and a checkpoint pair (`goal.ts`); the Go team plane gains a `goals` table + `tasks.goal_id` (ADR-0039) and a `goal_id` param on `team_spawn_task`. The swarm loop is conversational (the lead's prompt drives it); a one-line `progress.json` is the lead's compaction checkpoint, not the goal record.

**Tech Stack:** Go 1.22+ (team plane `mnemonic/`, installer `skillgrid-cli/`), TypeScript (pi extensions, jiti-loaded, `skill-pi/`; the launcher `bin/skill-pi.js` is Node), SQLite (plane store), pi subagent SDK (sessions), git worktrees (`.worktrees/` convention). New third-party deps (each ADR-recorded): `@earendil-works/pi-coding-agent` (exact-pinned) + `@earendil-works/pi-tui` (ADR-0008 precedent); `tree-sitter`/`tree-sitter-bash` + `@pierre/diffs` for the permission extension (ADR-0037). The swarm slice (Tracer B) reuses existing machinery (git CLI, pi subagent SDK) and introduces none (ADR-0040).

**Spec:** `./briefing.md` (the what). Decisions: `./adr.md` manifest → ADR-0037/0038 (in-force, respected) + ADR-0039/0040/0041 (swarm slice) + the `@earendil-works/pi-*` dependency ADR (new, Task 1).

## Terms

- **skill-pi** — the opinionated, non-fork distribution of Pi built on the skillgrid standards (this feature).
- **cwd shim** — the per-project directory under `~/.skill-pi/cwd/<project>/` whose `package.json` carries `piConfig.configDir = ".skill-pi"`, through which pi's `join(cwd, CONFIG_DIR_NAME)` resolves to the real project's `.skill-pi/`.
- **extension port** — a `skill-pi/extensions/*.ts` file that ports the behavior of a `plugins/opencode/*.ts` plugin to Pi's `ExtensionAPI`.
- **lead** — the main pi session the user talks to; decomposes, routes, checkpoints. Not a subagent. (Sharpened this plan: the lead is a *role*, not a spawnable member.)
- **worker / reviewer** — the two v1 subagent roles (`agents/pi/roles/worker.md`, `agents/pi/roles/reviewer.md`). (Sharpened: roles are *files*, not code — ADR-0041.)
- **goal** — a `goals` row grouping tasks; the dashboard progress unit. (New — ADR-0039.)
- **worktree task** — a task whose brief carries `worktree: true`; runs in `.worktrees/swarm-<taskID>` and commits to `worktree-agent-swarm-<taskID>` (branch recorded in `tasks.worktree_branch`). (New — ADR-0040.)
- **checkpoint** — `.skill-pi/goal/progress.json`, the lead's orientation aid across compaction. Not the goal record.

## Must-Haves (goal-backward verification)

**Truths** (observable behaviors that must hold):

*Tracer A — the skill-pi foundation (R1–R10):*
- Running `skill-pi` in a skillgrid repo reads its global config from `~/.skill-pi/` (settings, mcp, trust, models) and its project config through the cwd shim to `<project-root>/.skill-pi/`; `~/.pi/` and `<project-root>/.pi/` are byte-identical before and after. (R1, R2)
- Every `skill-pi` invocation re-materializes the shim from scratch (no hash fast path); a new file added to the real project root is visible through the shim on the next launch without a manual sync. (R3)
- After first run, `<project-root>/.skill-pi/` is a real directory (not a symlink) containing `mcp.json`, `settings.json`, `SYSTEM.md`, `extensions/`, `skills/`, `prompts/`; it is gitignored (in `.gitignore`, ignored by `git status`, not untracked). (R4)
- A pi session run under `skill-pi` exposes the memory CRUD tools, code search/index tools, event capture, squad tools, and compaction summary from the five extension ports; a `mem_save` call from a pi session lands in the mnemonic store. (R5)
- With `defaultMode: "ask"` and no matching rule, an `edit` call triggers the approval prompt; a `deny` rule on a bash pattern blocks that call with `Permission denied` and no prompt; a tree-sitter parse of a piped command matches each pipe segment; "Allow always" persists to the local tier file. (R6)
- `ticket ready` returns ready-for-agent Backlog.md tasks; `ticket create` produces a `.backlog/tasks/task-NNN.md`; no `.tickets/` dir is created; a 50k-char list is truncated to 40k. (R7)
- `/skillgrid context usage` renders a usage map with a "Mnemonic" category; `/skillgrid context injections` shows hidden parts; crossing the compaction threshold emits a hint or triggers `/compact`. (R8)
- The TUI renders in an alt-screen terminal; at ≥140 cols the full layout (header + transcript + right rail) shows, below it the compact layout (bottom bar + agents widget); the context gauge escalates tone at 80% and 95%. (R9)
- `/memory-sync` writes `<project>/.skill-pi/memory/facts.jsonl` (valid JSONL, committed, `.gitattributes merge=union`); a second run does not duplicate rows; the SQLite DB is gitignored. (R10)
- After sync, a representative `~/.agents/skills/<name>/SKILL.md` is present in the pi skills location and a pi session lists it; `~/.agents/skills/` remains the single source of truth. (R11)
- `pi-version.txt` content equals the `package.json` dependency string; the version is exact (no `^`/`~`/range); `README.md` names the minimum pi version for `piConfig.configDir`. (R12)
- `skillgrid-cli install` produces `~/.skill-pi/node_modules/@earendil-works/pi-coding-agent` at the pinned version, puts `skill-pi` on PATH, and materializes `<project-root>/.skill-pi/`. (R13)
- `skill-pi version` prints the pinned pi version without entering the TUI; `skill-pi sync` re-runs the sync; `skill-pi --help` shows pi's help (not the launcher's); a pi flag like `--session` is forwarded and honored. (R14)
- `skill-pi --where` exits 0 and emits a parseable JSON object with the eight keys `skillPiRoot`, `piCli`, `agentDir`, `stateDir`, `shimDir`, `projectRoot`, `configDir`, `executable`; `configDir` equals `<project-root>/.skill-pi` and `agentDir` equals `~/.skill-pi`. (R15)
- `SYSTEM.md` contains no copy of the `gentle-ai:persona` or `gentle-ai:engram-protocol` blocks (grep count 0); the delta sections (zone discipline, commit contract) are present. (R16)
- No provider base URL is a literal in `settings.json` or `mcp.json`; the base URL is overridable so `mnemonic ctx wrap pi` can inject the loopback `:8787` base URL. (R17)
- Every pi spawn from the launcher carries `PI_SKIP_VERSION_CHECK=1` and `--no-skills --no-prompt-templates --no-themes --extension <skill-pi-root>`; pi does not print its version banner or auto-discover `~/.pi/agent/skills/`; only skill-pi's resources load (verified via `/skillgrid context injections`). (R18)
- A pi session exposes `mcp__<server>__<tool>` entries in its tool registry (mnemonic, backlog, fluxcd, codescene, trivy); the permission extension's `mcp__*` glob matches them. (R19)
- First `skill-pi` run in a fresh repo shows the trust prompt; after accepting, `~/.skill-pi/trust.json` contains a key equal to the shim's realpath; a second run shows no prompt. (R20)
- A pi session's context includes the injected date/time from `chronos.ts`, matching the system clock. (R21)
- The launcher spawns (not execs) pi; on child exit 75 it re-spawns; `/update` and `/restart` trigger the loop; a normal exit (0) does not re-spawn. (R22)

*Tracer B — the swarm slice (the team extension):*
- `team_spawn_task` with a `goal_id` creates a task whose `goal_id` column is set; the task is retrievable by goal. (swarm R6)
- A goal's tasks, listed via the plane, carry per-status counts (pending/in-progress/review_spec/done) that the dashboard renders. (swarm R6)
- The extension discovers `worker` and `reviewer` roles by globbing `<repo-root>/agents/pi/roles/*.md`; dropping a new `foo.md` makes `foo` a role with no code change. (swarm R3)
- Spawning a worker subagent for a task sets the subagent's memory `owner` to the task id and its cwd to the repo root (or the worktree path when `worktree: true`). (swarm R2, R5)
- A `worktree: true` task: the extension runs `git worktree add` under `.worktrees/`, and on completion commits to a branch `worktree-agent-swarm-<task_id>`; the branch name is recorded in the task's `worktree_branch` column (read back by the dashboard query). (swarm R5)
- A task reaches `done` only after both `spec_compliance` AND `code_quality` reviews are submitted with `passed=true`; a failed review routes the task back to the worker (re-review). (swarm R4)
- After each task step the lead's checkpoint write lands in `.skill-pi/goal/progress.json` with `current` + `status`; reading it back returns the orientation. (swarm R7)
- The swarm runs serially by default: one worker subagent is active at a time; a second is not spawned until the first's task is `done`. (swarm R8)

**Artifacts** (files that must exist with real implementation, not stubs):

*Tracer A — the skill-pi foundation:*
- `skill-pi/README.md` — names the minimum pi version for `piConfig.configDir`, the pin, and the install/sync flow.
- `skill-pi/package.json` — `name: "skill-pi"`, `piConfig: { name: "skill-pi", configDir: ".skill-pi" }`, `bin: { "skill-pi": "./bin/skill-pi.js" }`, exact-pinned `@earendil-works/pi-coding-agent` dep.
- `skill-pi/pi-version.txt` — single line, the exact pi version (matches `package.json`).
- `skill-pi/bin/skill-pi.js` — the launcher (env relocation, shim build, chdir, spawn, exit-75 loop, `--where`, owned subcommands).
- `skill-pi/mcp.json`, `skill-pi/settings.json` — pi global config (base URL overridable, no provider URL literal).
- `skill-pi/system/SYSTEM.md` — the skillgrid delta only (no persona/engram blocks).
- `skill-pi/system/APPEND_SYSTEM.md` — appendable delta.
- `skill-pi/extensions/mnemonic-memory.ts`, `mnemonic-codeindex.ts`, `skillgrid-events.ts`, `skillgrid-squad.ts`, `skillgrid-compaction.ts` — the five opencode-behavior extension ports (R5).
- `skill-pi/extensions/permission.ts` — ADR-0037 permission extension (R6).
- `skill-pi/extensions/tickets.ts` — ADR-0038 `ticket` tool + `/tickets` command (R7).
- `skill-pi/extensions/context.ts` — context extension (`/skillgrid context [usage|injections|compact]`) (R8).
- `skill-pi/extensions/mcp-bridge.ts` — registers MCP servers as native `mcp__` tools (R19).
- `skill-pi/extensions/chronos.ts` — date/time injection (R21).
- `skill-pi/tui/` — the `@earendil-works/pi-tui` TUI (right rail, header, bottom bar, petal state machine, context gauge, changes viewer, agents widget, startup banner) (R9).
- `skill-pi/prompts/memory-sync.md` — the `/memory-sync` prompt template (R10).
- `skill-pi/install/install.md`, `skill-pi/install/sync.sh` — the installer sync step (R11, R13).
- `skill-pi/skills/` — committed skill source (synced from `~/.agents/skills/`) (R11).
- `skill-pi/themes/` — TUI theme tokens (R9).
- `skill-pi/bin/skill-pi.test.js`, `skill-pi/extensions/*.test.ts`, `skill-pi/tui/*.test.ts` — the TS test harness (minimal `node --test` / vitest setup; recorded as an assumption).
- `skillgrid-cli/` (Go, modified) — `install`/`sync-repo`/`setup` extended to copy `skill-pi/` → `~/.skill-pi/`, `npm ci` the pinned pi, materialize `<project>/.skill-pi/` + the shim, put `skill-pi` on PATH (R13).
- `.skillgrid/artifacts/04-adr-00XX-skill-pi-dependencies.md` — the `@earendil-works/pi-coding-agent` + `pi-tui` dependency ADR (ADR-0008 precedent) (R12, R9).

*Tracer B — the swarm slice:*
- `skill-pi/extensions/skillgrid-squad.ts` — pi extension: 7 team-plane HTTP tools (the 6-tool port of `plugins/opencode/skillgrid-squad.ts` **plus a `squad_reviews` read tool** — the two-review gate's read path, defect A) + `swarm_spawn` + `goal_mark`/`goal_resume` + role discovery + worktree hook.
- `skill-pi/extensions/roles.ts` — glob `agents/pi/roles/*.md` → `{ name, prompt, model? }`.
- `skill-pi/extensions/spawn.ts` — build a subagent session config (role prompt, owner=task id, cwd/worktree) and spawn it via the pi subagent SDK.
- `skill-pi/extensions/worktree.ts` — `git worktree add`/commit/`worktree-agent-swarm-<task_id>` branch/cleanup.
- `skill-pi/extensions/goal.ts` — `goal_mark`/`goal_resume` over `.skill-pi/goal/progress.json`.
- `agents/pi/roles/worker.md` — v1 worker role prompt.
- `agents/pi/roles/reviewer.md` — v1 reviewer role prompt.
- `mnemonic/internal/store/migrations/056_team_goals.sql` — `goals` table + `tasks.goal_id` + `tasks.worktree_branch` (migration). Numbered above the newest existing migration (`055_context_revisions.sql`) so the filename-ordered runner applies it last.
- `mnemonic/internal/service/teams.go` — `SpawnTask` accepts `goal_id`; `ListTasks`/dashboard query groups by goal with per-status counts.
- `mnemonic/internal/mcp/tools_teams.go` — `team_spawn_task` exposes `goal_id` param.
- Dashboard view (in `skillgrid-ui/` or the existing dashboard route) rendering goal → tasks with per-status counts.
- `skill-pi/extensions/skillgrid-squad.test.ts` (or the pi extension's test harness) — unit tests for role discovery, worktree commit, checkpoint round-trip, and the two-review gate logic.

**Key links** (critical connections that must work together):

*Tracer A:*
- The launcher's `PI_CODING_AGENT_DIR` (global) and the cwd shim's `piConfig.configDir` (project) must resolve together — global to `~/.skill-pi/`, project through the shim to `<project>/.skill-pi/` — so that pi never reads or writes `~/.pi/`. A shim whose symlink does not point at the real project root makes pi resolve the wrong project config. (R1↔R2)
- `skill-pi sync` (the installer sync step) and the launcher's first-run materialization must produce the *same* `<project>/.skill-pi/` contents — a divergence means a fresh `skillgrid-cli install` and a bare `skill-pi` launch disagree about what the project config is. (R11↔R13)
- The five extension ports must register their tools in pi's registry **and** the `mcp-bridge` must register the MCP servers as `mcp__` tools **and** `permission.ts` must gate those `mcp__*` tools — an extension port that calls an MCP tool the bridge did not register, or a tool the permission ext does not match, silently fails. (R5↔R19↔R6)
- The TUI's context gauge and the context extension's Jev score must read the same token-pressure value — a gauge that escalates on a different number than the compaction advising is confusing and wrong. (R9↔R8)
- `SYSTEM.md` must reference (not duplicate) the `AGENTS.md` persona/engram blocks, and pi must read `AGENTS.md` directly for those — a duplicated persona in `SYSTEM.md` drifts from the opencode/kilo persona. (R16)
- The `pi-version.txt` pin and the `package.json` dependency string must be identical, and the installer's `npm ci` must install that exact version into `~/.skill-pi/node_modules/` — a version drift means the launcher's min-version guard (`piConfig.configDir` must exist) fires or the TUI imports a `pi-tui` API that does not exist. (R12↔R13)
- The launcher's exit-75 re-spawn loop and `/update`/`/restart` (which call `process.exit(75)`) must close the loop — a re-spawn that does not re-materialize the shim means a restart loses the project config. (R22↔R2)

*Tracer B:*
- `swarm_spawn` (extension) must set the subagent's memory `owner` to the task id **and** its cwd to the worktree path when `worktree: true` — a subagent that reads the wrong dir or writes to the wrong owner breaks the loop. (swarm R2↔R5)
- The lead's `goal_mark` checkpoint (`progress.json`) and the plane's `goals` table must agree on `current`/status — the checkpoint is the resume aid, the plane is the record; a divergence means the lead resumes on a stale task. (swarm R6↔R7)
- `worktree.ts`'s commit-on-done must run only after **both** reviews pass — committing a failed-review worktree strands an unreviewed branch. (swarm R4↔R5)
- `team_spawn_task.goal_id` (MCP/HTTP) must flow into the `tasks.goal_id` column, and the dashboard query must read that column — a spawn that drops the id shows an ungrouped task in the dashboard. (swarm R6)

**One-way-door decisions** (hard to reverse — explicit user approval before implementing):
- **ADR-0008-precedent: the `@earendil-works/pi-coding-agent` + `@earendil-works/pi-tui` npm dependency surface** — adding two third-party runtime deps to the repo. Reversible in the sense that the dep can be removed, but the exact pin + the `piConfig.configDir` min-version contract + the installer's `npm ci` flow are shaped around the pinned version, so the pin is the one-way part. User approved the "exact pin + documented min-version" approach in the brainstorming interview (Round 2, Q8).
- **ADR-0039: adding a `goals` table + nullable `tasks.goal_id` FK** — a schema migration. Nullable FK is additive (no data loss), but the table + index shape is hard to reshape later without another migration. User approved this in the brainstorming interview (chose "goals table + tasks.goal_id (one-way door)").
- **No other one-way doors.** ADR-0040 (worktree) adds a `tasks.worktree_branch` column *in the same ADR-0039 migration file* (Task 11) — additive/nullable like `goal_id`, not a separate one-way door; ADR-0041 (roles) is pure file discovery. The launcher's `PI_CODING_AGENT_DIR` relocation (R1) and the cwd-shim trick (R2) are reversible (an env var + a dir), not one-way doors.

## Global Constraints

- **Go 1.22+ minimum** (ADR-0031). A `go.mod` below 1.22 is a drift signal.
- **No new dependencies without an ADR** (AGENTS.md § Rules, ADR-0008 precedent). The `@earendil-works/pi-coding-agent` + `pi-tui` npm surface (Tasks 1) and the `tree-sitter`/`tree-sitter-bash` + `@pierre/diffs` permission surface (ADR-0037, Task 9) must be ADR-recorded before landing. The swarm slice (Tracer B) is expected to reuse existing machinery (git CLI, pi subagent SDK) and introduce none (ADR-0040).
- **Conventional commits only;** no AI-attribution trailers (commit-msg hook enforces).
- **Spec-zone changes commit before code-zone changes** (pre-commit zone guard). The `briefing.md`/`blueprint.md`/`acceptance.feature`/`tasks.md`/migration commit lands before the Go/TS code.
- **Serial development:** one change at a time (swarm R8 reflects this in the swarm's default).
- **Worktree convention:** worktrees go under `.worktrees/` (gitignored, `conventions.worktree_dir`), branch names match `precommit-guard.js`'s per-agent regex `^(agent-|worktree-agent-|worktree-wf_)[A-Za-z0-9._/-]+$` — the swarm branch is named `worktree-agent-swarm-<task_id>` (Task 16) so the pre-commit guard accepts it without an override.
- **Pi supply chain:** pi ships its own supply-chain hardening (pinned deps, shrinkwrap) — skill-pi pins the pi version exactly and does not vendor the pi monorepo (R12; findings.md). A 70+-skill eager catalog is an anti-pattern (GSD finding) — the skill sync imports from `~/.agents/skills/` but the TUI/catalog does not eager-load every skill (R11).
- **The shim is the price of pi having no `--project` flag** (findings.md): the cwd shim is mandatory for project config resolution; there is no pi-native alternative in v1.
- **BDD is always on:** every task SATISFIES a named `acceptance.feature` scenario.

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| **Process supervision (launcher spawn + exit-75 loop)** | Applicable: `bin/skill-pi.js` spawns pi as a child and intercepts exit 75 to re-spawn. | Spawn (not exec) pi with `child_process.spawn`, forward `process.argv`, and watch `exit` — re-spawn only on code 75, never on 0 or signal-driven exits; cap re-spawns (e.g. 10) to avoid a tight loop on a persistently-75 pi; forward `SIGINT`/`SIGTERM` to the child and exit with the child's code. | `skill-pi.test.js`: a fake child that exits 75 is re-spawned (assert a second spawn); a fake child that exits 0 is **not** re-spawned; a child that exits 75 eleven times in a row stops the loop (assert the cap). |
| **File I/O + symlinks (cwd shim materialization)** | Applicable: the launcher rebuilds `~/.skill-pi/cwd/<project>/` symlinks from the real project root on every launch; a broken or stale symlink makes pi resolve the wrong project config. | Rebuild the shim from scratch every launch (R3, no hash fast path): `rm -rf` the shim dir, `mkdir -p`, symlink each top-level entry of the real root, write the fixed `package.json`. Use the real (realpath'd) root as the symlink target so `join(cwd, ".skill-pi")` resolves through to `<project-root>/.skill-pi`. On a missing root, print the resolution error and exit non-zero (no pi spawn). | `skill-pi.test.js`: after `buildShim`, the shim's `package.json` has `piConfig.configDir = ".skill-pi"`; a new file added to the real root is visible through the shim on a rebuild without a manual sync; a missing root exits non-zero and does not spawn. |
| **Subprocess (npm ci in the installer)** | Applicable: `skillgrid-cli install` runs `npm ci` in `~/.skill-pi/` to install the pinned pi. | Run `npm ci` via `exec` with the `skill-pi/package.json` lockfile; on failure, surface the stderr and do not mark the install complete; the launcher's min-version guard (R12) catches a drift at run time as a second line of defense. | `skillgrid-cli` Go test (the install seam): a stubbed `npm ci` that exits non-zero makes the install report failure (no `~/.skill-pi/node_modules/@earendil-works/pi-coding-agent` marked installed). |
| **File I/O (permission settings 3-tier merge)** | Applicable: `permission.ts` reads `~/.skill-pi/permission.settings.json` (global), `<project>/.skill-pi/permission.settings.json` (project), `<project>/.skill-pi/permission.settings.local.json` (local) and merges local > project > global. | Each tier file is optional; a missing tier contributes no rules (treat as empty, not an error); the merge is a last-wins overlay by tier; "Allow always" writes a new rule to the **local** tier file (atomic write) so it does not clobber the committed project tier. | `permission.test.ts`: a rule in the project tier is overridden by the same tool+pattern in the local tier; a missing local file does not error; "Allow always" appends to the local file only (the project file is unchanged). |
| **Parsing (tree-sitter bash matching in `permission.ts`)** | Applicable: a `deny` rule on a bash pattern must match the parsed command; a parse failure must not crash the gate. | Parse the command with `tree-sitter` + `tree-sitter-bash`; a pipe `a | b` matches if **either** segment matches the rule (the rule `rm -rf *` matches `git status \| rm -rf /` via the `rm` segment); on a parse failure, fall back to wildcard matching (fail open: whole command as one token) so a syntax pi's parser chokes on is not silently allowed by a crash. | `permission.test.ts`: a tree-sitter parse of `git status | grep error` matches both `git status` and `grep error` rules; a command with a parse error (e.g. an unbalanced quote) falls back to wildcard matching and still evaluates the rule. |
| **HTTP (mnemonic MCP base URL override)** | Applicable: `settings.json`/`mcp.json` must not hardcode a provider base URL; the CTX proxy (`mnemonic ctx wrap pi`) injects a loopback `:8787` base URL. | No provider base URL literal in `settings.json`/`mcp.json` (R17); the base URL is read from an env var / settings field; the five extension ports and the `mcp-bridge` read the base URL from `SKILLGRID_CHECKPOINT_URL`/`SKILLGRID_MNEMONIC_HTTP_URL` (default `http://127.0.0.1:7438`) — the same env resolution as `plugins/opencode/*.ts`. | a grep test: `grep -rE "https?://[a-z0-9.-]+/(v1|api)" skill-pi/settings.json skill-pi/mcp.json` returns no provider base URL; a unit test: setting the override env var changes the effective base URL the extension ports use. |
| **File I/O (`/memory-sync` JSONL export)** | Applicable: `/memory-sync` writes `<project>/.skill-pi/memory/facts.jsonl` (committed, `merge=union`); re-running must not duplicate rows. | Export is a union merge: read the existing `facts.jsonl` (if any), merge by observation id (a key field), and write the union atomically; an empty store writes an empty (valid) JSONL; the SQLite DB path is gitignored, only the JSONL is committed. | `memory-sync.test.ts`: a store with N observations exports N rows; a second run with the same store exports N rows (not 2N); an empty store exports 0 rows (valid empty JSONL). |
| **Executable-file classification / process integration (pi TUI render)** | Applicable: the TUI renders in an alt-screen terminal with a responsive breakpoint (≥140 cols full vs compact). | Build on `@earendil-works/pi-tui` primitives (`Component`, `VStack`, `HStack`, `ScrollView`, `TuiAltScreen`, `matchesKey`, `visibleWidth`); the breakpoint is a pure function of the terminal width (no side effects); the context gauge tone is a pure function of the usage ratio (escalates at 80% and 95%). | `tui.test.ts`: at width ≥140 the full layout (header + transcript + right rail) is selected; at width <140 the compact layout (bottom bar + agents widget) is selected; the gauge tone escalates at 0.80 and 0.95 usage ratios. |
| **Subprocess (git worktree)** | Applicable: `worktree.ts` shells out to `git worktree add`/`commit`/`remove`. | Run git via a single `execFile` wrapper with a bounded arg array (no shell string concat of the task id); reject a `task_id` that is not `[A-Za-z0-9-]+` before it reaches a git arg; on `git` failure, do not delete the worktree (leave it for inspection) and surface the stderr. | `worktree.test.ts`: a `task_id` containing a space/quote is rejected before any `git` call (assert the exec args are the sanitized set); a `git add` that exits non-zero leaves the worktree path on disk and returns the stderr. |
| **Mnemonic tool contract (team plane HTTP)** | Applicable: the extension is an HTTP client over `POST /teams/tasks` etc. (ADR-0032 mirrors `skillgrid-squad.ts`). | Keep the 6-tool surface byte-compatible with `plugins/opencode/skillgrid-squad.ts`; add `goal_id` as an *optional* field so existing clients are unaffected; the extension never mutates the plane's status machine (it only calls the documented endpoints). | `skillgrid-squad.test.ts`: spawning a task with a `goal_id` posts the field; the plane's `tasks.goal_id` is set (integration: assert via the dashboard query). A spawn *without* `goal_id` posts `null`/omits it and the task's `goal_id` is `NULL`. |
| **Version-control automation (branch naming)** | Applicable: the extension creates a `worktree-agent-swarm-<task_id>` branch and the pre-commit guard enforces a per-agent branch regex. | Name the branch `worktree-agent-swarm-<task_id>` (matches the existing `worktree-agent-*` alternative) so `precommit-guard.js` accepts it without an override; test the name against the regex. | `worktree.test.ts`: the produced branch name matches `^(agent-|worktree-agent-|worktree-wf_)[A-Za-z0-9._/-]+$` for a real task id (a uuid task id, whose `[0-9a-f-]` chars are a subset of `[A-Za-z0-9._/-]`, matches). |
| **File I/O (checkpoint `progress.json`)** | Applicable: `goal.ts` reads/writes `.skill-pi/goal/progress.json`. | Atomic write (write temp + rename); validate the JSON on read and treat a corrupt/absent file as "no checkpoint" (start fresh), never throwing a crash. | `goal.test.ts`: a corrupt `progress.json` (truncated JSON) yields a "no checkpoint" result, not an error; a round-trip write→read returns the same `{current, status}`. |
| **Routing (dashboard goal view)** | Applicable: a new dashboard route/query renders goal → tasks. | The query is a single `SELECT` grouped by `goal_id` with `GROUP BY status`; an ungrouped task (`goal_id IS NULL`) is shown under a "(no goal)" bucket, not dropped. | dashboard test: a goal with tasks in 3 statuses renders 3 counts; a task with `goal_id IS NULL` appears under "(no goal)" and is not lost. |
| **Trust persistence (`~/.skill-pi/trust.json`)** | Applicable: pi's trust decision is keyed by the shim's canonicalized (realpath'd) path; a wrong key means the trust prompt fires every run. | The trust store lives at `~/.skill-pi/trust.json` (under the relocated global dir, R20); the key is the shim's realpath (not the shim path, not the real root) so that the per-project shim and its realpath are stable; the first-run prompt fires once per project and persists. | `skill-pi.test.js` (or `permission.test.ts`): after accepting the trust prompt, `~/.skill-pi/trust.json` contains a key equal to the shim's realpath; a second run in the same project does not fire the prompt. |
| **PR automation** | N/A: no PR creation in this change; worktree branches are committed locally, merge-back is offered, not automated. | — | — |
| **`_shared/{planning,craft,verification,knowledge}/` files** | N/A: this change adds a new top-level `skill-pi/` module + Go installer/plane changes; it does not edit shared skill/standard files. (ADR files are added under `.skillgrid/artifacts/`, which is the knowledge zone, but they are new ADRs, not edits to `_shared/`.) | — | — |

## File Structure

```
skill-pi/                              # NEW top-level dir at repo root (sibling to plugins/, mnemonic/)
  roles/                               # role prompt files (ADR-0041)
    worker.md                          # v1 worker role (frontmatter: optional model:)
    reviewer.md                        # v1 reviewer role (frontmatter: optional model:)
  extensions/                          # pi TS extensions (jiti-loaded)
    skillgrid-squad.ts                 # the 6 team-plane HTTP tools (port) + swarm_spawn + goal_mark/goal_resume + wiring
    roles.ts                           # glob roles/*.md -> Role[]
    spawn.ts                           # build + spawn a subagent session (role prompt, owner, cwd/worktree)
    worktree.ts                        # git worktree add/commit/remove + branch naming
    goal.ts                            # progress.json checkpoint (goal_mark/goal_resume)
    skillgrid-squad.test.ts            # unit tests: role discovery, worktree, checkpoint, two-review gate
  .skill-pi/                           # pi project config (gitignored; materialized by installer)
    goal/
      progress.json                    # runtime checkpoint (gitignored)

mnemonic/
  internal/store/migrations/056_team_goals.sql   # goals table + tasks.goal_id + tasks.worktree_branch (ADR-0039/0040)
  internal/service/teams.go              # SpawnTask +goal_id; ListTasksByGoal w/ per-status counts
  internal/mcp/tools_teams.go            # team_spawn_task +goal_id param
  internal/http/teams.go                 # POST /teams/tasks +goal_id; GET /teams/goals + GET /teams/goals/{id}/tasks
  internal/integration/step07_team_goals_test.go # migration + spawn-by-goal + dashboard-query integration

skillgrid-ui/  (or the existing dashboard route — locate at exec time; see Task 9)
  <goal-view route/component>            # renders goal -> tasks with per-status counts
```

**Note on the dashboard location:** the exploration confirmed the team plane's HTTP routes live in `mnemonic/internal/http/teams.go` and the squad plugin hits them over HTTP. The *dashboard* that renders goal progress is a UI concern — at execution time, locate the existing dashboard surface (grep `skillgrid-ui/` for the team/tasks view) and add the goal view there. Task 9 pins this down; the blueprint names the *query contract* (`GET /teams/goals/{id}/tasks` returns `{goal, tasks:[{id,title,status,worktree_branch}]}`) so the UI task is contract-first.

---

## Tracer A — the skill-pi foundation (R1–R22)

The foundation is proven end-to-end before the swarm slice (Tracer B) is layered on: a `skill-pi` launch in a skillgrid repo loads memory + skills + tools from skill-pi paths, leaves `~/.pi/` untouched, and exposes the permission/ticket/context/TUI/`/memory-sync` surface. The tasks below build it bottom-up: the module skeleton (T1), the launcher + shim (T2), the global config + system prompt (T3), the five opencode-behavior extension ports (T4), the four new extensions — permission, tickets, context, mcp-bridge (T5), the TUI (T6), the `/memory-sync` command + skills sync (T7), the installer extension + `--where` + trust (T8), the chronos + version-pair + supply-chain tasks (T9), and the foundation e2e tracer (T10).

### Task 1: `skill-pi/` module skeleton + the `@earendil-works/pi-*` dependency ADR (R12)

> ⚠ **one-way:** ADR-0008-precedent — adding `@earendil-works/pi-coding-agent` + `@earendil-works/pi-tui` is the one-way part (exact pin + the `piConfig.configDir` min-version contract + the installer `npm ci` flow are shaped around the pinned version). User approved the "exact pin + documented min-version" approach (brainstorming Round 2, Q8).

**Files:**
- Create: `skill-pi/` (top-level dir at repo root, sibling to `plugins/`, `mnemonic/`)
- Create: `skill-pi/package.json` — `name: "skill-pi"`, `piConfig: { name: "skill-pi", configDir: ".skill-pi" }`, `bin: { "skill-pi": "./bin/skill-pi.js" }`, exact-pinned `@earendil-works/pi-coding-agent` dep (the pin chosen at exec time from findings.md; no `^`/`~`/range).
- Create: `skill-pi/pi-version.txt` — single line, the exact pi version (must equal the `package.json` dependency string).
- Create: `.skillgrid/artifacts/04-adr-0042-skill-pi-dependencies.md` — the `@earendil-works/pi-coding-agent` + `@earendil-works/pi-tui` dependency ADR (ADR-0008 precedent; names the min pi version for `piConfig.configDir`, the pin, the supply-chain posture: pinned deps + shrinkwrap, no vendoring).
- Create: `skill-pi/README.md` — names the minimum pi version for `piConfig.configDir`, the pin, and the install/sync flow.

**Interfaces:**
- Consumes: findings.md (the pi version + the `piConfig.configDir` min-version contract).
- Produces: a `skill-pi/` module with its identity (`package.json`), its version pin (`pi-version.txt`), and its dependency ADR (in `.skillgrid/artifacts/`, indexed in `.skillgrid/ASSUMPTIONS.md` § `LOCKED`).
- Seam: the `package.json` + `pi-version.txt` pair — every later task (launcher, installer, TUI) reads these; a version drift between them is the single most likely silent failure (R12↔R13).
- Deletion test: removing `package.json`'s `piConfig` block makes pi ignore `configDir` and fall back to `~/.pi/` (violates R1). Removing `pi-version.txt` leaves the launcher's min-version guard (R12) with nothing to read.
- Adapters: 1 (the real `package.json`); the test asserts the pair's equality.

**SATISFIES:** `skill-pi-module-skeleton`

- [ ] **Step 1: Write the failing test**
  A node test (the TS harness, minimal `node --test` — recorded as an assumption) that reads `skill-pi/package.json` and `skill-pi/pi-version.txt` and asserts the dependency string for `@earendil-works/pi-coding-agent` is exact (no `^`/`~`/range) and equals the `pi-version.txt` content.
- [ ] **Step 2: Run to verify it fails**
  Run: `node --test skill-pi/version.test.js`. Expected: FAIL — files not written.
- [ ] **Step 3: Write the files**
  Create the `skill-pi/` dir, the `package.json` (with the `piConfig` identity + exact pin + `bin`), `pi-version.txt` (matching pin), the ADR `04-adr-0042-skill-pi-dependencies.md` (Context/Decision/Consequences; names the min pi version for `configDir`), the `README.md`. Index the ADR in `.skillgrid/ASSUMPTIONS.md` § `LOCKED`.
- [ ] **Step 4: Run to verify it passes**
  Run: `node --test skill-pi/version.test.js`. Expected: PASS.
- [ ] **Step 5: Commit**
  ```bash
  git add skill-pi/ .skillgrid/artifacts/04-adr-0042-skill-pi-dependencies.md .skillgrid/ASSUMPTIONS.md skill-pi/version.test.js
  git commit -m "feat(skill-pi): module skeleton + pi dependency ADR (ADR-0042)"
  ```

### Task 2: Launcher CLI + cwd shim (R1, R2, R3, R14, R15, R18, R20, R22)

**Files:**
- Create: `skill-pi/bin/skill-pi.js` — the launcher (Node, on PATH via `package.json` `bin`).
- Create: `skill-pi/bin/skill-pi.test.js` — the launcher tests (the process-supervision + shim-materialization threat rows).

**Interfaces:**
- Consumes: `~/.skill-pi/` (the relocated global dir, set via `PI_CODING_AGENT_DIR`); the real project root (realpath'd); the pinned pi CLI (`~/.skill-pi/node_modules/.bin/pi` or the global `pi` at the pinned version).
- Produces: a `skill-pi` executable that: (a) sets `PI_CODING_AGENT_DIR=~/.skill-pi` before spawning pi (R1 — global relocation); (b) builds the cwd shim `~/.skill-pi/cwd/<project>/` from scratch every launch (R3 — `rm -rf`, `mkdir -p`, symlink each top-level entry of the real root, write the fixed `package.json` with `piConfig.configDir = ".skill-pi"`; R2 — project config resolves through the shim to `<project-root>/.skill-pi/`); (c) chdirs into the shim (so `join(cwd, ".skill-pi")` resolves to the real project config); (d) spawns (not execs) pi with `PI_SKIP_VERSION_CHECK=1 --no-skills --no-prompt-templates --no-themes --extension <skill-pi-root>` forwarded (R18); (e) re-spawns on child exit 75, capped (R22); (f) owns `init`/`sync`/`version` subcommands (handled before handoff) and passes everything else through verbatim (R14); (g) `--where` exits 0 with the 8-key JSON (R15).
- Seam: the launcher process (the single entry point between the user and pi) — the shim build, the env relocation, and the spawn loop are its three internal functions.
- Deletion test: removing the `PI_CODING_AGENT_DIR` env set makes pi read `~/.pi/` (violates R1). Removing the shim build makes pi's `join(cwd, ".skill-pi")` resolve to the real root's `.skill-pi/` only if the user has one — a fresh repo with no `.skill-pi/` gets no project config (violates R4). Removing the exit-75 loop makes `/update`/`/restart` kill the session (violates R22).
- Adapters: 1 (the real pi child process); the tests stub the child (a fake node script that exits with a given code) to assert the re-spawn loop.

**SATISFIES:** `skill-pi-launcher`, `cwd-shim`, `spawn-respawn-loop`, `owned-subcommands`, `where-json`, `trust-first-run`

- [ ] **Step 1: Write the failing tests**
  `skill-pi.test.js`: (a) a fake child that exits 75 is re-spawned (assert a second spawn); a fake child that exits 0 is not re-spawned; a child that exits 75 eleven times in a row stops the loop (the cap). (b) after `buildShim`, the shim's `package.json` has `piConfig.configDir = ".skill-pi"`; a new file added to the real root is visible through the shim on a rebuild without a manual sync; a missing root exits non-zero and does not spawn. (c) `--where` emits the 8-key JSON with `configDir === <project-root>/.skill-pi` and `agentDir === ~/.skill-pi`. (d) `version` prints the pinned version without entering the TUI; a pi flag like `--session` is forwarded (assert the child's argv contains it). (e) the spawn env contains `PI_SKIP_VERSION_CHECK=1` and the `--no-skills --no-prompt-templates --no-themes --extension <root>` args. (f) on a fresh (no `~/.skill-pi/trust.json`) run the trust prompt fires; after accepting, `~/.skill-pi/trust.json` contains a key equal to the shim's realpath; a second run shows no prompt.
- [ ] **Step 2: Run to verify it fails**
  Run: `node --test skill-pi/bin/skill-pi.test.js`. Expected: FAIL — launcher not written.
- [ ] **Step 3: Implement the launcher**
  `skill-pi.js`: resolve the real project root (walk up to `.git` or the configured root; realpath it). `buildShim(realRoot)`: `rm -rf` the shim dir under `~/.skill-pi/cwd/<project>/`, `mkdir -p`, symlink each top-level entry of `realRoot` into the shim, write the fixed `package.json` (`piConfig.configDir = ".skill-pi"`). On a missing root, print the resolution error and exit non-zero (no pi spawn). Set `PI_CODING_AGENT_DIR=~/.skill-pi`. `spawnPi(argv)`: `child_process.spawn("pi", [...ownedSubcommandPassThrough, "--no-skills", "--no-prompt-templates", "--no-themes", "--extension", skillPiRoot, ...argv])` with the env; on `exit`, if code === 75 and respawnCount < 10, re-run `buildShim` + `spawnPi` (re-materialize the shim so a restart re-resolves project config — R22↔R2); else exit with the child's code. Forward `SIGINT`/`SIGTERM` to the child. Own `init`/`sync`/`version` (parse before handoff); `--where` prints the 8-key JSON and exits 0. Trust: on a fresh run (no `~/.skill-pi/trust.json` entry for the shim realpath), prompt; on accept, write the key (atomic).
- [ ] **Step 4: Run to verify it passes**
  Run: `node --test skill-pi/bin/skill-pi.test.js`. Expected: PASS.
- [ ] **Step 5: Commit**
  ```bash
  git add skill-pi/bin/skill-pi.js skill-pi/bin/skill-pi.test.js
  git commit -m "feat(skill-pi): launcher CLI + cwd shim + exit-75 respawn loop"
  ```

### Task 3: Global config + system prompt (R16, R17)

**Files:**
- Create: `skill-pi/mcp.json` — pi global MCP config (no provider base URL literal; the base URL is overridable via env).
- Create: `skill-pi/settings.json` — pi global settings (no provider base URL literal; the base URL is read from `SKILLGRID_CHECKPOINT_URL`/`SKILLGRID_MNEMONIC_HTTP_URL`, default `http://127.0.0.1:7438`).
- Create: `skill-pi/system/SYSTEM.md` — the skillgrid delta only (zone discipline, commit contract); references (not duplicates) the `AGENTS.md` persona/engram blocks.
- Create: `skill-pi/system/APPEND_SYSTEM.md` — appendable delta (the lead's goal-loop prompt lands here in Task 20).
- Create: `skill-pi/version.test.js` (extend — the no-base-URL-literal grep).

**Interfaces:**
- Consumes: the opencode plugins' base-URL resolution (`SKILLGRID_CHECKPOINT_URL`/`SKILLGRID_MNEMONIC_HTTP_URL` default `http://127.0.0.1:7438` — the same env resolution as `plugins/opencode/*.ts`).
- Produces: pi global config that (a) carries no provider base URL literal (R17), (b) lets `mnemonic ctx wrap pi` inject the loopback `:8787` base URL via the env override (R17), and (c) a `SYSTEM.md` that is a pure delta — zero grep hits for the `gentle-ai:persona` / `gentle-ai:engram-protocol` blocks (R16).
- Seam: the two config files + the system prompt — the extension ports (Task 4) and the mcp-bridge (Task 5) read the base URL from these; the TUI (Task 6) reads the system prompt for its header.
- Deletion test: a provider base URL literal in `settings.json` makes `mnemonic ctx wrap pi`'s override a no-op (violates R17). A duplicated persona block in `SYSTEM.md` drifts from the opencode/kilo persona (violates R16).
- Adapters: 1 (the real pi reading these files); the tests are grep + a base-URL-override unit test.

**SATISFIES:** `global-config-no-url`, `system-delta-only`

- [ ] **Step 1: Write the failing tests**
  (a) `grep -rE "https?://[a-z0-9.-]+/(v1|api)" skill-pi/settings.json skill-pi/mcp.json` returns no provider base URL (the `127.0.0.1:7438` default is a loopback, not a provider — allow it; reject a remote `https://.../v1` or `https://.../api`). (b) a unit test: setting `SKILLGRID_MNEMONIC_HTTP_URL` changes the effective base URL the extension ports resolve (assert the resolution function reads the env, not a literal). (c) `grep -c "gentle-ai:persona\|gentle-ai:engram-protocol" skill-pi/system/SYSTEM.md` is 0.
- [ ] **Step 2: Run to verify it fails**
  Run: `node --test skill-pi/version.test.js -t config`. Expected: FAIL — files not written.
- [ ] **Step 3: Write the config + system prompt**
  `mcp.json` + `settings.json`: the pi global config with the base URL read from the env vars (the loopback default, no remote literal). `SYSTEM.md`: the skillgrid delta (zone discipline, commit contract) with a pointer to `AGENTS.md` for the persona/engram blocks — no copy of them. `APPEND_SYSTEM.md`: an empty appendable delta (the lead prompt lands in Task 20).
- [ ] **Step 4: Run to verify it passes**
  Run: `node --test skill-pi/version.test.js -t config`. Expected: PASS.
- [ ] **Step 5: Commit**
  ```bash
  git add skill-pi/mcp.json skill-pi/settings.json skill-pi/system/ skill-pi/version.test.js
  git commit -m "feat(skill-pi): global config (no provider URL) + system prompt delta"
  ```

### Task 4: The five opencode-behavior extension ports (R5)

**Files:**
- Create: `skill-pi/extensions/mnemonic-memory.ts` — port of `plugins/opencode/mnemonic-memory.ts` to Pi's `ExtensionAPI`.
- Create: `skill-pi/extensions/mnemonic-codeindex.ts` — port of `plugins/opencode/mnemonic-codeindex.ts`.
- Create: `skill-pi/extensions/skillgrid-events.ts` — port of `plugins/opencode/skillgrid-events.ts`.
- Create: `skill-pi/extensions/skillgrid-compaction.ts` — port of `plugins/opencode/skillgrid-compaction.ts`.
- Create: `skill-pi/extensions/skillgrid-squad.ts` — the squad port is built in Tracer B (Task 14); this task creates the **five** foundation ports (memory, codeindex, events, compaction) and reserves the squad slot.
- Create: `skill-pi/extensions/*.test.ts` — the port tests (a `mem_save` call lands in the mnemonic store; the code search tools return results; the event capture records; the compaction summary emits).

**Interfaces:**
- Consumes: the mnemonic MCP/HTTP base URL (Task 3's env resolution); the five `plugins/opencode/*.ts` plugins (the behavior to port — `tool()` signatures, the `BASE_URL` resolution, the in-memory failure latch pattern).
- Produces: pi `tool(...)` registrations: the memory CRUD tools (`mem_save`, `mem_search`, `mem_get_observation`, `mem_session_start`, `mem_session_end`, `mem_context`, `mem_session_summary`, …), the code search/index tools (`code_search`, `code_read`, `code_status`, `code_index`, …), the event capture (`events_*`), and the compaction summary — the same names/params/behavior as the opencode plugins, re-registered under Pi's `ExtensionAPI.tool(name, {description, parameters}, handler)`.
- Seam: the HTTP client wrapper (one function per endpoint, the `BASE_URL`/`AGENT` env resolution) — the pi `tool(...)` registrations are adapters over it.
- Deletion test: removing a port removes that capability from a pi session (a `mem_save` from pi does not land in the store). The opencode plugin is the parallel reference, not a shared adapter.
- Adapters: 1 (the pi `tool` shape) per port over the shared HTTP wrapper.

**SATISFIES:** `extension-ports-register`

- [ ] **Step 1: Write the failing tests**
  For each port: stub the HTTP layer, assert the registered tools post/get the right endpoint with the right body (byte-compatible with the opencode plugin's tool names + params), and that a mutation failure sets the in-memory failure latch (port the `SKILLGRID_SQUAD_FAILURE`-style handoff). A `mem_save` call lands in the mnemonic store (integration: a local store).
- [ ] **Step 2: Run to verify it fails**
  Run: `node --test skill-pi/extensions/*.test.ts -t port`. Expected: FAIL — ports not written.
- [ ] **Step 3: Port the five extensions**
  Copy each `plugins/opencode/*.ts`'s tool logic into `skill-pi/extensions/<name>.ts`, replacing opencode's `tool(...)` API with pi's `ExtensionAPI.tool(name, {description, parameters}, handler)` (confirm the exact pi registration signature from findings.md). Keep `BASE_URL`/`AGENT` env resolution identical to the opencode plugin (read from the Task 3 env vars, not a literal).
- [ ] **Step 4: Run to verify it passes**
  Run: `node --test skill-pi/extensions/*.test.ts -t port`. Expected: PASS.
- [ ] **Step 5: Commit**
  ```bash
  git add skill-pi/extensions/mnemonic-memory.ts skill-pi/extensions/mnemonic-codeindex.ts skill-pi/extensions/skillgrid-events.ts skill-pi/extensions/skillgrid-compaction.ts skill-pi/extensions/*.test.ts
  git commit -m "feat(skill-pi): five opencode-behavior extension ports"
  ```

### Task 5: The four new extensions — permission, tickets, context, mcp-bridge (R6, R7, R8, R19)

**Files:**
- Create: `skill-pi/extensions/permission.ts` — ADR-0037 permission extension (R6).
- Create: `skill-pi/extensions/tickets.ts` — ADR-0038 `ticket` tool + `/tickets` command (R7).
- Create: `skill-pi/extensions/context.ts` — context extension (`/skillgrid context [usage|injections|compact]`) (R8).
- Create: `skill-pi/extensions/mcp-bridge.ts` — registers MCP servers as native `mcp__` tools (R19).
- Create: `skill-pi/extensions/*.test.ts` (extend — the permission/tickets/context/mcp-bridge cases, the threat-matrix RED tests for the 3-tier merge, the tree-sitter bash matching, the 50k-char truncation).

**Interfaces:**
- Consumes: `~/.skill-pi/permission.settings.json` (global), `<project>/.skill-pi/permission.settings.json` (project), `<project>/.skill-pi/permission.settings.local.json` (local); the Backlog.md tracker (`.skillgrid/tasks/`, the `backlog` MCP); the pi tool registry (for the `mcp__*` glob + the context usage map); `tree-sitter`/`tree-sitter-bash` + `@pierre/diffs` (ADR-0037).
- Produces: (a) `permission.ts`: a 3-tier merge (local > project > global, each tier optional, a missing tier contributes no rules) + a tree-sitter bash matcher (a pipe `a | b` matches if either segment matches; a parse failure falls back to wildcard matching, fail open) + an `edit`-call gate (with `defaultMode: "ask"` and no matching rule, an approval prompt; a `deny` rule on a bash pattern blocks with `Permission denied` and no prompt) + "Allow always" persists to the local tier file (atomic write). (b) `tickets.ts`: a `ticket` tool (`ticket ready` returns ready-for-agent Backlog.md tasks; `ticket create` produces a `.backlog/tasks/task-NNN.md`; a 50k-char list is truncated to 40k) + a `/tickets` command. (c) `context.ts`: `/skillgrid context usage` (a usage map with a "Mnemonic" category), `/skillgrid context injections` (the hidden parts), and a compaction-threshold hint (crossing the threshold emits a hint or triggers `/compact`). (d) `mcp-bridge.ts`: registers the MCP servers (mnemonic, backlog, fluxcd, codescene, trivy) as native `mcp__<server>__<tool>` tools in pi's registry — the permission ext's `mcp__*` glob matches them.
- Seam: the four extensions' `tool(...)` + command registrations — the permission gate is the cross-cutting seam (it must match every `mcp__*` tool the bridge registers, or those tools silently fail).
- Deletion test: removing the mcp-bridge makes the five extension ports' MCP calls fail (R5↔R19). Removing the permission ext makes `edit` calls ungated (violates R6). Removing the context ext makes the TUI's context gauge (Task 6) have no Jev score to read (violates R8↔R9).
- Adapters: 4 (one per extension) over the shared pi `ExtensionAPI` + the MCP/Backlog HTTP.

**SATISFIES:** `permission-gate`, `ticket-tool`, `context-command`, `mcp-bridge-registry`

- [ ] **Step 1: Write the failing tests**
  Permission: a rule in the project tier is overridden by the same tool+pattern in the local tier; a missing local file does not error; "Allow always" appends to the local file only (the project file is unchanged); a tree-sitter parse of `git status | grep error` matches both `git status` and `grep error` rules; a command with a parse error (an unbalanced quote) falls back to wildcard matching and still evaluates the rule; a `deny` rule on `rm -rf *` blocks `git status | rm -rf /` via the `rm` segment. Tickets: `ticket ready` returns ready-for-agent tasks; `ticket create` produces `.backlog/tasks/task-NNN.md`; no `.tickets/` dir is created; a 50k-char list is truncated to 40k. Context: `/skillgrid context usage` renders a "Mnemonic" category; `/skillgrid context injections` shows hidden parts; crossing the threshold emits a hint. Mcp-bridge: a pi session exposes `mcp__<server>__<tool>` entries for mnemonic/backlog/fluxcd/codescene/trivy; the permission ext's `mcp__*` glob matches them.
- [ ] **Step 2: Run to verify it fails**
  Run: `node --test skill-pi/extensions/*.test.ts -t "permission|ticket|context|bridge"`. Expected: FAIL.
- [ ] **Step 3: Implement the four extensions**
  `permission.ts`: read the 3 tier files (each optional), merge last-wins by tier (local > project > global); parse the command with `tree-sitter` + `tree-sitter-bash` (a pipe matches if either segment matches; a parse failure falls back to wildcard matching, fail open); gate `edit` calls (`defaultMode: "ask"` → approval prompt; a `deny` rule → `Permission denied`, no prompt); "Allow always" writes a new rule to the local tier file (atomic write). ADR-record the `tree-sitter`/`tree-sitter-bash` + `@pierre/diffs` deps (ADR-0037 precedent). `tickets.ts`: the `ticket` tool (ready/create, the Backlog.md adapter per ADR-0038) + the `/tickets` command; the 40k-char truncation. `context.ts`: the `/skillgrid context [usage|injections|compact]` commands; the Jev score the TUI gauge reads (Task 6). `mcp-bridge.ts`: register the MCP servers as `mcp__<server>__<tool>` tools.
- [ ] **Step 4: Run to verify it passes**
  Run: `node --test skill-pi/extensions/*.test.ts -t "permission|ticket|context|bridge"`. Expected: PASS.
- [ ] **Step 5: Commit**
  ```bash
  git add skill-pi/extensions/permission.ts skill-pi/extensions/tickets.ts skill-pi/extensions/context.ts skill-pi/extensions/mcp-bridge.ts skill-pi/extensions/*.test.ts
  git commit -m "feat(skill-pi): permission + tickets + context + mcp-bridge extensions"
  ```

### Task 6: The `@earendil-works/pi-tui` TUI (R9)

**Files:**
- Create: `skill-pi/tui/` — the TUI on `@earendil-works/pi-tui` primitives (`Component`, `VStack`, `HStack`, `ScrollView`, `TuiAltScreen`, `matchesKey`, `visibleWidth`).
- Create: `skill-pi/themes/` — TUI theme tokens.
- Create: `skill-pi/tui/*.test.ts` — the TUI tests (the breakpoint + gauge-tone threat rows).

**Interfaces:**
- Consumes: `@earendil-works/pi-tui` (the pin from Task 1); the context extension's Jev score (Task 5) for the context gauge; the pi session's transcript + agent state.
- Produces: a TUI that renders in an alt-screen terminal; at ≥140 cols the full layout (header + transcript + right rail) shows, below it the compact layout (bottom bar + agents widget); the context gauge escalates tone at 80% and 95% usage; a changes viewer; a petal state machine; a startup banner.
- Seam: the breakpoint (a pure function of the terminal width) + the gauge tone (a pure function of the usage ratio) — both are pure, testable, and side-effect-free.
- Deletion test: removing the TUI makes `skill-pi` fall back to pi's default UI (no skillgrid chrome, no context gauge). A gauge that escalates on a different number than the compaction advising (Task 5's Jev score) is confusing and wrong (R9↔R8).
- Adapters: 1 (the real `pi-tui` runtime); the tests assert the pure functions (breakpoint, gauge tone) directly.

**SATISFIES:** `tui-breakpoint`, `tui-gauge-tone`

- [ ] **Step 1: Write the failing tests**
  `tui.test.ts`: at width ≥140 the full layout (header + transcript + right rail) is selected; at width <140 the compact layout (bottom bar + agents widget) is selected; the gauge tone escalates at 0.80 and 0.95 usage ratios. The breakpoint + gauge tone are pure functions — test them directly, no render.
- [ ] **Step 2: Run to verify it fails**
  Run: `node --test skill-pi/tui/*.test.ts`. Expected: FAIL — TUI not written.
- [ ] **Step 3: Implement the TUI**
  Build on `@earendil-works/pi-tui`: the `TuiAltScreen` root, the header (session title + context gauge), the transcript (`ScrollView`), the right rail (agents widget + changes viewer), the bottom bar (compact layout). The breakpoint is `width >= 140 ? full : compact` (a pure function). The gauge tone is `ratio >= 0.95 ? "critical" : ratio >= 0.80 ? "warning" : "normal"` (a pure function, reads the Task 5 Jev score). The theme tokens live in `skill-pi/themes/`.
- [ ] **Step 4: Run to verify it passes**
  Run: `node --test skill-pi/tui/*.test.ts`. Expected: PASS.
- [ ] **Step 5: Commit**
  ```bash
  git add skill-pi/tui/ skill-pi/themes/
  git commit -m "feat(skill-pi): pi-tui TUI (breakpoint + context gauge)"
  ```

### Task 7: `/memory-sync` command + skills sync (R10, R11)

**Files:**
- Create: `skill-pi/prompts/memory-sync.md` — the `/memory-sync` prompt template (R10).
- Create: `skill-pi/install/sync.sh` — the installer sync step (R11, R13).
- Create: `skill-pi/install/install.md` — the install flow doc.
- Create: `skill-pi/skills/` — committed skill source (synced from `~/.agents/skills/`) (R11).
- Create: `skill-pi/extensions/memory-sync.test.ts` — the JSONL export test (the threat-matrix RED test).

**Interfaces:**
- Consumes: the mnemonic store (the observations to export); `~/.agents/skills/` (the single source of truth for skills); the installer (`skillgrid-cli`, Task 8).
- Produces: (a) `/memory-sync`: writes `<project>/.skill-pi/memory/facts.jsonl` (valid JSONL, committed, `.gitattributes merge=union`); a union merge by observation id (a re-run does not duplicate rows); an empty store writes an empty (valid) JSONL; the SQLite DB path is gitignored, only the JSONL is committed. (b) the skills sync: after sync, a representative `~/.agents/skills/<name>/SKILL.md` is present in the pi skills location and a pi session lists it; `~/.agents/skills/` remains the single source of truth (the sync imports from it; the TUI/catalog does not eager-load every skill — the 70+-skill anti-pattern, Global Constraints).
- Seam: the JSONL export (a union merge by observation id) — a duplicate-on-rerun is the silent failure (violates R10).
- Deletion test: removing the union merge makes a second `/memory-sync` duplicate every row. Removing the skills sync makes pi's skills location empty (a pi session lists no skillgrid skills, violates R11).
- Adapters: 1 (the real mnemonic store) for the export; the sync is a file copy from `~/.agents/skills/`.

**SATISFIES:** `memory-sync-jsonl`, `skills-sync`

- [ ] **Step 1: Write the failing tests**
  `memory-sync.test.ts`: a store with N observations exports N rows; a second run with the same store exports N rows (not 2N); an empty store exports 0 rows (valid empty JSONL). Skills sync: after `sync.sh`, a representative `~/.agents/skills/<name>/SKILL.md` is present in the pi skills location; `~/.agents/skills/` is unchanged (still the source of truth).
- [ ] **Step 2: Run to verify it fails**
  Run: `node --test skill-pi/extensions/memory-sync.test.ts`. Expected: FAIL.
- [ ] **Step 3: Implement**
  `/memory-sync` (the `memory-sync.md` prompt + the export logic): read the existing `facts.jsonl` (if any), merge by observation id (a key field), write the union atomically; an empty store writes an empty JSONL; add `<project>/.skill-pi/memory/facts.jsonl` to `.gitattributes` with `merge=union`; gitignore the SQLite DB path. `sync.sh`: copy `~/.agents/skills/` → the pi skills location (the sync is a copy; the source is never modified).
- [ ] **Step 4: Run to verify it passes**
  Run: `node --test skill-pi/extensions/memory-sync.test.ts`. Expected: PASS.
- [ ] **Step 5: Commit**
  ```bash
  git add skill-pi/prompts/memory-sync.md skill-pi/install/ skill-pi/skills/ skill-pi/extensions/memory-sync.test.ts .gitattributes
  git commit -m "feat(skill-pi): /memory-sync JSONL export + skills sync"
  ```

### Task 8: Installer extension + `--where` + trust persistence (R13, R15, R20)

**Files:**
- Modify: `skillgrid-cli/` (Go) — `install`/`sync-repo`/`setup` extended to copy `skill-pi/` → `~/.skill-pi/`, `npm ci` the pinned pi, materialize `<project>/.skill-pi/` + the shim, put `skill-pi` on PATH (R13).
- Create: `skillgrid-cli/` (Go) — the install-seam test (a stubbed `npm ci` that exits non-zero makes the install report failure).
- Modify: `skill-pi/bin/skill-pi.js` (Task 2) — confirm `--where` (R15) and the trust persistence (R20) are wired (the `--where` JSON + the `~/.skill-pi/trust.json` realpath key).

**Interfaces:**
- Consumes: `skill-pi/` (the module to copy); the pinned pi (Task 1's `pi-version.txt`/`package.json`); the real project root.
- Produces: `skillgrid-cli install` that (a) copies `skill-pi/` → `~/.skill-pi/`, (b) runs `npm ci` in `~/.skill-pi/` (installs the pinned pi into `~/.skill-pi/node_modules/@earendil-works/pi-coding-agent` at the pinned version), (c) materializes `<project>/.skill-pi/` (the real project config) + the shim `~/.skill-pi/cwd/<project>/`, (d) puts `skill-pi` on PATH. A failed `npm ci` surfaces the stderr and does not mark the install complete. `--where` (the launcher, Task 2) exits 0 with the 8-key JSON. Trust (the launcher, Task 2): the first `skill-pi` run in a fresh repo shows the trust prompt; after accepting, `~/.skill-pi/trust.json` contains a key equal to the shim's realpath; a second run shows no prompt.
- Seam: the `npm ci` subprocess (the installer) + the `--where`/trust (the launcher) — a failed `npm ci` that marks the install complete leaves `~/.skill-pi/node_modules/` empty (the launcher's min-version guard, R12, catches it at run time as a second line of defense).
- Deletion test: removing the `npm ci` step leaves `~/.skill-pi/node_modules/` empty → `skill-pi` cannot spawn pi. Removing the trust persistence makes the prompt fire every run (violates R20). Removing `--where` removes the debug surface (violates R15).
- Adapters: 1 (the real `npm ci` subprocess); the test stubs it.

**SATISFIES:** `installer-npm-ci`, `where-json`, `trust-first-run`

- [ ] **Step 1: Write the failing test**
  Go test (the install seam): a stubbed `npm ci` that exits non-zero makes the install report failure (no `~/.skill-pi/node_modules/@earendil-works/pi-coding-agent` marked installed). A successful stubbed `npm ci` + the copy + the materialization produces `~/.skill-pi/node_modules/` at the pinned version, `<project>/.skill-pi/` materialized, the shim built, and `skill-pi` on PATH. (The `--where` + trust cases are in Task 2's `skill-pi.test.js` — re-run them here to confirm the installer's materialization satisfies them.)
- [ ] **Step 2: Run to verify it fails**
  Run: `go test ./skillgrid-cli/ -run TestInstall -v`. Expected: FAIL — install not extended.
- [ ] **Step 3: Extend the installer**
  `skillgrid-cli` Go: `install` copies `skill-pi/` → `~/.skill-pi/`, runs `npm ci` in `~/.skill-pi/` (the `skill-pi/package.json` lockfile; on failure, surface the stderr, do not mark complete), materializes `<project>/.skill-pi/` + the shim, puts `skill-pi` on PATH. `sync-repo`/`setup` mirror the sync step (R11↔R13: `skillgrid-cli install` and a bare `skill-pi` launch produce the *same* `<project>/.skill-pi/` contents).
- [ ] **Step 4: Run to verify it passes**
  Run: `go test ./skillgrid-cli/ -run TestInstall -v`. Expected: PASS.
- [ ] **Step 5: Commit**
  ```bash
  git add skillgrid-cli/ skill-pi/bin/skill-pi.js
  git commit -m "feat(skillgrid-cli): install skill-pi (npm ci + materialize + PATH)"
  ```

### Task 9: Chronos injection + version-pair + supply-chain (R12, R21)

**Files:**
- Create: `skill-pi/extensions/chronos.ts` — date/time injection (R21).
- Create: `skill-pi/version.test.js` (extend — the version-pair + supply-chain assertions).

**Interfaces:**
- Consumes: the system clock; the pinned pi (Task 1).
- Produces: (a) `chronos.ts`: injects the current date/time into a pi session's context, matching the system clock (R21). (b) the version-pair guard: `pi-version.txt` content equals the `package.json` dependency string (R12 — re-asserted here as a supply-chain check). (c) the supply-chain posture: the pi deps are exact-pinned (no `^`/`~`/range), the shrinkwrap is present, no vendoring of the pi monorepo (Global Constraints, R12).
- Seam: the `chronos.ts` injection + the version-pair equality — a version drift between `pi-version.txt` and `package.json` is the single most likely silent failure (R12↔R13).
- Deletion test: removing `chronos.ts` makes a pi session's context lack the injected date/time (violates R21). Removing the version-pair guard lets the pin drift (violates R12).
- Adapters: 1 (the real system clock) for chronos; the version-pair is a file-equality check.

**SATISFIES:** `chronos-injection`, `version-pair-equality`

- [ ] **Step 1: Write the failing tests**
  (a) a pi session's context includes the injected date/time from `chronos.ts`, matching the system clock (assert the injected value is within 1s of `new Date()`). (b) `pi-version.txt` content equals the `package.json` dependency string (re-assert the Task 1 test here as the supply-chain gate). (c) the `package.json` deps for `@earendil-works/pi-*` are exact (no `^`/`~`/range).
- [ ] **Step 2: Run to verify it fails**
  Run: `node --test skill-pi/version.test.js -t "chronos|supply"`. Expected: FAIL.
- [ ] **Step 3: Implement**
  `chronos.ts`: a pi extension that injects the current date/time into the session context (a `tool(...)` or a context-injection hook per the pi `ExtensionAPI`; confirm the exact hook from findings.md). The version-pair + supply-chain assertions extend `version.test.js`.
- [ ] **Step 4: Run to verify it passes**
  Run: `node --test skill-pi/version.test.js -t "chronos|supply"`. Expected: PASS.
- [ ] **Step 5: Commit**
  ```bash
  git add skill-pi/extensions/chronos.ts skill-pi/version.test.js
  git commit -m "feat(skill-pi): chronos injection + version-pair + supply-chain gate"
  ```

### Task 10: Foundation e2e tracer (R1–R22 proof)

**Files:**
- Create: `skill-pi/e2e.test.js` — the foundation e2e integration test (a real `skill-pi` launch in a temp skillgrid repo).
- Modify: `skill-pi/` (confirm the five ports + the four new extensions + the TUI + `/memory-sync` + the launcher + the installer are all wired).

**Interfaces:**
- Consumes: every module from Tasks 1–9.
- Produces: a working end-to-end foundation: a `skill-pi` launch in a skillgrid repo (a) reads global config from `~/.skill-pi/` + project config through the cwd shim to `<project-root>/.skill-pi/`, leaving `~/.pi/` and `<project-root>/.pi/` byte-identical (R1, R2); (b) re-materializes the shim from scratch (R3); (c) materializes a real gitignored `<project>/.skill-pi/` (R4); (d) exposes the five extension ports' tools (R5); (e) gates `edit` calls via the permission ext (R6); (f) exposes the `ticket` tool (R7); (g) renders the context commands (R8); (h) renders the TUI at the breakpoint (R9); (i) exports `/memory-sync` JSONL (R10); (j) lists synced skills (R11); (k) honors the version pin (R12); (l) installs via the Go installer (R13); (m) forwards pi flags + owns subcommands (R14); (n) emits `--where` (R15); (o) keeps `SYSTEM.md` a delta (R16); (p) overrides the base URL (R17); (q) spawns with the skip-version + no-skills args (R18); (r) exposes the `mcp__*` tools (R19); (s) persists trust (R20); (t) injects the date/time (R21); (u) re-spawns on exit 75 (R22).
- Seam: the e2e test — it is the proof that the foundation works end-to-end, the build-shape's "proven end-to-end before the swarm slice is layered on" gate.
- Deletion test: removing the e2e leaves the foundation's integration unproven — the swarm slice (Tracer B) would be layered on an unverified foundation.
- Adapters: 1 (the real pi launch in a temp repo); the test drives the full surface.

**SATISFIES:** `foundation-e2e`

- [ ] **Step 1: Write the failing e2e test**
  A temp skillgrid repo + a temp `~/.skill-pi/`. Drive a `skill-pi` launch (the real launcher, the real shim, the real extensions) and assert the full R1–R22 surface: the config resolution (R1/R2), the shim rebuild (R3), the real gitignored `.skill-pi/` (R4), the five ports' tools registered (R5), the permission gate (R6), the `ticket` tool (R7), the context commands (R8), the TUI breakpoint (R9), the `/memory-sync` JSONL (R10), the synced skills (R11), the version pin (R12), the installer (R13), the flag forwarding + subcommands (R14), the `--where` JSON (R15), the `SYSTEM.md` delta (R16), the base-URL override (R17), the spawn args (R18), the `mcp__*` tools (R19), the trust (R20), the chronos injection (R21), the exit-75 loop (R22).
- [ ] **Step 2: Run to verify it fails**
  Run: `node --test skill-pi/e2e.test.js`. Expected: FAIL — the loop is not wired (the foundation modules exist from T1–T9; this is the integration).
- [ ] **Step 3: Wire the foundation**
  Connect the launcher + the shim + the extensions + the TUI + the installer so the full R1–R22 surface runs (the modules from Tasks 1–9 already exist; this is the integration).
- [ ] **Step 4: Run to verify it passes**
  Run: `node --test skill-pi/e2e.test.js`. Expected: PASS.
- [ ] **Step 5: Commit**
  ```bash
  git add skill-pi/e2e.test.js skill-pi/
  git commit -m "feat(skill-pi): foundation e2e tracer (R1-R22 proof)"
  ```

---

## Tracer B — the swarm slice (the team extension, R1–R8 of the swarm)

The swarm slice layers on the proven foundation: the main pi lead runs a goal-loop swarm over the existing mnemonic team plane, spawning Pi SDK subagent sessions as worker/reviewer, enforcing the two-review done gate, with opt-in per-task worktree isolation and a `goals` container so the dashboard shows goal progress. The tasks below build it: the plane schema + spawn-by-goal (T11–T13), the pi squad extension + role discovery + worktree + spawner + checkpoint/gate (T14–T18), the dashboard (T19), and the swarm e2e tracer (T20).

### Task 11: `goals` table + `tasks.goal_id` migration (ADR-0039)

> ⚠ **one-way:** ADR-0039 — adding a `goals` table + nullable `tasks.goal_id` FK is a schema migration. User-approved (chose the goals-table option). The FK is nullable/additive (no data loss); the table + index shape is the hard-to-reshape part. This migration also carries the `tasks.worktree_branch` column (ADR-0040) so the schema is complete in one additive pass.

**Files:**
- Create: `mnemonic/internal/store/migrations/056_team_goals.sql` (numbered **above the newest existing migration** — `055_context_revisions.sql` — so the filename-ordered runner applies it last)
- Create: `mnemonic/internal/integration/step07_team_goals_test.go`

**Interfaces:**
- Consumes: the existing `tasks` table (migration 009 block) and the migration runner.
- Produces: a `goals` table (`id TEXT PK, team_id TEXT NOT NULL, title TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'open', created_at TEXT NOT NULL`), a `tasks.goal_id TEXT NULL REFERENCES goals(id)` column, a `tasks.worktree_branch TEXT NULL` column, an index `idx_tasks_goal ON tasks(goal_id)`, and an index `idx_tasks_worktree_branch ON tasks(worktree_branch)`.
- Seam: the migration file (schema) — behavior is asserted by the integration test, not the SQL text.
- Deletion test: removing the migration re-creates the pre-goal schema; `team_spawn_task` loses its `goal_id` param and the dashboard loses the goal view.
- Adapters: 1 (the real SQLite store). The test is the only consumer at this stage.

**SATISFIES:** `goal-container-migration`

- [ ] **Step 1: Confirm the migration runner applies separate files by filename**
  The runner (`mnemonic/internal/store/store.go`) does `//go:embed migrations/*.sql` and applies **separate files in filename order**, tracked in `index_meta`. `001_schema.sql` is one such file (it contains numbered blocks, but it is a *file*); later migrations are *separate files* (`011_*` … `055_*`). **This migration is a new separate file, `056_team_goals.sql`** — it is NOT appended to `001_schema.sql`. Confirm the newest existing filename at exec time and number this file one higher (the critic verified `055_context_revisions.sql` is the newest as of writing; if a newer one has landed, bump the number to stay above it). The filename number is what makes the ordering correct — a file named `010_*` would sort *before* `040`–`055` and `ALTER TABLE tasks` against a schema that lacks the columns those later migrations add.
- [ ] **Step 2: Write the failing integration test**
  In `step07_team_goals_test.go`: open the store, assert the `goals` table exists, insert a goal, insert a task with `goal_id` set, assert the task's `goal_id` column reads back the goal id, and assert a task with no goal has `goal_id IS NULL`.
- [ ] **Step 3: Run the test to verify it fails**
  Run: `go test ./internal/integration/ -run TestTeamGoals -v` (adjust the test name to match). Expected: FAIL — no such table/column.
- [ ] **Step 4: Write the migration DDL**
  Create `056_team_goals.sql`: the `goals` table + `ALTER TABLE tasks ADD COLUMN goal_id TEXT` + `ALTER TABLE tasks ADD COLUMN worktree_branch TEXT` + `REFERENCES goals(id)` on `goal_id` + both indexes. Match the existing `001_schema.sql` style (IF NOT EXISTS, same column conventions, `ON DELETE CASCADE` on `team_id`). The `worktree_branch` column is ADR-0040's storage for the branch name (the `tasks` table has no `summary` column — only `task_results.summary`, one row per `SubmitOutput`, which is not surfaced in `ReadTask`/`ListTasksByGoal`; a dedicated nullable column is the clean home for it, and it lives in this same additive migration).
- [ ] **Step 5: Run the test to verify it passes**
  Run: `go test ./internal/integration/ -run TestTeamGoals -v`. Expected: PASS.
- [ ] **Step 6: Commit**
  ```bash
  git add mnemonic/internal/store/migrations/ mnemonic/internal/integration/step07_team_goals_test.go
  git commit -m "feat(teams): add goals table and tasks.goal_id (ADR-0039)"
  ```

### Task 12: `SpawnTask` + `team_spawn_task` accept `goal_id`; add `GET /teams/tasks/{id}/reviews` (ADR-0039, defect A)

**Files:**
- Modify: `mnemonic/internal/service/teams.go` (`SpawnTask` — add `GoalID` to the param struct; the `INSERT INTO tasks` includes `goal_id`; add a `ListReviews(ctx, taskID)` read)
- Modify: `mnemonic/internal/mcp/tools_teams.go` (`team_spawn_task` — add `goal_id` optional string param; pass through)
- Modify: `mnemonic/internal/http/teams.go` (`POST /teams/tasks` — accept `goal_id` in the body; pass through; **add `GET /teams/tasks/{id}/reviews`** — the read route the two-review gate needs, defect A)
- Create: `mnemonic/internal/integration/step07_team_goals_test.go` (extend — spawn-by-goal + reviews-read assertions)

**Interfaces:**
- Consumes: the `goals` table from Task 11; the existing `SpawnTask` param struct; the existing `reviews` store (the plane already persists `spec_compliance`/`code_quality` reviews on `SubmitReview`).
- Produces: `SpawnTask(ctx, param)` where `param.GoalID` is `*string` (nil = ungrouped); `ListReviews(ctx, taskID) ([]Review, error)` where `Review = {ReviewType, Passed, Comments}`; the HTTP/MCP surface exposes `goal_id` as optional **and** `GET /teams/tasks/{id}/reviews` returns the task's reviews.
- Seam: `SpawnTask` (the service method) — the MCP and HTTP handlers are two adapters over it.
- Deletion test: removing the `GoalID` field makes a spawn-by-goal drop the id → the task is ungrouped → the dashboard shows it under "(no goal)".
- Adapters: 2 (MCP handler + HTTP handler over the same service method).

**SATISFIES:** `spawn-task-by-goal`

- [ ] **Step 1: Write the failing test**
  Extend `step07_team_goals_test.go`: (a) create a goal, call `SpawnTask` with `GoalID` set, assert the returned task's `goal_id` equals the goal id; call `SpawnTask` with `GoalID` nil, assert `goal_id IS NULL`; (b) submit a `spec_compliance` review and a `code_quality` review for a task, then call `ListReviews(taskID)` and assert both come back with the right `ReviewType`/`Passed` (the gate's read path, defect A).
- [ ] **Step 2: Run to verify it fails**
  Run: `go test ./internal/integration/ -run 'TestSpawnByGoal|TestListReviews' -v`. Expected: FAIL — `GoalID` not yet a field; `ListReviews` not yet defined.
- [ ] **Step 3: Add `GoalID` to `SpawnTask` + `ListReviews` + the reviews read route**
  Add `GoalID *string` to the `SpawnTask` param struct; in the `INSERT INTO tasks`, include `goal_id` and bind `*param.GoalID` (nil → NULL). Add the optional `goal_id` string param to `tools_teams.go`'s `team_spawn_task` and the HTTP handler's body parse, passing a `*string` (empty/absent → nil). Add `ListReviews(ctx, taskID) ([]Review, error)` to the service (reads the existing reviews store) and a `GET /teams/tasks/{id}/reviews` HTTP route over it.
- [ ] **Step 4: Run to verify it passes**
  Run: `go test ./internal/integration/ -run 'TestSpawnByGoal|TestListReviews' -v`. Expected: PASS.
- [ ] **Step 5: Commit**
  ```bash
  git add mnemonic/internal/service/teams.go mnemonic/internal/mcp/tools_teams.go mnemonic/internal/http/teams.go mnemonic/internal/integration/step07_team_goals_test.go
  git commit -m "feat(teams): team_spawn_task accepts goal_id; add reviews read route (ADR-0039)"
  ```

### Task 13: `ListTasksByGoal` with per-status counts (ADR-0039, dashboard query contract)

**Files:**
- Create: `mnemonic/internal/service/goals.go` (new — `ListGoals`, `ListTasksByGoal`)
- Modify: `mnemonic/internal/http/teams.go` (add `GET /teams/goals` and `GET /teams/goals/{id}/tasks`)
- Create: `mnemonic/internal/integration/step07_team_goals_test.go` (extend — the query contract)

**Interfaces:**
- Consumes: `goals` + `tasks` tables (Tasks 11–12).
- Produces: `ListGoals(ctx, teamID) ([]GoalWithCounts, error)` where `GoalWithCounts = {ID, Title, Status, Counts map[status]int}`; `ListTasksByGoal(ctx, goalID) ([]TaskRow, error)` where `TaskRow = {ID, Title, Status, WorktreeBranch}` (`worktree_branch` is its own nullable column added in Task 1 — not parsed from a summary). The HTTP `GET /teams/goals/{id}/tasks` returns `{goal, tasks:[{id,title,status,worktree_branch}]}` directly from that column.
- Seam: `ListTasksByGoal` (service) — the HTTP route is the adapter.
- Deletion test: removing it leaves the dashboard with no goal→tasks query; the per-status counts are the whole progress view.
- Adapters: 1 (the HTTP route) at this stage; the UI (Task 19) is the second consumer.

**SATISFIES:** `goal-progress-query`

- [ ] **Step 1: Write the failing test**
  Seed a goal with tasks in 3 statuses + 1 task with `goal_id IS NULL`. Assert `ListGoals` returns the goal with correct per-status counts; assert `ListTasksByGoal` returns the goal's tasks (not the NULL one).
- [ ] **Step 2: Run to verify it fails**
  Run: `go test ./internal/integration/ -run TestGoalProgressQuery -v`. Expected: FAIL — functions not defined.
- [ ] **Step 3: Implement the query**
  `ListGoals`: `SELECT g.id, g.title, g.status, COUNT(t.status) ... FROM goals g LEFT JOIN tasks t ON t.goal_id = g.id GROUP BY g.id, t.status` (per-status counts via a pivot or two queries). `ListTasksByGoal`: `SELECT id, title, status, worktree_branch FROM tasks WHERE goal_id = ? ORDER BY priority DESC, created_at ASC`. Add the two HTTP routes.
- [ ] **Step 4: Run to verify it passes**
  Run: `go test ./internal/integration/ -run TestGoalProgressQuery -v`. Expected: PASS.
- [ ] **Step 5: Commit**
  ```bash
  git add mnemonic/internal/service/goals.go mnemonic/internal/http/teams.go mnemonic/internal/integration/step07_team_goals_test.go
  git commit -m "feat(teams): goal progress query with per-status counts (ADR-0039)"
  ```

### Task 14: pi extension scaffold — 6 team-plane HTTP tools (port of `skillgrid-squad.ts`) (swarm R1)

**Files:**
- Create: `skill-pi/extensions/skillgrid-squad.ts`
- Create: `skill-pi/extensions/skillgrid-squad.test.ts`
- Create: `agents/pi/roles/worker.md`, `agents/pi/roles/reviewer.md` (placeholders only — full prompts are Task 7; the scaffold needs the files to exist so `roles.ts` has something to glob)
- Modify: `skill-pi/package.json` (confirm the `piConfig: { name: "skill-pi", configDir: ".skill-pi" }` identity is present — per findings.md the launcher shim declares it; if `skill-pi/package.json` does not exist yet, create it with the identity block)

**Interfaces:**
- Consumes: the team-plane HTTP endpoints (`POST /teams/tasks`, `POST /teams/tasks/pull`, `GET /teams/tasks/{id}`, `POST /teams/tasks/{id}/output`, `POST /teams/tasks/{id}/reviews`, `GET /teams/tasks/{id}/reviews`, `POST /teams/tasks/{id}/done`) — the same `BASE_URL`/`AGENT` env resolution as `plugins/opencode/skillgrid-squad.ts`. Note: the plane currently has a **POST-only** reviews route (`POST /teams/tasks/{id}/reviews`); `GET /teams/tasks/{id}/reviews` is a **new read route** added in Task 12 (defect A — the two-review gate has no read path today, and `squad_read_task` returns only `{task, brief}`).
- Produces: pi `tool(...)` registrations: `squad_spawn_task`, `squad_pull_next_task`, `squad_read_task`, `squad_submit_output`, `squad_submit_review`, **`squad_reviews`** (new read), `squad_mark_done` — the 6-tool surface is **byte-compatible** with the opencode plugin (same names, same params, same failure-latch behavior), plus the one new `squad_reviews` read the gate needs.
- Seam: the HTTP client wrapper (one function per endpoint) — the pi `tool(...)` registrations are adapters over it.
- Deletion test: removing the extension removes the pi harness's access to the team plane; the opencode plugin still works (independent transport).
- Adapters: 1 (the pi `tool` shape) over the 6 HTTP calls; the opencode plugin is a parallel reference, not a shared adapter.

**SATISFIES:** `squad-http-adapter`

- [ ] **Step 1: Write the failing test**
  In `skillgrid-squad.test.ts`: stub the HTTP layer (or use a local mock server), and assert each of the **7** tools posts/gets the right endpoint with the right body (the 6 ported + `squad_reviews` → `GET /teams/tasks/{id}/reviews`), and that a mutation failure sets the in-memory failure latch (port the `SKILLGRID_SQUAD_FAILURE` handoff from `skillgrid-squad.ts` lines 9–90).
- [ ] **Step 2: Run to verify it fails**
  Run: `node --test skill-pi/extensions/skillgrid-squad.test.ts` (or the pi extension's test runner — confirm at exec time; if jiti-loaded, the test may need a ts loader). Expected: FAIL — extension not written.
- [ ] **Step 3: Port the extension + add the reviews read**
  Copy `plugins/opencode/skillgrid-squad.ts`'s 6-tool logic into `skill-pi/extensions/skillgrid-squad.ts`, replacing opencode's `tool(...)` API with pi's `ExtensionAPI` `tool(...)` shape (confirm the exact pi registration signature from the existing pi extension pattern in findings.md; if no pi extension exists yet to reference, the shape is the standard pi `ExtensionAPI.tool(name, {description, parameters}, handler)`). Keep `BASE_URL`/`AGENT` env resolution identical. Add the `goal_id` pass-through to `squad_spawn_task` (optional, nil-when-absent) so the R6 path works. **Add a `squad_reviews` read tool** that calls the new `GET /teams/tasks/{id}/reviews` route (the route is added to the Go plane in Task 12; the extension tool and the plane route are two halves of the gate's read path, defect A).
- [ ] **Step 4: Run to verify it passes**
  Run: `node --test skill-pi/extensions/skillgrid-squad.test.ts`. Expected: PASS.
- [ ] **Step 5: Commit**
  ```bash
  git add skill-pi/extensions/skillgrid-squad.ts skill-pi/extensions/skillgrid-squad.test.ts agents/pi/roles/worker.md agents/pi/roles/reviewer.md skill-pi/package.json
  git commit -m "feat(skill-pi): scaffold squad extension (6 team-plane HTTP tools)"
  ```

### Task 15: Role discovery — `roles.ts` (swarm R3, ADR-0041)

**Files:**
- Create: `skill-pi/extensions/roles.ts`
- Create: `skill-pi/extensions/skillgrid-squad.test.ts` (extend — role discovery cases)
- Modify: `agents/pi/roles/worker.md`, `agents/pi/roles/reviewer.md` (give them real minimal prompts + the existing frontmatter convention + a `model:` frontmatter example)

**Interfaces:**
- Consumes: the filesystem (`agents/pi/roles/*.md`).
- Produces: `discoverRoles(rolesDir: string): Role[]` where `Role = { name: string, prompt: string, model?: string }`; `name` is the filename minus `.md`; `model` is parsed from `model:` frontmatter (absent → undefined). Expose a `swarm_role` tool (list discovered roles) so the lead can see them.
- Seam: `discoverRoles` — the spawner (Task 6) and the `swarm_role` tool are consumers.
- Deletion test: removing it makes roles hardcoded in the spawner → adding a role requires a code change (violates R3).
- Adapters: 2 (the `swarm_role` tool + the spawner's role lookup).

**SATISFIES:** `role-discovery`

- [ ] **Step 1: Write the failing test**
  In a temp roles dir, write `a.md`, `b.md` (with `model: x`), and a non-`.md` file. Assert `discoverRoles` returns `{a},{b}` (not the non-`.md`), and `b.model === "x"`. Then write `c.md` and re-call — assert `c` appears (the "new file = new role" property).
- [ ] **Step 2: Run to verify it fails**
  Run: `node --test skill-pi/extensions/skillgrid-squad.test.ts -t role`. Expected: FAIL.
- [ ] **Step 3: Implement `discoverRoles`**
  Glob `<rolesDir>/*.md`; for each, read the file, parse `model:` from a leading `---` frontmatter block (a simple line scan, no YAML dep — ADR-0041: no new deps), strip the frontmatter from the prompt. Register the `swarm_role` tool returning the discovered names.
- [ ] **Step 4: Run to verify it passes**
  Run: `node --test skill-pi/extensions/skillgrid-squad.test.ts -t role`. Expected: PASS.
- [ ] **Step 5: Commit**
  ```bash
  git add skill-pi/extensions/roles.ts skill-pi/extensions/skillgrid-squad.test.ts agents/pi/roles/worker.md agents/pi/roles/reviewer.md
  git commit -m "feat(skill-pi): file-driven role discovery (ADR-0041)"
  ```

### Task 16: Worktree isolation — `worktree.ts` + branch naming (swarm R5, ADR-0040)

**Files:**
- Create: `skill-pi/extensions/worktree.ts`
- Create: `skill-pi/extensions/skillgrid-squad.test.ts` (extend — worktree cases)
- Modify: `skill-pi/extensions/skillgrid-squad.ts` (call `worktree.ts` on a `worktree: true` spawn; record the branch in the task's `worktree_branch` column)

**Interfaces:**
- Consumes: the `git` CLI; the `.worktrees/` convention; `precommit-guard.js`'s branch regex `^(agent-|worktree-agent-|worktree-wf_)[A-Za-z0-9._/-]+$`.
- Produces: `createWorktree(repoRoot, taskID): Promise<{path, branch}>` (branch = `worktree-agent-swarm-<taskID>` — matches the guard's `worktree-agent-*` alternative); `commitWorktree(path, branch, taskID): Promise<void>` (stages the worktree, commits to the branch, and records the branch on the task via a new `swarm_set_branch { task_id, branch }` tool that updates the plane's `tasks.worktree_branch` column — Task 2's `ListTasksByGoal`/dashboard reads it back); `removeWorktree(path, branch): Promise<void>` (best-effort cleanup). A `task_id` not matching `[A-Za-z0-9-]+` is rejected before any `git` call.
- Seam: the three exported functions — the spawner (Task 7) and the extension's spawn path call them.
- Deletion test: removing it makes a `worktree: true` task edit the serial working tree (no isolation) and leaves no branch.
- Adapters: 1 (the real git CLI); the test stubs git for the failure-path cases.

**SATISFIES:** `worktree-isolation`

- [ ] **Step 1: Write the failing test**
  In a temp git repo: `createWorktree` produces a path under `.worktrees/` and a branch matching the guard regex; a `task_id` with a space is rejected (assert no `git` exec for the bad id); `commitWorktree` on a repo with changes produces a commit on the branch; a `git` failure (stub a non-zero exit) leaves the worktree path on disk and returns stderr.
- [ ] **Step 2: Run to verify it fails**
  Run: `node --test skill-pi/extensions/skillgrid-squad.test.ts -t worktree`. Expected: FAIL.
- [ ] **Step 3: Implement `worktree.ts`**
  Use `execFile("git", [...args])` (no shell). `createWorktree`: validate `taskID` against `[A-Za-z0-9-]+`, then `git worktree add .worktrees/swarm-<taskID> -b worktree-agent-swarm-<taskID>`. `commitWorktree`: `git -C <path> add -A && git commit -m "swarm(<taskID>): <summary>"`. `removeWorktree`: `git worktree remove <path>` + `git branch -D <branch>` (best-effort, ignore errors). On any `git` error, return `{ok:false, stderr}` and do not delete.
- [ ] **Step 4: Run to verify it passes**
  Run: `node --test skill-pi/extensions/skillgrid-squad.test.ts -t worktree`. Expected: PASS.
- [ ] **Step 5: Commit**
  ```bash
  git add skill-pi/extensions/worktree.ts skill-pi/extensions/skillgrid-squad.test.ts skill-pi/extensions/skillgrid-squad.ts
  git commit -m "feat(skill-pi): opt-in per-task worktree isolation (ADR-0040)"
  ```

### Task 17: Subagent spawner — `spawn.ts` (swarm R2, R8) + role prompts

**Files:**
- Create: `skill-pi/extensions/spawn.ts`
- Create: `skill-pi/extensions/skillgrid-squad.test.ts` (extend — spawn cases)
- Modify: `agents/pi/roles/worker.md`, `agents/pi/roles/reviewer.md` (full role prompts: worker = read brief, do the work, submit output; reviewer = read output, emit spec_compliance OR code_quality verdict + passed bool)
- Modify: `skill-pi/extensions/skillgrid-squad.ts` (register `swarm_spawn { role, task_id, worktree? }` — the lead-facing tool that ties `roles.ts` + `worktree.ts` + the pi subagent SDK together; enforce serial: a second `swarm_spawn` while one is active is rejected)

**Interfaces:**
- Consumes: `discoverRoles` (Task 15), `createWorktree`/`commitWorktree` (Task 16), the pi subagent SDK (the exact spawn call signature — confirm from the pi SDK docs at exec time; the pattern is "spawn a session with a system prompt + scoped memory owner + cwd"), the team-plane `squad_read_task`/`squad_submit_output`/`squad_submit_review`.
- Produces: `swarmSpawn({ role, taskID, worktree? }): Promise<{taskID, status}>` — resolves the role (error if unknown), creates a worktree if `worktree` (else cwd = repo root), spawns the subagent session with the role prompt + `owner=taskID` + cwd, and (for a worker) the subagent's job is to `squad_read_task` → do the work → `squad_submit_output`; (for a reviewer) `squad_read_task` → emit `squad_submit_review` with the requested `review_type`. Serial enforcement: a module-level `activeTaskID` guard — a second `swarmSpawn` while `activeTaskID` is set throws "a worker is already active" (R8).
- Seam: `swarmSpawn` — the lead calls it; `roles.ts`/`worktree.ts`/the subagent SDK are its dependencies.
- Deletion test: removing it makes the lead unable to spawn a member; the roles and worktree modules are orphaned.
- Adapters: 2+ (the worker role + the reviewer role are two consumers of the same `swarmSpawn` with different prompts; the subagent SDK is the spawn mechanism).

**SATISFIES:** `swarm-spawn-subagent`, `serial-by-default`

- [ ] **Step 1: Write the failing test**
  Stub the pi subagent SDK (a fake that records the session config it was given). Call `swarmSpawn({role:"worker", taskID:"t1"})` — assert the fake received `owner="t1"`, `cwd=<repoRoot>`, and the worker role prompt. Call with `worktree:true` — assert `cwd` is the worktree path. Call a second `swarmSpawn` while the first is "active" — assert it throws the serial guard. Call with `role:"nope"` — assert an unknown-role error.
- [ ] **Step 2: Run to verify it fails**
  Run: `node --test skill-pi/extensions/skillgrid-squad.test.ts -t spawn`. Expected: FAIL.
- [ ] **Step 3: Implement `spawn.ts` + full role prompts**
  `swarmSpawn` resolves the role via `discoverRoles` (unknown → throw), sets `cwd` (worktree if requested, via Task 6, else repo root), and calls the pi subagent SDK to spawn a session with the role prompt + `owner=taskID` + `cwd`. Set/clear `activeTaskID` around the spawn for the serial guard. Write the two role `.md` prompts (worker: "read the brief via squad_read_task, implement, submit via squad_submit_output; if worktree, your cwd is the worktree — commit is handled by the extension on done"; reviewer: "read the task via squad_read_task, review, submit via squad_submit_review with review_type=<spec_compliance|code_quality> and a passed bool + comments").
- [ ] **Step 4: Run to verify it passes**
  Run: `node --test skill-pi/extensions/skillgrid-squad.test.ts -t spawn`. Expected: PASS.
- [ ] **Step 5: Commit**
  ```bash
  git add skill-pi/extensions/spawn.ts skill-pi/extensions/skillgrid-squad.test.ts skill-pi/extensions/skillgrid-squad.ts agents/pi/roles/worker.md agents/pi/roles/reviewer.md
  git commit -m "feat(skill-pi): subagent spawner + worker/reviewer roles"
  ```

### Task 18: Checkpoint — `goal.ts` + the two-review gate in the extension (swarm R7, R4)

**Files:**
- Create: `skill-pi/extensions/goal.ts`
- Create: `skill-pi/extensions/skillgrid-squad.test.ts` (extend — checkpoint + gate cases)
- Modify: `skill-pi/extensions/skillgrid-squad.ts` (register `goal_mark { task_id, status }` and `goal_resume`; encode the two-review gate so a task is only `mark_done`'d after both reviews pass)

**Interfaces:**
- Consumes: the filesystem (`.skill-pi/goal/progress.json`); the team-plane `squad_reviews` (Task 14's read tool, defect A) and `squad_mark_done` (for the gate).
- Produces: `goalMark(taskID, status): Promise<void>` (atomic write: temp + rename; shape `{current, status, updated}` — **only the orientation fields**; the task list is NOT stored in the checkpoint, it is always re-read from `GET /teams/goals/{id}/tasks` at resume so the checkpoint can never drift into a second goal record, defect E); `goalResume(): Promise<{current, status, updated} | null>` (read + validate; corrupt/absent → `null`). The two-review gate: a helper `bothReviewsPassed(taskID): Promise<boolean>` that calls `squad_reviews(taskID)` (the plane's `GET /teams/tasks/{id}/reviews`) and returns true only if a `spec_compliance` AND a `code_quality` review with `passed=true` exist. `squad_mark_done` is gated: it calls `bothReviewsPassed` first and refuses (with a structured error) if either is missing/failed. **The gate owns commit-on-done (critic's B):** `squad_mark_done`, on a passing gate, is what triggers `worktree.ts`'s `commitWorktree` for a `worktree: true` task — so a worktree is committed *only* after both reviews pass, never before, and a failed-review retry (re-`swarmSpawn` the worker) leaves the worktree uncommitted until the next passing gate. This makes the commit a *code* guarantee (gated in the extension), not a conversational one in the lead's prompt.
- Seam: `goalMark`/`goalResume` (checkpoint) + `bothReviewsPassed` (gate, which owns commit-on-done) — the lead and `squad_mark_done` call them.
- Deletion test: removing `goal.ts` makes a compaction lose the lead's place; removing the gate lets a single review mark a task done (violates R4).
- Adapters: 2 (the `goal_mark`/`goal_resume` tools + the `squad_mark_done` gate hook).

**SATISFIES:** `checkpoint-roundtrip`, `two-review-gate`

- [ ] **Step 1: Write the failing test**
  Checkpoint: `goalMark` then `goalResume` round-trips `{current, status}`; a corrupt file (write truncated JSON) → `goalResume` returns `null` (not a throw). Gate: seed a task with only a passing `spec_compliance` review → `bothReviewsPassed` is false and `squad_mark_done` refuses; add a passing `code_quality` → `bothReviewsPassed` true and `squad_mark_done` proceeds; a `code_quality` with `passed=false` → still false.
- [ ] **Step 2: Run to verify it fails**
  Run: `node --test skill-pi/extensions/skillgrid-squad.test.ts -t "checkpoint|gate"`. Expected: FAIL.
- [ ] **Step 3: Implement**
  `goal.ts`: atomic write (write `<path>.tmp` + `rename`), read + `JSON.parse` in a try/catch → `null` on any failure; store only `{current, status, updated}`. `bothReviewsPassed`: call `squad_reviews(taskID)` (Task 14's read tool over `GET /teams/tasks/{id}/reviews`) and return true only when both a passing `spec_compliance` and a passing `code_quality` review exist. Gate `squad_mark_done` on it.
- [ ] **Step 4: Run to verify it passes**
  Run: `node --test skill-pi/extensions/skillgrid-squad.test.ts -t "checkpoint|gate"`. Expected: PASS.
- [ ] **Step 5: Commit**
  ```bash
  git add skill-pi/extensions/goal.ts skill-pi/extensions/skillgrid-squad.test.ts skill-pi/extensions/skillgrid-squad.ts
  git commit -m "feat(skill-pi): goal checkpoint + two-review done gate"
  ```

### Task 19: Dashboard goal-progress view (swarm R6)

> Locate the existing dashboard surface at exec time: grep `skillgrid-ui/` (and `skillgrid-cli/` if the dashboard is CLI-rendered) for the current team/tasks view. The query contract is pinned by Task 3 (`GET /teams/goals/{id}/tasks` → `{goal, tasks:[{id,title,status,worktree_branch}]}`, and `GET /teams/goals` → per-status counts). This task renders it.

**Files:**
- Modify: `<located dashboard route/component>` (add a goal list + a goal detail view: goal title, per-status counts, task list with status + `worktree_branch`)
- Create/Modify: the dashboard's test for the goal view (assert a goal with 3-status tasks renders 3 counts; a `goal_id IS NULL` task renders under "(no goal)")

**Interfaces:**
- Consumes: `GET /teams/goals` and `GET /teams/goals/{id}/tasks` (Task 13).
- Produces: a rendered goal-progress view.
- Seam: the dashboard route — the plane query is the data seam.
- Deletion test: removing it leaves the goal data in the plane but invisible in the UI (the "value is invisible" problem the briefing names).
- Adapters: 1 (the UI) over the two plane endpoints.

**SATISFIES:** `dashboard-goal-progress`

- [ ] **Step 1: Locate + write the failing test**
  Find the dashboard's team/tasks view and its test harness. Add a test: a goal with tasks in pending/in-progress/review_spec renders the 3 counts; a goal-less task renders under "(no goal)".
- [ ] **Step 2: Run to verify it fails**
  Run the dashboard's test command (confirm at exec time). Expected: FAIL — no goal view.
- [ ] **Step 3: Implement the view**
  Add a goal list (from `GET /teams/goals`) and a goal detail (from `GET /teams/goals/{id}/tasks`) showing per-status counts + the task list with `worktree_branch`. Route a `goal_id IS NULL` task to a "(no goal)" bucket.
- [ ] **Step 4: Run to verify it passes**
  Run the dashboard's test command. Expected: PASS.
- [ ] **Step 5: Commit**
  ```bash
  git add <dashboard files>
  git commit -m "feat(dashboard): goal progress view (ADR-0039)"
  ```

### Task 20: End-to-end tracer (the swarm build-shape proof) — wire the full loop

**Files:**
- Modify: `skill-pi/extensions/skillgrid-squad.test.ts` (add an e2e integration test against a real temp store + temp git repo: the lead decomposes a goal → `squad_spawn_task` with `goal_id` → `swarmSpawn` worker (worktree) → `squad_submit_output` → reviewer spec → reviewer code → `squad_mark_done` → assert the goal shows 1 done in the plane + the branch exists)
- Modify: the lead's goal-loop prompt (where the lead's system prompt lives — `skill-pi/system/APPEND_SYSTEM.md` or a `prompts/` command per findings.md) to encode the loop: decompose → for each task: `goal_mark` → `swarmSpawn` worker → `swarmSpawn` reviewer(spec) → on pass `swarmSpawn` reviewer(code) → on both pass `squad_mark_done` + `goal_mark done` → advance; on a failed review, re-`swarmSpawn` the worker.

**Interfaces:**
- Consumes: every module from Tasks 11–19.
- Produces: a working end-to-end loop + the lead's goal-loop prompt (the conversational driver, R7).
- Seam: the prompt (the lead's behavior) — it is what the verifier checks: does the loop actually close?
- Deletion test: removing the prompt leaves the modules unconnected — the lead does not know the sequence.
- Adapters: the e2e test is the proof adapter.

**SATISFIES:** `e2e-goal-loop`

- [ ] **Step 1: Write the failing e2e test**
  Temp store + temp git repo. Drive the sequence exactly as the lead would (spawn by goal, worker in worktree, both reviews, mark done). Assert: the task's `goal_id` is set, the goal's per-status count shows `done:1`, the branch `worktree-agent-swarm-<taskID>` exists with a commit, and the checkpoint `progress.json` reflects `done`.
- [ ] **Step 2: Run to verify it fails**
  Run: `node --test skill-pi/extensions/skillgrid-squad.test.ts -t e2e`. Expected: FAIL — the loop is not wired.
- [ ] **Step 3: Wire the loop + the lead prompt**
  Connect the extension's tools so the sequence runs (the modules from Tasks 11–19 already exist; this is the integration). Write the lead's goal-loop prompt encoding R7's sequence + the checkpoint discipline.
- [ ] **Step 4: Run to verify it passes**
  Run: `node --test skill-pi/extensions/skillgrid-squad.test.ts -t e2e`. Expected: PASS.
- [ ] **Step 5: Commit**
  ```bash
  git add skill-pi/extensions/skillgrid-squad.test.ts skill-pi/system/ skill-pi/prompts/
  git commit -m "feat(skill-pi): end-to-end goal-loop swarm (tracer complete)"
   ```
 
 ---
 
 ## Addendum: the auth extension — multi-provider OAuth + API-key (post-2026-10-09 user request)
 
 A user-requested extension that extends pi's provider list with (a) **multi-account** auth
 (multiple credentials for the same provider), (b) **Kilo Code** device-flow OAuth,
 (c) **opencode zen** device-flow OAuth, (d) **Ollama Cloud** API-key auth, and (e) a
 **local OpenAI-compatible** endpoint (Ollama / vLLM) with optional API key. Sources:
 `hjanuschka/pi-multi-pas`, `MasuRii/pi-multi-auth`, `fgrehm/pi-ollama-cloud`,
 `luongnv89/pi-extensions`. This is a new tracer-level capability layered on the proven
 foundation; it is additive (a new `extensions/auth.js`) and does not modify T1–T10.
 
 ### Design decisions (locked for this addendum)
 
 1. **Multi-account mechanism.** pi's native `auth.json` stores one credential per provider
    id. Multiple accounts for one provider = **register the provider under multiple ids**
    (`kilo`, `kilo-2`, `kilo-work`, …) — the exact pattern `pi-multi-auth` uses
    (`AuthWriter` writes a credential per provider id). The extension exposes a
    `/skillgrid-account <provider> [suffix]` command that (re)registers `provider[-suffix]`
    with the shared provider config, so the user can hold several logged-in identities
    side-by-side and switch with the normal `/model` flow. This is *registration*
    multiplicity, not a rotation/quota engine.
 
 2. **No rotation / quota / failover engine.** `pi-multi-pas`'s pool + subscription model is
    intentionally excluded. pi handles credential storage, refresh, and rotation natively once
    a provider is registered; this extension only supplies the provider definition and the
    OAuth login flow.
 
 3. **All four targets via `pi.registerProvider(name, config)`.** Confirmed against the
    installed `pi` 1.1.0 `ProviderConfig` (core/extensions/types.d.ts:1390): `{ name?,
    baseUrl?, apiKey?, api?, headers?, authHeader?, models?, refreshModels?(ctx),
    oauth?: { name, isSubscription?, login(cb), refreshToken(cred,signal), getApiKey(cred) } }`.
    Chat models (`ProviderChatModelConfig`) require `{ id, name, input, cost, reasoning,
    contextWindow, maxTokens }`.
 
 ### Endpoints (verified from source; placeholders marked)
 
 | Provider | id | wire (`api`) | baseUrl | auth | models |
 |---|---|---|---|---|---|
 | Kilo Code | `kilo` (+ `kilo-<suffix>`) | `openai-completions` | `https://api.kilo.ai` | **device-flow OAuth** | `refreshModels` → `GET /v1/models` |
 | opencode zen | `opencode-zen` (+ suffix) | `openai-completions` | `https://opencode.ai/zen/v1` *(placeholder — confirm against opencode source)* | **device-flow OAuth** | `refreshModels` → `GET /v1/models` |
 | Ollama Cloud | `ollama-cloud` | `openai-completions` | `https://ollama.com/v1` | API-key `apiKey: "$OLLAMA_API_KEY"`, `authHeader: true` | `refreshModels` → `GET /v1/models` + `POST /api/show` |
 | Local OpenAI (Ollama/vLLM) | `local-openai` | `openai-completions` | env `SKILL_PI_LOCAL_BASE_URL` (default `http://localhost:11434/v1`) | optional `apiKey` (env or none) | `refreshModels` → `GET /v1/models` |
 
 **Kilo device-flow (confirmed):** `POST https://api.kilo.ai/api/device-auth/codes`
 → `{ code, verificationUrl, expiresIn }`; poll `GET …/api/device-auth/codes/{code}`
 (202 pending / 403 denied / 410 expired); token reused as access+refresh, ~365-day expiry;
 `getApiKey: cred => cred.access`.
 
 **opencode zen (skeleton):** same device-code shape as Kilo; exact authorize/poll URLs are
 placeholders pending `anomalyco/opencode` source confirmation. `getApiKey` maps the issued
 token. `OPENCODE_API_KEY` env is the API-key fallback (registered as `opencode-zen-key` if
 the env is present and no OAuth login has run).
 
 ### Files
 
 - **`skill-pi/extensions/auth.js`** (new, CJS) — `factory(pi)` registers the four providers;
   exports pure internals (`makeKiloOAuth`, `makeZenOAuth`, `makeOllamaCloudConfig`,
   `makeLocalOpenAIConfig`, `registerAccounts`, `__test`) for tests. No `fetch` is performed
   at registration time; `refreshModels` closures fetch lazily when pi requests the catalog.
 - **`skill-pi/extensions/auth.test.js`** (new) — TDD red→green: each provider registers with
   the required `ProviderChatModelConfig` fields; OAuth `login` drives `onDeviceCode` +
   `onPrompt` and returns `{ access, refresh, expires }`; `getApiKey` maps access→key;
   `refreshModels` maps a mocked `/v1/models` body to `ProviderModelConfig[]`;
   `registerAccounts` registers `kilo`, `kilo-2` under distinct ids.
 - **`skill-pi/package.json`** — add `./extensions/auth.js` to `pi.extensions`.
 
 ### Verification
 
 - `node --test skill-pi/extensions/auth.test.js` green (red→green discipline).
 - Full suite `node --test` in `skill-pi/` green (no regression to splash/launcher tests).
 - Smoke: `node -e "require('./skill-pi/extensions/auth.js')"` loads without throwing;
   `__test` surface is importable.
 
 ### Acceptance (see `acceptance.feature` — "happy path auth-providers-register")
 
 A pi session under skill-pi exposes `kilo`, `opencode-zen`, `ollama-cloud`, and
 `local-openai` as selectable providers; `/skillgrid-account kilo work` registers a second
   `kilo-work` id; Ollama Cloud sends `Authorization: Bearer $OLLAMA_API_KEY`; the local
   endpoint honors `SKILL_PI_LOCAL_BASE_URL` and works keyless.
 
 ---
 
 ## Self-Review

1. **Spec coverage:** *Tracer A (R1–R22):* R1→T2, R2→T2, R3→T2, R4→T2/T3, R5→T4, R6→T5, R7→T5, R8→T5, R9→T6, R10→T7, R11→T7, R12→T1/T9, R13→T8, R14→T2, R15→T2/T8, R16→T3, R17→T3, R18→T2, R19→T5, R20→T2/T8, R21→T9, R22→T2. *Tracer B (swarm R1–R8):* R1→T14, R2→T17, R3→T15, R4→T18, R5→T16, R6→T11/T12/T13/T19, R7→T18/T20, R8→T17. All requirements have a task. ✓
2. **Must-haves coverage:** each truth maps to a task's test (foundation: module-skeleton→T1, launcher/shim/respawn/where/subcommands→T2, config/system-delta→T3, ports→T4, permission/tickets/context/bridge→T5, TUI→T6, memory-sync/skills→T7, installer/trust→T8, chronos/version-pair→T9, e2e→T10; swarm: migration→T11, spawn-by-goal→T12, query→T13, role→T15, worktree→T16, spawn/serial→T17, gate/checkpoint→T18, dashboard→T19, e2e→T20). ✓
3. **One-way-door completeness:** ADR-0042 (the `@earendil-works/pi-*` deps) is tagged `> ⚠ one-way:` on Task 1; ADR-0039 (the `goals` table + `tasks.goal_id` FK) is tagged `> ⚠ one-way:` on Task 11. ADR-0040/0041 are not one-way doors (no separate migration / no contract break) — confirmed. ✓
4. **Placeholder scan:** no "TBD"/"implement later"/"similar to Task N". Task 19 names the locate-at-exec-time step explicitly (the dashboard path is genuinely unknown until located — it is a *find*, not a placeholder; the query contract is pinned by T13). ✓
5. **Type consistency:** *Tracer A:* the launcher's 8-key `--where` JSON, the shim's `piConfig.configDir = ".skill-pi"`, the 3-tier permission merge, the `mcp__<server>__<tool>` registration, the JSONL union merge by observation id — used consistently across T1–T10. *Tracer B:* `Role{name,prompt,model?}`, `createWorktree→{path,branch}`, `goalMark(taskID,status)` / `goalResume→{current,status,updated}|null` (checkpoint stores only orientation, not the task list — defect E), `bothReviewsPassed(taskID)→bool` (reads via `squad_reviews` — defect A), `swarmSpawn({role,taskID,worktree?})→{taskID,status}` — used consistently across T15–T20. The plane side: `SpawnTask.GoalID *string`, `ListReviews(taskID)→[]Review`, `ListGoals→[]GoalWithCounts`, `ListTasksByGoal→[]TaskRow`, `TaskRow{ID,Title,Status,WorktreeBranch}` (own column, not a summary — defect #1). ✓

## Owed-Decision Gate (input coverage)

| Required value | Named source | Status |
|---|---|---|
| `goals` table shape | ADR-0039 (this blueprint, Task 1) | named |
| `tasks.goal_id` nullability | ADR-0039 (nullable FK) | named |
| Branch name format | ADR-0040 + Global Constraints (must match `precommit-guard.js` regex → `worktree-agent-swarm-<taskID>`) | named |
| Checkpoint file path + shape | R7 / Task 8 (`.skill-pi/goal/progress.json`) | named |
| Role file dir | R3 / ADR-0041 (`<repo-root>/agents/pi/roles/`) | named |
| Per-role model hint | ADR-0041 (optional `model:` frontmatter) | named |
| Review types | the plane's existing `spec_compliance`/`code_quality` (confirmed in `service/teams.go`) | named |
| Dashboard query contract | Task 3 (`GET /teams/goals[/{id}/tasks]`) | named |
| Dashboard *render* location | **locate at exec time** (Task 9) — the existing dashboard surface path is unknown until grepped; the query contract is pinned, so this is a find, not an owed decision | **find (not owed)** |
| pi subagent SDK spawn signature | pi SDK (confirm at exec time, Task 7) — the exact call shape is a pi-runtime API, not a value this build invents | **find (not owed)** |
| `squad_read_task` returns reviews? | **Resolved (defect A):** the critic verified `GET /teams/tasks/{id}` returns only `{task, brief}` and the plane has a POST-only reviews route. The gate's read path is a **new `GET /teams/tasks/{id}/reviews` route** added in Task 2 + a `squad_reviews` tool added in Task 4. No longer a find. | **named (Task 2 + Task 4)** |
| `tasks.worktree_branch` storage | **Resolved (defect #1):** the `tasks` table has no `summary` column (only `task_results.summary`, not surfaced in `ReadTask`). The branch lives in a **`tasks.worktree_branch TEXT NULL` column** added in Task 1's migration; `ListTasksByGoal` selects it directly. | **named (Task 1)** |

**Owed decisions: none.** Every load-bearing value has a named source (ADR, spec requirement, or an existing plane mechanism). The two remaining "find at exec time" items (dashboard render location, pi subagent SDK spawn signature) are lookups of existing facts — a path and an API signature — not invented values; they do not owe ratification. → **Proceed to the Plan Review gate.**

## Execution Handoff

The blueprint has 20 tasks in two tracers — slicing is warranted (a clear wave structure: *Tracer A* T1→T2→T3 is the module+launcher+config foundation, T4→T5→T6→T7→T8→T9 are the pi capabilities + installer + supply-chain, T10 is the foundation e2e proof; *Tracer B* T11→T12→T13 is the plane foundation, T14→T15→T16→T17→T18 are the pi swarm capabilities, T19 is the dashboard, T20 is the swarm e2e proof). Invoke `skillgrid:slicing` next to produce `tasks.md` with dependency edges and waves. Tracer A must complete (T10's e2e passes) before Tracer B starts (T11) — the swarm slice layers on the proven foundation.

**Two execution options:**
1. **Subagent-Driven (recommended)** — dispatch a fresh subagent per task/ticket, review between tasks, fast iteration.
2. **Inline Execution** — run tasks in this session via `skillgrid:simple-execution`, batch with checkpoints.

> **Escalation note:** this change touches a schema migration (ADR-0039, one-way door) + a new HTTP contract + subprocess (git). Per `_shared/planning/rigor-tiers.md`, the final review should be **escalated to `skillgrid:parallel-code-review`** (multi-reviewer fan-out), not the lightweight two-axis pass.
