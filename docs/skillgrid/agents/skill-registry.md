# Skill Registry

Index of installed skills available to agents in this repo. Each entry points to the full `SKILL.md` for trigger text and usage rules — read the skill file, not just this table.

**Scan rules**: project-level skills (`.agents/skills/`) are preferred over user-level (`~/.agents/skills/`). `sdd-*`, `_shared`, and `skill-registry` entries are workflow machinery, not project skills, and are excluded from this listing. Convention files (`AGENTS.md`, `CLAUDE.md`, `.cursorrules`, `GEMINI.md`, `CONTRIBUTING.md`) are listed at the bottom.

## Project skills

| name | trigger | path | scope |
|---|---|---|---|
| codebase-design | Shared vocabulary for designing deep modules. Use when designing or improving a module's interface, finding deepening opportunities, deciding where a seam goes, making code more testable, or when another skill needs the deep-module vocabulary. | `.agents/skills/codebase-design/SKILL.md` | project |
| creating-skills | Create a new agent skill (a SKILL.md plus optional scripts/references) from a real task, domain expertise, or user request. Use when the user asks to make, write, scaffold, or refine a skill; when a repeating workflow, project convention, or repeated correction should become a skill; or when validating an existing skill's structure. | `.agents/skills/creating-skills/SKILL.md` | project |
| debugging | Use when encountering any bug, test failure, or unexpected behavior, before proposing fixes | `.agents/skills/debugging/SKILL.md` | project |
| design-spike | Build a throwaway prototype to answer a design question — state machine, data shape, or UI look. Use when the user wants to sanity-check whether a state model or logic feels right, or explore what a UI should look like before committing; not for production code or feature work. | `.agents/skills/design-spike/SKILL.md` | project |
| dispatching-parallel-agents | Use when an SDD phase or debugging task has 2+ independent work items that can be executed without shared state or sequential dependencies. Provides the decision protocol for fanning out work to parallel sub-agents and the per-agent prompt structure. | `.agents/skills/dispatching-parallel-agents/SKILL.md` | project |
| finishing-a-development-branch | Use when an SDD change is complete (all steps PASS, all `[x]` marks in place) and you need to decide how to integrate the work. Verifies tests on the integrated tree, detects environment, presents the merge/PR/keep menu, and owns worktree cleanup. | `.agents/skills/finishing-a-development-branch/SKILL.md` | project |
| glossary | Use when authoring or reviewing change.md, tasks, glossary entries, domain/technical terms, or wording consistency. Enforces term reuse; fold into main artifacts — no companion glossary-reference files. | `.agents/skills/glossary/SKILL.md` | project |
| handoff | Compact the current conversation into a handoff document for another agent to pick up. Use when ending a session, switching agents, or the user asks for a handoff / summary / context transfer. | `.agents/skills/handoff/SKILL.md` | project |
| investigate | Investigate a question against high-trust primary sources and capture the findings as one cited Markdown file. Use when you need to research a topic, gather docs or API facts, compare library options, delegate the reading legwork to a background agent, or produce the upstream research for an SDD change. | `.agents/skills/investigate/SKILL.md` | project |
| isolated-workspace | Use when starting feature work that needs isolation from current workspace or before executing implementation plans — ensures an isolated workspace exists via native tools or git worktree fallback | `.agents/skills/isolated-workspace/SKILL.md` | project |
| issue-creation | Create, triage, and publish issues to the repository's tracker — Jira, GitHub, GitLab, or Backlog.md. Trigger: when the user asks to create an issue/epic/task/ticket, report a bug, file a feature request, or map SDD tasks to tracker issues. | `.agents/skills/issue-creation/SKILL.md` | project |
| judgment-day | Trigger: judgment day, dual review, adversarial review, juzgar. Run explicit blind dual review with at most two scoped fix/re-judgment rounds. | `.agents/skills/judgment-day/SKILL.md` | project |
| mnemonic-memory | ALWAYS ACTIVE — Persistent memory protocol. You MUST save decisions, conventions, bugs, and discoveries to Mnemonic proactively. Do NOT wait for the user to ask. | `.agents/skills/mnemonic-memory/SKILL.md` | project |
| playwright | Playwright E2E testing patterns. Trigger: When writing E2E tests — Page Objects, selectors, MCP workflow. | `.agents/skills/playwright/SKILL.md` | project |
| questioning | Stress-test a plan, decision, or idea branch by branch before implementation, using a design tree, frontier, and rounds with recommendations. Use when you need to clarify intent, the orchestrator delegates a clarification round (explore/propose/init), or a request must be classified before design. | `.agents/skills/questioning/SKILL.md` | project |
| requesting-code-review | Use after completing an SDD step that carries high-risk signals, before sdd-archive, or optionally when stuck / before refactor / after a complex bug fix. Dispatches a fresh code-reviewer sub-agent with crafted context — never your session history — and acts on the findings. | `.agents/skills/requesting-code-review/SKILL.md` | project |
| review-reception | Use when receiving code review feedback, before implementing suggestions — verify the finding against the codebase first, push back with technical evidence when warranted, implement one item at a time with tests | `.agents/skills/review-reception/SKILL.md` | project |
| simple-execution | Execute SDD (or general) implementation plans inline — one task at a time in the current context, with a strict RED/GREEN/TRIANGULATE/REFACTOR cycle when the project requires it, marking each task [x] as it completes, and producing the per-step evidence that sdd-verify will audit. Use when the plan is small, tightly coupled, or below the dispatch threshold — not when the workload decision recommends chained or stacked PRs. | `.agents/skills/simple-execution/SKILL.md` | project |
| subagent-execution | Execute an implementation plan by dispatching a fresh implementer subagent per task, running a task-scoped spec+quality review after each (not a final pass), closing findings in a bounded fix loop, and keeping a per-plan work directory of briefs, reports, review packages, and a progress ledger that survives context loss. Use when there is an implementation plan to execute and the work should be delegated to fresh subagents with review between tasks, rather than done inline in a single context. | `.agents/skills/subagent-execution/SKILL.md` | project |
| tdd | Use when implementing any feature or bugfix, before writing implementation code — write the failing test first, watch it fail, write minimal code to pass, then refactor | `.agents/skills/tdd/SKILL.md` | project |
| use-skillgrid | Use at conversation start and whenever feature/change work begins. Routes Skillgrid SDD: uninitialized → sdd-init; else → sdd-explore then propose → spec → apply → verify → archive. | `.agents/skills/use-skillgrid/SKILL.md` | project |
| verification | Use when about to claim work is complete, fixed, or passing, before committing, creating PRs, or marking a task done — run verification and read the output before any success claim; evidence before assertions always | `.agents/skills/verification/SKILL.md` | project |

