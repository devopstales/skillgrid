# Start here

**Read the [README](../../README.md) first.** It makes the case: your AI context decays across sessions, and Skillgrid fixes that by turning your project into **agentic engineering** — a set of verbs the agent runs on your codebase (**interview → blueprint → slice → verify**) that write what they learn to disk so the next agent starts from your project's actual state. This page is the guide entry: the shape of the hub and where each topic lives.

**AISkillGrid** is a configuration hub for opinionated AI-assisted development: a set of reusable **skills** under `.agents/skills/`, **git + agent hooks** under `hooks/` and `git-hooks/`, agent capture-bridge **plugins** under `plugins/`, and templates/references that agents consume at runtime.

It turns "chat with an agent" into a repeatable pipeline — interview → blueprint → slice → execute → review → QA-gate → ship → reflect — with BDD acceptance, test-driven gates, persistent memory, and hook-enforced discipline that survive compaction.

## How it holds up

Five mechanisms turn the verbs into something that doesn't collapse under a long-running agent:

1. **A written blueprint** — `blueprint.md`, sliced `tasks.md`, and Gherkin acceptance before code; load-bearing decisions named up front, or recorded as an `ASSUMED` blueprint.
2. **A hard approval gate** — brainstorming and QA stop and ask; agents do not auto-ship.
3. **Evidence-based QA** — `G<n>` gates with `CHECK:`/`EXPECT:` oracles; a gate is met only when freshly run; effort scaled to risk via rigor tiers (T0–T3).
4. **Hooks that guarantee** — git + agent Stop hooks block bad commits and red stops; "a rule asks, a hook guarantees."
5. **A second brain outside the chat** — Mnemonic holds the project's decisions, a live code map, and the research cache, so the next agent starts from what the project already knows instead of a blank chat.

## Main logics

The four verbs are the spine; the practices are how each verb stays honest:

| Practice | Verb | What AISkillGrid does with it |
|----------|------|-------------------------------|
| **Intent-Driven Development (IDD)** | Interview | `interviewing` grills ambiguity via a design-tree frontier until a weighted clarity gate passes; the brainstorming approval gate makes intent explicit. Terms are recorded into glossary/ADRs as they resolve. |
| **Spec-Driven Development (SDD)** | Blueprint | `writing-blueprints` turns the spec into a falsifiable plan with Must-Haves, one-way-door tags, a recorded **build shape**, and the **Owed-Decision Gate** (decisions named or recorded as `ASSUMED`). `blueprint.md` + sliced `tasks.md` under `.skillgrid/` are the source of truth. |
| **Vertical slices** | Slice | `slicing` breaks the blueprint into tracer-bullet tickets with dependency edges and execution waves, ordered by the blueprint's recorded build shape. |
| **Test-Driven Development (TDD)** | Verify | `test-driven-development`: iron-law red → green → refactor before claiming a task done. |
| **Behavior-Driven Development (BDD)** | Verify | `acceptance-test-authoring`: Gherkin-in-Markdown `acceptance.feature` is non-negotiable (`bdd.enabled` always true). |
| **Match effort to risk** | Verify | **Rigor tiers (T0–T3)** scale the QA floor and review topology to the change's risk; the change's shape still sets a floor the tier can't lower. T3 requires a fresh-model reviewer. |
| **Smart-side / dumb-side (≤ ~40% context)** | (all) | Keep the orchestrator for routing/decisions; push heavy reads, research, and implementation into fresh subagents (`parallel-execution`, `subagent-execution`), disk artifacts, and Mnemonic. |

Entry skill: **`using-skillgrid`** — invoke the relevant skill before ANY response, including clarifying questions. It checks config/resume/domain-model state, enforces context discipline, skill priority, and platform adaptation.

```
onboarding → interviewing → writing-blueprints → slicing → [approval gate] → apply ⇄ qa → ship
   (setup)     INTERVIEW          BLUEPRINT         SLICE                         VERIFY
              ↑                    ↑
        optional spike / sketch / research / code-research
```

## Quick path

1. Read the [README](../../README.md) — the case for agentic engineering and the four verbs
2. [Understand the layout](01-layout.md) — what lives where in `.agents/`
3. Read [Skills](02-skills.md) — the 30 skills and what each owns
4. Follow [Workflow usage](03-workflow-usage.md) for your first change
5. Wire [Hooks](04-hooks.md) so discipline is enforced, not just asked
6. Read [Concepts](08-concepts.md) — rigor tiers, assumed blueprints, build shapes, the QA gate, and every other recurring idea

## Guide map

| Doc | Topic |
|-----|--------|
| [01-layout](01-layout.md) | Directory structure, config, naming |
| [02-skills](02-skills.md) | The 30 skills, grouped by role |
| [03-workflow-usage](03-workflow-usage.md) | SDD day-to-day |
| [04-hooks](04-hooks.md) | Git + agent Stop hooks |
| [05-memory-and-indexing](05-memory-and-indexing.md) | Mnemonic memory + code index |
| [06-multi-agent-work](06-multi-agent-work.md) | Parallel / subagent work |
| [07-ticketing](07-ticketing.md) | Backlog.md, GitHub, GitLab, Jira |
| [08-concepts](08-concepts.md) | The recurring ideas: practices, PIV loop, gates, clarity gate, **rigor tiers**, **assumed blueprints + ratify**, **build shapes**, waves, QA gate, ADRs, firewall, review discipline, Depth Tree, ticketing, Mnemonic, resume, Gherkin, branch finishing |
