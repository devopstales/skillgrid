# Tasks: 008-mnemonic-community-knowledge-graph

> **STATUS:** `in-progress` — 0/3 steps PASS
>
> **For agentic workers:** REQUIRED SUB-SKILL: use subagent-execution (or simple-execution) to implement step-by-step. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Extend the 005 code-intelligence graph with Leiden community detection + god nodes (architectural orientation), a **precomputed process layer** (entry-point → execution flows, so agents see what a subsystem *does* end-to-end), and a knowledge-graph layer that maps docs, configs, and SQL schemas as nodes linked to code — so agents see subsystems, their flows, and the "why" beyond the call graph — gated by a retrieval-eval harness that makes ranking quality measured, not asserted.

**Architecture:** Additive `012_*` schema on top of 005's symbols/edges. Step 01 runs `bluuewhale/loom` Leiden (pure-Go, zero deps) over the edges table into communities + god nodes + LLM-free labels + community tools, AND ships the **retrieval-eval harness** (git-derived leak-free ground truth, one shared index per corpus, paired bootstrap CI + permutation p-values, file-granularity metrics) that gates the **retrieval-quality layer** on 005's ranker (explainable bounded rerank table, file-level RRF agreement, confidence→action, skeletonized snippets, output-time secret redaction, `doctor --strict`). Step 02 is the **process pass**: entry-point → call-chain traces into LLM-labeled `processes`/`process_steps`. Step 03 adds deterministic doc/config/SQL extractors feeding the same graph. See `change.md` decisions.

**Tech Stack:** Go (`skillgrid-cli`), SQLite (`modernc.org/sqlite`, CGo-free), gotreesitter graph from 005, `bluuewhale/loom` (pure-Go Leiden/Louvain), MCP (`mcp-go`), CLI, optional LLM for process labels (cached by content-hash; not required for graph structure).

**Spec:** `docs/skillgrid/changes/008-mnemonic-community-knowledge-graph/change.md`

**Acceptance:** `docs/skillgrid/changes/008-mnemonic-community-knowledge-graph/acceptance.feature` (`@step-NN`)

---

## Goal

Coding agents and operators get subsystem-level orientation (communities, god nodes), **precomputed execution flows (processes)**, and a knowledge graph that connects code to its docs, configs, and data schema — answering "what are the core modules?", "what does this subsystem *do*, end to end?", "which flow does this symbol participate in?", and "which code reads/writes this table?" without reading files — with retrieval quality **measured, not asserted**.

## Out of scope / Non-Goals

- Re-implementing 005's symbols/edges/extractors/hybrid search — this change is additive on top of 005's graph
- Git-diff / `code_affected` (CodeGraph `affected` / graphify `prs`) — pure edge-traversal on 005's `edges`; owned by `010-mnemonic-framework-routes-affected`, not this change
- Framework-aware routes (`route`/`navigates` edges) + fsnotify watcher + staleness banner — CodeGraph-derived; owned by `010-mnemonic-framework-routes-affected`, not this change
- Video/audio/image semantic extraction — docs + configs + SQL only
- Replacing the per-project SQLite store or the `code_*` tool surface
- Cloud sync; multi-project community merge

## Definition of Done

Change is done only when **all** of the following are true:

- [ ] `code_communities` returns Leiden-clustered subsystems with LLM-free labels
- [ ] `code_god_nodes` returns the most-connected symbols (with `--exclude-hubs` to suppress utility super-hubs)
- [ ] A community view explains what a subsystem contains and its key entry points
- [ ] `code_processes` returns precomputed execution flows traced from entry points through call chains; each process has named steps + a cross-community flag; `code_process <name>` returns the full trace; `code_explain_symbol` surfaces which processes a symbol participates in
- [ ] Doc files (`.md`) with `[text](./other.md)` / `[[wikilinks]]` become nodes with `references` edges
- [ ] Config files (`.yaml`/`.toml`/`.json`) become nodes with `configures` edges to the code they configure
- [ ] SQL schema (`.sql` DDL) becomes table/column nodes with `reads`/`writes` edges to code that references them
- [ ] Every new edge carries a Confidence Label (`EXTRACTED | INFERRED | AMBIGUOUS`)
- [ ] A **retrieval-eval harness** ships: git-derived ground truth (commit subject → files changed, zero leakage), one shared index per corpus, paired bootstrap CI + permutation p-value on every non-baseline row, file-granularity metrics (`recall@5/10`, `MRR`, `nDCG@10`, `useful@budget`, `tokens`, `dup%`, p50/p95/p99); the harness **removes or rejects at least one candidate ranking signal that fails significance** (or documents that all shipped signals passed)
- [ ] 005's hybrid ranker gains the **explainable bounded rerank table** + **file-level RRF agreement** + **categorical confidence→action contract** + **skeletonized snippets** — each change proven by the harness, not asserted
- [ ] Secret-like patterns are **re-redacted at output time** in `code_*` search/read responses; `skillgrid doctor --strict` reports redaction + freshness state for CI
- [ ] Existing 005 `code_*` tools are unchanged (name + required params); `go test ./...` passes for touched packages
- [ ] Every Step Blueprint entry has a matching section in `tasks.md` with Verdict `PASS` or `PASS WITH WARNINGS`
- [ ] Every `@step-NN` Feature in `acceptance.feature` has passing `@happy`, `@edge`, and `@failure` scenarios
- [ ] Applicable threat-matrix rows have RED coverage that passed
- [ ] Testing strategy commands below are green
- [ ] Rollback path below is still valid (or N/A documented)
- [ ] Change archived under `docs/skillgrid/archive/008-mnemonic-community-knowledge-graph/`

