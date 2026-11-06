---
type: Review
title: "2026-10-24-faster-install — code review"
archive: "2026-10-24-faster-install"
status: complete
---

# Code Review: Faster Install

**Diff range:** `e1b65e52..3ded35df` (scoped to `skillgrid-cli/internal/install/`)
**Independence grade:** A (fresh subagent contexts, no shared history with implementer)

## Standards

**Grade:** A
**Assessment:** Standards met with fixes

### Fixed (in 3ded35df)
- **S3** `installNPMBatch` re-inlined `--allow-scripts` switch → now uses `npmInstallGlobalArgs`
- **S4/P1** `npmBinForPkg` hardcoded switch → now derives from `AvailableAgents()` + `GlobalTools()`
- **S5** `Run` doc comment stale → updated to reflect parallel fan-out
- **S6** `runFunc` → renamed to `commandRunner`

### Deferred
- **S7** `collectNPMPackages` silently swallows `LoadToolsConfig` error (minor, pre-existing pattern)
- **S8** `npmBinForPkg` returns `"backlog"` but `GlobalTools()` says `Bin: "backlog.md"` (needs decision)
- **S9** `fn := fn` / `t := t` pre-Go-1.22 hygiene (noise, not wrong)
- **S10** Old `installAgents`/`installInstallMcp`/`installMCPServers`/`installTools` now dead code (separate cleanup)

## Spec

**Grade:** A
**Assessment:** Spec met with fixes

### Fixed (in 3ded35df)
- **P1** `npmBinForPkg` hardcoded → skip-if-installed degrades for MCP packages (behavior regression) → derived from catalogs
- **P3** `TestRunPhasesParallel` didn't assert npm batch → now verifies no phase-level error + config file survived

### Deferred
- **P2** Combined error omits succeeded list (error message shape, minor)
- **P4** B+C folded into one goroutine (deviation from 4 independent phases, works, costs no wall time)

## Security

**Grade:** A
**Assessment:** 1 Medium, 2 Low. Nothing exploitable.

### Fixed (in 3ded35df)
- **C1** `runParallel` panic/defer ordering: panic leaks semaphore token → defer order fixed (sem released before `wg.Done()`)

### Deferred
- **C2** Concurrent `Out()` interleaving on stderr (cosmetic, acknowledged)
- **C3** `commandRunner` package-global is a test hazard under `t.Parallel()` (latent, no current tests use `t.Parallel()`)

### Non-Issues (checked and cleared)
- Command injection via npm package names: `exec.Command` uses arg arrays, no shell — non-issue
- `--allow-scripts` with `=` in package name: single argv element — non-issue
- `RepoURL`/`Branch` flag injection: pre-existing, same trust domain — non-issue
- `runParallel` semaphore pattern: correct, race-clean — non-issue

## Floor Verdict

**PASS** — all Critical and Important findings fixed. Deferred items are Minor or acknowledged non-issues. No open Critical findings.

## What "Important" Means

A finding is Important if it violates a documented standard, introduces a behavior regression, or makes a test pass without asserting the behavior it claims. All Important findings from the three axes were fixed in commit 3ded35df.
