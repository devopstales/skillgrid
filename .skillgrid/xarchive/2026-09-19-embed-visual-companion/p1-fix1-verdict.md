# P1 Fix Round 1 — Re-review Verdict

Scope: `.skillgrid/sdd/2026-09-19-embed-visual-companion/p1-fix1.diff` (commit `7cc50f4`, range `8bc12bf..7cc50f4`). Two files: `skillgrid-cli/internal/mnemonic/http/decisions.go`, `decisions_test.go`. The 5 parked minors from the prior review are out of scope (controller parked; not re-litigated).

## Item 1 — FINDING (R6: re-answer was an idempotent no-op; blueprint threat matrix requires version append)

**Verdict: ADDRESSED**

Handler (`skillgrid-cli/internal/mnemonic/http/decisions.go:201-243`):
- The `switch` on `c.State` collapses to `case decisionStatePending, decisionStateAnswered:` — both fall through to the single update path. The old `decisionStateAnswered` no-op branch (`writeJSON(200, ... "noOp": true); return`) is gone.
- Both paths then: `c.State = answered`, `c.AnsweredOption = in.OptionID`, `c.AnswerNote = in.Note`, `c.UpdatedBy = "user:"+name` → `memory.UpdateContent(...)` → same governed path mem_update uses (appends an `observation_versions` row, bumps `revision_count`/`updated_at`). On re-answer the state stays answered, the new answer overwrites the latest content.
- 404 paths unchanged: unknown id (row scan error, `:192-195`) and `UpdateContent` "not found" (`:236-238`). 409 paths unchanged: malformed/missing-state content (`:196-200`), superseded (`:206-210`), unrecognized state (`:211-214`). 422 for unknown optionId unchanged.

Tests (`decisions_test.go`):
- `TestDecisionAnswerAppendsVersion`: answers twice, asserts `COUNT(*) FROM observation_versions` == **2** (was 1), and re-fetches via `GET /mnemonic/decisions` asserting latest content is `state=answered, answeredOption=a, answerNote=second` — exactly the "TWO version rows, latest carries the second answer" expectation.
- `TestDecisionAnswerIdempotent` (now asserting re-answer-updates): re-answer with option b/second → asserts recorded answer flipped to `b`/`second`, `UpdatedBy == "user:unknown"` reflects the latest answerer, and version count == **2**.

## Item 2 — NEW BREAKAGE (fix diff only)

**none**

- `go build ./...` clean; `go test -count=1 ./internal/mnemonic/http/ -run 'TestDecision'` → ok, all 7 decision tests pass (`Records`, `AppendsVersion`, `Idempotent`, `404`, `409MissingState`, `RequiresAuth`, `400MissingOption`).
- Only other decision tests are the untouched 404/409/auth/400 paths, which the diff leaves intact and which pass.
- The removed `noOp` response key has no other consumers (`rg 'noOp'` over `internal/` matches only `decisions.go`, whose no-op branch is the deleted one).
- Version ordering: `UpdateContent` is the existing append path already exercised by mem_update; both new assertions use COUNT + latest-content checks and pass, so ordering/insertion is intact.
- Diff is confined to the 2 files; no other callers of the answer handler or the `decisionStateAnswered` constant behavior changed.

## Final

FINDING: ADDRESSED
NEW BREAKAGE: none
VERDICT: CLEAN
