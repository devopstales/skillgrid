# SDD Integration Proposal for Skillgrid

## Intent

Integrate gentleman-ai's Spec-Driven Development (SDD) methodology into the Skillgrid ecosystem as a subagent-driven workflow layer on top of OpenSpec. OpenSpec remains the main store for specs and artifacts in `openspec/changes/`. The `.skillgrid/sdd/` directory serves as sub-agent communication scratch space, similar to how superpowers coordinates sub-agents, while Mnemonic provides persistent memory for the SDD phase contracts.

## Scope

### In Scope

- Keep `openspec/changes/` as the primary artifact store for specs and design documents, managed by existing `openspec-*` skills
- Introduce `.skillgrid/sdd/` as sub-agent communication scratch space for the `subagent-driven-development` workflow, similar to superpowers coordination patterns
- Replace Engram filesystem conventions (`sdd/{project}/...`, `sdd/{change-name}/...`) with Mnemonic-native `topic_key` patterns for SDD phase state and progress
- Define hybrid storage modes (`memory | filesystem | hybrid | none`) using Mnemonic + `.skillgrid/sdd/` scratch space, with OpenSpec as the canonical artifact store
- Establish persistent contracts between SDD phases using structured status envelopes passed via Mnemonic and `.skillgrid/sdd/`
- Map existing OpenSpec skills as the source of truth, with new SDD skills orchestrating sub-agents that read from `openspec/changes/`
- Create new SDD orchestration skills: `sdd-init`, `sdd-explore`, `sdd-propose`, `sdd-design`, `sdd-spec`, `sdd-tasks`, `sdd-apply`, `sdd-verify`, `sdd-archive`

### Out of Scope

- Modifying the existing OpenSpec CLI or `openspec/` directory structure
- Breaking changes to existing `openspec-*` skills
- Engram MCP server integration (replaced entirely by Mnemonic)

## Engram → Mnemonic Migration

### What Changes

gentleman-ai's SDD skills persist artifacts as Engram observations using `mem_save` with paths like `sdd/{change-name}/proposal`. In Skillgrid, these become Mnemonic observations using the same `mem_*` tool family but with Skillgrid-native conventions.

### Topic Key Convention

| gentleman-ai (Engram) | Skillgrid (Mnemonic) |
|---|---|
| `sdd/{project}/testing-capabilities` | `sdd/{project}/testing-capabilities` |
| `sdd/{change-name}/proposal` | `sdd/{change-name}/proposal` |
| `sdd/{change-name}/design` | `sdd/{change-name}/design` |
| `sdd/{change-name}/tasks` | `sdd/{change-name}/tasks` |
| `sdd/{change-name}/apply-progress` | `sdd/{change-name}/apply-progress` |
| `sdd/{change-name}/verify-report` | `sdd/{change-name}/verify-report` |
| `sdd/{change-name}/archive-report` | `sdd/{change-name}/archive-report` |

### mem_save Adapter

gentleman-ai uses `mem_update(id, content)` to patch existing observations. Mnemonic does not have `mem_update`. The adapter pattern is:

```javascript
// Read current observation
const current = await mem_get_observation(id);
// Merge new content with existing
const merged = mergeProgress(current.content, newProgress);
// Re-save under the same topic_key (Mnemonic upserts by topic_key)
await mem_save({
  title: current.title,
  type: current.type,
  topic_key: current.topic_key,
  content: merged.content,
  scope: current.scope
});
```

### Filesystem Artifacts

OpenSpec continues to use `openspec/changes/` as its canonical artifact structure. The `.skillgrid/sdd/` directory is reserved for sub-agent communication scratch space when the SDD orchestrator runs phases as sub-agents in `filesystem` or `hybrid` mode, similar to how superpowers coordinates sub-agents. The Engram convention of skipping filesystem writes in `engram` mode maps directly to `memory` mode in Skillgrid.

## OpenSpec Integration

OpenSpec remains the main store for specs and artifacts in `openspec/changes/`. The SDD workflow layers on top of OpenSpec as a subagent-driven orchestration layer. The `.skillgrid/sdd/` directory provides scratch/communication space for sub-agents, similar to how superpowers coordinates sub-agents. Existing `openspec-*` skills continue to manage the canonical artifact store.

