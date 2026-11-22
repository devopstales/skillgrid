# Smoke Evidence — TICKET-01 (2026-11-17-skillgrid-host-plugins)

- **Ticket:** TICKET-01 (door check) — Tracker `t_06bb0c13`
- **Change:** `2026-11-17-skillgrid-host-plugins`
- **Worktree:** `/Users/paladm/git/ai-test/skillgrid-v2/.worktrees/t_06bb0c13` (branch `wt/t_06bb0c13`, base `d3026233`)

## TICKET-01 door check — bootstrap asset + cursor hook

Hard gate for the blueprint tracer thread (blueprint Task 1, Step 5). Hook-level
oracles below are the enforceable gate; the live cursor session clause is the
held-out backstop (recorded ABANDON, see § Live door check).

### Oracle 1 — cwd `/tmp`: no bootstrap content, exit 0

Command:

```bash
printf '{"cwd":"/tmp"}' | hooks/cursor-session-start.sh
```

Output:

```json
{"additional_context":"Mnemonic is active. Start with mem_session_start (title), save decisions/bugfixes via mem_save, and close with mem_session_summary + mem_session_end. Prefer mem_context / mem_search before re-deriving prior work. Call code_explore before rg."}
```

Exit: `0`. Bootstrap content present: **no**. Verdict: **PASS**.

### Oracle 2 — cwd = worktree root: bootstrap present in `additional_context`

Command:

```bash
printf '{"cwd":"/Users/paladm/git/ai-test/skillgrid-v2/.worktrees/t_06bb0c13"}' | hooks/cursor-session-start.sh
```

Output (excerpt — bootstrap block then mnemonic prime):

```json
{"additional_context":"# Skillgrid Router Bootstrap\n\nThis project is configured with Skillgrid.\n\nFirst action: read `.agents/skills/using-skillgrid/SKILL.md` (or the host's skill-load command).\n\nRequest classes: start / continue / status / review-only / bounded.\n\nAnnounce: `Using skillgrid:using-skillgrid to {purpose}`.\n\nLoad hints:\n- pi: `/skill:using-skillgrid` + `using-skillgrid/references/pi-tools.md`\n- opencode: `skill` / `@using-skillgrid` + `using-skillgrid/references/opencode-tools.md`\n- kilo: read the path\n- cursor: `/using-skillgrid` or read\n- omp: `/skill:using-skillgrid`\n\nMnemonic is active. Start with mem_session_start (title), save decisions/bugfixes via mem_save, and close with mem_session_summary + mem_session_end. Prefer mem_context / mem_search before re-deriving prior work. Call code_explore before rg."}
```

Exit: `0`. Bootstrap content present: **yes** (front of `additional_context`). Verdict: **PASS**.

### Oracle 3 — missing asset: static fallback present, exit 0 (fail-open)

Command:

```bash
mv .agents/skills/_shared/agent-config/router-bootstrap.md /tmp/router-bootstrap.md.bak
printf '{"cwd":"/Users/paladm/git/ai-test/skillgrid-v2/.worktrees/t_06bb0c13"}' | hooks/cursor-session-start.sh
mv /tmp/router-bootstrap.md.bak .agents/skills/_shared/agent-config/router-bootstrap.md
```

Output (excerpt):

```json
{"additional_context":"skillgrid: router bootstrap fallback - first action: read `.agents/skills/using-skillgrid/SKILL.md`; announce `Using skillgrid:using-skillgrid to {purpose}`.\n\nMnemonic is active. Start with mem_session_start (title), save decisions/bugfixes via mem_save, and close with mem_session_summary + mem_session_end. Prefer mem_context / mem_search before re-deriving prior work. Call code_explore before rg."}
```

Exit: `0`. Static fallback present: **yes**. No user-visible error. Verdict: **PASS**.

### Oracle 4 — asset edit propagation (live read, no reinstall)

- Edit: appended `# ORACLE-MARKER-EDIT-TEST` to `.agents/skills/_shared/agent-config/router-bootstrap.md`.
- Re-ran in-cwd pipe: marker found in `additional_context` output (grep count `1`). Exit `0`.
- Restored: restored byte-identical content (567 bytes, original line set intact).
- Re-ran in-cwd pipe after restore: marker absent. Exit `0`.

Verdict: **PASS** (live read propagates without reinstall).

### Oracle 5 — hook syntax

Command: `bash -n hooks/cursor-session-start.sh` → exit `0`.

Verdict: **PASS**.

### Live door check (blueprint Task 1 Step 5) — PASS (controller, 2026-11-19)

Run by the controller after the implementer's headless ABANDON, using
`cursor agent -f -p` (cursor v3.29.15 — CLI agent mode fires plugin
sessionStart hooks; no display server needed).

**Plugin install layout (finding, verified empirically):** cursor loads local
plugins only from REAL directories under `~/.cursor/plugins/local/<name>/` — a
symlinked plugin directory is NOT loaded (tested). The manifest must be
`plugin.json` at the directory root; the repo's `.cursor-plugin/plugin.json`
subdirectory layout is not recognized for local plugins. Working layout used:
real dir `~/.cursor/plugins/local/skillgrid/` with a copied root `plugin.json`
+ symlinks to the worktree's `hooks/.agents/rules/agents/plugins` subdirs (hook
edited propagate live). T6 installer must use this layout.

**Positive (skillgrid project = worktree root):** fresh session
(`cursor agent -f -p "…does your session context contain a 'Skillgrid Router
Bootstrap' block?…"`) — the sessionStart hook fired (verified by a temporary
instrumentation line appended to the hook, removed before commit), and the
saved transcript
(`~/.cursor/projects/…-worktrees-t-06bb0c13/agent-transcripts/1662c7ec-…/1662c7ec-….jsonl`)
records the agent's first-turn reasoning: *"…a Skillgrid Router Bootstrap
block, and it does, under hooks_context. The first step is to read the
using-skillgrid SKILL.md file."* — bootstrap present in first-turn context,
first action correctly extracted. Verdict: **PASS**.

**Negative (non-skillgrid dir = /tmp):** fresh session, same prompt — hook
fired (project condition false → no bootstrap), agent answered: *"NO
BOOTSTRAP … my session context has no 'Skillgrid Router Bootstrap' block, so
there's no first action to state."* Verdict: **PASS**.

### Gates

| Gate | Command | Result |
|---|---|---|
| Hook syntax | `bash -n hooks/cursor-session-start.sh` | exit 0, PASS |
| Go build (modules) | `go build ./...` (mnemonic, skillgrid-cli) | exit 0, green |
| Go test | `go test ./...` (mnemonic, skillgrid-cli) | in progress / recorded at QA |

## SATISFIES (TICKET-01 acceptance mapping)

- Happy path — one-line edit reaches next session without reinstall: **Oracle 4 PASS**.
- Happy path — cursor session start injects router bootstrap even with memory server down: **injected before mnemonic block; server-down independent** (injection is a prepend into `additional_context`, independent of the mnemonic health/prime path) — hook-level only.
- Happy path — skillgrid project session receives bootstrap: **Oracle 2 PASS**.
- Edge — non-skillgrid dir receives nothing: **Oracle 1 PASS**.
- Edge — installed copy takes precedence over checkout asset: **in this environment installed copy absent** (`/Users/paladm/.skillgrid/router-bootstrap.md` not present); checkout leg exercised (Oracle 2). Installed-copy precedence is installer-owned (TICKET-06) and not re-testable here without writing to `$HOME` — noted as residual.
- Edge — hosts without tool-mapping file get plain first-action hint: asset line set includes kilo/cursor/omp plain loads — read-back PASS (asset content).
- Failure — bootstrap asset missing at session start: **Oracle 3 PASS**.
- Failure — session-start hook failure never blocks session start: fail-open shape retained (final node `|| echo '{}'`, `exit 0`).
- Failure — neither installed nor checkout asset resolvable: **Oracle 3 PASS** (installed absent + asset moved aside → static fallback).
- Failure — cursor session-start hook absent: not re-testable here (host config), noted.
- Happy path — hosts with a tool-mapping file get mapping hint: asset load-hint table references existing `using-skillgrid/references/pi-tools.md` + `opencode-tools.md`; kilo/cursor/omp plain load. Read-back PASS.

