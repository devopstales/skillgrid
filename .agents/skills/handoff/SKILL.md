---
name: handoff
description: Compact the current conversation into a handoff document for another agent to pick up. Use when ending a session, switching agents, or the user asks for a handoff / summary / context transfer.
argument-hint: "What will the next session focus on?"
---

Write a handoff document summarising the current conversation so a fresh agent can continue the work. Save to the OS temporary directory (e.g. `/tmp` on Linux/macOS, `%TEMP%` on Windows), NOT the current workspace.

## Required sections

1. **Context** — What was the goal, what was accomplished, and what remains.
2. **Current state** — Exact branch, commit, worktree, or session state if relevant.
3. **Key decisions** — Architecture choices, tradeoffs accepted, conventions established.
4. **Artifacts** — Reference existing artifacts by path/URL instead of duplicating content:
   - SDD specs/plans: `docs/skillgrid/changes/<NNN-slug>/`
   - Issues: GitHub/Jira URLs
   - Code: relative paths from repo root
5. **Suggested skills** — Name which skills the next agent should load via the Skill tool. Prioritise:
   - `mnemonic-memory` (always active, but remind them to call `mem_session_start` first)
   - Domain skills matching the next work (`sdd-apply`, `verification`, `tdd`, etc.)
   - Process skills if the next step is exploratory (`investigate`, `brainstorming`, `questioning`)
6. **Next steps** — Concrete, ordered actions for the next session.
7. **Gotchas** — Non-obvious gotchas, environment quirks, or things that broke.

## Rules

- **Do not duplicate** content already captured in specs, plans, ADRs, issues, commits, or diffs. Reference them by path or URL.
- **Redact secrets**: API keys, passwords, tokens, PII. If unsure, redact.
- **Tailor to arguments**: If the user passed arguments, treat them as the next session's focus and weight the doc accordingly.
- **Be concise**: A handoff is a jump rope, not a history book. Prefer bullet points over prose.

## Mnemonic integration

If the conversation produced memory-worthy findings:
- Remind the next agent to call `mem_context` and `mem_search` before asking the user to repeat context.
- If you just made a decision or fixed a bug, call `mem_save` now so it persists across sessions.