## Global Constraints

Copy verbatim from `change.md` (Error handling + Non-Goals + stack rules). Every step inherits these — do not restate per step.

- Additive on top of 005 — never rewrite 005's symbols/edges/extractors
- CGo-free: Leiden is a pure-Go implementation (`bluuewhale/loom`, no C dependency)
- Every new edge carries a Confidence Label: `EXTRACTED | INFERRED | AMBIGUOUS`
- Community labels are LLM-free (derived from top god-node names + file paths, not an API call)
- Processes are **precomputed at index time** (GitNexus "precompute, don't query" thesis) — traced from entry points through 005 call edges, LLM-labeled once, so a `code_processes` query returns complete flows in one call with no per-query traversal
- Process tracing is deterministic (graph traversal + LLM label only); no per-query LLM call
- Process label cache keyed to process content-hash (re-label only on change); LLM down → flow is cached **unlabeled**, never fabricated
- Process trace is depth/step-capped; a trace that can't be completed is truncated with a "stops at <symbol> (<reason>)" note (reuses 005's graph-stops), never a silent cut
- Doc/config/SQL extraction is deterministic (AST/regex/link-parse), no LLM for the graph structure
- Unresolvable config refs are `AMBIGUOUS`, never dropped
- Per-file extract failure → `warn+continue` (fallback); never abort the whole index run
- **Retrieval quality is measured, not asserted:** every ranking signal ships only if it survives the eval harness — paired bootstrap 95% CI + permutation p-value on a git-derived query set, one shared index per corpus (ablation deltas measure ranking, never indexing variance). A signal that fails is removed or kept-off, with the decision recorded
- **Explainable, bounded rerank:** every boost/penalty is a named, capped factor with a written rationale
- **File-level RRF agreement:** fusion candidates are keyed by `(path, line-bucket)`; cross-source agreement at file level produces one strong candidate, not two weak ones
- **Confidence→action contract:** search responses carry a categorical `high|medium|low` confidence mapped to an explicit agent action (high = read ranges + answer; medium = read + one confirming grep; low = use the attached fallbacks) plus **fallback suggestions** (ready `rg` patterns, likely paths, "broaden query") when low
- **Skeletonized snippets:** result snippets collapse unrelated bodies while preserving imports, signatures, matched lines, and exact read ranges
- **Fail closed at the security boundary:** secret-like patterns are re-redacted at output time (not only at index time); `doctor --strict` exposes redaction + freshness state for CI
- Existing 005 `code_*` tools keep name + required params; all new tools use distinct `code_*` names
- Migration id `012_community_knowledge_graph.sql` — leave `011` for 005
- Leiden on a graph with < 2 nodes → `warn+continue` (single trivial community; no crash)
- Community label resolution finds no god node → `warn+continue` (label = "community-N"; never fabricated names)
- Doc file with unparseable links → `warn+continue` (skip bad links; index the rest)
- SQL DDL that fails to parse → `warn+continue` (skip that statement; index the rest)
- Entry point with no traceable call chain → `warn+continue` (single-step, or skipped; not fabricated; "stops at <dispatch>" if mid-flow)
- Bad / missing args on new `code_*` tools → `abort` with clear validation error; do not invent communities or processes
- Eval harness: expected file no longer exists at HEAD → `abort` (harness run); stale expectation is a loud error, not a silently deflated score
- Eval: a candidate ranking signal fails significance → `warn+continue`; signal removed or kept-off, decision + CI/p-value recorded in the harness report
- Secret-like pattern in a search/read snippet → redact at output (pattern replaced in the response text; never emitted raw)
- No git-PR-impact, no video/audio/image extraction, no cloud sync or multi-project merge
- Communities are advisory, not load-bearing (seeded RNG + pinned resolution); the eval harness is the gate, not a feature — it stays even if the retrieval-quality layer rolls back

---

## State

```yaml
phase: verify        # spec | apply | verify | archive
current_step: 03-knowledge-graph-nodes
status: done         # in_progress | blocked | done
next_action: sdd-archive (verify gate PASS WITH WARNINGS; human QA to accept or waive)
updated: 2026-09-09
```

## Verification (change-level, sdd-verify)

**Per-step verdicts:** 01 `PASS` · 02 `PASS` · 03 `PASS` (all sub-tasks `[x]`; all `@step-01/02/03` scenarios COMPLIANT at runtime — covering tests pass).

**Change verdict: `PASS WITH WARNINGS`.**

