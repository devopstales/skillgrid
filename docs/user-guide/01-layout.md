# Layout

What lives in the hub and where agents pick it up.

## Quick path

```text
skillgrid/
├── .agents/
│   └── skills/        # 28 skills (SKILL.md + references/ + templates/ + scripts/)
├── hooks/             # 9 hook scripts (bash) — the enforcement logic
├── git-hooks/         # 4 thin shims (2 git hooks + 2 agent Stop hooks)
├── plugins/           # agent capture-bridge plugins (opencode, kilo, cursor)
├── docs/
│   ├── user-guide/    # this guide
│   ├── plans/         # planning notes (workflow plan, companion)
│   └── assets/
├── README.md
└── LICENSE
```

`skillgrid install` mirrors the whole repo tree (`$REPO/*`, recursively) to
`~/.skillgrid/` and wires git's `core.hooksPath` at the mirrored git-hooks:
plugins install from `~/.skillgrid/plugins`, git hooks run from
`~/.skillgrid/git-hooks`, implementations are used (copied) from
`~/.skillgrid/hooks`. The shims resolve `../hooks/` relatively, so they work
identically from the repo checkout and the mirror. Excluded from the mirror:
`.git` and `node_modules` (any depth); top-level `.skillgrid`, `dist`, `out`
(worktree state, not content). Never touched at the destination: `mnemonic`,
`repos`, `backup`, `bin`, `tmp`, `logs`, `config.d` (live operational state).

## `.agents/skills/`

Each skill is a directory with a `SKILL.md` plus optional `references/`, `templates/`, and `scripts/`. Agents read the `SKILL.md` when the task matches the skill's description.

| Kind | Examples |
|------|----------|
| **Entry / router** | `using-skillgrid` |
| **Workflow stages** | `onboarding`, `writing-blueprints`, `slicing`, `qa`, `ticketing` |
| **Discovery** | `interviewing`, `brainstorming`, `research`, `code-research`, `spike`, `sketch` |
| **Execution** | `simple-execution`, `subagent-execution`, `parallel-execution`, `work-unit-commits`, `resume` |
| **Quality** | `test-driven-development`, `test-driven-verification`, `acceptance-test-authoring`, `requesting-code-review`, `receiving-code-review`, `parallel-code-review` |
| **Cross-cutting** | `mnemonic`, `isolated-workspace`, `structured-debugging`, `architectural-decision-records`, `ponytail` |

## `hooks/`

The enforcement logic. `checkpoint-state.js` is the dispatcher; the rest are the individual guards and the agent Stop hooks. See [Hooks](04-hooks.md).

## `git-hooks/`

Thin shims (`set -euo pipefail`, resolve their own dir, `exec` into `../hooks/`). Two are git hooks (`pre-commit`, `commit-msg`); two are agent-harness Stop hooks (`stop`, `gate-stop`) that live in the same directory. `skillgrid install` stages them to `~/.skillgrid/git-hooks/` alongside a copy of `hooks/`, and points git's global `core.hooksPath` at the staged dir.

## `plugins/`

Harness adapters (opencode, kilo) plus shared memory protocol. Cursor’s plugin
surface lives at the **repo root** (`.cursor-plugin/`, `rules/mnemonic.mdc`,
`hooks/hooks-cursor.json`) — add-from-folder selects the skillgrid checkout.
OpenCode/Kilo adapters install from the staged copy at `~/.skillgrid/plugins/`.

## Project config

Runtime state for a project lives under `.skillgrid/` in the target repo:

| Path | Role |
|------|------|
| `.skillgrid/config.yaml` | Project facts: stack, `testing.runner`, tracker, `bdd.specs_dir`, `worktree_dir` |
| `.skillgrid/sdd/checkpoint.json` | Derived resume handle (from the last commit's `[skillgrid-context]` block) |
| `.skillgrid/specs/YYYY-MM-DD-<topic>/` | Active change artifacts (spec zone, committed): `briefing.md`, `acceptance.feature`, `blueprint.md`, `tasks.md`, `findings.md`, `qa-report.md` |
| `.skillgrid/state.yaml` | Pipeline state: `current_phase`, `current_change`, `completed_changes` |
| `.skillgrid/.lock` | Advisory session lock (JSON: `change`, `session_id`, `acquired_at`) — surfaces concurrent-session conflicts |
| `.skillgrid/WINDOWS.md` | Cross-change defect register — advisory findings (WARNING/SUGGESTION) not fixed before archive |

Hooks and skills read these at runtime. The `[skillgrid-context]` block in a commit body is the **durable** record; `checkpoint.json` is regenerated from `git log -1`, so it cannot drift from history.

### Multi-phase changes

When a blueprint covers multiple independent phases (each with its own tasks, deliverables, and execution wave), it splits into **one change folder per phase** as siblings under `.skillgrid/specs/`, plus a parent folder holding the shared blueprint index and shared context:

```text
.skillgrid/specs/
  2026-10-01-kubedash-5.0/              ← parent: blueprint index + shared context
    blueprint.md                         ← header (status, tier, goal, architecture,
                                            tech stack, spec ref) + phases table
                                            linking to each phase folder + shared
                                            context link
    shared/
      context.md                         ← hypothesis, terms, must-haves, global
                                            constraints, file structure (shared
                                            across all phases)
  2026-10-01-kubedash-5.0-phase-1/      ← change: Phase 1 (e.g. MVP)
    spec.md                              ← phase header (back-links to
                                            ../<parent>/blueprint.md and
                                            ../<parent>/shared/context.md) +
                                            phase goal + all tasks for this phase
  2026-10-01-kubedash-5.0-phase-2/      ← change: Phase 2
    spec.md
  2026-10-01-kubedash-5.0-phase-3/      ← change: Phase 3
    spec.md
  2026-10-01-kubedash-5.0-phase-4/      ← change: Phase 4
    spec.md
```

Rules:

- Each phase folder is a **valid standalone change** — it has its own `spec.md` with the phase's tasks (TDD shape, SATISFIES BDD links, one-way-door tags).
- The parent folder is **not a change** — it holds the blueprint index and `shared/context.md` that every phase links to.
- Phase folder naming: `<parent-change-name>-phase-<N>` (e.g. `2026-10-01-kubedash-5.0-phase-1`).
- Back-links in each phase `spec.md` use relative paths: `../<parent>/blueprint.md` and `../<parent>/shared/context.md`.
- The phases table in the parent `blueprint.md` uses relative links: `../<parent>-phase-N/`.
- Do **not** nest phase folders inside the parent — they must be siblings at the `.skillgrid/specs/` level.
- Each phase is sliced, executed, and shipped independently via its own change folder.

This structure applies when a blueprint is large enough to warrant phase separation (typically 4+ phases or 20+ tasks). For smaller changes, use the standard single-folder layout above.

## Naming conventions

- Skill dir = skill name (kebab-case). Frontmatter `name:` matches the dir for 26 of 28 skills; `acceptance-test-authoring` and `mnemonic` carry no frontmatter (prose header instead).
- Gates are `G<n>`; steps/tickets are `NN-name`; never reuse or renumber after creation.

## Checklist

- [ ] Target repo has `.skillgrid/config.yaml`
- [ ] Hooks installed via `skillgrid install` (stages `hooks/`, `git-hooks/`, `plugins/` to `~/.skillgrid/` and wires `core.hooksPath`)
- [ ] `using-skillgrid` is the first skill an agent loads

## Next step

[Skills](02-skills.md) — what each skill owns.
