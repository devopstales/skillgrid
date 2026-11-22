---
type: Reference
title: "Glossary — Business"
---

# Glossary — Business

One meaning per term. Do not use a synonym in its place.

## People

| Term | Definition | Use when | Avoid |
|---|---|---|---|
| **Programmer** | The person directing SkillGrid: owns the repository, states the idea, answers gates, approves ship. The role this skill addresses. | Any instruction, question, or checkpoint addressed to the person running the skill. | "user", "author", "operator", "developer" (unless the host's own UI uses it). |
| **User** | The person who runs the finished program. A role in the product's world, not in the build. | Describing end-user behavior, UX, "user-facing", simulated users in blind validation. | "programmer" (the builder), "customer" unless that is the business term. |
| **Maintainer** | A person who approves merge or release when that authority is not the programmer's. | Release approval, protected branches, upstream projects. | "user", "owner" (ambiguous with code ownership). |
| **Reviewer** | An agent or person that reads a diff or artifact and returns findings, independent of the producer. | Code review, blueprint review, adversarial audit. | "editor", "critic" when a reviewer is meant. |

## Work

| Term | Definition | Use when | Avoid |
|---|---|---|---|
| **Program** | The software being built. | The product scope, README, delivery. | "book", "manuscript", "sourcecode" (one word). |
| **Change** | One self-contained pipeline unit under `.skillgrid/change-specs/<topic>/`, from briefing to archive. | Referring to the unit of work end to end. | "ticket" (a smaller slice), "PR" (an integration artifact). |
| **Work unit** | A slice of a change that can ship as one focused commit or PR, with its own test and rollback boundary. | Slicing, execution waves, PR planning. | "chapter", "step" (overloaded with SDD steps). |
| **Ticket** | The tracker record for a work unit (Backlog.md, GitHub, GitLab, Jira). | Ticketing, dependency edges, status transitions. | "task" when the tracker record is meant; "change" (broader). |
| **Phase** | A named stage of the pipeline (`briefing`, `blueprint`, `acceptance`, `slicing`, `execution`, `qa`, `review`, `ship`, `reflect`). | State, routing, checkpoints. | "step" (tickets and SDD steps are different). |
| **Gate** | A persisted decision point in `state.yaml` (`blueprint-validation`, `user-gate-slice`, `qa-verdict`, `ship-decision`) that waits for the programmer. | Check-in, resume, blocking. | "checkpoint" (the screen shown at a gate), "pause" (not persisted). |
| **Artifact** | A file a phase persists (`briefing.md`, `blueprint.md`, `tasks.md`, `acceptance.feature`, `report.md`, `review.md`). | Cross-referencing phase outputs. | "document" (too generic). |
| **Ship** | Integrate verified work to its base branch (merge, PR, or keep) and move the change folder to `change-archive/`. | The integration phase. | "publish", "release" (release mechanics are out of scope). |

## Project terms

| Term | Definition | Use when | Avoid |
|---|---|---|---|
| **Tier (tooling tier)** | Who creates/manages worktrees — harness-native → worktrunk → git fallback. | Worktree setup discussions. | "layer" (the two-layer commit record is different). |
| **Scratch checkpoint** | A non-committing `refs/skillgrid/` snapshot of HEAD+index+worktree between commits. | Checkpoint/restore discussions. | "commit" (a commit is the durable record; a scratch ref is the in-between safety net, pruned). |
| **Atomic pair** | The branch+worktree-root pair recorded in a checkpoint; restore verifies both. | Restore safety. | "pair" without naming which pair. |
| **Linked session** | A Pi session created at a worktree's directory when worktrunk's directive moves the session; the source session remains resumable. | Session-movement discussions. | "fork" (Pi's `/fork` owns conversation state). |
| **No-replay recovery** | Recovery to a surviving worktree that does not execute a command requested after the original directory died. | Cwd-removal recovery. | "retry" (the point is the command is NOT run). |