### Relationship

```
openspec/ (canonical artifact store)     .skillgrid/sdd/ (sub-agent scratch/communication)
├── changes/                            ├── <change-id>/
│   └── <change-id>/                        ├── context/           (sub-agent handoff)
│       ├── proposal.md                     ├── phase-status/     (status envelopes)
│       ├── design.md                       ├── work-unit/        (task assignments)
│       ├── specs/                          └── progress/         (progress markers)
│       │   └── <capability>/
│       │       └── spec.md
│       └── tasks.md
├── specs/
└── adr/
```

In all modes, `openspec/changes/` is the source of truth for artifacts. The `.skillgrid/sdd/` scratch space is used by sub-agents to communicate phase status, handoff context, and progress markers. In `memory` mode, `.skillgrid/sdd/` files are ephemeral and Mnemonic is the persistent record. In `filesystem` mode, `.skillgrid/sdd/` provides the persistent coordination layer. In `hybrid` mode, both Mnemonic and `.skillgrid/sdd/` persist coordination state.

## Hybrid Storage Modes

| Mode | Mnemonic | `.skillgrid/sdd/` | OpenSpec | Use Case |
|---|---|---|---|---|
| `memory` | Yes | Ephemeral | `openspec/changes/` | Pure Mnemonic coordination; sub-agent scratch is transient |
| `filesystem` | No | Persistent | `openspec/changes/` | Persistent sub-agent coordination via `.skillgrid/sdd/` scratch space |
| `hybrid` | Yes | Persistent | `openspec/changes/` | Mnemonic + `.skillgrid/sdd/` for sub-agent coordination; OpenSpec unchanged |
| `none` | No | No | `openspec/changes/` | Planning only, no SDD coordination persistence |

### Mode Detection

```yaml
# openspec/config.yaml or .skillgrid/config.yaml
sdd:
  mode: hybrid
  strict_tdd: true
  storage:
    primary: mnemonic
    scratch: .skillgrid/sdd
    openspec_mirror: true
```

## Persistent Contract

Each SDD phase returns a structured status envelope consumed by the next phase. This replaces ad-hoc context passing with explicit contracts.

### Status Envelope Schema

```yaml
status: ready | blocked | all_done
executive_summary: string
artifacts:
  - path: string
    observation_id: number | null
    status: pending | done | blocked
next_recommended: string
risks:
  - description: string
    mitigation: string
```

### Phase Contracts

| From Phase | To Phase | Contract Contents |
|---|---|---|
| `sdd-init` | `sdd-explore` | Project context, testing capabilities, stack, mode |
| `sdd-explore` | `sdd-propose` | Findings, risks, constraints, recommended approach |
| `sdd-propose` | `sdd-design` | Intent, scope, capabilities, approach, affected areas |
| `sdd-design` | `sdd-spec` | Architecture decisions, patterns, interfaces |
| `sdd-spec` | `sdd-tasks` | Requirements, scenarios, acceptance criteria |
| `sdd-tasks` | `sdd-apply` | Task list, workload forecast, chain strategy, PR boundary |
| `sdd-apply` | `sdd-verify` | Completed tasks, files changed, deviations, evidence |
| `sdd-verify` | `sdd-archive` | Verification report, gate status, warnings, test results |
| `sdd-archive` | next cycle | Archive report, synced specs, final state |

### Workload Decision Contract

The `sdd-tasks` phase produces a `Review Workload Forecast` that gates `sdd-apply`:

```yaml
review_workload_forecast:
  changed_lines_estimate: 450
  budget_risk: High
  chained_prs_recommended: true
  decision_needed_before_apply: true
  chain_strategy: stacked-to-main | feature-branch-chain | null
  delivery_mode: auto-chain | single-pr | exception-ok | ask-on-risk
```

If `decision_needed_before_apply` is true, `sdd-apply` blocks until the orchestrator/user provides a delivery path.

## `.skillgrid/sdd/` Structure

