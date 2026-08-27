## Context

Skillgrid's CLI installer (`skillgrid-cli`) runs a pipeline of steps: clone/sync → node → npm → gated per-tool steps → plugins/skills paths → skills → MCP merge → rules. Each step is gated by the presence of a tool in `config.d/tools.yaml`. The gryph integration adds a new gated step (step 4c) after npm install.

Gryph is a local-only audit-trail and security-policy tool. It installs hooks into agents that emit every tool call to a local SQLite store. Its YAML policy (block/warn/guide) applies in real time. Gryph has adapters for many agents but **not Kilo** — so the Kilo integration uses the same plugin-copy pattern already established for the engram bridge.

## Goals / Non-Goals

**Goals:**
- Gryph install triggered by `@safedep/gryph` in `config.d/tools.yaml`
- Hooks for OpenCode via `gryph install --agent opencode`
- Hooks for Kilo via copying the OpenCode plugin file (no native adapter)
- Policy scaffolded and enabled after hooks installed
- Dry-run support (prints planned actions, performs zero execs/writes)
- Agent-selection gating (OpenCode hooks only when opencode selected, etc.)

**Non-Goals:**
- Per-project/repo gryph policy management (global policy only)
- MCP entry for gryph (it's a hook/audit tool, not an MCP provider)
- Kilo-native hook adapter or OpenCode hooks.json manipulation
- Windows-specific path logic beyond existing `mustExpandHomePath`
- Retention/DB tuning (gryph defaults apply)
- Uninstall support in skillgrid v1

## Decisions

### 1. Kilo integration via plugin copy

**Decision:** Copy the OpenCode gryph plugin to Kilo's plugins directory instead of writing Kilo hooks JSON manually.

**Alternatives considered:**
- Writing Kilo hooks JSON manually — rejected because the plugin file is the whole integration surface; copying preserves gryph's marker/idempotency logic
- Waiting for a Kilo-native gryph adapter — rejected because it doesn't exist upstream

**Rationale:** First-write-wins copy mirrors the existing engram bridge pattern (`steps.go` lines 151-201). Gryph's plugin contains a marker that makes re-runs idempotent.

### 2. Install trigger via tools.yaml

**Decision:** Presence of `@safedep/gryph` in `config.d/tools.yaml` gates the gryph step.

**Alternatives considered:**
- New top-level config concept (e.g., `audit: gryph`) — rejected because `config.d/` declares installs, not semantics
- Always install gryph — rejected because not every user needs auditing

**Rationale:** Matches the existing agent-browser gate convention. No new config file or concept.

### 3. Policy scope

**Decision:** Global default policy from `gryph policy init`, not a skillgrid-managed policy file.

**Alternatives considered:**
- skillgrid-managed policy file in `config.d/` — rejected because `config.d/` should declare installs, not gryph's policy semantics
- Per-project policy — rejected as out of scope for v1

**Rationale:** Keeps `config.d/` honest. Gryph's `policy init` scaffolds an example policy and refuses to overwrite existing ones.

## Risks / Trade-offs

- **`gryph` binary missing** -> Fallback to bare `gryph` on PATH; if unresolved, log warning and continue
- **Kilo plugin source missing** -> Log warning with missing path; no crash; Kilo simply has no gryph hooks
- **Policy init fails** -> Log warning, continue to validate
- **Re-runs** -> Idempotent (gryph plugin marker check + first-wins copy + non-force `policy init`)
- **One more process on PATH + one SQLite DB** -> User-visible new dependency

## Migration Plan

1. Add `@safedep/gryph` to `config.d/tools.yaml`
2. Implement `installGryph` in `cmd/steps.go`
3. Add gated call in `cmd/install.go` after the agent-browser block
4. Write `cmd/gryph_test.go` with dry-run and real-path tests
5. Update docs (usage + config reference + NOTE.md)

**Rollback:** `gryph uninstall --restore-backup` per agent + remove `~/.config/kilo/plugins/gryph.js` + remove the `tools.yaml` line.

## Open Questions

None — all resolved during brainstorming.
