# Testing enforcement — implementation plan

> **For agentic workers:** Use superpowers:subagent-driven-development or executing-plans. Checkbox syntax for tracking.

**Goal:** Enforce the test-layer model (L0–L5) across SDD/IDD/BDD/TDD workflows with a project manifest, AGENTS rules, and CI examples.

**Spec:** [2026-08-26-testing-enforcement-design.md](../specs/2026-08-26-testing-enforcement-design.md)

**Architecture:** Manifest-driven — `docs/testing-capabilities.yaml` declares runners and gates; skills read it; CI mirrors it. BDD tools from skillgrid npm prefix; Go tests for skillgrid-cli itself.

---

## Quick reference — test type × tool

Use this when choosing what to run for a change:

| You are doing… | Test type | Tool (default) | Command example |
|----------------|-----------|----------------|-----------------|
| Any code change | L0 static | go vet / eslint / ruff | `go vet ./...` |
| SDD/IDD apply step | L1 unit (strict TDD) | go test / vitest / pytest | `go test ./internal/bdd/... -run TestFoo` |
| Cross-module feature | L2 integration | httptest / supertest | `go test ./... -tags=integration` |
| IDD + BDD feature | L3 acceptance | cucumber-js + gherkin-lint | `node docs/acceptance-tests/cucumber.cjs` |
| UI in Gherkin | L4 E2E | Playwright | `npx playwright test` |
| Before merge | L5 branch coverage | go cover / vitest --coverage | `go test -coverprofile=c.out ./...` |
| Critical module | L5 mutation | Stryker / go-mutesting | `npx stryker run` |
| Test suite hygiene | L5 duplication | jscpd | `npx jscpd '**/*.test.ts'` |

**Enforcement order during apply:**

```
strict TDD (L1) per plan task  →  integration if needed (L2)  →  Cucumber GREEN (L3)  →  coverage gate (L5)  →  verification-before-completion
```

---

## Task 1: Document test layers in usage guide

**Files:**
- Modify: `docs/02-usage.md`

- [ ] **Step 1:** Add section "Test layers and tools" linking to this spec.
- [ ] **Step 2:** Add one table: workflow × required layers (copy from design spec).
- [ ] **Step 3:** Add pointer to `docs/testing-capabilities.yaml` template.

**Verify:** 02-usage mentions L0–L5 and links to enforcement design.

---

## Task 2: `testing-capabilities.yaml` template

**Files:**
- Create: `config.d/templates/testing-capabilities.yaml` (go, node, python variants)
- Create: `config.d/skills/idd-workflow/references/testing-capabilities.md` (how agents read it)

- [ ] **Step 1:** Author Go template matching skillgrid-cli (`go test`, `-coverprofile`, optional integration tag).
- [ ] **Step 2:** Author Node template (vitest, cucumber paths under `docs/acceptance-tests/`).
- [ ] **Step 3:** Author Python template (pytest, optional pytest-bdd note).
- [ ] **Step 4:** Document `strict_tdd: true` flag and gate thresholds.

**Verify:** Template validates as YAML; documents all L0–L5 commands.

---

## Task 3: AGENTS.md testing block

**Files:**
- Modify: `config.d/AGENTS.md`

- [ ] **Step 1:** Add marker block `<!-- skillgrid:testing -->` with:
  - Read `docs/testing-capabilities.yaml` before apply
  - Iron law: no production code without failing test
  - Run verification-before-completion commands from manifest
- [ ] **Step 2:** Reference BDD promote gate (L3 green when acceptance available).

**Verify:** Re-run install merges block; idempotent append.

---

## Task 4: Wire acceptance-test-authoring to manifest

**Files:**
- Modify: `config.d/skills/acceptance-test-authoring/SKILL.md`
- Modify: `config.d/skills/bdd-workflow/SKILL.md` (when created)

- [ ] **Step 1:** On first BDD scaffold, offer to create `docs/testing-capabilities.yaml` from template.
- [ ] **Step 2:** Read manifest for extract/lint/cucumber commands instead of hardcoded paths only.
- [ ] **Step 3:** Fail promote if `layers.acceptance.available` and cucumber command exits non-zero.

**Verify:** Skill references manifest; cucumber path matches design spec.

