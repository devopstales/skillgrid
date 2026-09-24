# AISkillGrid

## The problem: your AI context decays

Every AI-assisted change is a small bet that the agent still knows what it's doing. It doesn't, for long. The prompt that nailed the first feature is buried under twenty sessions of "just a quick fix." Context fills up. Intent drifts. The agent re-derives the same design decisions you already made, and "done" becomes a claim with no evidence behind it.

You keep writing the same instructions. You keep watching the agent re-decide the same things. You keep merging work that passed its own tests but never actually did the thing. The knowledge that made the first change good was in a chat window, and the chat window is gone.

**This is not an agent problem. It's a context problem** — and it compounds.

## The fix: turn your project into agentic engineering

Skillgrid stops you from re-deciding. It gives the agent a set of **verbs it runs on your codebase** instead of a wall of instructions you have to repeat:

- **Interview** — the agent grills the ambiguity in your request and writes the answers down where the next session can find them.
- **Blueprint** — the agent turns a spec into a written, falsifiable plan with the decisions named *before* a line of code exists.
- **Slice** — the blueprint becomes vertical tracer-bullet tickets the agent can each finish in one fresh context window.
- **Verify** — the agent gates the result on fresh evidence, not on the fact that the suite is green.

Every verb writes what it learned to disk — the glossary, the ADRs, the blueprint, the acceptance criteria — so the *next* agent starts from your project's actual state, not from a blank chat. The verbs are **skills** under `.agents/skills/`, and they run as a repeatable, evidence-gated pipeline: **interview → blueprint → slice → execute → review → QA-gate → ship → reflect**.

---

## skillgrid (CLI)

The `skillgrid` CLI installs the hub into this machine. It creates
`~/.skillgrid/`, copies the repo into `~/.skillgrid/repos/skillgrid`,
verifies Node is available, lets you pick which agent (Kilo, OpenCode,
Cursor) to install, runs `npm install -g` for the chosen agents and the
shared tools (`skills`, `cucumber`), and copies the hub's `.agents/`
into `~/.agents/`.

```
skillgrid --version                 print version
skillgrid --help                    show help
skillgrid                           install (default)
skillgrid --dry-run                 print planned changes, don't write
skillgrid --verbose                 print detailed changes
skillgrid --yes                     use defaults (opencode + kilo)
skillgrid --skip-clone              skip the git clone step
skillgrid --agents opencode,kilo    preset agents (skip prompt)
skillgrid --skip-tools              skip global npm tool install
skillgrid --skip-agents             skip the ~/.agents override step
skillgrid --sync-repo PATH          sync a local repo path into the hub instead of cloning
```

### Install

Once built (and `dist/skillgrid-linux-amd64` is present), run:

```bash
task install
# or, anywhere on PATH:
skillgrid install
```

`install` reuses the same code path: `skillgrid install` will
clone-or-sync the hub repo, check Node, install the selected agents
(`opencode-ai`, `@kilocode/cli`, etc.) and the shared tools, then
copy `.agents/` into `~/.agents/`.

### Layout

```
~/.skillgrid/
├── bin/                      # skillgrid binary (put on PATH)
└── repos/
    └── skillgrid/            # cloned or synced hub
        ├── .agents/          # source that is copied into ~/.agents/
        ├── scripts/
        │   └── install_node.sh   # used when Node is missing
        └── ...
```

It ships:

- **30 skills** under `.agents/skills/` — workflow stages + general capabilities the agent loads at the right moment.
- **Git + agent hooks** under `hooks/` and `git-hooks/`, agent **plugins** under `plugins/` — "a rule asks; a hook guarantees."
- **Templates, references, and a router** the agent consumes at runtime.

## Why

The verbs are only half of it. A skill is an *ask* — and a model will rationalize its way past an ask, especially a long-running one. Skillgrid pairs every ask with a **guarantee**: git + agent Stop hooks that block bad commits, red stops, and skipped gates. "A rule asks; a hook guarantees." That pairing is what makes the pipeline hold under pressure instead of collapsing into "the agent said it was done."

Two properties matter most:

