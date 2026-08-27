## Why

Skillgrid supports multiple workflows (SDD, IDD, BDD, strict TDD) but has no unified definition of which test layers apply to which change type, which tools run each layer, or what evidence is required before promote/merge. This leads to inconsistent testing practices across projects and agents skipping verification steps.

## What Changes

- Define test layers (L0 static through L5 quality gates) and their role per workflow
- Define tools per stack (Go, Node, Python) for each layer
- Create `docs/testing-capabilities.yaml` project manifest for declaring runners/commands/thresholds
- Define enforcement mechanisms (skills, hooks, CI pipeline order)
- Define strict TDD enforcement checklist

## Capabilities

### New Capabilities

- `static-layer`: L0 — lint, format, typecheck on every commit/CI
- `unit-layer`: L1 — Mockist TDD, one behavior per cycle
- `integration-layer`: L2 — modules + real collaborators
- `acceptance-layer`: L3 — macro BDD from Gherkin scenarios
- `e2e-layer`: L4 — browser/full stack for UI-heavy features
- `quality-gates`: L5 — branch coverage, mutation, duplication

### Modified Capabilities

None — testing enforcement is a new cross-cutting concern.

## Impact

- **Affected docs**: New `docs/testing-capabilities.yaml` template, updates to `02-usage.md`
- **Affected skills**: `test-driven-development`, `verification-before-completion`, `bdd-workflow` gain reference to the manifest
- **Affected CI**: Projects adopt the CI pipeline order defined here
- **Users**: All skillgrid projects with testing requirements
