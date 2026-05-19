---
name: sdd-brainstorm
id: sdd-brainstorm
category: Workflow
description: Start a new SDD change — runs exploration, clarification, proposal, specs, design (including UI), ADRs, PRD, and tasks
agent: odin
---

Follow the SDD orchestrator workflow for starting a new change named "$ARGUMENTS".

WORKFLOW:
1. **Read `.skillgrid/project/CONTEXT.md`** if it exists. Note any relevant glossary terms, assumptions, or success criteria before proceeding.
2. Launch `sdd-explore` sub-agent. The explore prompt **must** require:
   - Load `deep-research` (`.agents/skills/deep-research/SKILL.md`)
   - **First search rule:** Exa/Tavily/Firecrawl web research **before** application code
   - `### External research (first search)` in exploration output
3. Present the exploration summary to the user
4. Launch `sdd-clarify` sub-agent to refine understanding:
   - It will ask questions to resolve ambiguity about goals, scope, and constraints.
   - It captures the mutual understanding in a `clarifications.md` artifact.
5. Present the clarifications summary and ask the user to confirm or iterate.
6. Once mutual understanding is confirmed, launch `sdd-propose` sub-agent
7. Present the proposal summary and ask the user if they want to continue with specs and design

### UI DESIGN INTEGRATION POINT

If the proposal involves user-facing changes (components, layouts, interactions, visual updates):

- Flag the change with `ui_scope: true` in the proposal metadata
- After `sdd-design`, launch the UI design sub-flow:
  a. `engram-ui-elements` — validate component composition & reusability
  b. `engram-visual-language` — apply spacing, typography, color, and accessibility rules
  c. `sdd-ui-design` (if available) — generate wireframes, interaction flows, or ASCII/Mermaid mockups
- Merge UI artifacts into the technical design before proceeding to PRD

### ARCHITECTURAL DECISION RECORDS (ADR) INTEGRATION POINT

If exploration or clarification uncovers significant architectural decisions, capture them **inside `sdd-design`** (not a separate phase):

- Instruct the design sub-agent to load `architectural-decision-records` when ADR triggers apply
- ADR triggers: technology stack changes, framework selections, data storage decisions, API patterns, security approaches
- One ADR per decision in `.skillgrid/adr/`; link related ADRs; update `.skillgrid/project/ARCHITECTURE.md` under **Durable decisions**
- ADRs should precede PRD finalization

### CONTINUE WITH STANDARD PHASES

Run these sub-agents in sequence after clarification (and UI design if applicable):

1. `sdd-propose` — create the proposal (using the clarifications)
2. `sdd-spec` — write functional & technical specifications
3. `sdd-design` — technical design architecture **and ADRs** (when triggers apply; see ADR integration above)
    → **If UI scope**: trigger UI design sub-flow above, then merge outputs
4. `sdd-prd` — consolidate into PRD (`.skillgrid/prd/PRD<NN>_$ARGUMENTS.md`)
    - Include UI decisions, wireframe references, and accessibility notes in the PRD
5. `sdd-tasks` — break down into vertically sliced implementation tasks
    - Tag UI-related tasks with `area: frontend` or `area: ui`
6. `beads-sync` — sync OpenSpec changes to Beads tasks (run after brainstorm completes)

CONTEXT:
- Working directory: !`echo -n "$(pwd)"`
- Current project: !`echo -n "$(basename $(pwd))"`
- Change name: $ARGUMENTS
- Artifact store mode: hybrid
- UI scope detection: Analyze proposal for keywords: [ui, ux, frontend, component, layout, visual, wireframe, mockup, css, styling, accessibility]

ENGRAM NOTE:
Sub-agents handle persistence automatically. Each phase saves its artifact to engram with topic_key:
- `"sdd/$ARGUMENTS/{type}"` where type ∈ {explore, clarify, propose, spec, design, prd, tasks}
- UI-specific artifacts use: `"sdd/$ARGUMENTS/ui/{wireframes, decisions, tokens}"`

FILESYSTEM PERSISTENCE:
- Read `.agents/skills/_shared/skillgrid-handoff.md` for filesystem persistence instructions.
- UI artifacts in hybrid mode should also write to:

```
openspec/changes/$ARGUMENTS/
  ├── ui-wireframes.md      # ASCII/Mermaid diagrams
  ├── ui-decisions.md       # Design rationale & tradeoffs
  └── ui-tokens.md          # Color, spacing, typography references (if project uses design tokens)
```

