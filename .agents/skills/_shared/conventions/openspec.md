# OpenSpec File Convention (shared across all SDD skills)

## Phase Order

`init → explore → propose → design → spec → tasks → apply → verify → archive`.

Design runs **before** spec: the design decides what to build, and the delta spec (`ADDED`/`MODIFIED`/`REMOVED`/`RENAMED`) is expressed relative to that design. `sdd-spec` consumes `design.md`, not just `proposal.md`.

## Directory Structure

```
openspec/
├── config.yaml              <- Project-specific SDD config
├── specs/                   <- Source of truth (main specs)
│   └── {domain}/
│       └── spec.md
└── changes/                 <- Active changes
    ├── archive/             <- Completed changes (YYYY-MM-DD-{change-name}/)
    └── {change-name}/       <- Active change folder
        ├── state.yaml       <- DAG state (survives compaction)
        ├── exploration.md   <- (optional) from sdd-explore
        ├── proposal.md      <- from sdd-propose
        ├── design.md        <- from sdd-design
        ├── specs/           <- from sdd-spec (consumes design.md)
        │   └── {domain}/
        │       └── spec.md  <- Delta spec
        ├── tasks.md         <- from sdd-tasks (updated by sdd-apply)
        └── verify-report.md <- from sdd-verify
```

## Artifact File Paths

| Skill | Phase order | Creates / Reads | Path |
|-------|------------|----------------|------|
| orchestrator | — | Creates/Updates | `openspec/changes/{change-name}/state.yaml` |
| sdd-init | 1 | Creates | `openspec/config.yaml`, `openspec/specs/`, `openspec/changes/`, `openspec/changes/archive/` |
| sdd-explore | 2 | Creates (optional) | `openspec/changes/{change-name}/exploration.md` |
| sdd-propose | 3 | Creates | `openspec/changes/{change-name}/proposal.md` |
| sdd-design | 4 | Creates (reads proposal) | `openspec/changes/{change-name}/design.md` |
| sdd-spec | 5 | Creates (reads proposal + design) | `openspec/changes/{change-name}/specs/{domain}/spec.md` |
| sdd-tasks | 6 | Creates (reads design + delta spec) | `openspec/changes/{change-name}/tasks.md` |
| sdd-apply | 7 | Updates | `openspec/changes/{change-name}/tasks.md` (marks `[x]`) |
| sdd-verify | 8 | Creates | `openspec/changes/{change-name}/verify-report.md` |
| sdd-archive | 9 | Moves + Updates | `openspec/changes/{change-name}/` → `openspec/changes/archive/YYYY-MM-DD-{change-name}/`; merges deltas into `openspec/specs/{domain}/spec.md` |

## Reading Artifacts

```
Proposal:   openspec/changes/{change-name}/proposal.md
Design:     openspec/changes/{change-name}/design.md
Specs:      openspec/changes/{change-name}/specs/  (all domain subdirectories)
Tasks:      openspec/changes/{change-name}/tasks.md
Verify:     openspec/changes/{change-name}/verify-report.md
Config:     openspec/config.yaml
Main specs: openspec/specs/{domain}/spec.md
```

## Writing Rules

- Always create the change directory before writing artifacts
- If a file already exists, READ it first and UPDATE it (don't overwrite blindly)
- If the change directory already exists with artifacts, the change is being CONTINUED
- Use `openspec/config.yaml` `rules` section for project-specific constraints per phase

## Delta Spec Sections

Delta specs MAY include these sections:

```markdown
## ADDED Requirements
## MODIFIED Requirements
## REMOVED Requirements
## RENAMED Requirements
```

- `ADDED` appends new requirements to the main spec.
- `MODIFIED` replaces the full matching requirement block in the main spec. The delta MUST contain the entire updated requirement, including unchanged scenarios that must be preserved.
- `REMOVED` deletes the matching requirement from the main spec. Each removed requirement MUST include `(Reason: ...)` and SHOULD include `(Migration: ...)` when consumers or persisted behavior are affected.
- `RENAMED` changes a requirement heading/name without changing behavior unless the delta also includes a `MODIFIED` block for the new requirement. Each rename MUST state old and new names explicitly.

## Config File Reference

```yaml
# openspec/config.yaml
schema: spec-driven

context: |
  Tech stack: {detected}
  Architecture: {detected}
  Testing: {detected}
  Style: {detected}

rules:
  proposal:
    - Include rollback plan for risky changes
  specs:
    - Use Given/When/Then for scenarios
    - Use RFC 2119 keywords (MUST, SHALL, SHOULD, MAY)
  design:
    - Include sequence diagrams for complex flows
    - Document architecture decisions with rationale
  tasks:
    - Group by phase, use hierarchical numbering
    - Keep tasks completable in one session
  apply:
    guidelines:
      - Follow existing code patterns
    tdd: false           # Set to true to enable RED-GREEN-REFACTOR
    test_command: ""
  verify:
    test_command: ""
    build_command: ""
    coverage_threshold: 0
  archive:
    - Warn before merging destructive deltas
```

## Archive Structure

When archiving, the change folder moves to:
```
openspec/changes/archive/YYYY-MM-DD-{change-name}/
```

Use today's date in ISO format. The archive is an AUDIT TRAIL — never delete or modify archived changes.
