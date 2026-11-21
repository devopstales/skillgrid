# Host agent file

The block onboarding renders into the project's `AGENTS.md` (or `CLAUDE.md`). It points the host agent at the shared rules and at the project's own facts; it is not a second copy of them.

When to load: Phase 0, when writing the project's agent file. Role: orchestrator.

```markdown
# Standards

## Environment & Tooling

- **Language**: {language}
- **Package manager**: {package_manager}
- **Task runner**: {task_runner}
- **Install**: {install}
- **Test**: {test}
- **Lint**: {lint}
- **Format**: {format}
- **Build (dev)**: {build}
- **Run (dev)**: {run}
- **Docker (dev deps)**: {docker}
- **Security scan**: {security_scan}

## Key Directories

{directories_table}

## Engineering Standards

{engineering_standards}

## Security & Escalation Boundaries

{security_boundaries}

## Dependency Policies

{dependency_policies}

## Architecture Constraints

{architecture_constraints}

## Definition of Done

{definition_of_done}

## Issue Tracker

{issue_tracker_line}

## Memory

Mnemonic — reference `.agents/skills/_shared/rules/mnemonic-memory.md`

## Code Indexing

Mnemonic — reference `.agents/skills/_shared/rules/mnemonic-memory.md` (Code Index Ladder)

## Testing standards

Reference `.agents/skills/skillgrid/references/code-standards.md`

## SDD Standards

Reference `.agents/skills/_shared/rules/sdd-structure.md`
```

Rules:

- Fill the placeholders from detected facts, never from guesses. A brownfield repo is read, not assumed.
- Empty values render as `<detect>`; empty lists drop their section.
- Point at the shared rule; do not paste it. A pasted rule goes stale the first time the shared file changes.
- The paths are repo-relative (`.agents/skills/...`), not `~/.agents/...`, unless the host installs skills globally; say which in the project's README.
