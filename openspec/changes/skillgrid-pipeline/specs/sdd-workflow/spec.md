## ADDED Requirements

### Requirement: SDD phase lifecycle

The system SHALL provide a Spec-Driven Development (SDD) phase lifecycle orchestrated as a subagent-driven layer on top of OpenSpec and Mnemonic.

#### Scenario: Canonical phase order
- **GIVEN** a user starts an SDD change in a project
- **WHEN** the orchestrator runs the workflow
- **THEN** phases execute in the order `init → explore → propose → design → spec → tasks → apply → verify → archive`

#### Scenario: No phase skipped without rationale
- **GIVEN** an SDD phase would be skipped
- **WHEN** the orchestrator advances to the next phase
- **THEN** the skip is recorded explicitly in the phase status envelope with a rationale

#### Scenario: Each phase is a dedicated sub-agent
- **GIVEN** a phase is ready to run
- **WHEN** the orchestrator launches it
- **THEN** the phase executes as a dedicated sub-agent loaded via the `task()` primitive, not inline

#### Scenario: Blocked phase surfaces to user
- **GIVEN** a phase returns a `blocked` status envelope
- **WHEN** the orchestrator resolves the next phase
- **THEN** the blocked state propagates forward and is surfaced to the user

### Requirement: OpenSpec as canonical artifact store

The system SHALL keep `openspec/changes/` as the canonical store for SDD artifacts and SHALL NOT modify the OpenSpec directory structure.

#### Scenario: Proposal persisted to OpenSpec
- **GIVEN** the `sdd-propose` phase completes
- **WHEN** it writes artifacts
- **THEN** the proposal is written to `openspec/changes/<change-id>/proposal.md`

#### Scenario: Design persisted to OpenSpec
- **GIVEN** the `sdd-design` phase completes
- **WHEN** it writes artifacts
- **THEN** the design is written to `openspec/changes/<change-id>/design.md`

#### Scenario: Specs persisted to OpenSpec
- **GIVEN** the `sdd-spec` phase completes
- **WHEN** it writes artifacts
- **THEN** delta specs are written to `openspec/changes/<change-id>/specs/<capability>/spec.md`

#### Scenario: Tasks persisted to OpenSpec
- **GIVEN** the `sdd-tasks` phase completes
- **WHEN** it writes artifacts
- **THEN** the task list is written to `openspec/changes/<change-id>/tasks.md`

#### Scenario: Archive syncs delta specs to main specs
- **GIVEN** the `sdd-archive` phase completes
- **WHEN** it finalizes a change
- **THEN** delta specs from `openspec/changes/<change-id>/specs/` are synced to `openspec/specs/` and the change is moved to archive

#### Scenario: Existing OpenSpec workflow untouched
- **GIVEN** SDD skills are added to a project
- **WHEN** the existing `openspec-*` skills run
- **THEN** they operate unchanged and the `openspec/` directory structure is not modified

### Requirement: `.skillgrid/sdd/` sub-agent scratch space

The system SHALL provide `.skillgrid/sdd/` as sub-agent communication scratch space, distinct from the canonical OpenSpec artifact store.

#### Scenario: Scratch namespace per project and change
- **GIVEN** SDD runs for a project and a change
- **WHEN** sub-agents coordinate
- **THEN** project context lives under `.skillgrid/sdd/<project-name>/` and phase coordination under `.skillgrid/sdd/<change-id>/`

#### Scenario: Phase status envelopes written to scratch
- **GIVEN** an SDD phase completes
- **WHEN** it returns its status envelope
- **THEN** it writes `.skillgrid/sdd/<change-id>/phase-status/<phase>-status.md` (e.g. `explore-status.md`, `apply-status.md`)

#### Scenario: Context handoff between phases
- **GIVEN** one SDD phase hands off to the next
- **WHEN** the next phase sub-agent loads
- **THEN** it reads `.skillgrid/sdd/<change-id>/context/phase-input.md` for inputs, dependencies, and constraints