---

## Task 5: skillgrid-cli self-test bar

**Files:**
- Existing: `skillgrid-cli/**/*_test.go`

- [ ] **Step 1:** Add `docs/testing-capabilities.yaml` for this repo (stack: go).
- [ ] **Step 2:** Set gates: `branch_coverage_min` on `internal/bdd` when zone-guard lands.
- [ ] **Step 3:** Document in repo `AGENTS.md`: `go test ./...` is L1; no BDD until IDD pack merged.

**Verify:** `go build ./... && go test ./...` in skillgrid-cli.

---

## Task 6: CI example workflows

**Files:**
- Create: `config.d/templates/ci/github-testing.yml`
- Create: `config.d/templates/ci/gitlab-testing.yml`

Pipeline stages (in order):

1. static (L0)
2. extract-gherkin + gherkin-lint (L3 prep, if BDD)
3. unit + integration (L1–L2)
4. cucumber (L3)
5. coverage gate (L5)
6. mutation (L5, optional job on label `run-mutation`)

- [ ] **Step 1:** GitHub Actions template reading commands from env vars matching manifest keys.
- [ ] **Step 2:** GitLab CI equivalent.
- [ ] **Step 3:** Document skip rules: no cucumber job when `layers.acceptance.available: false`.

**Verify:** Dry-run YAML syntax; commands match design spec CI section.

---

## Task 7: Strict TDD enforcement skill fragment

**Files:**
- Create: `config.d/rules/strict-tdd.md` (merged into AGENTS or referenced by idd-workflow)

- [ ] **Step 1:** Copy anti-pattern table from 02-usage (horizontal slicing, fake TDD, weak assertions).
- [ ] **Step 2:** Add git evidence rule: one commit per vertical slice when `strict_tdd: true`.
- [ ] **Step 3:** Add mutation/coverage follow-up when gates enabled in manifest.

**Verify:** Referenced from idd-workflow and 02-usage.

---

## Task 8: Pre-commit hook (optional)

**Files:**
- Create: `config.d/hooks/pre-commit-testing.sh.example`

- [ ] **Step 1:** Run L0 on staged files only.
- [ ] **Step 2:** Run targeted L1 if `*_test.go` or `*.test.ts` staged alongside source.
- [ ] **Step 3:** Do **not** run full cucumber in pre-commit (too slow); CI only.

**Verify:** Example exits 0 on clean tree; documented as opt-in.

---

## Task 9: Integration smoke

**Files:**
- Use: `.tmp-bdd-smoke/` or fixture project

- [ ] **Step 1:** Fixture project with `testing-capabilities.yaml`, one `-design.md`, extract, RED cucumber.
- [ ] **Step 2:** Implement minimal step def + code; GREEN cucumber.
- [ ] **Step 3:** Confirm promote blocked when L3 fails after code change.

**Verify:** Full L1 + L3 loop documented in test log.

---

## Rollout by project type

| Project | Start with | Add later |
|---------|------------|-----------|
| skillgrid-cli (Go) | L0 + L1 strict TDD | L5 coverage on `internal/bdd` |
| New IDD feature | L0 + L1 + manifest | L3 when BDD opted in |
| UI product | L0 + L1 + L3 + L4 Playwright | L5 mutation on checkout module |
| Library / CLI | L0 + L1 + L5 coverage | L3 only if public API scenarios in Gherkin |

---

## Open questions

1. **Mutation in CI:** default off; enable per-module via manifest — confirm threshold for skillgrid-cli.
2. **Commit extracted `.feature` files:** if gitignored, CI must always extract before cucumber (design recommends extract-in-CI).
3. **Engram mode:** store testing-capabilities as Engram observation instead of YAML for some harnesses — out of scope for file-based skillgrid v1.

---

## Manual acceptance

In a clean agent session after Tasks 1–4:

1. Prompt: "Add export with BDD and strict TDD."
2. Agent reads/creates `docs/testing-capabilities.yaml`.
3. Agent runs extract → gherkin-lint → cucumber RED before code.
4. Agent runs targeted unit test RED → GREEN per task.
5. Agent runs cucumber GREEN + `go test ./...` before promote.
6. Agent refuses promote if any command fails.
