# Memory Checkpoint and Memory Index — Design Briefing

> **STATUS:** `approved` (2026-10-02) — user: "Approved as designed"; serial-development constraint overridden for this change ("start execution right after the spec is committed").

**Date:** 2026-10-02
**Tier:** T2
**Classification:** New Function (observer + injection layer over the existing memory architecture)

## Problem / Intent

Every tool call Cursor, OpenCode, and Kilo make is recorded, but almost none of it becomes memory, and the next session starts knowing only the last step, the branch, and the changed files. The user asked for claude-mem's behavior: captured work compressed into typed observations, a summary at stop, and an index of both injected at session start — without an external LLM or API key.

## Purpose & Success Criteria

- **Purpose:** A session's work becomes typed observations and a structured summary without the agent being asked to remember the protocol; the next session in the project opens with an index of recent summaries and observations it can expand on demand; private spans never reach the store; the Sessions view shows observations arriving live.
- **Success criteria:**
  - A Cursor session with ≥5 tool calls gets one follow-up turn that produces ≥1 observation and one summary; a second stop within the cooldown produces no follow-up (G1, G4).
  - `skillgrid prime` in this repository prints a Memory Index of 5 summaries + 20 observations under 800 tokens from the same store the hooks write to (G8, G9).
  - `<private>…</private>` content posted by a hook or saved via MCP is absent from every row and export (G10, G11).
  - A new observation shows in the open session's timeline without reload (G12).
  - All Go, hook, and UI gates in `acceptance.feature` pass; no new dependency.
- **Out of scope:** wiring any external LLM backend into the existing seams (separate ADR); per-tool-call compression; a dedicated Memory page in the UI; prompt-capture hooks (`UserPromptSubmit`/`message.part.updated`); cloud sync; changing `mem_search` ranking.

## Context

- **Capture today:** `postToolUse` / `tool.after.*` → `tool-call-capture.js` → `POST /sessions/{id}/tool-calls` → `session_events`; `promote.go` writes an observation only for errors, repeated writes, or "we decided" text. `cursor-session-end.sh` (stop + sessionEnd) runs `skillgrid compact` (task/branch/diffstat as the session summary) and reports usage. OpenCode/Kilo `session.idle` does the same via `plugins/*/hooks.yaml`.
- **Injection today:** both harnesses run `skillgrid prime` at session start; it renders last step, branch, changed files, impact, and one sentence pointing at the MCP tools. `session_inject` already has `DistillSummary`, `AutoPrepend` (resume-only L1), `RenderContextBlock` (query-driven L2), and privacy filters. `RecentContext(limit)` and `RecentObservations(limit)` exist on the memory service; pinned observations exist (`Pin`/`Pinned`).
- **LLM seams:** `ExtractionLLM`, `DedupLLM`, `DreamLLM`, `layer.LLM`, `AskLLM` exist; no backend is wired anywhere in the CLI. Every pass runs its deterministic floor.
- **Harness contracts (verified 2026-10-02):** Cursor `stop` receives `{conversation_id, generation_id, model, status, loop_count}` and may return `{followup_message}`; `loop_limit` is a per-hook-entry setting. opencode-yaml-hooks `session.idle` can run `command`/`tool` actions (an LLM step) but a `bash` action cannot gate later actions and `session.idle` cannot be `async`, so a yaml hook cannot carry a conditional prompt; an OpenCode plugin can (`session.idle` event → `client.session.prompt`). Kilo loads the same plugin shape.
- **Project identity (VERIFIED, defect):** the Go resolver (`project.Resolve`: config → identity binding → child → hash) maps this repository to `aiskillgrid`, the store with the real memory (88 observations, 117 sessions). `tool-call-capture.js` resolves on its own: the pinned `~/.skillgrid/mnemonic/active-project` file (written by whichever repo last ran `prime`), else git-remote basename → `skillgrid`. Tool events therefore live in `skillgrid.sqlite`, and a `prime` in another repo re-pins every harness's writes to that repo. The index is useless unless both sides agree, so this change moves resolution to the server.
- **Activity stream:** `/activity/stream` already emits `activity` frames for new observation rows (with session id) alongside `tool` frames.
- **Locked constraints respected:** Go 1.22+, no new dependency (stdlib HTTP server-side; OpenCode plugin API client-side), Trivy advisory, spec-zone commits before code, repo is source of truth. Serial development overridden by the user for this change (recorded in `adr.md`).
- **Codebase feasibility check:** Fits existing patterns: **yes-with-notes**. The claim route follows `policy.go`/`toolevents.go` (handler → memory service → SQL, tests at the HTTP seam with `newToolCallsServer`); the index renderer joins `session_inject` next to `RenderContextBlock`; config keys extend the `mnemonic:` block the way `extraction.llm` did. Note: the Cursor stop hook today always prints `{}`; it gains a conditional JSON output, which is the same shape `cursor-policy.sh` already uses. Note: `setup opencode|kilo` gains a plugin-file copy next to the existing `hooks.yaml` copy and `upsertPluginKey` call.

