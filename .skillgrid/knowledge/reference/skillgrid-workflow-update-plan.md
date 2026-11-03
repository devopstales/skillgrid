---
type: Reference
title: "Skillgrid Release/2 — Engineering Operating System Plan"
---

# Skillgrid Release/2 — Engineering Operating System Plan

**Status:** Architecture and implementation planning
**Project:** `devopstales/skillgrid`
**Target branch:** `release/2`
**Canonical SDD authority:** `.agents/skills/_shared/rules/sdd-structure.md`
**Memory subsystem:** Mnemonic, written in Go
**User interface:** Enterprise JavaScript WebUI served by `skillgrid serve`

---

# 1. Vision and Objective

Skillgrid should evolve from a skill and rules installer into an **artifact-driven engineering operating system for humans and AI agents**.

Its responsibility is to manage the engineering lifecycle from initial intent to verified delivery. It must support product discovery, requirements, behavioral specifications, architecture, implementation planning, execution, review, verification, engineering memory, governance, recovery, and learning.

Skillgrid should not be merely:

* A collection of agent prompts.
* A coding-agent wrapper.
* A task tracker.
* A documentation generator.
* A memory search interface.
* An OpenSpec or other methodology wrapper.
* A WebUI that only displays agent conversations.

The core product is a deterministic orchestration and governance layer that connects engineering intent, durable artifacts, execution, evidence, and delivery.

## Guiding principle

> Skillgrid owns the engineering process. Agents perform the work. Mnemonic remembers the engineering. Git records source history. Canonical state records the current reality. Policy controls authority. Independent review challenges results. Verification proves defined claims. Humans retain control over consequential decisions.

# 2. Canonical Workflow Authority

The existing Skillgrid SDD structure remains authoritative.

All workflow changes must be reconciled with:

`.agents/skills/_shared/rules/sdd-structure.md`

The plan must not introduce a parallel workflow that contradicts the canonical SDD rules. New capabilities, playbooks, artifact types, and adapters must integrate with the existing structure.

The `.skillgrid/sdd` directory is an important part of the project's established workflow and must be preserved and extended according to the repository's actual conventions.

OpenSpec, BMAD, Superpowers, intent-driven templates, and other projects are research inputs. They do not become runtime dependencies or workflow authorities by default.

# 3. Architectural Invariants

1. **Canonical workflow first:** All capabilities conform to Skillgrid's SDD authority.
2. **Artifacts over conversation history:** Durable engineering state must not depend on an agent remembering previous messages.
3. **Typed interfaces:** Skills and agents exchange structured artifacts, contracts, and findings.
4. **Evidence before state:** A claimed outcome is not a verified outcome.
5. **Authority before automation:** Execution, approval, and delivery have distinct permissions.
6. **Immutable review targets:** Review and verification identify the exact candidate they evaluate.
7. **Independent evaluation:** The generator should not be the sole judge of its own work when risk warrants independence.
8. **Deterministic enforcement:** LLMs may propose and interpret; deterministic systems enforce schemas, permissions, transitions, and mechanical checks.
9. **Adaptive rigor:** Process cost must be proportional to uncertainty, risk, and change impact.
10. **Recoverability:** Work can be resumed, reconciled, rolled back, or sent back to an earlier stage.
11. **Replaceable agents:** No agent vendor or conversation format owns workflow state.
12. **Local-first operation:** The default experience should work locally, including on CPU-only systems.
13. **Traceability:** Requirements, decisions, code, verification, and delivery remain connected.
14. **No unnecessary ceremony:** Introduce process only when its expected engineering value exceeds its cost.

# 4. High-Level Architecture

```text
Human Goal
    |
    v
Intent and Problem Framing
    |
    v
Discovery and Decision Readiness
    |
    v
Adaptive Process Sizing
    |
    v
Capability Router
    |
    v
Workflow / Playbook Selection
    |
    v
Skill Composition
    |
    v
Typed Artifact Pipeline
    |
    v
Execution Packet
    |
    v
Policy and Authorization
    |
    v
Agent Adapter
    |
    v
Immutable Candidate
    |
    +----------------------+
    |                      |
    v                      v
Independent Review     Verification
    |                      |
    +-----------+----------+
                |
                v
        Findings and Evidence
                |
                v
      Bounded Correction Loop
                |
                v
        Canonical State Update
                |
                v
      Mnemonic Knowledge Update
                |
                v
      Approval and Delivery
                |
                v
    Post-Delivery Verification
                |
                v
      Reconciliation and Learning
```

Supporting components:

