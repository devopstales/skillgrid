## Bite-Sized Task Granularity

**Each step is one action (2-5 minutes):**
- "Write the failing test" - step
- "Run it to make sure it fails" - step
- "Implement the minimal code to make the test pass" - step
- "Run the tests and make sure they pass" - step
- "Commit" - step

## Plan Document Header

**Every plan MUST start with this header** (the `../templates/blueprint.md` file
includes it plus a sample task — use it as the starting scaffold):

```markdown
# [Feature Name] Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:subagent-execution (recommended) or skillgrid:simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** PROPOSED (default. Set to `ASSUMED` only via the Owed-Decision
Gate option 3 — the Assumed block follows the header; cleared only by Ratify.)

**Tier:** T<n> (rigor tier — from `briefing.md`'s `Tier:` line, else
`rules.tiers.default` in config, else inferred from the Change Classification
and noted as an assumption. See `../../../_shared/conventions/rigor-tiers.md`. T1 →
light blueprint (header + Must-Haves, no threat matrix unless applicable);
T2/T3 → full blueprint.)

**Build shape:** <Tracer thread | Smallest usable whole | Facade | Journey>
(the recorded delivery strategy that orders the first execution wave — see the
Build Shape section; `skillgrid:slicing` reads this line.)

**Goal:** [One sentence describing what this builds]

**Architecture:** [2-3 sentences about approach]

**Tech Stack:** [Key technologies/libraries]

**Spec:** [path to the spec/design doc this plan implements — the plan
argues from the spec, so the spec travels with it; executors read both]

**Findings:** [path to the change's consolidated
`.skillgrid/specs/YYYY-MM-DD-<topic>/findings.md` if it exists — produced by
`skillgrid:research` / `skillgrid:code-research` (Research sections),
`skillgrid:spike` (Spike sections), or `skillgrid:sketch` (Sketch sections).
Cite it for every design decision that rests on a research fact, a feasibility
result (e.g. "X is liftable", "approach A is cheaper than B"), or a chosen
layout/interaction. Omit if the change ran no research/spike/sketch.]

 ## Global Constraints

  [The spec's project-wide requirements — version floors, dependency limits,
  naming and copy rules, platform requirements — one line each, with exact
  values copied verbatim from the spec. Every task's requirements implicitly
  include this section. For a constraint or decision that is NOT earned by this
  change's spec but inherited from a locked constraint or a prior ADR, CITE it
  rather than restate it (`per ASSUMPTIONS.md § Locked constraints <x>` /
  `per .skillgrid/artifacts/04-adr-NNNN-slug.md`) per
  `_shared/conventions/cite-dont-restate.md` — the citation is the
  constraint; the cited artifact keeps authority singular.]

---
```

## Task Structure

````markdown
### Task N: [Component Name]

**Files:**
- Create: `exact/path/to/file.py`
- Modify: `exact/path/to/existing.py:123-145`
- Test: `tests/exact/path/to/test.py`

**Interfaces:**
- Consumes: [what this task uses from earlier tasks — exact signatures]
- Produces: [what later tasks rely on — exact function names, parameter
  and return types. A task's implementer sees only their own task; this
  block is how they learn the names and types neighboring tasks use.]

**SATISFIES:** [scenario-name]
(BDD is always on: name the acceptance scenario in
`.skillgrid/specs/<id>/acceptance.feature` that this task makes green.
The RED step in Step 1/2 targets this scenario; TDD Evidence references
the scenario name.)

- [ ] **Step 1: Write the failing test**

```python
def test_specific_behavior():
    result = function(input)
    assert result == expected
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pytest tests/path/test.py::test_name -v`
Expected: FAIL with "function not defined"

- [ ] **Step 3: Write minimal implementation**

```python
def function(input):
    return expected
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pytest tests/path/test.py::test_name -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add tests/path/test.py src/path/file.py
git commit -m "feat: add specific feature"
```
````

## No Placeholders

Every step must contain the actual content an engineer needs. These are **plan failures** — never write them:
- "TBD", "TODO", "implement later", "fill in details"
- "Add appropriate error handling" / "add validation" / "handle edge cases"
- "Write tests for the above" (without actual test code)
- "Similar to Task N" (repeat the code — the engineer may be reading tasks out of order)
- Steps that describe what to do without showing how (code blocks required for code steps)
- References to types, functions, or methods not defined in any task

