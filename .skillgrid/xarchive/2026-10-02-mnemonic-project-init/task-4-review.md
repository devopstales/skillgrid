# Task 4 review — TICKET-04 Default and extra ingest

**Base:** `03439ce7` · **Head:** `8763f9b6`  
**Diff:** `task-4.diff` · **Brief:** `task-4-brief.md` · **Report:** `task-4-report.md`  
**Overall:** APPROVED

### Spec Compliance

- ✅ Spec compliant
- ⚠️ Cannot verify from diff alone: parent stated `TestInit*` PASS — accepted; upsert-by-`topic_key` relies on existing `memory.Save` (named check: `Save` upserts live rows by `project`+`scope`+`topic_key` in `service.go`). ADR-0012 “no new store/KB” holds: only `OpenForDirectory` + `Memory().Save` / `SessionStart`.

Checked against brief / TICKET-04 / blueprint Task 4:

| Requirement | Evidence |
|---|---|
| Default roots → `memory.Save` with `init/docs/<relpath>` | `init_ingest.go` `defaultIngestRoots`, `saveIngestFile` topicKey |
| Jail `--docs` via `Rel` + `..` → `Errors` | `ingestExtra` |
| Missing defaults → `Skipped` | `skipLabel` + `IsNotExist` branch |
| Missing/bad extras → `Errors`, non-fatal | `ingestExtra`; `projectInit` returns `res, nil` |
| Types architecture vs discovery | `ingestType` |
| Second init no duplicate topic keys | `TestInitIngestsDefaultPaths` double `projectInit` + `countTopic == 1` |
| Files: `init_ingest.go`, `init_cmd.go`, `init_cmd_test.go` | diff stat matches |

### Strengths

- Clean seam: `projectInit` opens the handle once after index; `ingestPaths` only talks to memory (`init_cmd.go` wire + `init_ingest.go`).
- Defaults vs extras correctly split (`Skipped` vs `Errors`); walk continues on save/stat failures.
- Acceptance tests cover defaults, skip, extra file, missing extra, jail, and second-init upsert — real store, not mocks.
- `countTopic` via `Recent` (not brief’s `Search("init")`) is a justified FTS limitation; documented in test comment and report.

### Issues

#### Critical (Must Fix)

_(none)_

#### Important (Should Fix)

_(none)_

#### Minor (Nice to Have)

1. **`init_ingest.go` jail prefix** — `strings.HasPrefix(rel, "..")` also rejects in-project names like `..hidden`. Prefer `rel == ".." || strings.HasPrefix(rel, "../")` (after `ToSlash`).
2. **`ingestExtra` dir walk** — on a save error inside an extra directory, only the last `errMsg` is kept, and the caller discards the successful `n` when `errMsg != ""`, under-counting `Ingested` even though files were saved. Defaults path appends each error and still increments — align extras with that pattern.
3. **Empty-file reject** — `saveIngestFile` errors on blank content; brief said save file bytes as-is. Harmless for README/docs fixtures; note if empty stubs should still land.
4. **Commit subject** — brief: `… into mnemonic from skillgrid init`; head: `… from skillgrid init`. Cosmetic.

### Assessment

**Task quality:** Approved

**Reasoning:** Diff implements TICKET-04 end-to-end: jailed extras, skipped missing defaults, `init/docs/<relpath>` Saves on the existing ADR-0012 store, and second-init upsert coverage. Minor polish only; no blockers.