* **Skillgrid Runtime:** Workflow orchestration and state transitions.
* **Capability Router:** Selects capabilities, playbooks, skills, and suitable agents.
* **Policy Engine:** Enforces permissions, risk constraints, and authorization.
* **Artifact Engine:** Validates, versions, links, and transitions engineering artifacts.
* **Execution Engine:** Creates bounded execution packets and coordinates agents.
* **Verification Engine:** Executes checks against identified candidates.
* **Review Engine:** Coordinates independent, evidence-based review.
* **Reconciliation Engine:** Detects inconsistencies between source, artifacts, workflow state, and memory.
* **Mnemonic:** Provides durable engineering memory and semantic retrieval.
* **Git Integration:** Records source history and supports provenance.
* **CLI:** Exposes the core runtime.
* **WebUI:** Provides enterprise visibility and controlled interaction.

# 5. Product Intent to Verified Delivery

Introduce a standard lifecycle playbook connecting product-level intent to delivery.

```text
Goal
  -> Discovery
  -> Proposal
  -> Requirements
  -> Behavior Contract
  -> Architecture and Design
  -> Architecture Decisions
  -> Implementation Plan
  -> Execution Packets
  -> Candidate
  -> Review
  -> Verification
  -> Evidence
  -> Approval
  -> Delivery
  -> Post-Delivery Verification
  -> Learning
```

This is a reusable workflow, not a mandatory linear pipeline. The runtime must support non-linear entry, parallel work, branching, backtracking, supersession, replanning, and recovery.

Each stage must have a defined purpose, artifact contract, completion condition, and permitted transitions.

# 6. Discovery Capability

Introduce Discovery as a first-class capability for turning broad or ambiguous product intent into actionable engineering work.

Discovery should support:

* Personas and user needs.
* Problem statements and desired outcomes.
* User journeys and system boundaries.
* Constraints, assumptions, and unknowns.
* Risk and dependency identification.
* Prioritization using explicit criteria.
* Release planning and thin vertical slices.
* Measurable success conditions.
* Questions requiring human decisions.
* Incremental adoption in existing repositories.

For large initiatives, Discovery may produce:

`Product Goal → Discovery Brief → Release Plan → Vertical Slices → Proposals`

For small, well-understood changes, Discovery can be minimal or bypassed under the applicable workflow rules.

# 7. Problem Framing and Decision Readiness

Before execution, Skillgrid should assess whether the request is sufficiently understood.

Evaluate:

* Goal clarity.
* Observable completion conditions.
* Scope boundaries.
* Important unknowns.
* Reversibility of decisions.
* Impact radius.
* Security and operational risk.
* Existing architecture and constraints.
* Availability of verification methods.

The system should ask targeted questions when an unresolved decision materially affects the outcome. It should not ask questions simply to complete a checklist.

Possible outcomes:

* Ready for execution.
* Ready with recorded assumptions.
* Needs clarification.
* Needs discovery or design.
* Requires explicit human decision.
* Unsafe or unauthorized.

# 8. Adaptive Process Sizing and Workflow Complexity Budget

The workflow must match the task.

A typo fix should not require the same artifacts and review stages as a database migration or security-sensitive architectural change.

Process sizing should consider:

* Change size and semantic impact.
* Uncertainty and ambiguity.
* Reversibility.
* Number of affected components.
* External dependencies.
* Security, data, and production risk.
* Verification cost.
* Required approval authority.

A workflow complexity budget should prevent unnecessary skills, agents, reviews, and artifacts from accumulating.

Every additional stage must justify its expected value. The system should distinguish mandatory policy requirements from optional engineering practices.

# 9. Capability Router, Playbooks, and Skills

Keep three concepts distinct.

**Capabilities** describe what the system can do: discovery, architecture, implementation, testing, security review, migration, performance analysis, and so on.

**Playbooks** describe how capabilities are combined to accomplish a class of goal.

**Skills** are executable procedures with explicit contracts.

Example:

```text
Goal: Introduce an API endpoint
  -> Capability Router
  -> API Development Playbook
  -> Architecture / Implementation / Testing Skills
  -> Execution Packets
  -> Review and Verification
```

The router should consider artifact type, risk, project context, available tools, agent capabilities, policy, and verification needs.

## Skill Runtime Contract

Each skill should define:

* Metadata and version.
* Activation conditions.
* When to use and when not to use.
* Preconditions and required inputs.
* Hard rules and decision gates.
* Execution procedure.
* Output artifact contract.
* Validation and evidence requirements.
* Common rationalizations and red flags.
* Backtracking conditions.
* Dependencies and compatibility.
* Examples and behavioral evaluations.

Keep primary skill files concise. Put extensive references, examples, and supporting assets in separate files.

## Skill TDD and evaluation

Treat skills as software artifacts. Test them with representative scenarios, expected outputs, adversarial cases, and regression fixtures.

