# Broken Windows

> Cross-change defect register. Advisory findings (WARNING/SUGGESTION) not
> fixed before archive. Auto-appended by `skillgrid:qa` at gate-render time.
> Waived items are tracked here until fixed or explicitly closed.
>
> **Lifecycle:** `qa` appends open findings → `ship` archives the change
> (windows stay) → future changes' `qa` surfaces open windows as context.

| Window | Change | Severity | Finding | Waived By | Date | Status |
|--------|--------|----------|---------|-----------|------|--------|
| W001 | 2026-09-24-mnemonic-memory-improvements | WARNING | handleMemSearch drops an EmbedQuery error with no test that enters that branch | qa-auto | 2026-10-02 | closed |
| W002 | 2026-09-24-mnemonic-memory-improvements | WARNING | six tickets have no separable RED commit (strict-TDD off) | qa-auto | 2026-10-02 | open |
| W003 | 2026-09-24-mnemonic-memory-improvements | WARNING | qa-gate.mjs state-drift and size-budget scripts not found | qa-auto | 2026-10-02 | closed |
| W004 | 2026-09-24-mnemonic-memory-improvements | SUGGESTION | Service decayCfg comment says zero config enables decay; blend reranks only when Enabled is true | qa-auto | 2026-10-02 | open |
| W005 | 2026-09-24-mnemonic-memory-improvements | SUGGESTION | go vet context leak in budget.go:114 is pre-existing | qa-auto | 2026-10-02 | open |
| W006 | 2026-09-24-mnemonic-memory-improvements | SUGGESTION | private-row hiding has a single test | qa-auto | 2026-10-02 | open |
| W007 | 2026-09-24-mnemonic-memory-improvements | SUGGESTION | qa-gate.mjs looks in .agents/skills/qa/scripts; helpers live under verification/qa/scripts | qa-auto | 2026-10-02 | open |
| W008 | 2026-10-02-mnemonic-webui-rewrite | SUGGESTION | GraphPage has no mounted test; G4 is a source-text grep | qa-auto | 2026-10-02 | closed |
| W009 | 2026-10-02-mnemonic-webui-rewrite | SUGGESTION | lib/api.ts and lib/apiBase.ts have no direct unit test | qa-auto | 2026-10-02 | closed |
| W010 | 2026-10-02-mnemonic-webui-rewrite | SUGGESTION | dompurify 3.4.15 → 3.4.16 (GHSA-p98j-92pf-mc4p, LOW, not reachable) | qa-auto | 2026-10-02 | open |
| W011 | 2026-10-02-mnemonic-webui-rewrite | SUGGESTION | L3 browser walk of /mnemonic/graph against skillgrid serve was not run | qa-auto | 2026-10-02 | open |
| W012 | 2026-10-02-mnemonic-memory-checkpoint | WARNING | G7 OpenCode/Kilo plugin not-due and no-server paths are manual-only | qa-auto | 2026-10-02 | closed |
| W013 | 2026-10-02-mnemonic-memory-checkpoint | WARNING | Live Cursor stop door G4/G5 EVIDENCE not recorded (script contract covered by test-hooks) | qa-auto | 2026-10-02 | closed |
| W014 | 2026-10-02-mnemonic-llm-provider | WARNING | acceptance.feature gate ids G4-G8 do not match as-built test names (qa-gate.mjs would report gate test missing) | qa-auto | 2026-10-05 | open |
| W015 | 2026-10-02-mnemonic-llm-provider | WARNING | TDD evidence STALE - change uncommitted in working tree, no RED/GREEN commit hashes resolve | qa-auto | 2026-10-05 | open |
| W016 | 2026-10-03-local-ollama-models | WARNING | briefing reqs 3/5 (per-kind HTTP smoke + dimension recording + mnemonic.ollama.models list) descoped to a no-op smokeProbe hook / omitted in the executed blueprint; invisible to acceptance.feature | qa-auto | 2026-10-06 | closed |
| W017 | 2026-10-03-local-ollama-models | SUGGESTION | state.yaml line 10 malformed YAML (nested mapping in compact mapping) breaks state-drift check (pre-existing) | qa-auto | 2026-10-06 | open |
| W018 | 2026-10-06-kanban-milestones | WARNING | hooks.ts milestones-changed listener has no unit test driving a synthetic SSE event to assert a re-fetch (source-verified only) | qa-auto | 2026-10-06 | closed |
| W019 | 2026-10-06-kanban-milestones | SUGGESTION | GitHub/GitLab/Jira Milestones() empty-return adapters have no dedicated unit test (low risk, no milestone files for these providers) | qa-auto | 2026-10-06 | closed |
| W020 | 2026-10-07-replace-opencode-hooks-with-plugins | WARNING | 5 native TS plugins have no runtime route-contract test; a path typo would 404 silently in the live harness | qa-auto | 2026-10-08 | closed |
| W021 | 2026-10-07-replace-opencode-hooks-with-plugins | WARNING | FindRepoRoot negative case (no new plugin files -> error, S20) is UNTESTED | qa-auto | 2026-10-08 | closed |
| W022 | 2026-10-07-replace-opencode-hooks-with-plugins | WARNING | S23 (pnpm test) PRESENT_BEHAVIOR_UNVERIFIED - no pnpm harness for the 5 plugins | qa-auto | 2026-10-08 | closed |
| W023 | 2026-10-07-replace-opencode-hooks-with-plugins | SUGGESTION | Trivy HIGH golang.org/x/text CVE-2026-56852 (no fix) - pre-existing transitive dep in mnemonic/go.mod, outside this diff | qa-auto | 2026-10-08 | closed |
| W024 | 2026-10-07-replace-opencode-hooks-with-plugins | SUGGESTION | ADR-0032/0033/0034/0035/0036 use bold-list frontmatter but check-adr-invariants.mjs expects YAML frontmatter | qa-auto | 2026-10-08 | closed |
| W025 | 2026-10-07-replace-opencode-hooks-with-plugins | WARNING | state.yaml long context values (lines 10-16) unquoted -> malformed YAML, breaks state-drift (W017 pre-existing, now fixed + quoted) | qa-auto | 2026-10-08 | closed |
| W026 | 2026-10-10-obs-split-hnsw | WARNING | governance write path (decision answers, review marks) reads through compat view; 404 on freshly-saved obs (6 http + 1 mcp test) | qa-auto | 2026-10-11 | closed (c6cab1d9: session_promotion + review_cycle write split tables directly) |
| W027 | 2026-10-10-obs-split-hnsw | WARNING | observations_fts FTS5 content='observations' (the view) -> delete-by-terms SQLITE_CORRUPT (TestStatusExplicit) | qa-auto | 2026-10-11 | closed (c6cab1d9: migration 059 -> standalone FTS5 + fts*Obs DELETE+rebuild/UPDATE-in-place) |
| W028 | 2026-10-10-obs-split-hnsw | WARNING | P1 pass rate ~93% < 95% threshold (3 pkgs: http/mcp/service — W026/W027 knock-ons) | qa-auto | 2026-10-11 | closed (c6cab1d9: P1 pass rate 100%) |
| W029 | 2026-10-10-obs-split-hnsw | WARNING | TestHnswWarmLatency asserts <10ms warm; ~19ms on loaded host (passes isolated) | qa-auto | 2026-10-11 | open |
| W030 | 2026-10-10-obs-split-hnsw | SUGGESTION | cmd/mnemonic init_cmd_test.go blockMDPath relative depth wrong since cmd move a78dab58 (pre-existing) | qa-auto | 2026-10-11 | open |
| W031 | 2026-10-10-obs-split-hnsw | SUGGESTION | 92 pre-existing gofmt drift files in mnemonic module (not touched by this change) | qa-auto | 2026-10-11 | open |
| W032 | 2026-10-08-context-orchestrator | WARNING | skillgrid-cli/internal/install TestMemoryAndPersonasBoundary + TestSetupAgentsIntegration fail (missing relative plugins/opencode/skillgrid-context.ts in test tmp project) — pre-existing, outside CTX diff (CTX staged zero skillgrid-cli files) | qa-auto | 2026-10-14 | closed (306351bc: seeded 6th plugin in both fixtures; root cause = 2457a453 wired the 6th plugin, fixtures kept the old 5) |
| W033 | 2026-10-08-context-orchestrator | SUGGESTION | cmd/mnemonic/ctx.go:84-92 ~ 137-145 exact clone (9 lines, 97 tokens) — runCtx verb-dispatch shape duplicated; extract helper when adding verbs | qa-auto | 2026-10-14 | open |
| W034 | 2026-10-08-context-orchestrator | WARNING | spec F1: `ctx metrics` + `ctx traffic` CLI verbs named in tasks.md Unit 5 / blueprint 185-186 but unimplemented (HTTP surfaces /metrics + /stats cover the intent) — follow-on ticket | review | 2026-10-14 | open |
| W035 | 2026-10-08-context-orchestrator | SUGGESTION | spec F4b: Live-view header mode reflection not wired (UI concern; /stats Mode field met); F3: session summary is tick-driven, not SessionEnd-hook enqueue (row semantics met) | review | 2026-10-14 | open |
| W036 | 2026-10-08-context-orchestrator | SUGGESTION | SEC2: non-loopback proxy bind is warn-only (opt-in by design); consider an explicit ack env var as defense-in-depth | review | 2026-10-14 | open |
| W037 | 2026-10-10-obs-split-hnsw | WARNING | SEC1: FTS5 'rebuild' command in ftsDeleteObs + INSTEAD OF DELETE trigger re-reads the FTS table's own rows — a surviving core row without a FTS row loses its index entries on every delete (mem_search blind to it) | review | 2026-10-14 | closed (rebuild now SELECTs from live observations_core; regression test TestFtsDeleteSurvivingRowsStayIndexed) |
| W038 | 2026-10-10-obs-split-hnsw | WARNING | SEC2: splitSQLStatements inTrig state word-matches END without tracking string literals — a future trigger body with 'END'/'BEGIN' in a literal shreds the migration into a half-applied store that re-fails every open (latent) | review | 2026-10-14 | closed (inS sub-state applied inside inTrig; regression test TestSplitSQLStatementsTriggerLiteralEnd) |
| W039 | 2026-10-10-obs-split-hnsw | WARNING | S1: ANN adapters (HNSW/IVF) have no production writer — indexer embedPass still calls UpsertSymbol/UpsertChunk (vec0+BLOB) only; HnswSymbols/HnswChunks degrade gracefully (ok=false) but the graph is never populated; follow-up ticket: route embedPass dual-write through the cached HnswIndex/IVF upserts (carries S2 IVF wiring, S3 hnsw_meta drop-or-use, S5 real-model-through-Upsert, S6-R4 dim unpin + cross-dim typed error) | review | 2026-10-14 | open |
| W040 | 2026-10-24-global-llm-wiki | SUGGESTION | gofmt pre-existing drift in service/parallel_search_many_test.go, retrieval.go, teams.go (not introduced by this change) | qa-auto | 2026-10-25 | open |
| W041 | 2026-10-24-global-llm-wiki | SUGGESTION | mcp full-package test suite times out at 600s (pre-existing slow codeindex fixture TestCodeSearchAdditiveGains) | qa-auto | 2026-10-25 | open |
