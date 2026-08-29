# SDD Workflows

## Skillgrid SDD Workflow

```
init -> explore -> propose -> design -> spec -> tasks -> apply -> verify -> archive
```

**Note**: OpenSpec (`openspec/changes/`) is the canonical artifact store. The `.skillgrid/sdd/` directory provides sub-agent communication scratch space for the `subagent-driven-development` workflow, similar to how superpowers coordinates sub-agents.

```bash
workflow:
  # Canonical artifacts remain in openspec/
  openspec:
    proposal:
      -> openspec/changes/<change-id>/proposal.md
    design:
      -> openspec/changes/<change-id>/design.md
    specs:
      -> openspec/changes/<change-id>/specs/<spec-id>/spec.md
    tasks:
      -> openspec/changes/<change-id>/tasks.md

  # Sub-agent communication in .skillgrid/sdd/
  sdd-scratch:
    init:
      -> .skillgrid/sdd/<project-name>/testing-capabilities.yaml
      -> Mnemonic topic: sdd/<project-name>/testing-capabilities
    explore:
      -> .skillgrid/sdd/<change-id>/context/phase-input.md
      -> .skillgrid/sdd/<change-id>/phase-status/explore-status.md
      -> Mnemonic topic: sdd/<change-id>/explore
    propose:
      -> .skillgrid/sdd/<change-id>/context/phase-input.md
      -> .skillgrid/sdd/<change-id>/phase-status/propose-status.md
      -> Mnemonic topic: sdd/<change-id>/proposal
    design:
      -> .skillgrid/sdd/<change-id>/context/phase-input.md
      -> .skillgrid/sdd/<change-id>/phase-status/design-status.md
      -> Mnemonic topic: sdd/<change-id>/design
    spec:
      -> .skillgrid/sdd/<change-id>/context/phase-input.md
      -> .skillgrid/sdd/<change-id>/phase-status/spec-status.md
      -> Mnemonic topic: sdd/<change-id>/spec/<capability>
    tasks:
      -> .skillgrid/sdd/<change-id>/context/phase-input.md
      -> .skillgrid/sdd/<change-id>/phase-status/tasks-status.md
      -> .skillgrid/sdd/<change-id>/work-unit/assignments.md
      -> Mnemonic topic: sdd/<change-id>/tasks
    apply:
      -> .skillgrid/sdd/<change-id>/context/phase-input.md
      -> .skillgrid/sdd/<change-id>/phase-status/apply-status.md
      -> .skillgrid/sdd/<change-id>/work-unit/results/
      -> .skillgrid/sdd/<change-id>/progress/task-progress.md
      -> .skillgrid/sdd/<change-id>/progress/evidence/
      -> Mnemonic topic: sdd/<change-id>/apply-progress
    verify:
      -> .skillgrid/sdd/<change-id>/context/phase-input.md
      -> .skillgrid/sdd/<change-id>/phase-status/verify-status.md
      -> .skillgrid/sdd/<change-id>/progress/evidence/
      -> Mnemonic topic: sdd/<change-id>/verify-report
    archive:
      -> .skillgrid/sdd/<change-id>/context/phase-input.md
      -> .skillgrid/sdd/<change-id>/phase-status/archive-status.md
      -> Mnemonic topic: sdd/<change-id>/archive-report
```

```bash
# Canonical artifact store (unchanged)
openspec/
├── changes/
│   └── <change-id>/
│       ├── proposal.md
│       ├── design.md
│       ├── specs/
│       │   └── <capability>/
│       │       └── spec.md
│       ├── tasks.md
│       └── adr.md
├── specs/
└── adr/

# Sub-agent communication scratch space
.skillgrid/sdd/
├── <project-name>/
│   └── testing-capabilities.yaml
├── <change-id>/
│   ├── context/
│   │   └── phase-input.md
│   ├── phase-status/
│   │   ├── init-status.md
│   │   ├── explore-status.md
│   │   ├── propose-status.md
│   │   ├── design-status.md
│   │   ├── spec-status.md
│   │   ├── tasks-status.md
│   │   ├── apply-status.md
│   │   ├── verify-status.md
│   │   └── archive-status.md
│   ├── work-unit/
│   │   ├── assignments.md
│   │   └── results/
│   └── progress/
│       ├── task-progress.md
│       └── evidence/
└── registry/
```