A skill is not considered reliable merely because its instructions look clear.

# 10. Typed Artifact Transformation Pipeline

Engineering artifacts are the durable interfaces between workflow stages.

| Artifact                     | Responsibility                                          |
| ---------------------------- | ------------------------------------------------------- |
| Goal / Intent                | Desired outcome and success conditions                  |
| Discovery Brief              | Users, problems, journeys, constraints, priorities      |
| Proposal                     | Motivation, scope, and intended change                  |
| Requirement                  | Normative system obligation                             |
| Behavior Contract            | Observable and testable behavior                        |
| Design Artifact              | Structure, interfaces, boundaries, and trade-offs       |
| Architecture Decision Record | Decision, alternatives, rationale, consequences         |
| Implementation Plan          | Tasks, dependencies, sequence, acceptance conditions    |
| Execution Packet             | Bounded instructions, context, permissions, and outputs |
| Candidate                    | Immutable snapshot submitted for evaluation             |
| Review Findings              | Evidence-based issues and recommended actions           |
| Verification Receipt         | Target identity, checks, results, and evidence          |
| Delivery Record              | Authorization, delivered version, and outcome           |

Every artifact type must define its identity, version, schema, lifecycle state, producer, consumers, prerequisites, validation rules, lineage, supersession policy, and evidence obligations.

Artifacts must be machine-readable where practical while remaining understandable and editable by humans.

# 11. Effective Engineering Contract

The Effective Engineering Contract is the resolved definition of what the system is currently expected to do.

```text
Canonical Specifications
  + Active Change Deltas
  + Accepted Decisions
  + Current Constraints
  + Applicable Policies
  - Superseded Requirements
  - Archived Changes
  =
Effective Engineering Contract
```

The contract must be reproducible and versioned. Every execution packet, review, and verification run should identify the contract version used.

Skillgrid must distinguish:

* Current normative requirements.
* Proposed but unaccepted changes.
* Accepted decisions.
* Superseded requirements.
* Historical and archived material.
* Informative documentation that is not normative.

Superseded behavior is not merely a skipped test. It is excluded from the current effective contract while remaining available for historical analysis.

# 12. Normative Contracts and Implementation Bindings

Separate the definition of required behavior from runtime-specific implementations.

```text
Normative Contract
  ├── Go Runtime Binding
  ├── JavaScript WebUI Binding
  ├── CLI Binding
  ├── Verification Runner Binding
  └── Agent Adapter Binding
```

Apply this pattern to workflow transitions, artifact schemas, memory operations, policy evaluation, verification, agent execution, and interfaces.

The normative contract defines what must happen. Bindings implement that contract in specific technologies.

Use conformance tests to detect divergence. Different implementations of the same contract should agree on required semantics and observable outcomes.

# 13. Behavior Contracts and Spec-as-Source

Behavior specifications must be first-class engineering artifacts.

For workflows that use specification-first development:

```text
Behavior Change
  -> Update Behavior Contract
  -> Define Acceptance Scenarios
  -> Verify Expected Failure
  -> Implement
  -> Run Acceptance and Regression Checks
  -> Review Evidence
```

The acceptance contract defines success before implementation. The implementation agent cannot redefine success after observing its own result.

Support Markdown-first behavioral specifications, structured requirements, and optional Gherkin scenarios.

Behavior-driven development is one capability, not the entire Skillgrid methodology. Other work may use TDD, API contracts, property-based testing, browser tests, security tests, performance tests, or custom verification.

## Scenario lifecycle

Scenarios need stable identities and explicit states, including draft, active, accepted, superseded, and archived.

The effective acceptance suite is derived from current normative specifications and applicable active changes.

## Behavioral traceability

Maintain links across:

```text
Goal
  -> Requirement
  -> Behavior Scenario
  -> Design Element
  -> Decision
  -> Task
  -> Code Change
  -> Verification
  -> Evidence
  -> Delivery
```

Support both forward traceability and reverse impact analysis.

# 14. Architecture, HLD, LLD, and ADRs

Architecture must be an explicit lifecycle capability.

Support:

* System context and external actors.
* C4-style context, container, component, and deployment views where appropriate.
* High-Level Design (HLD).
* Low-Level Design (LLD).
* Component and interface contracts.
* Data models and ownership.
* Dependencies and integration boundaries.
* Security and trust boundaries.
* Operational and deployment architecture.
* Architecture risks and trade-offs.
* Architecture Decision Records (ADRs).

ADRs should record the decision, status, context, constraints, alternatives, rationale, consequences, affected requirements, and supersession relationships.

Architecture artifacts should be traceable to goals and behavior contracts. Architecture diagrams are useful views of structured design information, not substitutes for it.

