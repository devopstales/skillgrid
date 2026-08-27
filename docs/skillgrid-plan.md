## Workflow

```
idea -> proposal -> specs -> (design -> adr) -> tasks

AGENTS.md -> proposal/ -> specs/ -> (design, adr) -> "Epic" / "Stories"
```

### Phase 1: Idea

Before writing any code or generating detailed specs, classify the incoming request or idea into one of three paths. This "right-sizing" ensures you don't over-process simple fixes or under-plan complex architectures.

* Primary Agents: Analyst or PM
* Required Skills:
    * `project-context` (BMAD) -> AGENTS.md
    * `brainstorming` (Superpowers) -> spec
    * `grilling`
* Artifacts:
    * "Project Cotext": AGENTS.md or opensec config
    * "Specs":

### Phase 2: Proposal (The "Why")

Before generating implementation details, you must explicitly capture the intent. This prevents AI from solving the wrong problem.

* Primary Agents: Analyst or PM
* Required Workflows:
    * `forge-idea` (BMAD): Helps refine the raw idea into a structured proposal.md document stored in `openspec/changes/<id>/`.
* Artifacts:
    * "Proposal": Document why the change matters.

### Phase 3: Specification (The "What")

This is the core of the workflow. You must define observable behavior and ensure that specification context travels with the implementation plan.

* Primary Agents: PM
* Required Skills:
    * `gherkin-authoring` (intent-driven-template): Authoring the Gherkin scenarios themselves.
    * `acceptance-test-authoring` (intent-driven-template): Turning specs into acceptance tests.
* Artifacts:
    * "Specs": Define the expected behavior using testable, Gherkin-style scenarios (Given/When/Then).
* Test Mapping: Every Gherkin scenario MUST map 1:1 to a runnable acceptance test (via `acceptance-test-authoring`). The Phase 7 verification gate executes these tests — a scenario with no runnable test is not a verified scenario.

### Phase 4: Design & Architectural Decisions (The "How")

Translate the spec into a technical reality while preserving decisions for future AI context windows.

* Primary Agents: Architect
* Required Skills:
    * `build` (BMAD, Design Phase): Winston analyzes the specs and generates data flow diagrams, API contracts, and trade-off analyses.
    * `writing-designs` (Superpowers) -> design
* Artifacts:
    * "Design Document": Explain the implementation approach, data flows, and technical trade-offs.
    * "Architectural Decision Records" (ADRs): Explicitly record durable decisions (e.g., "We chose vector DB X over Y because..."). This prevents the AI from re-explaining or re-deriving these choices in every chat session.

### Phase 5: Task Planning

Break the design down into actionable work, but handle conflicts with a "never-stall" doctrine.

* Primary Agents: Architect
* Required Skills:
    * `writing-designs` (Superpowers, renamed writing-plans): Translates the design into a strict, step-by-step execution plan. It enforces DRY, YAGNI, and ensures no "placeholder" code is left in the instructions.
* Artifacts:
    * Task Generation: Turn the accepted intent, behavior, and design into discrete, actionable tasks.
    * "Epic" / "Stories"

### Phase 6: Dispatch & Execution (Subagent-Driven Development)

Execute the tasks using specialized AI coding agents, optimizing for efficiency and token usage.

* Primary Skill (The Controller): 
    * `subagent-driven-development` (SDD) (Superpowers). This is the master orchestrator. It reads the Traveling Spec, sets up the .skillgrid/sdd/ workspace, and dispatches subagents.
    * `executing-designs` (Superpowers, renamed executing-plans) to implement this design task-by-task.
* Supporting Skills:
    * `using-git-worktrees` (Superpowers): Ensures the AI builds in an isolated git worktree so it doesn't corrupt your main branch.

* Context Loading: Pass the "traveling spec," ADRs, and design documents to the implementation agents.
* Batching Micro-Tasks: Do not dispatch a new agent for every 1-line change. Batch small, same-shape tasks (e.g., updating a constant across 10 files) into a single dispatch to reduce context-switching overhead and cost.
* Review Loop: Implementation agents write the code; separate "Reviewer" agents check the output against the Gherkin scenarios defined in Phase 3.

### Phase 7: Review, Correction, and Learning

AI-driven development is an iterative loop, not a one-way street.

* Primary Skills:
    * `verification-before-completion` (Superpowers): A hard gate that forces the AI to prove the code works against the Gherkin specs before it is allowed to claim the task is done.
    * `requesting-code-review` & `receiving-code-review` (Superpowers): Dispatches a separate "Reviewer" subagent to check the Implementer's code. The reviewer evaluates the diff against the spec, not the session history.
    * `systematic-debugging` (Superpowers): Triggered only if tests fail. It forces the AI to find the root cause rather than guessing.
    * `finishing-a-development-branch` (Superpowers): Handles the PR creation, merging, and crucially, deletes the ephemeral .skillgrid/sdd/ workspace.

* Validation: Ensure the implemented code matches the observable behavior defined in the specs, not just the literal instructions.
* Correction: If the AI hallucinates or drifts, correct the behavior and update the ADR or Spec to prevent the same error in the future.
* Durable Context: Ensure that all corrections and new knowledge are saved as durable context. The AI should "learn" from the project history so it doesn't repeat mistakes in future sessions.

## File Structure

```
openspec/
└── changes/            # Per-change traveling artifacts
    └── <id>/
        ├── idea.md     # The "Intent" (Raw intent + right-sizing)
        ├── proposal.md # The "Why" (Business value, scope)
        ├── specs/      # The "What" (Gherkin scenarios)
        ├── design.md   # The "How" (Architecture, conditional)
        ├── adr.md      # Architectural Decision Records
        └── tasks.md    # Implementation tasks for SDD

```

```
.skillgrid/          # Hidden from normal view
└── sdd/               # Subagent-Driven Development workspace
    └── 001-guest-checkout/    # Plan-scoped (deleted after merge)
        ├── plan.md            # Master execution plan
        ├── ledger.md          # Conflict rulings (never-stall)
        ├── task-briefs/       # Context for implementer subagents
        ├── review-packages/   # Context for reviewer subagents
        └── finish-report.md   # Final summary + all rulings surfaced
```

```
.worktrees/
```

* Deep research
* automated testing (100% covarage)
* documentation

* Spec Driven Development
* Test Driven Development
* Intent Driven Development

* memory
* index
* browser automatization