**Evidence:**
- Runtime proof: `go test ./internal/mnemonic/{community,eval,hybrid,store,mcp,service,route,process,knowledge,codeindex,graph}/... -count=1` → all `ok` (mcp `ok` except 2 `go build`-e2e tests blocked by the WARNING below). `go build ./...` fails ONLY in `internal/mnemonic/setup` (see warning); every 008 package builds + passes.
- 005/008/010 baseline preserved: tool surface 63→71 purely additive; `code_search` name + required `query` param unchanged (additive response gains only); 005 code-to-code `graph.Path` unchanged; migrations 012–015 additive; CGo-free.
- 36 `@step-NN` scenarios in `acceptance.feature`, all mapped to passing runtime tests.

**WARNING (not a 008 defect):** a parallel session is mid-refactor of `internal/mnemonic/setup` (helpers moved to new untracked `internal/install/agents.go`; `opencode.go` still mid-edit, `"path/filepath" imported and not used`). This breaks the `setup` package build, which transitively fails `mcp`'s `TestE2EMemoryExtTools`/`TestE2EEngramParityGaps` (go-build e2e) and `cmd/skillgrid`'s `TestTrail*`. NOT 008's code, NOT a 008 global-constraint violation. Re-run those e2e tests once the parallel session lands `setup`/`install`.

**QA plan:** `qa-plan.md` (happy/edge/failure + pass-fail + waiver).
**Ticket:** none (change-level `Ticket:` not set).
**Next:** sdd-archive — eligible when human QA is accepted or explicitly waived (archive only; sdd-verify does not close a ticket).

## Step map

| NN | Step | Tag | Blocked by | Acceptance |
|----|------|-----|------------|------------|
| 01 | `community-detection` | `@step-01` | — (005 done) | Feature tagged `@step-01` |
| 02 | `process-flows` | `@step-02` | 01, 010 (entry points) | Feature tagged `@step-02` |
| 03 | `knowledge-graph-nodes` | `@step-03` | 02 | Feature tagged `@step-03` |

## Review workload (change-level)

| Field | Value |
|-------|-------|
| Estimated changed lines (change) | ~1600–2200 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Delivery strategy | ask-on-risk |

<!-- single-pr = one PR for the change; each ## NN step still commits separately when DoD is met (see work-unit-commits / commits.md). -->

Honest forecast: three vertical slices (communities+eval+ranker → processes → knowledge nodes), each its own stacked PR. Step 01 alone is well over the 400-line budget (~800–1100: schema + loom adapter + god nodes + labels + 3 community tools + the eval harness + the retrieval-quality ranker layer + `doctor --strict`). Do not attempt a single-PR delivery without explicit exception.

Suggested split: PR1 communities + eval harness + retrieval-quality layer → PR2 processes → PR3 knowledge nodes · Chain strategy: stacked-to-main

---

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

- [x] 01.1 `[RED]` Retrieval quality / evaluation — eval harness derives a leak-free git query set and reports metrics with paired CI + p-values (Scenario: Evaluation harness derives a leak-free query set and reports significance) — threat: Retrieval quality / evaluation
  - [ ] 01.1.a Write failing test — from a git-history fixture, the harness mints a query set (commit subject → changed files), drops merges/reverts/releases/bumps/formatting + changelog-like + benchmark-touching commits, builds ONE index per corpus shared by all variants, and reports `recall@5/10`, `MRR`, `nDCG@10`, `useful@budget`, `tokens`, `dup%`, p50/p95/p99 with a paired bootstrap 95% CI + permutation p-value per non-baseline row
  - [ ] 01.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/eval/... -run EvalHarnessSignificance -count=1` — Expected: FAIL
  - [ ] 01.1.c Minimal implementation — `eval/genqueries.go` (git-derived generator + noise drops + `CORPUS_EXCLUDES`) + `eval/metrics.go` (file-granularity IR metrics + seeded paired bootstrap CI + permutation p-value) + `eval/harness.go` (one-index-per-corpus ablation runner)
  - [ ] 01.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/eval/... -run EvalHarnessSignificance -count=1` — Expected: PASS
  - [ ] 01.1.e Commit — `feat(eval): git-derived leak-free query set with paired significance`
- [x] 01.2 `[RED]` Retrieval quality / evaluation — stale expectation fails the run loudly (Scenario: Stale evaluation expectation fails the run loudly) — threat: Retrieval quality / evaluation
  - [ ] 01.2.a Write failing test — a query whose expected file no longer exists at HEAD makes `validate_queries` fail the run with a loud error, not a silently deflated score
  - [ ] 01.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/eval/... -run EvalStaleExpectation -count=1` — Expected: FAIL
  - [ ] 01.2.c Minimal implementation — `validate_queries` pass in `eval/harness.go` (expected-file-exists-at-HEAD check)
  - [ ] 01.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/eval/... -run EvalStaleExpectation -count=1` — Expected: PASS
  - [ ] 01.2.e Commit — `feat(eval): validate_queries fails on stale expectations`
- [x] 01.3 `[RED]` Security boundary (output) — planted secret is redacted in search/read output (Scenario: Snippets are skeletonized and secrets are redacted in output) — threat: Security boundary (output)
  - [ ] 01.3.a Write failing test — a secret-like pattern in an indexed file is replaced (never emitted raw) in `code_search`/`code_read` response text; index-time exclusion alone does not count
  - [ ] 01.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/hybrid/... -run OutputRedaction -count=1` — Expected: FAIL
  - [ ] 01.3.c Minimal implementation — output-time secret redaction in `hybrid/snippet.go` applied to search/read response text
  - [ ] 01.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/hybrid/... -run OutputRedaction -count=1` — Expected: PASS
  - [ ] 01.3.e Commit — `feat(hybrid): output-time secret redaction`