Use Domain-Driven Design when domain complexity justifies it. Support ubiquitous language, bounded contexts, context maps, and domain models without forcing every project into DDD ceremony.

# 15. Glossary and Semantic Consistency

The project glossary is a first-class, versioned artifact connected to Mnemonic.

A glossary entry should support:

* Term and definition.
* Domain or bounded context.
* Approved aliases.
* Forbidden or misleading aliases.
* Related terms.
* Source and provenance.
* Confidence and status.
* References and supersession history.

Use semantic checks to detect inconsistent terminology across requirements, behavior contracts, designs, ADRs, tasks, and documentation.

Mnemonic provides retrieval and relationships; accepted canonical artifacts remain authoritative for normative definitions. Conflicting definitions must be surfaced rather than silently merged.

# 16. Adversarial Authoring and Independent Review

For high-risk or ambiguous artifacts, support a distinct author/reviewer workflow.

1. An author creates a proposal, behavior contract, design, or plan.
2. An independent reviewer examines the artifact against its contract.
3. The reviewer identifies ambiguity, contradictions, missing cases, unsupported assumptions, and failure modes.
4. Each finding includes evidence, impact, severity, and a recommended action.
5. The author resolves the finding or records a justified disposition.
6. The artifact is revalidated before dependent work proceeds.

Keep generation and evaluation logically independent when risk warrants it. Different models can help, but changing model identity alone does not guarantee independence.

Bound review cycles. Avoid endless debate and unproductive findings.

# 17. Execution Plans, Work Graph, and Execution Packets

Represent work as a dependency-aware graph rather than only a flat task list.

Support the hierarchy:

`Milestone → Slice → Task → Step`

Use context-bounded tasks with explicit:

* Objective and done conditions.
* Inputs and prerequisites.
* Must-haves and prohibited changes.
* Relevant files, components, and interfaces.
* Expected outputs.
* Verification requirements.
* Permission boundary.
* Completion and escalation conditions.

## Execution Packet

The Execution Packet is the boundary between Skillgrid orchestration and an agent.

It should contain:

* Task identity and objective.
* Effective Engineering Contract reference.
* Applicable SDD instructions.
* Relevant project and code context.
* Required artifacts and decisions.
* Scope and blast-radius constraints.
* Allowed tools and actions.
* Expected output format.
* Verification requirements.
* Completion criteria.
* Resume and escalation information.

Agents should receive enough context to work independently without being flooded with unrelated project history.

# 18. Specialist Routing and Interleaved Execution

Route work by artifact type, required capability, risk, context, permissions, and verification needs.

Potential specialists include product discovery, requirements, architecture, ADR review, implementation, acceptance-test authoring, security, performance, migration, operations, and independent verification.

Support small, interleaved execution units:

```text
Implementation Unit
  -> Focused Verification
  -> Findings and Evidence
  -> Targeted Correction
  -> Reverification
  -> Next Unit
```

Specialists exchange typed artifacts and findings, not unstructured assumptions. Each handoff preserves the relevant contract, context, constraints, and evidence.

Parallel work must use explicit dependency and ownership boundaries. Shared mutable state, conflicting file edits, and duplicate authority must be controlled.

# 19. Immutable Candidates and Review Pipeline

Review must identify the exact artifact or source snapshot being evaluated.

```text
Working Tree
  -> Candidate Snapshot
  -> Spec Compliance Review
  -> Code / Design Quality Review
  -> Risk-Specific Review
  -> Verification
  -> Evidence and Receipt
```

If the candidate changes after review or verification, invalidate affected results and rerun the necessary checks.

Review pipelines may include:

* Specification compliance.
* Architecture and interface integrity.
* Security and privacy.
* Reliability and operations.
* Performance.
* Maintainability.
* Regression and compatibility.
* Domain-specific correctness.

Findings should be classified, evidenced, prioritized, and linked to affected artifacts.

Review, verification, approval, and delivery are distinct operations.

# 20. Verification Engine and Quality Floors

Verification should be an explicit subsystem rather than a prompt telling an agent to test its work.

Support multiple modalities:

* Deterministic schema and policy validation.
* Unit and integration tests.
* Behavioral acceptance tests.
* API and contract tests.
* Static analysis and linting.
* Security and dependency checks.
* Browser and end-to-end tests.
* Performance and load tests.
* Migration and compatibility tests.
* Operational readiness checks.
* Human verification when automation is insufficient.

## Verification Receipt

Record at least:

* Candidate identity or digest.
* Effective contract version.
* Verification plan and runner.
* Environment and relevant configuration.
* Checks executed and outcomes.
* Logs and evidence references.
* Known limitations.
* Timestamp and provenance.
* Final status.

