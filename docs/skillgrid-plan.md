## Workflow


```
proposal → specs → design → adr → tasks
```

```bash
workflow:
  proposal:
    -> openspec/changes/<change-id>/proposal.md
  specs:
    -> openspec/changes/<change-id>/specs/<spec-id>/spec.md
  design:
    -> openspec/changes/<change-id>/design.md
  adr:
    -> openspec/changes/<change>/adr.md
    -> openspec/adr/YYYY-MM-DD-<topic>.md
  tasks:
    -> openspec/changes/<change-id>/tasks.md
```

```bash
openspec/adr/YYYY-MM-DD-<topic>.md
openspec/changes/archive/
openspec/changes/<change-id>/
openspec/changes/<change-id>/adr.md
openspec/changes/<change-id>/design.md
openspec/changes/<change-id>/proposal.md
openspec/changes/<change-id>/tasks.md
openspec/changes/<change-id>/specs/
openspec/changes/<change-id>/specs/<spec-id>/spec.md
```

## Artifact Map

| Workflow Stage | Skill | Creates | From Template |
|----------------|-------|---------|---------------|
| setup | project-context | `AGENTS.md` | — |
| setup | project-context | `openspec/config.yaml` | — |
| proposal | brainstorming | `openspec/changes/<change-id>/proposal.md` | `openspec/schemas/intent-driven/templates/proposal.md` |
| specs | spec-as-source | `openspec/changes/<change-id>/specs/<spec-id>/spec.md` | `.agents/skills/spec-as-source/references/spec.md` |
| design | brainstorming | `openspec/changes/<change-id>/design.md` | `openspec/schemas/intent-driven/templates/design.md` |
| adr | architectural-decision-records | `openspec/changes/<change-id>/adr.md` | `openspec/schemas/intent-driven/templates/adr.md` |
| adr | architectural-decision-records | `openspec/adr/YYYY-MM-DD-<topic>.md` | `.agents/skills/architectural-decision-records/templates/` (mad full/minimal, nygard, y-statement, custom) |
| tasks | write-tasks | `openspec/changes/<change-id>/tasks.md` | `openspec/schemas/intent-driven/templates/tasks.md` |

## Pending Decision: Ownership of proposal.md and design.md

**Conflict discovered** (2026-08-27): Two skills claim to author `openspec/changes/<change-id>/proposal.md` and `design.md`:

| Skill | Origin | What it does with these files |
|-------|--------|-------------------------------|
| `brainstorming` | superpowers (adapted, `.agents/skills/brainstorming/SKILL.md`) | Dialogue-first authoring: classify spike/bounded/architectural, ask questions, propose 2-3 approaches, get per-section approval, then write proposal.md + design.md (lines 100-101) |
| `openspec-propose` | OpenSpec native (`.agents/skills/openspec-propose/SKILL.md`) | One-shot generation: derive change name from request, run `openspec new change`, follow schema-driven artifact pipeline via `openspec instructions` — creates proposal.md, specs, design.md, tasks.md in a single workflow (lines 18-21) |

Both are legitimate; the question is which workflow owns the **proposal and design stages** while the other stays out of the way.

### Proposal A — brainstorms own authoring, openspec-propose demoted to scaffolding only (current state)

**Rules:**
- `brainstorming` is the sole author of `proposal.md` and `design.md` for new changes (architectural path).
- `openspec-propose` may still run `openspec new change` for directory/metadata scaffolding (`.openspec.yaml`), but its artifact-generation steps are skipped when a `brainstorming` run has already produced the artifacts.
- `openspec-explore` stays as the in-flight amendment path (edits proposal.md on scope change, design.md on design decisions).

**Pros:**
- Preserves the approval gate (HARD-GATE) — user sees each design section before anything is written.
- Two-approach proposals with tradeoffs give the user real choice, vs. one-shot generation.
- Matches the `AGENTS.md` preference of "think before coding".

**Cons:**
- Two overlapping skills — new agents may not know which to invoke; the `openspec-propose` description invites it strongly ("Use when the user wants to quickly describe what they want to build and get a complete proposal").
- More steps in the loop for simple changes; brainstorming's ceremonial overhead.
- `openspec-propose` is version-managed by OpenSpec — upstream updates could re-introduce the conflict.

**Mitigations:**
- Edit `openspec-propose` description to "Scaffold a change directory. Use only to create/change directories; `brainstorming` authors the artifacts. (local patch)"
- Add a line to `openspec/config.yaml` proposal stage: "Author: brainstorming; scaffolding: openspec-propose"
- Track the local patch in `AGENTS.md` so refreshes don't silently revert it.

