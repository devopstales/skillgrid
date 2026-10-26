---
okf_version: "0.2"
---

# Index

Root catalog of the `.skillgrid/knowledge/` bundle (OKF v0.2). Durable project knowledge, organized by folder; every concept is listed here or in its directory catalog. Sibling documents outside the five concept zones — `ARCHITECTURE.md` (live system design), `ASSUMPTIONS.md` (state of the project: VERIFIED/INFERRED/LOCKED), `reverse-index.md` (generated), `log.md` (changelog) — are bundle documents, not linted concepts. `WINDOWS.md` (defect register), `specs/`, `xarchive/`, `sdd/`, `cache/`, `prototypes/`, `config.yaml`, `state.yaml`, and `config.d/` are working state, config, and SDD artifacts at the `.skillgrid/` root, outside the OKF concept set (ADR-0048; dissolved into `knowledge/` by the global-llm-wiki 3-tier layout — ADR-0050/0051).

## ARCHITECTURE.md + ASSUMPTIONS.md — living documents (not linted concepts)

- [ARCHITECTURE.md](ARCHITECTURE.md) — Skillgrid v2 system architecture (components, store, transports, SDD state zone)
- [ASSUMPTIONS.md](ASSUMPTIONS.md) — State of the project: VERIFIED / INFERRED / LOCKED (the ADR index is § LOCKED)

## decisions/ — locked decisions (ADRs) + ADR index

The ADR index is the in-force table in `ASSUMPTIONS.md` § LOCKED (one pointer row per decision; the body is the linked file, ADR-0019). `decisions/03-adr-index.md` is the zone's topic note for that table. Numbering: gaps 0040/0041 are retired numbers, never reused.