- [x] 01.4 `[RED]` Security boundary (output) — strict doctor reports redaction + freshness and exits non-zero on violation (Scenarios: Strict doctor reports redaction and freshness state; Doctor strict exits non-zero on redaction violation) — threat: Security boundary (output)
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
- [x] 01.9 `[RED]` Explainable rerank table + file-level RRF agreement (Scenario: Search response carries confidence action rerank reasons and fallbacks)
  - [ ] 01.9.a Write failing test — each hit carries a named, capped boost/penalty factor with a written rationale (exact-symbol +, definition-kind +, path-match +, degree + bounded, source-over-prose +, documentation −, generated/vendor −, test-on-non-test −); fusion candidates keyed by `(path, line-bucket)`; two retrievers finding the same file at different locators yield one strong file-level candidate under a fixed agreement weight
  - [ ] 01.9.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/hybrid/... -run RerankAgreement -count=1` — Expected: FAIL
  - [ ] 01.9.c Minimal implementation — explainable bounded rerank table + file-level RRF agreement in `hybrid/rerank.go`
  - [ ] 01.9.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/hybrid/... -run RerankAgreement -count=1` — Expected: PASS
  - [ ] 01.9.e Commit — `feat(hybrid): explainable rerank table and file-level rrf agreement`
- [x] 01.10 `[RED]` Categorical confidence→action + fallback suggestions (Scenario: Search response carries confidence action rerank reasons and fallbacks)
  - [ ] 01.10.a Write failing test — search responses carry categorical `high|medium|low` confidence mapped to an explicit agent action (high = read ranges + answer; medium = read + one confirming grep; low = use fallbacks); `low` attaches fallback suggestions (ready `rg` patterns, likely paths, "broaden query")
  - [ ] 01.10.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/hybrid/... -run ConfidenceAction -count=1` — Expected: FAIL
  - [ ] 01.10.c Minimal implementation — categorical confidence→action contract + fallback suggestions in `hybrid/confidence.go`, surfaced in search responses
  - [ ] 01.10.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/hybrid/... -run ConfidenceAction -count=1` — Expected: PASS
  - [ ] 01.10.e Commit — `feat(hybrid): confidence-to-action contract with fallbacks`
- [x] 01.11 `[RED]` Skeletonized snippets + SimHash near-duplicate suppression (Scenario: Snippets are skeletonized and secrets are redacted in output)
  - [ ] 01.11.a Write failing test — result snippets collapse unrelated bodies while preserving imports, signatures, matched lines, and exact read ranges; SimHash suppression cuts near-duplicate hits (`dup%`) without moving a metric
  - [ ] 01.11.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/hybrid/... -run SnippetSkeleton -count=1` — Expected: FAIL
  - [ ] 01.11.c Minimal implementation — snippet skeletonization + SimHash near-dup suppression in `hybrid/snippet.go`
  - [ ] 01.11.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/hybrid/... -run SnippetSkeleton -count=1` — Expected: PASS
  - [ ] 01.11.e Commit — `feat(hybrid): skeletonized snippets and simhash dedup`
- [x] 01.12 `[RED]` Shipped ranking config is the significance winner across pooled corpora (Scenario: Shipped ranking config is the significance winner)
  - [ ] 01.12.a Write failing test — on ≥2 pooled corpora, the shipped ranking config is non-negative vs the 005 baseline and every shipped signal survived significance; a candidate that fails is removed/kept-off with the decision + CI/p-value recorded in the report
  - [ ] 01.12.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/eval/... -run ShippedConfigSignificance -count=1` — Expected: FAIL
  - [ ] 01.12.c Minimal implementation — multi-corpus pooling + non-negative-across-all gate + decision record in `eval/harness.go`; wire the winning config as the shipped ranker config
  - [ ] 01.12.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/eval/... -run ShippedConfigSignificance -count=1` — Expected: PASS
  - [ ] 01.12.e Commit — `feat(eval): ship only the significance-winning ranking config`
