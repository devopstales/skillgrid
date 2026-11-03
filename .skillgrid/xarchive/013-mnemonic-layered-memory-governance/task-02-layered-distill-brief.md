## 02-layered-distill

### Goal

A session's raw record distills upward (L0→L1 atoms→L2 scenario→L3 persona), provenance-linked, LLM-cached by content-hash with a deterministic no-LLM floor — so the next session bootstraps from stable layers; inspectable via `mem_layers`.

### Out of scope / Non-Goals

- Retrieval budgets / layered retrieval (step 03)
- The file-tiering side system (`tiered_contents`); multi-tenant; changing 005 tools

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-02` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 01-governance-fields

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/store/migrations/015_layered_memory_governance.sql`
- Create: `skillgrid-cli/internal/mnemonic/memory/layer/distill.go`
- Create: `skillgrid-cli/internal/mnemonic/memory/layer/layers.go`
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_memory_layer.go`
- Modify: `skillgrid-cli/internal/mnemonic/service/service.go`
- Test: `skillgrid-cli/internal/mnemonic/memory/layer/distill_test.go`, `skillgrid-cli/internal/mnemonic/memory/layer/layers_test.go`, `skillgrid-cli/internal/mnemonic/mcp/tools_memory_layer_test.go`, `skillgrid-cli/internal/mnemonic/service/distill_test.go`

**Interfaces:**
- Consumes: 01 governance (version-append for correctable L1 atoms; 015 schema seam); 005 session record + `CapturePassive` heuristics; 005 `mem_*` (stable)
- Produces: `observation_layers` (L0→L1→L2→L3 links + layer type) + `personas` (L3); the opt-in async session-close distillation hook; `mem_layers <session_id|topic_key>` — relied on by step 03 (L2/L3 bootstrap sources)

### Tasks

<!-- [RED] items are the applicable threat-matrix rows (Mnemonic tool surface + Provenance integrity), ordered BEFORE the production [AFK] tasks. Each uses the TDD micro-cycle a–e. Scenario names are the referenceable strings from acceptance.feature @step-02. -->

- [ ] 02.1 `[RED]` Threat "Provenance integrity" — distilled L1/L2/L3 carries a resolvable L0 link; a record with no resolvable source is **not created**; `mem_layers` surfaces the chain; no-LLM floor produces a provenance-linked ladder offline (Scenarios: `distilled-layer-carries-resolvable-l0-provenance`, `layer-with-unresolvable-source-not-created`, `mem-layers-inspects-l0-to-l3-chain`, `no-llm-floor-produces-provenance-ladder-offline`)
  - [ ] 02.1.a Write failing test
  - [ ] 02.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/layer/... -run Provenance -count=1` — Expected: FAIL
  - [ ] 02.1.c Minimal implementation
  - [ ] 02.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/layer/... -run Provenance -count=1` — Expected: PASS
  - [ ] 02.1.e Commit — `feat(memory/layer): provenance-linked distillation (no orphan layers)`
- [ ] 02.2 `[RED]` Threat "Mnemonic tool surface" — `mem_layers` registered + 005 tools still stable + bad layer args rejected (Scenarios: `mem-layers-registered-005-stable`, `bad-layer-args-rejected`)
  - [ ] 02.2.a Write failing test
  - [ ] 02.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run LayerTools -count=1` — Expected: FAIL
  - [ ] 02.2.c Minimal implementation
  - [ ] 02.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run LayerTools -count=1` — Expected: PASS
  - [ ] 02.2.e Commit — `feat(memory/layer): register mem_layers + keep 005 tools stable`
- [ ] 02.3 `[AFK]` Additive `015_*` layer schema — `observation_layers` (L0→L1→L2→L3 links + layer type) + `personas` (L3) — `Run: go test ./skillgrid-cli/internal/mnemonic/store/... -count=1` — Expected: PASS
- [ ] 02.4 `[AFK]` Session-close distillation (opt-in, async) refines L0 → L1 atoms + L2 scenario(s) + L3 persona delta, each linked to its L0 source (Scenario: `session-close-distill-l0-to-l1-l2-l3`) — `Run: go test ./skillgrid-cli/internal/mnemonic/service/... -run DistillHook -count=1` — Expected: PASS
- [ ] 02.5 `[AFK]` Distillation LLM-cached by content-hash; re-distill only on source change (Scenario: `distill-llm-cached-by-content-hash`) — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/layer/... -run CacheHash -count=1` — Expected: PASS
- [ ] 02.6 `[AFK]` No-LLM deterministic floor (005 `CapturePassive` Key-Learnings/Lesson/Discovery heuristics) produces L1 atoms so the ladder works offline (Scenarios: `no-llm-floor-produces-provenance-ladder-offline`, `no-llm-floor-produces-l1-atoms`) — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/layer/... -run NoLLMFloor -count=1` — Expected: PASS
- [ ] 02.7 `[AFK]` A session with no new L1-able content is a no-op (no empty atoms/scenarios/persona-delta fabricated) (Scenario: `session-with-no-l1-content-is-noop`) — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/layer/... -run NoOp -count=1` — Expected: PASS
- [ ] 02.8 `[AFK]` L1 atoms are correctable, not just deletable — an atom update appends a version (step 01) and the provenance link traces it back to its L0 source (Scenario: `l1-atom-correctable-with-traceable-provenance`) — `Run: go test ./skillgrid-cli/internal/mnemonic/service/... -run AtomCorrectable -count=1` — Expected: PASS
- [ ] 02.9 `[AFK]` `mem_layers <session_id|topic_key>` returns the L0→L1→L2→L3 chain with each layer's provenance link (Scenario: `mem-layers-inspects-l0-to-l3-chain`) — `Run: go test ./skillgrid-cli/internal/mnemonic/service/... -run MemLayers -count=1` — Expected: PASS
- [ ] 02.10 `[AFK]` 005 `mem_*` tools unchanged; bad layer args rejected clearly (Scenarios: `mem-layers-registered-005-stable`, `bad-layer-args-rejected`) — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/internal/mnemonic/service/... -count=1` — Expected: PASS

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/layer/... ./skillgrid-cli/internal/mnemonic/store/... -count=1` | PASS | | |
| Acceptance `@step-02` / `@p0` | `go test ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/internal/mnemonic/service/... -count=1` + BDD `@step-02 @p0` | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/... -count=1` | PASS | | |
| Rollback boundary | drop `015_*` layer portion + `memory/layer` package + `tools_memory_layer.go` + distill hook; 005 + step-01 governance intact | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(memory/layer): session-close distillation (L0→L1/L2/L3, provenance-linked, LLM-cached + no-LLM floor) + mem_layers`

---

