# Skillgrid host plugins — Implementation Blueprint (steps, files, verification)

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:subagent-execution (recommended) or skillgrid:simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

Change: `2026-11-17-skillgrid-host-plugins`

**Status:** PROPOSED

**Tier:** T2 (config default; 5-host fan-out + installer change → full blueprint)

**Build shape:** Tracer thread: W1 (bootstrap asset + cursor hook) is the thinnest end-to-end path through the whole pattern (host-native mechanism fires → shared asset → first-turn context) before the remaining hosts are stacked on. Host-API integration risk is the unknown.

**Classification:** risky (L3: full suite + build + coverage + lint; new trust boundary: a shell script + a node script executed by external host processes, cwd from stdin JSON / `process.cwd()`). Per `_shared/rules/verification-ladder.md`.

**Goal:** Deterministically auto-load the skillgrid router in every new session of the five agent hosts (omp, pi, opencode, Kilo Code, Cursor). A shared ~15-line bootstrap asset is injected at session start by each host's native mechanism; the pi-based launcher host (omp) keeps the already-shipped full-content `session-start.js` SessionStart hook. Pointer bootstrap, full router loads on demand, zero Go runtime code.

**Architecture:** One shared bootstrap asset, read live at session start (resolution order: `~/.skillgrid/router-bootstrap.md` → repo clone → static fallback), injected by:

- four thin **pointer injectors** — one small loader per host, no host-specific logic beyond load + inject: the pi project extension (`.pi/extensions/`), TS plugins for opencode (V1+V2) and kilo, and the cursor `sessionStart` hook; plus
- the shipped `hooks/session-start.js` **full-content** SessionStart hook for the omp host (the pi-based launcher), wired by the existing settings hooks bridge via `@hsingjui/pi-hooks`.

Project-conditional on `.skillgrid/config.yaml`, main agent only, fail-open (R4). The installer gains a host-agnostic bootstrap-distribution step + router-file copies for opencode/kilo. The `omp` agent key arrives via the in-progress `skill-pi`→`omp` rename (prerequisite, see Task 6). `sync-repo` refreshes the asset.

**Tech Stack:** Go 1.22+ (installer only — no new dependencies, no ADR needed per AGENTS.md § Rules); TypeScript (three host injectors: the pi project extension `.pi/extensions/skillgrid-router.ts` + the `plugins/opencode/` and `plugins/kilo/` modules — 1:1 with the pointer-injector rows in the Per-host mechanism table); Bash (cursor hook); JavaScript (`session-start.js` cwd guard); host-native plugin/extension APIs.

**Spec:** `briefing.md` (same directory) — requirements R1–R6. No ADR manifest for this change (no in-force ADRs modified; no new dependencies). Research: `/tmp/skillgrid-plugin-research/{superpowers,omp-local,pi-opencode,ide-rules}.md` (transient session artifacts, 2026-11-17; mechanism citations live in the Per-host mechanism table).

**Findings:** stage 5 zoom-out round (2026-11-18): 5 findings (1 Blocker — repo moved under the plan: d5148778 deleted `plugins/opencode/`+`plugins/kilo/`, `skill-pi`→`omp` rename in the working tree; 2 Major — the shipped `session-start.js` seam not evaluated, pi/omp row split conflated one host; 1 Minor — fail-open not anchored to the in-repo precedent; 1 Nit — context cost vs the shipped full-content hook unweighed) — resolved and folded in; stage 7 critic round (2026-11-18): 11 findings (3 Major, 5 Minor, 3 Nit) — resolved and folded in; stage 8 intermediate zoom-out (post-revision, 2026-11-18): 5 findings (2 Major — Task 6 bootstrap-distribution step unpinned to code seams, Task 4 tree re-creation unreconciled with the stale installer contract; 3 Minor — Kilo pointer deviation, cursor R3 guard, threat-matrix exec-bit row) — resolved and folded in; see `review/zoom_out_report.md`, `review/critic_report.md`, `review/intermediate_zoom_out_report.md` + `review-ledger.md`.

## Terms

This document's own vocabulary: "bootstrap asset" and "injector" are defined inline in the Design contract and Per-host mechanism sections (forward-referenced from the header block). No glossary additions.

SDD / process vocabulary — used, not redefined here (loaded by the executing skill from the cited sources): **Must-Haves (goal-backward verification)** method + the **`backstop`** truth tag (the writing-blueprints skill's Must-Haves section); **Tier T2** / **L3** rigor (`_shared/planning/rigor-tiers.md`); **RED** = failing-first test named in the Threat Matrix (`_shared/references/strict-tdd.md`); **tracer thread** = build shape, thinnest end-to-end slice first (this header's Build shape line).

## Decision notes (ratified at the blueprint-validation gate, 2026-11-18)

- **OQ1 — distribution v1 = installer only** (local machine). No marketplace publishing (Cursor / Kilo / omp); marketplace shape deferred, not decided in this change.
- **OQ2 — SUPERSEDED (stage 5, 2026-11-18):** "omp plugin = extension-file-only" is dropped — the omp host gets router injection via the already-shipped `session-start.js` SessionStart hook (full-content, wired by `installOmpHookBridge` + `@hsingjui/pi-hooks` since 2026-09-24). No new omp extension (stage 5 finding F-S5-2: the proposed `plugins/omp/skillgrid-router.ts` was redundant with the shipped seam).
- **D3 (stage 5) — plain pi = project-level extension:** plain pi (`~/.pi/agent`, no installer key) is covered by a committed project extension `.pi/extensions/skillgrid-router.ts` (auto-discovered on project trust; research pi-opencode.md lines 87-88, 192). No `pi` installer key in v1.
- Ratified by user `execute` at the shootdown gate (state.yaml journal P26-11-18-skillplugins-exec); D3 + the OQ2 supersession recorded at stage 5 and re-presented at the slice gate.

## Hypothesis

**Claim:** When this blueprint is complete, a fresh session of each of the five hosts in a skillgrid project satisfies two clauses. Clause (a): the first-turn system context carries the router (bootstrap on the four pointer hosts, full content on omp). Clause (b): the full router loads before the first task action. Both clauses are asserted separately in the Task 7 smoke. The delivery path is one shared asset, four thin pointer injectors + the shipped full-content omp hook, distributed by the existing installer.
**Right condition:** 5-host live smoke passes, asserting both clauses per host: (a) first-turn context check (transcript/JSONL) shows the bootstrap or full router present in a skillgrid repo cwd and absent in a non-skillgrid cwd; (b) first assistant turn announces `Using skillgrid:using-skillgrid to …` or makes its first tool call a read of the router. `go test ./...` + build + coverage + lint (the L3 gates, Global Constraints) pass.
**Wrong condition:** any host's fresh session lacks the bootstrap (mechanism mismatch), injection fires in a non-skillgrid dir, or an injector failure blocks session start.
**Thinnest MVP:** bootstrap asset + the cursor sessionStart hook on this machine — the first real end-to-end proof of the pattern.
**Door check:** W1 (bootstrap asset + cursor hook). If the cursor fresh session lacks the bootstrap or the hook breaks the session, stop and re-verify the cursor hook mechanism against the docs before building the other four hosts.

