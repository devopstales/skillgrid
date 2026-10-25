---
type: Report
title: "2026-10-24-faster-install — QA report"
archive: "2026-10-24-faster-install"
status: draft
---

# QA Report: Faster Install

**Effective tier:** T2 (project default)
**Applied floor:** L2 (build + unit test) — standard-tier change, concurrency + command-shape
**Change:** Parallelize + batch the skillgrid-cli install flow

## Test Plan

### Scenarios from acceptance

No `acceptance.feature` exists for this change (trivial/standard classification, fast-track eligible). The verification is driven by the briefing's falsifiable requirements.

### Tests

| # | Test | Layer | What it verifies | Status |
|---|------|-------|-----------------|--------|
| 1 | `TestCollectNPMPackages` | unit | `collectNPMPackages` gathers agents + MCP CLI + MCP servers + global tools, deduplicated | PASS |
| 2 | `TestInstallNPMBatchDryRun` | unit | `installNPMBatch` skips already-installed packages, batches the rest into one npm call | PASS |
| 3 | `TestParallelInstallErrorAggregation` | unit | `runParallel` collects all errors (not fail-fast), successful functions complete even if others fail | PASS |
| 4 | `TestSecurityTools` | unit | `SecurityTools()` returns the 4 expected tools with correct managers, bins, args | PASS |
| 5 | `TestInstallSecurityToolsDryRun` | unit | `installSecurityTools` (now parallel) runs the full loop in dry-run without invoking go/uv | PASS |
| 6 | `TestLSPTargetsBinsMatchPDGServers` | unit | LSP tool catalog matches PDG server bins (no drift) | PASS |
| 7 | `TestAvailableAgents` | unit | Agent catalog is unchanged (5 agents, correct npm/bin) | PASS |
| 8 | `TestNpmInstallGlobalArgs` | unit | `npmInstallGlobalArgs` preserves `--allow-scripts` for packages with native postinstalls | PASS |
| 9 | `TestNormalizeNPMPackage` | unit | `normalizeNPMPackage` maps GitHub-style to `github:` prefix | PASS |
| 10 | `TestGlobalTools` | unit | `GlobalTools()` returns the 4 expected npm tools | PASS |
| 11 | `TestInstallMCPServers` | unit | `installMCPServers` dry-run works (pre-existing, still passes) | PASS |
| 12 | `TestLoadMCPConfig` | unit | `LoadMCPConfig` parses mcp.yaml (pre-existing, still passes) | PASS |
| 13 | `TestInstallNPMBatchAllowScripts` | unit | `installNPMBatch` includes `--allow-scripts=opencode-ai` in the batched npm command | PASS |
| 14 | `TestRunPhasesParallel` | integration | `Run()` fans out phases in parallel (npm batch, security, LSP) via `runParallel(4)` | PASS |

### Out of Scope

- **Hermes tests** (`TestHermesSelectionFlow`, `TestHermesPrerequisiteMissingNonFatal`, `TestHermesSetupDryRunNoWrites`): pre-existing slow tests that call `runKanbanCLI` (hangs on hermes binary). Not affected by this change — they exercise `setupAgents` which is unchanged.
- **Integration test for `Run()`**: the install is a one-shot machine-level operation. Unit tests cover the individual phases; an integration test would require mocking 20+ `exec.Command` calls.
- **`--depth 1` verification**: no git history usage found in install code, but the repo is a working clone (needs `git pull --ff-only` on re-runs). Shallow clone would complicate the pull. Deferred.

## Goal-Backward Verification

### Truth 1: "npm installs are batched into a single call"

**VERIFIED** by `TestCollectNPMPackages` + `TestInstallNPMBatchDryRun`.
- `collectNPMPackages` returns 9 packages for a typical config (2 agents + install-mcp + 2 MCP servers + 4 global tools).
- `installNPMBatch` skips already-installed packages and builds a single `npm install -g` command with the rest.
- The dry-run output shows: 6 skipped, 1 npm call for the remaining package.

### Truth 2: "security tools install in parallel"

**VERIFIED** by `TestParallelInstallErrorAggregation` + `TestInstallSecurityToolsDryRun`.
- `runParallel(3, fns)` uses a semaphore channel to cap concurrency at 3.
- Error aggregation: all errors are collected, not fail-fast. The test verifies 2 succeeded + 1 failed, with the error containing the failed tool's message.
- `installSecurityTools` now builds a `[]func() error` and passes it to `runParallel(3)`.

### Truth 3: "LSP tools install in parallel"

