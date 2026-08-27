# Tasks: Testing Enforcement

## Epic 1: Test layer model

- [ ] 1-1 Define the six test layers (L0-L5) with roles and workflow mappings
- [ ] 1-2 Define the change type × test layer matrix
- [ ] 1-3 Document the tool reference matrix per stack (Go, Node, Python)

## Epic 2: Project manifest

- [ ] 2-1 Create `docs/testing-capabilities.yaml` template
- [ ] 2-2 Define schema: stack, strict_tails, layers (unit/integration/acceptance/e2e), gates
- [ ] 2-3 Document agent and CI consumption of the manifest

## Epic 3: Enforcement mechanisms

- [ ] 3-1 Document enforcement layers (skills, hooks, CI)
- [ ] 3-2 Define strict TDD enforcement checklist
- [ ] 3-3 Define CI pipeline order for BDD-enabled projects

## Epic 4: Documentation

- [ ] 4-1 Update `02-usage.md` with testing enforcement section
- [ ] 4-2 Update `test-driven-development` skill to reference the manifest
- [ ] 4-3 Update `bdd-workflow` skill to reference the manifest

## Epic 5: Validation

- [ ] 5-1 Run `openspec validate testing-enforcement --type change --strict` before archive
- [ ] 5-2 Archive via `openspec archive testing-enforcement`