## TICKET-02 — omp session-start.js project-conditional cwd guard + child-session probe (2026-11-19)

- **Ticket:** TICKET-02 — omp session-start cwd guard + child-session probe — Tracker `t_ff99e0ec`
- **Change:** `2026-11-17-skillgrid-host-plugins`
- **Worktree:** `/Users/paladm/git/ai-test/skillgrid-v2/.worktrees/t_ff99e0ec` (branch `wt/t_ff99e0ec`, base `ebdaeaa4`)

### What was implemented

1. `hooks/session-start.js` (the shipped omp `SessionStart` hook wired via `@hsingjui/pi-hooks`):
   project-conditional cwd guard — walk up from `process.cwd()` for `.skillgrid/config.yaml`
   (bounded at the filesystem root; `path.resolve` normalizes a relative cwd so the walk terminates),
   unreadable config treated as "no project" (fail-open); full router content injected
   (`IMPORTANT`) only when a skillgrid project is detected; INFO note + exit 0 when not; child
   sessions skip injection; every path fail-open (catch → INFO → exit 0).

### Node-level oracles (hook-level gate) — all PASS

| # | Oracle | Command | Exit | EXPECT | Verdict |
|---|---|---|---|---|---|
| 1 | in-repo main: full router content injected | `echo '{}' \| node <worktree>/hooks/session-start.js` | 0 | `IMPORTANT` + router content | **PASS** |
| 2 | non-skillgrid cwd (/tmp): no injection | same hook, cwd `/tmp` | 0 | `INFO` skip, no router | **PASS** |
| 3 | unreadable config: no injection, session starts | temp dir + `chmod 000` | 0 | `INFO` skip, no router | **PASS** |
| 4 | child session (agent_kind sub) in repo: no injection | hook stdin `{..., agent_kind: "sub"}` | 0 | `INFO` skip, no router | **PASS** |
| 5 | hook syntax | `node --check hooks/session-start.js` | 0 | syntax ok | **PASS** |
| 6 | node test suite | `node --test hooks/session-start.test.mjs` | 0 | 4/4 pass | **PASS** |

RED (pre-change hook): `node --test hooks/session-start.test.mjs` → 1 pass / 3 fail
(`/tmp`: expected `INFO` got `IMPORTANT`; unreadable: expected `INFO` got `IMPORTANT`; child:
expected `INFO` got `IMPORTANT`). GREEN (post-guard): 4/4 pass.

### Child-session probe (R3 observable) — guard added

Hook-level simulation of the child-session stdin shape (`{session_id, cwd, hook_event_name,
source, agent_kind}`) fired against the hook: pre-change behavior injected the full router
(unconditional); post-change behavior skips (INFO, exit 0). Probe verdict: **child sessions skip**
the router injection → main-agent guard added (Task 2 scope).

### Live check — ABANDON (reason)

Fresh headless `omp -p "Say hello in one short sentence."` runs (in the worktree and in `/tmp`) exited
0, but no SessionStart injection was observable in the session transcripts/logs
(`skillgrid loaded` absent; no `pi-hooks` custom entry in the new session files). Headless print mode
did not surface the injected context in this environment → live clause **ABANDON** with reason: headless
`omp -p` runs did not demonstrate the injection (MCP/model infra failures also present in headless runs).

### Gates

| Gate | Command | Result |
|---|---|---|
| Hook syntax | `node --check hooks/session-start.js` | exit 0, PASS |
| Node test | `node --test hooks/session-start.test.mjs` | 4/4 pass, PASS |
| Go build | `go build ./...` (mnemonic + skillgrid-cli modules) | exit 0, green |
| Go test | `go test ./...` (mnemonic + skillgrid-cli) | mnemonic: `TestNoStalePathReferences` FAIL (pre-existing); skillgrid-cli: `internal/install` + `internal/setup` FAILs (pre-existing) — recorded, not fixed |

### SATISFIES (TICKET-02 acceptance mapping)

- Happy path — an omp session start injects the full router content: **PASS** (oracle 1 + node suite in-repo case).
- Edge — a subagent session skips the bootstrap: **PASS** (child probe + guard).
- Failure — an unreadable skillgrid configuration is treated as no project: **PASS** (oracle 3).
- Fresh omp session in `/tmp`: no injection: **PASS** (oracle 2).

### Files changed (this task)

- `hooks/session-start.js` (modified)
- `hooks/session-start.test.mjs` (new)
- `review/smoke-evidence.md` (appended)

### Concerns

1. Pre-existing Go suite failures (not fixed, out of scope): `mnemonic` `TestNoStalePathReferences`;
   `skillgrid-cli/internal/install` (`TestHermesSelectionFlow`, `TestInstallCreatesOpenCodeHookLayout`);
   `skillgrid-cli/internal/setup` (`TestSetupKilo_*`, `TestSetupOpenCode_*`).
2. Live clause ABANDON (headless `omp -p` did not show the injection).
3. Pre-existing output-contract seam question: the shipped hook emits
   `{hookSpecificName, priority, message}` while the `@hsingjui/pi-hooks` bridge reads
   `hookSpecificOutput.additionalContext` — noted, out of scope for this task's guard.
## TICKET-03 — pi project extension (plain pi host)

Change: `2026-11-17-skillgrid-host-plugins` — TICKET-03 (tasks.md lines 88-104; blueprint Task 3, lines 284-310).

### What was implemented

- Created `.pi/extensions/skillgrid-router.ts` (new): extension factory `export default function (pi: ExtensionAPI)`; `before_agent_start` appends the bootstrap section to `event.systemPromptOptions.appendSystemPrompt`; live asset resolution (`~/.skillgrid/router-bootstrap.md` → repo-clone asset → static fallback); guards: main agent only (`ctx.agent.kind === "main"`), project-conditional (bounded walk-up for `.skillgrid/config.yaml` from `ctx.cwd`); fail-open (try/catch, never throws into the host).

### TDD evidence (RED → GREEN)

- RED: module did not exist → `node --experimental-strip-types .pi/extensions/skillgrid-router.test.ts` → MODULE_NOT_FOUND (exit 1).
- GREEN: after implementation → `node --experimental-strip-types .pi/extensions/skillgrid-router.test.ts` → 4/4 PASS (exit 0).

### Node tests (TICKET-03 acceptance set)

Command: `node --experimental-strip-types .pi/extensions/skillgrid-router.test.ts`

Output:

```text
PASS main agent in project: bootstrap appended exactly once
PASS subagent: unchanged
PASS non-project cwd: unchanged
PASS missing asset: static fallback
ALL TESTS PASS
```

Exit: `0`. Verdict: **PASS**.

### BDD scenario notes (SATISFIES mapping)

- Happy path — a pi session start injects the router bootstrap: **PASS** (main agent + project cwd → bootstrap appended exactly once).
- Edge — non-skillgrid directory session receives nothing: **PASS** (non-project cwd → unchanged).
- Failure — bootstrap asset missing at session start: **PASS** (static fallback).
- Failure — subagent context gains the bootstrap: **PASS** (subagent unchanged).

### Live check (blueprint Task 3 Step 5) — ABANDON (headless)

Run: fresh headless pi session in the worktree (trivial prompt) + non-skillgrid dir negative.

Result: **ABANDON** — pi CLI not on PATH (`pi` / `skill-pi` broken symlink); `@earendil-works/pi` not resolvable on npm (404); bare `npx -y pi@2.0.5` / `npx -y pi@latest` invocations returned version output (`3`) without exposing a headless prompt surface. No headless pi session could be driven in this environment; hook-level oracles are the hard gate.

### Gates

| # | Command | Exit | EXPECT | Verdict |
|---|---|---|---|---|
| 1 | `node --experimental-strip-types .pi/extensions/skillgrid-router.test.ts` | 0 | 4/4 PASS | **PASS** (match) |
| 2 | live fresh pi session (repo + /tmp) | n/a | bootstrap in first-turn context / absent | **ABANDON** — headless pi CLI infeasible (see above) |

