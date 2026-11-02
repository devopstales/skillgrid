# Tasks — mnemonic-v2 repo-map function

> **STATUS:** `sliced` (2026-11-01)

> Sliced from `.skillgrid/specs/2026-11-01-mnemonic-v2-repo-map/blueprint.md`.
> Vertical tracer-bullet tickets, dependency-ordered, sized for one fresh agent context window.
> **This change executes only after the code-index change ships** (serial development,
> AGENTS.md § Rules) — it needs the code-index `graph` package + tables AND the
> in-flight memory change's substrate.

## Epic Summary

Build the repo-map function of mnemonic v2 — a single read-only, ranked
(PageRank+IDF+context-prior), coverage-labeled, hub-flagged, token-budgeted
navigation view over the project's **code structure** (the code-index graph) **and
its accumulated memory** (the observations) — auto-injected as the agent's first-
minute context via the MCP session-start hook, re-rankable per turn in <100ms, with
a CI-able `--drift` check and a lean AGENTS.md trigger-table. **No new tables, no new
migration, no `Store` extension** (ADR-0047): the read goes through the existing
`RawQuery`/`Probe`/`ProbeInt`/`RawExec` seams + the code-index `graph` package.

The load-bearing requirement is the **7-reference fold (findings.md)**: the map
orients (it does not answer — the agent drills via `code_search`/`code_read`/
`mem_lookup`), every answer is coverage-labeled (a partial graph never reads as a
complete one), the Developer Consideration is **accumulated memory, not LLM-
generated**, and the whole read is a pure, deterministic, token-budgeted first verb.

Fast-Track: **standard** — standard classification (no new trust boundary, no public
data migration, no new dependency; the one public contract is the `repo_map` MCP tool
+ the AGENTS.md block edit, both additive; 7 tickets).

## Delivery Strategy

| Field | Value |
|-------|-------|
| Estimated changed lines | ~2,500–3,500 (7 tickets, ~35–45% test code, ~15 new files) |
| 400-line budget risk | Medium (TICKET-01 ~900 lines: the ranking engine + the thin end-to-end read + the coverage envelope, held whole as the door check) |
| Chained PRs recommended | Yes |
| Suggested split | 7 work units (W1 rank+door-check → W2 hub+memjoin → W3 kinds → W4 envelope → W5 emit/drift → W6 transport/CLI/AGENTS.md → W7 doctor/QA) |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main (serial development — one change at a time; waves execute in order; the whole change sits behind the code-index change) |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: Medium

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | PageRank+IDF+context-prior ranking engine + thin end-to-end read + coverage envelope (door check) (T1) | PR 1 | `cd mnemonic-v2 && go test ./internal/repomap/ -run 'TestDoorCheck|TestRanking|TestCoverage'` | N/A — table-driven over a seeded in-memory graph + a seeded SQLite store | `mnemonic-v2/internal/repomap/{types,rank,render,coverage}.go` |
| 2 | Hub flag + memory join (topic hubs + per-file considerations) (T2) | PR 2 | `cd mnemonic-v2 && go test ./internal/repomap/ -run 'TestHub|TestMemory'` | N/A — the fan-in + FTS join over a seeded store | `mnemonic-v2/internal/repomap/{hub,memjoin}.go` |
| 3 | Cross-graph kinds (focus/neighborhood, blast-radius, temporal) + focus-mode disambiguation (T3) | PR 3 | `cd mnemonic-v2 && go test ./internal/repomap/ -run 'TestKind|TestFocus'` | N/A — the kinds over a seeded store + the code-index `graph` package | `mnemonic-v2/internal/repomap/kinds.go` |
| 4 | Context envelope + session-start hook + per-turn re-rank (<100ms) (T4) | PR 4 | `cd mnemonic-v2 && go test ./internal/repomap/ -run 'TestContextEnvelope|TestPerTurnRerank|TestDynamicColdStart'` | N/A — the envelope over a seeded store; the <100ms bound on the in-memory adjacency | `mnemonic-v2/internal/repomap/envelope.go` |
| 5 | `--emit` doc (per-file field contract + header stamp) + `--drift` check (T5) | PR 5 | `cd mnemonic-v2 && go test ./internal/repomap/ -run 'TestEmit|TestDrift'` | A real temp-dir project root (the doc is a file) | `mnemonic-v2/internal/repomap/emit.go` |
| 6 | `repo_map` MCP tool + session-start auto-inject hook + CLI group + AGENTS.md trigger-table + onboarding re-emit (T6) | PR 6 | `cd mnemonic-v2 && go test ./internal/transport/... ./cmd/... -run 'TestRepoMap|TestAgentsMd|TestOnboardingReemit'` | Real MCP stdio round-trip via the memory server's profile mechanism + the block.md render | `mnemonic-v2/internal/transport/handler/handler_repo.go` + `mcp/tools_repo.go` + `cmd/mnemonic/repo.go` + `.agents/skills/_shared/agent-config/block.md` |
| 7 | Doctor map-health surface + QA measurability scenario (T7) | PR 7 | `cd mnemonic-v2 && go test ./internal/repomap/ -run 'TestDoctor|TestQAConsideration'` | N/A — the map-health over a seeded store | `mnemonic-v2/internal/repomap/health.go` + `cmd/mnemonic/doctor.go` |

