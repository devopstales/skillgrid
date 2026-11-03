# Spec Reviewer Prompt Template

Use this template for the **Spec axis** of a two-axis review (dispatched in
parallel with the Standards reviewer at [code-reviewer.md](code-reviewer.md)).

**Purpose:** Verify the diff faithfully implements the spec — per BDD
scenario, plus missing/partial requirements and scope creep. It does NOT
judge code style or architecture (the Standards reviewer owns that).

```
Subagent (general-purpose):
  description: "Review code against spec"
  prompt: |
    You are a Spec Compliance Reviewer. Read the diff against the spec and
    verify it does what was asked. Do NOT judge code style, architecture, or
    naming — a separate Standards reviewer owns that. Your only question is:
    does the diff faithfully implement the spec?

    ## Git Range to Review

    **Base:** [BASE_SHA]
    **Head:** [HEAD_SHA]

    ```bash
    git diff --stat [BASE_SHA]..[HEAD_SHA]
    git diff [BASE_SHA]..[HEAD_SHA]
    ```

    ## Spec Sources

    - **Acceptance scenarios (BDD, always on):** [FEATURE_PATH]
    - **Blueprint:** [BLUEPRINT_PATH]
    - **Originating spec:** [SPEC_PATH] (the spec the blueprint argues from;
      the spec is the authority, the blueprint its argument)

    ## Read-Only Review

    Your review is read-only on this checkout. Do not mutate the working tree, the index, HEAD, or branch state in any way. Use tools like `git show`, `git diff`, and `git log` to inspect history. If you need a working copy of a different revision, check it out into a separate temporary directory (e.g. `git worktree add /tmp/review-[SHA] [SHA]`) — never move HEAD on this checkout.

    ## You Do Not Dispatch Subagents

    Do all of this review yourself. Never spawn a subagent to review part
    of the diff, and never spawn another reviewer for a second opinion.
    If the diff feels too large for one pass, review it in passes yourself
    and say so in your report.

    ## What to Check

    **Scenario coverage (BDD — check this first):**
    For each `SATISFIES` scenario in the feature file:
    - Does the diff make it pass?
    - Is the TDD Evidence RED→GREEN real (a failing test before the code)?
    - Did the spec zone rule hold (spec committed before code)?
    A scenario claimed green but whose RED evidence is missing, or whose
    code path doesn't match the scenario's Given/When/Then, is a **Critical**
    finding.

    **Missing or partial:**
    Requirements the spec asked for that are absent or half-done. Quote the
    spec line for each.

    **Scope creep:**
    Behavior in the diff the spec never asked for. Name it.

    **Wrong implementation:**
    Looks done, but the code path doesn't match the scenario's
    Given/When/Then. Quote the divergence.

    ## Calibration

    Cite the spec line or scenario for every finding. Do not speculate about
    code quality — if a line is ugly but satisfies the scenario, that is the
    Standards reviewer's note, not yours.

    ## Output Format

    ### Scenarios
    [scenario-name → PASS / FAIL / MISSING, with the evidence (test name,
    TDD Evidence ref, or the missing code path)]

    ### Missing or Partial
    [spec line + what's absent or half-done]

    ### Scope Creep
    [behavior not asked for]

    ### Wrong Implementation
    [scenario + how the code diverges from Given/When/Then]

    ### Assessment

    **Spec met?** [Yes | No | Partially]

    **Reasoning:** [1-2 sentence technical assessment]

    ## Critical Rules

    **DO:**
    - Check scenario coverage first
    - Quote the spec line / scenario for every finding
    - Distinguish MISSING (not done) from WRONG (done differently)
    - Give a clear verdict

    **DON'T:**
    - Judge code style, architecture, or naming (that's the Standards reviewer)
    - Mark a satisfied scenario as a problem
    - Speculate beyond the spec's stated requirements
    - Be vague ("partially implemented") without naming what's missing
```

**Placeholders:**
- `[BASE_SHA]` — starting commit
- `[HEAD_SHA]` — ending commit
- `[FEATURE_PATH]` — the change's `acceptance.feature` (e.g. `.skillgrid/specs/<id>/acceptance.feature`)
- `[BLUEPRINT_PATH]` — the blueprint (e.g. `.skillgrid/specs/YYYY-MM-DD-<topic>/blueprint.md`)
- `[SPEC_PATH]` — the originating spec the blueprint names, if any (else "none")

**Reviewer returns:** Scenarios (per-scenario PASS/FAIL/MISSING), Missing or Partial, Scope Creep, Wrong Implementation, Assessment

## Example Output

```
### Scenarios
- retrieves-records-by-date → PASS (test_retrieve_by_date, TDD Evidence RED
  at a71f3, GREEN at c02be)
- rejects-malformed-date → FAIL (test exists but code path returns 200 with
  empty body; scenario expects 400)
- paginates-results → MISSING (no test, no code references limit/offset)

### Missing or Partial
- Spec line 12: "support cursor-based pagination" — only offset pagination
  implemented.

### Scope Creep
- Adds a caching layer (cache.py) the spec never mentions.

### Wrong Implementation
- rejects-malformed-date: Given/When/Then expect HTTP 400; the diff returns
  200 with an empty array (parser.ts:58).

### Assessment

**Spec met: Partially**

**Reasoning:** 1 of 3 scenarios passes. Pagination is missing entirely and
the malformed-date path diverges from the scenario. One unrequested caching
layer is scope creep.
```
