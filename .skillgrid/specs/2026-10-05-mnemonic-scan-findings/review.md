# Review — mnemonic scan findings + dependency graph

> Change: `.skillgrid/specs/2026-10-05-mnemonic-scan-findings/` (moves to `.skillgrid/archive/2026-10-05-mnemonic-scan-findings/` at ship)
> Generated: 2026-10-08T00:00:00Z (parallel-code-review — T2 fan-out)
> Diff range: `f7ac22911e8087f3460558bbdcc1f3c3a55a63c0..dcbc936ad9ba3ef82c3bf9513d08786dc8c7cefb`
> Rigor tier: T2 (Beta) — escalated to `parallel-code-review` fan-out (diff ≈ 3336 insertions / 37 files, well above the 50-line / high-risk threshold)
> Independence: Grade B per axis (fresh subagent context, same model family as the implementer) — see `## Independence`
>
> The durable audit record of the code review. Written in the spec zone next to
> `report.md` (which owns the QA gate). **ship** archives it with the folder;
> **reflect** cites it for lineage. The six review lenses are reported
> **side by side, never merged** — that separation is the point.

## What Important means

- **Critical** — must fix before merge. Security, data loss, broken functionality,
  a `SATISFIES` scenario whose RED evidence is missing or whose code path does not
  match the Given/When/Then. Blocks `ship`.
- **Important** — should fix before merge. Architecture drift from a term or an
  in-force ADR, a hard standard breach, a missing or partial requirement, a
  behavior the spec never asked for (scope creep), or a latent correctness defect
  on a realistic input that the tests cannot see. Unfixed Important issues do not proceed.
- **Minor** — nice to have. Baseline smells, style, optimization, doc polish. Capped
  to 5 listed per axis; the rest collapses into a count.

## Cap the nits

- Max 5 Minor findings listed per axis; the rest collapses into a count.
- A Minor that is actually a Critical/Important dressed as polish is re-labeled.

## Passes

> Six lenses (Standards, Spec, Edge cases, Verification gaps, Security, Performance)
> ran in parallel; **Red team** ran last with their merged findings. Each lens filed
> candidates; the coordinator (me) verified every load-bearing claim at its cited
> `file:line` and rendered exactly one verdict per entry. False findings are dropped
> with the disproving evidence noted. Never merged into one verdict.

### Standards

- **Worst issue (within this axis):** `scanDataDir` falls back to the workspace root (`h.Root()`) instead of the service data dir when `DefaultDataDir()` errors — raw scan artifacts could be written inside the user's repo (latent).
- **Findings:** 0 Critical (the "main.go exit-2" claim was verified FALSE — see below) / 2 Important (scanDataDir fallback; wapiti bin-name mismatch) / 5 Minor listed
- **Verdict:** met-with-fixes

### Spec

- **Worst issue (within this axis):** all 12 BDD scenarios PASS with named tests; the `scanners` parameter is dropped from `Start`/`scan_start` (hardcoded map instead) — a signature-level gap against the briefing's literal interface, not a scenario break.
- **Findings:** 0 Critical / 1 Important (akca scope creep — unrequested "store in mnemonic" section that also uses an unsupported `tool: "akca"`) / 2 Minor (scanners param dropped; trivy parser vuln-leg only, no SBOM leg)
- **Verdict:** met-with-fixes

### Edge cases

- **Worst issue (within this axis):** verified FALSE on the headline claims (`Runtime` import case — it IS lowercased via `importName`; nuclei blank line — `json.Decoder` skips whitespace). One real latent edge: `Ingest` with zero parsed purls builds an empty `IN ()` → SQL syntax error (guarded indirectly by `sbom has no components` returning earlier, so it is a defense-in-depth nit).
- **Findings:** 0 Critical / 0 Important / 4 Minor (negative/float `limit` clamp; `scanID` path-separator in `RawPath`; zero-purl `IN ()`; raw-file-missing on a stale `ok` scan)
- **Verdict:** met

### Verification gaps

