---
type: TechnicalDecision
status: in-force
supersedes: "none (supersedes the combined-server design decision established by the 2026-10-31-mnemonic-v2-code-index change — blueprint T16 + spec 008-transport §3, built on the memory change's D-PROFILE two-profile mechanism — which was a design decision, not a numbered ADR)"
date: 2026-11-08
---

# ADR-0061 — MCP family split: three single-purpose servers + family-first CLI

Change: 2026-11-08-mnemonic-mcp-split (briefing.md / blueprint.md; the `task → commit` chain is recoverable via the commit `[skillgrid-context]` blocks)

## Context

- The single `mnemonic mcp` stdio server (v2, from 2026-10-31-mnemonic-v2-memory) registers three tool families — `mem_*` (28), `code_*` (27), `projectwiki_*` (12) — onto one `Server` through explicit registration seams (`RegisterCodeTools`, `RegisterProjectWikiTools`), with the `agent`/`all` profiles (D-PROFILE) selecting only the admin-tool subset within the one combined surface. The server's `instructions` is a single combined block; the ProjectWiki section alone is ~4.3KB. Every client that mounts it pays the full tool budget and all three instruction blocks, whether or not it uses the family.
- The cross-family workflows (projectwiki pin-checks verifying live code, mem_search fusing web-cache findings) composed across one process — but they compose across separate stdio processes too, because every server opens the same SQLite store with per-process single-connection semantics (MaxOpenConns=1) and `004_sync_lease.sql`'s 60s-TTL lease + `busy_timeout` handle cross-process writers.
- The CLI was not family-consistent: `mem` and `projectwiki` are family groups, but the code family is named after its default verb (`mnemonic index`), and the transport sits at top level (`mnemonic mcp --profile …`) instead of under the family it serves.
- The module directory `mnemonic-v2/` still carried a dev-stage suffix although the product is version 0.2.0.
- mcp-go v1.5.0 serves tool calls on a worker pool of 5 — concurrent calls answer in completion order; the product requirement is strictly one response at a time per server.
- External baseline probes (2026-11-08; artifacts under `/tmp`, re-fetchable per the briefing's "Baseline artifacts" section): `engram mcp` 3.7.0 (memory — initialize + 27 `mem_*` tools, instructions 1959B), `openwiki mcp` 0.9.0 (projectwiki — initialize, instructions 4278B, 10 tools), `gitnexus mcp` 1.6.12 + `CodeGraph serve --mcp` 1.6.0 (code — initialize + tool descriptions + a live `codegraph_explore` response of 23947B against this repo's own index).

## Decision

1. **Rename** `mnemonic-v2/` → `mnemonic/` (module `github.com/devopstales/skillgrid/mnemonic`; server version `0.2.0`). `skillgrid-cli` (v1) is untouched; historical `.skillgrid/change-specs/2026-*-mnemonic-v2-*` folder names are kept as the record of that milestone.
2. **Family-first CLI tree** — the family is the top-level axis: `mem` group → `memory`; `index` group → `code` (`mnemonic index` → `mnemonic code index`, `mnemonic index search` → `mnemonic code search`, all other sub-verbs move unchanged); `projectwiki` unchanged. `mcp` becomes a verb under each family (`mnemonic memory mcp` / `mnemonic code mcp` / `mnemonic projectwiki mcp`); `--profile agent|all` stays a flag on those verbs. `serve` (deprecated no-op) removed; bare `mnemonic mcp` removed. Clean cutover: no aliases; unknown-verb errors carry "did you mean" suggestions. MCP tool prefixes unchanged (`mem_*`, `code_*`, `projectwiki_*` — the protocol surface is not renamed with the CLI).
3. **Three single-purpose MCP servers**, one mounted per client (the R7 contract):
   - `mnemonic memory mcp` ← baselined on engram's initialize: `serverInfo.name = mnemonic-memory`, version `0.2.0`, capabilities `{"tools":{"listChanged":true}}`, engram-derived instructions (positioning + CORE TOOLS one-liners = the agent-profile set + DEFERRED TOOLS = the admin set + PROACTIVE SAVE RULE + DELIVERY GUARANTEE + CONFLICT SURFACING), tool lists profile-aware.
   - `mnemonic projectwiki mcp` ← baselined on openwiki's initialize, including its instructions: `mnemonic-projectwiki`, capabilities `{"tools":{"listChanged":true}}`, openwiki's text as the literal base (retrieval discipline, trust rules, claims/stale rules); lifecycle section rewritten for `projectwiki_generate`/`validate`/`lint`; workspace/link paragraphs dropped (single-project vault).
   - `mnemonic code mcp` ← baselined on gitnexus's initialize: `mnemonic-code`, capabilities `{"tools":{},"resources":{},"prompts":{}}`, **no instructions** (gitnexus sends none — open item O1 resolved to none; guidance lives in the tool descriptions).
