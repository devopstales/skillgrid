---
name: vdd-decompose
description: >
  Convert goals into Beads - hierarchical decomposition into Epics, Issues, and Sub-issues.
license: Apache-2.0
metadata:
  author: skillgrid
  version: "1.2"
triggers:
  - "vdd-decompose"
  - "beads"
  - "decompose goal"
tools:
  - file_system
  - execute_command
---

## When to Use

Use this skill when:
- Starting a new VDD cycle
- Breaking down a complex goal into trackable work
- Integrating with OpenSpec artifacts for `/sdd-apply`
- Need granular verification checkpoints via Beads
- Referenced by `/sdd-apply` command before implementation starts

## Core Philosophy

Beads decomposes goals into atomic work units (beads) that:
- Each bead has verifiable completion criteria
- Beads form dependency chains for parallel execution
- Proactively detect gaps (missing tests, rollbacks, monitoring)

## Execution Process

### 1. Context Gathering

Read and UNDERSTAND:
- `openspec/changes/{change}/proposal.md` → WHY we're doing this
- `openspec/changes/{change}/tasks.md` → WHAT needs to happen
- `openspec/changes/{change}/design.md` (if exists) → HOW it's architected

### 2. Goal Analysis

From the proposal/tasks, identify:
- **Scope boundaries** - What is in/out of scope
- **Complexity signals** - migration, integration, infrastructure keywords
- **Dependencies** - What must exist before this can proceed

### 3. Epic Creation (Beads Pattern)

Each goal becomes an Epic with:
- ID: `epic-<random>`
- Title: The goal statement
- Description: Context and success criteria
- Issues: Array of work items

### 4. Issue Creation with Priority/Type Logic

| Type | When to Use |
|------|-------------|
| setup | Infrastructure, configuration, schema, migrations |
| feature | New functionality, implementation |
| chore | Documentation, cleanup |
| bug | Fix existing issues |

**Priority Logic:**
- P0: Infrastructure/setup, migrations, config, schema
- P1: Core business logic, implementation, tests, docs
- P2: UI polish, nice-to-haves

### 5. Sub-Issue Creation (Atomic Beads)

Each issue broken into atomic sub-issues:

```
- title: Short descriptive name
- description: What and why (2+ sentences)
- verification_criteria: Array of testable conditions
- status: pending | in-progress | verified | failed
```

### 6. Dependency Chain Construction

**Auto-detect dependencies:**
- Sequential within category (Task 1.2 blocks on 1.1)
- DB/migrations → block → API/business logic
- Config/setup → block → implementation
- Implementation → related → tests (parallel work, not blocking)

### 7. Proactive Gap Detection

**Auto-detect missing:**
- Rollback plans for migrations
- Tests for API implementations
- Error handling for external services
- Monitoring/metrics for features

### 8. Beads Storage

```bash
# Beads storage - uses .beads directory
BEADS_PATH=".beads"
mkdir -p "$BEADS_PATH"

# Store bead definitions as JSON
echo "$BEAD_JSON" > "$BEADS_PATH/{change-name}-beads.json"
```

## Anti-Patterns to AVOID

- [ ] Don't create 50 issues for 5 tasks
- [ ] Don't ignore missing test tasks
- [ ] Don't make everything high priority
- [ ] Don't lose the "why" - reference proposal.md
- [ ] Don't skip gap detection - add missing rollback/test beads

## Integration with SDD Commands

**Referenced by:** `/sdd-apply` - run before implementation starts
**Input:** OpenSpec change directory
**Output:** Beads in `.beads/`

For full Beads sync to issue tracker (optional):
- If `beads_enable: true` in config, also run beads-sync skill
- Otherwise, beads remain as local tracking structure