### Proposal B — openspec-propose owns the full artifact pipeline; brainstorming kept only for spike/bounded paths

**Rules:**
- `openspec-propose` runs end-to-end: proposal.md → specs → design.md → tasks.md.
- `brainstorming` shrinks to a pre-flight dialogue skill: spike and bounded paths only (no file writes). Architectural work goes straight to `openspec-propose`.
- Approval gate moves into `openspec/config.yaml` proposal rules: "User must approve each generated artifact before the next is created."

**Pros:**
- One authoritative artifact pipeline — no duplication, fewer steps for new agents to learn.
- Schema-driven (`openspec instructions`) — adding a new artifact type (e.g., `adr.md` in the pipeline) is a schema change, not a skill edit.
- Upgrades cleanly with OpenSpec releases.
- Aligns with the `intent-driven-template` function set where `spec-as-source` already owns spec authoring via OpenSpec conventions.

**Cons:**
- One-shot generation loses the per-section approval gate the `brainstorming` HARD-GATE enforces; users may find themselves approving a full design they didn't shape.
- The "2-3 approaches with tradeoffs" step is lost — the pipeline produces one design.
- `AGENTS.md` rule "Think before coding / simplicity first" is weakened for the most consequential changes (architectural).

**Mitigations:**
- Keep `brainstorming` as an optional pre-step that runs *before* `openspec-propose` when the user wants approach-level tradeoffs; it contributes a requirements brief that `openspec-propose` consumes.
- Add a `config.yaml` rule: "After generation, present each artifact for approval before implementation tasks are written."

### Proposal C — hybrid: separate the concerns by artifact type, not by workflow

**Rules:**
- `brainstorming` owns **proposal.md** (what & why) — its strength is intent discovery.
- `openspec-propose` owns **design.md + specs + tasks.md** generation — its strength is schema-driven, one-shot artifact authoring.
- Pipeline: `brainstorming` (dialogue → approved proposal) → `openspec new change` scaffolding → `openspec-propose` (generates design/specs/tasks from the approved proposal) → `architectural-decision-records` (ADR stage) → `write-tasks` refines tasks.md.
- `openspec-explore` remains the amendment path for all artifacts mid-change.

**Pros:**
- Each skill does what it's best at; no step is lost.
- The approval gate survives (it's on the proposal, which is the most important artifact to align on before downstream generation).
- Downstream artifacts (design/specs/tasks) are generated consistently from the schema — fewer freeform writes.
- The `brainstorming` architectural path shrinks to one artifact, which simplifies its checklist (steps 6-10 collapse to step 6, then hand off).

**Cons:**
- Two skills still involved in planning — the handoff between them must be explicit (approved proposal.md is the handoff contract).
- `openspec-propose` currently generates proposal.md itself; a local patch is needed to allow it to skip the proposal and consume the existing one (or rely on `openspec instructions` per-artifact generation).
- Most complex ruleset — new agents need to know the handoff contract.

**Mitigations:**
- Update `brainstorming` step 10: hand off with "Approved proposal at `openspec/changes/<id>/proposal.md`. Next: invoke `openspec-propose` restricted to design+specs+tasks."
- Patch `openspec-propose` step 5-6 to detect an existing approved `proposal.md` and skip regeneration (check `openspec status --json` for `done` proposal before regenerating it).
- Record the handoff contract in `AGENTS.md` under Pitfalls.

### Comparison

| Criterion | A: brainstorming owns | B: openspec-propose owns | C: hybrid split |
|-----------|:---------------------:|:------------------------:|:---------------:|
| Approval gate preserved | ✅ | ⚠️ weakened | ✅ |
| Approach tradeoffs | ✅ | ❌ | ✅ (on proposal) |
| Single pipeline clarity | ❌ two paths | ✅ | ⚠️ handoff contract |
| Upstream upgrade safety | ❌ local patches | ✅ | ⚠️ one local patch |
| Fits intent-driven-template | ⚠️ | ✅ | ✅ |
| Agent-learning cost | low | lowest | high |

**Decision (TBD — awaiting user choice):** A, B, or C

## Functions

* intent-driven-template
  * test driven development
  * acceptance-test
  * gherkin - BDD
* BMAD
  * [X] project-context -> AGENTS.md and openspec/config.yaml
* superpowers
  * [X] brainstorming
  * [x] write-tasks
  * micro commits - commit as checkpoint
  * git worktree
    * https://intent-driven.dev/blog/2026/04/01/openspec-git-worktrees-opencode/
    * subagent-driven-development
    * executing-plans