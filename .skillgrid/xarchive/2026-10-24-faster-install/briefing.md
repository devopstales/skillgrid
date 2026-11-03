---
type: Spec
title: "2026-10-24-faster-install — briefing"
archive: "2026-10-24-faster-install"
status: draft

---

# Change: Parallelize + Batch the skillgrid-cli Install Flow

> **STATUS:** `ready-for-tasks`
>
> **Classification:** standard — touches the install flow's concurrency model, needs careful error handling for partial failure in parallel groups, and changes the npm command shape from N calls to 1 batched call. Not trivial (behavior change to the install run), not risky (no data loss, no trust boundary change, no migration).

## Goal

Cut the `skillgrid install` run time from ~15-20 min (fresh machine) to ~5-7 min by eliminating serial bottlenecks in the tool-install phases. Two mechanisms: (1) **batch all npm installs into a single `npm install -g` call**, and (2) **parallelize the independent install phases** (agents+MCP, security tools, LSP tools, provider setup) with a concurrency-capped `errgroup`.

## Problem

The install flow is **entirely serial** — 20+ `exec.Command` calls run one after another in `Run()`. The bottleneck is the tool-install phases:

| Phase | Commands | Why slow |
|-------|----------|----------|
| `installAgents` | `npm install -g` × N agents | npm is the slowest package manager call; each agent is a separate serial process |
| `installMCPServers` | `npm install -g` × N MCP packages | Same npm tax, serial |
| `installTools` | `npm install -g` × 4 global tools | Same |
| `installSecurityTools` | `go install` × 3 + `uv tool install` × 1 | `go install` compiles from source (akca, nuclei, trivy) — minutes each |
| `installLSPTools` | `go install` (gopls) + `npm` × 4 + `cargo` (rust-analyzer) | Mixed managers, all serial |
| `setupProvider` | Ollama model pull | Can be minutes |

On a fresh machine (no tools installed), the critical path is the sum of all these serial operations. The npm phases alone (14 packages) cost ~6-8 min of process spawn + registry lookup + install. The `go install` phases (akca, nuclei, trivy, gopls) cost another ~8-10 min of compilation.

## Design Decisions

1. **Batch npm installs into one command.** npm supports multiple packages in a single `npm install -g pkg1 pkg2 pkg3` call. Instead of N separate `exec.Command("npm", "install", "-g", pkg)` calls, collect all npm package names across `installAgents`, `installMCPServers`, and `installTools`, then make **one** `npm install -g` call with all of them. This cuts N process spawns + N registry lookups to 1. On a typical install: 14 npm installs → 1.

   **Why not just parallelize npm calls:** npm's global install uses a lock on `node_modules`. Parallel `npm install -g` can race on the lock. Batching avoids the race AND is faster (one process, one lock acquisition).

2. **Parallelize the independent top-level phases.** After repo sync + mirror + git hooks (which must run first), the following phases are independent of each other:

   ```
   Phase A: npm batch (agents + MCP + global tools) → setupAgents → config
   Phase B: installSecurityTools (go/uv, parallel within)
   Phase C: installLSPTools (go/npm/cargo, parallel within) — only if --lsp
   Phase D: setupProvider (ollama) — only if not --skip-provider
   ```

   Phase A's npm batch must complete before `setupAgents` (which reads MCP config and needs binaries on PATH), but A, B, C, D are independent of each other. Use `errgroup` with `SetLimit(4)` to fan them out.

3. **Parallelize within `installSecurityTools` and `installLSPTools`.** The `go install` calls (akca, nuclei, trivy, gopls) are the heaviest single items — each compiles from source. They have no dependencies on each other. Fan them out with `errgroup.SetLimit(3)` (cap to avoid disk thrash on the Go build cache). The `uv tool install` and `cargo install` calls can join the same group.

