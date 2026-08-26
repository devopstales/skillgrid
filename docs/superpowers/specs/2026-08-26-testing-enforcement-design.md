# Testing enforcement — design

> **STATUS: DRAFT (2026-08-26)**

**Plan:** [2026-08-26-testing-enforcement.md](../plans/2026-08-26-testing-enforcement.md)

**Related:** [2026-08-26-idd-bdd-design.md](2026-08-26-idd-bdd-design.md), [02-usage.md](../../02-usage.md)

## Summary

Define **what tests to run**, **with which tools**, and **how to enforce them** across SDD, IDD, BDD, and strict TDD workflows. Enforcement is layered: agent skills during apply, project manifest for commands/thresholds, hooks for commit boundaries, CI for gates before merge.

Based on [How TDD and BDD Fit Into SDD](https://intent-driven.dev/blog/2026/08/23/tdd-bdd-spec-driven-development/) — **BDD = macro**, **Mockist TDD = micro**, **SDD/IDD = spec anchor**.

## Goal

After adoption, every opted-in project can answer:

1. Which test layers are **required** for this change type?
2. Which **tool** runs each layer?
3. What **evidence** must exist before promote/merge?

## Test layers

| Layer | Role | Workflow home | Runs when |
|-------|------|---------------|-----------|
| **L0 — Static** | Lint, format, typecheck | All | Every commit / CI |
| **L1 — Unit (micro TDD)** | One behavior, Mockist contracts | TDD during apply | Every behavior change |
| **L2 — Integration** | Modules + real collaborators (DB, HTTP fakes) | TDD / apply | When design crosses boundaries |
| **L3 — Acceptance (macro BDD)** | User-visible scenarios from Gherkin | BDD opt-in | Before promote; RED before apply |
| **L4 — E2E / UI** | Browser or full stack | Optional | UI-heavy features only |
| **L5 — Quality gates** | Branch coverage, mutation, duplication | Strict TDD | Before merge; mutation optional nightly |

```
Spec (SDD/IDD)  →  what must be true (docs/)
L3 BDD macro    →  observable behavior (Cucumber)
L1–L2 TDD micro →  unit/integration (go test, vitest, pytest, …)
L4 E2E          →  browser (Playwright) when scenarios need UI
L5 gates        →  coverage + mutation + jscpd on tests
```

## Test types × workflow (requirements)

| Change type | L0 | L1 Unit | L2 Integration | L3 Acceptance | L4 E2E | L5 Gates |
|-------------|:--:|:-------:|:--------------:|:-------------:|:------:|:--------:|
| Spike | — | — | — | — | — | — |
| Bug fix | ✓ | ✓ repro test | if boundary | — | — | branch coverage on touched pkg |
| SDD feature | ✓ | ✓ strict TDD | if design says | — | — | branch coverage |
| IDD feature | ✓ | ✓ strict TDD | if ADR/design says | — | — | branch coverage |
| IDD + BDD feature | ✓ | ✓ strict TDD | if design says | ✓ RED→GREEN | if UI in Gherkin | branch + gherkin-lint |
| Refactor | ✓ | ✓ keep green | existing suite | existing if BDD | — | mutation on refactored module |

**Promote gate (IDD/BDD):** L1 green + L3 green (when BDD enabled) + `verification-before-completion` + L0 clean.

## Tools by stack

skillgrid installs shared tools under `~/.skillgrid/npm/node_modules/.bin/`. Projects declare their **primary test runner** in `docs/testing-capabilities.yaml` (or project `AGENTS.md`).

### Reference matrix

| Layer | JavaScript / TypeScript | Go (skillgrid-cli) | Python |
|-------|-------------------------|--------------------|--------|
| L0 static | `eslint`, `tsc`, `prettier --check` | `go vet`, `staticcheck`, `gofmt -l` | `ruff check`, `mypy` |
| L1 unit | **vitest** or **jest** | **`go test ./...`** | **pytest** |
| L2 integration | vitest + `@testing-library/*`, supertest | `go test` + `httptest` | pytest + httpx |
| L3 acceptance | **@cucumber/cucumber** + **gherkin-lint** + `extract-gherkin.cjs` | cucumber-js via Node pack in `docs/acceptance-tests/` | pytest-bdd or Node pack |
| L4 E2E | **@playwright/cli** (installed via skillgrid) | — | playwright |
| L5 coverage | vitest `--coverage` (v8/istanbul) | `go test -coverprofile` | `pytest --cov` |
| L5 mutation | **Stryker** | **go-mutesting** or gremlins | **mutmut** |
| L5 test dup | **jscpd** on `**/*.test.*` | — | — |

### skillgrid-owned BDD toolchain

| Tool | Location | Purpose |
|------|----------|---------|
| `extract-gherkin.cjs` | `docs/acceptance-tests/` (from acceptance-test-authoring skill) | `-design.md` → `.feature` |
| `cucumber.cjs` | same | Run acceptance |
| `gherkin-lint` | `npx gherkin-lint` | Lint extracted Gherkin |
| `@cucumber/cucumber` | `config.d/tools.yaml` → skillgrid npm prefix | Cucumber runner |

### Agent-only tools (not merge gates by default)

| Tool | Skill | Use |
|------|-------|-----|
| Playwright | `webapp-testing` | Agent-driven UI exploration during apply |
| agent-browser | tools.yaml | Lightweight browser automation |
| Gryph | planned | Supply-chain / dependency audit — not functional tests |

## Enforcement mechanisms

| Mechanism | Enforces | Owner |
|-----------|----------|-------|
| `test-driven-development` skill | L1 red→green; one behavior per cycle | superpowers |
| `verification-before-completion` skill | Run real commands; no “should pass” | superpowers |
| `bdd-workflow` + acceptance-test-authoring | L3 RED before apply, GREEN before promote | config.d/skills |
| Zone-guard hook | Docs vs code commit separation | config.d/hooks |
| `docs/testing-capabilities.yaml` | Declares runners, commands, thresholds | project |
| Project `AGENTS.md` | Test commands in Definition of Done | project |
| Pre-commit (optional) | L0 + targeted L1 on staged paths | project |
| CI pipeline | Full L0–L5 per change type | project |

## `docs/testing-capabilities.yaml` (project manifest)

Each opted-in project declares capabilities once (detected at init or hand-authored):

```yaml
# docs/testing-capabilities.yaml
stack: go                          # go | node | python
strict_tdd: true                   # require red-before-green evidence

layers:
  unit:
    framework: go test
    command: go test ./...
    targeted: go test ./internal/bdd/... -run TestName
  integration:
    available: true
    command: go test ./... -tags=integration
  acceptance:
    available: true
    runner: cucumber-js
    extract: node docs/acceptance-tests/extract-gherkin.cjs
    lint: npx gherkin-lint docs/acceptance-tests/*.feature
    command: node docs/acceptance-tests/cucumber.cjs
  e2e:
    available: false

gates:
  branch_coverage_min: 80          # on touched packages
  mutation:
    enabled: false                 # true for critical modules
    tool: go-mutesting
    command: go-mutesting ./...
  test_duplication:
    tool: jscpd
    command: npx jscpd '**/*.test.ts' --min-lines 5
```

Agents read this before apply; CI reads the same file for parity.

## CI pipeline order (when BDD enabled)

```bash
# 1. Static
go vet ./... && gofmt -l . | wc -l | xargs test 0 -eq

# 2. Extract + lint acceptance (docs zone artifacts)
node docs/acceptance-tests/extract-gherkin.cjs ...
npx gherkin-lint docs/acceptance-tests/*.feature

# 3. Unit + integration
go test ./... -coverprofile=coverage.out

# 4. Acceptance (macro)
node docs/acceptance-tests/cucumber.cjs

# 5. Coverage gate (branch, touched packages only)
go tool cover -func=coverage.out  # compare to threshold

# 6. Mutation (optional, label: run-mutation)
# go-mutesting ./internal/bdd/...
```

## Strict TDD enforcement checklist

From [intent-driven.dev TDD post](https://intent-driven.dev/blog/2026/08/23/tdd-bdd-spec-driven-development/) — agent Definition of Done per behavior:

- [ ] One failing test written before production code
- [ ] Failure observed for **missing feature** (not typo/error)
- [ ] Minimal code to green; no extra branches
- [ ] Full suite still green
- [ ] Refactor; tests still green
- [ ] One git commit for this vertical slice
- [ ] Assertions pin exact outputs / error types / collaborator args
- [ ] Boundary cases at thresholds included
- [ ] Branch coverage on touched code ≥ project minimum
- [ ] (Optional) Mutation survivors addressed

## Non-goals

- Mandating Playwright for every project
- Mandating mutation testing on every PR (too slow; opt-in per module)
- Replacing superpowers TDD skill with a skillgrid copy
- Python/behave as default (Node cucumber-js pack is reference)

## Success criteria

1. Every workflow in [02-usage.md](../../02-usage.md) maps to concrete test layers and tools.
2. Projects can declare capabilities in one manifest; agents and CI use the same commands.
3. BDD promote blocked unless L3 GREEN; code promote blocked unless L1 GREEN.
4. Strict TDD anti-patterns documented and checkable via commit history + coverage/mutation.

## References

- [02-usage.md](../../02-usage.md) — command orders
- [TDD + BDD + SDD](https://intent-driven.dev/blog/2026/08/23/tdd-bdd-spec-driven-development/)
- [Glossary + adversarial spec review](https://intent-driven.dev/blog/2026/06/27/sdd-adversarial-authoring-glossary/)
- [2026-08-26-idd-bdd-design.md](2026-08-26-idd-bdd-design.md)