## Quality floors

Critical failures cannot be averaged away by a high overall score.

Use evidence-based statuses such as PASS, WARN, FAIL, and BACKTRACK. Numeric scores may help diagnose quality but should not independently override hard requirements.

# 21. Bounded Correction and Escalation

Failed review or verification should produce an explicit Correction Work Order.

It includes:

* Finding and supporting evidence.
* Affected requirement and artifact.
* Required correction.
* Scope boundary.
* Prohibited unrelated changes.
* Required regression checks.
* Retry limit and escalation rule.

Prefer the smallest change that resolves the issue without introducing unrelated modifications.

After correction, verify the new candidate. Never transfer a previous candidate's passing status to modified source without checking whether the evidence remains valid.

# 22. Mnemonic as Engineering Memory

Mnemonic is Skillgrid's durable engineering memory layer, implemented in Go.

It should preserve and retrieve:

* Architecture and technical decisions.
* Invariants and contracts.
* Domain knowledge and terminology.
* Why important design choices were made.
* Code-to-knowledge relationships.
* Historical changes and supersession.
* Debugging insights.
* Environment-specific knowledge.
* Verification and operational lessons.
* Relevant relationships between artifacts.

Mnemonic is the memory authority, not the workflow authority. It must not silently override canonical artifacts or workflow state.

## Memory operations

Provide well-defined operations for saving, searching, retrieving, relating, summarizing, validating, superseding, and managing memory scope.

Support session, task, project, organization, and environment scopes with explicit access controls.

## Retrieval discipline

Use the sequence:

`Search → Retrieve → Read → Apply → Validate → Update`

Agents should search existing engineering knowledge before making consequential changes.

## Built-in embeddings

Provide a useful default embedding capability out of the box:

* CPU-based operation by default.
* Optional GPU acceleration.
* Optional local Ollama model.
* Optional remote embedding provider.
* Configurable model and provider.
* Reproducible embedding metadata.
* Version-aware reindexing.
* Clear behavior when embeddings are unavailable.

The basic installation should not require a separately managed model service just to obtain useful default functionality.

# 23. Code Indexing and Code-Linked Knowledge

Build a code index that connects source structure to engineering knowledge.

Potential capabilities:

* Repository and file indexing.
* Symbol and reference indexing.
* Language-aware parsing where available.
* Semantic and lexical search.
* Code-to-requirement and code-to-decision links.
* Change impact analysis.
* Incremental updates based on Git changes.
* Index freshness and integrity checks.

Use a hybrid retrieval approach rather than depending on embeddings alone.

Code indexing and Mnemonic memory should be related but distinct: the index describes and locates source; memory preserves knowledge, rationale, and relationships.

## Code-linked knowledge

Support explicit knowledge types such as:

* Technical Decision.
* Invariant Contract.
* Business Workflow.

Knowledge records should identify their scope, source, supporting evidence, validity, and affected code or symbols.

For protected code areas, consider requiring relevant knowledge to be attached before high-risk edits. This must be risk-based and should not block routine low-risk work unnecessarily.

## Deterministic knowledge attestation

The LLM may propose knowledge updates. Deterministic checks verify attachment, references, source state, and required metadata.

Support Git-based validation in CLI, hooks, and CI, with consistent rules across execution environments.

## Memory drift and supersession

Detect knowledge that may have become stale due to code, architecture, or requirement changes.

Support the lifecycle:

`Change → Affected Knowledge → Review → Update or Supersede → Attest`

Do not automatically convert every observation into a permanent rule. Learned rules require provenance, validation, and appropriate approval.

# 24. Canonical Project State and Canonical Engineering State

Keep operational state separate from durable engineering knowledge.

**Canonical Project State** records what is happening now:

* Active workflow and stage.
* Current work items and dependencies.
* Artifact versions and status.
* Candidate and verification status.
* Outstanding findings.
* Approvals and blockers.
* Resume points and active execution sessions.

**Canonical Engineering State** records the current engineering model:

* Components and interfaces.
* Domain terms.
* Architecture boundaries.
* Invariants and decisions.
* Dependencies and constraints.
* Risks and verification obligations.

Neither should depend on reconstructing state from a long conversation.

# 25. State Keeper, Evaluator, and Generator Separation

Where risk warrants it, keep these responsibilities distinct:

* **Generator:** Produces an artifact or candidate.
* **State Keeper:** Records validated canonical state.
* **Evaluator:** Judges conformance and quality.
* **Verification Engine:** Produces reproducible evidence.
* **Correction Agent:** Applies bounded corrections.
* **Policy Engine:** Determines what actions are permitted.
* **Delivery Controller:** Enforces delivery authorization.