> The four plain-text guard lines above are the **contract** — `skillgrid:subagent-execution` matches them literally. Risk Medium → `Chained PRs recommended` MUST be `Yes` and every work unit names a focused test, a runtime harness (or `N/A` + reason), and a rollback boundary.

## Tickets

> **Precondition (all tickets):** T0 checklist is complete and committed — the **code-index change is SHIPPED** (`2026-10-31-mnemonic-v2-code-index`: `internal/codeindex` + the 7 tables + the `graph` package methods `Orient`/`Impact`/`Callees`/`Callers` exist); the **memory change is SHIPPED** (`2026-10-31-mnemonic-v2-memory`: `internal/core/store` + the `RawQuery`/`Probe`/`ProbeInt`/`RawExec` seams + `internal/memory` + the `observations`/`observations_fts`/`sessions`/`session_trails` tables + `internal/transport/mcp` exist, `cd mnemonic-v2 && go build ./...` exits 0); `findings.md` committed; ADR-0047 written under `.skillgrid/artifacts/` + indexed in ASSUMPTIONS.md; `state.yaml` repointed to `2026-11-01-mnemonic-v2-repo-map` **at T0 of execution** (this spec zone does not repoint it — serial development, code-index is current until it ships); `acceptance.feature` committed **before** any `mnemonic-v2/` code commit (pre-commit zone guard, AGENTS.md § Rules).
>
> **Tracker IDs are placeholders** (`TASK-XXX`) — `skillgrid:ticketing` publishes the real IDs to the configured tracker at execution.

### TICKET-01 — PageRank+IDF+context-prior ranking engine + thin end-to-end read + coverage envelope (door check)
- **Tracker ID:** TASK-XXX  (milestone 2026-11-01-mnemonic-v2-repo-map)
- **Scope:** The new `internal/repomap/` domain package: `types.go` (the Go result/param types), `rank.go` (the deterministic PageRank — damping 0.85, max 100 iters, ε=1e-6 — over the file reference graph, edges weighted by reference count × IDF(symbol name); the `context` param is a PageRank prior; `--focus` is a boost (path) or neighborhood (symbol)); `render.go` (the three-tier renderer: the signature-line walk, the token-budget stop, the dynamic cold-start budget — `inject_tokens` default 1024, expanded ~2× on cold start); `coverage.go` (the versioned per-answer envelope — `status: complete|partial|unavailable`, `sources[]` naming the per-language tier (D-TIER), `issues[]` naming the broken leg + stale subtree). **The door check** — if a `repo map` cannot rank a seeded code-index graph (deterministic, IDF, context prior) and emit a token-budgeted signature tree with a coverage envelope over a seeded memory store, stop: the repo-map's foundation does not hold.
- **Acceptance:** `go test ./internal/repomap/ -run 'TestDoorCheckRepoMapReadsAndRanks|TestRankingIDFDownweightsCommon|TestRankingContextPriorRaises|TestCoveragePartialTimedOutLeg|TestCoverageUnavailableBothMissing'` passes — the rank is deterministic (two runs, identical order); the IDF down-weights the common name; the context prior raises the rank (delta > 0); the walk stops at the token budget; each row carries the full signature line; the coverage envelope is `complete`/`partial`/`unavailable` per the leg status.
- **SATISFIES:** door-check-repo-map, ranking-deterministic, ranking-idf-downweights-common, ranking-context-prior-raises, signature-line-output, token-budget-walk-stops, coverage-complete, coverage-partial-timed-out-leg, coverage-unavailable-both-missing
- **Files:** `mnemonic-v2/internal/repomap/{types,rank,render,coverage}.go` + `{rank,render,door}_test.go`
- **Size:** ~900 (L) — held whole (the ranking engine + the two-leg read + the coverage envelope are the hypothesis being falsified)
- **Blocks:** TICKET-02, TICKET-03, TICKET-04, TICKET-05
- **Blocked by:** none (code-index + memory shipped)
- **Reversibility:** reversible (the domain package is pure logic over the seams + the `graph` package; no table, no migration, no `Store` method — ADR-0047)
- **Fails-when:** `TestDoorCheckRepoMapReadsAndRanks` does not emit a deterministic, token-budgeted signature tree with a coverage envelope, OR `TestRankingContextPriorRaises` shows a rank delta ≤ 0 (the aider fold broken).