`.skillgrid/sdd/` is sub-agent communication scratch space for the SDD orchestrator. It is NOT an alternative artifact store. The canonical artifacts (proposal, design, specs, tasks) remain in `openspec/changes/` and are managed by existing `openspec-*` skills. `.skillgrid/sdd/` provides coordination artifacts that sub-agents use to hand off context, report status, and track progress — similar to how superpowers coordinates sub-agents.

### Directory Layout

```
.skillgrid/sdd/
├── <project-name>/
│   └── testing-capabilities.yaml     (filesystem mirror of Mnemonic topic)
├── <change-id>/
│   ├── context/
│   │   └── phase-input.md            (sub-agent handoff: what this phase needs)
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
│   │   ├── assignments.md            (task assignments for sub-agents)
│   │   └── results/                  (sub-agent results)
│   └── progress/
│       ├── task-progress.md          (cumulative checkbox state)
│       └── evidence/                 (test outputs, screenshots)
```

`skill-registry.md` is stored at `.skillgrid/skill-registry.md`.

### File Contents

| File | Purpose | Phase |
|---|---|---|
| `context/phase-input.md` | Sub-agent handoff: inputs, dependencies, constraints for the next phase | All |
| `phase-status/<phase>-status.md` | Status envelope returned by each phase: `status`, `executive_summary`, `artifacts`, `next_recommended`, `risks` | All |
| `work-unit/assignments.md` | Task decomposition with work units, PR boundaries, and chain strategy | tasks → apply |
| `work-unit/results/` | Individual sub-agent results for parallel execution | apply |
| `progress/task-progress.md` | Cumulative task checkbox state and evidence tables | apply |
| `progress/evidence/` | Test outputs, screenshots, runtime harness results | apply, verify |

## Skill Mapping

### Existing Skillgrid Skills (reused)

| Skill | Purpose |
|---|---|
| `using-skillgrid` | Entry point, loads SDD workflow (integrate content of `setup-matt-pocock-skills` and mode selector) |
| `brainstorming` | Proposal and design artifact creation |
| `write-tasks` | Task decomposition from specs |
| `architectural-decision-records` | ADR creation |
| `subagent-driven-development` | Parallel task execution |
| `executing-tasks` | Sequential task execution |
| `test-driven-development` | TDD workflow |
| `acceptance-test-authoring` | Acceptance test setup per stack |
| `openspec-*` | OpenSpec artifacts in `openspec/changes/` via OpenSpec CLI |
| `mnemonic-memory` / `mnemonic-memory-protocol` | Memory persistence protocol |
| `verification-before-completion` | Verification gate |
| `systematic-debugging` | Debug workflow |
| `gitnexus-*` | Code intelligence and impact analysis |

### New SDD Skills (subagent-driven, `.skillgrid/sdd/` artifacts)

| Skill | Description |
|---|---|
| `sdd-init` | Initialize SDD context, detect stack, resolve mode, build registry |
| `sdd-explore` | Understand existing behavior, constraints, and risks |
| `sdd-propose` | Create proposal with intent, scope, capabilities |
| `sdd-design` | Create design with architecture, patterns, interfaces |
| `sdd-spec` | Create specs with requirements and scenarios |
| `sdd-tasks` | Decompose specs into tasks with workload forecast |
| `sdd-apply` | Implement tasks with TDD evidence and work unit evidence |
| `sdd-verify` | Validate behavior against specs, run regression tests |
| `sdd-archive` | Sync delta specs, move to archive, persist final state |
| `sdd-onboard` | Guided SDD workflow introduction |

## Phase Orchestration

### Canonical Order

```
init → explore → propose → design → spec → tasks → apply → verify → archive
```

### Orchestration Rules

1. **Never skip a phase** without explicit rationale recorded in the status envelope
2. **Each phase is a dedicated sub-agent** loaded via the `task()` primitive
3. **The orchestrator** reads the status envelope, resolves the next phase, and launches the corresponding sub-agent
4. **Blocked states** propagate forward — if a phase returns `blocked`, the orchestrator surfaces it to the user
5. **Session continuity** is maintained via Mnemonic `mem_context` at session start

### Sub-Agent Launch Pattern