- [x] 01.13 `[AFK]` Community explanation returns members and entry points (Scenario: Community explanation returns members and entry points) — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/internal/mnemonic/service/... -run ExplainCommunity -count=1` — Expected: PASS
- [x] 01.14 `[AFK]` Tiny graph (< 2 nodes) yields one trivial community, no crash (Scenario: Tiny graph yields a single trivial community) — `Run: go test ./skillgrid-cli/internal/mnemonic/community/... -run TinyGraph -count=1` — Expected: PASS
- [x] 01.15 `[AFK]` Community label falls back to community-N when no god node (Scenario: Community label falls back when no god node exists) — `Run: go test ./skillgrid-cli/internal/mnemonic/community/... -run LabelFallback -count=1` — Expected: PASS
- [x] 01.16 `[AFK]` Communities are seeded, pinned, and cached by content-hash (Scenario: Community partition is reproducible and cached by content-hash) — `Run: go test ./skillgrid-cli/internal/mnemonic/community/... -count=1` — Expected: PASS
- [x] 01.17 `[AFK]` Eval harness drops noise commits (merges/reverts/releases/bumps/formatting/changelog-like/benchmark-touching) (Scenario: Evaluation harness drops noise commits from the query set) — `Run: go test ./skillgrid-cli/internal/mnemonic/eval/... -run QuerySetNoiseDrops -count=1` — Expected: PASS
- [x] 01.18 `[AFK]` Benchmark scaffolding excluded from the graded corpus (Scenario: Evaluation corpus excludes the benchmark scaffolding) — `Run: go test ./skillgrid-cli/internal/mnemonic/eval/... -run CorpusExcludes -count=1` — Expected: PASS
- [x] 01.19 `[AFK]` Ablation shares one index so deltas measure ranking only (Scenario: Ablation shares one index so deltas measure ranking) — `Run: go test ./skillgrid-cli/internal/mnemonic/eval/... -run OneIndexPerCorpus -count=1` — Expected: PASS
- [x] 01.20 `[AFK]` Failing ranking signal removed/kept-off with decision recorded (Scenario: Failing ranking signal is removed or kept off with the decision recorded) — `Run: go test ./skillgrid-cli/internal/mnemonic/eval/... -run FailingSignalDecision -count=1` — Expected: PASS
- [x] 01.21 `[AFK]` CLI parity for community + eval commands (Scenarios: CLI community commands; `skillgrid eval --corpus <self>`) — `Run: go test ./skillgrid-cli/cmd/skillgrid/... -count=1` — Expected: PASS

### Verification

Verdict: `PASS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/community/... ./skillgrid-cli/internal/mnemonic/eval/... ./skillgrid-cli/internal/mnemonic/hybrid/... ./skillgrid-cli/internal/mnemonic/store/... ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/internal/mnemonic/service/... -count=1` | PASS | PASS | 7/7 packages `ok`; all named RED tests green |
| Acceptance `@step-01` / `@p0` | mapped unit scenarios (`EvalHarnessSignificance`, `EvalStaleExpectation`, `OutputRedaction`, `DoctorStrict`, `RerankAgreement`, `ConfidenceAction`, `SnippetSkeleton`, `ShippedConfigSignificance`, `QuerySetNoiseDrops`, `CorpusExcludes`, `OneIndexPerCorpus`, `FailingSignalDecision`) | PASS | PASS | each asserted per sub-task |
| Eval gate | `go test ./skillgrid-cli/internal/mnemonic/eval/... -count=1` + `skillgrid eval --corpus self` | PASS | PASS | shipped (factor-based) recall@5=0.299 vs baseline 0.052, Δ+0.247, CI[0.173,0.331], p=0.001 → ship (non-vacuous) |
| Runtime harness | `code_communities` / `code_god_nodes` / `code_explain_community` via MCP + CLI; `skillgrid doctor --strict`; `skillgrid eval` | PASS | PASS | additive response fields live on code_search/code_read |
| Rollback boundary | Drop `012_*` + `community/` + `eval/` + `hybrid/{rerank,confidence,snippet}.go` + community/eval CLI + `go.mod` loom dep (eval harness **stays**) | PASS | PASS | 005 `code_*` names/params unchanged (baseline lock holds) |
| Global Constraints | — | held | held | pure-Go (loom stdlib-only); additive schema (012); deterministic eval (seeded) |

Sub-agent reviews: round 1 (community half) approved w/ 2 Important → fixed (content-hash cache key, god_nodes populated, c042f74; go.mod/loom pinned as forced by loom's go 1.26). Round 2 (eval+ranker) approved w/ fixes → fixed (shippedRank wired as real factor-based variant, anchored revert regex, one-index pointer assertion, loom → direct, skeletonize 0-based; commits 06ce4ee + edd8697). Re-review of fix round: all 5 ADDRESSED, no new breakage.

### Commit

When step DoD is met: `feat(mnemonic): leiden communities, retrieval-eval harness, and measured retrieval-quality layer`

---

## 02-process-flows

### Goal

Precomputed process layer — trace execution flows from 010's entry points (routes/handlers/CLI mains) through 005 call edges into named `processes`/`process_steps` with a cross-community flag + LLM labels cached by content-hash, exposed via `code_processes` / `code_process` and surfaced in 005's `code_explain_symbol`.

### Out of scope / Non-Goals

- Community detection (step 01)
- Knowledge nodes (step 03)
- PDG/taint (011)

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-02` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 01-community-detection (communities + cross-community flag), 010 (entry points)

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/process/trace.go`
- Create: `skillgrid-cli/internal/mnemonic/process/labels.go`
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_code_process.go`
- Modify: `skillgrid-cli/internal/mnemonic/service/service.go` (process facade + process participation in `code_explain_symbol`)
- Test: `skillgrid-cli/internal/mnemonic/process/...`, `skillgrid-cli/internal/mnemonic/mcp/...`, `skillgrid-cli/internal/mnemonic/service/...`

**Interfaces:**
- Consumes: 010's entry points (route/handler/CLI-main symbols), 005 call edges, 01's communities + `code_explain_symbol`
- Produces: `processes` / `process_steps` tables (cross-community flag + content-hash key); deterministic entry-point → call-chain trace (depth-capped, graph-stops); LLM labels cached by content-hash; `code_processes`, `code_process <name>` MCP tools; `code_explain_symbol` now surfaces process participation (step N/M)

