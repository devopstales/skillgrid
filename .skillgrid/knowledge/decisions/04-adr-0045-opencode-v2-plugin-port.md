---
type: TechnicalDecision
title: "OpenCode V2 plugin port: dual V1+V2 entrypoints for the 6 native plugins (V1 named export kept + V2 `Plugin.define` default), per-file local `wrap()` (JSON-Schema converter), `skillgrid-context` via `ctx.session.hook(\"context\")`, no shim — every file in plugins/ must default-export {id, setup}"
status: "accepted"
supersedes: none
date: 2026-10-17
---

# ADR-0045: OpenCode V2 plugin port (dual V1+V2 entrypoints)
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

- **Per-file V2 default exports** (V1 named export kept verbatim). Verified
  against the opencode v2.0.28 binary (`PluginModule.load`): the default export
  must be `{id, setup(fn)}` or `{id, effect(fn)}` — OpenCode v2 discovers
  **every file** in `~/.config/opencode/plugins/` and rejects any file without
  a default definition, so there is no separate bootstrap shim.
  - `mnemonic-memory.ts`, `mnemonic-codeindex.ts`, `skillgrid-squad.ts` —
    `ctx.tool.transform` over the same tool map. Each file carries a local
    `wrap()` that converts the V1 arg schema (the `tool.schema.*` calls, which
    collapse to plain objects) into JSON Schema; the result is stringified
    (V1's `JSON.stringify` contract).
  - `skillgrid-events.ts` — `ctx.tool.hook("execute.before"/"execute.after")`
    reusing the shared `handleToolBefore`/`handleToolAfter`; the policy-block
    rethrow is preserved (a block must propagate out of the V2 hook to block
    the write).
  - `skillgrid-compaction.ts` — `ctx.event.subscribe()` for
    `session.created`/`session.idle`/`session.compacted`, plus
    `session.compaction.started` for the pre-compact context file (the V2
    equivalent of V1's `experimental.session.compacting`).
  - `skillgrid-context.ts` — ported to V2 via `ctx.session.hook("context")`.
    The binary's LLM request path fires the `context` session hook on every
    request (primary / compaction / generate) with `{messages, system, tools}`
    — the same shape the V1 `experimental.chat.messages.transform` sees. The
    CLM withhold logic is reused verbatim; off by default (byte-identical path
    when CLM is disabled), every step fail-open.

- **`@opencode/plugin` is a bundled OpenCode dependency**, so the V2 entrypoint
  import resolves at load in the real harness. The repo smoke test (plain Node)
  uses a best-effort `import("@opencode/plugin")` with a local
  `Plugin.define` stub fallback (same pattern as the Pi extension's TypeBox
  fallback), so it verifies the V2 registrations without the package.

- **Installer:** `nativePluginBases` in `mnemonic/internal/setup/setup.go`
  is the 6 plugin file bases (no shim — there is none). The Go tests iterate
  `nativePluginBases`, so they cover every installed file.
  - **OpenCode** copies into `~/.config/opencode/plugins/` — the v2 discovery
    directory — via `installOpenCodeNativePlugins` (a new opencode-specific
    function). The `plugin` config array is kept (the v2.0.28 binary's config
    schema reads `"plugin"` and appends to discovery; the docs call it
    `plugin`, not `plugins`); v2 discovery + the array together load the
    plugins.
  - **Kilo** keeps the v1 `~/.config/kilo/plugin/` dir + config registration
    (`installNativePlugins`), because Kilo reads its own config file and has
    no v2 discovery equivalent.

## Consequences

- **Positive:** the V2 harness loads all 6 skillgrid plugins — memory,
  code-index, squad, policy gate, session capture, compaction, and the CLM
  context hook. V1 harnesses (and kilo/cursor, which copy from the same
  source) are unaffected.
- **Negative / caveats:**
  - `skillgrid-context`'s CLM transform now runs on **every** LLM request via
    the `context` session hook, not only the chat-messages path. The cost
    guard is cheap (a directory walk + config read) and the whole hook is
    no-op when CLM is disabled (off by default), but it is a per-request cost
    the V1 transform did not have.
  - The V2 tool hooks' exact event field shape (`event.input`/`event.result`/
    `event.status`) is inferred from the V2 docs' examples, not verified
    against a live V2 tool call. The event→`{input,output}` mapping is
    fail-open: a field mismatch degrades capture to no-ops, it never blocks.
- **One-way gate:** when OpenCode drops V1 plugin support, delete the V1 named
  exports and the `importPlugin()` fallback; the V2 default exports stay.

## Installer defaults (companion decision, same change)

While fixing the plugin target, the agent-selection default was changed to
**all unselected**: a plain `skillgrid install` no longer auto-configures any
harness. `--yes` configures all agents; non-interactive runs without `--agents`
skip the agent step entirely. `skill-pi` and `pi` remain available agents
(ADR-0043) but are never auto-selected. This is an operational default, not a
new ADR — it lives in `selectAgents` (`skillgrid-cli/internal/install/install.go`).
