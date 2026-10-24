# P1 Report — Visual Companion (Mnemonic Decision Bridge): TICKET-01 + TICKET-02

**Status:** DONE
**Branch:** `release/2`
**Commits:** `c3b07ca` (RED tests, TICKET-01), `8bc12bf` (implementation, TICKET-02)

## Scope

Two backend routes over the existing `observations` table — no new table, no
migration, no new store:

- `GET /mnemonic/decisions?project=…&state=pending|answered|all&topic=…` —
  lists `type='decision' AND deleted_at IS NULL` rows (private rows excluded),
  parses the JSON `content` into a typed `decisionContent`, orders ascending by
  `created_at`, flags malformed rows with `parseError` (never a 500).
- `POST /mnemonic/decisions/{id}/answer` (wrapped in `requireWriteAuth`) —
  body `{"optionId": "...", "note": "..."}` (RULING R5 shape); flips
  `content.state` pending→answered, records `answeredOption`/`answerNote`/
  `updatedBy=user:<name>` (name from `?answerer=`, else `unknown`), persists
  through `memory.UpdateContent` (append `observation_versions`, bump
  `revision_count`, set `updated_at`). Idempotent no-op 200 on an already
  answered decision; 404 bad id; 409 missing/invalid `content.state`; 400
  missing `optionId`; 422 optionId not in the decision's options.

Plus `memory.Service.UpdateContent` in `internal/mnemonic/memory/governance.go`
(content-only sibling of `SetStatus`/`SetVisibility`, reusing `AppendVersion`).

## Files changed

| File | Change |
|---|---|
| `skillgrid-cli/internal/mnemonic/http/decisions_test.go` | new (T01), edited in T02 (test bug fix, see Decisions) |
| `skillgrid-cli/internal/mnemonic/http/decisions_stub.go` | new in T01 (forward-declared `decisionRow`/`decisionContent`/`decisionOption` so the RED package compiles), deleted in T02 |
| `skillgrid-cli/internal/mnemonic/http/decisions.go` | new in T02: types, `parseDecisionContent`, `registerDecisionRoutes`, `handleDecisions`, `handleDecisionAnswer` |
| `skillgrid-cli/internal/mnemonic/http/server.go` | +1 line: `s.registerDecisionRoutes()` in `registerRoutes()` before `registerUIRoutes()`. Whole file staged (pre-existing handoff hunks ride along, per plan). `/mnemonic/decisions` NOT added to `apiPrefixes`. |
| `skillgrid-cli/internal/mnemonic/memory/governance.go` | +45 lines: `UpdateContent` method |

No other pre-existing dirty files staged. `.skillgrid/specs/**` untouched.

## TDD evidence

### RED (T01, commit `c3b07ca`)

Command: `cd skillgrid-cli && go test -count=1 ./internal/mnemonic/http/ -run 'TestDecision' -v`

Package **compiled** (forward-declared types via `decisions_stub.go`) and failed
on behavior, not build:

```
--- FAIL: TestDecisionListRoundtrip (0.30s)
    decisions_test.go:115: decode decisions: invalid character '<' looking for beginning of value
--- FAIL: TestDecisionSkipsMalformed (0.24s)
--- FAIL: TestDecisionVisibilityGate (0.24s)
--- FAIL: TestDecisionOrderingAscending (0.22s)
--- FAIL: TestDecisionAnswerRecords (0.24s)
    decisions_test.go:188: answer: got 405: Method Not Allowed
--- FAIL: TestDecisionAnswerAppendsVersion (0.24s)
    decisions_test.go:228: expected >=2 version rows after two answers, got 0
--- FAIL: TestDecisionAnswerIdempotent (0.24s)
    decisions_test.go:240: re-answer should be a 200 no-op, got 405: Method Not Allowed
--- FAIL: TestDecisionAnswer404 (0.24s)
    decisions_test.go:270: bad id should 404, got 405 (Method Not Allowed)
--- FAIL: TestDecisionAnswer409MissingState (0.24s)
    decisions_test.go:281: missing state should 409, got 405 (Method Not Allowed)
--- FAIL: TestDecisionAnswerRequiresAuth (0.25s)
    decisions_test.go:322: no-auth answer with token set: got 405 want 401
--- PASS: TestDecisionUpsertBumpsRevision (0.25s)   ← existing mem_save upsert behavior (frozen contract)
--- FAIL: TestDecisionAnswer400MissingOption (0.25s)
```

Failure modes were exactly "routes not yet registered / content not yet
parsed": GET returned the SPA shell (decode error on `<`), POST returned 405.

### GREEN (T02, commit `8bc12bf`)

Same command after implementation: all 12 `TestDecision*` pass.

## Test results (final verification)

```
$ cd skillgrid-cli && go build ./...
$ go test -count=1 ./internal/mnemonic/http/... ./internal/mnemonic/memory/... ./internal/mnemonic/mcp/...
ok  .../internal/mnemonic/http        33.070s
ok  .../internal/mnemonic/http/docs    4.019s
ok  .../internal/mnemonic/http/tracker 20.360s
ok  .../internal/mnemonic/memory      41.873s
ok  .../internal/mnemonic/memory/layer  5.468s
ok  .../internal/mnemonic/mcp         56.598s
```

MCP surface untouched (mcp package green, no mcp files staged).

## Decisions (smallest-reasonable, per instructions)

