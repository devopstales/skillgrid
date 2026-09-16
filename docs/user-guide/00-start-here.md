# Start here

**AISkillGrid** is a configuration hub for opinionated AI-assisted development: a set of reusable **skills** under `.agents/skills/`, **git + agent hooks** under `.agents/hooks/` and `.agents/git-hooks/`, and templates/references that agents consume at runtime.

It turns "chat with an agent" into a repeatable pipeline — interview → blueprint → slice → execute → review → QA-gate — with BDD acceptance, test-driven gates, persistent memory, and hook-enforced discipline that survive compaction.

## Why AISkillGrid

Agents are strong at local edits and weak at long-horizon work. Context fills up, intent drifts, and "done" becomes a claim without evidence.

AISkillGrid fixes that with:

1. **A written blueprint** — `blueprint.md`, sliced `tasks.md`, and Gherkin acceptance before code.
2. **A hard approval gate** — brainstorming and QA stop and ask; agents do not auto-ship.
3. **Evidence-based QA** — `G<n>` gates with `CHECK:`/`EXPECT:` oracles; a gate is met only when freshly run.
4. **Hooks that guarantee** — git + agent Stop hooks block bad commits and red stops; "a rule asks, a hook guarantees."
5. **Memory outside the chat** — Mnemonic stores decisions, code search, and research cache so sessions stay lean.

## Main logics

| Practice | What AISkillGrid does with it |
|----------|-------------------------------|
| **Spec-Driven Development (SDD)** | Pipeline `onboarding → writing-blueprints → slicing → apply ⇄ qa → ticketing`. Blueprints and sliced `tasks.md` under `.skillgrid/` are the source of truth. |
| **Test-Driven Development (TDD)** | `test-driven-development`: iron-law red → green → refactor before claiming a task done. |
| **Intent-Driven Development (IDD)** | `interviewing` grills ambiguity via a design-tree frontier until a weighted clarity gate passes; the brainstorming approval gate makes intent explicit. Terms are recorded into glossary/ADRs as they resolve. |
| **Behavior-Driven Development (BDD)** | `acceptance-test-authoring`: Gherkin-in-Markdown `acceptance.feature` is non-negotiable (`bdd.enabled` always true). |
| **Vertical slices** | `slicing` breaks the blueprint into tracer-bullet tickets with dependency edges and execution waves. |
| **Smart-side / dumb-side (≤ ~40% context)** | Keep the orchestrator for routing/decisions; push heavy reads, research, and implementation into fresh subagents (`parallel-execution`, `subagent-execution`), disk artifacts, and Mnemonic. |

Entry skill: **`using-skillgrid`** — invoke the relevant skill before ANY response, including clarifying questions. It checks config/resume/domain-model state, enforces context discipline, skill priority, and platform adaptation.

```
onboarding → interviewing → writing-blueprints → slicing → [approval gate] → apply ⇄ qa → ticketing
              ↑                    ↑
        optional spike / sketch / research / deep-research
```

## Quick path

1. [Understand the layout](01-layout.md) — what lives where in `.agents/`
2. Read [Skills](02-skills.md) — the 28 skills and what each owns
3. Follow [Workflow usage](03-workflow-usage.md) for your first change
4. Wire [Hooks](04-hooks.md) so discipline is enforced, not just asked

## Guide map

| Doc | Topic |
|-----|--------|
| [01-layout](01-layout.md) | Directory structure, config, naming |
| [02-skills](02-skills.md) | The 28 skills, grouped by role |
| [03-workflow-usage](03-workflow-usage.md) | SDD day-to-day |
| [04-hooks](04-hooks.md) | Git + agent Stop hooks |
| [05-memory-and-indexing](05-memory-and-indexing.md) | Mnemonic memory + code index |
| [06-multi-agent-work](06-multi-agent-work.md) | Parallel / subagent work |
| [07-ticketing](07-ticketing.md) | Backlog.md, GitHub, GitLab, Jira |
| [08-concepts](08-concepts.md) | The recurring ideas: practices, PIV loop, gates, clarity gate, waves, QA gate, ADRs, firewall, review discipline, Depth Tree, ticketing, Mnemonic, resume, Gherkin, branch finishing |
