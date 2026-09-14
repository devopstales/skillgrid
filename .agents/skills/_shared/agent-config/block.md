# Agent config block (shared payload)

Single source of truth for the `## Skillgrid` block that `onboarding` writes into a project's agent config file (`AGENTS.md` or `CLAUDE.md`). The per-target files only decide *which file* to use — they must not drift in content.

The block already lives in the target file in most projects. The sentinels make the write idempotent: update in place, never duplicate.

## The block

Copy exactly, fill the `{placeholders}`, wrap between the sentinels. Do not add prose beyond it — these lines load into context in many tools on every run.

```markdown
<!-- skillgrid:start -->
## Skillgrid

This project is configured with Skillgrid. Project: **{project}**.

Config: `.skillgrid/config.yaml` — read it before running any Skillgrid skill.

### Artifacts

| Artifact | Path |
|----------|------|
| Specs (briefing, blueprint, tasks) | `.skillgrid/specs/` |
| Execution ledger | `.skillgrid/sdd/` (gitignored) |
| PRD | `docs/PRD.md` |
| Architecture | `docs/ARCHITECTURE.md` |
| Domain glossary | `.skillgrid/glossary/` (business.md + technical.md) |
| ADRs | `.skillgrid/adr/` |

**Domain model:** Before designing or implementing, read `.skillgrid/glossary/` for the project's vocabulary and the ADRs in `.skillgrid/adr/` for the area you're touching.

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
