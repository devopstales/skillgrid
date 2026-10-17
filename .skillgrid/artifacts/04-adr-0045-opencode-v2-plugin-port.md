# ADR-0045: OpenCode V2 plugin port (dual V1+V2 entrypoints)

---
status: "accepted"
supersedes: none
date: 2026-10-17
---

**Supersedes:** nothing (extends ADR-0032's native-plugin architecture)

## Context

The 6 native TS plugins under `plugins/opencode/` (ADR-0032) were written
against the **V1** OpenCode plugin API: a V1 entrypoint is a function returning
an object of hooks/tools, and V2 **will not run V1 implementations** — the
official migration guide is explicit that "moving a file or renaming its config
entry is not enough." The config key also changed (`plugin` → `plugins`), and
the discovery directory changed (`~/.config/opencode/plugin/` →
`~/.config/opencode/plugins/`).

The installed harness (opencode v2.0.28) therefore loaded **none** of the
skillgrid plugins: `~/.config/opencode/plugins/` was empty and
`opencode.jsonc` had no `plugin`/`plugins` array. Memory, code-index, squad,
events, and compaction capture were silently dead in the V2 harness.

## Decision

**Port the plugins to V2 with dual entrypoints** — each file keeps its V1
named export (for V1.18.29+ harnesses) and gains a V2 `Plugin.define({id,
setup(ctx)})` default export. This preserves the kilo/cursor/V1 users while
making the V2 harness load the same source.

- **`plugins/opencode/opencode-v2.ts`** (new, shared shim): restores the two
  module-load globals V1 relied on and V2 no longer injects.
  - `makeTool()` — a JSON-Schema builder (`.describe`/`.optional` fluent chain
    that emits JSON Schema, V2's validation shape) + `wrap()` that turns a V1
    `{description, args, execute}` into a V2 `{name, description, input,
    execute}` with a stringified result (V1's `JSON.stringify` contract).
  - `globalThis.opencode.event.on` — a deferred event bus (handlers register at
    module load; `setEventStream` wires it to the V2 server stream).
  - The shim is a dependency of the V2 entrypoints, not a hook owner.

- **Per-file V2 default exports** (V1 named export kept verbatim):
  - `mnemonic-memory.ts`, `mnemonic-codeindex.ts`, `skillgrid-squad.ts` —
    `ctx.tool.transform` over the same tool map.
  - `skillgrid-events.ts` — `ctx.tool.hook("execute.before"/"execute.after")`
    reusing the shared `handleToolBefore`/`handleToolAfter`; the policy-block
    rethrow is preserved (a block must propagate out of the V2 hook to block
    the write).
  - `skillgrid-compaction.ts` — `ctx.event.subscribe()` for
    `session.created`/`session.idle`/`session.compacted`.
  - `skillgrid-context.ts` — **not ported to V2.** It needs
    `experimental.chat.messages.transform` (CLM budget withholding), which has
    no V2 hook equivalent. It stays V1-only; in the V2 harness the CLM request
    path is simply inactive (the mirror is byte-identical when CLM is off, so
    no behavior change by default).

- **`@opencode/plugin` is a bundled OpenCode dependency**, so the V2 entrypoint
  import resolves at load in the real harness. The repo smoke test (plain Node)
  uses a best-effort `import("@opencode/plugin")` with a local
  `Plugin.define` stub fallback (same pattern as the Pi extension's TypeBox
  fallback), so it verifies the V2 registrations without the package.

- **Installer:** `nativePluginBases` in `mnemonic/internal/setup/setup.go`
  gains `opencode-v2` so the shim ships next to the plugins for the V2
  entrypoints' `./opencode-v2.ts` import to resolve. The Go tests iterate
  `nativePluginBases`, so they cover the shim automatically.
  - **OpenCode** copies into `~/.config/opencode/plugins/` — the v2 discovery
    directory — via `installOpenCodeNativePlugins` (a new opencode-specific
    function). The legacy `plugin` config array is kept only to carry the
    retired-entry cleanup; v2 discovery is what actually loads the plugins.
  - **Kilo** keeps the v1 `~/.config/kilo/plugin/` dir + config registration
    (`installNativePlugins`), because Kilo reads its own config file and has
    no v2 discovery equivalent.

## Consequences

- **Positive:** the V2 harness loads all skillgrid plugins again — memory,
  code-index, squad, policy gate, session capture, and compaction. V1 harnesses
  (and kilo/cursor, which copy from the same source) are unaffected.
- **Negative / dropped in V2:**
  - `skillgrid-context`'s CLM `experimental.chat.messages.transform` is
    inactive in V2 (no equivalent hook). Off by default, so no change unless
    CLM is enabled.
  - `skillgrid-compaction`'s `experimental.session.compacting` pre-compact
    context file is dropped in V2 (no equivalent hook); the session row's
    summary/observations are still refreshed by `session.compacted`.
  - The V2 tool hooks' exact event field shape (`event.input`/`event.result`/
    `event.status`) is inferred from the V2 docs' examples, not verified
    against a live V2 tool call (the installed harness's model auth returns
    HTTP 400 in this environment). The event→`{input,output}` mapping is
    fail-open: a field mismatch degrades capture to no-ops, it never blocks.
- **One-way gate:** when OpenCode drops V1 plugin support, delete the V1 named
  exports and the `importPlugin()` fallback; the V2 default exports and the
  shim stay. `skillgrid-context` should be ported (or retired) at that point,
  since its V1-only CLM transform will stop loading.

## Installer defaults (companion decision, same change)

While fixing the plugin target, the agent-selection default was changed to
**all unselected**: a plain `skillgrid install` no longer auto-configures any
harness. `--yes` configures all agents; non-interactive runs without `--agents`
skip the agent step entirely. `skill-pi` and `pi` remain available agents
(ADR-0043) but are never auto-selected. This is an operational default, not a
new ADR — it lives in `selectAgents` (`skillgrid-cli/internal/install/install.go`).
