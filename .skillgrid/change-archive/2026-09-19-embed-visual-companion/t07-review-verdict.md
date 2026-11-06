# T07 review verdict — openapi + docs + MCP round-trip guard

Reviewed: commits 49b3a25 (RED, tests-only) + 8148b5d (final), full diff in t07-review.md, plus live verification (git show, yaml.v3 parse of pre/post files, focused test runs).

## Spec compliance

| Constraint | Result | Evidence |
|---|---|---|
| GET /mnemonic/decisions (project/state/topic; state enum pending\|answered\|superseded\|all) | ✅ | openapi.yaml:2320-2380 — all 3 params, enum, DecisionRow[] 200 w/ example, 400/404/500 via $ref responses |
| POST /mnemonic/decisions/{id}/answer, body {optionId,note}; 200/404/409; write-auth documented like existing write routes | ✅ (plus extras) | openapi.yaml:2382-2455 — DecisionAnswerInput required [optionId]; 200/400/401/404/409/422/500; description documents "Write-gated (Bearer token, SKILLGRID_HTTP_TOKEN)" in the same style as /mnemonic/memories/{id}/status (openapi.yaml:2180-2184). Extras (401/422/answerer) match the implemented handler (decisions.go:155-245) — superset, no conflict |
| GET /prototype/{id...} — 200 text/html, 400 traversal, 404 unknown | ✅ (plus 500) | openapi.yaml:2825-2870 — text/html 200, 400, 404, 500; nosniff + sandboxed-iframe documented. Extra 500 matches route semantics |
| Existing openapi.yaml conventions (components, tags, $ref error responses) | ✅ | tag mnemonic-decisions added (line 28); all error bodies reuse #/components/responses/Bad|Internal; schemas added to components/schemas in alphabetical slot (Decision* between SearchResult/DocChange) |
| Doc: content JSON schema (ALL fields incl. visual) | ✅ | 10-decision-companion.md field table covers question/options/recommended/state/answeredOption/answerNote/updatedBy/visual |
| Doc: topic_key interview/<slug>/<question-id>, visibility=team | ✅ | "topic-key and visibility convention" section; also documents mem_save-has-no-visibility-arg → mem_share widening (correct for frozen surface) |
| Doc: poll loop (dashboard polls list; agent polls mem_search for answered) | ✅ | "The loop" section steps 1-4; also notes FTS reality (agent queries a content word) |
| Doc: re-answer APPENDS A VERSION, latest content carries the second answer (R6, NOT no-op) | ✅ | "Re-answer appends a version … the latest content carries the latest answer" — matches decisions.go:204-207 + memory.UpdateContent version-append; also in openapi.yaml answer description |
| Doc: .skillgrid/prototype/<topic>/<variant>.html throwaway; distinct from .stitch/ and sketch.dir; sandboxed iframe; sketch-skill config note | ✅ | comparison table (companion vs Phase 7 gallery), sandbox="allow-scripts" w/o allow-same-origin, sketch.dir config note for .skillgrid/config.yaml |
| Doc mirrors 09-serve-dashboard.md structure | ✅ | same Quick-path→sections→tables→"Next step" shape |
| TestDecisionRoundTrip: seed → list exactly it → mem_search (same service as MCP) consistent → answer → FRESH mem_search sees state=answered + option | ✅ | decisions_test.go:227-344; searchDecisions (350-369) opens a NEW service+handle per call (real SQLite re-read, no in-memory reuse); SearchOwnerScoped is exactly the call MCP mem_search runs (mcp/tools_memory.go:341); reader = seed owner so private-vs-team visibility is genuinely enforced |
| MCP surface (mem_save/mem_update/mem_search) unchanged | ✅ | git show --stat: only 4 owned files (openapi.yaml, decisions_test.go, openapi_phase8_test.go, doc); mcp pkg untouched |
| No new dependency | ✅ | gopkg.in/yaml.v3 already in go.mod:15; no go.mod/go.sum diff in either commit |