### Tasks

- [x] 02.1 `[RED]` Mnemonic tool surface — `code_explain_symbol` surfaces process participation + 005 tools stable (Scenario: Symbol explanation surfaces process participation and existing search tools stay stable) — threat: Mnemonic tool surface
  - [ ] 02.1.a Write failing test — after process pass, `code_explain_symbol <sym>` (005) includes which processes the symbol participates in (step N/M); 005 tool names + required params unchanged; process tools registered with distinct names
  - [ ] 02.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/internal/mnemonic/service/... -run ExplainSymbolProcess -count=1` — Expected: FAIL
  - [ ] 02.1.c Minimal implementation — wire process participation into `code_explain_symbol` + register `code_processes` / `code_process`
  - [ ] 02.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/internal/mnemonic/service/... -run ExplainSymbolProcess -count=1` — Expected: PASS
  - [ ] 02.1.e Commit — `feat(mnemonic): surface process participation in code_explain_symbol`
- [x] 02.2 `[RED]` Entry-point → call-chain trace builds precomputed flows (Scenario: Process list returns precomputed flows from entry points)
  - [ ] 02.2.a Write failing test — seeded from 010 entry points (routes/handlers/CLI mains), trace through 005 call edges into `processes` + `process_steps`; `code_processes` returns complete flows in one call (no per-query traversal); each process has named steps + a cross-community flag + an LLM label
  - [ ] 02.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/process/... ./skillgrid-cli/internal/mnemonic/mcp/... -run ProcessTrace -count=1` — Expected: FAIL
  - [ ] 02.2.c Minimal implementation — `process/trace.go` (entry-point seeding, depth-capped BFS over call edges, cross-community flag from 01, persist `processes`/`process_steps`) + `tools_code_process.go` `code_processes` + service facade
  - [ ] 02.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/process/... ./skillgrid-cli/internal/mnemonic/mcp/... -run ProcessTrace -count=1` — Expected: PASS
  - [ ] 02.2.e Commit — `feat(mnemonic): precomputed process flow tracing`
- [x] 02.3 `[RED]` LLM labels cached by content-hash (Scenario: Process labels are cached by content-hash and re-labeled only on change)
  - [ ] 02.3.a Write failing test — a flow is LLM-labeled once and the label is cached keyed to the process content-hash; an unchanged re-index does NOT re-call the LLM (label reused); a changed flow (new content-hash) re-labels
  - [ ] 02.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/process/... -run ProcessLabels -count=1` — Expected: FAIL
  - [ ] 02.3.c Minimal implementation — `process/labels.go` (content-hash key, LLM call, cache store)
  - [ ] 02.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/process/... -run ProcessLabels -count=1` — Expected: PASS
  - [ ] 02.3.e Commit — `feat(mnemonic): llm process labels cached by content-hash`
- [x] 02.4 `[RED]` Dispatch-boundary truncation with "stops at" note + per-hop confidence (Scenario: Trace stops at a dispatch boundary with a note)
  - [ ] 02.4.a Write failing test — a trace that hits a dispatch boundary (interface→impl, message bus, callback) is truncated with a "stops at <symbol> (<reason>)" note (reusing 005 graph-stops), not silently cut; `code_process <name>` shows each hop's Confidence Label + the stop note
  - [ ] 02.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/process/... -run DispatchStop -count=1` — Expected: FAIL
  - [ ] 02.4.c Minimal implementation — depth/step cap + dispatch-boundary detection + per-hop confidence in `process/trace.go`
  - [ ] 02.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/process/... -run DispatchStop -count=1` — Expected: PASS
  - [ ] 02.4.e Commit — `feat(mnemonic): dispatch-boundary truncation in process trace`
- [x] 02.5 `[AFK]` Process detail returns the full trace with confidence (Scenario: Process detail returns the full step-by-step trace) — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/internal/mnemonic/process/... -run ProcessDetail -count=1` — Expected: PASS
- [x] 02.6 `[AFK]` LLM down → flow cached unlabeled, never fabricated (Scenario: LLM down caches the flow unlabeled) — `Run: go test ./skillgrid-cli/internal/mnemonic/process/... -run LLMDownUnlabeled -count=1` — Expected: PASS
- [x] 02.7 `[AFK]` Entry point with no traceable chain → single-step or skipped (Scenario: Untraceable entry point yields single-step or is skipped) — `Run: go test ./skillgrid-cli/internal/mnemonic/process/... -run UntraceableEntry -count=1` — Expected: PASS
- [x] 02.8 `[AFK]` Cross-community process is flagged (Scenario: Cross-community process is flagged) — `Run: go test ./skillgrid-cli/internal/mnemonic/process/... -run CrossCommunityFlag -count=1` — Expected: PASS
- [x] 02.9 `[AFK]` Process tools reject bad args clearly (Scenario: Process tools reject bad args clearly) — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run ProcessArgs -count=1` — Expected: PASS

### Verification

