# Start here

**Skillgrid** is an AI-assisted development hub: a Go CLI, Spec-Driven Development (SDD) skills, persistent memory (Mnemonic), and wiring for OpenCode, Kilo, and Cursor.

It turns “chat with an agent” into a repeatable pipeline — propose → spec → implement → verify → archive — with tests, tickets, and long-term memory that survive compaction.

## Why Skillgrid

Agents are strong at local edits and weak at long-horizon work. Context fills up, intent drifts, and “done” becomes a claim without evidence.

Skillgrid fixes that with:

1. **A written change contract** — `change.md`, `tasks.md`, and Gherkin acceptance before code.
2. **A user gate** — you approve Implement vs Revise after spec; agents do not auto-code.
3. **Evidence-based verify** — agent proof + human QA plan; findings re-enter apply.
4. **Memory outside the chat** — Mnemonic stores decisions, code search, and research cache so sessions stay lean.

## Main logics

| Practice | What Skillgrid does with it |
|----------|-----------------------------|
| **Spec-Driven Development (SDD)** | Pipeline `onboard → propose → spec → apply ⇄ verify → archive`. Specs under `docs/skillgrid/changes/` are the source of truth for a change. |
| **Test-Driven Development (TDD)** | Apply uses the `tdd` skill: red → green → refactor before claiming a step done. |
| **Intent-Driven Development** | Propose locks *why* and user-visible end state first; questioning grills ambiguity before tasks. |
| **Behavior-Driven Development (BDD)** | `acceptance.feature` (Gherkin) describes observable behaviour; verify traces work back to scenarios. |
| **Vertical slices** | Each step is a thin end-to-end path (UI → API → data), not a horizontal layer dump. |
| **Smart-side / dumb-side (≤ ~40% context)** | Keep the **smart zone** (orchestrator session) for routing, decisions, and pointers. Push heavy reads, research, and implementation into fresh subagents, disk artifacts, and Mnemonic. Target roughly **≤ 40%** of the context window for plan/execute work so quality does not degrade in the “dumb” remainder of the window. |

Entry skill: **`use-skillgrid`** — detects init state, routes the phase, enforces the user gate, resumes from `tasks.md` `## State.phase`.

```
onboard → propose → spec → [user gate] → apply ⇄ verify → archive
                 ↑
        optional explore / design-spike
```

## Sources of logic

Skillgrid does not invent this stack from scratch. It unifies practices from:

| Source | What we take |
|--------|----------------|
| [superpowers](https://github.com/obra/superpowers) | Bootstrap discipline, brainstorming/grill patterns, TDD, debugging, worktrees, subagent-driven execution, verification before “done” |
| [mattpocock/skills](https://github.com/mattpocock/skills) | Intent grilling, research → tickets, domain modeling, implement/prototype shapes |
| [addyosmani-agent-skills](https://github.com/addyosmani/agent-skills) | Planning/spec craft, shipping and launch hygiene |
| [gentleman-ai](https://github.com/Gentleman-Programming/gentle-ai) | SDD stage skills (propose/spec/apply/verify/archive), Engram-style memory protocol |
| [BMAD](https://github.com/bmad-code-org/BMAD-METHOD) | Spec/build/trace roles and structured build phases |
| [gsd-core](https://github.com/open-gsd/gsd-core) | Context engineering, smart-zone budgets, fresh-context subagents, onboard/execute loops |
| [intent-driven-template](https://github.com/intent-driven-dev/intent-driven-template) | OpenSpec-style artifacts, Gherkin-as-source, acceptance-first tasks |

## Quick path

1. [Install the CLI and wire agents](01-installation.md)
2. Open a project and ask the agent to run Skillgrid onboard / `use-skillgrid`
3. Follow [Workflow usage](02-workflow-usage.md) for your first change

## Manual map

| Doc | Topic |
|-----|--------|
| [01-installation](01-installation.md) | CLI install, project init, agents |
| [02-workflow-usage](02-workflow-usage.md) | SDD day-to-day |
| [03-skills](03-skills.md) | Workflow vs general skills |
| [04-hooks](04-hooks.md) | Agent hooks |
| [05-mcp-servers](05-mcp-servers.md) | MCP merge and servers |
| [06-multi-agent-work](06-multi-agent-work.md) | Parallel / subagent work |
| [07-memory-and-indexing](07-memory-and-indexing.md) | Mnemonic memory + code index |
| [08-ticketing-integrations](08-ticketing-integrations.md) | Backlog.md, GitHub, GitLab, Jira |
| [09-plugins](09-plugins.md) | Agent plugins |
| [10-webui](10-webui.md) | Mnemonic data viewer |
