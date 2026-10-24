# Task 3 review — TICKET-03 Forced code index

**Base:** 9dd840f5 · **Head:** 03439ce7  
**Diff:** `.skillgrid/sdd/2026-10-02-mnemonic-project-init/task-3.diff`  
**Constraints checked:** same `svc` for `ResolveProject` + `RunCodeIndex` after `writeBootFile`; index errors in `Errors`; exit 0 when boot succeeded; no ingest; ADR-0012.

**Spec:** PASS  
**Quality:** approved  
**Overall:** APPROVED

---

### Spec Compliance

- ✅ Spec compliant
- ⚠️ Cannot verify from diff: focused `TestInit*` PASS (parent re-verified; not re-run here).

**Checked against brief / TICKET-03:**

| Requirement | Evidence |
|---|---|
| Call index after `writeBootFile` | `projectInit` builds `res` from boot, then `initRunIndex(ctx, svc, dir)` (`init_cmd.go`) |
| Same `svc` — no second `service.New` | `initRunIndex` uses the `svc` argument; `projectInit` no longer blanks `_ = svc` |
| `ResolveProject` + `RunCodeIndex` | Default `initRunIndex`: `svc.ResolveProject(dir)` then `svc.RunCodeIndex(ctx, dir)`; returns `st.FilesIndexed` |
| `Indexed = FilesIndexed` on success | `res.Indexed = indexed` when `indexErr == nil` |
| Index errors → `Errors`, still `err == nil` | `append(..., "index failed: %v", …)`; `return res, nil` |
| Exit 0 when boot succeeded | `runProjectInit` exits 1 only on `projectInit` error; index failure stays nil error → process exit 0 |
| Happy path indexes a `.go` fixture | `TestInitIndexesTheProject` (`Indexed < 1` fails) |
| Non-fatal failure | `TestInitIndexFailureIsNonFatal` stubs `initRunIndex` → boom; asserts boot exists + `"index failed"` in `Errors` |
| Reuses existing indexer (no second store) | Calls `service.Service.RunCodeIndex` (same pipeline as `skillgrid index` / MCP code tools) |
| No ingest (T04) | `_ = extraDocs` retained; no `ingestPaths` / `Save` / `OpenForDirectory`; `Ingested` untouched |
| ADR-0012 | Indexes into existing mnemonic SQLite via the passed `svc`; no second store / KB / markdown wiki |
| Spec zone already committed; code-only | Diff is `init_cmd.go` + `init_cmd_test.go` only |

**SATISFIES mapping**

| Scenario | Verdict |
|---|---|
| happy path init indexes the project | PASS — `TestInitIndexesTheProject` |
| init reuses the existing indexer | PASS — code path is `ResolveProject` + `RunCodeIndex` on same `svc` (inspection; brief deletion test = shell `skillgrid index`) |
| index failure after boot file still exits 0 | PASS — `TestInitIndexFailureIsNonFatal` + `return res, nil` |

**Missing / Extra / Misunderstood:** none for T03 scope.

Naming note: brief sketched package var `runIndex`; implementer used `initRunIndex` to avoid clash with CLI `runIndex` in `mcp.go`. Behavior matches the brief; rename is justified and documented in the report.

---

### Strengths

- Thin orchestrator: boot then index; error policy matches blueprint (index soft-fail, boot hard-fail).
- Stub seam (`initRunIndex`) keeps the non-fatal test deterministic without chmod/OS races the brief floated.
- Real happy-path test exercises production `RunCodeIndex` on a `.go` fixture (not only a stub).
- Scope held: no ingest, no new deps, no second `Service`.

---

### Issues

#### Critical (Must Fix)

None.

#### Important (Should Fix)

None.

#### Minor (Nice to Have)

1. **Prior `TestInit*` now pay full index cost.** Report already notes slower runs / stderr progress noise. Acceptable for a shared orchestrator; consider a no-op `initRunIndex` stub in boot-only tests later if CI time bites.
2. **Failure path does not assert process exit code via `runProjectInit`.** Unit test pins `err == nil` + `Errors`; exit-0 is implied by `runProjectInit`'s `if err != nil { os.Exit(1) }` contract. Fine for this ticket; an optional CLI-level smoke could lock G6 wording literally.
3. **Commit subject** is `feat(cli): run code index from skillgrid init` vs brief `… run existing code index from skillgrid init` — equivalent intent; not blocking.

---

### Assessment

**Task quality:** Approved

**Reasoning:** Diff implements TICKET-03: after boot, the same `svc` runs `ResolveProject` + `RunCodeIndex`; successes populate `Indexed`; failures append `index failed:…` and still return nil so the CLI exits 0; no ingest and no ADR-0012 violation. Acceptance tests cover both gates; no Critical or Important defects.

**Tests:** Trusted parent re-verification of focused `TestInit*` PASS; no re-run (no specific behavioral doubt from reading the diff).
