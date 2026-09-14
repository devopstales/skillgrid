# AISkillGrid

A configuration hub for opinionated AI-assisted development. It turns "chat with an agent" into a repeatable, evidence-gated pipeline — **interview → blueprint → slice → execute → review → QA-gate → ship → reflect** — with BDD acceptance, test-driven gates, persistent memory, and hook-enforced discipline that survives compaction.

It ships:

- **30 skills** under `.agents/skills/` — workflow stages + general capabilities the agent loads at the right moment.
- **Git + agent hooks** under `.agents/hooks/` and `.agents/git-hooks/` — "a rule asks; a hook guarantees."
- **Templates, references, and a router** the agent consumes at runtime.

## Why

Agents are strong at local edits and weak at long-horizon work. Context fills up, intent drifts, and "done" becomes a claim without evidence. AISkillGrid fixes that with a written blueprint before code, a hard approval gate before execution, evidence-based QA (`G<n>` gates with `CHECK:`/`EXPECT:` oracles), hooks that block bad commits and red stops, and memory that lives outside the chat.

**The test pyramid is a first-class concept.** `qa` selects the *highest layer that fits* each behavior (unit / integration / E2E), applies a duplicate-coverage guard (a test at the wrong layer is a slower, flakier way to test something a cheaper layer already covers), and gates on mutation score + P0/P1 pass-rate thresholds. The full strategy is in [`.agents/skills/qa/references/test-strategy.md`](.agents/skills/qa/references/test-strategy.md).

## The pipeline

```
onboarding → interviewing → writing-blueprints → slicing → [approval gate]
                                                              ↓
reflect ← ship ← review ← qa ⇄ apply (simple / subagent / parallel)
                              ↑
        optional spike / sketch / research / deep-research (before the blueprint)
```

| Phase | You get |
|-------|---------|
| **Onboard** | `.skillgrid/config.yaml`, glossary stubs, an `AGENTS.md` block |
| **Interview** | Ambiguity resolved via a design-tree frontier; terms recorded into the glossary/ADRs |
| **Blueprint** | `blueprint.md` — Must-Haves, one-way-door tags, no placeholders |
| **Slice** | `tasks.md` — vertical tracer-bullet tickets, dependency edges, execution waves |
| **Approval gate** | Go / Revise — never skipped |
| **Apply** | Slices executed TDD-first; commits via the canonical commit protocol |
| **Review + QA** | Two-axis or up-to-8-specialist review (a11y + performance added when the diff touches UI or data); 4-state QA gate (PASS / CONCERNS / FAIL / WAIVED) over the **test pyramid** — layer selection, duplicate-coverage guard, mutation + P0/P1 pass-rate thresholds |
| **Ship** | Integrate to base (merge / PR / keep) + mechanical `specs/` → `archive/` move |
| **Reflect** | Sourced retrospective + acceptance verdict + final-state `archive-report.md` |

## Quick start

1. **Onboard** a project: ask the agent to run `onboarding` — it detects your stack/testing/tracker and writes `.skillgrid/config.yaml` (blocking until you confirm the facts).
2. **Route** every request through `using-skillgrid` — it establishes which skill to use and enforces context discipline.
3. **Wire the hooks** so discipline is enforced, not just asked:

   ```bash
   bash <install-root>/scripts/install-hooks.sh            # git guards (zone, protected-ref, commit-msg)
   bash <install-root>/scripts/install-hooks.sh --with-stop  # + Stop-phase test + gate gates
   ```

4. **Make a change**: `using-skillgrid` routes it through the pipeline; you approve at the blueprint gate; the agent executes, gates, and ships.

For the full walkthrough see [docs/user-guide/03-workflow-usage.md](docs/user-guide/03-workflow-usage.md).

## Skills

30 skills under `.agents/skills/`, grouped by role. Stages load general skills — a capability is not duplicated as a stage.

### Entry / router
| Skill | Use when |
|-------|----------|
| `using-skillgrid` | Starting any conversation — routes the phase, enforces context discipline |

### Workflow stages
| Skill | Use when |
|-------|----------|
| `onboarding` | Setting up a new project (once) — detect stack, write config |
| `writing-blueprints` | You have a spec/requirements for a multi-step task, before code |
| `slicing` | After the blueprint — break it into tracer-bullet tickets + waves |
| `qa` | A change is ready for the quality gate, before final review/merge |
| `ticketing` | After slicing — publish tickets, run the status machine |