Mnemonic topics (persistent coordination state):
```bash
sdd/<project-name>/testing-capabilities
sdd/<change-id>/explore
sdd/<change-id>/proposal
sdd/<change-id>/design
sdd/<change-id>/spec/<capability>
sdd/<change-id>/tasks
sdd/<change-id>/apply-progress
sdd/<change-id>/verify-report
sdd/<change-id>/archive-report
```

### Artifact Map

| Workflow Stage | Canonical Artifact (OpenSpec) | Sub-Agent Scratch (.skillgrid/sdd/) | Mnemonic Topic |
|---|---|---|---|
| init | — | `testing-capabilities.yaml` | `sdd/<project>/testing-capabilities` |
| explore | — | `context/phase-input.md`, `phase-status/explore-status.md` | `sdd/<change-id>/explore` |
| propose | `openspec/changes/<id>/proposal.md` | `context/phase-input.md`, `phase-status/propose-status.md` | `sdd/<change-id>/proposal` |
| design | `openspec/changes/<id>/design.md` | `context/phase-input.md`, `phase-status/design-status.md` | `sdd/<change-id>/design` |
| spec | `openspec/changes/<id>/specs/<cap>/spec.md` | `context/phase-input.md`, `phase-status/spec-status.md` | `sdd/<change-id>/spec/<capability>` |
| tasks | `openspec/changes/<id>/tasks.md` | `context/phase-input.md`, `phase-status/tasks-status.md`, `work-unit/assignments.md` | `sdd/<change-id>/tasks` |
| apply | — | `context/phase-input.md`, `phase-status/apply-status.md`, `work-unit/results/`, `progress/task-progress.md`, `progress/evidence/` | `sdd/<change-id>/apply-progress` |
| verify | — | `context/phase-input.md`, `phase-status/verify-status.md`, `progress/evidence/` | `sdd/<change-id>/verify-report` |
| archive | — | `context/phase-input.md`, `phase-status/archive-status.md` | `sdd/<change-id>/archive-report` |

### Skill Dependency

* init - `sdd-init`
  * `using-skillgrid`
  * `project-context`
  * `mnemonic-memory`
  * `mnemonic-code-index`
* explore - `sdd-explore`
  * `mnemonic-memory-protocol`
  * `systematic-debugging`
  * `gitnexus-impact-analysis`
* propose - `sdd-propose`
  * `brainstorming`
  * `grill-me`
* design - `sdd-design`
  * `brainstorming`
  * `architectural-decision-records`
  * `c4-diagrams`
* spec - `sdd-spec`
  * `gherkin-authoring`
  * `spec-as-source`
  * `glossary`
* tasks - `sdd-tasks`
  * `write-tasks`
  * `openspec-sync-specs`
* apply - `sdd-apply`
  * `subagent-driven-development`
  * `executing-tasks`
  * `test-driven-development`
* verify - `sdd-verify`
  * `verification-before-completion`
  * `acceptance-test-authoring`
  * `gitnexus-impact-analysis`
* archive - `sdd-archive`
  * `verification-before-completion`
  * `gitnexus-cli`

## Discarded / Replaced Skills

The following gentleman-ai skills are **not used** in the Skillgrid SDD workflow. They are replaced by Mnemonic-native equivalents or integrated into the new SDD phase skills:

