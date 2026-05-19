---
name: sdd-verify
id: sdd-verify
category: Workflow
description: Validate implementation matches specs, design, and tasks
agent: odin
subtask: true
---

You are an SDD sub-agent. Read the skill file at `.agents/skills/sdd-verify/SKILL.md` FIRST, then follow its instructions exactly.

**VDD VERIFICATION — run alongside standard verification:**
- Read `.agents/skills/vdd-verify/SKILL.md`
- After running standard verification steps, execute the VDD verification checklist:
  - Run project test suite and record pass/fail (use existing test infrastructure)
  - Run type check if applicable (TypeScript, Python, etc.)
  - Run linter for code quality
  - Verify code builds/compiles successfully
  - Check documentation exists
  - Detect hallucinated tests (tests that pass but don't test the feature)

**VDD HALLUCINATION DETECTION:**
- Flag tests with assertions that always pass (e.g., `expect(true).toBe(true)`)
- Flag tests that don't cover their stated requirements
- Cross-reference verified requirements with existing test coverage

**VDD CONVERGE — check zero-slop during verify:**
- Read `.agents/skills/vdd-converge/SKILL.md`
- Run adversarial review (vdd-roast) and collect critiques with severity levels
- For each critique, verify: does the referenced code exist? is the problem real? would fixing it improve correctness?
- A critique is hallucinated when: criticizing code that doesn't exist, suggesting changes to working/tested code without valid reason, finding problems that aren't problems, repeating previously addressed issues
- Calculate hallucination ratio: `HALLUCINATIONS / TOTAL_CRITIQUES`
- Use threshold from `.skillgrid/config.json` (default: 0.7)
- Convergence result MUST be included in the verify return envelope (`converged: true/false`, `hallucination_ratio`, `legitimate_flaws`)
- If converged: add to `next_recommended`: "Zero-slop achieved. Safe to proceed to /sdd-archive."
- If not converged: add to `next_recommended`: "Not yet converged. N legitimate flaws remain. Address before archive."
- Do NOT allow archive progression on unresolved critical findings from converge check.

CONTEXT:
- Working directory: !`echo -n "$(pwd)"`
- Current project: !`echo -n "$(basename $(pwd))"`
- Artifact store mode: hybrid

TASK:
Verify the active SDD change. Read the proposal, specs, design, and tasks artifacts. Then:

MANDATORY PRECHECK:
- Run `.skillgrid/scripts/sdd-gate.sh verify --change {change-name}` before verification (labels, artifacts, slices, persona hard gates — does **not** require review artifacts).
- If validation fails, report a CRITICAL gate failure and return FAIL.
- If required artifacts (`proposal`, `spec`, `design`, `tasks`) are missing, fail closed with `status: failed`.
- After PASS, run `/sdd-review` — `sdd-gate.sh review` blocks until verify evidence exists.

CHECKPOINT (after verification completes):
- When verdict is **PASS** or **PASS WITH WARNINGS**, before returning:
  ```bash
  .skillgrid/scripts/checkpoint-record.sh --change {change-name} --name verify-pass --trigger verify-pass --phase verify --evidence "verify PASS"
  ```

ENGRAM PERSISTENCE (artifact store mode: engram):
CRITICAL: mem_search returns 300-char PREVIEWS, not full content. You MUST call mem_get_observation(id) for EVERY artifact.
STEP A — SEARCH (get IDs only):
  mem_search(query: "sdd/{change-name}/spec", project: "{project}") → save spec_id
  mem_search(query: "sdd/{change-name}/design", project: "{project}") → save design_id
  mem_search(query: "sdd/{change-name}/tasks", project: "{project}") → save tasks_id
STEP B — RETRIEVE FULL CONTENT (mandatory):
  mem_get_observation(id: spec_id) → full spec
  mem_get_observation(id: design_id) → full design
  mem_get_observation(id: tasks_id) → full tasks
Save report:
  mem_save(title: "sdd/{change-name}/verify-report", topic_key: "sdd/{change-name}/verify-report", type: "architecture", project: "{project}", content: "{verification report}")
FILESYSTEM PERSISTENCE:
  Read `.agents/skills/_shared/skillgrid-handoff.md` for filesystem persistence instructions.

Then:
1. **Read `.skillgrid/project/CONTEXT.md`** if it exists. Note any relevant glossary terms, assumptions, or success criteria before proceeding.
2. Check completeness — are all tasks done?
3. Check correctness — does code match specs?
4. Check coherence — were design decisions followed?
5. Run tests and build (real execution)
6. Build the spec compliance matrix

BOARD ESCALATION (critical/conflict path):
PERSONA INVOCATIONS (coordinator — see `sdd-verify/SKILL.md`):
- Required: `tyr` → `spec-compliance`
- When applicable: `heimdall` → `security-review` / `release-gate`; `frigg` → `ux-clarity`; `loki` → `assumption-stress-test` on conflict
Protocol: `.agents/skills/_shared/sdd-persona-delegation.md`

- If verification evidence is conflicting, dispatch additional persona invocations from `sdd-verify/SKILL.md` before final status.
- Use board presets based on decision type:
  - architecture -> `odin`, `thor`, `tyr`, `loki`
  - security -> `heimdall`, `tyr`, `thor`, `loki`
  - go-no-go-release -> `odin`, `tyr`, `heimdall`, `thor`, `frigg`
- Persist board outputs to:
  - `.skillgrid/tasks/research/<change-id>/`
  - `.skillgrid/tasks/context_<change-id>.md`
  - `.skillgrid/tasks/events/<change-id>.jsonl`
- Enforce hard block semantics:
  - `tyr` critical finding => `status: failed`
  - `heimdall` critical finding => `status: failed`
  - unresolved persona conflict => `status: blocked` (HITL required)

ENFORCEMENT CONTRACT:
- Canonical enforcement is centralized in `.agents/skills/_shared/sdd-enforcement-contract.md`.
- This workflow MUST apply that shared contract for:
  - phase routing and stop conditions
  - mandatory skill-gate checks
  - two-stage review gate
  - standard return envelope
- Verify-specific progression rule:
  - any critical finding in Stage 1 or Stage 2 => `status: failed` with explicit remediation in `next_recommended`
  - board escalation failures or unresolved board conflicts must prevent progression to `/sdd-archive`
  - both stages pass => allow progression to `/sdd-archive`

Return the standard SDD envelope per `.agents/skills/_shared/sdd-return-envelope.md`. Include verify/VDD extension fields from that contract when applicable.

---

## OpenSpec CLI supplement (integrated from former opsx-verify)

Verify that an implementation matches the change artifacts (specs, tasks, design).

**Input**: Optionally specify a change name after `/sdd-verify` (e.g., `/sdd-verify add-auth`). If omitted, check if it can be inferred from conversation context. If vague or ambiguous you MUST prompt for available changes.

**Steps**

1. **If no change name provided, prompt for selection**

   Run `openspec list --json` to get available changes. Use the **AskUserQuestion tool** to let the user select.

   Show changes that have implementation tasks (tasks artifact exists).
   Include the schema used for each change if available.
   Mark changes with incomplete tasks as "(In Progress)".

   **IMPORTANT**: Do NOT guess or auto-select a change. Always let the user choose.

2. **Check status to understand the schema**
   ```bash
   openspec status --change "<name>" --json
   ```
   Parse the JSON to understand:
   - `schemaName`: The workflow being used (e.g., "spec-driven")
   - Which artifacts exist for this change

3. **Get the change directory and load artifacts**

   ```bash
   openspec instructions apply --change "<name>" --json
   ```

   This returns the change directory and context files. Read all available artifacts from `contextFiles`.

4. **Initialize verification report structure**

   Create a report structure with three dimensions:
   - **Completeness**: Track tasks and spec coverage
   - **Correctness**: Track requirement implementation and scenario coverage
   - **Coherence**: Track design adherence and pattern consistency

   Each dimension can have CRITICAL, WARNING, or SUGGESTION issues.

5. **Verify Completeness**

   **Task Completion**:
   - If tasks.md exists in contextFiles, read it
   - Parse checkboxes: `- [ ]` (incomplete) vs `- [x]` (complete)
   - Count complete vs total tasks
   - If incomplete tasks exist:
     - Add CRITICAL issue for each incomplete task
     - Recommendation: "Complete task: <description>" or "Mark as done if already implemented"

   **Spec Coverage**:
   - If delta specs exist in `openspec/changes/<name>/specs/`:
     - Extract all requirements (marked with "### Requirement:")
     - For each requirement:
       - Search codebase for keywords related to the requirement
       - Assess if implementation likely exists
     - If requirements appear unimplemented:
       - Add CRITICAL issue: "Requirement not found: <requirement name>"
       - Recommendation: "Implement requirement X: <description>"

6. **Verify Correctness**

   **Requirement Implementation Mapping**:
   - For each requirement from delta specs:
     - Search codebase for implementation evidence
     - If found, note file paths and line ranges
     - Assess if implementation matches requirement intent
     - If divergence detected:
       - Add WARNING: "Implementation may diverge from spec: <details>"
       - Recommendation: "Review <file>:<lines> against requirement X"

   **Scenario Coverage**:
   - For each scenario in delta specs (marked with "#### Scenario:"):
     - Check if conditions are handled in code
     - Check if tests exist covering the scenario
     - If scenario appears uncovered:
       - Add WARNING: "Scenario not covered: <scenario name>"
       - Recommendation: "Add test or implementation for scenario: <description>"

7. **Verify Coherence**

   **Design Adherence**:
   - If design.md exists in contextFiles:
     - Extract key decisions (look for sections like "Decision:", "Approach:", "Architecture:")
     - Verify implementation follows those decisions
     - If contradiction detected:
       - Add WARNING: "Design decision not followed: <decision>"
       - Recommendation: "Update implementation or revise design.md to match reality"
   - If no design.md: Skip design adherence check, note "No design.md to verify against"

   **Code Pattern Consistency**:
   - Review new code for consistency with project patterns
   - Check file naming, directory structure, coding style
   - If significant deviations found:
     - Add SUGGESTION: "Code pattern deviation: <details>"
     - Recommendation: "Consider following project pattern: <example>"

8. **Generate Verification Report**

   **Summary Scorecard**:
   ```
   ## Verification Report: <change-name>

   ### Summary
   | Dimension    | Status           |
   |--------------|------------------|
   | Completeness | X/Y tasks, N reqs|
   | Correctness  | M/N reqs covered |
   | Coherence    | Followed/Issues  |
   ```

   **Issues by Priority**:

   1. **CRITICAL** (Must fix before archive):
      - Incomplete tasks
      - Missing requirement implementations
      - Each with specific, actionable recommendation

   2. **WARNING** (Should fix):
      - Spec/design divergences
      - Missing scenario coverage
      - Each with specific recommendation

   3. **SUGGESTION** (Nice to fix):
      - Pattern inconsistencies
      - Minor improvements
      - Each with specific recommendation

   **Final Assessment**:
   - If CRITICAL issues: "X critical issue(s) found. Fix before archiving."
   - If only warnings: "No critical issues. Y warning(s) to consider. Ready for archive (with noted improvements)."
   - If all clear: "All checks passed. Ready for archive."

**Verification Heuristics**

- **Completeness**: Focus on objective checklist items (checkboxes, requirements list)
- **Correctness**: Use keyword search, file path analysis, reasonable inference - don't require perfect certainty
- **Coherence**: Look for glaring inconsistencies, don't nitpick style
- **False Positives**: When uncertain, prefer SUGGESTION over WARNING, WARNING over CRITICAL
- **Actionability**: Every issue must have a specific recommendation with file/line references where applicable

**Graceful Degradation**

- If only tasks.md exists: verify task completion only, skip spec/design checks
- If tasks + specs exist: verify completeness and correctness, skip design
- If full artifacts: verify all three dimensions
- Always note which checks were skipped and why

**Output Format**

Use clear markdown with:
- Table for summary scorecard
- Grouped lists for issues (CRITICAL/WARNING/SUGGESTION)
- Code references in format: `file.ts:123`
- Specific, actionable recommendations
- No vague suggestions like "consider reviewing"