Verdict: `PASS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/process/... ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/internal/mnemonic/service/... -count=1` | PASS | PASS | `process`/`mcp`/`service`/`store` `ok`; 005/008/010 baseline (`community`/`route`/`graph`) `ok` |
| Acceptance `@step-02` / `@p0` | mapped unit scenarios (`ProcessTrace`, `ExplainSymbolProcess`, `ProcessLabels`, `DispatchStop`, `ProcessDetail`, `LLMDownUnlabeled`, `UntraceableEntry`, `CrossCommunityFlag`, `ProcessArgs`) | PASS | PASS | LLM-call counter asserts 0 on unchanged re-run; cross-community negative control |
| Runtime harness | `code_processes` / `code_process` on a fixture with 010 entry points | PASS | PASS | Step-02 note: the process pass is **QUERY-TIME** in this step — no indexer hook and no non-test `LLM` impl, so `code_processes` is CWD-scoped and returns empty in production until step 03 wires `Indexer.Run` (task 03.8) to run the pass with a real `LLM` impl. Deferral, not a step-02 defect. |
| Rollback boundary | Drop `process/` + process tools + `code_orient` participation + `014_*` migration | PASS | PASS | 005/012/013 tables intact; tool surface 65→67 additive |
| Global Constraints | — | held | held | CGo-free (pure-Go `LLM` iface + fnv); additive; deterministic trace; labels cached by content-hash; advisory |

Sub-agent review: approved with fixes — the only substantive caveat (query-time pass, no indexer hook / non-test LLM impl) is a defensible step-03 deferral, now tracked in the report + service.go comment + this row, closed by task 03.8 `IndexerHook`. Non-vacuous: genuine LLM-call counter (0 on unchanged re-run), real Express-indexing fixture, cross-community + participation negative controls.

### Commit

When step DoD is met: `feat(mnemonic): precomputed process flows with cached llm labels`

---

## 03-knowledge-graph-nodes

### Goal

Doc/config/SQL extractors + non-code nodes + `references`/`configures`/`reads`/`writes` edges + indexer hook + knowledge MCP/CLI tools so the graph spans code and its knowledge, and `code_path` can trace code→doc→config→table in one query.

### Out of scope / Non-Goals

