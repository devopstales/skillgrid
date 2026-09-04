# Agent config block (shared payload)

Single source of truth for the `## Agent skills` block that `sdd-init` writes into a project's agent config file (`AGENTS.md`, `CLAUDE.md`, or `GEMINI.md`). The per-target files ([agents.md](agents.md), [claude.md](claude.md), [gemini.md](gemini.md)) only decide *which file* and *what platform note* to use — they must not drift in content. Keep this file the one place the payload lives; change wording here, not in the target wrappers.

The block already lives in the target file in most projects. The sentinels make the write idempotent: update in place, never duplicate.

## The block

Copy exactly, fill the `{placeholders}`, wrap between the sentinels. Do not add prose beyond it — these lines load into context in many tools on every run.

```markdown
<!-- skillgrid-sdd:start -->
## Agent skills

Skillgrid SDD is active in this repo. The workflow, registry, and tracker below are the source of truth for agent work here.

### Workflow
`init → explore → propose → spec → apply → verify → archive`

Entry: invoke **`use-skillgrid`** for change work (uninitialized → `sdd-init`; else → `sdd-explore` then down the pipeline). No platform hook required.

- Skill registry (index of installed skills + triggers): `{registry}`
- Project facts (stack, testing, tracker, conventions): `docs/skillgrid/config.yaml` and Mnemonic (`sdd/{project}/…`)
- Triage labels: `docs/skillgrid/agents/issue-tracker.md` + the tracker's label map

### Issue tracker
{tracker-line}
<!-- skillgrid-sdd:end -->
```

## Placeholders

| placeholder | default | fill from |
|---|---|---|
| `{project}` | — | detected `project_name` (Mnemonic `sdd/{project}/project_name`) |
| `{registry}` | `docs/skillgrid/agents/skill-registry.md` | sdd-init step 5.1 — fixed |
| `{tracker-doc}` | `docs/skillgrid/agents/issue-tracker.md` | sdd-init step 5.3 — fixed |
| `{tracker-line}` | — | one-line tracker summary, chosen per active tracker |

`{tracker-line}` — pick the active tracker (identifiers from `init → tracker`):

- Backlog.md: `Issues: Backlog.md tickets under `.backlog/tasks/` (backlog CLI).`
- GitHub: `Issues: GitHub Issues (gh CLI, this repo).`
- GitLab: `Issues: GitLab Issues (glab CLI, this repo).`
- Jira: `Issues: Jira (jira CLI, project key {jira-key}).`

## Idempotent upsert (required)

1. Search the target file for `<!-- skillgrid-sdd:start -->`.
2. **Found** → replace everything from that marker to `<!-- skillgrid-sdd:end -->` (inclusive) with the freshly rendered block. Never append a second copy.
3. **Not found** → append the block at the end of the file, per the target file's placement rules.
4. Never create a second root config file: if `AGENTS.md` exists, update it; else create the one the user chose (see [agents.md](agents.md), [claude.md](claude.md), [gemini.md](gemini.md)).

## Gotchas

- The sentinels are HTML comments — invisible in rendered markdown, safe in AGENTS/CLAUDE/GEMINI alike. Do not replace them with visible headings.
- If both `AGENTS.md` and a platform file (`CLAUDE.md`/`GEMINI.md`) exist, write the full block to `AGENTS.md` (source of truth) and put only a one-line pointer in the platform file. Two full blocks = two sources that drift.