ORCHESTRATOR RULES:
- When delegating `sdd-explore`, inject: `SKILL: Load deep-research` and `First search before codebase`.
- Do NOT execute phase work inline — delegate to sub-agents.
- When delegating `sdd-design`, include ADR triggers and context from explore/clarify if architectural decisions were discovered
- Before launching `sdd-tasks`, verify:
  - Technical design is complete
  - UI artifacts are merged (if ui_scope: true)
  - Accessibility considerations are documented
  - All artifacts are persisted per artifact_store mode
- Before launching `beads-sync`, verify:
  - `proposal.md` exists in `openspec/changes/$ARGUMENTS/`
  - `tasks.md` exists in `openspec/changes/$ARGUMENTS/`
  - `beads_enable` is `true` in `.skillgrid/config.json`
- If user interrupts or requests changes, use `mem_get_observation` to reload prior phase artifacts instead of re-running.

BEADS INTEGRATION:
- Read `beads_enable` from `.skillgrid/config.json` to determine if syncing is enabled
- The `beads-sync` skill reads `openspec/changes/$ARGUMENTS/proposal.md` and `tasks.md`
- Creates an epic with child beads for each implementation task
- Run `bd ready` to see available work after sync completes
- `beads-retrospective` runs in `sdd-verify` after implementation is validated

ENFORCEMENT CONTRACT:
- Canonical enforcement is centralized in `.agents/skills/_shared/sdd-enforcement-contract.md`.
- This workflow MUST apply that shared contract for:
  - phase routing and stop conditions
  - mandatory skill-gate checks
  - two-stage review gate
  - standard return envelope (`.agents/skills/_shared/sdd-return-envelope.md`) from each delegated phase
- Brainstorm-specific override:
  - For planning-only execution, Stage 2 (`code quality`) is `not_applicable` unless implementation code was produced in error.

UI DESIGN HANDOFF CONTRACT:
When UI sub-agents complete, ensure the design artifact includes:

```yaml
ui_design_summary:
components_affected: [List<ComponentName>]
wireframe_refs: ["path/to/wireframe.md#section"]
accessibility_notes: ["contrast ratios", "keyboard nav", "ARIA labels"]
responsive_breakpoints: ["mobile", "tablet", "desktop"]
design_token_usage: ["color.primary", "spacing.md", "typography.body"]
risks: ["browser compatibility", "performance impact"]
```

Read the **Odin-primary** coordinator rules (`skillgrid-sdd-orchestrator.mdc` + `skillgrid-gentle-orchestrator-extended.mdc`) to coordinate this workflow. Do NOT execute phase work inline — delegate to sub-agents.
---

## OpenSpec CLI supplements (integrated from former opsx-new, opsx-continue, opsx-propose)

Use these when the user drives **`openspec/changes/`** via the **OpenSpec CLI** instead of only delegated SDD sub-agents. Subsections preserve the prior step-by-step contracts.

### OpenSpec — new change (stepwise, former opsx-new)

Start a new change using the experimental artifact-driven approach.

**Input**: The argument after `/sdd-brainstorm (OpenSpec new-change steps below)` is the change name (kebab-case), OR a description of what the user wants to build.

**Steps**

1. **If no input provided, ask what they want to build**

   Use the **AskUserQuestion tool** (open-ended, no preset options) to ask:
   > "What change do you want to work on? Describe what you want to build or fix."

   From their description, derive a kebab-case name (e.g., "add user authentication" → `add-user-auth`).

   **IMPORTANT**: Do NOT proceed without understanding what the user wants to build.

2. **Determine the workflow schema**

   Use the default schema (omit `--schema`) unless the user explicitly requests a different workflow.

   **Use a different schema only if the user mentions:**
   - A specific schema name → use `--schema <name>`
   - "show workflows" or "what workflows" → run `openspec schemas --json` and let them choose

   **Otherwise**: Omit `--schema` to use the default.

3. **Create the change directory**
   ```bash
   openspec new change "<name>"
   ```
   Add `--schema <name>` only if the user requested a specific workflow.
   This creates a scaffolded change at `openspec/changes/<name>/` with the selected schema.

4. **Show the artifact status**
   ```bash
   openspec status --change "<name>"
   ```
   This shows which artifacts need to be created and which are ready (dependencies satisfied).

5. **Get instructions for the first artifact**
   The first artifact depends on the schema. Check the status output to find the first artifact with status "ready".
   ```bash
   openspec instructions <first-artifact-id> --change "<name>"
   ```
   This outputs the template and context for creating the first artifact.