### Files changed (this task)

- `.pi/extensions/skillgrid-router.ts` (new)
- `.pi/extensions/skillgrid-router.test.ts` (new)
- `.pi/extensions/package.json` (new — `{"type":"module"}` ESM marker required by `node --experimental-strip-types` for the TS ESM test)
- `review/smoke-evidence.md` (appended)

### Concerns

1. `.pi/extensions/package.json` (`{"type":"module"}`) added — required for `node --experimental-strip-types` to load ESM-syntax TS from `.pi/extensions/` (repo-root package.json has no `type`, so `.ts` is otherwise treated as CJS). Mirrors the opencode ESM-marker precedent.
2. Live pi session could not be driven headless; hard gate satisfied by the node test oracles.
## TICKET-05 — kilo router plugin (config + messages.transform)

Change: `2026-11-17-skillgrid-host-plugins` — TICKET-05 (tasks.md lines 120-134; blueprint Task 5, lines 357-396).

### What was implemented

- Created `plugins/kilo/skillgrid-router.ts` (new): kilo TS plugin mirroring the proven `~/.config/kilo/plugin/superpowers.js` pattern — `config` hook pushes the repo `.agents/skills` dir into `config.skills.paths`; `experimental.chat.messages.transform` injects the bootstrap into the first user message (present-guard: skip if already present; skip subagent sessions); `experimental.chat.system.transform` fallback for non-interactive paths. Live asset resolution (`~/.skillgrid/router-bootstrap.md` → repo-clone asset → static fallback); project-conditional (bounded walk-up for `.skillgrid/config.yaml` from the plugin `directory`); fail-open (fallback).

### TDD evidence (RED → GREEN)

- **RED:** before implementation, `node --experimental-strip-types plugins/kilo/skillgrid-router.test.ts` → MODULE_NOT_FOUND (exit 1).
- **GREEN:** after implementation → `node --experimental-strip-types plugins/kilo/skillgrid-router.test.ts` → 5/5 PASS (exit 0).

### Node tests (TICKET-05 acceptance set)

Command: `node --experimental-strip-types plugins/kilo/skillgrid-router.test.ts`

Output:

```text
PASS: all kilo router plugin cases green
```

Exit: `0`. Verdict: **PASS**.

### Acceptance cases (TICKET-05)

- Happy path — first user message + project cwd + main agent → bootstrap injected exactly once: **PASS**.
- Present-guard — second transform → no-op: **PASS**.
- Subagent → unchanged: **PASS**.
- Non-project cwd → unchanged: **PASS**.
- Missing asset → static fallback: **PASS**.

### Live check (blueprint Task 5 Step 5) — ABANDON (headless)

Run: fresh headless kilo session in the worktree (trivial prompt) + non-skillgrid dir negative.

Result: **ABANDON** — `kilo run` exited 1 with `Error: System message must be at the beginning.` before turn completion, in both the skillgrid worktree and a non-skillgrid dir; headless run not drivable to a positive signal in this environment.

### Gates

| # | Command | Exit | EXPECT | Verdict |
|---|---|---|---|---|
| 1 | `node --experimental-strip-types plugins/kilo/skillgrid-router.test.ts` | 0 | all cases PASS | **PASS** (match) |

### Files changed (this task)

- `plugins/kilo/skillgrid-router.ts` (new)
- `plugins/kilo/skillgrid-router.test.ts` (new)
- `plugins/kilo/package.json` (new)
- `review/smoke-evidence.md` (appended)

### Concerns

1. `plugins/kilo/package.json` (`{"type":"module"}`) added — required for `node --experimental-strip-types` to load ESM-syntax TS from `plugins/kilo/` (repo-root package.json has no `type`, so `.ts` is otherwise treated as CJS). Mirrors the pi/opencode ESM-marker precedent.
2. Live check ABANDON (headless) — hard gate satisfied by the node test oracles.
3. Live `kilo run` failed with `Error: System message must be at the beginning.` — inject-then-system ordering not yet satisfiable headlessly in this environment.
## TICKET-04 — opencode router module (V2 + V1 dual export)

- **Ticket:** TICKET-04 (opencode router module — V2 + V1 dual export) — Tracker `t_7df6b1f4`
- **Change:** `2026-11-17-skillgrid-host-plugins`
- **Worktree:** `/Users/paladm/git/ai-test/skillgrid-v2/.worktrees/t_7df6b1f4` (branch `wt/t_7df6b1f4`, base `ebdaeaa4`)

### Oracle 1 — V2 context entry present in-project (node test green)

Command:

```bash
node --experimental-strip-types plugins/opencode/test/skillgrid-router.test.ts
```

Output:

```text
ok - v2 default export is Plugin.define-shaped ({id, setup})
ok - v2 in-project: session.created puts the bootstrap context entry
ok - v2 non-project: session.created puts nothing
ok - v2 subagent: session.created puts nothing
ok - v2 missing asset: static fallback is put
ok - v1 named export SkillgridRouter is a function
ok - v1 in-project: session.created puts the bootstrap context entry
ok - v1 non-project: session.created puts nothing

8 checks passed
```

Exit: `0`. Verdict: **PASS**.

### Oracle 2 — live headless opencode session (in-repo positive)

Fresh headless session via `opencode run "Reply with the single word: ok"` in the worktree root with
the router plugin present in the live plugin dir (`~/.config/opencode/plugins/skillgrid-router.ts`,
copied from the worktree module for the live pass).

- Run: exit `0`; the model answered `ok`.
- Context-entry injection path: exercised by the green node tests (in-project cwd → `put` with key
  `skillgrid-router` + bootstrap value); the live `<context key="skillgrid-router">` block was not
  directly readable from the CLI transcript in this pass.

### Oracle 3 — live headless opencode session (non-skillgrid negative)

Fresh headless session via `opencode run` in `/tmp`.

- Run: failed at the model step — `Error: Add credits to continue, or switch to a free model`.
- Negative injection path (non-project cwd → no put) is asserted green in the node tests; the live
  negative run is blocked by the provider error in this pass.

### Gates

| Gate | Command | Result |
|---|---|---|
| Node test (V2 + V1) | `node --experimental-strip-types plugins/opencode/test/skillgrid-router.test.ts` | exit 0, PASS |

TICKET-02 TEST LINE

## TICKET-02 — omp session-start.js project-conditional cwd guard + child-session probe (2026-11-19)

- **Ticket:** TICKET-02 — omp session-start cwd guard + child-session probe — Tracker `t_ff99e0ec`
- **Change:** `2026-11-17-skillgrid-host-plugins`
- **Worktree:** `/Users/paladm/git/ai-test/skillgrid-v2/.worktrees/t_ff99e0ec` (branch `wt/t_ff99e0ec`, base `ebdaeaa4`)

### What was implemented

1. `hooks/session-start.js` (the shipped omp `SessionStart` hook wired via `@hsingjui/pi-hooks`):
   project-conditional cwd guard — walk up from `process.cwd()` for `.skillgrid/config.yaml`
   (bounded at the filesystem root; `path.resolve` normalizes a relative cwd so the walk terminates),
   unreadable config treated as "no project" (fail-open); full router content injected
   (`IMPORTANT`) only when a skillgrid project is detected; INFO note + exit 0 when not; child
   sessions skip injection; every path fail-open (catch → INFO → exit 0).

### Node-level oracles (hook-level gate) — all PASS

| # | Oracle | Command | Exit | EXPECT | Verdict |
|---|---|---|---|---|---|
| 1 | in-repo main: full router content injected | `echo '{}' \| node <worktree>/hooks/session-start.js` | 0 | `IMPORTANT` + router content | **PASS** |
| 2 | non-skillgrid cwd (/tmp): no injection | same hook, cwd `/tmp` | 0 | `INFO` skip, no router | **PASS** |
| 3 | unreadable config: no injection, session starts | temp dir + `chmod 000` | 0 | `INFO` skip, no router | **PASS** |
| 4 | child session (agent_kind sub) in repo: no injection | hook stdin `{..., agent_kind: "sub"}` | 0 | `INFO` skip, no router | **PASS** |
| 5 | hook syntax | `node --check hooks/session-start.js` | 0 | syntax ok | **PASS** |
| 6 | node test suite | `node --test hooks/session-start.test.mjs` | 0 | 4/4 pass | **PASS** |

