# Workflows

## Simple OpenSpec workflod

## Skillgrid Workflows

```
setup -> proposal -> design -> specs -> adr -> tasks -> apply -> acceptance -> ship
```

```bash
workflow:
  proposal:
    -> openspec/changes/<change-id>/proposal.md
  design:
    -> openspec/changes/<change-id>/design.md
  specs:
    -> openspec/changes/<change-id>/specs/<spec-id>/spec.md
  adr:
    -> openspec/changes/<change>/adr.md
    -> openspec/adr/YYYY-MM-DD-<topic>.md
  tasks:
    -> openspec/changes/<change-id>/tasks.md
```

```bash
openspec/adr/YYYY-MM-DD-<topic>.md
openspec/changes/archive/
openspec/changes/<change-id>/
openspec/changes/<change-id>/adr.md
openspec/changes/<change-id>/design.md
openspec/changes/<change-id>/proposal.md
openspec/changes/<change-id>/tasks.md
openspec/changes/<change-id>/specs/
openspec/changes/<change-id>/specs/<spec-id>/spec.md
```

### Artifact Map

| Workflow Stage | Skill | Creates | From Template |
|----------------|-------|---------|---------------|
| setup | `project-context` | `AGENTS.md` | — |
| setup | `project-context` | `openspec/config.yaml` (incl. `stack:` for acceptance) | — |
| setup | `using-skillgrid` | — | — |
| proposal | `brainstorming` | `openspec/changes/<change-id>/proposal.md` | `openspec/schemas/intent-driven/templates/proposal.md` |
| design | `brainstorming` | `openspec/changes/<change-id>/design.md` | `openspec/schemas/intent-driven/templates/design.md` |
| specs | `spec-as-source` + `gherkin-authoring` | `openspec/changes/<change-id>/specs/<spec-id>/spec.md` | `.agents/skills/spec-as-source/references/spec.md` (fenced Gherkin — overrides schema template) |
| adr | `architectural-decision-records` | `openspec/changes/<change-id>/adr.md` | `openspec/schemas/intent-driven/templates/adr.md` |
| adr | `architectural-decision-records` | `openspec/adr/YYYY-MM-DD-<topic>.md` | `.agents/skills/architectural-decision-records/templates/` (mad full/minimal, nygard, y-statement, custom) |
| tasks | `write-tasks` | `openspec/changes/<change-id>/tasks.md` | `openspec/schemas/intent-driven/templates/tasks.md` |
| apply | `subagent-driven-development` or `executing-tasks` | — | — |
| acceptance | `acceptance-test-authoring` (per `stack:`) | `.skillgrid/acceptance-tests/` (runner, step defs, page objects) | `.agents/skills/acceptance-test-authoring/references/<stack>/SETUP.md` |
| ship | — | — | — |

### Skill Dependecy

* setup - `using-skillgrid`
  * `project-context`
  * `setup-skillgrid-skills`
* proposal - `brainstorming`
* specs - `spec-as-source`
  * `gherkin-authoring`
* adr - `architectural-decision-records`
* tasks - `write-tasks`
  * `openspec-sync-specs`
  * manage tickets
* acceptance - `acceptance-test-authoring`
  * `openspec-verify-change`
  * `openspec-git-discipline` - merge-before-archive
* ship - ???