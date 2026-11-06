# Final whole-branch review — Visual Companion (Mnemonic Decision Bridge)

**Branch:** release/2, `e490ed1..8148b5d` (9 commits, 20 files, +2714/-2)
**Method:** full diff read in slices; 4-way schema check against live files; route-registration audit; `go vet` (http+memory), `go test ./internal/mnemonic/http/ ./internal/mnemonic/memory/` (both ok), `vitest run` (32/32), `npm run build:check` (exit 0); cross-check of all 4 per-task verdicts + ledger + qa-harness.md.

---

## Axis A — Standards: findings list

1. **HIGH — Handoff UI import committed against an untracked file (branch-level, from carried WIP).**
   `skillgrid-ui/src/app.tsx:95` (`handoffRoute` → `import('./features/handoff/HandoffHubPage')`) + `skillgrid-ui/src/components/layout/AppLayout.tsx:17` (nav `/handoff`) landed in 76ec335, but `skillgrid-ui/src/features/handoff/` is **untracked** (`git status` → `??`; `git ls-tree 8148b5d` shows no handoff dir; never committed in any ref). A clean checkout of 8148b5d fails `npm run build:check`: `tsc -b` errors on the unresolvable `./features/handoff/HandoffHubPage` module (dynamic imports are still type-resolved by tsc; vite would fail the same). — Why: the branch's own delivery claim (size:exception single PR, rollback-boundary table names `app.tsx`/`AppLayout.tsx` as decisions-owned files) is undercut; the handoff owner's uncommitted files are frozen into this branch's committed app shell. — Fix: before ship, either (a) commit `skillgrid-ui/src/features/handoff/` (+ the Go handoff files it pairs with) into this PR, or (b) strip the two handoff hunks from app.tsx/AppLayout.tsx and let the handoff owner re-add them.
2. **MEDIUM — Decisions view + iframe depend on handoff code in the same files; rollback boundaries not independently revertable as documented.** `app.tsx`/`AppLayout.tsx` interleave handoff and decisions hunks in the same commit (76ec335). Reverting 76ec335 removes *both*; reverting only the decisions hunks requires a hand-picked patch. `server.go` is clean (handoff and decision registrations are disjoint route lines, no coupling). — Why: the work-unit rollback table claims `app.tsx`/`AppLayout.tsx` revert cleanly with decisions; they do not while the handoff import is present. — Fix: resolves itself if finding 1 is fixed by option (a) (then 76ec335 is a legit two-feature commit) — document it, or (b).
3. **MEDIUM — `handleDecisionAnswer` ignores `curState` after scanning it (dead variable + misleading scan).** `decisions.go:188-191` scans `COALESCE(status,'')` into `curState`; `decisions.go:212` discards it with `_ = curState`. The `status` column plays no role in the gate (correctly — the gate is the content `state` field). — Why: dead code that reads as if `status` mattered; `go vet` doesn't flag it. — Fix: drop the `status` column from the SELECT (or use it in a comment-free way); one-line change.
4. **LOW — `updatedBy` is overwritten by the answer even when the latest writer was the agent.** `decisions.go:229` unconditionally sets `user:<name>`; the agent's next in-place `mem_save` upsert (new content) is fine, but between answer and next upsert the "latest writer" field loses the agent's identity. Matches the documented convention ("`updatedBy` = `user:<name>` after an answer"), so acceptable. — Fix: none required.
5. **LOW — `normalized_hash` goes stale after an answer.** `UpdateContent` (`governance.go:196-201`) rewrites content without recomputing `normalized_hash` (mem_update does, `timeline_update.go:72`). Low impact for decision rows (agent unlikely to re-`mem_save` the same topic_key post-answer); already parked from P1. — Fix: follow-up (recompute in `UpdateContent`), not merge-blocking.
6. **LOW — test name drift after R6.** `TestDecisionAnswerIdempotent` (`decisions_test.go:254`) asserts re-answer *records the new answer + appends a version* — "Idempotent" no longer describes it. — Fix: rename to `TestDecisionAnswerReAnswerUpdates` (cosmetic).
7. **LOW — coarse visibility gate + half-proved test.** `decisions.go:103` (`visibility != 'private'`) is coarser than `visibilityFilter` (a `restricted`/no-grant decision of another owner is listed); accepted as the team-scoped dashboard model in P1. `TestDecisionVisibilityGate` only proves the private-exclusion half. — Fix: follow-up — add a restricted-no-grant case to document the boundary.
8. **NIT — 400 shadowing on the answer route.** Body/id validation (400) runs before existence (404): a bad id + empty `optionId` → 400, not 404. Documented in openapi ordering; tests use well-formed bodies. Acceptable as-is.
9. **Standards pass items (verified):** conventional commits, 9/9 subjects compliant, no AI-attribution trailers (full bodies empty); **no new Go dependencies** (go.mod/go.sum untouched; only pre-existing `yaml.v3` used); UI devDeps added (`jsdom`, `@testing-library/react`, `@testing-library/dom`) are the standard first component-test set — `vitest` itself was already a devDep and the `"test"` script pre-existed; `go vet` clean on both packages (the one warning, `memory/budget.go:114` context cancel, is pre-existing at base e490ed1); no secrets in diff (test tokens `"secret-token"`/`qa-token-123` are test/env values); test names consistent (`TestDecision*`, `TestPrototype*`, `TestPhase8_OpenAPI`); no dead files; `LIMIT 200` cap documented in openapi.yaml:2328 (the cap is not repeated in the user guide — informational only, see triage).