An agent must not be able to declare its own output accepted merely by writing a success message or editing workflow metadata.

# 26. Workflow Schema and Methodology Capabilities

Represent workflows as versioned, validated definitions managed by Skillgrid.

A workflow definition should declare:

* Artifact types and schemas.
* States and allowed transitions.
* Prerequisites and completion conditions.
* Capability and skill requirements.
* Permissions and approval requirements.
* Verification and evidence obligations.
* Failure, retry, and escalation policies.
* Backtracking and supersession rules.
* Compatibility and version requirements.

The intent-to-behavior-to-design-to-decision-to-task lifecycle is a reusable playbook, not a hard-coded universal pipeline.

Support optional methodology capabilities such as DDD, BDD, TDD, security review, migration planning, and architecture analysis without forcing every project to use all of them.

# 27. Brownfield Adoption

Skillgrid must work with existing repositories, not only greenfield projects.

Provide a progressive adoption path:

```text
Repository Discovery
  -> Existing Architecture
  -> Existing Behavior
  -> Existing Decisions
  -> Existing Constraints
  -> Initial Canonical State
  -> Incremental Artifact Linking
  -> Verification and Reconciliation
```

Distinguish observed facts from inferred architecture and unverified assumptions.

Do not require teams to rewrite all documentation or code before receiving value. Allow incremental adoption with explicit coverage and confidence indicators.

# 28. Workflow State, Resume, and Recovery

Persist enough information to resume work after a context reset, agent failure, model switch, or interrupted session.

Support:

* Resume points and context handoffs.
* Task, slice, and milestone summaries.
* Persistent debugging sessions.
* Agent replacement.
* Workflow reconciliation.
* Dependency-aware replanning.
* Backtracking to earlier artifacts.
* Candidate invalidation.
* Rollback and recovery.
* Explicit deferred work.
* Capture and triage of newly discovered work.

Reconciliation must compare canonical state, actual artifacts, Git state, execution results, and memory. It should detect discrepancies rather than silently choose one source.

# 29. Policy and Authorization

Separate consent, policy, approval, and execution permissions.

Support explicit authorization modes, such as:

`OBSERVE → READ → ANALYZE → PROPOSE → EXECUTE`

The exact permissions depend on the configured policy. Moving from analysis to execution must not happen implicitly when a consequential action requires approval.

Policy should cover:

* File and repository boundaries.
* Tool permissions.
* Network access.
* Secrets and sensitive data.
* Production and infrastructure operations.
* External side effects.
* Approval requirements.
* Agent and model restrictions.
* Artifact modification authority.
* Delivery and rollback authority.

Enforce policy in the runtime, not only in agent instructions.

# 30. Agent and Model Abstraction

Provide thin adapters for supported agents and model providers.

The core runtime should not depend on one agent's private configuration format or conversational state.

Maintain an Agent Capability Matrix describing supported operations, context limits, tool capabilities, execution modes, permissions, and known constraints.

Model routing should consider task complexity, required capabilities, risk, cost, latency, privacy, and verification needs.

Record model and agent provenance when it matters for auditability or reproducibility. Agent replacement must not destroy workflow continuity.

# 31. CLI Architecture

The CLI is the primary operational interface and should expose the same core runtime used by the WebUI.

Planned command families include:

* `skillgrid serve`
* `skillgrid doctor`
* `skillgrid explain`
* `skillgrid validate`
* `skillgrid workflow`
* `skillgrid memory`
* `skillgrid verify`
* `skillgrid rollback`

Exact command syntax should follow the existing CLI conventions.

Provide dry-run, clear diagnostics, machine-readable output where useful, and consistent exit codes. Keep policy, validation, and workflow logic in reusable packages rather than duplicating it in command handlers.

# 32. Enterprise JavaScript WebUI

Serve the enterprise WebUI through `skillgrid serve`.

The WebUI must be a first-class interface to the same runtime, canonical state, policy, artifact, and memory services as the CLI.

## Core workspaces

* Project Dashboard.
* Workflow and State Explorer.
* Artifact Explorer.
* Discovery and Requirements.
* Behavior Contract Explorer.
* Architecture and Design Workspace.
* ADR and Decision Ledger.
* Execution and Agent Workspace.
* Review Workspace.
* Verification Workspace.
* Mnemonic Memory Explorer.
* Code Index and Knowledge Graph.
* Policy and Authorization Workspace.
* Skill and Capability Registry.
* Audit and Evidence Explorer.
* Recovery and Reconciliation Workspace.

## Enterprise requirements

* Authentication and SSO integration.
* RBAC and project isolation.
* Auditable state-changing operations.
* Accessible and responsive UI.
* Clear status and error reporting.
* Controlled execution and approval.
* Real-time or refreshable workflow state.
* Artifact lineage and provenance.
* Secure session and API handling.
* Local and air-gapped operation where configured.

