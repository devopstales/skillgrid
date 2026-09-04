# Agent Config

This file is the cross-agent standard config for all AI agents (Kilo, OpenCode, Cursor, Claude, Codex, Gemini). It is the **primary** source of truth for SDD configuration in this repo.

<!-- skillgrid-sdd:start -->
## Agent skills

Skillgrid SDD is active in this repo. The workflow, registry, and tracker below are the source of truth for agent work here.

### Workflow
`init → explore → propose → spec → apply → verify → archive`

Entry: invoke **`use-skillgrid`** for change work (uninitialized → `sdd-init`; else → `sdd-explore` then down the pipeline). No platform hook required.

- Skill registry (index of installed skills + triggers): `docs/skillgrid/agents/skill-registry.md`
- Project facts (stack, testing, tracker, conventions): `docs/skillgrid/config.yaml` and Mnemonic (`sdd/skillgrid/…`)
- Triage labels: `docs/skillgrid/agents/issue-tracker.md` + the tracker's label map

### Issue tracker
Issues: Backlog.md tickets under `.backlog/tasks/` (backlog CLI).
<!-- skillgrid-sdd:end -->

## Code Index

Mnemonic's code index is active. The index lives at `~/.skillgrid/mnemonic/skillgrid.sqlite`
and is accessed via `skillgrid mcp` (MCP) or the `skillgrid index` CLI. See
`.agents/skills/_shared/conventions/mnemonic-code-indexing.md` for the full protocol.

**Status**: `code_status` reports stale or empty → run `code_index` before `code_search`.

## Conventions

Project conventions are declared in `.agents/skills/_shared/`:

- [Mnemonic Memory Protocol](.agents/skills/_shared/conventions/mnemonic-memory.md) — persistent memory, session lifecycle, when to save.
- [SDD Structure](.agents/skills/_shared/conventions/sdd-structure.md) — changes/archive layout, NNN-slug numbering, `change.md` / `tasks.md` / `acceptance.feature`.

<!-- BACKLOG.MD GUIDELINES START -->
<!-- backlog.md-instructions-version: 1.51.0 -->
<CRITICAL_INSTRUCTION>

## Backlog.md Workflow

This project uses Backlog.md for task and project management.

**At the beginning of each conversation in this project, run `backlog instructions overview` before answering or taking action. Re-read it only if you have not read it yet in the current conversation.**

Use the overview to decide whether to search, read, create, or update Backlog tasks.

Before task lifecycle actions, read the matching detailed guide:
- `backlog instructions task-creation` before creating or splitting tasks
- `backlog instructions task-execution` before planning, changing status or assignee, adding a plan or implementation notes, or implementing task work
- `backlog instructions task-finalization` before checking acceptance criteria, writing final summaries, or moving tasks to terminal statuses

Use `backlog <command> --help` before running unfamiliar commands. Help shows options, fields, and examples.

Do not edit Backlog task, draft, document, decision, or milestone markdown files directly. Use the `backlog` CLI so metadata, relationships, and history stay consistent.

</CRITICAL_INSTRUCTION>
<!-- BACKLOG.MD GUIDELINES END -->
