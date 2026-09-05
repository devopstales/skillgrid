# Skills

Two kinds of skills. Do not duplicate a general capability as an `sdd-*` stage skill.

## Quick path

| Kind | Rule | Examples |
|------|------|----------|
| **Workflow** | Owns a pipeline stage (or the entry orchestrator). Writes stage artifacts under `docs/skillgrid/`. | `use-skillgrid`, `sdd-propose`, `sdd-apply` |
| **General** | Reusable across stages and outside SDD. Stages invoke them; they do not own `## State.phase`. | `tdd`, `questioning`, `mnemonic-memory` |

Priority: `use-skillgrid` → `sdd-*` stage → general skills the stage loads.

## Workflow skills

| Skill | Role |
|-------|------|
| `use-skillgrid` | Detect, classify, route, resume, enforce user gate |
| `sdd-onboard` | Bootstrap orchestrator (greenfield / brownfield) |
| `sdd-init` | Detect facts; write skeleton + AGENTS block |
| `sdd-map-codebase` | Optional brownfield narrative map |
| `sdd-agent-context` | AGENTS Skillgrid block / harness pointers |
| `sdd-constraints` | Quality bar into `config.yaml` `rules.*` |
| `sdd-domain` | Glossary bootstrap |
| `sdd-propose` | Reserve `NNN`; write `change.md` |
| `sdd-explore` | Helper: change-scoped `research.md` |
| `sdd-spec` | `tasks.md` (blocking DAG) + `acceptance.feature` |
| `sdd-apply` | Execute unblocked tasks |
| `sdd-verify` | Verdicts, QA plan, review; findings → apply |
| `sdd-archive` | Move change → `archive/` |

Skills live under `.agents/skills/` (hub copied to `~/.agents/` on install). Project copies may shadow for local customization.

## General skills (cross-cutting)

| Skill | Typical callers |
|-------|-----------------|
| `questioning` | Init, propose, revise after user gate |
| `investigate` | Explore, Q&A |
| `design-spike` | Before locking `change.md` |
| `codebase-design` | Propose technical approach |
| `glossary` | Propose, spec, domain |
| `tdd` | Apply |
| `debugging` | Apply, verify FAIL |
| `isolated-workspace` | Apply start |
| `subagent-execution` / `dispatching-parallel-agents` / `simple-execution` | Apply |
| `verification` | Verify, any “done” claim |
| `requesting-code-review` / `review-reception` / `judgment-day` | Verify |
| `finishing-a-development-branch` | Archive / ship |
| `mnemonic-memory` | All stages |
| `handoff` | Peel out-of-scope side work mid-change |
| `issue-creation` | When tracker tickets are forced |
| `work-unit-commits` | Apply |
| `creating-skills` / `playwright` | As needed |

Optional generated index: `docs/skillgrid/agents/skill-registry.md` — never an init gate.

## Where skills come from

Skillgrid unifies stage and general skills inspired by superpowers, mattpocock, addyosmani, gentleman-ai, BMAD, gsd-core, and intent-driven-template. See [Start here](00-start-here.md#sources-of-logic).

Managed install also uses the global `skills` npm CLI for additional skill packages configured in the hub.

## Next step

[Hooks](04-hooks.md) · [MCP servers](05-mcp-servers.md)