```javascript
// Entry point: using-skillgrid loads the SDD workflow
// Orchestrator pattern
const phase = nextPhase(statusEnvelope);
const skill = sddSkills[phase];
const result = await task({
  subagent_type: "general",
  prompt: `Load skill ${skill}. ${phase === 'sdd-init' ? '' : `Read context from Mnemonic topics: ${contextTopics}`}`,
  description: `SDD ${phase}`
});
```

## Testing Capabilities Contract

`sdd-init` detects and caches testing capabilities as a structured artifact:

```yaml
testing_capabilities:
  runner: jest | pytest | go-test | cargo-test | etc.
  layers:
    - unit
    - integration
    - e2e
  coverage_tool: jest --coverage | pytest --cov | go test -cover
  linter: eslint | ruff | golangci-lint | etc.
  type_checker: tsc | mypy | etc.
  formatter: prettier | black | etc.
  strict_tdd: true | false
```

This artifact is persisted to:
- Mnemonic: `sdd/{project}/testing-capabilities`
- Filesystem: `.skillgrid/sdd/<project-name>/testing-capabilities.yaml`

## Mnemonic `mem_update` Implementation Plan

gentleman-ai's SDD skills rely on `mem_update(id, content)` to patch existing observations without replacing them. Mnemonic currently lacks this primitive. To preserve compatibility, implement `mem_update` as a Mnemonic-side helper that performs a read/merge/upsert cycle atomically.

### Proposed Implementation

```javascript
// Mnemonic-side helper: mem_update
async function mem_update(id, contentUpdates) {
  const current = await mem_get_observation(id);
  if (!current) throw new Error(`Observation ${id} not found`);
  
  const merged = {
    ...current,
    content: typeof contentUpdates === 'string' 
      ? contentUpdates 
      : mergeDeep(current.content, contentUpdates)
  };
  
  // Re-save under the same topic_key — Mnemonic upserts by topic_key
  await mem_save({
    title: merged.title,
    type: merged.type,
    topic_key: merged.topic_key,
    content: merged.content,
    scope: merged.scope
  });
  
  return merged;
}
```

### Integration Points

| gentleman-ai Skill | Current `mem_update` Usage | Mnemonic Replacement |
|---|---|---|
| `sdd-apply` | Mark tasks complete in `tasks` observation | `mem_update(tasksId, { checkboxes: updatedMarkdown })` |
| `sdd-archive` | Update `tasks` with reconciliation notes | `mem_update(tasksId, { reconciliation: notes })` |
| `sdd-init` | Persist `testing-capabilities` | `mem_save` with `topic_key` upsert (no update needed) |

### Migration Strategy

1. **Phase 1**: Add `mem_update` to Mnemonic MCP as an experimental helper
2. **Phase 2**: Update gentleman-ai skills to use `mem_update` via Mnemonic instead of Engram
3. **Phase 3**: Deprecate Engram-specific paths; all SDD artifacts flow through Mnemonic

### Risks

- **Race condition**: Two concurrent `mem_update` calls on the same observation may overwrite each other. Mitigation: serialize updates per `topic_key` in the orchestrator.
- **Content merge ambiguity**: Deep-merging structured YAML/JSON content requires schema-aware merge logic. Mitigation: use line-based merge for Markdown artifacts, structured merge for YAML.

## Engram Filesystem → Mnemonic Conversion

### Before (gentleman-ai)

```
sdd/
├── my-project/
│   └── testing-capabilities
├── add-dark-mode/
│   ├── explore
│   ├── proposal
│   ├── design
│   ├── spec
│   ├── tasks
│   ├── apply-progress
│   ├── verify-report
│   └── archive-report
└── fix-session-expiry/
    └── ...
```

### After (Skillgrid)

```
openspec/changes/ (canonical artifacts)
├── add-dark-mode/
│   ├── proposal.md
│   ├── design.md
│   ├── specs/
│   │   └── theme/
│   │       └── spec.md
│   └── tasks.md

.skillgrid/sdd/ (sub-agent scratch/communication)
├── add-dark-mode/
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
└── fix-session-expiry/
    └── ...
```

And in Mnemonic:
```
topic_key: sdd/add-dark-mode/proposal
topic_key: sdd/add-dark-mode/design
topic_key: sdd/add-dark-mode/tasks
topic_key: sdd/add-dark-mode/apply-progress
topic_key: sdd/add-dark-mode/verify-report
topic_key: sdd/add-dark-mode/archive-report
```

