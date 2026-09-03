---
name: dispatching-parallel-agents
description: Use when an SDD phase or debugging task has 2+ independent work items that can be executed without shared state or sequential dependencies. Provides the decision protocol for fanning out work to parallel sub-agents and the per-agent prompt structure.
license: MIT
metadata:
  author: skillgrid
  version: "1.0"
  source: derived from superpowers dispatching-parallel-agents, adapted for skillgrid's Mnemonic + orchestrator model
---

# Dispatching Parallel Agents

You delegate tasks to specialized sub-agents with **isolated context**. By precisely crafting their instructions and context, you ensure they stay focused and succeed at their task. They never inherit your session's context or history — you construct exactly what they need. This also preserves your own context for coordination work.

When you have multiple unrelated failures (different test files, different subsystems, different bugs), investigating them sequentially wastes time. Each investigation is independent and can happen in parallel.

**Core principle:** dispatch one sub-agent per independent problem domain. Let them work concurrently.

## When to Use

```
Multiple unrelated work items?
├── no  → single agent handles all
└── yes → Are they independent (root-cause level)?
         ├── no  (related; fixing one may fix others) → single agent investigates together
         └── yes → Can they run truly in parallel (no shared state, no shared files)?
                  ├── no  (shared files, shared resource, single Mnemonic topic_key) → sequential
                  └── yes → PARALLEL DISPATCH
```

**Use when:**
- 3+ test files failing with different root causes
- Multiple subsystems broken independently
- Each problem can be understood without context from others
- No shared state between investigations (no shared files, no shared resources, no shared `mem_save` `topic_key`)

**Don't use when:**
- Failures are related (fixing one might fix others)
- Need to understand full system state to triage
- Agents would edit the same files
- Agents would write to the same Mnemonic `topic_key` (memory clobber risk)
- Exploratory debugging — you don't know what's broken yet

## The Pattern

### 1. Identify Independent Domains

Group work by what's broken / what needs doing:
- File A tests: Tool approval flow
- File B tests: Batch completion behavior
- File C tests: Abort functionality

Each domain is independent — fixing tool approval doesn't affect abort tests.

### 2. Create Focused Agent Tasks

Each agent gets:
- **Specific scope:** one file, one subsystem, one step, or one test area
- **Clear goal:** the observable outcome (tests pass, RED test written, file created, etc.)
- **Constraints:** what they must NOT change ("do not edit other files", "do not touch the other adapter")
- **Expected output:** structured return — summary of root cause / changes / blockers
- **Mnemonic isolation:** if the agent writes to Mnemonic, give it a unique `topic_key` per agent (e.g. `sdd/<NNN-slug>/parallel/<domain>`). Never have two parallel agents write the same `topic_key` — the second upsert silently overwrites the first.

### 3. Dispatch in Parallel

**Mechanical rule:** issue all parallel dispatches in a single response. Multiple `task` calls in one response = parallel execution. One `task` per response = sequential.

```text
task(description: "Fix agent-tool-abort tests", subagent_type: "general", prompt: "...")
task(description: "Fix batch-completion tests", subagent_type: "general", prompt: "...")
task(description: "Fix tool-approval race tests", subagent_type: "general", prompt: "...")
# All three run concurrently.
```

### 4. Review and Integrate

When agents return:
1. **Read each summary** — what they changed, what they found
2. **Check for conflicts** — same-file edits, contract drift, vocabulary drift
3. **Run full suite** — agents can make systematic errors that cancel in isolation
4. **Spot-check** — at least one acceptance scenario per agent

## Agent Prompt Structure

Good prompts are:
1. **Focused** — one clear problem domain
2. **Self-contained** — all context the sub-agent needs (never your session history)
3. **Specific about output** — what should the agent return?

```markdown
Fix the 3 failing tests in src/agents/agent-tool-abort.test.ts:

1. "should abort tool with partial output capture" — expects 'interrupted at' in message
2. "should handle mixed completed and aborted tools" — fast tool aborted instead of completed
3. "should properly track pendingToolCount" — expects 3 results but gets 0

These are timing/race condition issues. Your task:

1. Read the test file and understand what each test verifies
2. Identify root cause — timing issues or actual bugs?
3. Fix by:
   - Replacing arbitrary timeouts with event-based waiting
   - Fixing bugs in abort implementation if found
   - Adjusting test expectations if testing changed behavior

Do NOT just increase timeouts — find the real issue.

## Project standards (auto-resolved)

[compact rules injected by orchestrator — see _shared conventions]

## Mnemonic isolation

If you write memory, use topic_key: `sdd/<NNN-slug>/parallel/abort-domain` (this is YOUR slot; other parallel agents have their own).

## Return format

Summary of what you found and what you fixed. Cite file:line for every change.
```

## Common Mistakes

| Mistake | Fix |
|---|---|
| "Fix all the tests" (too broad) | One test file or subsystem per agent |
| "Fix the race condition" (no context) | Paste the error messages and test names |
| No constraints (agent refactors everything) | "Do NOT change production code" or "Fix tests only" |
| "Fix it" (vague output) | "Return summary of root cause and changes" |
| Same `topic_key` for two parallel agents (silent clobber) | Unique `topic_key` per agent (e.g. `…/parallel/<domain>`) |
| Edits to the same file (merge conflict) | Pre-allocate files per agent in the prompt |

## When NOT to Use

- **Related failures:** fixing one might fix others — investigate together first
- **Need full context:** understanding requires seeing the entire system
- **Exploratory debugging:** you don't know what's broken yet
- **Shared state:** agents would interfere (editing same files, using same resources, same Mnemonic key)

## Integration with skillgrid

This skill is the **decision protocol** the orchestrator uses to decide whether to fan out:

- **`sdd-apply`** already has a similar split (`simple-execution` vs `subagent-execution`). Use the rule above to pick: 2+ independent work items with no shared state → fan out (one `subagent-execution` slot per domain). Otherwise → single inline route.
- **`sdd-explore`** can use this for parallel investigation of multiple subsystems in a large codebase.
- **Debugging parallel failures** (e.g. flaky tests across N files): dispatch one agent per file with a shared "no cross-file edits" constraint.

## References

- [../subagent-execution/SKILL.md](../subagent-execution/SKILL.md) — the dispatch loop that owns fan-out for a full SDD step.
- [../simple-execution/SKILL.md](../simple-execution/SKILL.md) — the inline per-task loop (no fan-out).
- [../_shared/conventions/mnemonic-memory.md](../_shared/conventions/mnemonic-memory.md) — `topic_key` isolation rules; the orchestrator injects these into every sub-agent prompt.
