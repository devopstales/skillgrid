# TICKET-07 report — openapi + docs + MCP round-trip guard

**Status:** GREEN. `cd skillgrid-cli && go build ./... && go test -count=1 ./internal/mnemonic/http/...` all pass (incl. the new `TestPhase8_OpenAPI` which parses the YAML, and `TestDecisionRoundTrip`).

## Commits

| SHA | Message |
|-----|---------|
| `49b3a25` | `test(http): RED round-trip guard for decision MCP read-back (TICKET-07)` |
| `8148b5d` | `docs(http): openapi + user guide for decision companion (TICKET-07)` |

RED-first: the round-trip test **passed on first run** (the routes from T01/T02 already exist — no route defect to fix), so commit 1 carries the guard tests while `openapi.yaml` still lacked the Phase-8 entries (verified RED — all five markers below are absent from `HEAD:openapi.yaml`):

```
MISSING (RED): /mnemonic/decisions:
MISSING (RED): operationId: mnemonicDecisions
MISSING (RED): operationId: mnemonicDecisionAnswer
MISSING (RED): /prototype/{id...}:
MISSING (RED): DecisionRow
```

Commit 2 adds the entries + doc → GREEN.

## What changed

### 1. `openapi.yaml` (3 routes + 4 schemas + 1 tag)
- **`GET /mnemonic/decisions`** (`operationId: mnemonicDecisions`, tag `mnemonic-decisions`): query params `project` (required), `state` (enum `pending|answered|superseded|all`), `topic` (topic_key prefix). 200 = `DecisionRow[]` (empty array when none match) with a worked example; 400 (bad/missing project), 404 (unknown project), 500.
- **`POST /mnemonic/decisions/{id}/answer`** (`operationId: mnemonicDecisionAnswer`, write-gated): path `id` (string, integer), query `project` (required) + optional `answerer`; body `DecisionAnswerInput {optionId, note}`. 200 `{ok:true, state:"answered"}`, 400 (bad project/id/body, missing optionId), 401 (bearer token), 404 (unknown id), 409 (no valid state / superseded), 422 (optionId not in options), 500. Auth documented as Bearer `SKILLGRID_HTTP_TOKEN`, matching how the existing write routes (pin/share/status) document write-gating.
- **`GET /prototype/{id...}`** (`operationId: prototypeDecision`): path `id` (relative to `.skillgrid/prototype/`). 200 `text/html` (nosniff, sandboxed iframe), 400 (traversal/absolute), 404 (unknown), 500.
- **Schemas:** `DecisionOption`, `DecisionContent` (all fields incl. `visual`), `DecisionRow` (with `parseError`), `DecisionAnswerInput`. Added tag `mnemonic-decisions`.
- Followed file conventions exactly: `$ref`-based error responses (`#/components/responses/Bad|Internal`), `schema: { type: ... }` flow style, `example:` blocks, description `|-` blocks.

### 2. `docs/user-guide/10-decision-companion.md` (new)
Agent-side convention, mirroring `09-serve-dashboard.md` structure (Quick-path → sections → tables → Next step). Covers:
- The decision content JSON schema (all fields incl. `visual`), with a post-answer example.
- The `topic_key: interview/<slug>/<question-id>` convention, `visibility: team` (and that `mem_save` has no visibility arg, so the agent widens via `mem_share`).
- The poll loop (dashboard polls `GET /mnemonic/decisions`; agent polls `mem_search`).
- Answer semantics incl. **re-answer appends a version — latest content carries the second answer**, append-only history; the full 200/400/401/404/409/422 error contract.
- The `.skillgrid/prototype/<topic>/<variant>.html` convention (throwaway, served by `GET /prototype`, sandboxed iframe, distinct from `.stitch/` and `sketch.dir`), with a comparison table and the **config note** to point the `sketch` skill at `.skillgrid/prototype/<topic>/` (`sketch.dir` in `.skillgrid/config.yaml`).
- Durability (SQLite, versioned, owned, governed; survives restart/resume).

### 3. Round-trip guard test (`decisions_test.go`)
`TestDecisionRoundTrip` — seeds a `type=decision`/`state=pending` observation via the real memory service (session FK + `mem_share` to team), then:
1. `GET /mnemonic/decisions?state=pending` lists **exactly** the seeded decision (parsed).
2. Agent-side `mem_search` — via `Service.SearchOwnerScoped` (the **same service the MCP `mem_search` tool runs**) as a fresh service — finds it with a query consistent with the list.
3. `POST /mnemonic/decisions/{id}/answer` `{optionId:"b", note}`.
4. A **FRESH** `mem_search` (new service over the same data dir, simulating the agent polling later in a resumed session) sees `content.state=answered` + `answeredOption=b` + `updatedBy=user:*`.