RED (pre-change hook): `node --test hooks/session-start.test.mjs` → 1 pass / 3 fail
(`/tmp`: expected `INFO` got `IMPORTANT`; unreadable: expected `INFO` got `IMPORTANT`; child:
expected `INFO` got `IMPORTANT`). GREEN (post-guard): 4/4 pass.

### Child-session probe (R3 observable) — guard added

Hook-level simulation of the child-session stdin shape (`{session_id, cwd, hook_event_name,
source, agent_kind}`) fired against the hook: pre-change behavior injected the full router
(unconditional); post-change behavior skips (INFO, exit 0). Probe verdict: **child sessions skip**
the router injection → main-agent guard added (Task 2 scope).

### Live check — ABANDON (reason)

Fresh headless `omp -p "Say hello in one short sentence."` runs (in the worktree and in `/tmp`) exited
0, but no SessionStart injection was observable in the session transcripts/logs
(`skillgrid loaded` absent; no `pi-hooks` custom entry in the new session files). Headless print mode
did not surface the injected context in this environment → live clause **ABANDON** with reason: headless
`omp -p` runs did not demonstrate the injection (MCP/model infra failures also present in headless runs).

### Gates

| Gate | Command | Result |
|---|---|---|
| Hook syntax | `node --check hooks/session-start.js` | exit 0, PASS |
| Node test | `node --test hooks/session-start.test.mjs` | 4/4 pass, PASS |
| Go build | `go build ./...` (mnemonic + skillgrid-cli modules) | exit 0, green |
| Go test | `go test ./...` (mnemonic + skillgrid-cli) | mnemonic: `TestNoStalePathReferences` FAIL (pre-existing); skillgrid-cli: `internal/install` + `internal/setup` FAILs (pre-existing) — recorded, not fixed |

### SATISFIES (TICKET-02 acceptance mapping)

- Happy path — an omp session start injects the full router content: **PASS** (oracle 1 + node suite in-repo case).
- Edge — a subagent session skips the bootstrap: **PASS** (child probe + guard).
- Failure — an unreadable skillgrid configuration is treated as no project: **PASS** (oracle 3).
- Fresh omp session in `/tmp`: no injection: **PASS** (oracle 2).

### Files changed (this task)

- `hooks/session-start.js` (modified)
- `hooks/session-start.test.mjs` (new)
- `review/smoke-evidence.md` (appended)

### Concerns

1. Pre-existing Go suite failures (not fixed, out of scope): `mnemonic` `TestNoStalePathReferences`;
   `skillgrid-cli/internal/install` (`TestHermesSelectionFlow`, `TestInstallCreatesOpenCodeHookLayout`);
   `skillgrid-cli/internal/setup` (`TestSetupKilo_*`, `TestSetupOpenCode_*`).
2. Live clause ABANDON (headless `omp -p` did not show the injection).
3. Pre-existing output-contract seam question: the shipped hook emits
   `{hookSpecificName, priority, message}` while the `@hsingjui/pi-hooks` bridge reads
   `hookSpecificOutput.additionalContext` — noted, out of scope for this task's guard.

## TICKET-02 — omp session-start.js project-conditional cwd guard + child-session probe (2026-11-19)

- **Ticket:** TICKET-02 — omp session-start cwd guard + child-session probe — Tracker `t_ff99e0ec`
- **Change:** `2026-11-17-skillgrid-host-plugins`
- **Worktree:** `/Users/paladm/git/ai-test/skillgrid-v2/.worktrees/t_ff99e0ec` (branch `wt/t_ff99e0ec`, base `ebdaeaa4`)

### What was implemented

1. `hooks/session-start.js` (the shipped omp `SessionStart` hook wired via `@hsingjui/pi-hooks`):
   project-conditional cwd guard — walk up from `process.cwd()` for `.skillgrid/config.yaml`
   (bounded at the filesystem root; `path.resolve` normalizes a relative cwd so the walk terminates),
   unreadable config treated as "no project" (fail-open); full router content injected
   (`IMPORTANT`) only when a skillgrid project is detected; INFO note + exit 0 when not; child
   sessions skip injection; every path fail-open (catch → INFO → exit 0).

### Node-level oracles (hook-level gate) — all PASS

| # | Oracle | Command | Exit | EXPECT | Verdict |
|---|---|---|---|---|---|
| 1 | in-repo main: full router content injected | `echo '{}' \| node <worktree>/hooks/session-start.js` | 0 | `IMPORTANT` + router content | **PASS** |
| 2 | non-skillgrid cwd (/tmp): no injection | same hook, cwd `/tmp` | 0 | `INFO` skip, no router | **PASS** |
| 3 | unreadable config: no injection, session starts | temp dir + `chmod 000` | 0 | `INFO` skip, no router | **PASS** |
| 4 | child session (agent_kind sub) in repo: no injection | hook stdin `{..., agent_kind: "sub"}` | 0 | `INFO` skip, no router | **PASS** |
| 5 | hook syntax | `node --check hooks/session-start.js` | 0 | syntax ok | **PASS** |
| 6 | node test suite | `node --test hooks/session-start.test.mjs` | 0 | 4/4 pass | **PASS** |

RED (pre-change hook): `node --test hooks/session-start.test.mjs` → 1 pass / 3 fail
(`/tmp`: expected `INFO` got `IMPORTANT`; unreadable: expected `INFO` got `IMPORTANT`; child:
expected `INFO` got `IMPORTANT`). GREEN (post-guard): 4/4 pass.

### Child-session probe (R3 observable) — guard added

Hook-level simulation of the child-session stdin shape (`{session_id, cwd, hook_event_name,
source, agent_kind}`) fired against the hook: pre-change behavior injected the full router
(unconditional); post-change behavior skips (INFO, exit 0). Probe verdict: **child sessions skip**
the router injection → main-agent guard added (Task 2 scope).

### Live check — ABANDON (reason)

Fresh headless `omp -p "Say hello in one short sentence."` runs (in the worktree and in `/tmp`) exited
0, but no SessionStart injection was observable in the session transcripts/logs
(`skillgrid loaded` absent; no `pi-hooks` custom entry in the new session files). Headless print mode
did not surface the injected context in this environment → live clause **ABANDON** with reason: headless
`omp -p` runs did not demonstrate the injection (MCP/model infra failures also present in headless runs).

### Gates

| Gate | Command | Result |
|---|---|---|
| Hook syntax | `node --check hooks/session-start.js` | exit 0, PASS |
| Node test | `node --test hooks/session-start.test.mjs` | 4/4 pass, PASS |
| Go build | `go build ./...` (mnemonic + skillgrid-cli modules) | exit 0, green |
| Go test | `go test ./...` (mnemonic + skillgrid-cli) | mnemonic: `TestNoStalePathReferences` FAIL (pre-existing); skillgrid-cli: `internal/install` + `internal/setup` FAILs (pre-existing) — recorded, not fixed |

### SATISFIES (TICKET-02 acceptance mapping)

- Happy path — an omp session start injects the full router content: **PASS** (oracle 1 + node suite in-repo case).
- Edge — a subagent session skips the bootstrap: **PASS** (child probe + guard).
- Failure — an unreadable skillgrid configuration is treated as no project: **PASS** (oracle 3).
- Fresh omp session in `/tmp`: no injection: **PASS** (oracle 2).

### Files changed (this task)

- `hooks/session-start.js` (modified)
- `hooks/session-start.test.mjs` (new)
- `review/smoke-evidence.md` (appended)

### Concerns

1. Pre-existing Go suite failures (not fixed, out of scope): `mnemonic` `TestNoStalePathReferences`;
   `skillgrid-cli/internal/install` (`TestHermesSelectionFlow`, `TestInstallCreatesOpenCodeHookLayout`);
   `skillgrid-cli/internal/setup` (`TestSetupKilo_*`, `TestSetupOpenCode_*`).