## Axis B — Spec: Must-Haves table

| # | Must-have | Implemented (file:line) | Tested by |
|---|-----------|------------------------|-----------|
| 1 | Agent mem_save decision → listed by GET /mnemonic/decisions with parsed content | `decisions.go:98-137` (SQL type='decision' + parse) | `TestDecisionListRoundtrip` (decisions_test.go:110) + `TestDecisionRoundTrip` step 2 (424-491, seeded via `memory.Save` — the mem_save core) + QA harness steps 1-3 |
| 2 | Only type='decision' + not deleted; malformed content skipped/flagged, never 500 | `decisions.go:102-103` (WHERE), `:63-69` (parseDecisionContent), `:120-124` (flag) | `TestDecisionSkipsMalformed` (:131) |
| 3 | Answer sets state=answered + records optionId/note + appends observation_versions | `decisions.go:226-235` + `memory/governance.go:173-209` (UpdateContent via AppendVersion) | `TestDecisionAnswerRecords` (:181), `TestDecisionAnswerAppendsVersion` (:217, R6), `TestDecisionAnswerIdempotent` (:254) + QA steps 4-6, 20 |
| 4 | Agent reads answer back via mem_search (existing tool) | doc poll loop (`10-decision-companion.md:82-100`) | `TestDecisionRoundTrip` steps 3+5 (:493-540, `SearchOwnerScoped` = the MCP mem_search call path, fresh service) — *backstop* at runtime: QA harness covers serve-side, agent-side via service |
| 5 | topic_key reuse upserts (revision_count bump + version) | existing `memory.Save` upsert path, verified not bypassed | `TestDecisionUpsertBumpsRevision` (:361) |
| 6 | team visible to dashboard reader; private not | `decisions.go:103` visibility filter | `TestDecisionVisibilityGate` (:147) — team-visible + private-hidden (restricted case not covered, see A7) |
| 7 | Answer route write-gated; token set + no-auth → 401 | `decisions.go:73` requireWriteAuth wrap | `TestDecisionAnswerRequiresAuth` (:322) + QA steps 16-17 |
| 8 | Dashboard renders pending cards (question + lettered options + recommended highlight), posts choice | `DecisionsPage.tsx`, `DecisionCard.tsx:87-128` (letters `:110`, recommended `:103,114`), `api.ts:64-77` POST | `DecisionsPage.test.tsx` + `DecisionCard.test.tsx` (vitest 9/9) + QA step 19 |
| 9 | Round-trip survives server restart (SQLite) | store-backed by construction | QA harness step 20 (3 version rows across two server instances on one store) + `TestDecisionRoundTrip` (fresh service over same dataDir) |
| 10 | GET /prototype/{id...} serves .skillgrid/prototype/; rejects absolute/..-escape (400); 404 unknown | `prototype.go:19-63` (guard shape = stitchFile) | `TestPrototypeServeAndTraversal` (prototype_test.go:41) + QA steps 11-15 |
| 11 | visual set → sandboxed iframe in card (allow-scripts, no allow-same-origin); no visual → no iframe | `DecisionCard.tsx:140-149` | `DecisionCard.test.tsx` (src + exact sandbox attr; null iframe when unset) |

