---
name: vdd-converge
description: >
  Convergence detection - determine when code is "zero-slop" by detecting hallucinated critiques.
  Gate for exit from VDD refinement loop before archive.
license: Apache-2.0
metadata:
  author: skillgrid
  version: "1.1"
triggers:
  - "vdd-converge"
  - "convergence check"
  - "zero-slop"
tools:
  - file_system
  - execute_command
---

## When to Use

Use this skill when:
- Checking if VDD cycle should exit
- Detecting when adversary is hallucinating
- Determining "zero-slop" achievement
- **Final verification before archive**
- **Referenced by `/sdd-archive`** as gate check

## Core Philosophy

Zero-slop = adversary forced to invent problems.
- When all critiques are hallucinations, code is robust
- Hallucination threshold configurable in `.skillgrid/config.json`
- More hallucinations = closer to convergence

## Execution Process

### 1. Collect Critiques

Run adversarial review and collect:
- List of critiques from vdd-roast
- Severity levels (HIGH/MEDIUM/LOW)
- Code references (file:line)

### 2. Validate Each Critique

For each critique, verify:
- Does the referenced code exist?
- Is the suggested problem real?
- Would fixing it improve correctness?

**A critique is hallucinated when:**
- Criticizing code that doesn't exist (wrong file/line)
- Suggesting changes to working, tested code without valid reason
- Finding problems that aren't problems
- Repeating previously addressed issues

### 3. Count Hallucinations

```bash
# Hallucination threshold from config (default: 0.7 ratio)
HALLUCINATION_THRESHOLD=0.7

# Calculate hallucination ratio
HALLUCINATIONS=$(count hallucinated critiques)
TOTAL_CRITIQUES=$(count total critiques)
RATIO=$(echo "scale=2; $HALLUCINATIONS / $TOTAL_CRITIQUES" | bc)

if (( $(echo "$RATIO >= $HALLUCINATION_THRESHOLD" | bc -l) )); then
  echo "CONVERGED"
fi
```

### 4. Zero-Slop Signal

The code is zero-slop when:
- Adversary is forced to invent problems
- No legitimate flaws remain
- All edge cases handled
- Tests verify behavior
- Hallucination ratio >= threshold

## Exit Criteria

### Convergence Detected

```
CONVERGENCE DETECTED
- Legitimate flaws: 0
- Hallucinations: N
- Status: ZERO-SLOP ACHIEVED
- Ratio: X%

Recommendation: Exit VDD loop - code is robust.
```

### Not Yet Converged

```
NOT CONVERGED
- Legitimate flaws: N
- Hallucinations: M
- Status: CONTINUE REFINEMENT
- Next: Address high-severity items
```

## Integration with SDD Commands

**Referenced by:** `/sdd-archive` - run `/vdd-converge` before exit
**When:** After final adversarial review
**Exit:** If converged, allow archive. If not, return to apply.