1. **`TestDecisionAnswerAppendsVersion` semantics — contradiction resolved
   in favor of the task ruling.** The task's binding constraint says:
   "`answered` → idempotent no-op 200", and `TestDecisionAnswerIdempotent`
   asserts the no-op keeps the FIRST answer. But the blueprint threat matrix
   (line 52) says "A re-answer appends another version (append-only history),
   never overwrites" with `TestAnswerAppendsVersion — answering twice yields
   two observation_versions rows; latest content has the second answer` —
   the exact opposite (re-answer overwrites with the second answer + 2nd
   version row). Both can't hold. I applied the task instruction "the
   blueprint wins over the tasks.md loose names" — i.e. the blueprint's
   R5 idempotency note (an answered decision is a no-op) over the threat
   matrix's looser "re-answer" phrasing. Implemented: first answer flips
   state + appends 1 version; re-answer is a 200 no-op keeping the first
   answer, no second version row. `TestDecisionAnswerAppendsVersion` was
   adjusted accordingly (1 row after first answer; still 1 after re-answer).
   **Flag for the reviewer:** if the threat-matrix reading is the intended
   one (re-answer records the new answer + appends a version), the change is
   one line in `handleDecisionAnswer` (drop the `noOp` early-return) plus
   the idempotent test's assertions.
2. **RED stub shape.** Instead of a separate forward-declaration file with a
   second struct name, `decisions_stub.go` declares the exact production type
   names (`decisionRow`, `decisionContent`, `decisionOption`) plus a
   placeholder `parseDecision` that always returns false. This keeps the test
   file identical between RED and GREEN (no `decisionRow`-type churn) and
   compiles RED purely on behavior. Stub deleted in T02.
3. **`handleDecisionAnswer` option-existence check** returns 422 for an
   optionId not in the decision's options (blueprint sketch behavior). Not in
   the brief's explicit test list; one test covers the 400-missing-optionId
   path. An empty option list accepts any id (a parsed-but-optionless payload
   stays answerable by convention).
4. **`state=superseded` on answer** → 409 ("decision is superseded"). The
   agent owns the pending→superseded transition; the answer gate does not
   apply. Not explicitly in the brief's test list; smallest reasonable
   handling.
5. **List visibility predicate.** `COALESCE(visibility,'') != 'private'` —
   matches the blueprint's intended behavior (team/restricted/agent/NULL
   visible, private excluded) and the dashboard-reader-is-team-scoped
   rationale. The memories list route does not filter by visibility at all
   (it's an admin-style browser); the decisions inbox does, per the threat
   matrix row "Visibility gate".
6. **`updatedBy` format.** `user:<name>` exactly as the binding constraint
   says (the blueprint sketch's bare `"user"` was a divergence it itself
   flagged; R5/constraint wins).

## Concerns

- **`server.go` carries pre-existing handoff hunks** in both commits (it was
  dirty before P1 started and I staged the whole file, per the plan's
  size:exception note). The T01 commit therefore contains unrelated
  handoff diff in `server.go`; the T02 commit adds only the
  `registerDecisionRoutes()` line on top.
- **Test file was edited after the RED commit** (T02 commit): a test-setup bug
  (upsert test used a non-existent session id → FK failure) and the
  appends-version assertion (see Decision 1). The RED evidence above is from
  the as-committed T01 test file; the fixes are in T02.
- No live `skillgrid serve` + curl round-trip was run (the brief's TICKET-02
  acceptance lists it, but the task's binding Verify list is the go test
  commands, all green). The route logic is fully exercised via
  `httptest` against a real seeded store.
- No build after the final commit (repo rule).

## Fix round 1

Controller ruling R6: the blueprint threat matrix wins — a re-answer of an
already-`answered` decision appends another `observation_versions` row and
updates the content to the second answer (append-only history, latest
content = latest answer); `updatedBy` reflects the latest answerer.

**Change made** (`skillgrid-cli/internal/mnemonic/http/decisions.go`):
- `handleDecisionAnswer`: the `decisionStateAnswered` case no longer returns
  an early no-op — it falls through to the same update path as the
  pending→answered flip (state stays `answered`, `answeredOption` /
  `answerNote` / `updatedBy` set to the new values, version appended,
  `revision_count` bumped). 409 for missing/invalid `state` and 404 for a
  bad id are unchanged.
- Response payload: dropped the `noOp` field; success is now always
  `{"ok":true,"state":"answered"}`.
- Doc comment updated to describe append-only re-answer semantics.

**Covering tests** (`skillgrid-cli/internal/mnemonic/http/decisions_test.go`):
- `TestDecisionAnswerAppendsVersion` — two answers → exactly 2
  `observation_versions` rows; latest content carries the second answer
  (note).
- `TestDecisionAnswerIdempotent` — re-answer is a 200 that records the NEW
  answer (option `b`, note `second`, `updatedBy: user:unknown`) and appends
  a version (2 rows).

**Command run:**
```
cd skillgrid-cli && go test -count=1 ./internal/mnemonic/http/... ./internal/mnemonic/memory/...
```

**Output:**
```
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/http	30.710s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/http/docs	1.025s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/http/tracker	19.011s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory	40.969s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory/layer	3.333s
```

`go build ./...` green; all 12 `TestDecision*` pass (RED-verified before the
handler change: both re-answer tests failed on behavior — 1 version row,
first answer kept). Commit: `7cc50f4`
`fix(http): re-answer appends a version instead of no-op (R6)`.