## Approaches Considered

- **Chosen:** host agent as observer, one server-gated Memory Checkpoint per stop/idle (ADR-0022) — no key, no dependency, one loop guard and one prompt for every harness.
- **Rejected:** external LLM endpoint called after every tool call (claude-mem's model) — user declined the key/cost; it is also the seam-wiring decision the repo has deferred twice.
- **Rejected:** heuristics only — `promote.go` already is this; the quality gap is the problem.
- **Rejected within the chosen approach:** prompting after every tool call — one extra agent turn per call; and an unconditional yaml `command` action at `session.idle` — loops, because the idle after the agent's reply fires it again.

## Requirements

1. **Checkpoint claim gating:** `POST /sessions/{id}/checkpoint/claim` answers `{due, prompt, reason}`; due iff `checkpoint.enabled`, events-since-last-write ≥ `min_events`, and no claim within `cooldown`; a claim stamps `sessions.checkpoint_claimed_at`.
   - **Current:** no route; nothing tracks claims.
   - **Target:** route + two session columns (`checkpoint_claimed_at`, `last_memory_write_at`), the latter updated by `Save` (session-scoped) and `SessionSummary`.
   - **Acceptance:** G1, G2, G14.
   - **Acceptance scenario:** `claim-due-after-enough-events` → `acceptance.feature`

2. **Checkpoint prompt content:** the prompt is rendered server-side from the store.
   - **Current:** no prompt; the memory protocol lives in a rule file the agent may or may not follow.
   - **Target:** `checkpoint.RenderPrompt(digest, existingTitles, cfg)` with a bounded digest (newest events first, "N older omitted"), the allowed types (`decision|bugfix|discovery|pattern|learning`), `max_observations`, the summary sections (Goal, Discoveries, Accomplished, Next Steps, Relevant Files), the session id to pass to `mem_save`/`mem_session_summary`, and "reply in one line: `memory saved: N observations`".
   - **Acceptance:** G3; digest never contains Private Span text.
   - **Acceptance scenario:** `prompt-lists-new-events-and-existing-titles` → `acceptance.feature`

3. **Cursor stop follow-up:** `cursor-session-end.sh` claims on `stop` and prints `{"followup_message": prompt}` when due and `loop_count < 2`; `{}` otherwise, on `sessionEnd`, and on any error within 2 s. `hooks-cursor.json` and `setup cursor` set `loop_limit: 2` on the stop entry.
   - **Current:** the hook always prints `{}`.
   - **Target:** conditional output via `tool-call-capture.js checkpoint` mode (same pattern as `policy` mode).
   - **Acceptance:** G4, G5.
   - **Acceptance scenario:** `stop-returns-followup-when-due` → `acceptance.feature`

4. **OpenCode/Kilo idle prompt:** a Skillgrid plugin (`plugins/opencode/skillgrid-checkpoint.ts`, mirrored for kilo) claims on `session.idle` and prompts the session when due; `setup opencode|kilo` copies it into the harness plugin directory and registers it once.
   - **Current:** only `opencode-yaml-hooks` is registered; the yaml `session.idle` hook runs compact/usage.
   - **Target:** plugin installed and idempotently registered; the yaml hook unchanged.
   - **Acceptance:** G6; G7 manual.
   - **Acceptance scenario:** `setup-installs-checkpoint-plugin` → `acceptance.feature`

5. **Memory Index at session start:** `skillgrid prime` appends `## Memory` rendered by `session_inject.RenderIndex`: `inject.summaries` recent session summaries (title, one line, date) then `inject.observations` observation lines (pinned first, then newest; `#id type title · date · ~tok`), capped at `inject.max_tokens` by dropping the oldest observation lines first and stating the omitted count; footer names `mem_get_observation`, `mem_timeline`, `mem_search`. Empty store → no section.
   - **Current:** `prime` prints step/branch/files/impact and one protocol sentence.
   - **Target:** index section present when the store has content; `prime.txt` page updated accordingly.
   - **Acceptance:** G8.
   - **Acceptance scenario:** `prime-lists-summaries-and-observation-index` → `acceptance.feature`

6. **One project resolver:** hooks send `directory` and omit `project`; the HTTP handlers resolve with `project.Resolve(directory)` when `project` is absent; `tool-call-capture.js` drops its own heuristic and the `active-project` pin. `prime`, MCP, and HTTP agree on the id for the same directory.
   - **Current:** two resolvers; this repo's tool events and its memory live in different stores; a cross-repo `prime` re-pins hook writes.
   - **Target:** server-side resolution; `MNEMONIC_PROJECT` still overrides; the pin file is no longer read by hooks.
   - **Acceptance:** G9; existing tool-call tests still pass with `project` omitted.
   - **Acceptance scenario:** `prime-and-hooks-share-one-project` → `acceptance.feature`

7. **Private spans never stored:** `memory.StripPrivate(s)` removes `<private>…</private>` (case-insensitive, unterminated tag removes to end) and is applied in the tool-call handler, `Save`/`SaveWithAction`, `SessionSummary`, `SavePrompt`, and the checkpoint digest; `tool-call-capture.js` keeps its hook-side strip.
   - **Current:** hook replaces spans with `[REDACTED]`; server stores whatever arrives via MCP; inject-time filter hides observations whose title contains "private".
   - **Target:** server-side strip on every write path; the inject-time filter stays for old rows.
   - **Acceptance:** G10, G11.
   - **Acceptance scenario:** `tool-call-with-private-span` → `acceptance.feature`

8. **Live observations in the Sessions view:** `ToolTimeline` interleaves the session's observations (type badge, title, expand to full content) with tool calls by time; `activity` SSE frames with a matching session id append live; the sessions list shows an `observations` count.
   - **Current:** timeline shows tool calls and policy notes only; `activity` frames are ignored by the Sessions page.
   - **Target:** `GET /sessions/{id}/events` response gains an `observations` array (id, type, title, created_at, token estimate); list items gain `observations`; UI renders both.
   - **Acceptance:** G12.
   - **Acceptance scenario:** `observation-appears-live` → `acceptance.feature`

9. **Config:** `mnemonic.checkpoint {enabled: true, min_events: 5, cooldown_minutes: 10, max_observations: 5}` and `mnemonic.inject {summaries: 5, observations: 20, max_tokens: 800}` in `.skillgrid/config.yaml`; invalid values fall back to defaults with a warning.
   - **Current:** `mnemonic:` has `enabled`, `data_dir`, plus `extraction`/`dedup`/`improve` blocks read by the Go config loader.
   - **Target:** two new blocks with validation; documented in the user guide.
   - **Acceptance:** G13, G14.
   - **Acceptance scenario:** `defaults-apply-without-keys` → `acceptance.feature`

## Implementation Decisions

- **Modules to build/modify:**
  - New `internal/mnemonic/checkpoint` (Go): `Decide(state, cfg, now) (due bool, reason string)`, `BuildDigest(events, maxChars)`, `RenderPrompt(in PromptInput) string`. Pure functions; the HTTP handler owns I/O.
  - Memory service: `SessionCheckpointState(ctx, id) (events int, lastWrite, lastClaim time.Time, titles []string)`, `ClaimCheckpoint(ctx, id, at)`, and `last_memory_write_at` maintenance in `Save`/`SessionSummary`. Migration `049_session_checkpoint.sql`.
  - HTTP: `handleCheckpointClaim` (POST), `GET /sessions/{id}/events` gains `observations`, sessions list gains `observations`; `projectFromRequest` falls back to `project.Resolve(directory)`.
  - `session_inject.RenderIndex(summaries, observations, cfg) string` + `prime` wiring.
  - `memory.StripPrivate` + call sites.
  - Hooks: `tool-call-capture.js` gains `checkpoint` mode and drops `resolveProject`; `cursor-session-end.sh` emits the follow-up on `stop` only (Cursor sends `hook_event_name`); `hooks-cursor.json` + `setup/cursor.go` add `loop_limit: 2`.
  - Plugin: `plugins/opencode/skillgrid-checkpoint.ts` (also kilo); `setup/opencode.go` + `setup/kilocode.go` copy and register it.
  - UI: `ToolTimeline` observation rows; `SessionsPage` consumes `activity` frames; `sessions/api.ts` types.
  - Config: `config.Load` new blocks; `openapi.yaml` new route and fields; `docs/user-guide/04-hooks.md` + `05-memory-and-indexing.md`.
- **Interfaces:**
  - `POST /sessions/{id}/checkpoint/claim?project=|directory=` → `200 {due: bool, prompt: string, reason: "below_min_events"|"cooldown"|"disabled"|"unknown_session"|""}`. Always 200 with a body; unknown session → `due:false`.
  - `GET /sessions/{id}/events` adds `observations: [{id, type, title, created_at, tokens}]`.
  - Hook stdin/stdout: Cursor `stop` → `{followup_message?}`; the capture script prints nothing when not due and the shell wrapper prints `{}`.
  - Plugin: `session.idle` → `POST claim` (1.5 s timeout) → if due `client.session.prompt({sessionID, parts:[{type:"text", text: prompt}]})`.
- **Data flow:** tool events → `session_events` (as now) → stop/idle → claim → due → prompt → agent → `mem_save`×N + `mem_session_summary` → `observations`/`sessions.summary` (+ `last_memory_write_at`) → `activity` SSE → Sessions view; next `prime` → `RenderIndex` reads `RecentContext` + `RecentObservations`/`Pinned`.
- **Error handling:** every hook/plugin path is fail-open (timeout 1.5–2 s, any error → no prompt, exit 0). Claim on a disabled checkpoint or unknown session returns `due:false` with a reason, never 4xx/5xx for the hook's sake. `RenderIndex` on store error returns "" so `prime` degrades to today's output. Config validation logs and falls back.
- **Dependencies:** none new. The checkpoint package depends on memory types only; the HTTP layer adapts. The UI uses the existing SSE client.

## Testing Decisions

- **What makes a good test:** drive the HTTP seam and the rendered strings; assert on responses, rows, and output text, not on internal counters.
- **Modules to test:** `checkpoint` (Decide/BuildDigest/RenderPrompt table tests), HTTP claim handler (`policy_test.go` style with `newToolCallsServer`), `StripPrivate` + write paths, `RenderIndex` budget, config defaults/validation, `setup` plugin install idempotence, `scripts/test-hooks.mjs` new `checkpoint` and `private` groups (fake server on a free port), Vitest for timeline interleave and live append (`SessionsPage.test.tsx` already fakes SSE).
- **Prior art:** `policy_test.go`, `toolevents_test.go`, `cursor_test.go` (`TestUpsertCursorHooks`), `session_inject/summary_test.go`, `SessionsPage.test.tsx` live-policy test.
- **Edge cases:** claim race (two stops within a second → one due, one cooldown); session with events but no summary ever; cooldown boundary exactly at 10 min; 400+ events digest cap; unterminated `<private>`; `prime` with store error; pinned observation also among the newest (no duplicate line); observation count when policy rows exist (exclude, as tool_calls does).

## Impact on Global Docs

- `.skillgrid/artifacts/00-prd.md`: None.
- `.skillgrid/ASSUMPTIONS.md`: in-force row 0022 added; highest sequence 0022. New VERIFIED fact to add on execution start: "Project identity is resolved server-side for hook writes (one resolver)".
- `.skillgrid/ARCHITECTURE.md`: file does not exist; nothing to update.

## Clarity Report

| Dimension           | Score | Min  | Status | Notes                              |
|---------------------|-------|------|--------|------------------------------------|
| Goal Clarity        | 0.90  | 0.75 | ok     | "like claude-mem" decomposed into five behaviors, all selected |
| Boundary Clarity    | 0.85  | 0.70 | ok     | external LLM, per-call compression, prompt capture, Memory page out |
| Constraint Clarity  | 0.85  | 0.65 | ok     | no dependency, fail-open, serial-dev override recorded |
| Acceptance Criteria | 0.80  | 0.70 | ok     | 14 gates, 9 requirements           |
| **Clarity**         | 0.15  | ≤0.20| pass   |                                    |

**Interview log:**

| Round | Question summary | Decision locked |
|-------|------------------|-----------------|
| 1 | Which claude-mem behaviors; who compresses; injection size | all five behaviors; the host agent (no external LLM); lean budget 5 summaries + 20 observations (~800 tokens) |
| 2 | Approve design; serial-development constraint | approved as designed, `checkpoint.enabled` defaults true; override — execute right after the spec commits |

## Open Questions & Assumptions

- **Question:** should this repository's existing `skillgrid.sqlite` tool events be merged into `aiskillgrid.sqlite` once the resolver is unified? Operator action via `mem_merge_projects`; not code. Default: leave as-is, document.
- **Assumption:** Kilo loads OpenCode-shaped plugins from `~/.config/kilo/plugin/` as it loads `hooks.yaml` from `~/.config/kilo/hook/`. Verified during execution (G7 manual); if not, Kilo keeps yaml hooks only and the briefing is amended.
- **Assumption:** the agent's follow-up turn is acceptable UX for the user (it was chosen knowing it costs plan tokens); thresholds keep it to roughly one per ten minutes of active work.
- **Assumption:** Cursor's `stop` payload for a hook installed at user level carries `conversation_id` as the session id, as the existing `cursor-session-end.sh` already assumes.

## Decisions (ADR)

- `.skillgrid/artifacts/04-adr-0022-host-agent-memory-checkpoint.md` — the host agent is the memory observer; checkpoints are server-gated; external-LLM seams stay unwired.

## Terms

- `Memory Checkpoint`, `Checkpoint Claim`, `Memory Index`, `Private Span` — added to `.skillgrid/artifacts/02-technical-terms.md`. `Session Context Injection`, `L1 Summary`, `Auto-Prepend`, `Context Block` — relied on, unchanged.