**Gaps:** none missing. Every must-have has at least one automated test; #4 and #9 carry their blueprint *backstop* (live agent / live restart) in the QA harness, which passed 20/20.

## Cross-task consistency: the 4-way schema check

Compared field-by-field: Go `decisionContent`/`decisionRow` (decisions.go:28-58) ↔ openapi.yaml `DecisionOption`/`DecisionContent`/`DecisionRow`/`DecisionAnswerInput` (3015-3052 + 2319-2438) ↔ `docs/user-guide/10-decision-companion.md` schema table (37-56) ↔ TS `Decision`/`DecisionContent`/`DecisionOption` (DecisionCard.tsx:13-39) + answer body `{optionId, note}` (api.ts:66).

**RESULT: all four agree.** `question`, `options[{id,label,rationale?}]`, `recommended?`, `state`, `answeredOption?`, `answerNote?`, `updatedBy?`, `visual?`; row: `id, topicKey, title, createdAt, updatedAt, visibility, content, parseError?`; answer: `{optionId, note?}`. One cosmetic asymmetry (not drift): Go `Options` has no `omitempty` (empty list serializes as `[]`, never absent — matches openapi `type: array` and TS `options: DecisionOption[]`, both non-optional). R5 ruling (blueprint wins: `optionId`/`note`, not tasks.md's `answer`/`rationale`) is consistently applied in all four artifacts.

## Route registration audit (server.go final state)

- `registerDecisionRoutes()` called at `server.go:150`, **before** `registerUIRoutes()` (`:151`, whose SPA fallback is in embed.go). ✅
- `GET /mnemonic/decisions` + `POST /mnemonic/decisions/{id}/answer` each registered once (`decisions.go:71-74`). No collision with existing `/mnemonic/*` routes (`/mnemonic/memories`, `/mnemonic/sessions`, `/mnemonic/search`, `/mnemonic/graph/*`, `/mnemonic/files/*`, `/mnemonic/audit` — all distinct literal paths; Go 1.22 mux matches the most specific).
- `GET /prototype/{id...}` registered once at `server.go:210` (inside `registerGraphRoutes`, before `registerUIRoutes`), distinct subtree from plural `GET /prototypes` + `/prototypes/{id...}` (`:203-204`); no `prototype`/`prototypes` entry in `apiPrefixes` 404 catch-all (embed.go:28-36). ✅
- Handoff routes (`GET /activity/snapshots`, `GET /handoff/status`, `:185-186`) disjoint from all decisions/prototype paths. ✅
- **Independence check (handoff ↔ decisions):** decisions code (decisions.go, prototype.go, governance.go UpdateContent, features/decisions/*, decisions route/nav lines, prototype route line) shares **zero** symbols with handoff code. The only entanglement is the shared files: app.tsx/AppLayout.tsx (interleaved hunks, same commit) and server.go (disjoint lines). Go side independently revertable; UI side not until finding 1 is resolved.

## Parked-minors triage

| Parked item (source) | Disposition |
|---|---|
| P1: 400-shadowing on answer (bad id + empty optionId → 400) | **Acceptable as-is** (A8) |
| P1: coarse visibility gate + test proves only private half | **Follow-up ticket** — add restricted-no-grant case (A7) |
| P1: UpdateContent fails on AppendVersion error (mem_update is best-effort) | **Acceptable as-is** — strictly safer; divergence is intentional and documented |
| P1: normalized_hash stale after answer | **Follow-up ticket** (A5) |
| P1: "handoff hunks in both commits" report inaccuracy | **Acceptable** — report wording only, verified wrong |
| P2: answer POST mocked at function level, not fetch boundary | **Follow-up ticket** — one fetch-boundary test (URL/Bearer/body) |
| P2: no live-store round-trip in UI layer | **Acceptable as-is** — closed by QA harness (20/20 PASS, live serve + curl) |
| P2: handoff import untracked in committed tree | **MUST-FIX-BEFORE-MERGE** (A1) — see clean-checkout section |
| P3: sandbox contract duplicated (SandboxPreview.tsx:31 vs DecisionCard.tsx:145) | **Follow-up ticket** — extract shared const when a 3rd usage appears; 2 usages is YAGNI-tolerable |
| P3: bare-.. 307 is mux clean-path, not handler | **Acceptable as-is** — no-leak invariant asserted per case; documented in test |
| P3: committed server.go byte-identical to HEAD (report inaccuracy) | **Acceptable** — report wording only |
| T07: report overclaims 2nd pre-existing YAML parse error | **Acceptable** — wording only; line-2649 error independently reproduced |
| T07: round-trip test is service-level, not live MCP-stdio | **Acceptable as-is** — `SearchOwnerScoped` is verified to be the exact call MCP mem_search runs (mcp/tools_memory.go); full tool-runner variant optional follow-up |
| T07: doc mentions 200-row cap (informational) | **Acceptable as-is** — verified: cap is NOT in `10-decision-companion.md` but IS in openapi.yaml:2328; the ledger entry refers to t07-report.md:102. No doc error to fix |

## Clean-checkout build question (8148b5d standalone)

**`go build ./...` (skillgrid-cli): YES — builds standalone.** All handoff Go code (`main.go`, `http/activity*.go`, `mcp/*`, `relay/*`, `server.go` handoff lines) is committed and self-consistent; the 9 plan commits add only additive Go. (Verified indirectly: `go vet` + `go test` over http+memory pass on the committed tree; no untracked Go file is referenced by committed Go code that would break compilation — handoff files referenced by committed Go, e.g. activity handlers, are themselves committed.)

**`npm run build:check` (skillgrid-ui): NO — fails on a clean checkout.** Evidence:
- `git ls-tree 8148b5d skillgrid-ui/src/features/` → no `handoff` entry; `git log --all --oneline -- skillgrid-ui/src/features/handoff` → empty (never committed).
- `git status --short` → `?? skillgrid-ui/src/features/handoff/` (untracked, present only in this working tree).
- `app.tsx:95` at 8148b5d imports `./features/handoff/HandoffHubPage`; `tsc -b` (first step of `build:check`) type-checks dynamic-import specifiers against the module graph → unresolved-module error. The local `npm run build:check` PASS (exit 0, HandoffHubPage chunk emitted) is valid **only** because the untracked files sit in the working tree.
- Impact on the **decisions** feature itself: none — decisions code has no handoff imports; the failure is purely the carried-WIP handoff route. Fix is either committing the handoff files or stripping its 2 hunks (see A1/A2).

## VERDICT

**FIXES REQUIRED** — 1 merge-blocking finding.

**Merge-blocking:**
- **A1:** `app.tsx:95`/`AppLayout.tsx:17` (handoff route + nav, landed in 76ec335) reference `./features/handoff/HandoffHubPage`, which is untracked/never-committed — a clean checkout of 8148b5d fails `npm run build:check`. Before ship: commit the handoff UI files into the PR (preferred, keeps rollback honest) or strip the two handoff hunks.

Everything else (axis A findings 2-8, all parked minors except the one above) is acceptable-as-is or follow-up-ticket material and does not block ship.
