# Tasks — Shared LLM Provider

> **STATUS:** `in-progress` (2026-10-03)

> Sliced from `.skillgrid/specs/2026-10-02-mnemonic-llm-provider/blueprint.md`.
> Vertical tracer-bullet tickets, dependency-ordered, sized for one fresh agent context window.

**Execution choice:** Subagent-Driven when this change is current (same as project-init unless overridden).
**Queued behind:** `2026-10-02-mnemonic-project-init` (memory-checkpoint must also clear). Do not take `state.yaml` `current_change` until then. per ASSUMPTIONS.md § Locked constraints

## Epic Summary

One OpenAI-compatible Completer attaches ask/dedup/extraction/dream at boot; `skillgrid install` runs natural Local/External provider setup (ensure Ollama + wire, or wire external) for llm + embedder. Closes task-029. per ADR-0023.

## Delivery Strategy

| Field | Value |
|-------|-------|
| Estimated changed lines | 700–1000 |
| 400-line budget risk | High — prefer two work units if PR pressure |
| Chained PRs recommended | Optional: (1) Completer+attach (2) install provider |
| Suggested split | single PR on release/2 if serial queue allows |
| Delivery strategy | auto-chain |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

Ruling: auto-chain plus the plan's suggested single series on release/2, and the serial queue is clear (project-init reflected). One commit series, two revertable work units. Cost if wrong: the diff may exceed the 400-line review budget and need a split before ship.

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Rollback boundary |
|------|------|-----------|----------------------|-------------------|
| 1 | Completer + config + AttachSharedLLM + fail-open | PR A | `go test ./skillgrid-cli/internal/mnemonic/llm ./skillgrid-cli/internal/mnemonic/config ./skillgrid-cli/internal/mnemonic/service -count=1 -run 'TestComplete\|TestLLM\|TestAttach\|TestLLMError\|TestAttached'` | revert unit 1 commits |
| 2 | Install provider setup + close task-029 | PR B | `go test ./skillgrid-cli/internal/install -count=1 -run TestSetupProvider` | revert unit 2 commits |

## Tickets

### TICKET-01 — Completer + mnemonic.llm config

- **Tracker ID:** TASK-039.01
- **Parent:** TASK-039

**Blueprint tasks:** 1–2  
**DoD:** G1–G4 gates green; no new LLM SDK in go.mod  
**Acceptance:** `happy path llm config defaults off`, `happy path openai-compatible complete succeeds`

### TICKET-02 — AttachSharedLLM + fail-open

- **Tracker ID:** TASK-039.02
- **Parent:** TASK-039

**Blueprint tasks:** 3–4  
**Depends on:** TICKET-01  
**DoD:** G5–G8 green; flags off use floors  
**Acceptance:** `happy path attach wires all seams from one client`, `happy path llm error fails open to floors`

### TICKET-03 — Natural install provider setup

- **Tracker ID:** TASK-039.03
- **Parent:** TASK-039

**Blueprint tasks:** 5  
**Depends on:** TICKET-01 (home llm keys); can parallel TICKET-02 after config lands  
**DoD:** G10–G11 green; `--yes` local ensure; `--skip-provider` no-op  
**Acceptance:** `happy path install yes ensures local ollama and wires config`

### TICKET-04 — Close task-029

- **Tracker ID:** TASK-039.04
- **Parent:** TASK-039

**Blueprint tasks:** 6  
**Depends on:** TICKET-02 (and ideally TICKET-03)  
**DoD:** G9; backlog task Done  
**Acceptance:** `happy path task-029 superseded by this change`
