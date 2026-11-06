# QA runtime harness — Visual Companion (decision bridge)

Live `skillgrid serve` + curl round-trip against a real (throwaway) SQLite store.
Covers the runtime-harness column of the work-unit table that httptest/vitest cannot.

## Environment

| Field | Value |
|-------|-------|
| Binary | `/tmp/skillgrid-qa` (`go build -o /tmp/skillgrid-qa ./cmd/skillgrid`, exit 0) |
| Port (open pass) | `18131` — `SKILLGRID_HTTP_TOKEN` unset (loopback open mode) |
| Port (auth pass) | `18132` — `SKILLGRID_HTTP_TOKEN=qa-token-123` |
| Data dir | `/var/folders/g1/c0fkk4s12mv4rqxrypxlndqc0000gq/T/opencode/qa-viz/data` → store `qa.sqlite` (project `qa`) |
| Docs root | `/var/folders/g1/c0fkk4s12mv4rqxrypxlndqc0000gq/T/opencode/qa-viz/docs` (`SKILLGRID_DOCS_CWD`), holds `.skillgrid/prototype/qa/a.html` |
| User data | untouched — `~/.skillgrid/mnemonic` left in place |

## Seed method

The `mem` CLI has **no `save` subcommand** (only layers/governance/share/search/context/timeline/graph/relations/provenance/list/...), so the faithful `mem_save` path is the MCP surface, not the CLI. Seeded via the **same SQLite store the server uses**, with an INSERT matching `decisions_test.go`'s `seedDecision`: a `sessions` row (`s-qa`) plus `observations` rows — one `type=decision`/`visibility=team`/`state=pending` (`interview/qa/demo-1`), one `type=note` (for the non-decision case), and a second `type=decision` with **no** `content.state` (`interview/qa/nostate`) for the 409 case. Store auto-created+migrated via a `mem list --dir` read.

## Step results

| # | Step | Command | Expected | Actual | Result |
|---|------|---------|----------|--------|--------|
| 1 | Build | `go build -o /tmp/skillgrid-qa ./cmd/skillgrid` | exit 0 | exit 0 | PASS |
| 2 | Serve (open) | `serve -port 18131` (token unset) | listening | `skillgrid serve listening on http://127.0.0.1:18131`, `/health` 200 | PASS |
| 3 | List pending | `GET /mnemonic/decisions?project=qa&state=pending` | exactly the seeded decision, content parsed | 1 row, `question`/`options[a,b]`/`recommended=a`/`state=pending` parsed | PASS |
| 4 | Answer | `POST .../1/answer` `{"optionId":"b","note":"qa check"}` | 200 | `{"ok":true,"state":"answered"}` 200 | PASS |
| 5 | Re-list | `GET ...&state=pending` then `GET ...` (all) | pending empty; all shows answered | pending `[]`; all shows `state=answered`,`answeredOption=b`,`answerNote="qa check"`,`updatedBy=user:unknown` | PASS |
| 6 | Re-answer | `POST .../1/answer` `{"optionId":"b","note":"qa re-check"}` | 200 + versions=2 | 200; `observation_versions` rows for obs 1 = 2 (rev 1,2); latest content note=`qa re-check` | PASS |
| 7 | Bad id | `POST .../999/answer` | 404 | 404 `no such decision` | PASS |
| 8 | Non-decision/invalid state | `POST .../3/answer` (type=decision, no state) | 409 | 409 `decision content has no valid state` (note row id=2 → 404 first, since answer query filters `type='decision'`) | PASS |
| 9 | Bad id type | `POST .../abc/answer` | 400 | 400 `id must be an integer` | PASS |
| 10 | Missing optionId | `POST .../1/answer` `{"note":"x"}` | 400 | 400 `optionId is required` | PASS |
| 11 | Prototype 200 | `GET /prototype/qa/a.html` | 200 text/html | 200 `text/html; charset=utf-8`, body `<h1>qa proto</h1>`, `X-Content-Type-Options: nosniff` | PASS |
| 12 | Prototype 404 | `GET /prototype/qa/missing.html` | 404 | 404 | PASS |
| 13 | Traversal 400 | `GET /prototype/%2e%2e%2f` and `/%2e%2e%2f%2e%2e%2fetc%2fpasswd` | 400 | 400 `invalid prototype id: ../` / `.../etc/passwd` | PASS |
| 14 | Traversal bare `..` | `GET /prototype/../../etc/passwd` (no-follow) | 307 (mux clean-path) then SPA 200 shell — per `prototype_test.go` | 307 → `Location:/etc/passwd` → 200 SPA shell (no secret). **Note:** curl auto-normalizes `../`, so it shows 200 directly; raw request shows 307. Matches documented test (307) | PASS |
| 15 | Traversal abs id | `GET /prototype/%2Fetc%2Fpasswd` | 400 | 400 (handler rejects absolute); bare `//etc/hosts` → 307 mux rewrite | PASS |
| 16 | Auth: no Bearer | `POST .../1/answer` (port 18132, no header) | 401 | 401 `unauthorized` | PASS |
| 17 | Auth: with Bearer | `POST .../1/answer` (Bearer qa-token-123) | 200 | 200 `{"ok":true,"state":"answered"}` | PASS |
| 18 | Read route open | `GET /mnemonic/decisions` (both ports, with/without Bearer) | 200 | 200 (reads open on loopback) | PASS |
| 19 | Dashboard render | `GET /` and `/decisions` | SPA shell + assets 200 | `/` 200 html, `/decisions` 200 html, JS asset `index-r47YdLdY.js` 200 `text/javascript` (embedded ui/dist) | PASS |
| 20 | Shared-store durability | `observation_versions` after 3 answers across both servers | 3 rows (append-only) | 3 rows (rev 1–3), notes qa check / qa re-check / auth check | PASS |

## Notes

- **Version count:** step 6 shows 2 rows (the two answers on :18131). After the auth-pass answer on :18132 (same shared store), total is 3 — confirming append-only, cross-restart durability.
- **Non-decision 409:** the answer handler's row lookup filters `type='decision'`, so a `type=note` row returns 404, not 409. The 409 "no valid state" path requires a `type=decision` row whose content lacks a `state` field — that is what row 3 (`interview/qa/nostate`) exercises, matching `TestDecisionAnswer409MissingState`.
- **Traversal `../../`:** bare `..` segments are rewritten by Go 1.22's `net/http` ServeMux clean-path logic into a 307 to a path outside `/prototype`, which falls through to the SPA shell (200). This is exactly what `prototype_test.go` documents and asserts (it never serves `secret.html`). %-encoded traversal reaches the handler and is 400. No root escape in either case.
- **`updatedBy=user:unknown`:** no `answerer` query param was sent, so the handler defaults to `unknown` (per decisions.go:222-225). Correct.
- **Browser render:** the SPA shell + hashed JS asset serve 200 from the embedded `ui/dist`; the `/decisions` page data comes from the API verified in steps 3-8. A live headless-browser render is not required (vitest 9/9 + build check cover it), so none was performed.

## Verdict

**PASS.** All 20 runtime steps match the documented contract (route paths, params, auth model, error codes 400/401/404/409, version-append durability, and the prototype traversal guard). Zero failures. The two nuances — the `type=decision` filter turning a non-decision row into 404 (not 409), and the mux's 307 clean-path rewrite for bare `..` — are both documented in the code and tests and are the intended behavior, not defects.