The WebUI must not create a second source of workflow truth. All state transitions go through the runtime's authorized operations.

# 33. Audit, Observability, and Provenance

Record significant state changes and actions, including:

* Actor, agent, or service identity.
* Operation and target artifact.
* Previous and resulting state.
* Policy and authorization result.
* Relevant version or candidate digest.
* Verification and review evidence.
* Approval and delivery decisions.
* Failure, retry, and recovery events.

Keep audit records distinct from mutable summaries and conversational memory.

Provide structured logs, workflow events, latency and cost information, error reporting, and diagnostic commands without exposing secrets.

# 34. Security, Privacy, and Supply-Chain Governance

Treat skills, adapters, models, tools, and workflow definitions as governed components.

Track identity, version, provenance, permissions, dependencies, compatibility, and validation status.

Support:

* Project isolation.
* Secret redaction and secret boundaries.
* Least-privilege execution.
* Dependency and skill validation.
* Policy versioning.
* Model and agent provenance.
* Controlled external integrations.
* Offline and air-gapped operation.
* Backup, restore, and rollback.
* Audit of privileged operations.

# 35. Testing and Conformance Strategy

Test Skillgrid at several levels.

**Unit tests:** Artifact schemas, policy rules, state transitions, and individual capabilities.

**Contract tests:** Runtime APIs, agent adapters, memory operations, and verification interfaces.

**Workflow tests:** Complete and partial lifecycle scenarios, backtracking, recovery, and supersession.

**Behavioral evaluations:** Whether skills and workflows produce expected outcomes in representative and adversarial cases.

**Integration tests:** Mnemonic, Git, CLI, WebUI, model providers, and supported agent integrations.

**End-to-end tests:** User goal through execution, review, verification, approval, and delivery.

**Regression tests:** Prevent known failures, policy bypasses, invalid state transitions, and workflow rationalizations.

**Cross-binding conformance:** Ensure different implementations satisfy the same normative contract.

Golden fixtures should cover success, failure, interruption, conflicting changes, invalid artifacts, missing evidence, and unauthorized operations.

# 36. Engineering Insights and Continuous Improvement

Support a governed learning loop.

Capture useful engineering insights, debugging findings, workflow friction, repeated review failures, and verification gaps.

Before turning an insight into a permanent skill or rule:

1. Preserve its source and context.
2. Determine whether it is project-specific or general.
3. Check for duplication and conflicts.
4. Validate it against examples.
5. Assess the scope of its consequences.
6. Require approval when appropriate.
7. Track its effectiveness and supersession.

Do not allow automatic rule creation to produce an uncontrolled accumulation of redundant instructions.

# 37. What Skillgrid Should Not Become

Skillgrid should not become:

* An OpenSpec wrapper.
* A hard-coded linear methodology.
* A coding-only lifecycle.
* A single-agent orchestration framework.
* A prompt collection without evaluation.
* A memory database that silently overrides project truth.
* A UI-only workflow tracker.
* A process engine that forces every task through every stage.
* An LLM-only enforcement system.
* A system that treats generated documentation as verified truth.
* A system that equates review approval with delivery authorization.

# 38. Implementation Roadmap

## Phase 1 — Runtime Foundation

* Confirm canonical SDD integration.
* Define core artifact schemas and identities.
* Implement workflow state and transition validation.
* Implement Canonical Project State.
* Define execution modes and authorization boundaries.
* Add workflow complexity budgeting.
* Implement Skill Registry and Skill Runtime Contract.
* Define Execution Packets.
* Implement Capability Router and initial playbooks.
* Define the Policy Engine.
* Establish the Agent Capability Matrix.
* Integrate Mnemonic.
* Define evidence and provenance models.

## Phase 2 — Intent and Specification

* Add Discovery capability.
* Formalize Goal and Done-When.
* Add typed requirements and Behavior Contracts.
* Implement Effective Engineering Contract.
* Add scenario lifecycle and supersession.
* Implement behavioral traceability.
* Define architecture and design artifact contracts.
* Add ADR lifecycle and lineage.
* Add glossary and semantic consistency checks.
* Implement artifact transformation validation.
* Add brownfield discovery and incremental adoption.

## Phase 3 — Bounded Execution

* Implement execution packet creation and validation.
* Add thin agent adapters.
* Implement dependency-aware work graphs.
* Add context handoffs and resume points.
* Support interleaved specialist execution.
* Implement immutable candidate creation.
* Add bounded correction work orders.
* Implement deterministic workflow reconciliation.

## Phase 4 — Review and Verification

