# Task 3 — Global config + system prompt delta

**Ticket:** TICKET-03 / TASK-071
**SATISFIES:** `global-config-no-url`, `system-delta-only`
**Blocked by:** TICKET-01 (complete), TICKET-02 (complete) — both landed.
**Blocks:** TICKET-04, TICKET-05

## What to build

Create four files under `skill-pi/` and extend `skill-pi/version.test.js`:

1. `skill-pi/mcp.json` — pi global MCP config. NO provider base URL literal. The LLM/HTTP base URL must be left overridable via env (the CTX transparent proxy `mnemonic ctx wrap pi` → loopback `:8787` will override it later; this ticket ships the structure, not the proxy wiring).
2. `skill-pi/settings.json` — pi global settings. NO provider base URL literal. Base URL read from `SKILLGRID_CHECKPOINT_URL` / `SKILLGRID_MNEMONIC_HTTP_URL`, default `http://127.0.0.1:7438`.
3. `skill-pi/system/SYSTEM.md` — the skillgrid DELTA ONLY (zone discipline, commit contract). It must REFERENCE (pointer), not duplicate, the `gentle-ai:persona` and `gentle-ai:engram-protocol` blocks — pi reads `AGENTS.md` directly for those. Zero grep hits for those block markers.
4. `skill-pi/system/APPEND_SYSTEM.md` — an empty appendable delta (the lead's goal-loop prompt lands here in Task 20; for now it is an empty / minimal placeholder).
5. `skill-pi/version.test.js` (extend) — add a `-t config`-tagged test group covering the acceptance oracles below.

## The base-URL resolution contract (consumes opencode plugins)

The opencode plugins resolve the base URL exactly like this (`plugins/opencode/mnemonic-memory.ts`, `skillgrid-compaction.ts`, etc.):

```ts
const BASE_URL =
  process.env.SKILLGRID_CHECKPOINT_URL ||
  process.env.SKILLGRID_MNEMONIC_HTTP_URL ||
  "http://127.0.0.1:7438"
```

`settings.json`/`mcp.json` must NOT hardcode a remote provider URL. The extension ports (Task 4) and mcp-bridge (Task 5) read the base URL from these env vars — the config files carry no literal that would shadow the env override.

## Acceptance oracles (the TDD gates — make them RED first, then GREEN)

Run: `node --test skill-pi/version.test.js -t config`

(a) **No provider base URL literal.** `grep -rE "https?://[a-z0-9.-]+/(v1|api)" skill-pi/settings.json skill-pi/mcp.json` returns NO remote provider base URL. The loopback default `http://127.0.0.1:7438` is allowed (it is loopback, not a provider). Reject any remote `https://.../v1` or `https://.../api`.

(b) **Base URL is env-overridable.** A unit test asserting that a resolution function reads the env, not a literal: setting `SKILLGRID_MNEMONIC_HTTP_URL` changes the effective base URL. Assert the default (no env) is `http://127.0.0.1:7438` and that setting the env var yields that value. (The config files themselves are data; the resolution function the extension ports will use must be testable — you may add a tiny `skill-pi/lib/base-url.js` CJS helper that the ports (Task 4) import, and test that. This keeps the "reads the env, not a literal" oracle real rather than a grep-only fiction. Do NOT add pi to `dependencies` — only a stdlib-only helper.)

(c) **System prompt is a pure delta.** `grep -c "gentle-ai:persona\|gentle-ai:engram-protocol" skill-pi/system/SYSTEM.md` is 0.

## Constraints

- **CJS** (no `"type":"module"` in `skill-pi/package.json`). Use `require`/`module.exports`.
- **No new runtime dependencies.** stdlib only (`node:fs`, `node:path`, `node:process`, etc.). The base-URL helper must not import pi.
- **No provider base URL literal** in the two JSON config files. The `127.0.0.1:7438` loopback default is the only URL that may appear, and even that is allowed to live in the helper, not a required literal in the JSON.
- **SYSTEM.md references AGENTS.md, does not copy the persona/engram blocks.** A pointer line like "Persona + memory protocol: see AGENTS.md (gentle-ai:persona / gentle-ai:engram-protocol)" is correct; pasting those blocks is a defect (drift).
- **TDD:** write the `-t config` test group FIRST, run it to RED (files absent), then create the files + helper to GREEN.
- **Commit** per `work-unit-commits`: conventional commit, no AI attribution. Suggested:
  `feat(skill-pi): global config (no provider URL) + system prompt delta`
  Stage only: `skill-pi/mcp.json`, `skill-pi/settings.json`, `skill-pi/system/`, `skill-pi/lib/base-url.js` (if created), `skill-pi/version.test.js`.

## Verify (parent re-runs these after your report)

- `node --test skill-pi/version.test.js` → ALL green (existing skeleton test + new `-t config` group).
- `grep -rE "https?://[a-z0-9.-]+/(v1|api)" skill-pi/settings.json skill-pi/mcp.json` → no remote provider URL.
- `grep -c "gentle-ai:persona\|gentle-ai:engram-protocol" skill-pi/system/SYSTEM.md` → 0.

## Out of scope (do NOT do)

- The CTX proxy wiring (`mnemonic ctx wrap pi`, `:8787`) — that is a later delta (findings.md). Ship the overridable structure only.
- The five extension ports (Task 4), permission/tickets/context/mcp-bridge (Task 5).
- Provider/login configuration (the "No models available" TUI warning is expected until then).