4. **Serial stdio:** `server.WithWorkerPoolSize(1)` — one tool call in flight at a time, one complete response at a time. Protocol frames are already mutex-serialized in mcp-go. Post-startup, nothing but protocol frames may reach stdout (the `outW()` seam flips to stderr after `openRuntime` in the mcp path).
5. **Code tool response contract (equivalent-or-better):** `code_search` hits each gain a blast-radius line (caller count + test files) + a snippet — self-contained per result. `code_explore` is revived on the v2 engine as the code server's primary composite tool (v1 had it; v2 dropped it; CodeGraph's single-primary-tool model is the reference): query → ranked symbols + call paths + verbatim line-numbered source per file + blast radius, in one call — CodeGraph's response format plus the v2 extras that make it strictly better (per-edge EXTRACTED/INFERRED provenance tags, `degraded` flag, fusion provenance). Floor: no duplicate entries in edge lists (gitnexus `context` returns `run` twice in `incoming.calls` — dedup is the minimum bar).
6. **Store unchanged; parallel profiles are safe** (verified, not reworked): `004_sync_lease.sql`'s 60s-TTL lease already elects one sync worker across N MCP processes; the DSN carries `busy_timeout(1000)` + `_txlock=immediate` for cross-process writers. Pre-existing, unchanged edge: the code-index 409 active-run guard is per-process; the fingerprint gate + SQLite locks backstop it.

## Alternatives

- **One server + a family flag** (`mnemonic mcp --families mem,code,wiki`): the same registration seams, zero new process cost, per-client family selection via flag. Rejected in the planning session: the user asked for clean separation — separate server identities, focused per-server instruction blocks, and a one-server-per-client mount discipline, with the store's lease + busy_timeout making parallel family processes safe.
- **Split the servers, keep the CLI**: rejected — the CLI's family inconsistency (`mnemonic index` as a family name, top-level `mcp`) is the same root cause; a split without the rename leaves two conventions.
- **Keep the combined instructions block**: rejected — it is the single largest per-mount context cost (the ProjectWiki section alone is ~4.3KB); per-family blocks are the point of the split.

## Consequences

- Client configs mount one mnemonic server per session (the in-repo onboarding template + user-guide 05 encode it). An agent that needs two families in one session mounts two servers — verified safe (lease + busy_timeout; the existing parallel-process lease test passes; no `database is locked` under standard test load).
- The three servers carry distinct `serverInfo.name`s (`mnemonic-memory` / `mnemonic-code` / `mnemonic-projectwiki`) — a client can distinguish them in its tool listings, and each mount carries only its own family's instructions.
- The K6 handler seam is preserved: the three servers are three thin adapters over the same shared handler + the same store (T3: `NewServer(h, profile, family)`). The split is at the transport-registration level, not the engine level — deleting any one server does not touch the handler.
- The module rename moves every import path, `go.work`, Taskfile build entries, and the installer's packaging (v1 is untouched).
- mcp-go worker pool: `WithWorkerPoolSize(1)` is per-server; with one server per client there is exactly one response at a time on any mounted stream.

## Verification (2026-11-08)

Held-out tests against the baselines: `TestInitializeShapePerFamily` (per-family initialize contract — serverInfo / capabilities / instruction shape, shape-pinned against the engram/openwiki/gitnexus baselines), `TestToolIsolationPerFamily` (per-profile family counts, exact tool-name sets), the serial gate (`TestSerialToolCalls` — concurrent `tools/call`s processed one at a time, incl. a 128-call deep phase), `TestExploreResponseShape` / `TestExploreTagsInferredEdges` / `TestExploreEmptyIndex` / `TestExploreDegradedLeg` (the code response contract), and the parallel-process lease test (`TestTwoServersOneStoreParallel` — two family servers on one store, no sync double-push). Full suite `-race` green for both modules; Trivy `fs . --scanners vuln` run and reported (advisory).