**Match the effort to the risk.** A throwaway prototype shouldn't be pushed through mutation testing and an eight-specialist review; a payment system shouldn't get away with a self-check. **Rigor tiers (T0–T3)** are a per-change verification dial — T0 self-checks, T3 runs the full gauntlet with a fresh-model reviewer — and a per-change `Tier:` line overrides the project default. The change's shape (a migration, a new trust boundary) still sets a floor the tier can't lower. See [08-concepts](docs/user-guide/08-concepts.md#rigor-tiers—match-effort-to-risk).

**Decisions live in files, not in chat.** When the agent hits a decision nobody made yet, the **Owed-Decision Gate** catches it *mechanically* (not by the agent feeling uncertain), and it can either resolve it or record it as an **`ASSUMED` blueprint** — a named assumption in a file that survives `/clear`, that teammates read, and that `qa` flags as decision debt until **Ratify** settles it. That single move is what stops the agent from silently inventing the load-bearing choice and burying it in code.

**The test pyramid is a first-class concept.** `qa` selects the *highest layer that fits* each behavior (unit / integration / E2E), applies a duplicate-coverage guard (a test at the wrong layer is a slower, flakier way to test something a cheaper layer already covers), and gates on mutation score + P0/P1 pass-rate thresholds. The full strategy is in [`.agents/skills/qa/references/test-strategy.md`](.agents/skills/qa/references/test-strategy.md).

## The pipeline

The four verbs are the spine. Each maps to one or more pipeline phases, and each writes its output to disk so the next verb — and the next session — starts from your project's actual state:

```
onboarding → interviewing → writing-blueprints → slicing → [approval gate]
   (setup)     INTERVIEW          BLUEPRINT         SLICE          ↓
reflect ← ship ← review ← qa ⇄ apply (simple / subagent / parallel)
   REFLECT      SHIP          VERIFY (review + QA gate over the applied slices)
                              ↑
        optional spike / sketch / research / code-research (before the blueprint)
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
| **Reflect** | Sourced retrospective + acceptance verdict + final-state `report.md` |

## External dependencies

Skillgrid's skills reference these external tools at runtime. Install them
if your agent doesn't already have them:

| Tool | Purpose | Install |
|------|---------|---------|
| [ripgrep](https://github.com/BurntSushi/ripgrep) (`rg`) | Fast content search — the skills' primary code-search primitive | `brew install ripgrep` / `apt install ripgrep` |
| [exa](https://github.com/ogham/exa) (`exa`) | Web search and file listing — used by research skills and file exploration | `brew install exa` / `go install github.com/ogham/exa/v6@latest` |
| [Trivy](https://github.com/aquasecurity/trivy) (`trivy`) | Security scanning — vulnerability, secret, misconfig, and license gates in QA | `brew install trivy` / `apt install trivy` |
| [Cucumber](https://github.com/cucumber/cucumber) (`cucumber`) | BDD test runner — executes Gherkin `acceptance.feature` scenarios | `npm install -g cucumber` / `gem install cucumber` |
| [jscpd](https://github.com/kucherenko/jscpd) (`jscpd`) | Copy-paste detection — code-quality gate in QA (duplication threshold) | `npm install -g jscpd` |
| [GitHub CLI](https://github.com/cli/cli) (`gh`) | GitHub issues, PRs, and code review — tracker + ship integration | `brew install gh` / `go install github.com/cli/cli@latest` |
| [GitLab CLI](https://gitlab.com/gitlab-org/cli) (`glab`) | GitLab issues, MRs, and code review — tracker + ship integration | `brew install glab` / `go install gitlab.com/gitlab-org/cli@latest` |
| [Jira CLI](https://github.com/ankitpokhrel/jira-cli) (`jira`) | Jira issues and boards — tracker integration | `brew install jira` / `go install github.com/ankitpokhrel/jira-cli@latest` |
| [Backlog.md](https://github.com/nstottard/backlog.md) (`backlog`) | Local file-based issue tracker — default tracker for `ticketing` | `brew install nstottard/tap/backlog` / `go install github.com/nstottard/backlog/cmd/backlog@latest` |
| [skills](https://github.com/vercel-labs/skills) (`npx skills`) | Open agent skills CLI — install/discover/update SKILL.md files across 75+ agents | `npx skills add <owner/repo>` |

All optional — the pipeline works without them, but the relevant gates
degrade (slower search, no security scan, BDD scenarios unexecuted, tracker
unavailable) when missing. The tracker CLIs (`gh`, `glab`, `jira`, `backlog`)
are only needed if your project uses that tracker (`ticketing.type` in config).

## Quick start

1. **Onboard** a project: ask the agent to run `onboarding` — it detects your stack/testing/tracker and writes `.skillgrid/config.yaml` (blocking until you confirm the facts).
2. **Route** every request through `using-skillgrid` — it establishes which skill to use and enforces context discipline.
3. **Wire the hooks** so discipline is enforced, not just asked (`skillgrid install` stages `hooks/`, `git-hooks/`, `plugins/` to `~/.skillgrid/` and points git's `core.hooksPath` at the staged git-hooks — guards, commit-msg, plus the agent Stop hooks):

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
| `code-research` | Wide or high-stakes question — parallel researchers + verify + red-team |

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

Each skill is a small, focused file: a `name` + `description` frontmatter (the description states **what it does** and **when to use it** — never the step list), an announcement line, the process, a `## Common Rationalizations` table, a `## Red Flags` list, and — for tail phases — a `prev-phase`/`next-phase`/`artifact` contract plus a Return Envelope. Shared material (conventions, the TDD cycle, the threat matrix) lives in `_shared/` and is referenced, never duplicated.

The shape is **enforced, not just documented**: `task skill-check` (→ `scripts/check-skillgrid-skills.mjs`) is a zero-dependency hygiene guard that checks every skill against [skill-anatomy](docs/skill-anatomy.md) — frontmatter, required sections, line budgets by tier, cross-skill reference form — plus **hot-path budgets** so the cumulative skill load of a canonical change path (blueprint / execution / QA / close) never grows past a ratcheted ceiling. It runs in CI. The full contract is in [docs/skill-anatomy.md](docs/skill-anatomy.md); contribution rules are in [CONTRIBUTING.md](CONTRIBUTING.md).

## Repository layout

```
.agents/
└── skills/            # the 30 skills + _shared/ (conventions, references)
hooks/                 # guard implementations (zone, protected-ref, stop, gate-*)
git-hooks/             # thin shims over hooks/ (exec ../hooks/...)
plugins/               # agent capture-bridge plugins (opencode, kilo, cursor)
.github/
├── prompts/           # canonical slash-command prompts (mirror source)
├── agents/            # canonical subagent definitions
└── workflows/         # CI: IDE-sync check + hook tests + skill hygiene
scripts/
├── sync-ide-assets.sh # mirror canonical prompts/agents to IDE copies
├── test-hooks.sh      # exercise the guard hooks against throwaway repos
└── check-skillgrid-skills.mjs  # skill-hygiene guard (anatomy + line + hot-path budgets)
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
| [09-serve-dashboard](docs/user-guide/09-serve-dashboard.md) | `skillgrid serve` web dashboard + views + tracker CLI |

## License

[MIT](LICENSE)