### Key Differences

1. **OpenSpec is canonical**: All specs, designs, proposals, and tasks live in `openspec/changes/` managed by existing `openspec-*` skills.
2. **`.skillgrid/sdd/` is scratch**: Sub-agent coordination only — context handoff, phase status, work-unit assignments, progress markers.
3. **No `mem_update`**: Mnemonic upserts by `topic_key`. To update, read current observation, merge content, re-save.
4. **Session-scoped**: Mnemonic requires `session_id` from `mem_session_start` for all saves.
5. **Web cache**: Mnemonic adds `web_*` tools for research caching — SDD exploration phase should cache findings.
6. **Code index**: Mnemonic adds `code_*` tools for code search — `sdd-explore` should index the codebase.

## Migration Path

### Phase 1: Foundation

1. Create `sdd-init` skill — detect stack, resolve mode, build registry
2. Create `.skillgrid/sdd/` sub-agent scratch space structure in projects
3. Update `AGENTS.md` with SDD workflow documentation
4. Create Mnemonic adapter for `mem_update` pattern

### Phase 2: Core Phases

5. Create `sdd-explore`, `sdd-propose`, `sdd-design`, `sdd-spec`, `sdd-tasks`
6. Map existing `brainstorming`, `write-tasks`, `architectural-decision-records` as skill delegates

### Phase 3: Execution and Verification

7. Create `sdd-apply` with TDD evidence and work unit evidence tables
8. Create `sdd-verify` with gate logic and CRITICAL issue blocking
9. Create `sdd-archive` with mechanical copy contract

### Phase 4: Polish

10. Create `sdd-onboard` for guided introduction
11. Update `docs/03-workflow.md` with SDD artifact map
12. Update `docs/10-skillgrid-plan.md` with SDD memory section

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| Mnemonic API differences break existing Engram patterns | Medium | Build adapter layer in `sdd-init`; test all `mem_*` calls |
| Hybrid mode conflicts with OpenSpec CLI | Medium | `.skillgrid/sdd/` is independent of `openspec/`; no CLI coupling; OpenSpec artifacts remain untouched |
| Sub-agent orchestration overhead | Low | Phase contracts are lightweight; orchestrator is thin |
| Skill proliferation (9 new skills) | Medium | Reuse existing skills as delegates; thin wrappers for phase boundaries |
| Topic key collisions | Low | Enforce `sdd/{project}/{change-id}/{artifact}` namespace |

## Rollback Plan

If the SDD integration proves disruptive:
1. Disable SDD skills in `.kilo/command/*.md` or project config
2. Existing OpenSpec workflow is untouched — no migration required
3. `.skillgrid/sdd/` directories can be deleted without affecting `openspec/`
4. Mnemonic observations under `sdd/*` can be purged via `mem_purge` (if available) or left as inert data

## Dependencies

- Mnemonic MCP with `mem_*`, `code_*`, `web_*` tool families
- Existing OpenSpec CLI (optional — SDD can run without it in `memory` or `filesystem` mode)
- Skillgrid CLI for `skillgrid install` (recommended for skill distribution)

## Success Criteria

- [ ] All 9 SDD skills created and loadable via `skill()`
- [ ] `sdd-init` correctly detects stack, resolves mode, and persists testing capabilities to Mnemonic
- [ ] Full SDD cycle (init → explore → propose → design → spec → tasks → apply → verify → archive) completes on a sample project
- [ ] Hybrid mode coordination state appears in both `.skillgrid/sdd/` scratch space and Mnemonic; OpenSpec artifacts remain in `openspec/changes/`
- [ ] Filesystem mode works without Mnemonic (`.skillgrid/sdd/` scratch space persists coordination state)
- [ ] Memory mode works without filesystem writes (pure Mnemonic observations)
- [ ] Topic keys follow `sdd/{project}/{change-id}/{artifact}` convention
- [ ] Phase status envelopes propagate correctly between sub-agents
- [ ] `sdd-archive` mechanical copy contract passes `diff -r` verification
- [ ] Existing OpenSpec workflow in `openspec/` is unaffected by SDD addition