## Must-Haves (goal-backward verification)

Derived backward from `briefing.md` requirements R1–R6: the observable outcomes that MUST be true when the plan is complete; the verifier checks the result against this list, not merely that tasks exist. Three categories follow: **Truths** (observable behaviors), **Artifacts** (files that must exist with real implementation), **Key links** (connections that must work together). A truth tagged **`backstop`** cannot be confirmed by reading a diff alone: it needs the held-out test named in the Tasks section (the Task 7 live smoke): a live host, a runtime, or state the diff does not show. Untagged truths a plain diff read settles.

**Truths** (observable behaviors that must hold):
- **(R1)** `router-bootstrap.md` exists at `.agents/skills/_shared/agent-config/` with: the skillgrid-project notice, the first-action line (read `.agents/skills/using-skillgrid/SKILL.md`), the one-line request-class list, the announce line, and the `{load_hint}` variants table — no project-specific content.
- **(R2/R3)** **pi** (plain pi, `~/.pi/agent`): a fresh pi session in this trusted skillgrid repo carries the bootstrap in the first-turn system context via the committed project extension `.pi/extensions/skillgrid-router.ts`; a non-skillgrid dir (no committed `.pi/extensions/`) does not. `backstop` (live host)
- **(R2/R3)** **opencode**: a fresh opencode session in a skillgrid repo carries the bootstrap as the `<context key="skillgrid-router">` entry (V2); a non-skillgrid dir does not. `backstop`
- **(R2/R3)** **kilo**: a fresh kilo session in a skillgrid repo carries the bootstrap injected into the first user message; a non-skillgrid dir does not; the `config` hook adds the repo `.agents/skills` dir to `skills.paths`. `backstop`
- **(R2/R3)** **omp**: a fresh omp session, after `skillgrid install --agents omp` (which writes the `~/.omp/agent/settings.json` hooks block the `@hsingjui/pi-hooks` plugin loads), carries the FULL router (via `session-start.js` SessionStart hook) in a skillgrid repo cwd; a non-skillgrid cwd gets nothing (cwd guard). `backstop`
- **(R3)** **session-start.js cwd guard**: `node hooks/session-start.js` emits the router payload when launched from within a skillgrid project (`.skillgrid/config.yaml` at or above `process.cwd()`) and stays silent (INFO note at most, exit 0) elsewhere — the fail-open precedent pattern preserved. `backstop` (Go/Node testable)
- **(R2/R3)** **cursor**: a fresh cursor session in a skillgrid repo receives the bootstrap as `additional_context` from the sessionStart hook, even with the mnemonic server down; a non-skillgrid cwd receives nothing. `backstop`
- **(R3)** subagent/child sessions on any host receive no injection (guard fires). On the four pointer hosts the guard is a main-agent check in each injector (pi/opencode/kilo: explicit kind check; cursor: structural — the hook's stdin payload exposes no subagent marker, and the live check in Task 1 records whether a subagent dispatch fires the hook, adding a guard on the observed signal if it does). On the omp full-content path the shipped hook has no child-session guard today: Task 2 probes a child session, and if the SessionStart hook fires there, Task 2 adds a child-session guard (shape fixed by the probe). `backstop` (named probe host: omp, Task 2 + Task 7)
- **(R4)** fail-open: an injector with a missing asset file uses the static fallback string; a bad cwd / missing binary exits 0 with no user-visible error and the session starts (pattern: `hooks/session-start.js` — catch → INFO payload → `process.exit(0)`). `backstop`
- **(R5)** `skillgrid install --agents omp` (dry-run) writes the `settings.json` hooks block (existing behavior, asserted) and copies the bootstrap asset to `~/.skillgrid/router-bootstrap.md`; the opencode/kilo keys copy their router plugin files; every host key distributes its injector files + the bootstrap; an unknown key errors. Go-tested (diff-settleable).
- **(R5)** `sync-repo` refreshes `~/.skillgrid/router-bootstrap.md` from the repo clone (edit the asset in the clone → re-sync → file changes). Go-tested (diff-settleable).
- **(R6)** the bootstrap's `{load_hint}` variants table carries the per-host tool-mapping hint referencing the existing files `using-skillgrid/references/pi-tools.md` (pi) and `using-skillgrid/references/opencode-tools.md` (opencode); kilo, cursor and omp have no tools file, so their hint is the plain first-action load. No new reference files are created.

**Artifacts** (files that must exist with real implementation, not stubs):
- `.agents/skills/_shared/agent-config/router-bootstrap.md` — the shared bootstrap asset + `{load_hint}` variants table
- `hooks/cursor-session-start.sh` — bootstrap prepend before the mnemonic-server block (cwd already parsed from stdin JSON)
- `hooks/session-start.js` — project-conditional cwd guard (walk up from `process.cwd()` for `.skillgrid/config.yaml`); fail-open + full-content behavior unchanged
- `.pi/extensions/skillgrid-router.ts` — **new** pi project extension (committed to repo; auto-discovered on project trust)
- `plugins/opencode/skillgrid-router.ts` + `plugins/opencode/package.json` — opencode V2 `Plugin.define` + V1 named export (re-creates the minimal tree deleted by d5148778; old mnemonic family retired by ADR-0067)
- `plugins/kilo/skillgrid-router.ts` — kilo TS plugin (re-creates the minimal tree; mirrors the local `superpowers.js` pattern)
- `skillgrid-cli/internal/install/agents.go` (+ `hooktree.go`, `install.go`, `sync.go`, `*_test.go`) — router-file entries in the opencode/kilo copy set, host-agnostic bootstrap step (`syncRouterBootstrap`) wired into `Run` + `Config.SyncRepo`, `sync-repo` refresh, dry-run/verbose parity, per-agent distribution tests

**Key links** (critical connections between artifacts that must work together):
- All four pointer injectors resolve the same live asset in the same order (`~/.skillgrid/router-bootstrap.md` → repo clone → static fallback) — a one-line edit to the asset must reach all four hosts without reinstall. The omp host reads the full router from the synced `~/.skillgrid/.agents/` mirror (existing `session-start.js` behavior) — refreshed by `sync-repo`.
- The installer's per-agent step distributes exactly the files that host loads (hook text edits, plugin copies, the bootstrap file) — dry-run assertions per host pin that set.
- The cursor hook prepends the bootstrap **before** the mnemonic-server block — injection holds even with the server down.
- The project-conditional predicate (`.skillgrid/config.yaml` at or above cwd) is identical in all five hosts (four pointer injectors + the `session-start.js` guard) — one drift breaks R3 for one host.

**One-way-door decisions:** None — every artifact is a new file or an additive installer key; uninstall = delete the file; no migrations, no public contract changes.

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| **Shell command** (cursor hook: bash script executed by the host as a subprocess, cwd from stdin JSON; `session-start.js`: node subprocess spawned by `@hsingjui/pi-hooks`, cwd = `process.cwd()` of the agent process) | Applicable | fail-open hooks — cursor: parse cwd from stdin JSON; session-start.js: walk up from `process.cwd()` for `.skillgrid/config.yaml`; resolve the asset, emit to stdout; exit 0 on every error path (precedent: existing `session-start.js` catch→INFO→exit 0) | `printf '{"cwd":"<skillgrid repo>"}' \| hooks/cursor-session-start.sh` → stdout contains the first-action line; `printf '{"cwd":"/tmp"}' \| …` → empty stdout, exit 0; `printf '{"cwd":"/tmp/a\"b"}' \| …` → exit 0 (no bash syntax error on a metacharacter cwd); `cd <repo> && node hooks/session-start.js` → router payload, `cd /tmp && node hooks/session-start.js` → silent/exit 0 |
| **Executable-file classification** (the `.sh` hook must stay executable through the installer copy; `.ts`/`.md`/`.json` distribute as data) | Applicable | the `cursor-session-start.sh` exec bit is maintained by the existing `installCursorHookScripts` (setup/cursor.go:130-133: `.sh` → 0755, else 0644) — pre-existing behavior, asserted unchanged; Task 6's new copies (`.ts`/`.md`) are data files via `copyFile` (hooktree.go, 0644) — no new `.sh` goes through that generic path | Go test: dry-run cursor install → `cursor-session-start.sh` mode has 0111 (pre-existing behavior preserved); the new router copies are 0644 |
| **Process integration** (four host plugin/extension APIs: pi project extension, opencode V1+V2, kilo plugin, cursor hook — plus the shipped `@hsingjui/pi-hooks` bridge for omp) | Applicable | thin loader per host of the shared asset; no host-specific logic beyond load + inject; omp rides the shipped seam (no new code beyond the cwd guard) | five live-smoke scenarios (one per host): fresh session in a skillgrid repo shows the bootstrap (or full router on omp) in the first-turn context; a non-skillgrid dir shows nothing |
| Routing (Go HTTP) | N/A: no Go handler touched — the only Go change is installer file-writing logic | — | — |
| Version-control automation | N/A: no git operations in the hook or installer (project detection = `.skillgrid/config.yaml` existence) | — | — |
| PR automation | N/A: no PR commands composed | — | — |
| Mnemonic tool contracts (`mem_*`/`code_*`/`web_cache_*`) | N/A: no new tool, param, return shape, or error code | — | — |
| Shared-convention drift (`_shared/*.md` edits) | N/A: `block.md` is not edited — the router-bootstrap extends its "per-target files decide" rule by convention, documented in the Design contract | — | — |

## Global Constraints

- Go 1.22+ floor (per AGENTS.md § Rules, ADR-0031); **no new Go dependency without an ADR** (per AGENTS.md § Rules) — this change asserts none needed.
- TS injectors add no new third-party packages — follow existing `plugins/**` package.json patterns.
- Spec-zone commits before code-zone commits; serial development, one change at a time (per AGENTS.md § Rules).
- Fail-open is a requirement, not a preference (R4): no injector may block session start.
- **L3 verification gates (operationalized):** the `risky` classification (header) means every task's final step runs, and the suite must stay green: `go build ./...`, `go test ./...` (full suite), coverage, and `gofmt -l .` (no diffs) per AGENTS.md. For the non-Go artifacts: the cursor hook gets `bash -n hooks/cursor-session-start.sh`; `session-start.js` gets `node --check hooks/session-start.js`; the TS injectors run through the existing Node test seam (the `plugins/opencode/test/` pattern, Task 4). No TS linter is configured in this repo (AGENTS.md Lint: `<detect>`), so the Node test run is the TS gate. A task that cannot run these gates is not done.

## Design contract (shared by all injectors)

**Pointer bootstrap, not full-content injection.** Superpowers injects the full
`using-superpowers` SKILL.md (small). Skillgrid's router is ~14KB (v1.2) — too large
to inline per session. Instead: one shared **bootstrap asset** (~15 lines) injected
at session start; each host's native skill system already auto-discovers
`.agents/skills/` (verified for all five hosts), so the full router loads on demand.

Bootstrap contract (R1, R3, R4):
- lives at `_shared/agent-config/router-bootstrap.md` (next to `block.md`, whose
  "per-target files decide which file, never drift" rule extends to it);
- content: skillgrid-project notice → **first action: read
  `.agents/skills/using-skillgrid/SKILL.md`** (or the host's skill-load command) →
  one-line request classes (start / continue / status / review-only / bounded) →
  "announce `Using [skill] to [purpose]`" → per-host `{load_hint}` entry
  (pi: `/skill:using-skillgrid` + reference `using-skillgrid/references/pi-tools.md`;
  omp: `/skill:using-skillgrid` (same pi command surface); opencode: `skill` tool /
  `@using-skillgrid` + reference `using-skillgrid/references/opencode-tools.md`;
  kilo: read the path; cursor: `/using-skillgrid` or read). R6: only pi and
  opencode have a tools file; kilo/cursor/omp get the plain first-action load.
- read **live** at session start by every injector (superpowers' runtime-read
  pattern — updates propagate without reinstall); resolution order:
  `~/.skillgrid/router-bootstrap.md` (installer-managed) → repo clone
  `~/.skillgrid/repos/skillgrid/.agents/skills/_shared/…` → static fallback string
  compiled into each injector (R4).
- project-conditional: injectors fire only when `.skillgrid/config.yaml` exists at
  or above cwd (R3); main agent only — skip subagent/child sessions
  (R3, mirrors `<SUBAGENT-STOP>` and superpowers' `parentID` skip).

**Full-content exception (omp host):** the pi-based launcher host keeps the
already-shipped `hooks/session-start.js` full-content SessionStart hook (the
2026-09-24 decision for hook-based harnesses — the 14KB router is inlined there).
**Evaluation record (stage 7 F-1):** what the shipped hook already guarantees —
full-content router injection on every `SessionStart`, fail-open
(catch → INFO payload → `process.exit(0)`). What it does not guarantee —
the project-conditional cwd guard (Task 2 adds it) and the child-session skip
(R3: Task 2 probes; guard added only if the probe shows child injection).
The pointer contract governs the four pointer injectors; unifying the two
injection shapes is out of scope for this change (follow-up). Context cost:
pointer injectors ~15 lines/session (vs superpowers' full ~4–8KB, strictly
lighter); the omp host keeps the shipped full-content hook (~14KB/session) —
the per-host split is deliberate (hook-based host rides the shipped seam;
extension/plugin hosts use the pointer). The stage 5 zoom-out rejected the
originally-proposed new omp TS extension as redundant with this seam (F-S5-2).

## Per-host mechanism (native strongest, from research; re-anchored 2026-11-18 post d5148778 + the `skill-pi`→`omp` rename)

| Host | Mechanism | Where it lives today | Change |
|---|---|---|---|
| **Cursor** | plugin `sessionStart` hook → `additional_context` (exactly superpowers' Cursor path) | `hooks/cursor-session-start.sh` + `hooks/hooks-cursor.json` + `.cursor-plugin/plugin.json` (already ships `skills: ./.agents/skills/**/SKILL.md`, hooks, rules, mcp) | Extend `cursor-session-start.sh`: prepend bootstrap to `additional_context` **before** the mnemonic-server block so it injects even with the server down; resolve cwd from stdin JSON (already parsed) |
| **Kilo** | TS plugin, auto-registered, no config entry | `~/.config/kilo/plugin/` (global, installer-managed); proven pattern on this machine: `superpowers.js` (config hook → `skills.paths`; `messages.transform` → first-user-message injection with present-guard) | New `plugins/kilo/skillgrid-router.ts` mirroring `superpowers.js`: `config` hook pushes the repo `.agents/skills` dir into `config.skills.paths` (belt for the compat dir); `experimental.chat.messages.transform` injects bootstrap into first user message (guard: skip if already present, skip subagent). **Tree re-created** — `plugins/kilo/` was emptied by d5148778 (old mnemonic family retired by ADR-0067) |
| **OpenCode (V2)** | V2 plugin `session.created` → `ctx.session.instructions.entry.put({sessionID, key:"skillgrid-router", value})` (durable `<context key>` block: the documented durable-instruction API, pi-opencode.md:127-128; `session.created` event timing: plan.ts:75 precedent, pi-opencode.md:119-122; live v2.0.28-verified hook surface: `ctx.session.hook("context")` — ~/.config/opencode/plugins/skillgrid-context.ts:126-165) | `~/.config/opencode/plugins/` (installer-managed copy). **Tree re-created** — `plugins/opencode/` was emptied by d5148778 | New module `plugins/opencode/skillgrid-router.ts` + minimal `package.json` (`{"type":"module"}`): V2 `Plugin.define({id:"skillgrid.router"})` → `ctx.event.subscribe({types:["session.created"]})` → `instructions.entry.put` (main sessions only); V1 named export for 1.x (`event: session.created` + `client.session.prompt`) |
| **Pi** (plain pi, `~/.pi/agent` — no installer key; in live use on this machine) | **project-level extension** `.pi/extensions/` — auto-discovered on project trust (research pi-opencode.md lines 80, 87-88, 192); `before_agent_start` systemPrompt append (persistent; no per-request user message) | none (plain pi has no skillgrid wiring; `~/.pi/agent/extensions/` holds unrelated user extensions) | **New** committed repo file `.pi/extensions/skillgrid-router.ts`: extension factory (`export default function (pi: ExtensionAPI)`), `before_agent_start` appends the bootstrap section (live asset read, main-agent guard `ctx.agent.kind === "main"` — the pi-based ExtensionAPI documents `agent (kind: "main" | "sub")`, omp-local.md:44-46, project-conditional double-check). No installer changes — the file ships with the repo |
| **omp** (pi-based launcher; installer key `omp`, renamed from `skill-pi` in the in-progress working-tree refactor) | SessionStart command hook via the settings hooks bridge: `installOmpHookBridge` writes `~/.omp/agent/settings.json` `hooks.SessionStart` → `@hsingjui/pi-hooks` npm plugin (verified loaded in `~/.omp/plugins/omp-plugins.lock.json`) executes `node ~/.skillgrid/hooks/session-start.js` | `hooks/session-start.js` (shipped 2026-09-24; full-content router injection, fail-open INFO+exit 0) — wired for the agent key but `~/.omp/agent/settings.json` not yet written on this machine (post-rename install pending) | **No new extension** (stage 5 F-S5-2: the originally-proposed `plugins/omp/skillgrid-router.ts` is redundant). Edit `session-start.js`: add the project-conditional cwd guard (walk up from `process.cwd()` for `.skillgrid/config.yaml`; silent elsewhere); keep fail-open + full-content otherwise. `skillgrid install --agents omp` (post-rename) then completes the wiring with zero new installer code for this host |

Skill delivery needs **no new copy step**: all five hosts auto-discover
`.agents/skills/` (project) and `~/.agents/skills/` (user mirror the installer
already manages). `disable-model-invocation` is honored by omp/pi/opencode/cursor
(router hidden, loadable by ID — which is why the bootstrap, not the skill list,
carries the trigger); kilo likely ignores it (router also appears in its list —
acceptable).

**Note on the old pi/omp row split:** the pre-stage-5 plan had separate "pi"
and "omp" rows that both distributed into the same `~/.omp/agent/extensions/`
dir after the rename (two rows, one host), while plain pi (`~/.pi`) had no
installer path at all. Stage 5 merged them: omp = shipped hook seam; plain pi =
committed project extension (D3). The `plugins/pi/mnemonic/` extension stays
untouched (memory concern — ADR-0043 / ADR-0067 territory, not this change).

## File Structure

- `.agents/skills/_shared/agent-config/router-bootstrap.md` — the shared bootstrap asset + `{load_hint}` variants table (Task 1)
- `hooks/cursor-session-start.sh` — cursor `sessionStart` hook; gains the bootstrap prepend before the mnemonic-server block (Task 1)
- `hooks/session-start.js` — omp `SessionStart` hook; gains the project-conditional cwd guard + the probe-driven child-session guard (Task 2)
- `hooks/session-start.test.mjs` — Node regression test for the guard + fail-open (Task 2)
- `.pi/extensions/skillgrid-router.ts` — pi project extension: `before_agent_start` systemPrompt append, live asset read, main-agent + project guards (Task 3)
- `.pi/extensions/skillgrid-router.test.ts` — Node test, stubbed `ExtensionAPI`, guard matrix (Task 3)
- `plugins/opencode/skillgrid-router.ts` — opencode V2 `Plugin.define` + V1 named export (Task 4; tree re-created — the old mnemonic family was deleted by d5148778 / ADR-0067)
- `plugins/opencode/package.json` — minimal `{"type":"module"}` ESM marker (Task 4, idempotent create)
- `plugins/opencode/test/skillgrid-router.test.ts` — Node test with the local `Plugin.define` stub (Task 4)
- `plugins/kilo/skillgrid-router.ts` — kilo TS plugin: `config` hook → `skills.paths` + `messages.transform` first-user-message injection (Task 5; tree re-created)
- `plugins/kilo/skillgrid-router.test.ts` — Node test, stubbed kilo hook surface, guard matrix (Task 5)
- `skillgrid-cli/internal/install/agents.go` (+ `config.go`) — `routerPluginRel` distribution facts + host-agnostic bootstrap step + `sync-repo` refresh, built on the post-rename API (Task 6)
- `skillgrid-cli/internal/install/install_test.go` — per-agent distribution tests (dry-run assertions, existing style) (Task 6)
- `README.md` — the `--agents` example line lists the post-rename host keys (Task 8)
- `docs/user-guide/04-hooks.md` — the `session-start.js` entry: guard + child-session behavior (Task 8)
- `docs/user-guide/11-plugins.md` — the Skillgrid router section: per-host table aligned with the revised design (pi row, omp row; Task 8)

Not touched: `.agents/skills/using-skillgrid/SKILL.md` (v1.2 entry already final — the bootstrap is a pointer to it); `AGENTS.md` / `block.md` (entry line already shipped 2026-11-17); `plugins/pi/mnemonic/` (memory concern — ADR-0043 / ADR-0067 territory, not this change).

## Tasks

Slicing (next phase) groups these into waves: W1 = Task 1 (door check); W2 =
Tasks 2–5 (parallel, independent files, shared contract = the Design
contract); W3 = Task 6 → Task 7 (Task 6 blocked on the external rename
commit); W4 = Task 8. Every task ends with the L3 gates (Global Constraints).

### Task 1: bootstrap asset + cursor hook (door check)

> **Checkpoint:** door check (Hypothesis). If the cursor fresh session lacks
> the bootstrap, or the hook breaks the session, STOP — re-verify the cursor
> hook mechanism against the docs before building the other hosts.

The cursor host is the tracer: it already ships a working `sessionStart` hook
(`hooks/cursor-session-start.sh` + `hooks/hooks-cursor.json` +
`.cursor-plugin/plugin.json`), so this task proves the whole pattern
(mechanism fires → shared asset → first-turn context) end to end. Risk
(recorded, stage 7): the cursor hook runs via the plugin (marketplace or
`~/.cursor/plugins/local/`) or the installer's user-level registration
(`~/.cursor/hooks.json` — setup/cursor.go:63-73: the user-level entries make
the hooks run "whether or not the plugin is enabled"); a machine with neither
the plugin nor the install is covered by the AGENTS.md entry line.

Risk
(recorded, stage 8 F-S8-4): the hook's stdin payload (`session_id`,
`conversation_id`, `cwd`, `workspace_roots` — cursor-session-start.sh:8-13,
20-28) exposes no subagent marker; Cursor subagents (Task tool) run inside
the parent session, so `sessionStart` fires once per session — the R3 guard
is structural on this host. Step 5's live door check records whether a
subagent dispatch fires the hook; if it does, add a guard on the observed
signal (the Task 2 probe pattern).

**Files:**
- Create: `.agents/skills/_shared/agent-config/router-bootstrap.md`
- Modify: `hooks/cursor-session-start.sh` (prepend the bootstrap before the mnemonic-server block)
- Test: the Threat-Matrix RED scenarios (Step 3) — the repo has no bash test harness; Task 7's live smoke is the held-out check

**Interfaces:**
- Consumes: the cursor `sessionStart` hook stdin JSON (`cwd` already parsed by the existing script); no earlier-task output (the asset is created in this task)
- Produces: the bootstrap asset at `.agents/skills/_shared/agent-config/router-bootstrap.md` per the Design contract — Tasks 2–6 all resolve this exact path
- Seam: the hook's stdout JSON (`additional_context`) — injection behavior is alterable without editing the host
- Deletion test: delete the asset and every host falls back to its compiled static string (R4); the loss is the live-update path, reappearing across all five injectors
- Adapters: five (the five host resolution paths) — the seam is justified

**SATISFIES:** T-asset-content, T-cursor-live

- [ ] **Step 1: Write the asset** — the Design contract content (R1 + R6): skillgrid-project notice, first-action line (read `.agents/skills/using-skillgrid/SKILL.md`), one-line request-class list (start / continue / status / review-only / bounded), announce line, and the `{load_hint}` table with all five hosts (pi + opencode entries reference the existing tools files; kilo/cursor/omp plain load). No project-specific content.
- [ ] **Step 2: Extend the hook** — in `cursor-session-start.sh`, before the mnemonic-server block: when cwd (from stdin JSON) has `.skillgrid/config.yaml` at or above it, resolve the asset (Design contract order) and put its content at the front of `additional_context`. Keep the fail-open shape: any error → emit the existing payload unchanged, exit 0.
- [ ] **Step 3: Verify the RED scenarios (Threat Matrix)**
  - `printf '{"cwd":"<this repo>"}' | hooks/cursor-session-start.sh` → stdout contains the first-action line
  - `printf '{"cwd":"/tmp"}' | hooks/cursor-session-start.sh` → no bootstrap, exit 0
  - `printf '{"cwd":"/tmp/a\"b"}' | hooks/cursor-session-start.sh` → exit 0 (no syntax error on a metacharacter cwd)
  - `bash -n hooks/cursor-session-start.sh`
- [ ] **Step 4: Asset content check (R1, R6)** — read the committed asset; verify the R1 checklist + the `pi-tools.md` / `opencode-tools.md` hints + no project-specific content.
- [ ] **Step 5: Live door check (this machine)** — fresh cursor session in this repo: first assistant turn announces `Using skillgrid:using-skillgrid to …` or first tool call reads the router; first-turn context (transcript) contains the bootstrap. Fresh cursor session in `/tmp`: no injection. Record transcript excerpts in `review/smoke-evidence.md`.
- [ ] **Step 6: L3 gates + commit** — `go build ./... && go test ./...` (no Go touched; the suite must stay green); `git add` the asset + hook; `git commit -m "feat(hooks): router bootstrap asset + cursor sessionStart injection"`.

### Task 2: `session-start.js` cwd guard + child-session probe (omp path)

The omp host rides the shipped full-content `SessionStart` hook. Today it
fires in every session of every repo: no project condition, no
child-session guard. This task adds the project condition and probes the
child-session question (R3: the named probe host). Evaluation record: see
the Design contract's full-content exception (guaranteed: full-content
injection + fail-open; not guaranteed: project condition, child-session
skip).

**Files:**
- Modify: `hooks/session-start.js`
- Create: `hooks/session-start.test.mjs` (Node test, `child_process.spawn` with `cwd:` override)

**Interfaces:**
- Consumes: the synced router file `~/.skillgrid/.agents/skills/using-skillgrid/SKILL.md` (existing behavior, unchanged); `process.cwd()` (the hook command inherits the agent process cwd)
- Produces: the guarded hook — the omp row of the Per-host mechanism table
- Seam: the hook's stdout JSON payload (same contract as before the change)
- Deletion test: pass-through — the guard is a pre-check; delete it and injection returns to unconditional (the pre-change behavior)
- Adapters: one (the omp hook) — no new seam; the guard follows the file's existing catch→INFO→exit 0 shape

**SATISFIES:** T-omp-guard, T-omp-child-probe

- [ ] **Step 1: Write the failing test** — `hooks/session-start.test.mjs`: spawn `node hooks/session-start.js` with `cwd:` = this repo → stdout contains the first-action line; `cwd:` = `/tmp` → no router payload, exit 0; router file missing (point at a temp sync dir without it) → exit 0, INFO at most.
- [ ] **Step 2: Run to verify it fails** — `node hooks/session-start.test.mjs` → the `/tmp` case fails (the hook currently emits the router payload from any cwd).
- [ ] **Step 3: Add the guard** — walk up from `process.cwd()` for `.skillgrid/config.yaml`; absent → INFO note (not the router payload), exit 0. Keep the existing try/catch fail-open shape.
- [ ] **Step 4: Run to verify it passes** — test green; `node --check hooks/session-start.js`; L3 gates.
- [ ] **Step 5: Child-session probe (investigation; record the result)** — in omp in this repo: fresh session, dispatch one subagent, inspect the child session's transcript for the router injection. If the child received the injection: add a child-session guard here (shape fixed by what the probe shows — e.g. a parent-session marker in the hook stdin payload, or a session-file path check) + a test for it, re-run Step 4. Record the probe outcome + the chosen guard (or "no guard needed") in `review/smoke-evidence.md`.
- [ ] **Step 6: Commit** — `git commit -m "feat(hooks): session-start.js project-conditional cwd guard"` (+ a second commit if the probe adds the child guard).

### Task 3: pi project extension (plain pi host)

Plain pi (`~/.pi/agent`) has no installer key (Decision D3): it is covered by
a committed project extension, auto-discovered on project trust (research
pi-opencode.md lines 80, 87-88, 192). Risk (recorded): an untrusted pi checkout
shows no injection — acceptable, same trust model as the AGENTS.md entry
line, which also only fires in a skillgrid repo.

**Files:**
- Create: `.pi/extensions/skillgrid-router.ts`
- Create: `.pi/extensions/skillgrid-router.test.ts` (Node test, stubbed `ExtensionAPI`)

**Interfaces:**
- Consumes: the bootstrap asset (Task 1) via the Design contract resolution order
- Produces: a persistent systemPrompt bootstrap section on the pi host (`before_agent_start`)
- Seam: the `ExtensionAPI` `before_agent_start` return — alterable without touching the router
- Deletion test: the pi host loses deterministic injection; the AGENTS.md entry line remains as the soft fallback
- Adapters: one (the pi host) — no port, a direct extension

**SATISFIES:** T-pi-live

- [ ] **Step 1: Write the failing test** — stub `ExtensionAPI` (capture the `before_agent_start` registration; research pi-opencode.md line 189: the systemPrompt-append point): main agent + project cwd → the returned systemPrompt contains the first-action line; subagent (`kind !== "main"`) → no append; non-project cwd → no append; missing asset file → the static fallback string is appended (R4).
- [ ] **Step 2: Run to verify it fails** — `node --experimental-strip-types .pi/extensions/skillgrid-router.test.ts` → the module does not exist.
- [ ] **Step 3: Implement** — `export default function (pi: ExtensionAPI)`: register `before_agent_start` with the main-agent guard (`ctx.agent.kind === "main"` — `agent (kind: "main" | "sub")` is documented for the pi-based ExtensionAPI surface, omp-local.md:44-46), the project-conditional check (`.skillgrid/config.yaml` at or above cwd), the live asset read (Design contract order), and the static fallback. Pin the exact API shape against the research file (it is in the reading list); the local `~/.pi/agent/extensions/gryph-hooks.ts` is a live example of the factory form.
- [ ] **Step 4: Run to verify it passes** — test green; L3 gates.
- [ ] **Step 5: Commit** — `git commit -m "feat(pi): committed project extension for deterministic router bootstrap"`.

### Task 4: opencode router module (V2 + V1 dual export)

**Context (stage 5):** `plugins/opencode/` was emptied by d5148778 (the old
mnemonic family, retired by ADR-0067). This task re-creates the tree with the
router module only. Tree creation is idempotent: if the ADR-0067 change
re-creates the tree (with its own `package.json`) first, this task creates
only the `.ts` files and joins the existing tree. Risk (recorded): OpenCode V1
users need the dual export — the new module ships dual-export from day one
(the deleted plugins' pattern).

**Reconciliation (stage 8 F-S8-2):** the live memory-plugin source is
`plugins/pi/mnemonic/` (committed; `SetupOmp` copies the folder wholesale —
setup/pi.go:12), while the opencode/kilo installer still keys its memory bases
on `plugins/opencode/<base>.ts` (`setup/opencode.go:61,83`,
`setup/kilocode.go`, `agents.go:35` `nativePluginRels`, `setup.go:172`
`nativePluginBases`) — dangling since d5148778, owned by the in-flight
ADR-0067 memory change (sliced 2026-11-18), out of scope here. This task's
`plugins/opencode/` re-creation deliberately carries the router module only:
per-harness source dirs are the live convention (precedent:
`plugins/pi/mnemonic/`, `plugins/cursor/`), and opencode's installer reads
exactly this dir (v2 auto-discovery of copied `.ts` files —
`installOpenCodeNativePlugins`, setup/opencode.go:75-95). Kilo's router ships
from `plugins/kilo/` via the standalone `routerPluginRel` (Task 6), not the
`nativePluginBases` loop — kilo's memory bases still copy from
`plugins/opencode/` as the single source (kilocode.go:11).

**Files:**
- Create: `plugins/opencode/skillgrid-router.ts`
- Create: `plugins/opencode/package.json` (minimal `{"type":"module"}`; skip if the ADR-0067 change already created one)
- Create: `plugins/opencode/test/skillgrid-router.test.ts` (Node test, the repo's Node test seam: `node --experimental-strip-types`, local `Plugin.define` stub — the pattern of the deleted `memory-prefetch.test.ts`)

**Interfaces:**
- Consumes: the bootstrap asset (Task 1) via the Design contract resolution order
- Produces: V2 — the durable `<context key="skillgrid-router">` entry via `ctx.session.instructions.entry.put({sessionID, key: "skillgrid-router", value})` on `session.created` (main sessions only; the durable-instruction API: pi-opencode.md:127-128 — plan.ts:75 is the synthetic-message precedent for the `session.created` event, not for this API); V1 — the named export for 1.x (`event: session.created` + `client.session.prompt`)
- Seam: the opencode plugin API event subscription — alterable without touching the router
- Deletion test: the opencode host loses deterministic injection; the AGENTS.md entry line remains
- Adapters: one (the opencode host) — no port, a direct plugin

**SATISFIES:** T-opencode-live

- [ ] **Step 1: Write the failing test** — stub the plugin ctx: `session.created` main session + project cwd → `instructions.entry.put` called with key `skillgrid-router` and the asset content; subagent session → not called; non-project cwd → not called; missing asset → the static fallback is put (R4).
- [ ] **Step 2: Run to verify it fails** — `node --experimental-strip-types plugins/opencode/test/skillgrid-router.test.ts` → the module does not exist.
- [ ] **Step 3: Implement** — V2 `Plugin.define({ id: "skillgrid.router" })` + `ctx.event.subscribe({ types: ["session.created"] })` + the guards (main session, project-conditional); V1 named export alongside (dual export).
- [ ] **Step 4: Run to verify it passes** — test green; L3 gates.
- [ ] **Step 5: Commit** — `git commit -m "feat(opencode): skillgrid.router plugin (V2 + V1 dual export)"`.

### Task 5: kilo router plugin

**Context:** `plugins/kilo/` was emptied by d5148778; this task re-creates it
with the router plugin only (idempotent, as Task 4). Proven pattern on this
machine: `~/.config/kilo/plugin/superpowers.js` (read it first; it is the
local proven shape). Risk (recorded, stage 7 re-scoped): kilo's
`disable-model-invocation` handling is undocumented — if honored, the router
is hidden from kilo's skill *list* (visibility only; injection is
`messages.transform` and is unaffected). The bootstrap's kilo `{load_hint}` is
a direct path read, which works either way; the `skills.paths` `config` hook +
the `/using-skillgrid` slash command are the visibility-side belt.

**Pointer deviation (stage 8 F-S8-3):** the proven `superpowers.js` pattern
injects the full router body via `messages.transform`; this task deviates to
the shared ~15-line pointer bootstrap for uniformity with the other three
pointer hosts (one shared asset — a one-line edit reaches all four, Key link)
and to cut kilo's per-session context cost from ~14KB to ~15 lines.
Full-router access on demand is preserved by the `skills.paths` `config` hook
+ the `{load_hint}` direct path read; the present-guard from `superpowers.js`
(skip if already present) is retained.

**Files:**
- Create: `plugins/kilo/skillgrid-router.ts`
- Create: `plugins/kilo/skillgrid-router.test.ts` (Node test, stubbed kilo hook surface — the `superpowers.js` hooks)

**Interfaces:**
- Consumes: the bootstrap asset (Task 1) via the Design contract resolution order
- Produces: (a) the `config` hook pushing the repo `.agents/skills` dir into `config.skills.paths`; (b) `experimental.chat.messages.transform` injecting the bootstrap into the first user message (present-guard: skip if already present; skip subagent)
- Seam: the kilo plugin hook API — alterable without touching the router
- Deletion test: the kilo host loses deterministic injection; the AGENTS.md entry line remains
- Adapters: one (the kilo host) — no port, a direct plugin

**SATISFIES:** T-kilo-live

- [ ] **Step 1: Write the failing test** — stub the hook surface: first user message + project cwd + main agent → the transformed message contains the first-action line exactly once (present-guard: a second transform is a no-op); subagent → unchanged; non-project cwd → unchanged; missing asset → static fallback (R4).
- [ ] **Step 2: Run to verify it fails** — `node --experimental-strip-types plugins/kilo/skillgrid-router.test.ts` → the module does not exist.
- [ ] **Step 3: Implement** — mirror `superpowers.js`: `config` hook → `skills.paths` entry; `messages.transform` → first-user-message injection with the guards.
- [ ] **Step 4: Run to verify it passes** — test green; L3 gates.
- [ ] **Step 5: Commit** — `git commit -m "feat(kilo): skillgrid-router plugin (config + messages.transform)"`.

### Task 6: installer distribution (router files + bootstrap step + sync refresh)

**Prerequisite (external, in-flight):** the working-tree `skill-pi`→`omp`
agent-key rename (the user's uncommitted refactor: `cmd/skillgrid/main.go`,
`install/{agents,config,hookbridge,hookbridge_test,install,install_test,lsp,lsp_test}.go`,
`setup/{pi,pi_test,setup}.go`, new untracked `install/{ompplugins,ompplugins_test}.go` —
`SetupPi`→`SetupOmp`, `~/.skill-pi/agent`→`~/.omp/agent`,
`SKILL_PI_AGENT_DIR`→`OMP_AGENT_DIR`, NPM `@devopstales/skill-pi`→
`@oh-my-pi/pi-coding-agent`) **must be committed before this task starts**.
The installer work targets the post-rename API (`SetupOmp`, `omp` key,
`installOmpHookBridge`); the `omp` key already exists post-rename (R5's
"add `omp` to `--agents`" is absorbed by the rename — no new key needed).

**Sequencing decision (stage 7 F-2):** the ADR-0067 memory change is
reworking the same trees + the `nativePluginBases` copy set. This task's
distribution fact is a standalone `routerPluginRel` constant — NOT folded
into `nativePluginBases` — so the rework cannot break this copy step. Tree
creation is idempotent (create-if-missing, Tasks 4/5); if ADR-0067 lands
first, this task's diff shrinks (no tree creation) and the delivered files
are identical. No conditional remains on the other change's internal shape.

**Files:**
- Modify: `skillgrid-cli/internal/install/agents.go` (the `routerPluginRel` facts + the per-agent copy-set entries for opencode/kilo)
- Modify: `skillgrid-cli/internal/install/hooktree.go` (new `syncRouterBootstrap(c)` next to `syncMirrorDirs`: copies the asset from `c.RepoDir` `.agents/skills/_shared/agent-config/router-bootstrap.md` → `c.RepoHome/router-bootstrap.md` via the existing `copyFile` — hooktree.go:196-213, 0644, unconditional overwrite; re-runs with unchanged content leave the file unchanged)
- Modify: `skillgrid-cli/internal/install/install.go` (`Run` calls `syncRouterBootstrap` immediately after `syncMirrorDirs` — install.go:64; host-agnostic: runs on every install run, before agent selection)
- Modify: `skillgrid-cli/internal/install/sync.go` (`Config.SyncRepo` calls the same step after the repo copy — covers the `sync-repo` verb, refresh from the synced copy)
- Modify: `skillgrid-cli/internal/install/install_test.go` (the existing per-agent dry-run test style; new router-distribution + bootstrap-step tests)

**Interfaces:**
- Consumes: the bootstrap asset path (Task 1); the router file paths (Tasks 4–5); the post-rename `SetupOmp` / `omp` key API (external prerequisite)
- Produces: the distribution surface — every host key distributes its injector files + the bootstrap; `sync-repo` refreshes the bootstrap; an unknown key errors
- Seam: the host-agnostic bootstrap step alongside `syncMirrorDirs` (every install run + `sync-repo`) + the installer's per-agent pipeline (alongside `installAgentConfig` / `setup.RunSetup`) for the router-file copies
- Deletion test: the hosts lose the managed copies; the repo-local mechanisms (pi project extension, cursor hook in the repo) still work from the checkout
- Adapters: one (the installer) — no port

**SATISFIES:** T-installer-distribution, T-sync-refresh

- [ ] **Step 1: Write the failing Go tests** (dry-run, existing style): `--agents omp` → the settings hooks block written (existing behavior, asserted) + `~/.skillgrid/router-bootstrap.md` copied; `--agents opencode` / `--agents kilo` → the router `.ts` file copied; every host key → the bootstrap copied; unknown key → error; `sync-repo` with an edited asset in the clone → the target file changes.
- [ ] **Step 2: Run to verify they fail** — `go test ./skillgrid-cli/internal/install/ -run TestRouterDistribution` (name per the new tests).
- [ ] **Step 3: Implement** — the `routerPluginRel` constant(s) + the copy-set entries; the host-agnostic bootstrap step; the `sync-repo` refresh; `--dry-run` / `--verbose` parity. No new Go dependencies.
- [ ] **Step 4: Run to verify they pass** — new tests green + full L3 gates (`go build ./...`, `go test ./...`, coverage, `gofmt -l .`).
- [ ] **Step 5: Commit** — `git commit -m "feat(install): router distribution + host-agnostic bootstrap step"`.

### Task 7: 5-host live smoke (backstop)

Held-out verification for every `backstop` truth (R2/R3/R4 across the five
hosts). Asserts **both** hypothesis clauses per host (stage 7 F-3):
(a) injection present — the first-turn context (transcript/JSONL) contains
the bootstrap (pointer hosts) or the full router (omp) in a skillgrid repo
cwd, and is absent in a non-skillgrid cwd; (b) router loaded before first
action — the first assistant turn announces `Using skillgrid:using-skillgrid
to …` or makes its first tool call a read of the router.

**Files:**
- Test: live, this machine (no code artifacts; evidence → `review/smoke-evidence.md`)

**Interfaces:**
- Consumes: Tasks 1–6 (all mechanisms installed)
- Produces: the smoke evidence the ship gate consumes
- Seam: none (verification task)
- Deletion test: n/a
- Adapters: n/a

**SATISFIES:** T-five-host-smoke

- [ ] **Step 1: Install** — `skillgrid install --agents omp,opencode,kilo,cursor` (post-rename keys); the cursor plugin installed (marketplace or `~/.cursor/plugins/local/`); pi: accept the project-trust prompt if shown.
- [ ] **Step 2: Per host (omp, pi, opencode, kilo, cursor):** fresh session in this repo → assert clause (a) positive + clause (b); fresh session in `/tmp` → assert clause (a) negative (no load, no announce, bootstrap absent from first-turn context). omp: re-confirm the child-session probe result (Task 2 Step 5) in a fresh session — it is the R3 subagent observable.
- [ ] **Step 3: Record** — transcript excerpts per host (positive + negative) in `review/smoke-evidence.md`. Any host failing → do not proceed to Task 8: fix, or park with a ruling (ledger).
- [ ] **Step 4: L3 gates** — full suite green.
- [ ] **Step 5: Commit the evidence** — `git commit -m "test: 5-host live smoke evidence (all pass / <failures>)"`.

### Task 8: docs

**Files:**
- Modify: `README.md` (the `--agents` example line, currently `skillgrid --agents opencode,kilo` — list the post-rename host keys: `opencode,kilo,cursor,omp`)
- Modify: `docs/user-guide/04-hooks.md` (the `session-start.js` entry: the project-conditional guard + the child-session behavior Task 2's probe fixed)
- Modify: `docs/user-guide/11-plugins.md` (the Skillgrid router section: its per-host table predates the revised design — the pi row says "existing system-prompt injection seam (shared with the Mnemonic memory protocol)" and the omp row says "extension under `~/.omp/agent/extensions/`"; both are stale against D3/OQ2)

**Interfaces:**
- Consumes: the final state of Tasks 1–7
- Produces: the user-facing install lines + the hook doc accuracy
- Seam: none (docs)
- Deletion test: n/a
- Adapters: n/a

**SATISFIES:** T-docs

- [ ] **Step 1: Update the README `--agents` line** to the post-rename host key set (hermes excluded: not a router host).
- [ ] **Step 2: Update `docs/user-guide/04-hooks.md`** — the `session-start.js` description: project-conditional (`.skillgrid/config.yaml` at or above cwd), fail-open, child-session behavior per the Task 2 probe result.
- [ ] **Step 3: Update `docs/user-guide/11-plugins.md`** — the Skillgrid router section's per-host table: align the pi row (committed project extension `.pi/extensions/skillgrid-router.ts`, `before_agent_start` append) and the omp row (shipped `session-start.js` full-content hook; no new extension) with the Per-host mechanism table.
- [ ] **Step 4: Verify** — read all three files back; the `--agents` keys match `install.AvailableAgents()` post-rename; the 11-plugins.md pi/omp rows match the blueprint's Per-host mechanism table.
- [ ] **Step 5: Commit** — `git commit -m "docs: router host install lines + session-start guard"`.

## Review Summary

**Verdict:** READY FOR EXECUTION — goldfish Round 1 complete (Consolidated preset, 5 stages, 2026-11-18): 47 findings (13 CRITICAL, 25 WARNING, 9 SUGGESTION) — 35 fixed, 4 partial with recorded remainder, 8 rejected with evidence; 1 persona not verified (stage-9 implementation-readiness, both passes unusable, capped per the dispatch protocol; actionability lens partially carried by the stage-7 structural critic + stage-9 codebase-details). No CRITICAL open. Resolution record: `review-ledger.md` (round entry, not this block); cross-stage dedup + traceability matrix: `review/consolidated_review_report.md`.

**Rounds run:** 1 (cap 5; converged in round 1 — no churn)

**Reviewed:** 2026-11-18 (final candidate blob b12cd7d1, 496 lines; round-1 fix commit 809adfd2)
