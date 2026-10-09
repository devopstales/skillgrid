# skill-pi — Design Briefing (requirements & intent)

> **STATUS:** `draft` (2026-10-09)

## Problem / Intent

skillgrid ships per-harness agent customization for opencode, cursor, and kilo
(plugins + skills + subagents), but the emerging Pi coding-agent ecosystem
(`@earendil-works/pi`) has no skillgrid presence. This change ships `skill-pi`:
an opinionated, non-fork distribution of Pi that runs with the skillgrid
standards, skills, memory (mnemonic MCP), code index, events, squad,
compaction, a permission extension, ticketing, a context extension, a custom
TUI, and the `/memory-sync` command — all from skill-pi's own config paths,
never touching a user's plain `pi` install. The intent: a user who runs
`skill-pi` in a skillgrid repo gets the same memory-aware, spec-disciplined
agent experience opencode/kilo users get, with Pi as the runtime.

## Purpose & Success Criteria

- **Purpose:** Let a developer run Pi with skillgrid identity, skills, memory,
  and workflow already loaded, from skill-pi's own config paths, without
  affecting a standalone `pi` install.
- **Success criteria:**
  - `skill-pi` (a launcher CLI) runs in a skillgrid repo and resolves its
    global config from `~/.skill-pi/` and its project config from
    `<project-root>/.skill-pi/`; `~/.pi/` and `<project-root>/.pi/` are
    byte-identical before and after.
  - The five opencode-behavior extension ports (memory, codeindex, events,
    squad, compaction) are present and wired to the mnemonic MCP surface.
  - The permission extension (ADR-0037) gates tool calls per its 3-tier config
    and fails closed by default.
  - The `ticket` tool manages Backlog.md via the `backlog` MCP server (one
    source of truth, no `.tickets/` dir).
  - The context extension exposes `/skillgrid context` (usage map + injections
    + mnemonic layer + compaction advising).
  - The TUI renders using `@earendil-works/pi-tui` primitives.
  - `/memory-sync` exports observations to a committed `facts.jsonl`.
  - `skill-pi init`, `skill-pi sync`, `skill-pi version` work as launcher-owned
    subcommands; all other args pass through to pi verbatim.
- **Out of scope:**
  - The CTX transparent proxy base-URL override wiring (separate later change,
    gated on `2026-10-06-context-harness-clm` + `2026-10-07-replace-opencode-hooks-with-plugins`).
  - Vendoring the Pi monorepo into this repo.
  - `mem_fork` (zero-copy subagent scoped view) — v2.
  - OS-level sandboxing, network permission model, path-based permission rules.
  - `pi install`-able distribution (v1 is installer-synced only).
  - A second ticket format (tk `.tickets/`).

## Context

- **Existing per-harness customization** lives in `plugins/opencode/`,
  `plugins/cursor/`, `plugins/kilo/`, with subagent definitions in
  `agents/opencode/`, `agents/cursor/`. skill-pi extends this pattern to a new
  top-level `skill-pi/` module (matching `mnemonic/`, `skillgrid-cli/`,
  `skillgrid-ui/`).
- **The installer** (`skillgrid-cli/`, Go) is the established owner of
  machine-local agent state: "Writes to `~/.skillgrid/`, `~/.agents/`, and
  agent config files; the only writer of agent MCP config"
  (`.skillgrid/ASSUMPTIONS.md` § LOCKED, ADR-0004 component map). skill-pi
  follows the same split: repo = source, installer = machine state.
