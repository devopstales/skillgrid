## 01-community-detection

### Goal

Additive `012_*` schema + pure-Go Leiden (`bluuewhale/loom`) + god nodes + LLM-free community labels + `code_communities` / `code_god_nodes` / `code_explain_community` MCP/CLI tools — AND the **retrieval-eval harness** (git-derived ground truth, significance-tested ablations) that gates 005's ranker — AND the **retrieval-quality layer** it proves (explainable bounded rerank table, file-level RRF agreement, categorical confidence→action + fallbacks, skeletonized snippets, output-time secret redaction, `doctor --strict`, `skillgrid eval` CLI).

### Out of scope / Non-Goals

- Doc/config/SQL knowledge nodes (step 03)
- Process flows (step 02)
- Changing 005 tool *names/required params* (the ranker internals are modified, the contract is not)

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-01` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Produces contracts listed under Interfaces are available to dependents
- [ ] No Global Constraint violated

> Depends on: none (005 done; 010 not required for this step)

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/store/migrations/012_community_knowledge_graph.sql`
- Create: `skillgrid-cli/internal/mnemonic/community/leiden.go`
- Create: `skillgrid-cli/internal/mnemonic/community/godnodes.go`
- Create: `skillgrid-cli/internal/mnemonic/community/labels.go`
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_code_community.go`
- Create: `skillgrid-cli/internal/mnemonic/eval/genqueries.go`
- Create: `skillgrid-cli/internal/mnemonic/eval/harness.go`
- Create: `skillgrid-cli/internal/mnemonic/eval/metrics.go`
- Create: `skillgrid-cli/internal/mnemonic/hybrid/confidence.go`
- Create: `skillgrid-cli/internal/mnemonic/hybrid/snippet.go`
- Modify: `skillgrid-cli/internal/mnemonic/hybrid/rerank.go`
- Modify: `skillgrid-cli/internal/mnemonic/service/service.go` (community facade)
- Modify: `skillgrid-cli/cmd/skillgrid/code_intel.go` (CLI community commands + `skillgrid eval` runner)
- Modify: `skillgrid-cli/cmd/skillgrid/doctor.go` (`doctor --strict`)
- Modify: `skillgrid-cli/go.mod` (add `bluuewhale/loom`)
- Test: `skillgrid-cli/internal/mnemonic/community/...`, `skillgrid-cli/internal/mnemonic/eval/...`, `skillgrid-cli/internal/mnemonic/hybrid/...`, `skillgrid-cli/internal/mnemonic/store/...`, `skillgrid-cli/internal/mnemonic/mcp/...`, `skillgrid-cli/cmd/skillgrid/...`

**Interfaces:**
- Consumes: 005's symbols/edges tables + existing hybrid ranker, existing store migration runner, 005 tool registration baseline
- Produces: `communities` / `community_meta` tables + god-node ranking; LLM-free community labels; `code_communities`, `code_god_nodes` (`--exclude-hubs`), `code_explain_community` MCP tools + CLI parity; `skillgrid eval` runner; git-derived leak-free query set + one-shared-index-per-corpus ablation + file-granularity metrics with paired bootstrap CI + permutation p-values; the retrieval-quality ranker layer (explainable rerank table, file-level RRF agreement, categorical confidence→action + fallback suggestions, skeletonized snippets, output-time redaction) exposed additively in 005 `code_search`/`code_read` responses; `doctor --strict` redaction + freshness state

### Tasks

- [ ] 01.1 `[RED]` Retrieval quality / evaluation — eval harness derives a leak-free git query set and reports metrics with paired CI + p-values (Scenario: Evaluation harness derives a leak-free query set and reports significance) — threat: Retrieval quality / evaluation
  - [ ] 01.1.a Write failing test — from a git-history fixture, the harness mints a query set (commit subject → changed files), drops merges/reverts/releases/bumps/formatting + changelog-like + benchmark-touching commits, builds ONE index per corpus shared by all variants, and reports `recall@5/10`, `MRR`, `nDCG@10`, `useful@budget`, `tokens`, `dup%`, p50/p95/p99 with a paired bootstrap 95% CI + permutation p-value per non-baseline row
  - [ ] 01.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/eval/... -run EvalHarnessSignificance -count=1` — Expected: FAIL
  - [ ] 01.1.c Minimal implementation — `eval/genqueries.go` (git-derived generator + noise drops + `CORPUS_EXCLUDES`) + `eval/metrics.go` (file-granularity IR metrics + seeded paired bootstrap CI + permutation p-value) + `eval/harness.go` (one-index-per-corpus ablation runner)
  - [ ] 01.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/eval/... -run EvalHarnessSignificance -count=1` — Expected: PASS
  - [ ] 01.1.e Commit — `feat(eval): git-derived leak-free query set with paired significance`
- [ ] 01.2 `[RED]` Retrieval quality / evaluation — stale expectation fails the run loudly (Scenario: Stale evaluation expectation fails the run loudly) — threat: Retrieval quality / evaluation
  - [ ] 01.2.a Write failing test — a query whose expected file no longer exists at HEAD makes `validate_queries` fail the run with a loud error, not a silently deflated score
  - [ ] 01.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/eval/... -run EvalStaleExpectation -count=1` — Expected: FAIL
  - [ ] 01.2.c Minimal implementation — `validate_queries` pass in `eval/harness.go` (expected-file-exists-at-HEAD check)
  - [ ] 01.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/eval/... -run EvalStaleExpectation -count=1` — Expected: PASS
  - [ ] 01.2.e Commit — `feat(eval): validate_queries fails on stale expectations`
- [ ] 01.3 `[RED]` Security boundary (output) — planted secret is redacted in search/read output (Scenario: Snippets are skeletonized and secrets are redacted in output) — threat: Security boundary (output)
  - [ ] 01.3.a Write failing test — a secret-like pattern in an indexed file is replaced (never emitted raw) in `code_search`/`code_read` response text; index-time exclusion alone does not count
  - [ ] 01.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/hybrid/... -run OutputRedaction -count=1` — Expected: FAIL
  - [ ] 01.3.c Minimal implementation — output-time secret redaction in `hybrid/snippet.go` applied to search/read response text
  - [ ] 01.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/hybrid/... -run OutputRedaction -count=1` — Expected: PASS
  - [ ] 01.3.e Commit — `feat(hybrid): output-time secret redaction`
- [ ] 01.4 `[RED]` Security boundary (output) — strict doctor reports redaction + freshness and exits non-zero on violation (Scenarios: Strict doctor reports redaction and freshness state; Doctor strict exits non-zero on redaction violation) — threat: Security boundary (output)
  - [ ] 01.4.a Write failing test — `doctor --strict` reports redaction state + freshness state, and exits non-zero when a redaction or freshness violation exists (CI-usable)
  - [ ] 01.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/cmd/skillgrid/... -run DoctorStrict -count=1` — Expected: FAIL
  - [ ] 01.4.c Minimal implementation — `--strict` flag + redaction/freshness checks + non-zero exit in `cmd/skillgrid/doctor.go`
  - [ ] 01.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/cmd/skillgrid/... -run DoctorStrict -count=1` — Expected: PASS
  - [ ] 01.4.e Commit — `feat(cli): doctor --strict redaction and freshness state`
- [x] 01.5 `[RED]` Mnemonic tool surface — 005 `code_*` schema stable (additive response gains only) before new tools land (Scenario: Community detection returns labeled subsystems and existing search tools stay stable) — threat: Mnemonic tool surface
  - [ ] 01.5.a Write failing test — assert 005 `code_*` tool names + required params are unchanged (baseline lock); `code_search` response schema may only *gain* confidence/reasons/redaction fields additively
  - [ ] 01.5.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run ToolSurfaceBaseline -count=1` — Expected: FAIL (red until the additive-gain assertion exists)
  - [ ] 01.5.c Minimal implementation — lock 005 tool-surface baseline + additive-response assertion
  - [ ] 01.5.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run ToolSurfaceBaseline -count=1` — Expected: PASS
  - [ ] 01.5.e Commit — `test(mnemonic): lock 005 tool surface baseline (additive gains only)`
- [x] 01.6 `[RED]` Leiden over 005 edges produces labeled communities + community tools (Scenario: Community detection returns labeled subsystems and existing search tools stay stable)
  - [ ] 01.6.a Write failing test — fixture graph of symbols/edges; after the community pass, community rows partition the graph coherently, each community carries an LLM-free label; `code_communities` returns the partition; 005 tools still registered
  - [ ] 01.6.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/community/... ./skillgrid-cli/internal/mnemonic/mcp/... -run Communities -count=1` — Expected: FAIL
  - [ ] 01.6.c Minimal implementation — `012_*` migration + `community/leiden.go` (loom `NodeRegistry` + `LeidenOptions{Seed, Resolution, MaxIterations, NumRuns}`, partition → community rows) + `labels.go` + `go.mod` dep + `tools_code_community.go` `code_communities` + service facade
  - [ ] 01.6.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/community/... ./skillgrid-cli/internal/mnemonic/mcp/... -run Communities -count=1` — Expected: PASS
  - [ ] 01.6.e Commit — `feat(mnemonic): leiden community detection with llm-free labels`
- [x] 01.7 `[RED]` God nodes ranked by degree with exclude-hubs (Scenario: God nodes rank hubs and hub exclusion suppresses utility symbols)
  - [ ] 01.7.a Write failing test — `code_god_nodes` returns symbols ranked by degree; `--exclude-hubs` suppresses utility super-hubs from the ranking
  - [ ] 01.7.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/community/... -run GodNodes -count=1` — Expected: FAIL
  - [ ] 01.7.c Minimal implementation — `community/godnodes.go` (degree ranking + `--exclude-hubs`) + `code_god_nodes` tool + CLI flag
  - [ ] 01.7.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/community/... -run GodNodes -count=1` — Expected: PASS
  - [ ] 01.7.e Commit — `feat(mnemonic): god-node ranking with exclude-hubs`
- [x] 01.8 `[RED]` Mnemonic tool surface — community + eval tools register and reject bad args (Scenarios: Community tools reject bad args clearly; bad community/eval args rejected) — threat: Mnemonic tool surface
  - [ ] 01.8.a Write failing test — `code_communities` / `code_god_nodes` / `code_explain_community` + `skillgrid eval` registered with distinct `code_*` names / CLI verbs; bad/missing args (e.g. non-existent community id, unknown eval corpus) rejected clearly with a validation error, no invented communities or runs
  - [ ] 01.8.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/cmd/skillgrid/... -run CommunityAndEvalArgs -count=1` — Expected: FAIL
  - [ ] 01.8.c Minimal implementation — arg validation in `tools_code_community.go` + `code_intel.go` eval runner + service
  - [ ] 01.8.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/cmd/skillgrid/... -run CommunityAndEvalArgs -count=1` — Expected: PASS
  - [ ] 01.8.e Commit — `feat(mnemonic): community and eval tool arg validation`
- [ ] 01.9 `[RED]` Explainable rerank table + file-level RRF agreement (Scenario: Search response carries confidence action rerank reasons and fallbacks)
  - [ ] 01.9.a Write failing test — each hit carries a named, capped boost/penalty factor with a written rationale (exact-symbol +, definition-kind +, path-match +, degree + bounded, source-over-prose +, documentation −, generated/vendor −, test-on-non-test −); fusion candidates keyed by `(path, line-bucket)`; two retrievers finding the same file at different locators yield one strong file-level candidate under a fixed agreement weight
  - [ ] 01.9.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/hybrid/... -run RerankAgreement -count=1` — Expected: FAIL
  - [ ] 01.9.c Minimal implementation — explainable bounded rerank table + file-level RRF agreement in `hybrid/rerank.go`
  - [ ] 01.9.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/hybrid/... -run RerankAgreement -count=1` — Expected: PASS
  - [ ] 01.9.e Commit — `feat(hybrid): explainable rerank table and file-level rrf agreement`
- [ ] 01.10 `[RED]` Categorical confidence→action + fallback suggestions (Scenario: Search response carries confidence action rerank reasons and fallbacks)
  - [ ] 01.10.a Write failing test — search responses carry categorical `high|medium|low` confidence mapped to an explicit agent action (high = read ranges + answer; medium = read + one confirming grep; low = use fallbacks); `low` attaches fallback suggestions (ready `rg` patterns, likely paths, "broaden query")
  - [ ] 01.10.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/hybrid/... -run ConfidenceAction -count=1` — Expected: FAIL
  - [ ] 01.10.c Minimal implementation — categorical confidence→action contract + fallback suggestions in `hybrid/confidence.go`, surfaced in search responses
  - [ ] 01.10.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/hybrid/... -run ConfidenceAction -count=1` — Expected: PASS
  - [ ] 01.10.e Commit — `feat(hybrid): confidence-to-action contract with fallbacks`
- [ ] 01.11 `[RED]` Skeletonized snippets + SimHash near-duplicate suppression (Scenario: Snippets are skeletonized and secrets are redacted in output)
  - [ ] 01.11.a Write failing test — result snippets collapse unrelated bodies while preserving imports, signatures, matched lines, and exact read ranges; SimHash suppression cuts near-duplicate hits (`dup%`) without moving a metric
  - [ ] 01.11.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/hybrid/... -run SnippetSkeleton -count=1` — Expected: FAIL
  - [ ] 01.11.c Minimal implementation — snippet skeletonization + SimHash near-dup suppression in `hybrid/snippet.go`
  - [ ] 01.11.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/hybrid/... -run SnippetSkeleton -count=1` — Expected: PASS
  - [ ] 01.11.e Commit — `feat(hybrid): skeletonized snippets and simhash dedup`
- [ ] 01.12 `[RED]` Shipped ranking config is the significance winner across pooled corpora (Scenario: Shipped ranking config is the significance winner)
  - [ ] 01.12.a Write failing test — on ≥2 pooled corpora, the shipped ranking config is non-negative vs the 005 baseline and every shipped signal survived significance; a candidate that fails is removed/kept-off with the decision + CI/p-value recorded in the report
  - [ ] 01.12.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/eval/... -run ShippedConfigSignificance -count=1` — Expected: FAIL
  - [ ] 01.12.c Minimal implementation — multi-corpus pooling + non-negative-across-all gate + decision record in `eval/harness.go`; wire the winning config as the shipped ranker config
  - [ ] 01.12.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/eval/... -run ShippedConfigSignificance -count=1` — Expected: PASS
  - [ ] 01.12.e Commit — `feat(eval): ship only the significance-winning ranking config`
- [x] 01.13 `[AFK]` Community explanation returns members and entry points (Scenario: Community explanation returns members and entry points) — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/internal/mnemonic/service/... -run ExplainCommunity -count=1` — Expected: PASS
- [x] 01.14 `[AFK]` Tiny graph (< 2 nodes) yields one trivial community, no crash (Scenario: Tiny graph yields a single trivial community) — `Run: go test ./skillgrid-cli/internal/mnemonic/community/... -run TinyGraph -count=1` — Expected: PASS
- [x] 01.15 `[AFK]` Community label falls back to community-N when no god node (Scenario: Community label falls back when no god node exists) — `Run: go test ./skillgrid-cli/internal/mnemonic/community/... -run LabelFallback -count=1` — Expected: PASS
- [x] 01.16 `[AFK]` Communities are seeded, pinned, and cached by content-hash (Scenario: Community partition is reproducible and cached by content-hash) — `Run: go test ./skillgrid-cli/internal/mnemonic/community/... -count=1` — Expected: PASS
- [ ] 01.17 `[AFK]` Eval harness drops noise commits (merges/reverts/releases/bumps/formatting/changelog-like/benchmark-touching) (Scenario: Evaluation harness drops noise commits from the query set) — `Run: go test ./skillgrid-cli/internal/mnemonic/eval/... -run QuerySetNoiseDrops -count=1` — Expected: PASS
- [ ] 01.18 `[AFK]` Benchmark scaffolding excluded from the graded corpus (Scenario: Evaluation corpus excludes the benchmark scaffolding) — `Run: go test ./skillgrid-cli/internal/mnemonic/eval/... -run CorpusExcludes -count=1` — Expected: PASS
- [ ] 01.19 `[AFK]` Ablation shares one index so deltas measure ranking only (Scenario: Ablation shares one index so deltas measure ranking) — `Run: go test ./skillgrid-cli/internal/mnemonic/eval/... -run OneIndexPerCorpus -count=1` — Expected: PASS
- [ ] 01.20 `[AFK]` Failing ranking signal removed/kept-off with decision recorded (Scenario: Failing ranking signal is removed or kept off with the decision recorded) — `Run: go test ./skillgrid-cli/internal/mnemonic/eval/... -run FailingSignalDecision -count=1` — Expected: PASS
- [ ] 01.21 `[AFK]` CLI parity for community + eval commands (Scenarios: CLI community commands; `skillgrid eval --corpus <self>`) — `Run: go test ./skillgrid-cli/cmd/skillgrid/... -count=1` — Expected: PASS

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/community/... ./skillgrid-cli/internal/mnemonic/eval/... ./skillgrid-cli/internal/mnemonic/hybrid/... ./skillgrid-cli/internal/mnemonic/store/... ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/internal/mnemonic/service/... -count=1` | PASS | | |
| Acceptance `@step-01` / `@p0` | BDD / mapped unit scenarios (incl. `EvalHarnessSignificance`, `OutputRedaction`, `DoctorStrict`, `RerankAgreement`, `ConfidenceAction`) | PASS | | |
| Eval gate | `go test ./skillgrid-cli/internal/mnemonic/eval/... -count=1` + `skillgrid eval --corpus <self> [--corpus <second-language>]` | PASS | | shipped config = significance-winner over 005 baseline on ≥2 corpora |
| Runtime harness | `skillgrid index` on fixture repo; `code_communities` / `code_god_nodes` via MCP + CLI; `skillgrid doctor --strict` | PASS | | |
| Rollback boundary | Drop `012_*` + `community/` + `eval/` + ranker-layer revert + community tools + `go.mod` dep (eval harness **stays**) | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): leiden communities, retrieval-eval harness, and measured retrieval-quality layer`

---

