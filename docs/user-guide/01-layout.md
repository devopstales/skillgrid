# Layout

What lives in the hub and where agents pick it up.

## Quick path

```text
skillgrid-skills/
├── .agents/
│   ├── skills/        # 28 skills (SKILL.md + references/ + templates/ + scripts/)
│   ├── hooks/         # 8 hook scripts (bash) — the enforcement logic
│   └── git-hooks/     # 4 thin shims (2 git hooks + 2 agent Stop hooks)
├── docs/
│   ├── user-guide/    # this guide
│   ├── plans/         # planning notes (workflow plan, companion)
│   └── assets/
├── README.md
└── LICENSE
```

## `.agents/skills/`

Each skill is a directory with a `SKILL.md` plus optional `references/`, `templates/`, and `scripts/`. Agents read the `SKILL.md` when the task matches the skill's description.

| Kind | Examples |
|------|----------|
| **Entry / router** | `using-skillgrid` |
| **Workflow stages** | `onboarding`, `writing-blueprints`, `slicing`, `qa`, `ticketing` |
| **Discovery** | `interviewing`, `brainstorming`, `research`, `deep-research`, `spike`, `sketch` |
| **Execution** | `simple-execution`, `subagent-execution`, `parallel-execution`, `work-unit-commits`, `resume` |
| **Quality** | `test-driven-development`, `test-driven-verification`, `acceptance-test-authoring`, `requesting-code-review`, `receiving-code-review`, `parallel-code-review` |
| **Cross-cutting** | `mnemonic`, `isolated-workspace`, `structured-debugging`, `architectural-decision-records`, `ponytail` |

## `.agents/hooks/`

The enforcement logic. `checkpoint-state.sh` is the dispatcher; the rest are the individual guards and the agent Stop hooks. See [Hooks](04-hooks.md).

## `.agents/git-hooks/`

Thin shims (`set -euo pipefail`, resolve their own dir, `exec` into `../hooks/`). Two are git hooks (`pre-commit`, `commit-msg`); two are agent-harness Stop hooks (`stop`, `gate-stop`) that live in the same directory. `install-hooks.sh` copies per-repo shims into the repo's active hooks dir, honoring an existing `core.hooksPath`.

## Project config

Runtime state for a project lives under `.skillgrid/` in the target repo:

| Path | Role |
|------|------|
| `.skillgrid/config.yaml` | Project facts: stack, `testing.runner`, tracker, `bdd.specs_dir`, `worktree_dir` |
| `.skillgrid/sdd/checkpoint.json` | Derived resume handle (from the last commit's `[skillgrid-context]` block) |
| `.skillgrid/specs/**` | `acceptance.feature` files (BDD spec zone) |

Hooks and skills read these at runtime. The `[skillgrid-context]` block in a commit body is the **durable** record; `checkpoint.json` is regenerated from `git log -1`, so it cannot drift from history.

## Naming conventions

- Skill dir = skill name (kebab-case). Frontmatter `name:` matches the dir for 26 of 28 skills; `acceptance-test-authoring` and `mnemonic` carry no frontmatter (prose header instead).
- Gates are `G<n>`; steps/tickets are `NN-name`; never reuse or renumber after creation.

## Checklist

- [ ] Target repo has `.skillgrid/config.yaml`
- [ ] Hooks installed via `install-hooks.sh` (git hooks + `--with-stop` for agent hooks)
- [ ] `using-skillgrid` is the first skill an agent loads

## Next step

[Skills](02-skills.md) — what each skill owns.
