## 09-session-promotion

### Goal

L0/L1 session → L2/L3 graph auto-promotion.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-09` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 07

**Files:**
- `internal/mnemonic/memory/session.go` — MODIFY: session-to-graph promotion on SessionEnd

**Interfaces:**
- Consumes: codeindex graph from step 07; `LayerSummary` populated during session; `graph_ref` column on observations
- Produces: permanent graph nodes on SessionEnd; promoted nodes queryable via mem_search and codeindex graph traversal

### Tasks

- [ ] 09.1 `[RED]` SessionEnd with valid summary creates permanent graph node
  - [ ] 09.1.a Write failing test (`TestSessionEndCreatesGraphNode`): start a session, save observations, end the session with a populated `LayerSummary`; verify a new graph node is created in the codeindex graph; verify the node links to all session observations via `graph_ref`; verify the node is queryable via `mem_search`
  - [ ] 09.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSessionEndCreatesGraphNode'` — Expected: FAIL
  - [ ] 09.1.c Minimal implementation — in `SessionEnd`, check if the session has a valid `LayerSummary` (quality threshold check); if so, create a permanent graph node in the codeindex graph representing the session; link the node to all session observations via `graph_ref`; the node is queryable via `mem_search` and codeindex graph traversal
  - [ ] 09.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSessionEndCreatesGraphNode'` — Expected: PASS
  - [ ] 09.1.e Commit — `feat(mnemonic): session-to-graph promotion on SessionEnd`
- [ ] 09.2 `[RED]` No promotion when summary is empty or below quality threshold
  - [ ] 09.2.a Write failing test (`TestSessionEndNoPromotionBelowThreshold`): start a session with no `LayerSummary` (empty); end the session; verify no graph node is created; start another session with a low-quality summary (below threshold); end it; verify no graph node is created; verify the session observations are still accessible via `mem_search`
  - [ ] 09.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSessionEndNoPromotionBelowThreshold'` — Expected: FAIL
  - [ ] 09.2.c Minimal implementation — add a quality threshold check in the promotion path: require `LayerSummary` to be non-empty and meet a minimum content length or structured section count; skip promotion (with a debug log) when the threshold is not met
  - [ ] 09.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSessionEndNoPromotionBelowThreshold'` — Expected: PASS
  - [ ] 09.2.e Commit — `feat(mnemonic): skip promotion when summary below quality threshold`
- [ ] 09.3 `[AFK]` Promotion is idempotent — no duplicate graph nodes
  - [ ] 09.3.a Write failing test (`TestSessionEndIdempotentPromotion`): end a session with a valid summary; verify a graph node is created; end the same session again (re-trigger); verify no duplicate graph node is created (dedup check); verify the existing node is reused
  - [ ] 09.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSessionEndIdempotentPromotion'` — Expected: FAIL
  - [ ] 09.3.c Minimal implementation — before creating a graph node, check if a node already exists for this session ID (dedup query on session_id); if found, skip creation and reuse the existing node; ensure promotion is idempotent
  - [ ] 09.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSessionEndIdempotentPromotion'` — Expected: PASS
  - [ ] 09.3.e Commit — `feat(mnemonic): idempotent session promotion with dedup`

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSessionEndCreatesGraphNode\|TestSessionEndNoPromotionBelowThreshold'` | PASS | | |
| Acceptance `@step-09` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/` | PASS | | |
| Rollback boundary | verify promotion only fires on SessionEnd with valid summary | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): L0/L1 session to L2/L3 graph auto-promotion`

---