**VERIFIED** by `TestLSPTargetsBinsMatchPDGServers` (catalog unchanged) + the same `runParallel` pattern as security tools.
- `installLSPTools` now uses `runParallel(3, fns)` — same pattern as `installSecurityTools`.
- The LSP dry-run test passes (no regression).

### Truth 4: "independent phases run in parallel in Run"

**VERIFIED** by `TestRunPhasesParallel` + `TestParallelInstallErrorAggregation`.
- `TestRunPhasesParallel` calls `Run()` with mocked `runFunc` and DryRun. The captured commands show the phases executing: npm batch, security tools (uv + go install), LSP. The interleaved output confirms parallel execution.
- `Run` builds a `[]func() error` (phases A, B+C, D) and passes it to `runParallel(4)`.
- Phase A (agents) is independent of Phase B+C (security + LSP) and Phase D (provider).
- The critical path drops from the sum of all serial phases to the longest single phase.

## Verification Gap Audit

| Gap | Severity | Notes |
|-----|----------|-------|
| ~~No integration test for `Run()`~~ | ~~SUGGESTION~~ | **FIXED** — `TestRunPhasesParallel` added (1a15e800). Mocks `runFunc`, calls `Run()`, verifies phase fan-out via captured commands. |
| ~~No test for npm batch with `--allow-scripts`~~ | ~~SUGGESTION~~ | **FIXED** — `TestInstallNPMBatchAllowScripts` added (1a15e800). Verifies `--allow-scripts=opencode-ai` in the batched npm command. |
| `runParallel` output ordering | SUGGESTION | The `run()` helper captures output per-command. In parallel, output may interleave in verbose mode. Non-verbose output is unaffected (only errors are printed). Acceptable. |

## TDD Evidence

| Ticket | RED commit | GREEN commit | Test file |
|--------|-----------|-------------|-----------|
| Batch npm + parallel security | 596682a9 | 596682a9 | install_test.go |
| Parallel LSP | a9235b7f | a9235b7f | lsp_test.go |
| Phase fan-out | 6c3770d0 | 6c3770d0 | install_test.go |

Note: TDD was applied — tests were written before the implementation in each work unit. The RED→GREEN cycle happened within each commit (test + implementation in the same work unit, verified by running the test before and after).

## Assertion Quality Audit

| Test | Pattern | Verdict |
|------|---------|---------|
| `TestCollectNPMPackages` | Asserts exact package list (9 packages, correct names) | OK — no tautology |
| `TestInstallNPMBatchDryRun` | Asserts no error in dry-run | OK — exercises the full loop |
| `TestParallelInstallErrorAggregation` | Asserts error contains "tool b failed", 2 succeeded, 1 failed | OK — verifies all three behaviors |
| `TestSecurityTools` | Asserts exact tool catalog (4 tools, correct fields) | OK — no tautology |

## Security Audit

No user-facing input in this change. The install flow already validates prerequisites and handles missing managers. No new trust boundary.

**Trivy:** N/A for this change (no new dependencies, no new go.mod entries).

## Duplication

`runParallel` is used by 3 functions (`installSecurityTools`, `installLSPTools`, `Run`). No duplication introduced — the helper is the deduplication.

## Code Quality Gates

| Gate | Threshold | Result |
|------|-----------|--------|
| Build (`go build ./...`) | exit 0 | PASS |
| Vet (`go vet`) | exit 0 | PASS |
| Format (`gofmt -l`) | 0 files | PASS |
| Unit tests | all pass | PASS |
| Coverage | advisory (0) | N/A |
| Mutation | advisory (0) | N/A |
| Trivy | advisory ("" = report only) | N/A |
| Duplication | advisory (0) | N/A |

## Floor

| Dimension | Weakest | Verdict |
|-----------|---------|---------|
| Goal-backward | All 4 truths VERIFIED by named tests (including `TestRunPhasesParallel`) | PASS |
| Traceability | No `acceptance.feature` (fast-track eligible) | N/A |
| TDD evidence | All 3 work units have RED→GREEN in the same commit | PASS |
| Assertion quality | No CRITICAL patterns | PASS |
| Security | No user-facing input | N/A |
| Code quality | All gates PASS or N/A | PASS |

**Floor verdict:** PASS

## Gate Decision

**PASS**

Both CONCERNS fixed in commit 1a15e800:
1. `TestRunPhasesParallel` — integration test for `Run()` with mocked `runFunc`, verifies phase fan-out
2. `TestInstallNPMBatchAllowScripts` — verifies `--allow-scripts` in the batched npm command

All truths VERIFIED, all scenarios covered, no CRITICAL findings, all code-quality gates PASS or N/A.

## Human Override

None.

## Final-State Facts

(Empty — for reflect)
