# Blueprint — skillgrid host plugins (auto-load the router)

Change: `2026-11-17-skillgrid-host-plugins` · rigor: full (5 hosts, installer changes)

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

- **OQ1 (user decision)**: distribution v1 = installer only (local machine), or also
  publish marketplaces (Cursor `.cursor-plugin/marketplace.json` exists; Kilo
  marketplace; omp marketplace)? Blueprint assumes installer-only v1.
- **OQ2**: omp plugin as full npm/git bundle (skills/ + extension, `omp plugin
  install`) vs extension-file-only (this blueprint). Extension-only is less code;
  full bundle is the "formal plugin" shape. [Inference: extension-only for v1.]
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
