# Skillgrid host plugins — Implementation Blueprint (steps, files, verification)

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:subagent-execution (recommended) or skillgrid:simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

Change: `2026-11-17-skillgrid-host-plugins`

**Status:** PROPOSED

**Tier:** T2 (config default; 5-host fan-out + installer change → full blueprint)

**Build shape:** Tracer thread — W1 (bootstrap asset + cursor hook) is the thinnest end-to-end path through the whole pattern (host-native mechanism fires → shared asset → first-turn context) before the remaining four hosts are stacked on; host-API integration risk is the unknown.

**Classification:** risky (L3 — full suite + build + coverage + lint; new trust boundary: a shell script executed by an external process, cwd taken from stdin JSON). Per `_shared/rules/verification-ladder.md`.

**Goal:** Deterministically auto-load the skillgrid router in every new session of the five agent hosts (omp, pi, opencode, Kilo Code, Cursor) — a shared ~15-line bootstrap asset injected at session start by each host's native mechanism; pointer bootstrap, full router loads on demand, zero Go runtime code.

**Architecture:** One shared bootstrap asset, read live at session start (resolution order: `~/.skillgrid/router-bootstrap.md` → repo clone → static fallback), injected by five thin per-host injectors (cursor sessionStart hook; TS plugins for pi / opencode V1+V2 / kilo / omp) — each a loader, no host-specific logic beyond load + inject. Project-conditional on `.skillgrid/config.yaml`, main agent only, fail-open (R4). The installer gains an `omp` agent key + bootstrap-distribution step; `sync-repo` refreshes the asset.

**Tech Stack:** Go 1.22+ (installer only — no new dependencies, no ADR needed per AGENTS.md § Rules); TypeScript (four host injectors, following existing `plugins/**` conventions); Bash (cursor hook); host-native plugin/extension APIs.

**Spec:** `briefing.md` (same directory) — requirements R1–R6. No ADR manifest for this change (no in-force ADRs modified; no new dependencies). Research: `/tmp/skillgrid-plugin-research/{superpowers,omp-local,pi-opencode,ide-rules}.md` (transient session artifacts, 2026-11-17; mechanism citations live in §2).

**Findings:** none in the change folder (research was session-local; see the Spec line).

## Terms

None beyond `briefing.md` — "bootstrap asset" and "injector" are defined inline in §1/§2; no glossary additions.

## Decision notes (ratified at the blueprint-validation gate, 2026-11-18)

- **OQ1 — distribution v1 = installer only** (local machine). No marketplace publishing (Cursor / Kilo / omp); marketplace shape deferred (see §6).
- **OQ2 — omp plugin = extension-file-only** (`plugins/omp/skillgrid-router.ts` → `~/.omp/agent/extensions/`), not a full npm/git plugin bundle. Full-bundle shape deferred (see §6).
- Ratified by user `execute` at the shootdown gate (state.yaml journal P26-11-18-skillplugins-exec).

## Hypothesis

**Claim:** When this blueprint is complete, a fresh session of each of the five hosts in a skillgrid project carries the router bootstrap in its first-turn system context and loads the full router on the first task — from one shared asset, via five thin native injectors, distributed by the existing installer.
**Right condition:** 5-host live smoke passes: each host's fresh session shows the bootstrap and the router read before first action ("Using [skill] to [purpose]" / start gates); non-skillgrid dirs show nothing; `go test ./...` + installer tests pass.
**Wrong condition:** any host's fresh session lacks the bootstrap (mechanism mismatch), injection fires in a non-skillgrid dir, or an injector failure blocks session start.
**Thinnest MVP:** bootstrap asset + the cursor sessionStart hook on this machine — the first real end-to-end proof of the pattern.
**Door check:** W1 (bootstrap asset + cursor hook). If the cursor fresh session lacks the bootstrap or the hook breaks the session, stop — re-verify the cursor hook mechanism against the docs before building the other four hosts.

## Must-Haves (goal-backward verification)