- [03-adr-index.md](decisions/03-adr-index.md) — ADR Index
- [04-adr-0001-prd-scope-whole-hub.md](decisions/04-adr-0001-prd-scope-whole-hub.md) — PRD scope is the whole Hub Product, not the binary alone
- [04-adr-0002-prd-engine-first-framing.md](decisions/04-adr-0002-prd-engine-first-framing.md) — PRD framing is engine-first, not installer-first
- [04-adr-0003-v1.0-line-engine-trustworthy.md](decisions/04-adr-0003-v1.0-line-engine-trustworthy.md) — v1.0 line is "the engine is trustworthy"; UI is Phase 2
- [04-adr-0004-four-component-decomposition.md](decisions/04-adr-0004-four-component-decomposition.md) — PRD component map is a 4-way decomposition
- [04-adr-0005-trust-boundary-mcp-spawn-model.md](decisions/04-adr-0005-trust-boundary-mcp-spawn-model.md) — Trust boundary: MCP-spawned process with project read access
- [04-adr-0006-vector-search-in-memory-brute-force.md](decisions/04-adr-0006-vector-search-in-memory-brute-force.md) — Vector search: in-memory brute-force cosine, no SQLite vector extension (amended by 0009)
- [04-adr-0007-sdd-multi-method-docs-viewer.md](decisions/04-adr-0007-sdd-multi-method-docs-viewer.md) — SDD multi-method docs viewer with generic, existence-gated doc roots
- [04-adr-0008-state-drift-guard-js-yaml.md](decisions/04-adr-0008-state-drift-guard-js-yaml.md) — State drift guard: Node.js + `yaml` package, read-only verifier
- [04-adr-0009-vector-search-in-sql-latency-viant-deferred.md](decisions/04-adr-0009-vector-search-in-sql-latency-viant-deferred.md) — Vector search: in-SQL sqlite-vec latency corrected; viant deferred; revisit path confirmed
- [04-adr-0010-verification-scope-discriminator.md](decisions/04-adr-0010-verification-scope-discriminator.md) — Verification scope discriminator: a zero is never a bare zero
- [04-adr-0011-observations-are-bitemporal.md](decisions/04-adr-0011-observations-are-bitemporal.md) — Observations are bi-temporal; the save path classifies each write as Add/Update/Delete/Noop
- [04-adr-0012-sqlite-as-second-brain.md](decisions/04-adr-0012-sqlite-as-second-brain.md) — SQLite as the second-brain store; llm-wiki markdown files rejected
- [04-adr-0013-repo-source-of-truth.md](decisions/04-adr-0013-repo-source-of-truth.md) — Source of truth: the repo is authoritative; Backlog task ID ↔ commit SHA is the linkage
- [04-adr-0014-removed.md](decisions/04-adr-0014-removed.md) — (removed 2026-09-30)
- [04-adr-0015-removed.md](decisions/04-adr-0015-removed.md) — (removed 2026-09-30)
- [04-adr-0016-second-brain-capability-layer.md](decisions/04-adr-0016-second-brain-capability-layer.md) — Mnemonic second-brain capability layer
- [04-adr-0017-d3-force-graph.md](decisions/04-adr-0017-d3-force-graph.md) — Embedded SPA code graph uses D3 force layout (replaces Sigma)
- [04-adr-0018-mem-search-additive-signals.md](decisions/04-adr-0018-mem-search-additive-signals.md) — mem_search returns additive per-signal scores; RRF runs only on the owner-scoped path
- [04-adr-0019-decisions-are-files.md](decisions/04-adr-0019-decisions-are-files.md) — Locked decisions are ADR files; ASSUMPTIONS.md holds the path
- [04-adr-0020-sdd-ledger-owns-execution.md](decisions/04-adr-0020-sdd-ledger-owns-execution.md) — Execution coordination lives in the SDD ledger; Mnemonic stores an index
- [04-adr-0021-pre-tool-policy-fail-open.md](decisions/04-adr-0021-pre-tool-policy-fail-open.md) — Pre-tool policy is opt-in, first-match field rules, and fails open
- [04-adr-0022-host-agent-memory-checkpoint.md](decisions/04-adr-0022-host-agent-memory-checkpoint.md) — The host agent is the memory observer; checkpoints are server-gated
- [04-adr-0023-openai-compatible-llm-provider.md](decisions/04-adr-0023-openai-compatible-llm-provider.md) — Mnemonic chat LLM is OpenAI-compatible HTTP only; one client attaches all seams
- [04-adr-0024-central-pg-shared-memory-state.md](decisions/04-adr-0024-central-pg-shared-memory-state.md) — Team-shared memory state: local SQLite + outbox journal against a VM PostgreSQL (LWW + tombstones, tokens-as-roles, schema-per-project)
- [04-adr-0025-context-harness-owner.md](decisions/04-adr-0025-context-harness-owner.md) — Context Harness owns the session context lifecycle (absorbs `session_inject`; capture, `ctx_query`, routing, `ctx` CLI)
- [04-adr-0026-indexed-files-reuses-observations-schema.md](decisions/04-adr-0026-indexed-files-reuses-observations-schema.md) — `indexed_files` reuses the observations schema shape; `ctx_search` fuses sandbox + indexed via RRF
- [04-adr-0027-clm-context-language-model.md](decisions/04-adr-0027-clm-context-language-model.md) — Context Harness implements the Context Language Model (mirror + overflow guard + calibration; Go owns state, Node owns request path; opt-in via `clm:` config)
- [04-adr-0028-context-revisions-session-scoped.md](decisions/04-adr-0028-context-revisions-session-scoped.md) — `context_revisions` is session-scoped audit (purged at session end; raw history + mirror are the source of truth)
- [04-adr-0029-compaction-advisory-steering.md](decisions/04-adr-0029-compaction-advisory-steering.md) — Compaction advisory gate (one combined LLM call, fail-open) + `steering` column on `context_revisions` + proactive build; extends ADR-0027/0028; opt-in via `mnemonic.compaction` config
- [04-adr-0030-scan-findings-structured-store.md](decisions/04-adr-0030-scan-findings-structured-store.md) — Scan findings store: structured `scan_*`/`dep_*` tables + FTS, not blobs or per-tool tables
- [04-adr-0031-go-version-floor.md](decisions/04-adr-0031-go-version-floor.md) — Go 1.22+ minimum to build (raised from a locked constraint to a decision)
- [04-adr-0032-opencode-kilo-native-plugins.md](decisions/04-adr-0032-opencode-kilo-native-plugins.md) — opencode + Kilo native TS plugin architecture: 5 plugins from `plugins/opencode/`, Kilo copy-at-install, shell hooks retired, facts HTTP routes
- [04-adr-0033-three-module-workspace-ctx.md](decisions/04-adr-0033-three-module-workspace-ctx.md) — Three-module Go workspace: `ctx/` is a library module importing `mnemonic`, no own DB handle
- [04-adr-0034-ctx-cli-mnemonic-subcommand.md](decisions/04-adr-0034-ctx-cli-mnemonic-subcommand.md) — `ctx` is a `mnemonic` subcommand group (`mnemonic ctx …`), not a standalone binary; `mnemonic stats` unchanged
- [04-adr-0035-local-ollama-judgment-model.md](decisions/04-adr-0035-local-ollama-judgment-model.md) — CTX judgment model is local Ollama (default `tev1:0.8b` via `/v1/systemone`), fail-open to pre-filter
- [04-adr-0036-ctx-config-configd.md](decisions/04-adr-0036-ctx-config-configd.md) — CTX config lives at `config.d/ctx.yaml` (machine-local default + repo-local override), reuses the `config.d` loader
- [04-adr-0037-embedded-ann-index.md](decisions/04-adr-0037-embedded-ann-index.md) — HNSW embedded ANN index: pure Go (no new dep), M=16, efCon=64, efSearch=96, persisted in SQLite (065), `source`-namespaced node ids (066, symbol/chunk), degrades to brute force
- [04-adr-0038-skill-pi-ticketing-backlog-adapter.md](decisions/04-adr-0038-skill-pi-ticketing-backlog-adapter.md) — skill-pi ticketing: `ticket` tool (10 actions) backed by the `backlog` MCP server over stdio, not vendored tk or direct file I/O
- [04-adr-0039-skill-pi-permission-extension.md](decisions/04-adr-0039-skill-pi-permission-extension.md) — skill-pi permission layer: 3-tier config, Tree-sitter bash matching, tool hiding, diff preview, hybrid approvals
- [04-adr-0042-skill-pi-dependencies.md](decisions/04-adr-0042-skill-pi-dependencies.md) — ADR-0042: skill-pi npm dependency surface (@earendil-works/pi-*)
- [04-adr-0043-skill-pi-mnemonic-extension.md](decisions/04-adr-0043-skill-pi-mnemonic-extension.md) — skill-pi Mnemonic memory extension: Pi-native folder `plugins/pi/mnemonic/` (index.ts entrypoint + mem_*/fact_*/code_* tool files over existing HTTP routes, prompt capture on before_agent_start, Memory Protocol injection, compaction context), `mnemonic setup pi` copies the folder into the agent-dir extensions
- [04-adr-0044-ctx-sandbox-http-and-pi-receipts.md](decisions/04-adr-0044-ctx-sandbox-http-and-pi-receipts.md) — context-harness sandbox over HTTP (POST /ctx/capture, GET /ctx/search, GET /ctx/get/{id}, GET /ctx/stats loopback-only, DELETE /ctx/sessions/{id}) + Pi tool_result overflow receipts ([skillgrid-ctx] marker, 4096-byte mirrored gate) + ctx_search/ctx_get/ctx_stats/ctx_purge Pi tools + promptSnippet on all 20 Pi tools
- [04-adr-0045-opencode-v2-plugin-port.md](decisions/04-adr-0045-opencode-v2-plugin-port.md) — OpenCode V2 plugin port: dual V1+V2 entrypoints for the 6 native plugins (V1 named export kept + V2 `Plugin.define` default), per-file local `wrap()` (JSON-Schema converter), `skillgrid-context` via `ctx.session.hook("context")`, no shim — every file in plugins/ must default-export {id, setup}
- [04-adr-0046-knowledge-class-dimension.md](decisions/04-adr-0046-knowledge-class-dimension.md) — Knowledge class dimension: nullable `knowledge_class` column (episodic\|convention\|doc\|unclassified) on observations, heuristic inference on save, re-runnable backfill, optional search filter (FTS-leg-only), migration 070 — additive on ADR-0012, no KB-per-class
- [04-adr-0047-fused-mem-search-contract.md](decisions/04-adr-0047-fused-mem-search-contract.md) — Fused mem_search contract: every mem_search also FTS-MATCHes web_cache_fts (project-scoped, expired excluded), RRF-fused (k=60) with the observations leg, web hits labeled corpus=web_cache + source/url/fetched_at/expires_at + web_cache_get retrieve pointer; vector leg stays observations-only — a BEHAVIOR SHIFT for every mem_search caller
- [04-adr-0048-hybrid-sot.md](decisions/04-adr-0048-hybrid-sot.md) — Hybrid source of truth: SQLite is the SoT for observations + code intelligence; the `.skillgrid/` Markdown bundle is the SoT for the knowledge base (R6 layout)
- [04-adr-0049-global-operational-memory.md](decisions/04-adr-0049-global-operational-memory.md) — Global operational memory: the `global` pseudo-project cross-project fact store at `global.sqlite` (amends ADR-0012)
- [04-adr-0050-harness2-host-integration.md](decisions/04-adr-0050-harness2-host-integration.md) — Harness² host integration: external subprocess, no Go dependency