2. Live clause ABANDON (headless `omp -p` did not show the injection).
3. Pre-existing output-contract seam question: the shipped hook emits
   `{hookSpecificName, priority, message}` while the `@hsingjui/pi-hooks` bridge reads
   `hookSpecificOutput.additionalContext` — noted, out of scope for this task's guard.

## TICKET-03 — pi project extension (plain pi host)

Change: `2026-11-17-skillgrid-host-plugins` — TICKET-03 (tasks.md lines 88-104; blueprint Task 3, lines 284-310).

### What was implemented

- Created `.pi/extensions/skillgrid-router.ts` (new): extension factory `export default function (pi: ExtensionAPI)`; `before_agent_start` appends the bootstrap section to `event.systemPromptOptions.appendSystemPrompt`; live asset resolution (`~/.skillgrid/router-bootstrap.md` → repo-clone asset → static fallback); guards: main agent only (`ctx.agent.kind === "main"`), project-conditional (bounded walk-up for `.skillgrid/config.yaml` from `ctx.cwd`); fail-open (try/catch, never throws into the host).

### TDD evidence (RED → GREEN)

- RED: module did not exist → `node --experimental-strip-types .pi/extensions/skillgrid-router.test.ts` → MODULE_NOT_FOUND (exit 1).
- GREEN: after implementation → `node --experimental-strip-types .pi/extensions/skillgrid-router.test.ts` → 4/4 PASS (exit 0).

### Node tests (TICKET-03 acceptance set)

Command: `node --experimental-strip-types .pi/extensions/skillgrid-router.test.ts`

Output:

```text
PASS main agent in project: bootstrap appended exactly once
PASS subagent: unchanged
PASS non-project cwd: unchanged
PASS missing asset: static fallback
ALL TESTS PASS
```

Exit: `0`. Verdict: **PASS**.

### BDD scenario notes (SATISFIES mapping)

- Happy path — a pi session start injects the router bootstrap: **PASS** (main agent + project cwd → bootstrap appended exactly once).
- Edge — non-skillgrid directory session receives nothing: **PASS** (non-project cwd → unchanged).
- Failure — bootstrap asset missing at session start: **PASS** (static fallback).
- Failure — subagent context gains the bootstrap: **PASS** (subagent unchanged).

### Live check (blueprint Task 3 Step 5) — ABANDON (headless)

Run: fresh headless pi session in the worktree (trivial prompt) + non-skillgrid dir negative.

Result: **ABANDON** — pi CLI not on PATH (`pi` / `skill-pi` broken symlink); `@earendil-works/pi` not resolvable on npm (404); bare `npx -y pi@2.0.5` / `npx -y pi@latest` invocations returned version output (`3`) without exposing a headless prompt surface. No headless pi session could be driven in this environment; hook-level oracles are the hard gate.

### Gates

| # | Command | Exit | EXPECT | Verdict |
|---|---|---|---|---|
| 1 | `node --experimental-strip-types .pi/extensions/skillgrid-router.test.ts` | 0 | 4/4 PASS | **PASS** (match) |
| 2 | live fresh pi session (repo + /tmp) | n/a | bootstrap in first-turn context / absent | **ABANDON** — headless pi CLI infeasible (see above) |

### Files changed (this task)

- `.pi/extensions/skillgrid-router.ts` (new)
- `.pi/extensions/skillgrid-router.test.ts` (new)
- `.pi/extensions/package.json` (new — `{"type":"module"}` ESM marker required by `node --experimental-strip-types` for the TS ESM test)
- `review/smoke-evidence.md` (appended)

### Concerns

1. `.pi/extensions/package.json` (`{"type":"module"}`) added — required for `node --experimental-strip-types` to load ESM-syntax TS from `.pi/extensions/` (repo-root package.json has no `type`, so `.ts` is otherwise treated as CJS). Mirrors the opencode ESM-marker precedent.
2. Live pi session could not be driven headless; hard gate satisfied by the node test oracles.

## TICKET-03 — pi project extension (plain pi host)

Change: `2026-11-17-skillgrid-host-plugins` — TICKET-03 (tasks.md lines 88-104; blueprint Task 3, lines 284-310).

### What was implemented

- Created `.pi/extensions/skillgrid-router.ts` (new): extension factory `export default function (pi: ExtensionAPI)`; `before_agent_start` appends the bootstrap section to `event.systemPromptOptions.appendSystemPrompt`; live asset resolution (`~/.skillgrid/router-bootstrap.md` → repo-clone asset → static fallback); guards: main agent only (`ctx.agent.kind === "main"`), project-conditional (bounded walk-up for `.skillgrid/config.yaml` from `ctx.cwd`); fail-open (try/catch, never throws into the host).

### TDD evidence (RED → GREEN)

- RED: module did not exist → `node --experimental-strip-types .pi/extensions/skillgrid-router.test.ts` → MODULE_NOT_FOUND (exit 1).
- GREEN: after implementation → `node --experimental-strip-types .pi/extensions/skillgrid-router.test.ts` → 4/4 PASS (exit 0).

### Node tests (TICKET-03 acceptance set)

Command: `node --experimental-strip-types .pi/extensions/skillgrid-router.test.ts`

Output:

```text
PASS main agent in project: bootstrap appended exactly once
PASS subagent: unchanged
PASS non-project cwd: unchanged
PASS missing asset: static fallback
ALL TESTS PASS
```

Exit: `0`. Verdict: **PASS**.

### BDD scenario notes (SATISFIES mapping)

- Happy path — a pi session start injects the router bootstrap: **PASS** (main agent + project cwd → bootstrap appended exactly once).
- Edge — non-skillgrid directory session receives nothing: **PASS** (non-project cwd → unchanged).
- Failure — bootstrap asset missing at session start: **PASS** (static fallback).
- Failure — subagent context gains the bootstrap: **PASS** (subagent unchanged).

### Live check (blueprint Task 3 Step 5) — ABANDON (headless)

Run: fresh headless pi session in the worktree (trivial prompt) + non-skillgrid dir negative.

Result: **ABANDON** — pi CLI not on PATH (`pi` / `skill-pi` broken symlink); `@earendil-works/pi` not resolvable on npm (404); bare `npx -y pi@2.0.5` / `npx -y pi@latest` invocations returned version output (`3`) without exposing a headless prompt surface. No headless pi session could be driven in this environment; hook-level oracles are the hard gate.

### Gates

| # | Command | Exit | EXPECT | Verdict |
|---|---|---|---|---|
| 1 | `node --experimental-strip-types .pi/extensions/skillgrid-router.test.ts` | 0 | 4/4 PASS | **PASS** (match) |
| 2 | live fresh pi session (repo + /tmp) | n/a | bootstrap in first-turn context / absent | **ABANDON** — headless pi CLI infeasible (see above) |

### Files changed (this task)

- `.pi/extensions/skillgrid-router.ts` (new)
- `.pi/extensions/skillgrid-router.test.ts` (new)
- `.pi/extensions/package.json` (new — `{"type":"module"}` ESM marker required by `node --experimental-strip-types` for the TS ESM test)
- `review/smoke-evidence.md` (appended)

### Concerns

1. `.pi/extensions/package.json` (`{"type":"module"}`) added — required for `node --experimental-strip-types` to load ESM-syntax TS from `.pi/extensions/` (repo-root package.json has no `type`, so `.ts` is otherwise treated as CJS). Mirrors the opencode ESM-marker precedent.
2. Live pi session could not be driven headless; hard gate satisfied by the node test oracles.

## TICKET-03 — pi project extension (plain pi host)

Change: `2026-11-17-skillgrid-host-plugins` — TICKET-03 (tasks.md lines 88-104; blueprint Task 3, lines 284-310).

### What was implemented

- Created `.pi/extensions/skillgrid-router.ts` (new): extension factory `export default function (pi: ExtensionAPI)`; `before_agent_start` appends the bootstrap section to `event.systemPromptOptions.appendSystemPrompt`; live asset resolution (`~/.skillgrid/router-bootstrap.md` → repo-clone asset → static fallback); guards: main agent only (`ctx.agent.kind === "main"`), project-conditional (bounded walk-up for `.skillgrid/config.yaml` from `ctx.cwd`); fail-open (try/catch, never throws into the host).