6. **STOP and wait for user direction**

**Output**

After completing the steps, summarize:
- Change name and location
- Schema/workflow being used and its artifact sequence
- Current status (0/N artifacts complete)
- The template for the first artifact
- Prompt: "Ready to create the first artifact? Run `/sdd-brainstorm (OpenSpec continue steps below)` or just describe what this change is about and I'll draft it."

**Guardrails**
- Do NOT create any artifacts yet - just show the instructions
- Do NOT advance beyond showing the first artifact template
- If the name is invalid (not kebab-case), ask for a valid name
- If a change with that name already exists, suggest using `/sdd-brainstorm (OpenSpec continue steps below)` instead
- Pass --schema if using a non-default workflow


### OpenSpec — continue (next artifact, former opsx-continue)

Continue working on a change by creating the next artifact.

**Input**: Optionally specify a change name after `/sdd-brainstorm (OpenSpec continue steps below)` (e.g., `/sdd-brainstorm (OpenSpec continue steps below) add-auth`). If omitted, check if it can be inferred from conversation context. If vague or ambiguous you MUST prompt for available changes.

**Steps**

1. **If no change name provided, prompt for selection**

   Run `openspec list --json` to get available changes sorted by most recently modified. Then use the **AskUserQuestion tool** to let the user select which change to work on.

   Present the top 3-4 most recently modified changes as options, showing:
   - Change name
   - Schema (from `schema` field if present, otherwise "spec-driven")
   - Status (e.g., "0/5 tasks", "complete", "no tasks")
   - How recently it was modified (from `lastModified` field)

   Mark the most recently modified change as "(Recommended)" since it's likely what the user wants to continue.

   **IMPORTANT**: Do NOT guess or auto-select a change. Always let the user choose.

2. **Check current status**
   ```bash
   openspec status --change "<name>" --json
   ```
   Parse the JSON to understand current state. The response includes:
   - `schemaName`: The workflow schema being used (e.g., "spec-driven")
   - `artifacts`: Array of artifacts with their status ("done", "ready", "blocked")
   - `isComplete`: Boolean indicating if all artifacts are complete

3. **Act based on status**:

   ---

   **If all artifacts are complete (`isComplete: true`)**:
   - Congratulate the user
   - Show final status including the schema used
   - Suggest: "All artifacts created! You can now implement this change with `/sdd-apply` or archive it with `/sdd-archive`."
   - STOP

   ---

   **If artifacts are ready to create** (status shows artifacts with `status: "ready"`):
   - Pick the FIRST artifact with `status: "ready"` from the status output
   - Get its instructions:
     ```bash
     openspec instructions <artifact-id> --change "<name>" --json
     ```
   - Parse the JSON. The key fields are:
     - `context`: Project background (constraints for you - do NOT include in output)
     - `rules`: Artifact-specific rules (constraints for you - do NOT include in output)
     - `template`: The structure to use for your output file
     - `instruction`: Schema-specific guidance
     - `outputPath`: Where to write the artifact
     - `dependencies`: Completed artifacts to read for context
   - **Create the artifact file**:
     - Read any completed dependency files for context
     - Use `template` as the structure - fill in its sections
     - Apply `context` and `rules` as constraints when writing - but do NOT copy them into the file
     - Write to the output path specified in instructions
   - Show what was created and what's now unlocked
   - STOP after creating ONE artifact

   ---

   **If no artifacts are ready (all blocked)**:
   - This shouldn't happen with a valid schema
   - Show status and suggest checking for issues

4. **After creating an artifact, show progress**
   ```bash
   openspec status --change "<name>"
   ```

**Output**

After each invocation, show:
- Which artifact was created
- Schema workflow being used
- Current progress (N/M complete)
- What artifacts are now unlocked
- Prompt: "Run `/sdd-brainstorm (OpenSpec continue steps below)` to create the next artifact"

**Artifact Creation Guidelines**

The artifact types and their purpose depend on the schema. Use the `instruction` field from the instructions output to understand what to create.

Common artifact patterns:

**spec-driven schema** (proposal → specs → design → tasks):
- **proposal.md**: Ask user about the change if not clear. Fill in Why, What Changes, Capabilities, Impact.
  - The Capabilities section is critical - each capability listed will need a spec file.
- **specs/<capability>/spec.md**: Create one spec per capability listed in the proposal's Capabilities section (use the capability name, not the change name).
- **design.md**: Document technical decisions, architecture, and implementation approach.
- **tasks.md**: Break down implementation into checkboxed tasks.