- **Pi extension seams** (source-verified, see `findings.md` § "Key Pi
  behaviors"): `PI_CODING_AGENT_DIR` relocates the entire global tree; the
  project config dir name is overridable via the startup package's
  `package.json` `piConfig.configDir` field (the gsd-pi precedent); project
  resources load from `join(cwd, CONFIG_DIR_NAME)` only when the project is
  trusted; trust is keyed by the canonicalized (realpath'd) cwd.
- **Mnemonic MCP surface** (`mem_*`, `code_*`, `web_*`, `session_events`,
  `mem_inject_session`) is the existing skillgrid memory server — skill-pi
  registers its tools in pi's registry via extension ports, adding zero new
  memory dependencies.
- **CTX spec** (`.skillgrid/specs/2026-10-08-context-orchestrator/briefing.md`,
  ADR-0033…0036) names `pi` in its `wrap:` opt-in agent list — the CTX proxy
  wiring is a later delta; this change only keeps the LLM base URL overridable.
- **ADR-0037** (permission extension) is already locked in
  `.skillgrid/ASSUMPTIONS.md` § LOCKED and its design is in `findings.md`.
- **ADR-0038** (ticketing: MCP path + `ticket` tool name) is already locked.
- **Operational rules** (`AGENTS.md` § Rules): Go 1.22+ floor (ADR-0031); no
  new dependencies without an ADR (ADR-0008 precedent); serial development;
  conventional commits, no AI attribution; spec-zone commits before code-zone;
  Trivy advisory-only.
- **Constraints from `findings.md`:** pi ships its own supply-chain hardening
  (pinned deps, shrinkwrap); a 70+-skill eager catalog is an anti-pattern (GSD
  finding); the shim is the price of pi having no `--project` flag.

## Approaches Considered

- **Chosen:** Top-level `skill-pi/` module; launcher CLI that relocates Pi's
  config via `PI_CODING_AGENT_DIR` (global) + a per-project cwd shim carrying
  a `package.json` with `piConfig.configDir = ".skill-pi"` (project); pin the
  `@earendil-works/pi-coding-agent` version (no vendoring); port (not
  duplicate) the five opencode plugins to pi extension ports; sync skills from
  `~/.agents/skills/`; installer-synced distribution.
  Rationale: matches the existing top-level module convention, keeps skill-pi
  a versionable `package.json` unit, reuses the proven gsd-pi identity trick,
  and keeps one distribution channel (the installer).
- **Rejected:** `plugins/pi/` — it is not just plugins; it carries skills,
  system prompt, prompts, install logic, and its own package. Rejected:
  `.skillgrid/skill-pi/` — that zone is spec/state, not shipped source.
  Rejected: `pi install`-able package — a second distribution channel that
  fights the installer's ownership of machine state. Rejected: vendoring the
  Pi monorepo — fights pi's own supply-chain hardening. Rejected: tk ticket
  format — would create a second ticket dir with no sync to Backlog.md.

## Requirements

1. **R1 — Global config relocation to `~/.skill-pi/`:** The launcher sets
   `PI_CODING_AGENT_DIR` so pi reads its entire global tree (settings, mcp,
   auth, models, trust, session store) from `~/.skill-pi/`.
   - **Current:** No skill-pi launcher exists; a plain pi install reads
     `~/.pi/agent/`.
   - **Target:** `skill-pi` runs pi with `PI_CODING_AGENT_DIR=~/.skill-pi`;
     `~/.pi/` is never read or written.
   - **Acceptance:** Run `skill-pi` in a repo; assert `~/.pi/` is
     byte-identical (checksums before/after) and `~/.skill-pi/settings.json`,
     `~/.skill-pi/mcp.json`, `~/.skill-pi/trust.json` are present and used.
   - **Acceptance scenario:** `happy path global-relocation` →
     `acceptance.feature`.

2. **R2 — Project config resolution via cwd shim:** The launcher builds a
   per-project cwd shim under `~/.skill-pi/cwd/<project>/` whose
   `package.json` declares `piConfig.configDir = ".skill-pi"` and
   `piConfig.name = "skill-pi"`, symlinks the real project root entries into
   it, and `chdir`s into it before exec'ing pi.
   - **Current:** No shim; pi's project config resolves to
     `<project-root>/.pi/`.
   - **Target:** `join(cwd, ".skill-pi")` resolves through the shim symlink to
     `<project-root>/.skill-pi/`; `piConfig.name = "skill-pi"` sets the
     identity.
   - **Acceptance:** Run `skill-pi --where` (R9); assert `configDir`
     resolves to `<project-root>/.skill-pi/` and `<project-root>/.pi/` is
     byte-identical before/after.
   - **Acceptance scenario:** `happy path project-shim` →
     `acceptance.feature`.

3. **R3 — Shim rebuild cadence:** The launcher rebuilds the shim symlinks on
   every launch (deterministic, always consistent); no hash-check fast path.
   - **Current:** No shim.
   - **Target:** Every `skill-pi` invocation re-materializes the shim from
     scratch before exec'ing pi.
   - **Acceptance:** Mutate the real project root (add a file); run `skill-pi`
     again; assert the new file is visible through the shim without a manual
     sync.
   - **Acceptance scenario:** `happy path shim-rebuild` →
     `acceptance.feature`.

4. **R4 — `<project>/.skill-pi/` is a real gitignored dir:** The project
   config dir is a real directory (not a symlink) at the project root,
   gitignored per repo, materialized by the launcher from the committed
   `skill-pi/` source.
   - **Current:** No `.skill-pi/` dir.
   - **Target:** `<project-root>/.skill-pi/` is a real dir containing
     `mcp.json`, `settings.json`, `SYSTEM.md`, `extensions/`, `skills/`,
     `prompts/`, and an optional `overrides/`; it is gitignored.
   - **Acceptance:** After first run, `<project-root>/.skill-pi/` exists as a
     real dir (`test -d` and not a symlink); `.gitignore` contains
     `.skill-pi/`; `git status` shows it as ignored, not untracked.
   - **Acceptance scenario:** `happy path project-dir` →
     `acceptance.feature`.

5. **R5 — Five opencode-behavior extension ports:** skill-pi ships five TS
   extensions that port the opencode plugin behavior to Pi's `ExtensionAPI`:
   `mnemonic-memory.ts`, `mnemonic-codeindex.ts`, `skillgrid-events.ts`,
   `skillgrid-squad.ts`, `skillgrid-compaction.ts`.
   - **Current:** The behavior exists only in `plugins/opencode/*.ts`.
   - **Target:** Five extensions under `skill-pi/extensions/` register the
     equivalent tools/hooks in pi's registry, wired to the mnemonic MCP
     surface (`mem_*`, `code_*`, `web_*`, `session_events`,
     `mem_inject_session`).
   - **Acceptance:** Each extension file exists; the pi session exposes the
     corresponding tools (memory CRUD, code search/index, event capture,
     squad dispatch, compaction summary) and a `mem_save` call from a pi
     session lands in the mnemonic store.
   - **Acceptance scenario:** `happy path extension-ports` →
     `acceptance.feature`.

6. **R6 — Permission extension (ADR-0037):** A `permission.ts` extension
   implements the 3-tier config (global/project/local), Tree-sitter bash
   matching, pre-start tool hiding, diff preview, and hybrid approvals;
   default mode is fail-closed (`ask`).
   - **Current:** No permission extension for pi.
   - **Target:** `permission.ts` reads `~/.skill-pi/permission.settings.json`
     (global), `<project>/.skill-pi/permission.settings.json` (project),
     `<project>/.skill-pi/permission.settings.local.json` (local); merges
     local > project > global; gates `tool_call` per `deny → ask → allow`;
     hides deny/ask tools pre-start; renders a unified diff for non-allowed
     `edit`/`write`; prompts `[Allow once] [Allow always] [Allow for session]
     [Reject] [Reject with feedback...]` on `ask`.
   - **Acceptance:** With `defaultMode: "ask"` and no matching rule, an `edit`
     call triggers the approval prompt; with a `deny` rule on `bash` pattern
     `rm -rf *`, that call is blocked with `Permission denied` and no prompt;
     a `tree-sitter` parse of `git status | grep error` matches both
     `git status` and `grep error`; "Allow always" persists to the local tier
     file.
   - **Acceptance scenario:** `happy path permission-gate` →
     `acceptance.feature`.

7. **R7 — Ticketing extension (ADR-0038):** A `tickets.ts` extension provides
   one `ticket` tool (10 actions) + one `/tickets` slash command, backed by
   the `backlog` MCP server (stdio), managing Backlog.md as the single source
   of truth.
   - **Current:** No ticket tool in pi.
   - **Target:** `tickets.ts` registers a `ticket` tool with actions
     `ready`, `list`, `show`, `create`, `start`, `complete`, `archive`,
     `note`, `status`, `update`; each maps to the corresponding `backlog_*`
     MCP tool; output is capped at `MAX_OUTPUT = 40_000` chars; `ready`
     results are line-clamped (default 20, max 100).
   - **Acceptance:** `ticket ready` returns ready-for-agent Backlog.md tasks;
     `ticket create` with a title produces a `.backlog/tasks/task-NNN.md`;
     no `.tickets/` dir is created; a 50k-char list is truncated to 40k.
   - **Acceptance scenario:** `happy path ticket-tool` →
     `acceptance.feature`.

8. **R8 — Context extension:** A context extension combines compaction
   advising (Jev two-question score + mnemonic token pressure), context
   visualization (system prompt, tools, skills, extension injections), and
   mnemonic context injection (`mem_inject_session`, `/context`,
   `/context/compaction`).
   - **Current:** No context extension for pi.
   - **Target:** The extension registers `/skillgrid context`,
     `/skillgrid context usage`, `/skillgrid context injections`,
     `/skillgrid context compact`; on `session_start` it captures the system
     prompt/tools and optionally calls `GET /context` for a baseline; on
     `agent_settled`/`turn_end` it updates the Jev score + usage map; on
     `session_before_compact` it calls `GET /context/compaction` and injects
     the result. The usage map includes a "Mnemonic" category showing
     `mem_inject_session` cost.
   - **Acceptance:** `/skillgrid context usage` renders a usage map with a
     "Mnemonic" category; `/skillgrid context injections` shows hidden parts;
     crossing the compaction threshold emits a hint or triggers `/compact`.
   - **Acceptance scenario:** `happy path context-view` →
     `acceptance.feature`.

9. **R9 — TUI via `@earendil-works/pi-tui`:** The custom TUI (right rail,
   header, bottom bar, petal state machine, context gauge, changes viewer,
   agents widget, startup banner) is built on `@earendil-works/pi-tui`
   primitives (`Component`, `VStack`, `HStack`, `ScrollView`, `LayoutNode`,
   `TuiAltScreen`, `matchesKey`, `visibleWidth`).
   - **Current:** No custom TUI for pi.
   - **Target:** The TUI renders using `@earendil-works/pi-tui`; the
   responsive breakpoint (full vs compact), rail width, theme tokens, and
   segment drop order follow the gentle-shell reference patterns.
   - **Acceptance:** The TUI renders in an alt-screen terminal; at ≥140 cols
     the full layout (header + transcript + right rail) shows; below it the
     compact layout (bottom bar + agents widget) shows; the context gauge
     escalates tone at 80% and 95%.
   - **Acceptance scenario:** `happy path tui-render` →
     `acceptance.feature`.

10. **R10 — `/memory-sync` command:** A `/memory-sync` prompt template + a
    small export script dumps `~/.skillgrid/` observations to
    `<project>/.skill-pi/memory/facts.jsonl` (committed, with
    `.gitattributes` `merge=union`); the SQLite DB stays gitignored.
    - **Current:** No JSONL memory export.
    - **Target:** `/memory-sync` writes `facts.jsonl` to
      `<project>/.skill-pi/memory/`; the file is git-committable; re-running is
      idempotent (union merge, no duplicates).
    - **Acceptance:** Run `/memory-sync`; assert
      `<project>/.skill-pi/memory/facts.jsonl` exists and is valid JSONL; a
      second run does not duplicate rows; the DB file is gitignored.
    - **Acceptance scenario:** `happy path memory-sync` →
      `acceptance.feature`.

11. **R11 — Skills sync from `~/.agents/skills/`:** The installer's sync step
    copies/symlinks the skill-pi skill set from the standard
    `~/.agents/skills/` location into skill-pi's skills location; pi reads
    `<configDir>/skills/`, not `.agents/skills/`.
    - **Current:** No pi skills location.
    - **Target:** `skill-pi/skills/` (committed source) + the synced
      `<project>/.skill-pi/skills/` + `~/.skill-pi/skills/` are populated from
      `~/.agents/skills/`; pi discovers them via `--extension <skill-pi-root>`
      (auto-discovery suppressed).
    - **Acceptance:** After sync, `~/.agents/skills/<name>/SKILL.md` for a
      representative skill is present in the pi skills location; a pi session
      lists it in its available skills; `~/.agents/skills/` remains the single
      source of truth (no second catalog in the repo).
    - **Acceptance scenario:** `happy path skills-sync` →
      `acceptance.feature`.

12. **R12 — Pi version pin:** The `@earendil-works/pi-coding-agent` version is
    exact-pinned in `pi-version.txt` (single line) and `package.json`
    (matching), plus a documented minimum version that
    `piConfig.configDir` must exist in.
    - **Current:** No pi pin.
    - **Target:** `skill-pi/pi-version.txt` and `skill-pi/package.json`
      `dependencies["@earendil-works/pi-coding-agent"]` hold the same exact
      version; `README.md` states the minimum pi version for
      `piConfig.configDir`.
    - **Acceptance:** `pi-version.txt` content equals the `package.json`
      dependency string; the version is exact (no `^`/`~`/range); `README.md`
      names a minimum version.
    - **Acceptance scenario:** `happy path version-pin` →
      `acceptance.feature`.

13. **R13 — Installer-synced distribution:** `skillgrid-cli` install/sync is
    extended to (a) copy `skill-pi/` → `~/.skill-pi/` and `npm ci` the pinned
    pi-coding-agent there, (b) build `~/.skill-pi/` from the committed source,
    (c) materialize `<project-root>/.skill-pi/` + the per-project cwd shim
    lazily (launcher does it on first run per project; installer does it for
    the current repo at install time), (d) put the `skill-pi` bin on PATH.
    No `pi install` surface in v1.
    - **Current:** The installer wires opencode/cursor/kilo but not pi.
    - **Target:** `skillgrid-cli install` and `sync-repo` also produce
      `~/.skill-pi/` and put `skill-pi` on PATH; no `package.json` `"pi"` key
      install path is exercised in v1.
    - **Acceptance:** Run `skillgrid-cli install` in a repo; assert
      `~/.skill-pi/node_modules/@earendil-works/pi-coding-agent` exists at the
      pinned version, `skill-pi` is on PATH, and `<project-root>/.skill-pi/`
      is materialized.
    - **Acceptance scenario:** `happy path installer-sync` →
      `acceptance.feature`.

14. **R14 — Launcher-owned subcommands + passthrough:** `skill-pi init`,
    `skill-pi sync`, `skill-pi version` are real subcommands owned by the
    launcher before handoff; all other args are forwarded verbatim to pi.
    - **Current:** No `skill-pi` CLI.
    - **Target:** `bin/skill-pi.js` intercepts `init`/`sync`/`version` and
      handles them; any other argv (e.g. `skill-pi --help`,
      `skill-pi mcp list`) is exec'd to the pinned pi unchanged.
    - **Acceptance:** `skill-pi version` prints the pinned pi version without
      entering the pi TUI; `skill-pi sync` re-runs the sync step;
      `skill-pi --help` shows pi's help (not the launcher's); a pi flag like
      `--session` is forwarded and honored.
    - **Acceptance scenario:** `happy path cli-passthrough` →
      `acceptance.feature`.

15. **R15 — `--where` diagnostic flag:** `skill-pi --where` prints a
    diagnostic JSON of the resolution.
    - **Current:** No diagnostic.
    - **Target:** `--where` prints
      `{ skillPiRoot, piCli, agentDir, stateDir, shimDir, projectRoot,
      configDir, executable }`.
    - **Acceptance:** `skill-pi --where` exits 0 and emits a parseable JSON
      object with all eight keys; `configDir` equals
      `<project-root>/.skill-pi` and `agentDir` equals `~/.skill-pi`.
    - **Acceptance scenario:** `happy path where-diagnostics` →
      `acceptance.feature`.

16. **R16 — SYSTEM.md is the delta over AGENTS.md:** `skill-pi/system/SYSTEM.md`
    carries only the skillgrid delta (zone discipline, commit contract,
    pi-specific rules); the persona + engram protocol are not duplicated — pi
    reads `AGENTS.md` directly for those.
    - **Current:** No `SYSTEM.md`.
    - **Target:** `SYSTEM.md` contains no copy of the `gentle-ai:persona` or
      `gentle-ai:engram-protocol` blocks; it references them and adds only the
      delta.
    - **Acceptance:** `grep -c "gentle-ai:persona" skill-pi/system/SYSTEM.md`
      returns 0 (the block is not inlined); the delta sections (zone
      discipline, commit contract) are present.
    - **Acceptance scenario:** `happy path system-md-delta` →
      `acceptance.feature`.

17. **R17 — LLM base URL overridable (CTX proxy compatibility):**
    `settings.json` / `mcp.json` do not hardcode a provider base URL; the
    base URL is left overridable so `mnemonic ctx wrap pi` can inject the
    loopback `:8787` base URL.
    - **Current:** No pi settings.
    - **Target:** No provider base URL is a literal in
      `skill-pi/settings.json` or `skill-pi/mcp.json`; an env var or settings
      field can override it.
    - **Acceptance:** `grep -rE "https?://[a-z0-9.-]+/(v1|api)" skill-pi/settings.json skill-pi/mcp.json`
      returns no provider base URL; setting the override env var changes the
      effective base URL.
    - **Acceptance scenario:** `happy path base-url-override` →
      `acceptance.feature`.

18. **R18 — Quiet startup + suppression flags:** The launcher sets
    `PI_SKIP_VERSION_CHECK=1` and injects `--no-skills --no-prompt-templates
    --no-themes --extension <skill-pi-root>` before spawn, so pi's own
    version banner is suppressed and skill-pi is the sole resource authority.
    - **Current:** No launcher.
    - **Target:** Every pi spawn from the launcher carries
      `PI_SKIP_VERSION_CHECK=1` and the four flags.
    - **Acceptance:** `skill-pi` does not print pi's version banner; pi does
      not auto-discover `~/.pi/agent/skills/` or other package skills; only
      skill-pi's resources are loaded (verified via `/skillgrid context
      injections`).
    - **Acceptance scenario:** `happy path quiet-startup` →
      `acceptance.feature`.

19. **R19 — `mcp-bridge` registers MCP tools as native pi tools:** An
    `mcp-bridge.ts` extension registers external MCP servers (mnemonic,
    backlog, fluxcd, codescene, trivy) as native pi tools with the `mcp__`
    prefix in the tool registry.
    - **Current:** No MCP bridge for pi.
    - **Target:** `mcp-bridge.ts` is loaded at session start and registers the
      MCP tools; ADR-0037's `mcp__*` permission glob matches them.
    - **Acceptance:** A pi session exposes `mcp__<server>__<tool>` entries in
      its tool registry; the permission extension's `mcp__*` glob matches
      them.
    - **Acceptance scenario:** `happy path mcp-bridge` →
      `acceptance.feature`.

20. **R20 — Trust model via `~/.skill-pi/trust.json`:** The trust store lives
    at `~/.skill-pi/trust.json` keyed by the shim's canonicalized (realpath'd)
    path; the first-run trust prompt fires once per project and persists.
    - **Current:** No skill-pi trust store.
    - **Target:** First run in a project shows pi's trust prompt; the decision
      is written to `~/.skill-pi/trust.json` under the shim's realpath key;
      subsequent runs skip the prompt.
    - **Acceptance:** First `skill-pi` run in a fresh repo shows the trust
      prompt; after accepting, `~/.skill-pi/trust.json` contains a key equal
      to the shim's realpath; a second run shows no prompt.
    - **Acceptance scenario:** `happy path trust-persist` →
      `acceptance.feature`.

21. **R21 — `chronos` date/time injection:** A one-line `chronos.ts` extension
    injects the authoritative current date/time into the session.
    - **Current:** No chronos extension.
    - **Target:** `chronos.ts` is loaded at session start and injects the
      current date/time.
    - **Acceptance:** A pi session's context includes the injected
      date/time; the value matches the system clock.
    - **Acceptance scenario:** `happy path chronos-inject` →
      `acceptance.feature`.

22. **R22 — Launcher spawn + restart loop:** The launcher spawns pi as a
    child and intercepts exit code 75 to re-spawn; `skill-pi /update` and
    `skill-pi /restart` signal restart via `process.exit(75)`.
    - **Current:** No launcher.
    - **Target:** `bin/skill-pi.js` spawns (not execs) pi; on child exit 75 it
      re-spawns; `/update` and `/restart` trigger the loop.
    - **Acceptance:** Triggering `/restart` causes a visible pi re-spawn; a
      normal exit (0) does not re-spawn.
    - **Acceptance scenario:** `happy path restart-loop` →
      `acceptance.feature`.

## Implementation Decisions

- **Modules to build/modify:**
  - NEW `skill-pi/` top-level module: `README.md`, `package.json` (name
    `skill-pi`, `piConfig: { name: "skill-pi", configDir: ".skill-pi" }`,
    `bin: { "skill-pi": "./bin/skill-pi.js" }`, exact-pinned
    `@earendil-works/pi-coding-agent` dep), `pi-version.txt`,
    `bin/skill-pi.js` (launcher), `mcp.json`, `settings.json`,
    `system/SYSTEM.md`, `system/APPEND_SYSTEM.md`, `extensions/` (
    `mnemonic-memory.ts`, `mnemonic-codeindex.ts`, `skillgrid-events.ts`,
    `skillgrid-squad.ts`, `skillgrid-compaction.ts`, `permission.ts`,
    `tickets.ts`, `mcp-bridge.ts`, `chronos.ts`), `skills/`, `prompts/`
    (incl. `/memory-sync`), `themes/`, `install/install.md`,
    `install/sync.sh`.
  - MODIFIED `skillgrid-cli/` (Go): extend `install` / `sync-repo` / `setup`
    to copy `skill-pi/` → `~/.skill-pi/`, `npm ci` the pinned pi, materialize
    `<project-root>/.skill-pi/` + the shim, put `skill-pi` on PATH.
  - UNCHANGED: `plugins/opencode/*.ts` stay the opencode-harness source of
    truth; the pi ports are reviewed for drift, not auto-synced.
- **Interfaces:**
  - `bin/skill-pi.js`: argv in; if `argv[1] ∈ {init, sync, version}` handle
    in-launcher; else set `PI_CODING_AGENT_DIR`, build the shim, `chdir`,
    spawn pi with `process.argv` forwarded + `PI_SKIP_VERSION_CHECK=1` +
    `--no-skills --no-prompt-templates --no-themes --extension
    <skill-pi-root>`. Exit-code-75 re-spawn loop. `--where` prints the 8-key
    JSON.
  - Shim `package.json` (fixed content): `{"name":"skill-pi-cwd","version":"0.0.0","piConfig":{"configDir":".skill-pi"}}`.
  - Extension ports: each `skill-pi/extensions/*.ts` exports a pi extension
    (jiti-loaded) that registers tools/hooks on the mnemonic MCP surface;
    signatures mirror `plugins/opencode/*.ts` behavior, not file structure.
  - `permission.ts`: 3-tier config loader (local > project > global),
    `tool_call` gate returning `{ block: true, reason }` on deny, pre-start
    `exposure: 'hidden'` for deny/ask tools, diff preview for `edit`/`write`,
    hybrid approval prompt.
  - `tickets.ts`: one `ticket` tool, 10 actions, each mapping to a
    `backlog_*` MCP tool; `MAX_OUTPUT = 40_000`; `ready` clamped.
  - Context extension: `/skillgrid context [usage|injections|compact]`;
    Jev score + `memory_pressure = mem_inject_session_tokens / max_tokens`.
- **Data flow:**
  - Launch: launcher → set env → build shim → chdir → spawn pi → pi reads
    `~/.skill-pi/` (global) + `<project>/.skill-pi/` (project) → extensions
    load → MCP tools register → session.
  - Memory: pi agent → `mem_save`/`mem_search`/`mem_inject_session` (MCP) →
    mnemonic store (`~/.skillgrid/mnemonic`). `/memory-sync` → `facts.jsonl`.
  - Tickets: pi agent → `ticket` tool → `backlog` MCP (stdio) →
    `.backlog/tasks/*.md`.
  - Permissions: pi `tool_call` → `permission.ts` rule eval → allow / ask
    (prompt) / deny (block).
  - Context: `session_start` → capture + `GET /context`; `turn_end` → Jev
    score + usage map; `session_before_compact` → `GET /context/compaction`.
- **Error handling:**
  - Shim build failure (real root missing / not a git repo): launcher prints
    the resolution error and exits non-zero; no pi spawn.
  - Pi version mismatch (`~/.skill-pi/pi-version.txt` ≠ installed): launcher
    reports the drift and offers `skill-pi sync`; no silent fallback.
  - Tree-sitter parse failure in `permission.ts`: fall back to wildcard
    matching (fail open: whole command as one token).
  - MCP server unreachable: extension surfaces the error in the tool result;
    the agent sees it, no silent drop.
  - `/memory-sync` on an empty store: writes an empty (valid) JSONL; idempotent.
- **Dependencies:**
  - `@earendil-works/pi-coding-agent` (exact-pinned) — new for this repo; ADR
    required (ADR-0008 precedent).
  - `@earendil-works/pi-tui` — new for this repo; ADR required.
  - `tree-sitter` + `tree-sitter-bash` — already in the project dependency
    surface (mnemonic code index); new for the skill-pi package; recorded in
    ADR-0037.
  - `@pierre/diffs` — new for the skill-pi package; recorded in ADR-0037.
  - `typebox` — pi-coding-agent peer dep, already in the surface.
  - `backlog` MCP server (stdio) — already wired for opencode/cursor/kilo;
    reused.
- **Prototype snippets:** None produced this round; the shim +
  `piConfig.configDir` resolution is source-verified in `findings.md` (pi
  `config.ts`, `settings-manager.ts`, `resource-loader.ts`, main branch,
  v1.1.0).

## Testing Decisions

- **What makes a good test:** Test the launcher's observable behavior (env
  vars set, shim contents, argv forwarded, exit codes) and each extension's
  tool surface (tool registered, correct result shape), not internal TS
  structure. The Go installer changes are tested at the install/sync seam.
- **Modules to test:**
  - `bin/skill-pi.js`: unit tests for argv routing (init/sync/version vs
    passthrough), shim materialization, `--where` output, exit-75 re-spawn.
  - `permission.ts`: rule precedence (deny → ask → allow), 3-tier merge,
    Tree-sitter pipe/subshell matching, fail-closed default, "Allow always"
    persistence.
  - `tickets.ts`: action → MCP tool mapping, output capping, `ready` clamp,
    no `.tickets/` created.
  - Context extension: Jev score threshold, usage-map mnemonic category,
    `session_before_compact` injection.
  - `skillgrid-cli` (Go): install/sync produces `~/.skill-pi/` + pinned pi +
    PATH entry + `.skill-pi/` materialized.
- **Prior art:** `go test ./...` for the Go installer (existing
  `skillgrid-cli/` tests); the TS extensions have no prior in-repo test
  harness — model after pi's own extension test patterns and add a minimal
  `node --test` or vitest setup under `skill-pi/` (decision recorded as an
  assumption; the Go floor is the repo's enforced runner).
- **Edge cases:** Real root with a `.git` file (worktree pointer) vs dir;
  macOS `/tmp` → `/private/tmp` realpath; a project with no
  `.skill-pi/overrides/`; an empty mnemonic store at `/memory-sync`; a pi
  version older than `piConfig.configDir` (min-version guard); a `rm -rf`
  command with command substitution; a 50k-char ticket list; a terminal < 140
  cols (TUI compact layout); MCP server down mid-session.

## Impact on Global Docs

- `.skillgrid/artifacts/00-prd.md`: Add `skill-pi` as a shipped component /
  feature (new harness distribution).
- `.skillgrid/ASSUMPTIONS.md`: New ADR row(s) for the `@earendil-works/pi-*`
  npm dependency surface (ADR-0008 precedent) — confirm ADR-0037 and ADR-0038
  rows are present; add the pi-tui dependency row.
- `.skillgrid/ARCHITECTURE.md`: New top-level component `skill-pi/` (launcher
  CLI + pi extension ports + TUI + install); new machine-state dir
  `~/.skill-pi/`; new per-project dir `<project>/.skill-pi/`; installer
  (ADR-0004) now also owns `~/.skill-pi/`.

## Clarity Report

| Dimension           | Score | Min  | Status | Notes                              |
|---------------------|-------|------|--------|------------------------------------|
| Goal Clarity        | 0.90  | 0.75 | ✓      | Full-surface v1 scope locked (R1–R22). |
| Boundary Clarity    | 0.85  | 0.70 | ✓      | Out-of-scope list explicit (CTX proxy, vendoring, mem_fork, pi install, tk). |
| Constraint Clarity  | 0.80  | 0.65 | ✓      | Operational rules + pi supply-chain + shim constraint all stated. |
| Acceptance Criteria | 0.85  | 0.70 | ✓      | Every R has Current/Target/Acceptance + BDD pointer. |
| **Clarity**         | 0.15  | ≤0.20| ✓      | Pass. |

**Interview log:**

| Round | Question summary | Decision locked |
|-------|-----------------|-----------------|
| 1 | Scope (Q1) | Full surface: structure + 5 opencode-behavior ports + permission (ADR-0037) + tickets + context extension + TUI + `/memory-sync`. |
| 1 | Packaging (Q3) | Installer-synced local resources; no `pi install` surface in v1. |
| 1 | CLI shape (Q5) | Passthrough + owned subcommands (`init`/`sync`/`version`). |
| 1 | Name/dir (Q6) | Package `skill-pi`, dir `skill-pi/`. |
| 2 | Skills subset (Q2) | Import all from the standard `~/.agents/skills/`. |
| 2 | Shim rebuild (Q7) | Rebuild every launch + real gitignored `<project>/.skill-pi/` dir. |
| 2 | TUI source | Use `@earendil-works/pi-tui`. |
| 2 | Pi version floor (Q8) | Exact pin + documented min-version. |
| 2 | System prompt (Q4) | Reuse AGENTS.md persona; `SYSTEM.md` = delta only. |

## Open Questions & Assumptions

- **Assumption:** The TS extension test harness (`node --test` / vitest under
  `skill-pi/`) is not yet established in the repo; the Go floor
  (`go test ./...`) is the enforced runner. A minimal TS test setup is an
  implementation detail, not a new dependency (the test runner is a dev tool,
  flagged for the dependency ADR if it is not already present).
- **Assumption:** `@earendil-works/pi-tui` is publicly installable at the
  pinned pi version (the gentle-shell reference imports it); if it is not in
  the pi tarball's dependency closure at the pin, the TUI requirement (R9)
  falls back to building the minimal TUI layer — a decision to re-confirm at
  blueprint time.
- **Assumption:** The `backlog` MCP server (stdio) is available to pi via
  `mcp.json` at the same transport it uses for opencode/cursor/kilo; no new
  server process is introduced.
- **Question (deferred):** Whether the context extension registers `/context`
  (shadowing `pi-context-view`) in addition to `/skillgrid context` — v1 uses
  `/skillgrid context` only; shadowing is a v2 candidate.

## Decisions (ADR)

- `.skillgrid/artifacts/04-adr-0037-permission-extension.md` — permission
  extension: 3-tier config, Tree-sitter bash matching, pre-start tool hiding,
  diff preview, hybrid approvals, fail-closed default.
- `.skillgrid/artifacts/04-adr-0038-ticketing-mcp-adapter.md` — ticketing:
  `ticket` tool backed by the `backlog` MCP server (Backlog.md single source
  of truth), not tk.
- New ADR (to be recorded): `@earendil-works/pi-coding-agent` +
  `@earendil-works/pi-tui` npm dependency surface (ADR-0008 precedent).

## Terms

- **skill-pi** — the opinionated, non-fork distribution of Pi built on the
  skillgrid standards (this feature). `.skillgrid/artifacts/02-technical-terms.md`
  (new term).
- **cwd shim** — the per-project directory under `~/.skill-pi/cwd/<project>/`
  whose `package.json` carries `piConfig.configDir = ".skill-pi"`, through
  which pi's `join(cwd, CONFIG_DIR_NAME)` resolves to the real project's
  `.skill-pi/`. `.skillgrid/artifacts/02-technical-terms.md` (new term).
- **extension port** — a `skill-pi/extensions/*.ts` file that ports the
  behavior of a `plugins/opencode/*.ts` plugin to Pi's `ExtensionAPI`.
  `.skillgrid/artifacts/02-technical-terms.md` (new term).
- **Extractor** — existing term (mnemonic code index); referenced by ADR-0037
  for the Tree-sitter bash-matching dependency. No change.