### TDD evidence (RED → GREEN)

- RED: module did not exist → `node --experimental-strip-types .pi/extensions/skillgrid-router.test.ts` → MODULE_NOT_FOUND (exit 1).
- GREEN: after implementation → `node --experimental-strip-types .pi/extensions/skillgrid-router.test.ts` → 4/4 PASS (exit 0).

### Node tests (TICKET-03 acceptance set)

Command: `node --experimental-strip-types .pi/extensions/skillgrid-router.test.ts`

Output:

```text
PASS main agent in project: bootstrap appended exactly once
PASS subagent: unchanged
PASS non-project cwd: unchanged
PASS missing asset: static fallback
ALL TESTS PASS
```

Exit: `0`. Verdict: **PASS**.

### BDD scenario notes (SATISFIES mapping)

- Happy path — a pi session start injects the router bootstrap: **PASS** (main agent + project cwd → bootstrap appended exactly once).
- Edge — non-skillgrid directory session receives nothing: **PASS** (non-project cwd → unchanged).
- Failure — bootstrap asset missing at session start: **PASS** (static fallback).
- Failure — subagent context gains the bootstrap: **PASS** (subagent unchanged).

### Live check (blueprint Task 3 Step 5) — ABANDON (headless)

Run: fresh headless pi session in the worktree (trivial prompt) + non-skillgrid dir negative.

Result: **ABANDON** — pi CLI not on PATH (`pi` / `skill-pi` broken symlink); `@earendil-works/pi` not resolvable on npm (404); bare `npx -y pi@2.0.5` / `npx -y pi@latest` invocations returned version output (`3`) without exposing a headless prompt surface. No headless pi session could be driven in this environment; hook-level oracles are the hard gate.

### Gates

| # | Command | Exit | EXPECT | Verdict |
|---|---|---|---|---|
| 1 | `node --experimental-strip-types .pi/extensions/skillgrid-router.test.ts` | 0 | 4/4 PASS | **PASS** (match) |
| 2 | live fresh pi session (repo + /tmp) | n/a | bootstrap in first-turn context / absent | **ABANDON** — headless pi CLI infeasible (see above) |

### Files changed (this task)

- `.pi/extensions/skillgrid-router.ts` (new)
- `.pi/extensions/skillgrid-router.test.ts` (new)
- `.pi/extensions/package.json` (new — `{"type":"module"}` ESM marker required by `node --experimental-strip-types` for the TS ESM test)
- `review/smoke-evidence.md` (appended)

### Concerns

1. `.pi/extensions/package.json` (`{"type":"module"}`) added — required for `node --experimental-strip-types` to load ESM-syntax TS from `.pi/extensions/` (repo-root package.json has no `type`, so `.ts` is otherwise treated as CJS). Mirrors the opencode ESM-marker precedent.
2. Live pi session could not be driven headless; hard gate satisfied by the node test oracles.

## TICKET-03 — pi project extension (plain pi host)

Change: `2026-11-17-skillgrid-host-plugins` — TICKET-03 (tasks.md lines 88-104; blueprint Task 3, lines 284-310).

### What was implemented

- Created `.pi/extensions/skillgrid-router.ts` (new): extension factory `export default function (pi: ExtensionAPI)`; `before_agent_start` appends the bootstrap section to `event.systemPromptOptions.appendSystemPrompt`; live asset resolution (`~/.skillgrid/router-bootstrap.md` → repo-clone asset → static fallback); guards: main agent only (`ctx.agent.kind === "main"`), project-conditional (bounded walk-up for `.skillgrid/config.yaml` from `ctx.cwd`); fail-open (try/catch, never throws into the host).

### TDD evidence (RED → GREEN)

- RED: module did not exist → `node --experimental-strip-types .pi/extensions/skillgrid-router.test.ts` → MODULE_NOT_FOUND (exit 1).
- GREEN: after implementation → `node --experimental-strip-types .pi/extensions/skillgrid-router.test.ts` → 4/4 PASS (exit 0).

### Node tests (TICKET-03 acceptance set)

Command: `node --experimental-strip-types .pi/extensions/skillgrid-router.test.ts`

Output:

```text
PASS main agent in project: bootstrap appended exactly once
PASS subagent: unchanged
PASS non-project cwd: unchanged
PASS missing asset: static fallback
ALL TESTS PASS
```

Exit: `0`. Verdict: **PASS**.

### BDD scenario notes (SATISFIES mapping)

- Happy path — a pi session start injects the router bootstrap: **PASS** (main agent + project cwd → bootstrap appended exactly once).
- Edge — non-skillgrid directory session receives nothing: **PASS** (non-project cwd → unchanged).
- Failure — bootstrap asset missing at session start: **PASS** (static fallback).
- Failure — subagent context gains the bootstrap: **PASS** (subagent unchanged).

### Live check (blueprint Task 3 Step 5) — ABANDON (headless)

Run: fresh headless pi session in the worktree (trivial prompt) + non-skillgrid dir negative.

Result: **ABANDON** — pi CLI not on PATH (`pi` / `skill-pi` broken symlink); `@earendil-works/pi` not resolvable on npm (404); bare `npx -y pi@2.0.5` / `npx -y pi@latest` invocations returned version output (`3`) without exposing a headless prompt surface. No headless pi session could be driven in this environment; hook-level oracles are the hard gate.

### Gates

| # | Command | Exit | EXPECT | Verdict |
|---|---|---|---|---|
| 1 | `node --experimental-strip-types .pi/extensions/skillgrid-router.test.ts` | 0 | 4/4 PASS | **PASS** (match) |
| 2 | live fresh pi session (repo + /tmp) | n/a | bootstrap in first-turn context / absent | **ABANDON** — headless pi CLI infeasible (see above) |

### Files changed (this task)

- `.pi/extensions/skillgrid-router.ts` (new)
- `.pi/extensions/skillgrid-router.test.ts` (new)
- `.pi/extensions/package.json` (new — `{"type":"module"}` ESM marker required by `node --experimental-strip-types` for the TS ESM test)
- `review/smoke-evidence.md` (appended)

### Concerns

1. `.pi/extensions/package.json` (`{"type":"module"}`) added — required for `node --experimental-strip-types` to load ESM-syntax TS from `.pi/extensions/` (repo-root package.json has no `type`, so `.ts` is otherwise treated as CJS). Mirrors the opencode ESM-marker precedent.
2. Live pi session could not be driven headless; hard gate satisfied by the node test oracles.

## TICKET-05 — kilo router plugin (config + messages.transform)

Change: `2026-11-17-skillgrid-host-plugins` — TICKET-05 (tasks.md lines 120-134; blueprint Task 5, lines 357-396).

### What was implemented

- Created `plugins/kilo/skillgrid-router.ts` (new): kilo TS plugin mirroring the proven `~/.config/kilo/plugin/superpowers.js` pattern — `config` hook pushes the repo `.agents/skills` dir into `config.skills.paths`; `experimental.chat.messages.transform` injects the bootstrap into the first user message (present-guard: skip if already present; skip subagent sessions); `experimental.chat.system.transform` fallback for non-interactive paths. Live asset resolution (`~/.skillgrid/router-bootstrap.md` → repo-clone asset → static fallback); project-conditional (bounded walk-up for `.skillgrid/config.yaml` from the plugin `directory`); fail-open (fallback).

### TDD evidence (RED → GREEN)

- RED: module did not exist → `node --experimental-strip-types plugins/kilo/skillgrid-router.test.ts` → MODULE_NOT_FOUND (exit 1).
- GREEN: after implementation → `node --experimental-strip-types plugins/kilo/skillgrid-router.test.ts` → PASS: all kilo router plugin cases green (exit 0).

### Node tests (TICKET-05 acceptance set)

Command: `node --experimental-strip-types plugins/kilo/skillgrid-router.test.ts`

Output:

```text
PASS: all kilo router plugin cases green
```

Exit: `0`. Verdict: **PASS**.