**Truths** (observable behaviors that must hold):
- **(R1)** `router-bootstrap.md` exists at `.agents/skills/_shared/agent-config/` with: the skillgrid-project notice, the first-action line (read `.agents/skills/using-skillgrid/SKILL.md`), the one-line request-class list, the announce line, and the `{load_hint}` variants table — no project-specific content.
- **(R2/R3)** **pi**: a fresh pi session in a skillgrid repo carries the bootstrap in the first-turn system context; a non-skillgrid dir does not. `backstop` (live host)
- **(R2/R3)** **opencode**: a fresh opencode session in a skillgrid repo carries the bootstrap as the `<context key="skillgrid-router">` entry (V2); a non-skillgrid dir does not. `backstop`
- **(R2/R3)** **kilo**: a fresh kilo session in a skillgrid repo carries the bootstrap injected into the first user message; a non-skillgrid dir does not; the `config` hook adds the repo `.agents/skills` dir to `skills.paths`. `backstop`
- **(R2/R3)** **omp**: a fresh omp session (after the full restart the installer prints) in a skillgrid repo carries the bootstrap in the system prompt; a non-skillgrid dir does not. `backstop`
- **(R2/R3)** **cursor**: a fresh cursor session in a skillgrid repo receives the bootstrap as `additional_context` from the sessionStart hook, even with the mnemonic server down; a non-skillgrid cwd receives nothing. `backstop`
- **(R3)** subagent/child sessions on any host receive no injection (guard fires). `backstop` (observable on at least one host in smoke)
- **(R4)** fail-open: an injector with a missing asset file uses the static fallback string; a bad cwd / missing binary exits 0 with no user-visible error and the session starts. `backstop`
- **(R5)** `skillgrid install --agents omp` (dry-run) copies the omp extension and writes `~/.skillgrid/router-bootstrap.md`; every one of the five host keys distributes its injector files + the bootstrap; an unknown key errors. Go-tested (diff-settleable).
- **(R5)** `sync-repo` refreshes `~/.skillgrid/router-bootstrap.md` from the repo clone (edit the asset in the clone → re-sync → file changes). Go-tested (diff-settleable).
- **(R6)** the bootstrap carries the per-host tool-mapping hint referencing `using-skillgrid/references/{pi,opencode,…}-tools.md` where those files exist.

**Artifacts** (files that must exist with real implementation, not stubs):
- `.agents/skills/_shared/agent-config/router-bootstrap.md` — the shared bootstrap asset + `{load_hint}` variants table
- `hooks/cursor-session-start.sh` — bootstrap prepend before the mnemonic-server block (cwd already parsed from stdin JSON)
- `plugins/kilo/skillgrid-router.ts` — kilo TS plugin (mirrors the local `superpowers.js` pattern)
- `plugins/opencode/skillgrid-router.ts` + `plugins/opencode/package.json` registration — opencode V2 `Plugin.define` + V1 named export
- `plugins/pi/mnemonic/skillgrid-router.ts` (+ `index.ts` wiring) — pi armed flag + system-prompt section
- `plugins/omp/skillgrid-router.ts` — omp extension factory
- `skillgrid-cli/internal/install/agents.go` (+ `config.go`, `*_test.go`) — `omp` key, bootstrap-distribution step, `sync-repo` refresh, dry-run/verbose parity, per-agent distribution tests

**Key links** (critical connections between artifacts that must work together):
- All five injectors resolve the same live asset in the same order (`~/.skillgrid/router-bootstrap.md` → repo clone → static fallback) — a one-line edit to the asset must reach all five hosts without reinstall.
- The installer's per-agent step distributes exactly the files that host loads (hook text edit, plugin copies, the bootstrap file) — dry-run assertions per host pin that set.
- The cursor hook prepends the bootstrap **before** the mnemonic-server block — injection holds even with the server down.
- The project-conditional predicate (`.skillgrid/config.yaml` at or above cwd) is identical in all five hosts — one drift breaks R3 for one host.

**One-way-door decisions:** None — every artifact is a new file or an additive installer key; uninstall = delete the file; no migrations, no public contract changes.

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| **Shell command** (cursor hook: bash script executed by the host as a subprocess, cwd from stdin JSON) | Applicable | fail-open hook — parse cwd from stdin JSON, resolve the asset, emit to stdout; exit 0 on every error path | `printf '{"cwd":"<skillgrid repo>"}' \| hooks/cursor-session-start.sh` → stdout contains the first-action line; `printf '{"cwd":"/tmp"}' \| …` → empty stdout, exit 0; `printf '{"cwd":"/tmp/a\"b"}' \| …` → exit 0 (no bash syntax error on a metacharacter cwd) |
| **Executable-file classification** (the `.sh` hook must stay executable through the installer copy; `.ts`/`.md`/`.json` distribute as data) | Applicable | installer sets the exec bit on `hooks/cursor-session-start.sh`; every other distributed file keeps its default mode | Go test: dry-run cursor install → `cursor-session-start.sh` mode has 0111; `.ts`/`.md` modes do not |
| **Process integration** (five host plugin/extension APIs: cursor hook, pi extension, opencode V1+V2, kilo plugin, omp extension) | Applicable | thin loader per host of the shared asset; no host-specific logic beyond load + inject | five live-smoke scenarios (one per host): fresh session in a skillgrid repo shows the bootstrap in the first-turn context; a non-skillgrid dir shows nothing |
| Routing (Go HTTP) | N/A: no Go handler touched — the only Go change is installer file-writing logic | — | — |
| Version-control automation | N/A: no git operations in the hook or installer (project detection = `.skillgrid/config.yaml` existence) | — | — |
| PR automation | N/A: no PR commands composed | — | — |
| Mnemonic tool contracts (`mem_*`/`code_*`/`web_cache_*`) | N/A: no new tool, param, return shape, or error code | — | — |
| Shared-convention drift (`_shared/*.md` edits) | N/A: `block.md` is not edited — the router-bootstrap extends its "per-target files decide" rule by convention, documented in §1 | — | — |