#### Scenario: Work-unit assignments for apply
- **GIVEN** the `sdd-tasks` phase produces task decomposition
- **WHEN** the `sdd-apply` phase runs
- **THEN** assignments are read from `.skillgrid/sdd/<change-id>/work-unit/assignments.md` and results written to `.skillgrid/sdd/<change-id>/work-unit/results/`

#### Scenario: Progress and evidence tracked
- **GIVEN** the `sdd-apply` and `sdd-verify` phases run
- **WHEN** they record progress
- **THEN** cumulative task state is in `.skillgrid/sdd/<change-id>/progress/task-progress.md` and evidence in `.skillgrid/sdd/<change-id>/progress/evidence/`

#### Scenario: Skill registry location
- **GIVEN** SDD initialization builds the skill registry
- **WHEN** the registry is written
- **THEN** it is stored at `.skillgrid/skill-registry.md`

### Requirement: Hybrid storage modes

The system SHALL support storage modes `memory`, `filesystem`, `hybrid`, and `none` that control how Mnemonic and `.skillgrid/sdd/` persist coordination state.

#### Scenario: Mode configured in config
- **GIVEN** a project configures SDD
- **WHEN** the orchestrator resolves the mode
- **THEN** it reads `sdd.mode` from `openspec/config.yaml` or `.skillgrid/config.yaml` (defaulting to `hybrid`)

#### Scenario: Memory mode
- **GIVEN** `sdd.mode` is `memory`
- **WHEN** SDD runs
- **THEN** Mnemonic holds persistent coordination state and `.skillgrid/sdd/` scratch is ephemeral, while `openspec/changes/` remains canonical

#### Scenario: Filesystem mode
- **GIVEN** `sdd.mode` is `filesystem`
- **WHEN** SDD runs
- **THEN** `.skillgrid/sdd/` provides persistent coordination and Mnemonic is not used

#### Scenario: Hybrid mode
- **GIVEN** `sdd.mode` is `hybrid`
- **WHEN** SDD runs
- **THEN** both Mnemonic and `.skillgrid/sdd/` persist coordination state while `openspec/changes/` is unchanged

#### Scenario: None mode
- **GIVEN** `sdd.mode` is `none`
- **WHEN** SDD runs
- **THEN** no coordination persistence is written and planning artifacts are limited to `openspec/changes/`

### Requirement: Mnemonic topic-key conventions

The system SHALL persist SDD coordination state to Mnemonic using `sdd/{project}/{change-id}/{artifact}` topic keys, replacing gentleman-ai's Engram filesystem conventions.

#### Scenario: Testing capabilities topic
- **GIVEN** `sdd-init` detects testing capabilities
- **WHEN** it persists them
- **THEN** it saves the Mnemonic topic `sdd/<project-name>/testing-capabilities`

#### Scenario: Phase artifact topics
- **GIVEN** an SDD phase completes and persists its artifact
- **WHEN** the Mnemonic observation is saved
- **THEN** it uses `sdd/<change-id>/explore`, `sdd/<change-id>/proposal`, `sdd/<change-id>/design`, `sdd/<change-id>/spec/<capability>`, `sdd/<change-id>/tasks`, `sdd/<change-id>/apply-progress`, `sdd/<change-id>/verify-report`, or `sdd/<change-id>/archive-report`

#### Scenario: Upsert without mem_update
- **GIVEN** an SDD phase must update an existing Mnemonic observation
- **WHEN** it patches progress
- **THEN** it reads the current observation via `mem_get_observation`, merges content, and re-saves under the same `topic_key` (Mnemonic upserts by topic_key)

#### Scenario: Session-scoped saves
- **GIVEN** SDD saves coordination state to Mnemonic
- **WHEN** any `mem_save` call is made
- **THEN** it includes the active `session_id` from `mem_session_start`

### Requirement: Phase status envelope contract

The system SHALL pass a structured status envelope between consecutive SDD phases instead of ad-hoc context.

#### Scenario: Envelope schema
- **GIVEN** an SDD phase returns its result
- **WHEN** the next phase consumes it
- **THEN** the envelope contains `status` (`ready`|`blocked`|`all_done`), `executive_summary`, `artifacts`, `next_recommended`, and `risks`