## glossary/ — project vocabulary

- [01-business-terms.md](glossary/01-business-terms.md) — Glossary — Business
- [02-technical-terms.md](glossary/02-technical-terms.md) — Glossary — Technical

## invariants/ — locked constraints

- [05-locked-constraints.md](invariants/05-locked-constraints.md) — Locked Constraints

## reference/ — PRD, research, roadmaps, plans

- [00-prd.md](reference/00-prd.md) — Product Requirements: skillgrid (Hub Product)
- [06-research-findings.md](reference/06-research-findings.md) — Cross-Change Research Findings
- [06-spec-second-brain-learnings.md](reference/06-spec-second-brain-learnings.md) — Spec — Second-Brain Learnings (from claude-os)
- [07-mnemonic-tool-surface.md](reference/07-mnemonic-tool-surface.md) — 07 — Mnemonic: Planned Functions
- [08-second-brain-roadmap.md](reference/08-second-brain-roadmap.md) — 08 — Second Brain Roadmap (mnemonic)
- [09-claude-os-deep-dive.md](reference/09-claude-os-deep-dive.md) — 09 — claude-os Deep Dive (what we steal)
- [README.md](reference/README.md) — Artifacts — Topic Index
- [context-orchestrator-plan.md](reference/context-orchestrator-plan.md) — ================================================================================
- [skillgrid-workflow-update-plan.md](reference/skillgrid-workflow-update-plan.md) — Skillgrid Release/2 — Engineering Operating System Plan

## workflows/

- [sdd-pipeline.md](workflows/sdd-pipeline.md) — the SDD pipeline as a concept (phases, zone rule, terminal close)

## Shipped changes

Closed (immutable) change folders live in `xarchive/` (renamed from `archive/` on 2026-10-20; history intact via `git log --follow`).