### TICKET-02 — Hub flag + memory join (topic hubs + per-file considerations)
- **Tracker ID:** TASK-XXX  (milestone 2026-11-01-mnemonic-v2-repo-map)
- **Scope:** The hub flag (`hub.go`): files with fan-in ≥ `hub_threshold` (config, default **3** importers) carry a `⚠️ HUB` tag + importer count in the orientation view (the JordanCoin fold). The memory join (`memjoin.go`): the **query-time FTS join** of `observations_fts` (title/content/topic_key) onto the file's path/package → the per-file Developer Consideration (the cyanheads fold, memory as the source; the deferred v2.1 door is the stored `code_ref` pointer); a file with no match gets no consideration line (not fabricated). `TopicHubs`: the `topic_key` tree → typed hubs → top-N by `importance`.
- **Acceptance:** `go test ./internal/repomap/ -run 'TestHubFlag3Importers|TestHubThresholdConfig|TestMemoryJoinSurfacesSeededObservation|TestMemoryTopicHubsByImportance|TestMemoryJoinAbsentNotFabricated'` passes — a 3-importer file is flagged + counted, a 2-importer file is not (until the threshold is lowered to 2); a seeded observation for a seeded file surfaces as its consideration; the topic hubs are ordered by importance; an empty memory store means no fabricated consideration.
- **SATISFIES:** hub-flag-3-importers, hub-threshold-config, memory-join-surfaces-seeded-observation, memory-topic-hubs-by-importance, memory-join-absent-not-fabricated
- **Files:** `mnemonic-v2/internal/repomap/{hub,memjoin}.go` + modify `render.go` (the hub tag + the consideration line) + `{hub,memjoin}_test.go`
- **Size:** ~450 (M)
- **Blocks:** TICKET-04, TICKET-05
- **Blocked by:** TICKET-01 (the `RankedFile` shape + the seam reads)
- **Reversibility:** reversible (the fan-in flag + the FTS join are pure reads over the seams)
- **Fails-when:** `TestHubFlag3Importers` flags a 2-importer file at the default threshold (the threshold is not strict ≥3), OR `TestMemoryJoinAbsentNotFabricated` fabricates a consideration for a file with no observation.

