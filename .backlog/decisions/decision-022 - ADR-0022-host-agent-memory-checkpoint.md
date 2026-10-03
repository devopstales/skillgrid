---
id: decision-022
title: "ADR-0022 The host agent is the memory observer; checkpoints are server-gated"
date: "2026-10-02"
status: "accepted"
---

Source: `.skillgrid/artifacts/04-adr-0022-host-agent-memory-checkpoint.md`

# The host agent is the memory observer; checkpoints are server-gated

---
status: "accepted"
supersedes: none
date: 2026-10-02
---

## Context and Problem Statement

Every tool call Cursor, OpenCode, and Kilo make is now recorded in `session_events`, but almost none of it becomes memory. `promote.go` writes an observation only for a failed call, a file written twice, or text containing "we decided"; the end-of-session summary exists only when the agent remembers to call `mem_session_summary`. The next session starts with `skillgrid prime` (last step, branch, changed files) and a sentence telling the agent that `mem_search` exists. claude-mem closes this gap with a second model: a worker that calls an LLM after every tool call to compress it into a typed observation, and at stop to write a summary, then injects an index of both at session start. Skillgrid has LLM seams (`ExtractionLLM`, `DedupLLM`, `DreamLLM`, `layer.LLM`, `AskLLM`) but no backend is wired, and the locked constraints forbid a new dependency without an ADR. Who compresses, when, and how the harness is asked are hard to change once prompts, hook contracts, and plugins are installed on users' machines.

## Considered Options

- An external LLM endpoint (OpenAI-compatible, Ollama, Anthropic) called by `skillgrid serve` over stdlib HTTP after every tool call, as claude-mem does
- The host agent itself, asked once per *memory checkpoint*: the stop/idle hook asks the server whether a checkpoint is due; when it is, the harness is given a follow-up prompt that makes the agent save observations and a summary through the existing MCP tools
- No model at all: widen the deterministic promotion heuristics

## Decision Outcome

Chosen option: "the host agent, asked once per server-gated memory checkpoint", because it needs no key, no new dependency, and no second model's cost; the agent has the full conversation context that a tool-call digest lacks; and gating the checkpoint in the server keeps one loop guard and one prompt for every harness.

- A checkpoint is due for a session when at least `min_events` tool events (default 5) arrived after the session's last observation or summary write, and no checkpoint was claimed in the last `cooldown` (default 10 minutes). `POST /sessions/{id}/checkpoint/claim` answers `{due, prompt}` and records the claim; a claim that the agent ignores is not repeated before the cooldown.
- The prompt is rendered by the server from the store: a digest of the new events, the titles already saved for the session, and the instruction to save at most `max_observations` (default 5) typed observations with `mem_save` and one structured summary with `mem_session_summary`, then answer in one line.
- Cursor receives the prompt as the `stop` hook's `followup_message`, only while `loop_count < 2`, and never from `sessionEnd`. OpenCode and Kilo receive it from a Skillgrid plugin on `session.idle` that calls the claim route and prompts the session; the yaml hooks cannot gate an LLM action, so they do not carry the checkpoint.
- The deterministic promotion heuristics stay as the floor for sessions that never reach a checkpoint. The external-LLM seams stay unwired; wiring one is a separate ADR.
- Everything fails open: no server, a timeout, or a malformed answer means no prompt and an unchanged agent loop.

### Consequences

- Good, because memory quality comes from the model the user already pays for and trusts, with no key management and no per-call network cost.
- Good, because one claim route and one prompt serve every harness; the loop guard is tested once in Go.
- Bad, because each checkpoint is one extra agent turn on the user's plan, visible in the conversation. The thresholds exist to keep that rare.
- Bad, because compression happens per checkpoint, not per tool call; a session aborted before its first checkpoint leaves only the heuristic floor.
- Bad, because the agent may ignore or half-follow the prompt; the cooldown bounds the retry, and the summary stays missing rather than being fabricated.