### Acceptance cases (TICKET-05)

- First user message + project cwd + main agent → first-action line exactly once: **PASS**.
- Second transform → no-op (present-guard): **PASS**.
- Subagent → unchanged: **PASS**.
- Non-project cwd → unchanged: **PASS**.
- Missing asset → static fallback: **PASS**.

### Live check (blueprint Task 5 Step 5) — ABANDON (headless)

Run: fresh headless kilo session in the worktree (trivial prompt) + non-skillgrid dir negative.

Result: **ABANDON** — `kilo run` exits 1 with `Error: System message must be at the beginning.` before turn completion, in both the skillgrid worktree and a non-skillgrid dir; headless run not drivable to a positive signal in this environment.

### Gates

| # | Command | Exit | EXPECT | Verdict |
|---|---|---|---|---|
| 1 | `node --experimental-strip-types plugins/kilo/skillgrid-router.test.ts` | 0 | all cases PASS | **PASS** (match) |

### Files changed (this task)

- `plugins/kilo/skillgrid-router.ts` (new)
- `plugins/kilo/skillgrid-router.test.ts` (new)
- `plugins/kilo/package.json` (new — `{"type":"module"}` ESM marker required by `node --experimental-strip-types` for the TS ESM test)
- `review/smoke-evidence.md` (appended)

### Concerns

1. `plugins/kilo/package.json` (`{"type":"module"}`) added — required for `node --experimental-strip-types` to load ESM-syntax TS from `plugins/kilo/` (repo-root package.json has no `type`, so `.ts` is otherwise treated as CJS). Mirrors the pi/opencode ESM-marker precedent.
2. Live check ABANDON (headless) — hard gate satisfied by the node test oracles.

## TICKET-04 controller fix: version-tolerant fallback + live verification (2026-11-19)

**Defect (found by controller live instrumentation):** the committed V2 path used
`ctx.session.instructions.entry.put`, which does not exist on the user's
opencode 2.0.28 (released 2026-10-13). That API was added by
PR anomalyco/opencode#56550 (merged ~2026-10-21; available from ~2.2.0, current
latest 2.3.2). On 2.0.28 `ctx.session` has no `instructions` namespace, so the
injection was a silent no-op (live-confirmed: instrumented plugin logged
`no put fn`, zero `instruction_entry` rows created).

**Probe findings (5 instrumented standalone runs, worktree cwd):**
- `ctx.session` keys on 2.0.28: `hook,create,get,remove,switchAgent,switchModel,prompt,generate,command,compact,synthetic,interrupt,update,move,wait,context` (no `instructions`)
- `ctx.experimental` = `{terminal}` only; `ctx.rpc` = Plugin.Rpc.fromEffect callable-client factory (method-dispatch call shapes fail internally); no TCP listener in `--standalone` (in-process server)
- The 2.0.28 mechanism (proven by the existing skillgrid.context plugin): `ctx.session.hook("context", handler)` fires on every LLM request path with `{messages, system, tools}`; `system` is an array of `{type:"text",text}` parts (a plain-string push fails server-side schema validation); the event also carries `sessionID` + `agent` (main session agent id observed: `"build"`)
- `ctx.session.get` from the hook callback does not settle within the process lifetime; async IIFE bodies do not execute from within the hook callback in the Bun/Effect host (sync statements do) — parent-id lookup dropped, agent filter used instead

**Fix:** V2 setup selects at load: (a) `session.created` -> `instructions.entry.put` when the namespace exists (opencode >= 2.2); (b) otherwise the `"context"` hook, registered once, appends the bootstrap as a `{type:"text",text}` part to each request's system prompt, project-conditional, agent-filtered (`isMainAgent` on `event.agent`), in-assembly dedup, fail-open.

**Tests:** 12/12 green (4 new fallback checks; `check()` made async-aware — the previous sync try/catch counted dispatched-but-unsettled async checks as passed).

**Live verification (opencode 2.0.28 standalone, model tidepool-alpha-free):**
- Positive (cwd = skillgrid worktree): asked "what does the skillgrid router bootstrap say the first action is?" -> `Yes: at the start of any skillgrid task, first read \`using-skillgrid\` — it classifies the request, runs the start gates, and names exactly one first skill.` (verbatim bootstrap content)
- Negative (cwd = /tmp/skillgrid-neg-test, no .skillgrid): same prompt -> `NO BOOTSTRAP`
- Residual env note: the standalone model provider is flaky on this machine (`Transport: The socket connection was closed unexpectedly` killed 2 of ~10 runs at 160-200s); retries succeeded.

## TICKET-07 — 5-host live smoke (backstop) (2026-11-22)

- **Ticket:** TICKET-07 — 5-host live smoke (backstop) — Tracker `t_7754f121`
- **Change:** `2026-11-17-skillgrid-host-plugins`
- **Worktree:** `/Users/paladm/git/ai-test/skillgrid-v2/.worktrees/t_7754f121` (branch `wt/t_7754f121`, base `9595cf42`, smoke at `8fcb3039`)

### Install (fresh, per runbook)

Installer checkout rebuilt as a local clone of the main repo at `wt/t_7754f121` tip
(`~/.skillgrid/repos/skillgrid`; the prior stale worktree there had a broken
`git pull --ff-only` path — removed).

```
NPM_CONFIG_PREFIX=$HOME/.npm-global /tmp/skillgrid-t7 install --agents opencode,kilo,cursor
→ done in 19.305s
```

- router plugins copied: `~/.config/opencode/plugins/skillgrid-router.ts`
  (+ config registration), `~/.config/kilo/plugin/skillgrid-router.ts`
  (+ config registration: `plugin: ["./plugin/skillgrid-router.ts"]`)
- hooks mirrored (fixed `session-start.js` with the pi-hooks v0.0.2 contract),
  bootstrap asset synced (`~/.skillgrid/router-bootstrap.md` unchanged)
- stale kilo `hook/hooks.yaml` plugin registration dropped (ADR-0032
  `dropRetiredPlugins`) — the missing `opencode-session-start.sh` reference is
  now dead config

Install ruling: `skillgrid install --agents skill-pi` fails at the npm batch on
this machine — `@devopstales/skill-pi` returns 404 from the npm registry
(unpublished; this machine had it only via `npm link` from the now-deleted
`git/ai-test/skill-pi` source). This is the pre-rename state; the
skill-pi→omp rename (user WIP) switches the key to `omp` with
`@oh-my-pi/pi-coding-agent` (installed). The omp host needs no installer wiring
for the router (native project extension, below), so the smoke proceeded with
`--agents opencode,kilo,cursor`. Environment note: non-interactive shells
resolve a Nix-managed npm whose global prefix is read-only —
`NPM_CONFIG_PREFIX=$HOME/.npm-global` (the de-facto global prefix) is required.

### opencode (2.0.28, vllm/qwen3.8-uncensored) — PASS

Fixes landed during this smoke (committed `f8309b89`):

1. **Marker**: dedup marker was `using-skillgrid`, which AGENTS.md itself
   contains — the bootstrap was suppressed in exactly the repo it must work
   in. Marker → case-insensitive `router bootstrap`.
2. **Daemon cwd**: `opencode run` connects to the persistent
   `opencode serve --service` daemon whose cwd is unrelated to the session's
   project (`/Users/paladm`), and `ctx.directory` is undefined in that mode —
   the project guard resolved from the daemon cwd and in-repo injection
   silently failed. New `resolveStartDir`: `ctx.directory` →
   `event.location.directory` → `session.get({sessionID}).location.directory`
   (probed live on 2.0.28) → `process.cwd()`.

- **In-repo positive (clause a)**: proxy-captured real LLM request — both
  requests of the run carry the bootstrap in the system prompt
  (system len 118391, "Skillgrid Router Bootstrap" present). Command:
  `opencode run --model vllm/qwen3.8-uncensored` from the worktree.
- **In-repo positive (clause b)**: fresh session, status prompt — transcript:

  ```
  > build · qwen3.8-uncensored
  I'll use skillgrid:using-skillgrid for this status query.
  → Read .agents/skills/using-skillgrid/SKILL.md
  ```

  (announce-style line + router-read as first action)
