# [Feature Name] Implementation Plan

> Copy this template to `.skillgrid/specs/YYYY-MM-DD-<feature-name>/blueprint.md`.
> Assume the engineer has zero context for the codebase. Document every
> file to touch, the code, the tests, and how to verify. DRY. YAGNI. TDD.
> Frequent commits. No placeholders. (User preferences for plan location
> override this default.)

> **For agentic workers:** REQUIRED SUB-SKILL: Use subagent-execution (recommended) or simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** [One sentence describing what this builds]

**Architecture:** [2-3 sentences about approach]

**Tech Stack:** [Key technologies/libraries]

**Spec:** [path to the spec this plan implements — the plan argues from the spec, so it travels with it; e.g., `../briefing.md` or `.skillgrid/specs/YYYY-MM-DD-<topic>/briefing.md`]

## Must-Haves (goal-backward verification)

[Derived from the spec's requirements. These are the observable outcomes that
MUST be true when the plan is complete. The verifier checks the result against
this list, not just that tasks exist. Every item must be verifiable.]

**Truths** (observable behaviors that must hold):
- [e.g., "Calling `createOrder()` with valid input returns an order with status `pending`"]
- [e.g., "The `/api/orders` endpoint returns 404 for a non-existent order id"]

**Artifacts** (files that must exist with real implementation, not stubs):
- [`src/orders/create.ts` — order creation logic with validation]
- [`src/orders/repository.ts` — persistence layer]
- [`tests/orders/create.test.ts` — passing tests for the truths above]

**Key links** (critical connections between artifacts that must work together):
- [`createOrder()` must call `repository.save()`, and the returned order must be retrievable via `repository.findById()`]
- [The API handler must pass the validated DTO to `createOrder()`, not raw request body]

**One-way-door decisions** (hard to reverse — flag for explicit user approval before implementing):
- [e.g., "Adding a NOT NULL column to the `orders` table — requires a migration, cannot be undone without data loss"]
- [e.g., "Changing the public `Order` API response shape — breaks existing consumers"]
- [If none: "None"]

## Global Constraints

[The spec's project-wide requirements — version floors, dependency limits,
naming and copy rules, platform requirements — one line each, with exact
values copied verbatim from the spec. Every task's requirements implicitly
include this section.]

## File Structure

[List the files that will be created or modified and what each is
responsible for. Design units with clear boundaries and one clear
responsibility. Files that change together live together.]

- `path/to/file.py` — [responsibility]
- `path/to/other.py` — [responsibility]
- `tests/path/to/test.py` — [tests for ...]

---

### Task 1: [Component Name]

> **Checkpoint:** [If this task implements a one-way-door decision, tag it
> here: "⚠ one-way: <decision>" and STOP to get explicit user approval before
> proceeding. If not, omit this line.]

**Files:**
- Create: `exact/path/to/file.py`
- Modify: `exact/path/to/existing.py:123-145`
- Test: `tests/exact/path/to/test.py`

**Interfaces:**
- Consumes: [what this task uses from earlier tasks — exact signatures]
- Produces: [what later tasks rely on — exact function names, parameter and return types]
- Seam: [where this module's interface lives — the place behaviour can be altered without editing that place. "none (in-process)" if there is no external seam]
- Deletion test: [if this module is deleted, what complexity reappears across which callers — name the callers. "pass-through, delete it" if none]
- Adapters: [concrete things that satisfy the interface at the seam. Count them: ≥2 (e.g. prod + test) to justify the seam; 1 = a hypothetical seam, don't build the port yet]

**SATISFIES:** [scenario-name] (the acceptance scenario in `acceptance.feature` this task makes green — BDD is always on)

- [ ] **Step 1: Write the failing test**

```python
def test_specific_behavior():
    result = function(input)
    assert result == expected
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pytest tests/path/to/test.py::test_name -v`
Expected: FAIL with "function not defined"
(BDD is always on: also confirm the `[scenario-name]` scenario in `acceptance.feature` is RED — `npx cucumber-js --dry-run` shows it, the suite run reports it pending/failing.)

- [ ] **Step 3: Write minimal implementation**

```python
def function(input):
    return expected
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pytest tests/path/to/test.py::test_name -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add tests/path/to/test.py src/path/to/file.py
git commit -m "feat: add specific feature"
```

### Task 2: [Component Name]

**Files:**
- Create: `exact/path/to/file.py`
- Test: `tests/exact/path/to/test.py`

**Interfaces:**
- Consumes: [from Task 1]
- Produces: [for later tasks]
- Seam: [where this module's interface lives, or "none (in-process)"]
- Deletion test: [complexity that reappears across which callers if deleted, or "pass-through, delete it"]
- Adapters: [count; ≥2 to justify the seam]

**SATISFIES:** [scenario-name]

- [ ] **Step 1: Write the failing test**
- [ ] **Step 2: Run test to verify it fails** (RED per scenario)
- [ ] **Step 3: Write minimal implementation**
- [ ] **Step 4: Run test to verify it passes**
- [ ] **Step 5: Commit**