For other schemas, follow the `instruction` field from the CLI output.

**Guardrails**
- Create ONE artifact per invocation
- Always read dependency artifacts before creating a new one
- Never skip artifacts or create out of order
- If context is unclear, ask the user before creating
- Verify the artifact file exists after writing before marking progress
- Use the schema's artifact sequence, don't assume specific artifact names
- **IMPORTANT**: `context` and `rules` are constraints for YOU, not content for the file
  - Do NOT copy `<context>`, `<rules>`, `<project_context>` blocks into the artifact
  - These guide what you write, but should never appear in the output


### OpenSpec — propose (all artifacts in one pass, former opsx-propose)

Propose a new change - create the change and generate all artifacts in one step.

I'll create a change with artifacts:
- proposal.md (what & why)
- design.md (how)
- tasks.md (implementation steps)

When ready to implement, run /sdd-apply

---

**Input**: The argument after `/sdd-brainstorm (OpenSpec propose steps below)` is the change name (kebab-case), OR a description of what the user wants to build.

**Steps**

1. **If no input provided, ask what they want to build**

   Use the **AskUserQuestion tool** (open-ended, no preset options) to ask:
   > "What change do you want to work on? Describe what you want to build or fix."

   From their description, derive a kebab-case name (e.g., "add user authentication" → `add-user-auth`).

   **IMPORTANT**: Do NOT proceed without understanding what the user wants to build.

2. **Create the change directory**
   ```bash
   openspec new change "<name>"
   ```
   This creates a scaffolded change at `openspec/changes/<name>/` with `.openspec.yaml`.

3. **Get the artifact build order**
   ```bash
   openspec status --change "<name>" --json
   ```
   Parse the JSON to get:
   - `applyRequires`: array of artifact IDs needed before implementation (e.g., `["tasks"]`)
   - `artifacts`: list of all artifacts with their status and dependencies

4. **Create artifacts in sequence until apply-ready**

   Use the **TodoWrite tool** to track progress through the artifacts.

   Loop through artifacts in dependency order (artifacts with no pending dependencies first):

   a. **For each artifact that is `ready` (dependencies satisfied)**:
      - Get instructions:
        ```bash
        openspec instructions <artifact-id> --change "<name>" --json
        ```
      - The instructions JSON includes:
        - `context`: Project background (constraints for you - do NOT include in output)
        - `rules`: Artifact-specific rules (constraints for you - do NOT include in output)
        - `template`: The structure to use for your output file
        - `instruction`: Schema-specific guidance for this artifact type
        - `outputPath`: Where to write the artifact
        - `dependencies`: Completed artifacts to read for context
      - Read any completed dependency files for context
      - Create the artifact file using `template` as the structure
      - Apply `context` and `rules` as constraints - but do NOT copy them into the file
      - Show brief progress: "Created <artifact-id>"

   b. **Continue until all `applyRequires` artifacts are complete**
      - After creating each artifact, re-run `openspec status --change "<name>" --json`
      - Check if every artifact ID in `applyRequires` has `status: "done"` in the artifacts array
      - Stop when all `applyRequires` artifacts are done

   c. **If an artifact requires user input** (unclear context):
      - Use **AskUserQuestion tool** to clarify
      - Then continue with creation

5. **Show final status**
   ```bash
   openspec status --change "<name>"
   ```

**Output**

After completing all artifacts, summarize:
- Change name and location
- List of artifacts created with brief descriptions
- What's ready: "All artifacts created! Ready for implementation."
- Prompt: "Run `/sdd-apply` to start implementing."

**Artifact Creation Guidelines**

- Follow the `instruction` field from `openspec instructions` for each artifact type
- The schema defines what each artifact should contain - follow it
- Read dependency artifacts for context before creating new ones
- Use `template` as the structure for your output file - fill in its sections
- **IMPORTANT**: `context` and `rules` are constraints for YOU, not content for the file
  - Do NOT copy `<context>`, `<rules>`, `<project_context>` blocks into the artifact
  - These guide what you write, but should never appear in the output

**Guardrails**
- Create ALL artifacts needed for implementation (as defined by schema's `apply.requires`)
- Always read dependency artifacts before creating a new one
- If context is critically unclear, ask the user - but prefer making reasonable decisions to keep momentum
- If a change with that name already exists, ask if user wants to continue it or create a new one
- Verify each artifact file exists after writing before proceeding to next