This proves the agent-side read-back works with the frozen MCP surface.

### 4. Openapi guard test (`openapi_phase8_test.go`, new)
`TestPhase8_OpenAPI` — substring assertions for all Phase-8 paths/operationIds/schemas/state-values/error-codes, **plus** a `yaml.v3` parse of the whole doc and a duplicate-key check on the top-level/`paths`/`schemas` maps. This is the first test that parses the spec (Phase 6/7 only substring-match).

## Route defects found + fixed
- **No functional route defects.** The round-trip test passed on first run — `decisions.go` / `prototype.go` behave as documented (list, parse, answer flip, re-answer version-append, 400/401/404/409/422 semantics, traversal guard).
- **Pre-existing `openapi.yaml` was invalid YAML** (never caught because the Phase 6/7 tests only substring-match). Two parse errors in the Phase 7 block, fixed in the final commit (owned file):
  1. `summary: List .stitch/ …` / `description: Returns … (Accept: text/html),` — a plain multi-line scalar with a `: ` inside → "mapping values are not allowed in this context" (line ~2649). Fixed by converting to a `|-` block.
  2. The `"404"` under `/prototypes/{id}` was indented 10 spaces (rest 9) → "did not find expected key" (line ~2819). Fixed the indent.
  (A third, self-introduced, was an unescaped `: ` in a flow-mapping `description:` and an unquoted `title: Decision: layout-1` example — fixed by quoting during the T07 additions.)
  Verified parse-clean + no duplicate keys via a throwaway `yaml.v3` checker (top-level 6 keys, paths 73, schemas 38, no dupes), now permanently enforced by `TestPhase8_OpenAPI`.

## Test evidence

```
$ cd skillgrid-cli && go build ./... && go test -count=1 ./internal/mnemonic/http/...
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/http        29.850s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/http/docs    1.140s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/http/tracker 17.347s

$ go test -count=1 ./internal/mnemonic/http/ -run 'TestDecision' -v
--- PASS: TestDecisionListRoundtrip
--- PASS: TestDecisionSkipsMalformed
--- PASS: TestDecisionVisibilityGate
--- PASS: TestDecisionOrderingAscending
--- PASS: TestDecisionAnswerRecords
--- PASS: TestDecisionAnswerAppendsVersion
--- PASS: TestDecisionAnswerIdempotent
--- PASS: TestDecisionAnswer404
--- PASS: TestDecisionAnswer409MissingState
--- PASS: TestDecisionAnswerRequiresAuth
--- PASS: TestDecisionUpsertBumpsRevision
--- PASS: TestDecisionAnswer400MissingOption
--- PASS: TestDecisionRoundTrip (0.28s)
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/http  4.446s

$ go test -count=1 ./internal/mnemonic/http/ -run 'TestPhase8_OpenAPI|TestPhase7_OpenAPI|TestPhase6_OpenAPI' -v
--- PASS: TestPhase6_OpenAPI
--- PASS: TestPhase7_OpenAPI
--- PASS: TestPhase8_OpenAPI
```

## Files changed (owned)
- `skillgrid-cli/internal/mnemonic/http/ui/openapi.yaml` (M) — 3 routes + 4 schemas + tag + 2 pre-existing YAML fixes
- `skillgrid-cli/internal/mnemonic/http/decisions_test.go` (M) — `TestDecisionRoundTrip` + `searchDecisions` helper
- `skillgrid-cli/internal/mnemonic/http/openapi_phase8_test.go` (new) — `TestPhase8_OpenAPI`
- `docs/user-guide/10-decision-companion.md` (new)

No route/service source changes were required. Zone guard respected (`.skillgrid/specs/` untouched); not pushed.

## Concerns
- **`mem_search` is FTS-based, not state-based.** The agent poll must query by a word of the question/content (or use `mem_get_observation` by id after locating the row) — `state=pending` is not a search field. The round-trip test uses a content word ("layout") and asserts the list↔search consistency on `topic_key`/id. This is inherent to the frozen MCP surface (no topic/state query param on `mem_search`); documented in the guide.
- **`GET /mnemonic/decisions` is capped at 200 rows** (ascending by created_at) — a very long-running interview could push old decisions out of the default list; `?state=`/`?topic=` narrow it.