* Implement the Verification Engine.
* Add immutable verification targets and receipts.
* Add independent review pipelines.
* Add quality floors and escalation.
* Implement minimal-change correction.
* Add behavioral verification adapters.
* Implement contract conformance tests.
* Add evidence-based state transitions.

## Phase 5 — Engineering Memory and Code Index

* Establish Mnemonic memory contracts.
* Add default CPU embedding capability.
* Add optional GPU, Ollama, and remote providers.
* Implement repository and code indexing.
* Add symbol-aware indexing where practical.
* Implement code-linked engineering knowledge.
* Add deterministic knowledge attestation.
* Add memory drift and supersession.
* Implement change-to-knowledge impact analysis.
* Add engineering traceability graph.
* Add Git, pre-commit, and CI validation.

## Phase 6 — CLI, Runtime Operations, and Recovery

* Complete CLI command families.
* Implement `skillgrid doctor`.
* Add `skillgrid explain` and `skillgrid validate`.
* Add workflow and memory inspection commands.
* Add verification and rollback commands.
* Implement dry-run and recovery mechanisms.
* Add operational diagnostics and audit events.
* Verify local-first and offline operation.

## Phase 7 — Enterprise WebUI

* Implement `skillgrid serve`.
* Define and expose the runtime API.
* Build the Project Dashboard and Workflow Explorer.
* Build the Artifact Explorer and lineage views.
* Add Discovery, Requirements, and Behavior Contract workspaces.
* Add Architecture and ADR workspaces.
* Add Agent and Execution workspaces.
* Add Review and Verification workspaces.
* Add Mnemonic and Knowledge Graph views.
* Add Policy, Approval, and Audit workspaces.
* Add Recovery and Reconciliation views.

## Phase 8 — Enterprise Governance

* Add SSO and RBAC integration.
* Implement organization and project isolation.
* Add policy administration.
* Add skill and adapter supply-chain governance.
* Add enterprise audit and provenance.
* Add organization-level knowledge controls.
* Validate security and offline deployment requirements.

# 39. Initial Implementation Order

The initial implementation should establish the runtime and its contracts before expanding the UI.

1. Canonical artifact schemas.
2. Workflow state model and transition validation.
3. Canonical Project State.
4. Execution modes and authorization.
5. Workflow complexity budget.
6. Skill Registry.
7. Skill Runtime Contract.
8. Execution Packet.
9. Capability Router.
10. Policy Engine.
11. Agent Capability Matrix.
12. Mnemonic integration.
13. Evidence model.
14. Goal and Done-When.
15. Behavior Contract.
16. Effective Engineering Contract.
17. Behavioral traceability.
18. Artifact transformation and lineage.
19. Immutable Candidate.
20. Verification Engine and receipts.
21. Independent Review.
22. Quality Floors.
23. Minimal-Change Correction.
24. Reconciliation Engine.
25. Discovery and brownfield adoption.
26. Architecture, design, ADR, and glossary contracts.
27. Code Index.
28. Code-Linked Knowledge.
29. CLI diagnostics and recovery commands.
30. `skillgrid serve`.
31. Enterprise runtime API.
32. JavaScript WebUI.
33. Enterprise governance and operational hardening.

This order is a starting dependency sequence, not a prohibition against parallel development. Implementation should be adjusted to actual repository structure and existing code.

# 40. Success Criteria

Skillgrid is successful when it can:

* Convert broad goals into appropriately sized engineering work.
* Preserve one authoritative workflow and artifact structure.
* Connect product intent, requirements, behavior, design, decisions, tasks, code, and evidence.
* Select suitable skills and agents without hard-coding one provider.
* Give agents bounded, context-rich execution packets.
* Maintain durable state independently of conversations.
* Produce reproducible, candidate-specific verification evidence.
* Separate generation, evaluation, verification, authorization, and delivery.
* Detect stale knowledge and reconcile inconsistent state.
* Resume work after interruption or agent replacement.
* Support both greenfield and brownfield repositories.
* Operate locally with a useful default memory experience.
* Provide an enterprise WebUI without creating a second workflow authority.
* Remain useful for small changes without imposing unnecessary process.
* Support complex, high-risk engineering work with appropriate rigor.

# 41. Final Definition

**Skillgrid is an artifact-driven engineering operating system that coordinates humans and interchangeable AI agents through a governed, adaptive, recoverable lifecycle—from product intent and discovery through specifications, architecture, implementation, independent review, deterministic verification, engineering memory, approval, and delivery.**

Its differentiator is not code generation. It is the ability to define what success means, choose an appropriate process, preserve why decisions were made, constrain execution, challenge results independently, prove outcomes with evidence, and maintain trustworthy engineering state over time.