### TICKET-03 — Cross-graph kinds (focus/neighborhood, blast-radius, temporal) + focus-mode disambiguation
- **Tracker ID:** TASK-XXX  (milestone 2026-11-01-mnemonic-v2-repo-map)
- **Scope:** The `kind` dispatch (`kinds.go`): `overview` (default: packages → file/symbol counts → topic hubs → recent decisions), `packages` (the package tree + import edges), `symbols` (the per-package symbol TOC with kinds), `topics` (the memory map), **`focus`/neighborhood** (a `--focus <symbol>` cross-graph join: the symbol's `Callers`/`Callees` + the observations that reference it), **`blast-radius`** (the code reverse-BFS via the existing `graph.Impact` + "which observations reference this"), **`temporal`** (weekly bins: code re-indexes from `files.indexed_at` + memory growth from `observations.created_at`). The **focus-mode disambiguation**: the CLI maps a `--focus` arg to `Mode: boost` (a path) or `Mode: neighborhood` (a symbol name); the MCP tool takes an explicit `mode` param.
- **Acceptance:** `go test ./internal/repomap/ -run 'TestKindBlastRadiusCrossGraph|TestKindTemporalWeeklyBins|TestFocusModeDisambiguation'` passes — a `blast-radius` kind returns the code reverse-BFS set + the observations that reference the symbol; a `temporal` kind returns weekly bins for both legs; a `focus <symbol>` (neighborhood) is distinct from a `focus <path>` (boost).
- **SATISFIES:** kind-blast-radius-cross-graph, kind-temporal-weekly-bins, focus-mode-disambiguation
- **Files:** `mnemonic-v2/internal/repomap/kinds.go` + `kinds_test.go`
- **Size:** ~400 (M)
- **Blocks:** TICKET-04
- **Blocked by:** TICKET-01 (the `rank.Run`), TICKET-02 (the `memjoin` for the cross-graph observations)
- **Reversibility:** reversible (the kind dispatch is pure logic over the seams + the `graph` package)
- **Fails-when:** `TestKindBlastRadiusCrossGraph` returns the code set without the observations (the cross-graph join broken), OR `TestFocusModeDisambiguation` returns the same result for a symbol and a path focus.

### TICKET-04 — Context envelope + session-start hook + per-turn re-rank (<100ms)
- **Tracker ID:** TASK-XXX  (milestone 2026-11-01-mnemonic-v2-repo-map)
- **Scope:** The **ONE** `ContextEnvelope` (`envelope.go`): the map (Tier 1+2, token-budgeted at `inject_tokens`, **expanded ~2× on cold start**) + the coverage contract + the hub warnings + a memory summary (top-3 recent decisions + topic hubs) + the `--for <intent>` prior + the compact/full budget marker (the JordanCoin context-protocol fold). The **per-turn re-rank**: the `rank.Run` re-run with the new context prior, **<100ms** (no re-parse — the index is unchanged; the aider per-turn fold). This is what gets auto-injected (the session-start hook, wired in TICKET-06).
- **Acceptance:** `go test ./internal/repomap/ -run 'TestContextEnvelopeOneCall|TestPerTurnRerankUnder100ms|TestDynamicColdStartBudget'` passes — the session-start call returns exactly one envelope (map + coverage + hubs + memory summary + intent prior); a per-turn re-rank after a new file is read completes <100ms and raises that file's rank; the cold-start inject budget is ~2× the mid-session budget.
- **SATISFIES:** context-envelope-one-call, per-turn-rerank-under-100ms, dynamic-cold-start-budget
- **Files:** `mnemonic-v2/internal/repomap/envelope.go` + `envelope_test.go`
- **Size:** ~450 (M)
- **Blocks:** TICKET-06
- **Blocked by:** TICKET-01 (the `rank.Run` + `render` + `coverage`), TICKET-02 (the `hub` + `memjoin`), TICKET-03 (the `kinds` for the map)
- **Reversibility:** reversible (the envelope is a pure composition over the legs)
- **Fails-when:** `TestPerTurnRerankUnder100ms` exceeds the bound (a re-parse crept in), OR `TestDynamicColdStartBudget` shows the cold-start budget equal to the mid-session budget (the aider dynamic-budget fold broken).

### TICKET-05 — `--emit` doc (per-file field contract + header stamp) + `--drift` check
- **Tracker ID:** TASK-XXX  (milestone 2026-11-01-mnemonic-v2-repo-map)
- **Scope:** The emitted doc (`emit.go`): `--emit` writes `.skillgrid/repo-map.md` (relative to the project root) with the **cyanheads per-file field contract** (Description = the signature-line floor; Developer Consideration = the `memjoin` observation; Maintenance Flag (Stable/Volatile/Generated/Unknown) from `files.content_hash` churn + the observation type mix; Architectural Role (Entrypoint/Service/API Route/Utility/Test/Configuration/…) from `edges` in/out-degree + `kind`; Critical Dependencies = top-N by confidence rank × fan-in, with one-line reasons) + a **header stamp** (`generated_at` + a content hash). **Gitignored by default**; the config flag `repo_map.commit: true` opts in. **`--drift`** (the agentmd fold): exit 0 = doc fresh, exit 1 = doc stale vs. live state (stamp + content-hash comparison), CI-able JSON.
- **Acceptance:** `go test ./internal/repomap/ -run 'TestEmitDocPerFileFields|TestDriftExitCodes'` passes — the doc has all five per-file fields + the header stamp; the doc is gitignored by default and committed when `repo_map.commit` is true; `--drift` exits 0 on a fresh doc, 1 after the tree changes (the hash mismatch), and emits valid CI JSON.
- **SATISFIES:** emit-doc-per-file-fields, drift-exit-codes
- **Files:** `mnemonic-v2/internal/repomap/emit.go` + `emit_test.go`
- **Size:** ~350 (M)
- **Blocks:** TICKET-06 (the CLI `--emit`/`--drift` flags)
- **Blocked by:** TICKET-01 (the `render`), TICKET-02 (the `memjoin` for the consideration)
- **Reversibility:** reversible (the doc is a derived file; the drift check is a pure comparison)
- **Fails-when:** `TestDriftExitCodes` exits 0 after the tree changes (the drift check is not comparing the hash), OR `TestEmitDocPerFileFields` is missing a field (the cyanheads contract broken).

### TICKET-06 — `repo_map` MCP tool + session-start auto-inject hook + CLI group + AGENTS.md trigger-table + onboarding re-emit
- **Tracker ID:** TASK-XXX  (milestone 2026-11-01-mnemonic-v2-repo-map)
- **Scope:** The one shared `handler_repo.go` handler set (the MCP + CLI meet here; never diverge); the `repo_map` MCP tool (JSON; params `mode`/`kind`/`focus`/`tokens`/`context`/`for`) on the **agent profile** (the memory change's two-profile mechanism); the **session-start auto-injection hook** (the `Envelope` call, the primary path, opt-in via the config `repo_map.auto_inject` flag); the `mnemonic repo map` CLI group (`--kind`/`--focus`/`--mode`/`--tokens`/`--context`/`--for`/`--emit`/`--drift`/`--json`); the **AGENTS.md trigger-table** in `.agents/skills/_shared/agent-config/block.md` (a ≤6-line `### Code Map` sub-block: a When → Command table + a one-line trust rule, NOT a prose paragraph, no new top-level section — the MCP hook is the PRIMARY path, the table is the FALLBACK for harnesses without the hook); the **onboarding re-emit** (when the code-index freshness flips `fresh→lag` or the memory schema version bumps, the onboarding CLI re-runs `emit.Emit`). Every `repo_map` call records ONE retrieval-trail row (INV-5).
- **Acceptance:** `go test ./internal/transport/... ./cmd/... -run 'TestRepoMapMcpCliParity|TestAgentsMdLeanTriggerTable|TestOnboardingReemitOnLag|TestSessionStartAutoInjectFlag'` passes — the MCP + CLI produce byte-identical JSON for the same args; the block.md `Code Map` sub-block is ≤6 lines (no prose paragraph, no new top-level section); onboarding re-emits the doc on the `fresh→lag` flip; auto-injection is off when the config flag is off.
- **SATISFIES:** mcp-cli-byte-identical, agents-md-lean-trigger-table, onboarding-reemit-on-lag, session-start-auto-inject-flag
- **Files:** `mnemonic-v2/internal/transport/handler/handler_repo.go` + `internal/transport/mcp/tools_repo.go` + `cmd/mnemonic/repo.go` + `.agents/skills/_shared/agent-config/block.md` + tests
- **Size:** ~700 (L)
- **Blocks:** none (the final transport)
- **Blocked by:** TICKET-01, TICKET-04 (the `Envelope` for the hook), TICKET-05 (the `--emit`/`--drift` CLI flags)
- **Reversibility:** one-way (the `repo_map` MCP tool contract — the opencode/pi plugins consume it; a rename / required param / return-shape change after they build breaks them). The AGENTS.md block edit is soft (a re-render, not a data contract).
- **Fails-when:** `TestRepoMapMcpCliParity` shows divergent JSON (the shared-handler seam broken), OR `TestAgentsMdLeanTriggerTable` shows a >6-line block or a prose paragraph (the block.md "Keep this block lean" rule broken).

### TICKET-07 — Doctor map-health surface + QA measurability scenario
- **Tracker ID:** TASK-XXX  (milestone 2026-11-01-mnemonic-v2-repo-map)
- **Scope:** The doctor map-health surface (`health.go`): the `doctor` command (the memory change's existing surface — this change adds the line, not a new command) reports the map health (the coverage status of the last map read + the leg status: code-index present/lag/absent, memory present/empty). **The QA measurability scenario** (the scenario the brief names): a seeded observation for a seeded file surfaces in the map's consideration join (the end-to-end proof the memory join works against a real store).
- **Acceptance:** `go test ./internal/repomap/ -run 'TestDoctorMapHealth|TestQAConsiderationJoinSeeded'` passes — `doctor` reports the coverage status + the leg status; a seeded observation for a seeded file surfaces in the map output's consideration join.
- **SATISFIES:** doctor-map-health, qa-consideration-join-seeded
- **Files:** `mnemonic-v2/internal/repomap/health.go` + modify `cmd/mnemonic/doctor.go` + `health_test.go`
- **Size:** ~200 (M)
- **Blocks:** none (the final backstop)
- **Blocked by:** TICKET-01 (the `coverage`), TICKET-02 (the `memjoin` for the QA scenario)
- **Reversibility:** reversible (the map-health is a pure read; the doctor line is additive)
- **Fails-when:** `TestQAConsiderationJoinSeeded` does not surface the seeded observation in the map output (the memory join's end-to-end proof broken).

## Execution Notes

- **Waves (serial):** W1 (T1) → W2 (T2) → W3 (T3) → W4 (T4) → W5 (T5) → W6 (T6) → W7 (T7). TICKET-02 and TICKET-03 both need T1 (and 02 needs the `memjoin` before 03's cross-graph join); the wave order above is the default. TICKET-05 (emit) can run in parallel with TICKET-04 (envelope) since both only need T1+T2; the wave order is the default.
- **The 7-reference fold is the acceptance backstop** (findings.md): the coverage contract (JordanCoin, TICKET-01/06), the hub flag (JordanCoin, TICKET-02), the signature lines + dynamic budget + per-turn re-rank (aider, TICKET-01/04), the PageRank+IDF + focus-as-boost (openclaude, TICKET-01/03), the per-file field contract (cyanheads, TICKET-05), the minimal-mode + drift (agentmd, TICKET-05), the boundary (SarthakMishra, the whole change). These are the difference between "the tool exists" and "the map orients well."
- **The boundary (the SarthakMishra fold):** the map orients; the agent drills. No RAG chat, no commits, no PRs, no second vector index. The `repo_map` tool is the first verb, not the whole composition.
- **No one-way structural doors.** The two deferred soft doors (a stored per-file LLM note fallback (`file_notes`); stored `code_ref` pointers instead of query-time FTS) are named in the blueprint + the overview, not built (v2.1). The only one-way door is the `repo_map` MCP tool contract (TICKET-06) — flag for explicit user approval at execution.
- **The substrate facts to pin at T0 (before any code commit):** the code-index `graph` package method signatures (`Orient`/`Impact`/`Callees`/`Callers`) — if a method is missing, the repo-map reads the `edges` table through the `RawQuery` seam directly (ADR-0047 still holds — no `Store` extension). The memory `observations`/`observations_fts` column set (confirmed against the in-flight memory change's `001_schema.sql`). The next ADR number (0047, confirmed against the code-index change's 0043–0046 claim).
- **ADR gates:** ADR-0047 (the repo-map read surface is a view, not a store extension) before TICKET-01.
- **Serial development:** the whole change sits behind the code-index change. `state.yaml` `pipeline.current_change` is repointed to `2026-11-01-mnemonic-v2-repo-map` at T0 (after the code-index change ships).
- **QA gate (per skillgrid:qa):** L3 — build + the `repomap` + `transport` + `cmd` package tests + go vet; mutation/security advisory (Trivy advisory-only). The QA measurability scenario (`TestQAConsiderationJoinSeeded`) + the coverage contract scenarios are the relevance backstops the QA gate must see green.
- **Definition of Done (per AGENTS.md § Rules):** the `repo_map` surface is on the two-profile MCP mechanism, the map is a pure read (no table, no migration, no `Store` method), the coverage contract holds on every answer, the ranking is deterministic, the hub flag is at the threshold, the memory join surfaces the seeded observation, the `--drift` exit codes are CI-able, and every degradation is explicit (a broken leg degrades out + is labeled, never a silent gap).