## Global Constraints

- Go 1.22+ floor (per AGENTS.md § Rules, ADR-0031); **no new Go dependency without an ADR** (per AGENTS.md § Rules) — this change asserts none needed.
- TS injectors add no new third-party packages — follow existing `plugins/**` package.json patterns.
- Spec-zone commits before code-zone commits; serial development, one change at a time (per AGENTS.md § Rules).
- Fail-open is a requirement, not a preference (R4): no injector may block session start.

## 1. Design core

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
  "announce `Using [skill] to [purpose]`" → per-host `{load_hint}` placeholder
  (e.g. omp/pi: `/skill:using-skillgrid`; opencode: `skill` tool / `@using-skillgrid`;
  kilo: read the path; cursor: `/using-skillgrid` or read);
- read **live** at session start by every injector (superpowers' runtime-read
  pattern — updates propagate without reinstall); resolution order:
  `~/.skillgrid/router-bootstrap.md` (installer-managed) → repo clone
  `~/.skillgrid/repos/skillgrid/.agents/skills/_shared/…` → static fallback string
  compiled into each injector (R4).
- project-conditional: injectors fire only when `.skillgrid/config.yaml` exists at
  or above cwd (R3); main agent only — skip subagent/child sessions
  (R3, mirrors `<SUBAGENT-STOP>` and superpowers' `parentID` skip).

## 2. Per-host mechanism (native strongest, from research)

| Host | Mechanism | Where it lives today | Change |
|---|---|---|---|
| **Cursor** | plugin `sessionStart` hook → `additional_context` (exactly superpowers' Cursor path) | `hooks/cursor-session-start.sh` + `hooks/hooks-cursor.json` + `.cursor-plugin/plugin.json` (already ships `skills: ./.agents/skills/**/SKILL.md`, hooks, rules, mcp) | Extend `cursor-session-start.sh`: prepend bootstrap to `additional_context` **before** the mnemonic-server block so it injects even with the server down; resolve cwd from stdin JSON (already parsed) |
| **Kilo** | TS plugin, auto-registered, no config entry | `~/.config/kilo/plugin/` (global, installer-managed); proven pattern on this machine: `superpowers.js` (config hook → `skills.paths`; `messages.transform` → first-user-message injection with present-guard) | New `plugins/kilo/skillgrid-router.ts` mirroring `superpowers.js`: `config` hook pushes the repo `.agents/skills` dir into `config.skills.paths` (belt for the compat dir); `experimental.chat.messages.transform` injects bootstrap into first user message (guard: skip if already present, skip subagent) |
| **OpenCode (V2)** | V2 plugin `session.created` → `ctx.session.instructions.entry.put({sessionID, key:"skillgrid-router", value})` (durable `<context key>` block; repo precedent: `plan.ts:75` synthetic-on-session.created) | `plugins/opencode/` (5 native TS plugins, dual V1+V2 export; installed to `~/.config/opencode/plugins/`) | New module `plugins/opencode/skillgrid-router.ts`: V2 `Plugin.define({id:"skillgrid.router"})` → `ctx.event.subscribe({types:["session.created"]})` → `instructions.entry.put` (main sessions only); V1 named export for 1.x (`event: session.created` + `client.session.prompt`); register in `plugins/opencode/package.json` |
| **Pi** | extension `session_start` + `before_agent_start` (returns persistent `message` and/or systemPrompt append) | `plugins/pi/mnemonic/index.ts` (already injects the Mnemonic Memory Protocol section into the system prompt — one injection seam) | New module `plugins/pi/mnemonic/skillgrid-router.ts` (or extend `index.ts`): `pi.on("session_start")` sets the armed flag (main agent, skillgrid project); the existing system-prompt injection appends the bootstrap section. Prefer extending the existing seam — one injection point, no per-request user message (lighter than superpowers' pi re-inject-every-request) |
| **omp** | TS extension `before_agent_start` system-prompt override (or `session_start` + `sendUserMessage`); discovery `~/.omp/agent/extensions/` | no omp target today; baseline already works via the AGENTS.md entry line (verified in live session JSONL 2026-11-17) | New `plugins/omp/skillgrid-router.ts` (extension factory, main-agent guard, project-conditional); installer copies it to `~/.omp/agent/extensions/` (global, like the other hosts); **known cost: omp extensions need a full restart after install, not `/reload-plugins`** — installer prints that. AGENTS.md entry line stays as the no-code fallback |

Skill delivery needs **no new copy step**: all five hosts auto-discover
`.agents/skills/` (project) and `~/.agents/skills/` (user mirror the installer
already manages). `disable-model-invocation` is honored by omp/pi/opencode/cursor
(router hidden, loadable by ID — which is why the bootstrap, not the skill list,
carries the trigger); kilo likely ignores it (router also appears in its list —
acceptable).

## 3. Installer changes (`skillgrid-cli/internal/install/`)

- **R5**: `--agents` gains `omp` (keys already: `opencode,kilo,cursor,skill-pi,hermes`).
- New step in the per-agent pipeline (alongside `installAgentConfig` /
  `setup.RunSetup`): copy the bootstrap asset → `~/.skillgrid/router-bootstrap.md`
  and wire the host injector (hook extension is a text edit to the copied hook;
  plugin files are copied next to the existing mnemonic plugins). `sync-repo`
  refreshes the bootstrap (repo clone → asset). `--dry-run`/`--verbose` parity with
  existing steps.
- No new Go dependencies (no ADR needed).

## 4. Files touched (planned)

| File | Change |
|---|---|
| `.agents/skills/_shared/agent-config/router-bootstrap.md` | **new** — the bootstrap asset + `{load_hint}` variants table |
| `hooks/cursor-session-start.sh` | prepend bootstrap to `additional_context` (pre-server block) |
| `plugins/kilo/skillgrid-router.ts` | **new** — mirror `~/.config/kilo/plugin/superpowers.js` pattern |
| `plugins/opencode/skillgrid-router.ts` | **new** — V2 `Plugin.define` + V1 named export |
| `plugins/opencode/package.json` | register the new module |
| `plugins/pi/mnemonic/skillgrid-router.ts` (+ `index.ts` wiring) | **new** — armed flag + system-prompt section |
| `plugins/omp/skillgrid-router.ts` | **new** — omp extension factory |
| `skillgrid-cli/internal/install/agents.go` (+ `config.go`) | `omp` agent key; bootstrap-distribution step; `sync-repo` refresh |
| `skillgrid-cli/internal/install/*_test.go` | per-agent distribution tests (dry-run assertions, matching existing test style) |
| `.agents/skills/using-skillgrid/SKILL.md` | no change (v1.2 entry already final) — bootstrap is a pointer to it |
| `AGENTS.md` / `block.md` | no change (entry line already shipped 2026-11-17) |

## 5. Test plan (per host, acceptance R1–R4)

1. **Unit (Go)**: installer dry-run per agent → expected files present
   (`~/.skillgrid/router-bootstrap.md`, extension/plugin copies); `omp` key
   accepted; bootstrap refresh on sync.
2. **Injector tests (TS)**: each injector module — bootstrap present in output for a
   skillgrid project cwd; absent for a non-skillgrid cwd; absent for subagent/child
   sessions; static fallback when the asset file is missing. (Node test seam per
   existing plugin test conventions, e.g. `plugins/opencode/test/`.)
3. **Live smoke (this machine, one per host)**: fresh session in this repo →
   transcript shows the router read before first action ("Using [skill] to
   [purpose]" or start gates); fresh session in a non-skillgrid dir → no injection.
   omp: full restart after install.

## 6. Risks / open questions

- **OQ1 — ratified: installer-only v1** (see Decision notes). Marketplace
  publishing (Cursor `.cursor-plugin/marketplace.json` exists; Kilo marketplace;
  omp marketplace) is deferred.
- **OQ2 — ratified: extension-file-only v1** (see Decision notes). The full
  npm/git bundle (skills/ + extension, `omp plugin install`) is the "formal
  plugin" shape, deferred.
- Kilo `disable-model-invocation` handling undocumented → router may self-list in
  kilo; if kilo ever honors it and ignores the bootstrap, the `skills.paths`
  `config` hook + `/using-skillgrid` slash command remain.
- OpenCode V1 users: dual export required (existing plugins already dual-export —
  follow the pattern).
- Cursor hook only runs when the cursor plugin is installed (marketplace or
  `~/.cursor/plugins/local/`); the AGENTS.md entry line covers non-plugin installs.
- Context cost: ~15 lines/session — vs superpowers' full ~4–8KB; strictly lighter.

## 7. Slicing preview (waves)

- **W1**: bootstrap asset + cursor hook (smallest end-to-end proof, host with
  existing hook seam).
- **W2** (parallel): kilo plugin · opencode module · pi module · omp extension
  (independent files; shared contract = bootstrap resolution order).
- **W3**: installer `omp` key + bootstrap distribution + tests + live smoke all 5.
- **W4**: docs (per-host install lines in repo docs/README; onboarding reference
  note), ship.
