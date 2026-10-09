# Skillgrid Release/2 — Engineering Operating System (Implementation Plan)

> **STATUS:** `draft` (2026-10-09) — Operationalizes the "Release/2 — Engineering Operating System" vision.
>
> **Master plan:** `.skillgrid/artifacts/skillgrid-workflow-update-plan.md`
> **Briefing:** `briefing.md` (spec vision and requirements)
> **Topic:** 2026-10-02-skillgrid-workflow-updates
> **Date:** 2026-10-09
> **Classification:** standard (T2)

---

## 1. Objective

Transform Skillgrid from a skill installer into a **deterministic, artifact-driven engineering operating system**. This plan provides the roadmap for the initial implementation order (33 items), grouped into 8 phases. 

The immediate goal is to establish the **Runtime Foundation** (Phase 1) and **Intent/Specification** (Phase 2) contracts, which will allow the system to manage the first "tracer-bullet" change from intent to delivery.

---

## 2. Roadmap & Phases

### Phase 1: Runtime Foundation (The Engine)
The goal of Phase 1 is to provide the "bones" of the OS: state management, policy enforcement, and artifact schemas.

1.  **Canonical Artifact Schemas**: Define YAML/JSON schemas for all core artifacts (Goal, Briefing, Blueprint, Tasks, Evidence).
2.  **Workflow State Model**: Implement a state machine in Go that validates transitions (e.g., `spec` → `blueprint` → `execution`).
3.  **Canonical Project State**: Extend `state.yaml` to be the machine-readable source of truth for the current phase.
4.  **Execution Modes & Authorization**: Define `local`, `CI`, and `remote` modes with distinct permission sets.
5.  **Workflow Complexity Budget**: Implement the "1% rule" logic to prevent process bloat.
6.  **Skill Registry**: Move skill metadata (version, preconditions, outputs) into a queryable registry.
7.  **Skill Runtime Contract**: Enforce that skills provide `metadata`, `activation conditions`, and `output contracts`.
8.  **Execution Packets**: Define the structure for "bounded" context handed to subagents.
9.  **Capability Router**: Implement the logic to map a goal to a specific playbook and set of skills.
10. **Policy Engine**: Implement deterministic checks for permissions, risk, and "fail-open" behaviors.
11. **Agent Capability Matrix**: Define which agents (OpenCode, Cursor, Claude) can perform which operations.
12. **Mnemonic Integration**: Wire the Go runtime to the Mnemonic MCP for memory operations.
13. **Evidence & Provenance Models**: Define how verification results and "receipts" are stored and linked to artifacts.

### Phase 2: Intent & Specification (The Brains)
The goal of Phase 2 is to ensure that before any code is written, the "what" and "why" are typed, validated, and linked.

14. **Discovery Capability**: Add a "discovery" mode to turn broad goals into actionable briefings.
15. **Goal & Done-When**: Formalize the "Goal" artifact with explicit success conditions.
16. **Behavior Contracts**: Implement "Behavior Contracts" (Cucumber-style) as a first-class artifact type.
17. **Effective Engineering Contract**: Combine requirements, constraints, and success criteria into a single "contract" for execution.
18. **Behavioral Traceability**: Link every requirement to a specific behavior contract and verification check.
19. **Artifact Transformation & Lineage**: Implement validation that ensures a `briefing` correctly "transforms" into a `blueprint`.
20. **Architecture & Design Contracts**: Define the `ARCHITECTURE.md` and `blueprint.md` schemas.
21. **ADR Lifecycle**: Implement ADR supersession and lineage tracking.
22. **Code Index & Knowledge**: Implement the "code-aware" discovery using the Mnemonic code index.

### Phase 3: Bounded Execution (The Hands)
The goal of Phase 3 is to execute work in a constrained, verifiable manner.

23. **Thin Agent Adapters**: Provide a unified interface for different agent harnesses.
24. **Dependency-Aware Work Graphs**: Manage task dependencies and "waves" of execution.
25. **Immutable Candidate Creation**: Ensure that execution produces "candidates" (commits/diffs) that are not modified during review.
26. **Reconciliation Engine**: Detect when the "reality" (git tree) diverges from the "plan" (artifacts).

### Phase 4: Review & Verification (The Judge)
The goal of Phase 4 is to prove that the work meets the "Effective Engineering Contract".

27. **Verification Engine**: Execute automated checks (tests, lints, behavioral contracts).
28. **Independent Review**: Implement a "reviewer" subagent that provides a fresh perspective on the candidate.
29. **Quality Floors & Escalation**: Define minimum quality bars and the path to "escalation" if they fail.
30. **Minimal-Change Correction**: Implement a "correction loop" that allows agents to fix findings without re-running the whole pipeline.

### Phase 5: Engineering Memory (The Long Term)
31. **Code-Linked Knowledge**: Link engineering decisions (ADRs, findings) to the specific code they affect.
32. **Memory Drift & Supersession**: Detect when "memory" (facts in Mnemonic) no longer matches the "reality" of the repo.
33. **Git & CI Integration**: Provide hooks that run the Skillgrid verification engine in CI.

---

## 3. Initial Slice: The "Tracer-Bullet"

To break the serial dependency and provide an immediate "useful whole," the first slice will implement a vertical slice of Phase 1 and Phase 2 for a single, simple task.

**Slice Components:**
1.  **Artifact Schemas**: Define the Go structures for `Briefing`, `Blueprint`, and `Validation` (the "Validation" matrix from Req 19).
2.  **State Validation**: A `skillgrid validate` command that checks if the artifacts in a spec folder follow the correct phase order.
3.  **Validation Matrix Script**: A JS script (`.mjs`) that checks if every requirement in the `briefing.md` has a corresponding row in the `## Validation` section of the `blueprint.md`.
4.  **Integration**: Wire the `writing-blueprints` skill to run this validation before it considers the blueprint "validated."

**Success Criteria for Slice:**
- The `skillgrid validate` command passes a valid spec folder and fails a folder with a missing validation row.
- The `writing-blueprints` skill is updated to call the validation script.
- A new ADR is recorded for the Go runtime architecture.

---

## 4. Constraints & Invariants

- **Determinism:** The Go runtime must perform all schema and state checks. LLMs must not "guess" if a transition is valid.
- **Fail-Open:** If the Skillgrid runtime is down, hooks should not block the agent; they should log a warning and allow the action.
- **Local-First:** All Phase 1-4 features must work without a network connection (excluding model inference).
- **No Parallel Workflow:** All changes must be reconciled with `.agents/skills/_shared/rules/sdd-structure.md`.

---

## 5. Next Steps

1.  **Create ADR-0032**: "Go Runtime Architecture for Release/2."
2.  **Define Schemas**: Write the initial JSON/YAML schemas for Phase 1 artifacts.
3.  **Implement `skillgrid validate`**: Start the Go CLI implementation for state and artifact validation.
4.  **Update Skills**: Modify `writing-blueprints` and `slicing` to use the new validation scripts.