- **/tmp negative**: 2 captured requests, 0 with the bootstrap.
- **Subagent skip**: documented residual on the 2.0.28 fallback —
  agent-filter only (no parent lookup from the hook event); unit-pinned.

### omp (18.16, vllm/qwen3.8-uncensored) — PASS

Mechanism fix landed during this smoke (committed `8fcb3039`): omp keeps the
shipped full-content hook (blueprint), and the pi-hooks SessionStart listener
never fires on omp (npm plugins register after the session_start dispatch;
`systemPromptOptions.appendSystemPrompt` not honored — both probe-verified) →
native project extension `.omp/extensions/skillgrid-router-omp.ts`
(before_agent_start pushes the full router as a system-prompt section;
main-agent + project guards; fail-open). Repo-tracked like the pi extension
(T3); unit tests 4/4.

- **In-repo positive (clause a)**: `omp -p "Quote exactly the first two lines
  of the skillgrid bootstrap/router notice..."` from the worktree:

  ```
  Yes, I have one. The first two lines, verbatim:
  skillgrid loaded. Route through the using-skillgrid flow before ANY response (including clarifying questions).
  (Line 2 is a blank line; the block then continues with `---` and the
  `using-skillgrid` skill definition.)
  ```

  Full router content in first-turn context (the INTRO line is only present
  in the injected section).
- **/tmp negative**: same prompt from `/tmp` → `NO-ROUTER`.
- **Subagent skip (R3 re-confirm, live)**: `omp -p` prompting the main agent
  to spawn a subagent and ask it about the router:

  ```
  SUBAGENT-ANSWER: No. A literal inspection of my current context finds no
  skillgrid router bootstrap notice — the string "skillgrid" appears only as a
  filesystem path (`.skillgrid/projectwiki/`) in the projectwiki and
  architectural-decision-records skill descriptions, with no routing or
  bootstrap instruction attached.
  ```

  Main session has the router (above); the child session does not.

### kilo (7.5.16, vllm/qwen3.8-uncensored) — ENV-LIMITED (ruling)

- Plugin installed + registered by the fresh install (router step).
- `kilo run` (in-repo): `Error: System message must be at the beginning.`
- **Isolation**: the identical error occurs with the plugin array emptied
  (core, not plugin) and in `/tmp` (cwd-independent). Pre-existing kilo-core
  headless bug — T5's finding re-confirmed after the fresh install.
- Second pre-existing defect found + fixed on this machine: `kilo.jsonc`
  carried an `env` key (`{"SKILLGRID_AGENT":"kilo"}` — shell-era hook
  variable, ADR-0032-retrieved) which kilo 7.5.16 rejects
  ("Unrecognized key: env"). Removed (backup `/tmp/kilo.jsonc.pre-envfix`);
  headless then reaches the core error above.
- Verdict: live clauses untestable via `kilo run` on this machine
  (env-limited with ruling, per runbook); plugin unit level green (T5),
  registration + distribution verified live.

### cursor (cursor-agent CLI) — PASS (stream quirk noted)

- **Hook-level oracle (in-repo)**:
  `printf '{"cwd":"<worktree>"}' | ~/.cursor/plugins/local/skillgrid/hooks/cursor-session-start.sh`
  → `additional_context` (809 chars) begins:

  ```
  Skillgrid Router Bootstrap

  This project is configured with Skillgrid.

  First action: read `.agents/skills/using-skillgr…
  ```

- **In-repo positive (clause a + b)**: `cursor-agent -f -p` (router question)
  from the worktree. The CLI stream reports
  `Connection lost, reconnecting to agentn.global.api5.cursor.sh …
  RetriableError: WritableIterable is closed`, but the agent transcript
  (`~/.cursor/projects/…t-7754f121/agent-transcripts/11fbea0d-….jsonl`) shows
  the model received the bootstrap and answered verbatim (repeated across
  stream retries):

  ```
  # Skillgrid Router Bootstrap

  This project is configured with Skillgrid.
  ```

  The model's replies ("The quote above was complete…") confirm first-turn
  context. Verdict: PASS with the backend stream quirk recorded (model-level
  success; CLI exit non-zero).
- **/tmp negative**: live PASS — same router question from `/tmp` →
  transcript (`~/.cursor/projects/private-tmp/agent-transcripts/6846b412-….jsonl`):

  ```
  NO-ROUTER

  I'm checking the context for a "skillgrid router bootstrap" to quote its
  first two lines, but I only see the Mnemonic hooks and the superpowers
  using-superpowers content, so there's no such bootstrap.
  ```

  (hook oracle for /tmp = T1 oracle 1: no bootstrap content, INFO path)
- Plugin layout note: `~/.cursor/plugins/local/skillgrid/` (real dir, T1
  door-check layout) with symlinked hooks/.agents/rules/agents/plugins into
  `.worktrees/t_06bb0c13` (still present; hook edits there propagate live).

### pi — ENV-LIMITED (ruling)

No pi/skill-pi runtime on this machine: `~/.npm-global/lib/node_modules/@devopstales/skill-pi`
is a dangling link to the deleted source repo `git/ai-test/skill-pi`, and the
package is 404 on the registry (see install ruling). The T3 pi extension +
tests are green at unit level (4/4 node); live clauses untestable without the
runtime. Consistent with the runbook ruling (2026-11-19).

### L3 gates (worktree `8fcb3039` + gofmt commit)

| Gate | Command | Result |
|---|---|---|
| Go build | `go build ./...` (skillgrid-cli module) | exit 0 |
| Go test (full) | `go test ./... -count=1` (skillgrid-cli) | exit 0 (299s) |
| Coverage (touched) | `go test -cover ./internal/install/ ./internal/setup/` | install 66.9% / setup 73.4% (ok) |
| gofmt | `gofmt -l skillgrid-cli/` | clean after formatting `cmd/skillgrid/main.go` (pre-existing misalignment from bb252ae2, outside T7 footprint — fixed to satisfy the gate) |
| opencode plugin tests | `node --experimental-strip-types plugins/opencode/test/skillgrid-router.test.ts` | 14/14 |
| omp extension tests | `node --experimental-strip-types .omp/extensions/skillgrid-router-omp.test.ts` | 4/4 |
| hook contract tests | `node --test hooks/session-start.test.mjs` | 4/4 |

### Commits (this ticket)

- `09350a03` fix(hooks): pi-hooks v0.0.2 output contract for session-start hook
- `f8309b89` fix(opencode): session-aware start dir + bootstrap marker
- `0c542a07` fix(install): skip missing native plugin and kilo logo sources
- `8fcb3039` feat(omp): full-router project extension
- (gofmt commit for `cmd/skillgrid/main.go` — see gates)

### SATISFIES (TICKET-07 acceptance mapping)

- clause (a) positive in-repo: opencode PASS (captured request), omp PASS
  (verbatim quote), cursor PASS (transcript), kilo ENV-LIMITED (ruling),
  pi ENV-LIMITED (ruling)
- clause (b) announce/router-read turn 1: opencode PASS (transcript above);
  omp + cursor positive carries the router in first-turn context (quote
  behavior); kilo/pi env-limited
- clause (a) negative in /tmp: opencode PASS (0/2), omp PASS (NO-ROUTER),
  cursor PASS (NO-ROUTER), kilo env-limited (same core error in /tmp),
  pi env-limited
- subagent-skip observable recorded: omp live (SUBAGENT-ANSWER above);
  opencode unit-pinned residual; omp extension unit 4/4
- L3 gates: green (table above)
- evidence committed: this section + gate re-run

### Concerns

1. kilo headless core bug ("System message must be at the beginning") blocks
   all `kilo run` sessions on this machine — pre-existing, plugin-independent;
   candidate for a separate kilo-core issue (ADR-0067 plugin-set rework scope).
2. cursor backend stream quirk: model answers are delivered but the CLI
   stream closes with RetriableError — exit code non-zero despite success;
   transcript is the ground truth.
3. The omp extension is project-tracked (per-checkout); non-repo skillgrid
   projects get no omp router (same property as the T3 pi extension).