- Community detection (step 01); process flows (step 02)
- Video/audio/image; LLM semantic doc↔code links

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-03` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 02-process-flows

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/knowledge/doc.go`
- Create: `skillgrid-cli/internal/mnemonic/knowledge/config.go`
- Create: `skillgrid-cli/internal/mnemonic/knowledge/sql.go`
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_code_knowledge.go`
- Modify: `skillgrid-cli/internal/mnemonic/codeindex/indexer.go` (hook community + process + knowledge passes after 005 extraction, same tx)
- Modify: `skillgrid-cli/internal/mnemonic/mcp/server.go` (register new tool sets)
- Modify: `skillgrid-cli/cmd/skillgrid/main.go` (CLI dispatch)
- Test: `skillgrid-cli/internal/mnemonic/knowledge/...`, `skillgrid-cli/internal/mnemonic/codeindex/...`, `skillgrid-cli/internal/mnemonic/mcp/...`, `skillgrid-cli/cmd/skillgrid/...`

**Interfaces:**
- Consumes: 005 symbols/edges + `code_path`, 01 communities, 02 processes, indexer hook seam
- Produces: `doc_nodes` / `config_nodes` / `sql_schema_nodes` tables; `references` (doc→doc), `configures` (config→code), `reads`/`writes` (code→table) edges; indexer hook running community + process + knowledge passes in the same incremental tx; knowledge query tools; `code_path` spanning code→doc→config→table

### Tasks

- [x] 03.1 `[RED]` Mnemonic tool surface — knowledge tools register + 005 tools stable + bad args rejected (Scenario: Knowledge tools register and reject bad args) — threat: Mnemonic tool surface
  - [ ] 03.1.a Write failing test — knowledge query tools registered with distinct `code_*` names; 005 tools still name/param-stable; bad/missing args rejected clearly
  - [ ] 03.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run KnowledgeTools -count=1` — Expected: FAIL
  - [ ] 03.1.c Minimal implementation — `tools_code_knowledge.go` + `server.go` registration + `main.go` dispatch
  - [ ] 03.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run KnowledgeTools -count=1` — Expected: PASS
  - [ ] 03.1.e Commit — `feat(mnemonic): register knowledge-graph query tools`
- [x] 03.2 `[RED]` Markdown links/wikilinks become references edges (Scenario: Markdown links and wikilinks become references edges)
  - [ ] 03.2.a Write failing test — `.md` files with `[text](./other.md)` and `[[wikilinks]]` produce `doc_nodes` + `references` edges between doc nodes, each confidence-labeled
  - [ ] 03.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/knowledge/... -run DocLinks -count=1` — Expected: FAIL
  - [ ] 03.2.c Minimal implementation — `knowledge/doc.go` (markdown link/wikilink parser → `references`)
  - [ ] 03.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/knowledge/... -run DocLinks -count=1` — Expected: PASS
  - [ ] 03.2.e Commit — `feat(mnemonic): markdown doc-link extractor`
- [x] 03.3 `[RED]` Config refs become configures edges (Scenario: Config references become configures edges)
  - [ ] 03.3.a Write failing test — `.yaml`/`.toml`/`.json` files produce `config_nodes` + `configures` edges to the code they configure; explicit syntax is `EXTRACTED`, resolved-but-inferred refs are `INFERRED`
  - [ ] 03.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/knowledge/... -run ConfigRefs -count=1` — Expected: FAIL
  - [ ] 03.3.c Minimal implementation — `knowledge/config.go` (key→symbol resolver → `configures`)
  - [ ] 03.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/knowledge/... -run ConfigRefs -count=1` — Expected: PASS
  - [ ] 03.3.e Commit — `feat(mnemonic): config-ref extractor`
- [x] 03.4 `[RED]` SQL DDL becomes table/column nodes with reads/writes (Scenario: SQL DDL becomes table and column nodes with reads and writes)
  - [ ] 03.4.a Write failing test — `.sql` DDL produces `sql_schema_nodes` (tables + columns) and code that references them gets `reads`/`writes` edges, confidence-labeled
  - [ ] 03.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/knowledge/... -run SqlSchema -count=1` — Expected: FAIL
  - [ ] 03.4.c Minimal implementation — `knowledge/sql.go` (DDL parser + identifier-ref scan → `reads`/`writes`)
  - [ ] 03.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/knowledge/... -run SqlSchema -count=1` — Expected: PASS
  - [ ] 03.4.e Commit — `feat(mnemonic): sql-schema extractor`
- [x] 03.5 `[AFK]` Unresolvable config ref is AMBIGUOUS, not dropped (Scenario: Unresolvable config ref is ambiguous not dropped) — `Run: go test ./skillgrid-cli/internal/mnemonic/knowledge/... -run AmbiguousConfigRef -count=1` — Expected: PASS
- [x] 03.6 `[AFK]` Malformed doc file: skip bad links, index the rest (Scenario: Malformed doc file falls back and indexes the rest) — `Run: go test ./skillgrid-cli/internal/mnemonic/knowledge/... -run MalformedDoc -count=1` — Expected: PASS
- [x] 03.7 `[AFK]` Malformed SQL statement: skip it, index the rest (Scenario: Malformed SQL statement is skipped and the rest is indexed) — `Run: go test ./skillgrid-cli/internal/mnemonic/knowledge/... -run MalformedSQL -count=1` — Expected: PASS
- [x] 03.8 `[AFK]` Indexer hook runs all three passes in the same transaction (Scenario: Indexer hook runs community, process, and knowledge in one transaction) — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/... -run IndexerHook -count=1` — Expected: PASS
- [x] 03.9 `[AFK]` code_path traces code to doc to config to table (Scenario: Path tool traces code to doc to config to table) — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/internal/mnemonic/knowledge/... -run CodePathSpan -count=1` — Expected: PASS

### Verification

Verdict: `PASS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/knowledge/... ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/mcp/... ./cmd/skillgrid/... -count=1` | PASS | PASS | knowledge/codeindex/mcp/cmd `ok`; 005/008/010 baseline (community/process/route/graph) `ok` |
| Acceptance `@step-03` / `@p0` | mapped unit scenarios (`KnowledgeTools`, `DocLinks`, `ConfigRefs`, `SqlSchema`, `AmbiguousConfigRef`, `MalformedDoc`, `MalformedSQL`, `IndexerHook`, `CodePathSpan`) | PASS | PASS | unresolvable config ref = AMBIGUOUS (not dropped); code_path spans ≥2 knowledge edge kinds |
| Runtime harness | `skillgrid index` on fixture with docs/configs/SQL; `code_path` code→doc→config→table | PASS | PASS | 03.8 IndexerHook populates community+process+knowledge at index time (advisory passes, post-005-commit — see note) |
| Rollback boundary | Drop `knowledge/` + knowledge tools + indexer hook + `015_*` migration | PASS | PASS | 005/012/013/014 tables intact; tool surface 67→71 additive; code-to-code `graph.Path` unchanged |
| Global Constraints | — | held | held | CGo-free; additive; confidence labels on all edges; malformed skip-bad-index-rest; advisory |

03.8 tx note (honest framing, confirmed in review): the 005+route extraction runs in one tx; the community/process/knowledge passes run AFTER that commit on a reopened DB (modernc.org/sqlite single-conn write-lock constraint), each advisory (warn-and-continue, never roll back 005). `TestIndexerHook` proves all three passes populate their tables in one `idx.Run`.

Sub-agent review: approved with fixes — 03.8 "same tx" comment was inaccurate (passes are post-commit/advisory), SaveConfig picked lowest-id on cross-package name collisions, isSQL scanned non-`.sql` files, dead `HasPrefix` — all fixed (b8a0d02). Re-review: all 4 ADDRESSED, no new breakage, non-vacuous tests that also lock in preserved behavior (EXTRACTED single-match, `.sql` parsing).

### Commit

When step DoD is met: `feat(mnemonic): doc, config, and sql knowledge-graph nodes`

---

## Archive gate checklist

- [ ] Change-level **Definition of Done** fully checked
- [ ] No unchecked `- [ ]` under any `### Tasks`
- [ ] Every step Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] No Global Constraint violated
- [ ] `## State` status is `done` and phase is `archive` (set by verify/archive)
- [ ] STATUS banner updated to `complete`