## SDD workflow skills

These are the SDD orchestration skills, available but excluded from the main index (they are workflow machinery).

Phase order (v3): `init → explore → propose → spec → apply → verify → archive`.

| name | path | scope | notes |
|---|---|---|---|
| sdd-init | `.agents/skills/sdd-init/SKILL.md` | project | |
| sdd-explore | `.agents/skills/sdd-explore/SKILL.md` | project | |
| sdd-propose | `.agents/skills/sdd-propose/SKILL.md` | project | writes `change.md` (absorbs design) |
| sdd-spec | `.agents/skills/sdd-spec/SKILL.md` | project | writes `tasks.md` + `acceptance.feature` (absorbs tasks) |
| sdd-apply | `.agents/skills/sdd-apply/SKILL.md` | project | |
| sdd-verify | `.agents/skills/sdd-verify/SKILL.md` | project | verdicts in `tasks.md` |
| sdd-archive | `.agents/skills/sdd-archive/SKILL.md` | project | |
| use-skillgrid | `.agents/skills/use-skillgrid/SKILL.md` | project | **entry router** (not a phase) |
| sdd-design | `.agents/skills/sdd-design/SKILL.md` | project | **retired** → redirect to sdd-propose |
| sdd-tasks | `.agents/skills/sdd-tasks/SKILL.md` | project | **retired** → redirect to sdd-spec |

## Convention files

| name | path | scope |
|---|---|---|
| AGENTS.md | `AGENTS.md` | project |
| _shared conventions | `.agents/skills/_shared/conventions/` | project |
| _shared agent config | `.agents/skills/_shared/agent-config/` | project |
| _shared issue-tracker seeds | `.agents/skills/_shared/issue-tracker/` | project |