4. **Error contract unchanged: non-fatal per-tool, fatal for the group's manager.** The existing contract is: if a tool's manager (go, uv, cargo) is missing from PATH, warn and skip that tool. If the install command fails, return an error. In the parallel version: each tool's failure is collected (not returned immediately). At the end, if any tool failed, return a combined error listing all failures. The install still succeeds if the *manager* was missing (skip = success); it fails if the *install command* failed.

   **Partial failure in a parallel group:** if 1 of 4 parallel `go install`s fails, the other 3 complete. The error message lists which tools failed and which succeeded. The install does NOT roll back successful tools (they're on PATH, independent of the failed ones).

5. **Output buffering per-group.** The `run()` helper already captures output per-command (line 526-546 of install.go). In the parallel version, each goroutine buffers its output; the group prints all output on completion (success → verbose only, failure → always). No interleaved chaos in the normal output.

6. **Repo sync stays serial and first.** Everything downstream needs the repo. No change.

7. **`git clone --depth 1`** if full history isn't needed. Check: does any install step use `git log`, `git blame`, or `git diff`? If not, a shallow clone saves the full-history download. (Verify before implementing — if any step needs history, keep the full clone.)

8. **`--skip-lsp` and `--skip-provider` flags already exist.** No new flags needed. The parallelization makes the remaining work faster; the flags let users skip entire phases.

## Steps

| # | Step | Deliverable | Key files |
|---|---|---|---|
| 1 | Extract npm package collection | A `collectNPMPackages(c *Config) []string` function that gathers all npm package names from `GlobalTools()`, MCP server packages, and selected agents. Deduplicates. Respects `--dry-run` (returns packages but doesn't install). | `skillgrid-cli/internal/install/install.go` |
| 2 | Batch npm install | Replace the 3 separate npm-install loops (`installAgents`, `installMCPServers`, `installTools`) with a single `installNPMBatch(c *Config)` that calls `npm install -g <pkg1> <pkg2> ... <pkgN>` once. Keep the per-package skip-if-on-PATH check (batch only the packages that need installing). The `--allow-scripts` flag for packages with native postinstalls is preserved per-package in the batch args. | `skillgrid-cli/internal/install/install.go` |
| 3 | Parallelize `installSecurityTools` | Add `errgroup.WithContext` + `SetLimit(3)`. Each `ExternalTool` in `SecurityTools()` runs in its own goroutine. Collect errors. Return combined error if any failed. Output: buffer per-tool, print on group completion. | `skillgrid-cli/internal/install/install.go` |
| 4 | Parallelize `installLSPTools` | Same `errgroup` pattern as step 3. Each LSP tool (go install, npm, cargo) runs in its own goroutine. The npm LSP tools (pyright, cclsp, typescript, jj-language-server) join the npm batch from step 2 if the batch hasn't run yet, OR get their own single `npm install -g` call in the group (simpler: separate npm call, since LSP is `--lsp`-gated and the main npm batch already ran). | `skillgrid-cli/internal/install/lsp.go` |
| 5 | Fan out top-level phases in `Run` | After `selectAgents`, replace the serial `if len(c.Agents) > 0 { ... }` + `if c.SkipTools { ... }` + `setupProvider` block with an `errgroup` that fans out Phase A (npm batch + setupAgents + config), Phase B (security tools), Phase C (LSP, if --lsp), Phase D (provider, if not --skip-provider). `SetLimit(4)`. `copyAgents` stays serial after the group (it needs the repo mirror, which is done). | `skillgrid-cli/internal/install/install.go` |
| 6 | Error aggregation | A `combinedError` helper that joins per-tool errors into a single message: `"install failed: [akca: <err>], [trivy: <err>] (succeeded: wapiti3, nuclei)"`. Printed to stderr. The function returns the combined error (install is non-zero exit). | `skillgrid-cli/internal/install/install.go` |
| 7 | Tests | `TestInstallNPMBatch`: verify the npm command shape (single call, all packages in args, `--allow-scripts` preserved). `TestInstallSecurityToolsParallel`: verify parallel execution (mock `run`, assert all tools called, error aggregation on partial failure). `TestRunPhasesParallel`: verify the top-level fan-out (mock all phase functions, assert they run concurrently, error propagation). | `skillgrid-cli/internal/install/install_test.go` |
| 8 | Verify `git clone --depth 1` | Check if any install step uses `git log`/`git blame`/`git diff`. If not, add `--depth 1` to the clone in `syncRepo`. If yes, keep full clone and note the decision. | `skillgrid-cli/internal/install/install.go` |

**Zone order:** All steps are code zone. No spec-zone changes (no config, no skill docs).

## Trust Boundaries

None new. The parallelization doesn't change what the install does — it changes the order and concurrency. The same packages are installed, with the same flags, to the same locations. The only behavior change: npm installs are batched (one process instead of N), which changes the lock acquisition pattern but not the end state.

## Threat Matrix

| # | Threat | Applicable | Mitigation |
|---|--------|-----------|------------|
| 1 | npm lock race on parallel `npm install -g` | Yes | Batched into one call (step 2) — no parallel npm. LSP npm tools get their own single call in the group (not parallel with the main batch). |
| 2 | Go build cache thrash on parallel `go install` | Yes | `SetLimit(3)` caps concurrency. Go's build cache handles concurrent access (content-addressed). |
| 3 | Partial failure: 1 of 4 parallel installs fails | Yes | Combined error (step 6). Successful tools stay installed. Failed tools get the manual hint. No rollback. |
| 4 | Output interleaving in parallel phases | Yes | Per-goroutine output buffering (decision 5). Printed on group completion. |
| 5 | `setupAgents` runs before npm batch completes | Yes | Phase A is ordered internally: npm batch → setupAgents → config. The errgroup only parallelizes A, B, C, D — not the steps within A. |
| 6 | Ollama pull runs concurrently with npm installs (disk I/O) | Low | Ollama writes to its own data dir, npm to `node_modules`. Minimal contention. `SetLimit(4)` caps total concurrency. |

## Verification Floor

L2 (build + unit test). The changes are concurrency + command-shape. The tests verify:
- npm batch command shape (correct args, `--allow-scripts` preserved, dedup)
- parallel execution (all tools called, error aggregation)
- top-level phase fan-out (concurrent execution, error propagation)
- `--dry-run` still works (no commands executed)

No integration test (the install is a one-shot machine-level operation, not a service).

## Rollback

- **Code:** revert the 3 Go file changes. The install flow reverts to serial.
- **No data migration.** The install is idempotent (skip-if-on-PATH). A rollback + re-run produces the same end state.

## Open Questions

1. **`git clone --depth 1`:** needs verification (step 8). If any step uses git history, keep full clone.
2. **npm `--allow-scripts` in batch:** when multiple packages in a batch need `--allow-scripts`, does npm accept multiple `--allow-scripts` flags? (Check: `npm install -g pkg1 --allow-scripts=pkg1 pkg2 --allow-scripts=pkg2`). If not, the batch may need to split into 2 calls: one for packages with scripts, one without. Verify before implementing.
