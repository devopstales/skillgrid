# Source of truth: the repo is authoritative; Backlog task ID ↔ commit SHA is the linkage

---
status: "accepted"
supersedes: none
date: 2026-09-29
---

## Context and Problem Statement

The Claude Academy AI-native SDLC playbook names three coexistence modes for projects with legacy trackers: repo-as-truth, legacy-as-truth, and linkage-as-minimum-bar. skillgrid was running repo-as-truth by accident — nothing had ever locked it, so a future reader could not tell whether Backlog.md or an in-session memory was authoritative, and the commit chain was the audit trail without anyone having said so.

## Decision Outcome

The repo is the single source of truth for project state: `.skillgrid/ASSUMPTIONS.md` (facts, the in-force ADR index, locked constraints), the ADR records under `.skillgrid/artifacts/04-adr-*.md`, the spec-zone artifacts under `.skillgrid/specs/`, the committed `state.yaml`, and the git history. When an external tracker, a session memory, or a dashboard disagrees with a committed artifact, the committed artifact wins and the other source is reconciled to it. Backlog.md tasks live under `.backlog/` in the repo, so the tracker is already in-repo. The required linkage is Backlog task ID ↔ commit SHA: every commit that closes a task names that task ID in the conventional-commit subject or body (`[skillgrid-context]` block), so the audit chain `task → commit → diff` is recoverable from git alone.

### Consequences

- Good, because any reviewer, agent, or future session can reconstruct the chain from the repo alone.
- Bad, because in-session memory is subordinate: a saved observation that contradicts a committed artifact is stale and must be updated. A merge that closes a task without naming its ID breaks the chain.

## Revisit

When a hosted Backlog sync re-introduces a second authoritative tracker, or when the `[skillgrid-context]` block gains a machine-checkable task-reference field.