#### Scenario: Handoff contents per phase
- **GIVEN** one phase completes and the next begins
- **WHEN** the orchestrator launches the next sub-agent
- **THEN** it passes the documented contract contents (e.g. `sdd-tasks → sdd-apply` carries task list, workload forecast, chain strategy, and PR boundary)

#### Scenario: Workload forecast gates apply
- **GIVEN** `sdd-tasks` produces a `review_workload_forecast` with `decision_needed_before_apply: true`
- **WHEN** the `sdd-apply` phase starts
- **THEN** `sdd-apply` blocks until the orchestrator or user provides a delivery path

### Requirement: SDD phase skills as subagent orchestrators

The system SHALL provide the SDD phase skills `sdd-init`, `sdd-explore`, `sdd-propose`, `sdd-design`, `sdd-spec`, `sdd-tasks`, `sdd-apply`, `sdd-verify`, `sdd-archive`, and `sdd-onboard` that reuse existing Skillgrid skills as delegates.

#### Scenario: Init resolves mode and registry
- **GIVEN** `sdd-init` runs
- **WHEN** it bootstraps a project
- **THEN** it detects the stack, resolves the storage mode, builds `.skillgrid/skill-registry.md`, and persists testing capabilities to Mnemonic and `.skillgrid/sdd/`

#### Scenario: Explore indexes and caches
- **GIVEN** `sdd-explore` runs
- **WHEN** it gathers context
- **THEN** it indexes the codebase via Mnemonic `code_*`, researches constraints, and caches findings via Mnemonic `web_*`

#### Scenario: Propose and design reuse brainstorming
- **GIVEN** `sdd-propose` and `sdd-design` run
- **WHEN** they generate artifacts
- **THEN** they delegate to `brainstorming` (and `sdd-design` also to `architectural-decision-records` and `c4-diagrams`) and write into `openspec/changes/`

#### Scenario: Spec reuses gherkin and spec-as-source
- **GIVEN** `sdd-spec` runs
- **WHEN** it creates delta specs
- **THEN** it reuses `gherkin-authoring` and `spec-as-source`

#### Scenario: Tasks reuses write-tasks
- **GIVEN** `sdd-tasks` runs
- **WHEN** it decomposes specs
- **THEN** it reuses `write-tasks` and `openspec-sync-specs` and writes `work-unit/assignments.md`

#### Scenario: Apply reuses execution skills
- **GIVEN** `sdd-apply` runs
- **WHEN** it implements tasks
- **THEN** it reuses `subagent-driven-development`, `executing-tasks`, and `test-driven-development`

#### Scenario: Verify and archive reuse verification skills
- **GIVEN** `sdd-verify` and `sdd-archive` run
- **WHEN** they validate and finalize
- **THEN** they reuse `verification-before-completion` and `acceptance-test-authoring`, and `sdd-archive` mechanically copies the change with `diff -r` verification

### Requirement: Discarded gentleman-ai equivalents

The system SHALL replace gentleman-ai's Engram-based SDD skills with Mnemonic-native Skillgrid equivalents and SHALL NOT integrate Engram MCP.

#### Scenario: Engram skills discarded
- **GIVEN** the SDD integration is applied
- **WHEN** skill selection occurs
- **THEN** `engram-memory`, `engram-memory-protocol`, and `engram-sdd-flow` are replaced by `mnemonic-memory`, `mnemonic-memory-protocol`, and the new `sdd-*` phase skills

#### Scenario: Engram MCP not integrated
- **GIVEN** the SDD workflow is configured
- **WHEN** persistence backends are chosen
- **THEN** Engram MCP server integration is excluded and Mnemonic is used exclusively for persistent memory

#### Scenario: gentleman-ai phase skills replaced
- **GIVEN** a phase skill is invoked
- **WHEN** the Skillgrid-native version runs
- **THEN** gentleman-ai `sdd-init`, `sdd-propose`, `sdd-apply`, and `sdd-archive` are superseded by Mnemonic-native equivalents that write OpenSpec artifacts and `.skillgrid/sdd/` scratch