| gentleman-ai Skill | Status | Replacement |
|---|---|---|
| `engram-memory` | Discarded | `mnemonic-memory` |
| `engram-memory-protocol` | Discarded | `mnemonic-memory-protocol` |
| `engram-sdd-flow` | Discarded | New `sdd-*` phase skills |
| `engram-*` (all domain skills) | Discarded | Not part of SDD core; available as optional delegates |
| gentleman-ai `sdd-init` | Replaced | New `sdd-init` (Mnemonic-native, `.skillgrid/sdd/` scratch) |
| gentleman-ai `sdd-propose` | Replaced | New `sdd-propose` (reads/writes `openspec/changes/`, scratch in `.skillgrid/sdd/`) |
| gentleman-ai `sdd-apply` | Replaced | New `sdd-apply` (no `mem_update`; uses Mnemonic upsert + `.skillgrid/sdd/` scratch) |
| gentleman-ai `sdd-archive` | Replaced | New `sdd-archive` (mechanical copy via shell, status in `.skillgrid/sdd/`) |

## Retained / Separate Workflows

The following skills and structures are **untouched** by the SDD integration:

| Skill / Path | Role |
|---|---|
| `openspec-*` skills | Primary filesystem planning workflow; canonical artifacts in `openspec/changes/` |
| `openspec/` directory | Source of truth for all OpenSpec artifacts |
| `.agents/skills/openspec-*` | OpenSpec CLI integration; unchanged |
| `brainstorming` | Reused by `sdd-propose` and `sdd-design` for artifact generation into `openspec/changes/` |
| `write-tasks` | Reused by `sdd-tasks` for task decomposition into `openspec/changes/` |
| `architectural-decision-records` | Reused by `sdd-design` for ADR creation into `openspec/changes/` |

## What Each Skill Does

| Skill | What It Does |
|---|---|
| `sdd-init` | Detects project stack, resolves SDD mode (`memory`/`filesystem`/`hybrid`/`none`), builds `.skillgrid/skill-registry.md`, persists testing capabilities to Mnemonic and `.skillgrid/sdd/` |
| `sdd-explore` | Indexes codebase via Mnemonic `code_*`, researches constraints, caches findings via Mnemonic `web_*`, produces exploration findings in `.skillgrid/sdd/<id>/phase-status/explore-status.md` |
| `sdd-propose` | Reads existing OpenSpec context from `openspec/changes/`, creates `proposal.md` with intent, scope, capabilities, approach, risks, rollback; writes status to `.skillgrid/sdd/<id>/phase-status/propose-status.md` and Mnemonic |
| `sdd-design` | Reads proposal from `openspec/changes/`, creates `design.md` with architecture, patterns, data flow, interfaces, ADRs; writes status to `.skillgrid/sdd/<id>/phase-status/design-status.md`; reuses `brainstorming` and `architectural-decision-records` |
| `sdd-spec` | Reads design from `openspec/changes/`, creates delta specs under `openspec/changes/<id>/specs/` with requirements and scenarios; writes status to `.skillgrid/sdd/<id>/phase-status/spec-status.md`; reuses `gherkin-authoring` and `spec-as-source` |
| `sdd-tasks` | Reads specs from `openspec/changes/`, decomposes into `tasks.md` with checkboxes, workload forecast, chain strategy, PR boundary; writes assignments to `.skillgrid/sdd/<id>/work-unit/assignments.md`; reuses `write-tasks` |
| `sdd-apply` | Reads tasks from `openspec/changes/`, implements with TDD/work-unit evidence, updates `.skillgrid/sdd/<id>/progress/task-progress.md` and Mnemonic `apply-progress` topic; writes results to `.skillgrid/sdd/<id>/work-unit/results/`; reuses `subagent-driven-development`, `executing-tasks`, `test-driven-development` |
| `sdd-verify` | Reads specs and tasks from `openspec/changes/`, validates behavior, produces gate report in `.skillgrid/sdd/<id>/phase-status/verify-status.md` and evidence in `progress/evidence/`; blocks on CRITICAL issues; reuses `verification-before-completion` and `acceptance-test-authoring` |
| `sdd-archive` | Syncs delta specs from `openspec/changes/<id>/specs/` to `openspec/specs/`, mechanically copies archive via `cp -R`/`mv` with `diff -r` verification, persists final state to `.skillgrid/sdd/<id>/phase-status/archive-status.md` and Mnemonic |
| `sdd-onboard` | Guided introduction to the SDD workflow for new users/projects |

