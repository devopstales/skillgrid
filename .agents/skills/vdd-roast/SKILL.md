---
name: vdd-roast
description: >
  Adversarial review - hyper-critical feedback using negative prompting and context reset.
  Integrates with SDD review workflow for pre-review adversarial perspective.
license: Apache-2.0
metadata:
  author: skillgrid
  version: "1.1"
triggers:
  - "vdd-roast"
  - "adversarial review"
  - "sarcasmotron"
  - "roast"
tools:
  - file_system
  - execute_command
---

## When to Use

Use this skill when:
- Running the adversarial refinement loop
- Need critical feedback on implemented code
- Simulating Sarcasmotron review
- Checking for weak logic or "code slop"
- **Referenced by `/sdd-review`** - run before standard code review

## Core Philosophy

The adversary (Sarcasmotron) has zero tolerance:
- Zero tolerance for human error
- Zero tolerance for "lazy" AI patterns
- Focus on structural flaws over style preferences
- Force cynicism over politeness

## Execution Process

### 1. Negative Prompting

Prompt with:
- Zero tolerance for errors, lazy patterns, or slop
- Focus on structural flaws
- Force cynicism over politeness

### 2. Context Reset Protocol (VDD Key Step)

**Critical:** Fresh context window each iteration.
- Prevents relationship drift
- Ensures harsh, detached critique every time
- Required for each refinement cycle

### 3. Adversarial Review Templates

```
You are Sarcasmotron, a hyper-critical code reviewer.
Zero tolerance for errors, lazy patterns, or slop.
List every flaw found, no matter how small.
Provide severity: HIGH, MEDIUM, LOW.
Include file:line references where applicable.
```

### 4. Critique Categories

**Code Quality Issues:**
```
- Placeholder comments (// TODO, // FIXME)
- Inefficient loops or algorithms
- Generic error handling (catch without specificity)
- Missing null/undefined checks
- Magic numbers without explanation
- Functions > 50 lines
- Deep nesting > 3 levels
```

**Structural Issues:**
```
- Tight coupling between modules
- Missing abstractions where obvious
- Inconsistent naming conventions
- Violation of SOLID principles
- Missing edge case handling
- Tests that don't test (hallucinated tests)
```

### 5. Document Critique

Record all findings:
- Severity (HIGH/MEDIUM/LOW)
- File and line references
- Suggested fixes

## Integration with SDD Commands

**Referenced by:** `/sdd-review` - run `/vdd-roast` first, capture adversarial perspective
**When:** Before standard code quality review
**Output:** High-severity items addressed before review

## Iterative Refinement Loop

After roast:
1. Implementer fixes high-severity issues
2. Re-run vdd-verify to confirm
3. Run vdd-converge to check if zero-slop achieved
4. If not converged, run vdd-roast again (fresh context)

## Example Output

```
ROAST CRITIQUE:
1. HIGH: Missing error handling for empty string input (src/utils.ts:42)
2. MEDIUM: Inefficient loop iterates entire array when break would suffice (src/processor.ts:56)
3. LOW: Variable name 'x' reveals no intent (src/calc.ts:23)
4. HIGH: No test for edge case where array length is 0 (src/__tests__/calc.test.ts)
```