**CLAIM CHECK (pre-existing invalid YAML):**
- (a) VERIFIED — `git show 1086a2b:.../openapi.yaml` fails parse: ruby psych and Go yaml.v3 both report `line 2649: mapping values are not allowed in this context` (the unquoted plain-scalar `description:` in the /prototypes block, visible in the diff context). Note: the report's second claimed error (mis-indented "404" under /prototypes/{id}, "did not find expected key") could NOT be independently reproduced — the parser reports only the first error (line 2649) before the file is repaired; line 2649 is what's proven. The report's phrasing "two parse errors" is thus only half-substantiated, though the substantive claim (file was invalid YAML before this ticket) is true.
- (b) TestPhase8_OpenAPI is a REAL guard — yaml.Unmarshal of the whole doc + duplicate-key check on top-level/paths/schemas maps (openapi_phase8_test.go:424-465); verified it parses the file and would fail on both a parse error and a duplicated path key.
- (c) No existing documented behavior altered — the only 2 deletions in openapi.yaml are the `description:` lines of the two Phase 7 prototype routes, converted to `|-` blocks with IDENTICAL text; all Phase 6/7 paths/schemas untouched (TestPhase6/7_OpenAPI still PASS, run locally).

## Quality findings

`LOW` — report overclaims a second pre-existing parse error (the "did not find expected key" at line ~2819) that the diff doesn't show and the parser doesn't reproduce (only the line-2649 error is real/verifiable) — t07-report.md:56-60 — misattributes a fix to a bug whose existence is unproven; the actual indent of "404" in 1086a2b is correct in the hunks shown — suggested fix: reword report to "one pre-existing parse error (line 2649 plain scalar); the second `|-` conversion was preventive". Non-blocking.
`LOW` — `TestDecisionRoundTrip` seeds the observation via the memory service + `Share` rather than the literal MCP `mem_save` tool entrypoint — decisions_test.go:260-276 — the MCP tool is a thin wrapper whose core is exactly `Save`+`SearchOwnerScoped` (verified at mcp/tools_memory.go:341), so the read-back leg is faithful; a full MCP-tool round-trip would be marginally stronger — suggested fix: optional follow-up asserting via the mcp package tool runner; not required since the service call is the frozen surface's actual read path.
`NIT` — `state` enum in openapi includes `superseded` while the global constraint text listed only `pending|answered|all` — openapi.yaml:2350 — this matches the implemented handler (decisions.go:79 comment: pending|answered|superseded; default all) and the doc, so the openapi is more correct than the brief's shorthand; no action needed.

## TDD evidence

- RED: 49b3a25 is tests-only (decisions_test.go +154, openapi_phase8_test.go +90; no other file) — verified via `git show --stat`. At that commit openapi.yaml contains zero Phase-8 markers (verified: `git show 49b3a25:...openapi.yaml | rg 'mnemonic/decisions|prototype/\{id'` → 0 matches), so TestPhase8_OpenAPI was genuinely RED. The round-trip test itself passed at RED time (routes from T01/T02/T06 already existed) — disclosed in the report; the RED obligation is carried by the openapi guard, which is a legitimate reading of the brief's Step 2.
- GREEN: 8148b5d adds exactly openapi.yaml + doc. Verified locally: `go test -count=1 ./internal/mnemonic/http/ -run 'TestDecisionRoundTrip|TestPhase8_OpenAPI'` → both PASS; `TestPhase6/7_OpenAPI` also PASS (no regression in prior-phase openapi assertions).
- Process clean: conventional commit messages, 2 commits, no source (route/service) changes needed, zone guard respected, report shows focused + package-level test runs and a full-suite ok line.

## Work Unit Evidence check

- Focused test: report shows `go test -count=1 ./internal/mnemonic/http/ -run 'TestDecision' -v` with per-test PASS lines (13 tests incl. TestDecisionRoundTrip 0.28s) — I re-ran the two key tests: PASS.
- Runtime harness (plan: openapi.yaml + docs lint): actual = `TestPhase8_OpenAPI` (real yaml.v3 parse + dup-key check, the docs-lint equivalent is the doc's testable markers — note the brief's TestDecisionsUserManual marker guard was NOT added; the doc's existence is instead pinned by the round-trip test's package + the report's claims). Openapi parse verified independently (yaml.v3 on pre-file: FAIL at line 2649; post-file: OK). No standalone docs-lint run reported, but the openapi guard is the substantive lint.
- Rollback boundary: 2 commits, revert 8148b5d restores pre-ticket openapi (invalid YAML state) + drops doc, revert 49b3a25 drops both tests; boundary is clean and independent.
- No new dependency (yaml.v3 pre-existing in go.mod); MCP surface frozen (mcp pkg untouched in both commits).

## Verdict

APPROVED — all binding constraints met; the load-bearing claims (pre-existing YAML parse error, real parse guard, no behavior change, R6 re-answer semantics, fresh re-query) are independently verified. The one unproven sub-claim (second parse error) is a report wording issue only.
