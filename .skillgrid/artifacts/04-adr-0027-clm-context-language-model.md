# Context Harness implements the Context Language Model

---
status: "accepted"
supersedes: none
date: 2026-10-06
---

## Context and Problem Statement

The Context Harness (ADR-0025) captures large tool output and provides structured query, but it does not let the agent *manage* its own context. The agent still sees the full raw history on every request, and context bloat accumulates until OpenCode's automatic compaction fires — which summarizes the raw transcript and discards any edits the agent may have made.

pi-clm (a Pi extension implementing the Context Language Models paper, arXiv:2609.37725) solves this by giving the model write access to its own context: the effective context is mirrored to a file, the model edits the file with ordinary tools, and the edited version becomes the next request. The raw session history is never rewritten. Two things run without the model: an **overflow guard** (withhold the oldest tool results when the estimated request exceeds the budget) and **calibration** (correct the size estimate against the provider's own token count so dense content does not slip past the budget).

The integration question is the split of labor. The Go harness is a long-running process with a persistent store, a token estimator (`session_inject.EstimateTokens`), and the capture path that already sees turn boundaries (the `checkpoint` mode of `tool-call-capture.js` is the Cursor stop hook; OpenCode's idle hook is the same path). The Node/OpenCode plugin is the only component in the request path that can transform the outgoing model call: OpenCode's `context` hook (v2 plugins) modifies `event.system` / `event.messages` / `event.tools` immediately before model dispatch, and "changes affect only the outgoing model call, not persisted history." That is the exact seam pi-clm needs.

The hard trade-off: the mirror must be a real file the model can edit with its ordinary file tools (Edit/Write/bash), but the budget math, the overflow-withhold decision, and the revision persistence belong in Go — they are deterministic, testable, and reusable. The mirror is ephemeral (temp dir, removed at shutdown); the revision is durable (SQLite). The model edits the ephemeral file; Go persists the durable record.

## Considered Options

- Full CLM in the Node plugin (pi-clm parity): the plugin computes budget, overflow, calibration, and persists the revision via HTTP
- Go owns state and decisions, Node owns the request path: Go computes budget, overflow-withhold, and calibration; persists the revision. The Node plugin renders the mirror before each request (via the `context` hook) and applies Go's withhold decision. The capture path reads the model's mirror edit at turn-end.
- No mirror: only the overflow guard and calibration (the "runs without the model" half), no model-editable context

Chosen option: "Go owns state and decisions, Node owns the request path," because the mirror must be a file the model edits with ordinary tools (so it lives in the OS temp dir, rendered by the plugin), but the budget math and the overflow decision are deterministic Go concerns that reuse the existing token estimator and the provider counts from the capture. The split keeps the Go side testable (pure functions over a message list) and the Node side thin (render, apply, read).

- The mirror is a `0600` file in the OS temp dir, written by the Node plugin before each model request (via the `context` hook). Format: a `[[LIVE_CONTEXT version=1 revision=N document=<nonce> baseline=<digest>]]` header followed by `[[CTX_TURN document=<nonce> index=I role=R id=<id> protected=false]]` blocks. The nonce is stable between accepted edits (derived from session + anchor digest + revision number), so ids the model reads on one call are valid on the next.
- The model edits the mirror with ordinary file tools during its turn. At turn-end, the capture path (the `checkpoint` mode of `tool-call-capture.js`, extended) reads the mirror file and POSTs it to Go. Go validates (nonce match, legal message sequence, tool-call-group repair), persists the revision to a `context_revisions` table, and activates it. The next request uses the activated revision.
- The overflow guard is computed in Go: when the estimated request exceeds `budget − reserve`, the oldest tool results after the last edit are withheld. The withhold decision (which tool-result ids to replace with one-line notes) is stored on the revision and applied by the Node plugin when it renders the mirror.
- Calibration: the size estimate starts at 4 chars/token and is corrected against the provider's own token count for each request (available in the capture path). The corrected factor is stored per session.
- CLM is **opt-in**: off by default. All CLM config lives in the `.skillgrid/config.yaml` `clm:` block: `enabled` (default `false`), `budget` (default model window), `reserve` (default 2048), `reminders` (default `50/75/90%`), `guard` (default `true`), `cap` (default `off`). Env overrides: `SKILLGRID_CTX_CLM` (on/off), `SKILLGRID_CTX_CLM_BUDGET`, `SKILLGRID_CTX_CLM_RESERVE`, etc.
- v1 scope: mirror + overflow guard + calibration. No TUI panel, no model-driven compact (`/ctx-compact`), no per-project budget file beyond the `clm:` block.

### Consequences

- Good, because the agent manages its own context: it can shorten stale tool output, delete or reorder blocks, insert notes, and grow a scratchpad — all with ordinary file tools. The raw history is never rewritten.
- Good, because the Go side is testable: the budget math, the overflow-withhold decision, and the mirror validation are pure functions over a message list, unit-testable without a Node plugin or an OpenCode session.
- Good, because the Node plugin is thin: render the mirror, apply Go's withhold decision, read the model's edit at turn-end. It does not recompute the budget.
- Good, because CLM is opt-in and config lives in `config.yaml` — no behavior change on upgrade, and the config is reviewable.
- Bad, because the mirror is a prompt-injection surface: injected text can induce the model to rewrite its own constraints. The system prompt stays out of the mirror; authored roles are lowered to plain text. This is a known limitation, not a fix.
- Bad, because the turn-end mirror read adds a file read + POST to the capture path. The fast path (model did not edit the mirror) must be cheap: the capture path compares the mirror's mtime/hash against the last render and skips the POST when unchanged.
- Bad, because the `context` hook is an OpenCode v2 plugin hook. The Node plugin must be shipped as a `.opencode/plugins/` module, and the config must be read at plugin load. This is a new shipping surface (the plugin is Hub Content, installed by `skillgrid install`).