- **Worst issue (within this axis):** the MCP scan/dep handlers are tested only at registration + arg-validation, never a successful end-to-end dispatch — a handler that wires the store or data dir wrongly ships undetected. Also: no test reads `findings.severity` from the DB, so a dead `NormalizeSeverity` (the semgrep `ERROR`→`HIGH` case, the reason normalization exists) is unobserved.
- **Findings:** 0 Critical / 2 Important (handler-level dispatch tests; DB-level normalized-severity assertion) / 2 Minor (no test pins the strictened `main.go` unknown-command path; no assertion on `findings.links`)
- **Verdict:** met-with-fixes

### Security

- **Worst issue (within this axis):** no exploitable finding at Critical/High. The scanner `target` is free-form and reaches `exec.CommandContext` (local-first single-user trust model → Low: a compromised MCP client can point the scanner at an arbitrary path/URL). `tool` is PATH-resolved from a 4-key map, NOT an allowlist-validated value (Local/Medium, low reach). `scans.error` stores the scanner's error text, which can echo a path (Low, informational).
- **Findings:** 0 Critical / 0 High / 2 Medium (unbounded exec with no timeout on a free-form target; scanner stderr/error text persisted and re-served) / 1 Low (no tool allowlist at the MCP layer)
- **Verdict:** secure-with-fixes

### Performance

- **Worst issue (within this axis):** the "dep_edges store bom-ref not purl → graph joins never match" finding was verified FALSE as stated (the real defect is the bom-ref-vs-purl *asymmetry* in the parse step, already filed by the coordinator — the graph matches whenever bom-ref == purl, as in the fixture). Real perf items: N+1 per-finding upsert (no tx/batch), per-BFS-node `Affected` query, unbounded `Graph`/`scan List`.
- **Findings:** 0 Critical / 2 Important (N+1 upsert + no transaction; per-BFS-node query) / 5 Minor (unbounded Graph/List, 10k-purl `NOT IN`, no default LIMIT clamp, Status full-table GROUP BY, raw-file no-retention)
- **Verdict:** met-with-fixes

### Red team (ran last, on the merged findings)

