---
name: vdd-verify
description: >
  Verification phase for VDD - automated tests plus human-in-the-loop validation.
  Integrates with SDD verification workflow.
license: Apache-2.0
metadata:
  author: skillgrid
  version: "1.1"
triggers:
  - "vdd-verify"
  - "verify vdd"
  - "adversarial validate"
tools:
  - file_system
  - execute_command
---

## When to Use

Use this skill when:
- Validating implemented code against specs
- Running the verification loop before adversarial review
- Ensuring test coverage meets requirements
- Checking for regressions
- **Referenced by `/sdd-verify`** command as enhancement

## Core Philosophy

Verification is NOT separate from TDD - it's the confirmation layer that:
- Runs existing project test runners (never reimplements)
- Validates coverage meets threshold
- Checks for hallucinated tests (tests that don't test what they claim)
- Prepares ground for adversarial review

## Execution Process

### 1. Run Project Test Suite

Use the project's existing test infrastructure:

```bash
# Detect and run appropriate test command
if [ -f "package.json" ]; then
  npm test 2>&1 | tee .skillgrid/reports/test-output.log
elif [ -f "Makefile" ] && grep -q "test:" Makefile; then
  make test 2>&1 | tee .skillgrid/reports/test-output.log
elif [ -f "pytest.ini" ] || [ -f "setup.py" ]; then
  python -m pytest 2>&1 | tee .skillgrid/reports/test-output.log
fi
```

### 2. Run Type Check (if applicable)

```bash
# TypeScript
npx tsc --noEmit 2>&1 | tee .skillgrid/reports/type-check.log

# Python
python -m mypy . 2>&1 | tee .skillgrid/reports/type-check.log 2>/dev/null || true
```

### 3. Run Linter

```bash
npm run lint 2>&1 | tee .skillgrid/reports/lint-output.log || true
```

**Note:** Use project's existing lint/test commands, don't reimplement.

### 4. Verify Coverage

Check coverage report:
- Parse coverage output from test run
- Verify meets minimum threshold from `.skillgrid/config.json`
- Flag if coverage dropped from baseline

### 5. Hallucination Detection

Check for hallucinated tests:
- Tests that pass but don't actually test the feature
- Tests with assertions that always pass (e.g., `expect(true).toBe(true)`)
- Missing tests for reported requirements

## Verification Checklist

### Automated Verification

- [ ] **Test Coverage**: Run tests and record pass/fail
- [ ] **Type Safety**: If TypeScript, run type check
- [ ] **Lint**: Run linter for code quality
- [ ] **Build**: Verify code compiles successfully
- [ ] **Docs**: Check documentation exists

### Human-in-the-Loop Validation

- [ ] **Spec Compliance**: Trace requirements to tests
- [ ] **Edge Cases**: Verify boundary conditions tested
- [ ] **Error Paths**: Confirm error handling works
- [ ] **Integration**: Test with existing code

## Integration with TDD Skill

This skill works alongside `skillgrid-tdd`:

- Each sub-issue should have a test
- Red-Green-Refactor loop verified
- Tests through public interfaces only
- **Enhancement**: After TDD cycle, run vdd-verify to confirm

## Integration with SDD Commands

**Referenced by:** `/sdd-verify` - add adversarial review before verification
**When:** After test execution, before human validation