### Discovery
| Skill | Use when |
|-------|----------|
| `brainstorming` | Before any creative work — explore intent + design |
| `interviewing` | Brainstorming needs to sharpen a design; stress-test thinking |
| `spike` | The plan rests on an unproven technical claim — falsifiable verdict |
| `sketch` | The design has 2+ layout options and the answer depends on *feeling* it |
| `research` | The design depends on a fact not in the codebase (lightweight) |
| `deep-research` | Wide or high-stakes question — parallel researchers + verify + red-team |

### Execution
| Skill | Use when |
|-------|----------|
| `simple-execution` | Executing a written plan in a separate session with review checkpoints |
| `subagent-execution` | Executing a plan with a fresh implementer subagent per task (current session) |
| `parallel-execution` | 2+ independent tasks with no shared state or sequential dependency |
| `work-unit-commits` | Committing work units — the canonical commit protocol |
| `resume` | Continuing prior work, or whenever you've lost your place |

### Quality
| Skill | Use when |
|-------|----------|
| `test-driven-development` | Implementing any feature/bugfix, before implementation code |
| `test-driven-verification` | About to claim work is done — run verification, evidence before assertions |
| `acceptance-test-authoring` | Authoring BDD acceptance tests before implementation |
| `requesting-code-review` | Completing tasks / before merge — two-axis review |
| `parallel-code-review` | Large or high-risk diff — 6 specialist reviewers in parallel |
| `receiving-code-review` | Handling review feedback with technical rigor |

### Close-out
| Skill | Use when |
|-------|----------|
| `ship` | A change passed QA + review — integrate + archive the change folder |
| `reflect` | After ship — terminal retrospective + session close |

### Cross-cutting
| Skill | Use when |
|-------|----------|
| `mnemonic` | Persisting/recalling project memory, code orientation, web research |
| `isolated-workspace` | Starting feature work that needs workspace isolation |
| `structured-debugging` | Any bug, test failure, or unexpected behavior, before fixes |
| `architectural-decision-records` | Editing the glossary or drafting/reviewing ADRs |
| `ponytail` | Enforcing the laziest working solution (7-rung ladder) |

## How a skill is shaped

Each skill is a small, focused file: a `name` + `description` frontmatter (the description states **what it does** and **when to use it** — never the step list), an announcement line, the process, a `## Common Rationalizations` table, a `## Red Flags` list, and — for tail phases — a `prev-phase`/`next-phase`/`artifact` contract plus a Return Envelope. Shared material (conventions, the TDD cycle, the threat matrix) lives in `_shared/` and is referenced, never duplicated. The full contract is in [docs/skill-anatomy.md](docs/skill-anatomy.md); contribution rules are in [CONTRIBUTING.md](CONTRIBUTING.md).

## Repository layout

```
.agents/
├── skills/            # the 30 skills + _shared/ (conventions, references)
├── hooks/             # guard implementations (zone, protected-ref, stop, gate-*)
└── git-hooks/         # thin shims over .agents/hooks/
.github/
├── prompts/           # canonical slash-command prompts (mirror source)
├── agents/            # canonical subagent definitions
└── workflows/         # CI: IDE-sync check + hook tests
scripts/
├── install-hooks.sh   # wire the hook set into a repo
├── sync-ide-assets.sh # mirror canonical prompts/agents to IDE copies
└── test-hooks.sh      # exercise the guard hooks against throwaway repos
docs/user-guide/       # the full walkthrough (layout, skills, workflow, hooks, memory, …)
```

## Guide map

| Doc | Topic |
|-----|-------|
| [00-start-here](docs/user-guide/00-start-here.md) | Why + the main logics |
| [01-layout](docs/user-guide/01-layout.md) | Directory structure, config, naming |
| [02-skills](docs/user-guide/02-skills.md) | Every skill, grouped by role |
| [03-workflow-usage](docs/user-guide/03-workflow-usage.md) | SDD day-to-day |
| [04-hooks](docs/user-guide/04-hooks.md) | Git + agent Stop hooks |
| [05-memory-and-indexing](docs/user-guide/05-memory-and-indexing.md) | Mnemonic memory + code index |
| [06-multi-agent-work](docs/user-guide/06-multi-agent-work.md) | Parallel / subagent work |
| [07-ticketing](docs/user-guide/07-ticketing.md) | Backlog.md / GitHub / GitLab / Jira |
| [08-concepts](docs/user-guide/08-concepts.md) | Gates, waves, clarity gate, PIV loop, ADRs, … |

## License

[MIT](LICENSE)