- **Worst issue (within this axis):** `storeFindings` partial-failure abort — a mid-loop upsert error returns with the row still `status='ok'`, `finding_count=0`, `finished_at` NULL: a phantom "clean" scan.
- **Findings:** 1 Important (partial-failure abort leaves a phantom ok row) / 4 Minor (cross-tool `LatestDiff` noise; `error` column is a SQLite-keyword-ish name; raw-file unbounded retention; retired purl's stale outgoing edges). Several red-team entries were verified FALSE or duplicate of already-filed findings (see `## Verified false` below).
- **Verdict:** met-with-fixes

## Findings

> Verified, deduplicated, and triaged. Source tags name the contributing specialist(s).
> `MULTI-REVIEWER CONFIRMED` = two or more specialists flagged the same defect.

### Critical

- none.

### Important (fix before merge)

- [Standards+Perf+RedTeam] `mnemonic/internal/dep/sbom.go:83` + `:110` — `parseCycloneDX` keys `dependencies` by purl (preferring `c.BomRef` then `c.Purl`) but writes `dep_edges` with the raw `d.Ref`/`dep` (the SBOM `bom-ref`), which in a real CycloneDX SBOM is a **label**, not a purl. — `Affected` (reverse BFS by `to_purl`) and `Graph` therefore misalign on any SBOM whose `bom-ref` ≠ purl; the graph test only passes because the fixture's `bom-ref` literally equals the purl. — **Fix:** normalize edge endpoints to the same identifier as the node (map `bom-ref`→purl during parse, or key both on purl). MULTI-REVIEWER CONFIRMED (Standards dep_edges-stale + Perf bom-ref + RedTeam asymmetric-id).

- [RedTeam] `mnemonic/internal/scan/service.go:166-208` — `StoreFindings` returns on a mid-loop upsert error with the scans row still `status='ok'` (set in `Start`), `finding_count=0`, `finished_at` NULL — a partial parse that succeeds then fails on row N leaves a phantom "clean" scan. — `scan_status`/`scan_list`/`scan_diff` report a scan that stored 0 findings as if it were clean. — **Fix:** on upsert error, mark the row `partial` (call `markScanPartial`) before returning.

- [Verification] `mnemonic/internal/mcp/tools_scan.go` / `tools_dep.go` — the 12 MCP handlers are tested only at registration + arg-validation; no successful end-to-end dispatch exists, so a handler that wires `scan.New(h.Store(), scanDataDir(h))` wrongly (or the `scanDataDir`→`h.Root()` fallback) ships undetected. — The only path agents actually use to reach the store. — **Fix:** one successful-path dispatch test per handler family (stubbed `Run` for scan; fixture SBOM for `dep_ingest` then read back via `dep_list`/`dep_get`).

- [Verification] `mnemonic/internal/scan/service.go:169` — no test reads `findings.severity` from the DB, so a dead `NormalizeSeverity` (the semgrep `ERROR`→`HIGH` / nuclei-lowercase cases — the whole reason normalization exists) is unobserved; the trivy test fixture is on-scale identity. — The 050 normalized-severity contract silently fails for exactly the off-scale tools. — **Fix:** `TestStoreFindingsNormalizesSeverity` — start a semgrep scan from the `semgrep.json` fixture, `StoreFindings`, assert the stored rows are `HIGH`/`MEDIUM`/`INFO` (never `ERROR`/`WARNING`/`NOTE`).

- [Spec] `.agents/skills/verification/akca/SKILL.md:134` — `akca` got an unrequested "Store results in mnemonic" section that also uses `tool: "akca"`, which is not in `Parse`'s dispatch nor `scannerCommands` (would fail-open to `partial`). — Briefing Req 6 names only trivy/wapiti/nuclei. — **Fix:** drop the akca store section (or wire akca as a real tool).

- [Standards] `mnemonic/internal/scan/service.go:70` — `scannerCommands["wapiti"]` is the bin name `wapiti`, but the installer ships `wapiti3` (`SecurityTools` bin `wapiti3`); `exec.CommandContext(ctx, "wapiti", …)` cannot resolve the installed binary. — Latent (wapiti is not in `SecurityTools`, so it is not installed/invoked today; fail-open hides it) but the wapiti parser is dead at runtime until aligned. — **Fix:** align the map key with the installed bin name (or add a per-tool bin override).

- [Standards] `mnemonic/internal/mcp/tools_scan.go:77-82` — `scanDataDir` falls back to `h.Root()` (the **workspace** dir) when `service.DefaultDataDir()` errors, not the service data dir; raw artifacts could be written under the user's repo. — Latent (UserHomeDir rarely fails) but encodes a wrong mental model and contradicts ADR-0030's "raw under the data dir". — **Fix:** the handle should expose the service data dir (or `scan.New` should receive it from the service), not `h.Root()`.

### Minor (capped to 5 listed per axis; the rest collapses into a count)

- [Perf] `mnemonic/internal/scan/service.go:166` — N+1 per-finding upsert with no transaction/batch (a 5k-finding scan = 5k autocommit round-trips + 5k FTS trigger writes). *(Fix: batch upserts in one tx.)*
- [Perf] `mnemonic/internal/dep/service.go:137` — `Affected` issues one `dep_edges` query per BFS node (O(V) queries, correct but slow on deep graphs). *(Fix: one `WHERE to_purl IN (frontier)` per level.)*
- [Perf] `mnemonic/internal/dep/service.go:181` / `scan/service.go:404` — `Graph` and `scan List` are unbounded (no default LIMIT). *(Fix: default cap.)*
- [Edge] `mnemonic/internal/mcp/tools_scan.go:120` — `limit` from the MCP request is `float`→`int` with no clamp; `limit:-1` renders `LIMIT -1` (query error). *(Fix: clamp to ≥0.)*
- [RedTeam] `mnemonic/internal/mcp/tools_scan.go:187` — `LatestDiff` diffs the two most recent scans **across all tools**; since `dedup_hash` is tool-prefixed, a cross-tool diff reports everything added/removed (pure noise). *(Fix: restrict to one tool, or document the single-tool expectation.)*
- (4 further Minor items — scan `error` column is a SQLite-keyword-ish name; raw-file unbounded retention under the data dir; a retired purl's stale outgoing `dep_edges` survive; trivy parser is vuln-leg only with no SBOM leg; zero-purl `Ingest` builds an empty `IN ()` — see the raw specialist reports, not this file.)

### FYI

- The `scanners` parameter is dropped from `Start`/`scan_start` (a hardcoded `scannerScanTypes` map writes the `scans` column). No scenario asserts a custom value, so it is a signature-level gap, not a behavior break. *(Defer — align the tool surface with the briefing if a custom scanner set is ever needed.)*

## Verified false (dropped — with the disproving evidence)

- **[Standards "Critical"] main.go exit-2 regression:** `git diff` shows the **no-arg** case is unchanged (`len(pos)>0` falls through to the old exit-0 path); the new block only adds `unknown command` + `exit 2` for a *non-empty* unknown positional. The reviewer's "exit 0→2 for no args" is wrong; the only behavior change is a new, untested (but intended) stricter path. Not a regression.
- **[Perf] "dep_edges store bom-ref → graph joins can never match / every affected query silently empty":** the fixture's `bom-ref` **equals** the purl (`pkg:pypi/app@1.0.0`), so the graph matches in the test. The real defect is the bom-ref-vs-purl *asymmetry* on real SBOMs (filed above as an Important), not "never match".
- **[Edge] `Runtime` import not lowercased:** `importName` (`dep/service.go:370`) lowercases via `strings.ToLower` before the `declaredSet`/`seen` lookups — the import set IS lowercase. No bug.
- **[Edge] nuclei "off-map values collapse to INFO, severity under-reported":** nuclei's own severity values (`critical/high/medium/low/info/unknown`) are exactly the `toolSeverityMaps["nuclei"]` keys, and they are lowercase in the real output — on-map. Only a genuinely novel nuclei value would collapse (correct fallback).
- **[Edge] nuclei blank JSONL line breaks the parse:** `json.Decoder` skips leading whitespace between values; a blank line is not an error (a malformed *object* mid-stream is, which `StoreFindings` correctly turns into a `partial` row).
- **[Security] "client-supplied `tool` is any PATH binary":** `tool` selects the args from the 4-key `scannerCommands` map and is the PATH-resolved binary name (`trivy`/`wapiti`/`nuclei`/`semgrep`); only `target` is free-form. Local-first single-user trust model → Low, not High.
- **[Perf] "10k-purl `NOT IN` is a full-table scan, no useful index":** the soft-retire `UPDATE ... NOT IN (…)` runs inside the single `Ingest` transaction and is bounded by the SBOM size; it is an O(V) statement, not a per-call hot path. Acceptable; Minor at most.

## Independence

| Axis | Grade | Rationale |
|------|-------|-----------|
| Standards | B | fresh subagent context, same model family as the implementer |
| Spec | B | fresh subagent context, same model family |
| Edge cases | B | fresh subagent context, same model family |
| Verification gaps | B | fresh subagent context, same model family |
| Security | B | fresh subagent context, same model family |
| Performance | B | fresh subagent context, same model family |
| Red team | B | fresh subagent context; read the merged findings as input (by design) |

**Reading the grade:** **A** = fresh subagent, no shared history (independent). **B** = fresh
context but same model family / toolchain as the implementer (independent, weaker).
**C** = inline self-review or a leaked verdict — diagnostic only. No axis here is C:
each was a fresh subagent that saw only the diff range + its named standards/spec sources
and none of the coordinator's session narrative or the QA gate's prior PASS.

## Verdict

- **Standards:** met-with-fixes
- **Spec:** met-with-fixes
- **Security:** secure-with-fixes
- **Edge / Verification / Perf / Red-team:** met / met-with-fixes
- **Floor (decides):** **met-with-fixes**
- **Worst issue (across all lenses):** `dep_edges`/`dependencies` identifier asymmetry (bom-ref label vs purl) — `Affected`/`Graph` misalign on real CycloneDX SBOMs whose `bom-ref` is not a purl.
- **Unfixed Important count (must be 0 to proceed):** 7 → **fix in the loop below, then re-render to 0.**
