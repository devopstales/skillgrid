# Agent config block (shared payload)

Single source of truth for the `## Skillgrid` block that `onboarding` writes into a project's agent config file (`AGENTS.md` or `CLAUDE.md`). The per-target files only decide *which file* to use — they must not drift in content.

The block already lives in the target file in most projects. The sentinels make the write idempotent: update in place, never duplicate.

## The block

Copy exactly, fill the `{placeholders}`, wrap between the sentinels. Do not add prose beyond it — these lines load into context in many tools on every run.

```markdown
<!-- skillgrid:start -->
## Skillgrid

This project is configured with Skillgrid. Project: **{project}**.

Config: `.skillgrid/config.yaml` (static) — read it before running any Skillgrid skill.
State: `.skillgrid/state.yaml` (dynamic) — where the project is right now (phase, current change, progress).

### Artifacts

| Artifact | Path |
|----------|------|
| Project knowledge (PRD, architecture, terms, ADRs, constraints, research) | `.skillgrid/artifacts/` |
| Project state (phase, current change, progress) | `.skillgrid/state.yaml` |
| Specs (briefing, blueprint, tasks) | `.skillgrid/specs/` |
| Execution ledger | `.skillgrid/sdd/` (gitignored) |

**Domain model:** read the vocabulary + ADRs under `.skillgrid/artifacts/` (index: `README.md`) before designing or implementing.

**Rules & standards:** locked project constraints render under `### Rules` below. Shared standards are referenced, never inlined — each loads via the skills that apply it.

- **Coding conventions** — reference `.agents/skills/_shared/rules/code-standards.md` for detailed coding conventions.
- **Testing conventions** — reference `.agents/skills/_shared/references/strict-tdd.md` for the TDD cycle and testing conventions.
- **Commits & verification** — reference `.agents/skills/_shared/rules/` for the commit contract, verification ladder, and rigor tiers.
- **Memory, code index & web cache** — reference the `skillgrid:mnemonic` skill (shared rules: `.agents/skills/_shared/rules/mnemonic-memory.md`).

### Rules

*Locked constraints only (source of truth: `### Locked constraints` in `.skillgrid/ASSUMPTIONS.md` — edit there, then mirror here).*

{rules_block}

### Issue Tracker

{tracker_line}

### Memory

{memory_line}

### Workflow

`brainstorming` → `writing-blueprints` → `slicing` → `ticketing` → execution (`subagent-execution` or `simple-execution`) → `qa` → `requesting-code-review` → `receiving-code-review`

Run `skillgrid:onboarding` to update config after stack changes.
<!-- skillgrid:end -->
```

## Placeholders

| placeholder | default | fill from |
|---|---|---|
| `{project}` | — | detected project name |
| `{tracker_line}` | — | one-line tracker summary, chosen per active tracker |
| `{memory_line}` | — | mnemonic status line |
| `{rules_block}` | — | bullet list from `ASSUMPTIONS.md` § `### Locked constraints` (one `-` per constraint); if the section is empty, `No locked constraints yet — see `.skillgrid/ASSUMPTIONS.md`.` |

`{tracker_line}` — pick the active tracker:

- Backlog.md: `Tickets live under `.backlog/tasks/`, managed via the `backlog` CLI.`
- GitHub: `GitHub issues via the `gh` CLI.`
- GitLab: `GitLab issues via the `glab` CLI.`
- Jira: `Jira issues via the `jira` CLI (project key: {jira_key}).`
- None: `None — work local-only from tasks.md.`

`{memory_line}`:

- Enabled: `Persistent memory is active (Mnemonic). Use `mem_save` for decisions, `mem_search` for recall, `code_status` → `code_index` → `code_search` for code orientation. Full protocol: `skillgrid:mnemonic` skill.`
- Disabled: `Mnemonic not detected — persistent memory is inactive. Install the `skillgrid` CLI to enable it.`

## Idempotent upsert (required)

1. Search the target file for `<!-- skillgrid:start -->`.
2. **Found** → replace everything from that marker to `<!-- skillgrid:end -->` (inclusive) with the freshly rendered block. Never append a second copy.
3. **Not found** → append the block at the end of the file.
4. Never create a second root config file: if `AGENTS.md` exists, update it; else create `AGENTS.md`. If `CLAUDE.md` also exists, add a one-line pointer (not the full block).

## Gotchas

- The sentinels are HTML comments — invisible in rendered markdown.
- If both `AGENTS.md` and `CLAUDE.md` exist, write the full block to `AGENTS.md` (source of truth) and put only a one-line pointer in `CLAUDE.md`. Two full blocks = two sources that drift.

## Keep this block lean (the block is injected into every request)

The `## Skillgrid` block loads into context on **every** prompt in most agents, so it is a *navigation spine*, not a knowledge dump. Rules for editing it:

- **Navigation, not conventions.** The block points at where things live (`.skillgrid/…`, tracker, memory). Project conventions, framework rules, delivery/PR steps, and how-to detail belong in the referenced files (`.skillgrid/artifacts/`, the skills, `docs/user-guide/`) — never inlined here.
- **Reference, don't write.** A standards line in the block names ONE canonical file (or skill) and says "reference X for Y" — e.g. `**Coding conventions** — reference `.agents/skills/_shared/rules/code-standards.md` for detailed coding conventions.`. Never restate the standard's content; if you find yourself summarizing it in the block, delete the summary and keep the pointer.
- **No "getting started / setup" section.** Onboarding, install, and first-run steps are not what the agent file is for; the agent already has the repo.
- **Grow it only on an observed failure.** Add a line to the block only when the agent actually did something wrong that this file can prevent. Don't preemptively document mistakes it hasn't made. (See `skillgrid:onboarding` → *Keep the AGENTS.md block lean*.)
- **Layout stays high-level.** A one-line "what's here" map is fine; never point at individual files or paths that churn — they go stale and mislead.
- **The spine is non-negotiable.** The `Workflow` line and the "read config before any Skillgrid skill" line are the entrypoint for the whole pipeline — keep them even though they're the only "how the system works" content here.
- **Nested files for areas.** Area-specific rules (a payments module, a UI kit) live in a nested `<area>/AGENTS.md`; the root block stays global and lean.
