# Tasks — mnemonic MCP family split (rename + family-first CLI + split servers)

> **STATUS:** `sliced` (2026-11-08)

> Sliced from `.skillgrid/change-specs/2026-11-08-mnemonic-mcp-split/blueprint.md` (approved 2026-11-08, critic round resolved).
> Vertical tickets, dependency-ordered, sized for one fresh agent context window.
> **Serial development** (AGENTS.md § Rules) — single wave, strict TICKET-01 → 07.

## Epic Summary

Ship the split surface: rename `mnemonic-v2/` → `mnemonic/` (module `github.com/devopstales/skillgrid/mnemonic`, v0.2.0), family-first CLI (`memory`/`code`/`projectwiki` + per-family `mcp` verb, `serve` out, no aliases), three single-purpose baseline-modeled MCP servers (memory←engram, projectwiki←openwiki, code←gitnexus+CodeGraph; one mounted per client), strictly serial stdio (`WithWorkerPoolSize(1)` + `WithQueueSize(1)`, stdout protocol-only), and the code response contract (per-hit blast radius on `code_search`; `code_explore` composite in the CodeGraph response format + EXTRACTED/INFERRED + fusion provenance). No store changes. Docs/convention files/installer follow; ADR last (user execution order: code first, ADR after).

Tracer thread: TICKET-01 is the door check (the rename must be purely mechanical — build + full suite green at the new path or the blueprint is void). TICKET-02–03 make one family (`memory`) real end-to-end at its new path; TICKET-04–05 thicken the code family's response contract; TICKET-06–07 close docs/installer + ADR.

Fast-Track: **standard** — high-risk classification (public CLI + MCP contract cutover, 4 one-way doors D1–D4, no data migration), 7 tickets, ~2,600–3,200 changed lines. No fast-track waiver; full tasks.md.

## Delivery Strategy

| Field | Value |
|-------|-------|
| Estimated changed lines | ~2,600–3,200 (7 tickets, ~30–40% test code, ~8 new/changed files in the mcp+handler packages) |
| 400-line budget risk | Medium (TICKET-03 ~700 and TICKET-05 ~700 held whole: the server split and the explore composite are the load-bearing slices; the rename ticket is large in touched lines but mechanical) |
| Chained PRs recommended | No — single branch (release/2), one commit per ticket, spec-zone commits before code-zone |
| Suggested split | 7 work units = the 7 tickets, strict sequence (serial development) |
| Delivery strategy | ask-on-risk |
| Chain strategy | single-branch serial (one change at a time) |

Decision needed before apply: No (blueprint approved; both scope decisions — installer patch, 05 rewrite — already user-resolved at the critic gate)
Chained PRs recommended: No
Chain strategy: single-branch serial
400-line budget risk: Medium

## Tickets

> **Precondition (all tickets):** blueprint approved + committed; `acceptance.feature` committed before any code-zone commit (pre-commit zone guard); `state.yaml` repointed to this change (done 2026-11-08); baseline probe artifacts re-fetchable per the briefing inventory (the /tmp copies are ephemeral).
> **One-way doors:** D1 (TICKET-01) · D2 + D3 (TICKET-02) · D4 (TICKET-03) — each ticket stops for explicit user approval before execution, per the blueprint.

### TICKET-01 — Rename module `mnemonic-v2/` → `mnemonic/` (door check) ✅ done 2026-11-08 (`08f8984c`)

> ⚠ one-way: D1 (module path + directory rename)

- **Tracker ID:** (assigned at ticketing) (milestone 2026-11-08-mnemonic-mcp-split)
- **Scope:** `git mv mnemonic-v2 mnemonic`; `go.mod` module line → `github.com/devopstales/skillgrid/mnemonic`; `go.work` `use ./mnemonic`; `Taskfile.yml` `MEM2_DIR` → `mnemonic`; the 115 in-module import paths (mechanical replace of the module path prefix).
- **Acceptance:** `go build ./...` + `go test ./...` green in `mnemonic/` AND at the root workspace (both modules); `task build` (or its wrapped `go build`) succeeds at the new path; `grep -r 'mnemonic-v2' go.work mnemonic/go.mod Taskfile.yml` → 0 hits; `skillgrid-cli` builds without importing the renamed path.
- **SATISFIES:** rename-builds-and-tests-green-at-new-path, rename-leaves-no-module-references, rename-keeps-v1-untouched
- **Files:** `mnemonic/` (whole tree via `git mv`), `mnemonic/go.mod`, `go.work`, `Taskfile.yml`
- **Size:** ~230 (M) — mechanical; the touched-line count is the module import set
- **Blocks:** TICKET-02
- **Blocked by:** none
- **Reversibility:** one-way (D1). In-repo only today; `skillgrid-cli` does not import the module.
- **Fails-when:** build or suite red at the new path (the door check — stop; the blueprint is void) OR any `mnemonic-v2` reference left in go.mod/go.work/Taskfile.

### TICKET-02 — Family-first CLI tree ✅ done 2026-11-08 (`5b047ddb`)

> ⚠ one-way: D2 (CLI cutover, no aliases) · D3 (top-level `mnemonic mcp` + `mnemonic serve` removed)

- **Tracker ID:** (assigned at ticketing) (milestone 2026-11-08-mnemonic-mcp-split)
- **Scope:** `main.go` dispatch — `case "mem"` → `"memory"`, `case "index"` → `"code"`; remove `case "mcp"` / `case "serve"`; unknown-verb "did you mean" suggestions (the three family groups + their verbs). `mem.go`/`index.go` group renames; `case "mcp"` in all three family runners calling `runMCP(fl, rest, family)` (TICKET-03 reworks the body — the interim body keeps today's behavior with the `family` param accepted-but-ignored). `server.go` gains `type Family` + `FamilyMemory`/`FamilyCode`/`FamilyProjectWiki` (TICKET-03 consumes them). `mnemonic code` (bare) still runs the index (default verb). Old-path invocations in `parity_test.go`/`index_test.go` updated to the new paths.
- **Acceptance:** `go test ./cmd/mnemonic/ -run 'TestFamilyDispatch|TestOldPathSuggestions|TestBareCodeDefaultsToIndex|TestServeGone|TestUnknownProfileRefuses'` passes — new paths dispatch with unchanged behavior; `mnemonic index` / `mnemonic mem save` / `mnemonic mcp` / `mnemonic serve` exit non-zero with "did you mean" + the new path; `mnemonic memory mcp --profile all` flag-parses (no serve in the test).
- **SATISFIES:** new-family-paths-dispatch, bare-code-defaults-to-index, old-paths-suggest-their-replacement, serve-is-gone, unknown-profile-refuses
- **Files:** `mnemonic/cmd/mnemonic/{main,mem,index,projectwiki,mcp}.go`, `cli_test.go`, `parity_test.go`, `index_test.go`, `mnemonic/internal/transport/mcp/server.go` (Family type only)
- **Size:** ~300 (M)
- **Blocks:** TICKET-03
- **Blocked by:** TICKET-01
- **Reversibility:** one-way (D2, D3).
- **Fails-when:** an old path still succeeds, a new path errors, or the "did you mean" line is missing on a removed path.

### TICKET-03 — Per-family MCP servers + serial stdio (tracer end-to-end) ✅ done 2026-11-08 (`39b7e2ed`)

> ⚠ one-way: D4 (server identity split: `mnemonic-v2` → `mnemonic-{memory,code,projectwiki}`)

- **Tracker ID:** (assigned at ticketing) (milestone 2026-11-08-mnemonic-mcp-split)
- **Scope:** `server.go` — `NewServer(h, profile, family)`: per-family server name + version (`serverVersion` const → `0.2.0`) + instructions + family-gated registrations (existing `RegisterCodeTools`/`RegisterProjectWikiTools`); memory instructions render the CORE/DEFERRED tool lists **per profile** (engram baseline); projectwiki instructions = openwiki baseline with the lifecycle section rewritten for `projectwiki_generate`/`validate`/`lint`; `codeInstructions` split out of `combinedInstructions`; the code family passes `WithToolCapabilities(false)` + `WithResourceCapabilities(false,false)` + `WithPromptCapabilities(false)` → the pinned `{"tools":{},"resources":{},"prompts":{}}`. `Start(ctx)` → `server.ServeStdio(s.mcp, server.WithWorkerPoolSize(1), server.WithQueueSize(1))`. `mcp.go` — `runMCP(fl, args, family)` registers only the selected family; `outW()` seam → stderr after `openRuntime`; startup warnings pre-serve on stderr; **all three family servers start the autosync manager when `sync.auto=true` (the 60s lease arbitrates — no per-family gating)**. Tool descriptions synced to the baselines (engram `mem_*`; gitnexus `WHEN TO USE:` style `code_*`; openwiki compact `projectwiki_*`).
- **Acceptance:** `go test ./internal/transport/mcp/ -run 'TestInitializeShapePerFamily|TestToolIsolationPerFamily|TestSerialToolCalls|TestStdoutProtocolOnly|TestTwoServersOneStoreParallel'` passes — initialize shapes per family (memory: engram-shaped, agent-mode tool list; projectwiki: openwiki-shaped, no workspace/link language; code: gitnexus-shaped, no instructions field); tools/list exact sets (memory 22/28, code 19/38, projectwiki 8/12) with zero cross-family names; two concurrent calls answer one at a time; >100 in-flight stays serial; post-startup diagnostics land on stderr, stdout gains exactly the protocol frames; memory+code servers in parallel on one store: ≤1 sync-lease holder, no `database is locked`.
- **SATISFIES:** memory-server-serves-memory-only, all-profile-adds-admin-tools-only, code-and-wiki-servers-are-isolated, memory-initialize-follows-engram, projectwiki-initialize-follows-openwiki, code-initialize-follows-gitnexus, concurrent-calls-answer-one-at-a-time, deep-concurrency-stays-serial, stdout-carries-only-protocol-frames, two-servers-one-store-parallel, mutation-of-serial-option-is-caught
- **Files:** `mnemonic/internal/transport/mcp/{server,tools_mem,tools_code,tools_projectwiki,server_test}.go`, `mnemonic/cmd/mnemonic/mcp.go`
- **Size:** ~700 (L) — held whole (the three baseline instruction texts + the family gating + the stdio option are one coherent slice)
- **Blocks:** TICKET-04, TICKET-05
- **Blocked by:** TICKET-02
- **Reversibility:** one-way (D4).
- **Fails-when:** a family server lists another family's tool, the initialize JSON deviates from the pinned shape (any field), two calls interleave, or a non-protocol byte reaches stdout post-startup.

### TICKET-04 — `code_search` enrichment (per-hit blast radius + snippet) ✅ done 2026-11-08 (`59195870`)

- **Tracker ID:** (assigned at ticketing) (milestone 2026-11-08-mnemonic-mcp-split)
- **Scope:** `handler_code.go` `Search`: after the hybrid result, join each resolved hit's symbol to the graph's in-edges — caller count = distinct caller symbols; test files = distinct `*_test.go` files among callers (the same test-file rule `code_affected` uses); snippet = the span-start line via the existing read path; dedup edges by (caller uid, target uid). Unresolved hits: fields omitted, no error. The search package stays pure (no `hybrid.go` edits).
- **Acceptance:** `go test ./internal/transport/handler/ -run 'TestSearchHitBlastRadiusShape|TestSearchUnresolvedHitDegrades|TestSearchDuplicateEdgesOnce'` passes — hit reports 2 callers + the test file + the first source line; unresolved hit returns with fields omitted, no error; a twice-recorded caller edge counts once.
- **SATISFIES:** search-hit-carries-blast-radius-and-snippet, unresolved-hit-degrades-gracefully, duplicate-edges-are-reported-once
- **Files:** `mnemonic/internal/transport/handler/{handler_code,handler_code_test}.go`
- **Size:** ~300 (M)
- **Blocks:** TICKET-05
- **Blocked by:** TICKET-03
- **Reversibility:** reversible (additive response fields; the renderer is TICKET-05's).
- **Fails-when:** a hit misses its blast-radius line, an unresolved hit errors, or any edge appears twice.

### TICKET-05 — `code_explore` composite (primary tool, CodeGraph-format response) ✅ done 2026-11-08 (`541ab49a`)

- **Tracker ID:** (assigned at ticketing) (milestone 2026-11-08-mnemonic-mcp-split)
- **Scope:** `explore_format.go` (new) — pure renderer over a typed `ExploreResult`: `**Exploration: <query>**` header, `Found N symbols across M files.`, `**Blast radius**` section (per-symbol `file:line — N callers; tests: <files>`), the provenance note, per-file `**\`file\`**` source blocks (line-numbered verbatim, gap-elided, edge-summary line), EXTRACTED untagged / INFERRED `(inferred)`, per-hit fusion-leg provenance from `Hit.Provenance`, `degraded` line iff a leg degraded, no duplicate edges. The CodeGraph baseline layout (`/tmp/base-codegraph-explore-response.txt`, re-fetched) is the reference. `handler_code.go` — `Explore(ctx, query)`: rank via the hybrid path (TICKET-04), source per file via the existing read path, blast radius via the existing impact join, edge provenance from the graph labels. `tools_code.go` — register `code_explore` (agent profile) with the CodeGraph-style description ("PRIMARY TOOL — call FIRST…", one-capped-call, treat source as already Read).
- **Acceptance:** `go test ./internal/transport/handler/ -run 'TestExploreResponseShape|TestExploreTagsInferredEdges|TestExploreEmptyIndex|TestExploreDegradedLeg'` passes — populated fixture: full baseline shape in order, no duplicate edge lines, inferred tags, fusion provenance present; empty index / non-git dir: `Found 0 symbols across 0 files.` + no source blocks + no degraded flag + no error; one degraded leg: the flag names the leg. Mutation gate: dropping the blast-radius section from the renderer fails the shape test.
- **SATISFIES:** explore-returns-the-full-baseline-shape, explore-tags-inferred-edges, explore-on-empty-index-degrades-silently, explore-declares-a-degraded-leg, mutation-of-renderer-is-caught
- **Files:** `mnemonic/internal/transport/handler/{handler_code,explore_format,explore_format_test}.go`, `mnemonic/internal/transport/mcp/tools_code.go`
- **Size:** ~700 (L) — held whole (renderer + handler + registration are one composite)
- **Blocks:** TICKET-06
- **Blocked by:** TICKET-04
- **Reversibility:** reversible (new tool + new renderer; removal restores the pre-ticket surface).
- **Fails-when:** the response is missing a baseline element, lists a duplicate edge, tags an EXTRACTED edge, or errors on the empty index.

### TICKET-06 — Docs / convention files / installer [OK] done 2026-11-08

- **Tracker ID:** (assigned at ticketing) (milestone 2026-11-08-mnemonic-mcp-split)
- **Scope:** (1) RED `TestNoStalePathReferences`: boundary-tolerant grep over `.agents/skills` + `docs` + `.skillgrid/projectwiki` with T-12's exclusion set (archive + historical change folders + `projectwiki/decisions/04-adr-*.md`) — fails before, passes after. (2) **user-guide 05 in-place v2 rewrite** (user decision): MCP/CLI/env sections → the v2 surface (three servers, per-family initialize shapes, `mnemonic <family> <verb>` tables, env table); v1-only content (dashboard :7438, web cache, fact memory, L0–L3) marked legacy with a pointer; `09-serve-dashboard.md` stays (named out-of-scope). (3) **`ARCHITECTURE.md`**: module paths, CLI dispatch table, `runMCP` signature anchor → the new tree; `mnemonic projectwiki index` same-edit. (4) **installer** (declared scope expansion, user decision): `skillgrid-cli/internal/setup/setup.go` MCP entries → `["mnemonic","<family>","mcp","--profile","agent"]` (memory family = default) + the three setup tests. (5) Convention files: onboarding template ladder → v2 family surface + per-family command lines; `_shared/rules/mnemonic-*.md` paths; `mnemonic/SKILL.md` + `projectwiki/SKILL.md` v2 refs; `qa-gate.mjs` `--anticipated "mnemonic/"`. v1 `skillgrid` command references untouched.
- **Acceptance:** `TestNoStalePathReferences` passes (zero hits outside the exclusion set); `cd skillgrid-cli && go test ./internal/setup/` passes with the new arg assertions; `mnemonic projectwiki validate` (or the anchor check) is clean for `ARCHITECTURE.md` after the same-edit reindex.
- **SATISFIES:** no-stale-path-references-remain, fresh-install-writes-live-mcp-commands, architecture-page-matches-the-new-tree
- **Files:** `docs/user-guide/05-memory-and-indexing.md`, `.skillgrid/projectwiki/ARCHITECTURE.md`, `skillgrid-cli/internal/setup/{setup.go,hermes_test.go,protocol_test.go,cursor_test.go}`, `.agents/skills/onboarding/templates/agents-mnemonic.md`, `.agents/skills/_shared/rules/mnemonic-{memory,code-indexing,artifacts}.md`, `.agents/skills/mnemonic/SKILL.md`, `.agents/skills/projectwiki/SKILL.md`, `.agents/skills/qa/scripts/qa-gate.mjs`, the RED test file under `mnemonic/` (or a scripts/ test per repo convention for the grep check)
- **Size:** ~400 (M) — docs-heavy, no engine code
- **Blocks:** TICKET-07
- **Blocked by:** TICKET-05
- **Reversibility:** reversible (docs/config).
- **Fails-when:** any live reference to an old path survives the exclusion set, an installer entry writes the dead bare form, or the architecture page's anchors are stale.

### TICKET-07 — ADR (last, per user execution order: code first, ADR after)

- **Tracker ID:** (assigned at ticketing) (milestone 2026-11-08-mnemonic-mcp-split)
- **Scope:** Author `.skillgrid/projectwiki/decisions/04-adr-NNNN-mcp-family-split.md` (NNNN = next free — the vault's "Highest sequence in use" line in `ASSUMPTIONS.md` is the reliable source, `ls decisions/ | sort | tail -1` cross-check): supersedes the combined-server decision (T16 of `2026-10-31-mnemonic-v2-code-index`, located via § LOCKED); records briefing decisions 1–6 (rename, family-first tree, three baseline-modeled servers + one-per-client, serial stdio, code response contract, store unchanged) with the baseline probe evidence. Add the `ASSUMPTIONS.md` LOCKED pointer row + this change's `adr.md` manifest entry; `mnemonic projectwiki index` same-edit.
- **Acceptance:** the ADR file exists at the next free number with a supersession pointer; the LOCKED index names it; the manifest lists it; final gate: full suite `-race` green for both modules (the `full-suite-green-race` scenario) + Trivy `fs . --scanners vuln` (advisory) run and reported.
- **SATISFIES:** adr-lands-and-indexes, full-suite-green-race
- **Files:** `.skillgrid/projectwiki/decisions/04-adr-NNNN-mcp-family-split.md`, `.skillgrid/projectwiki/ASSUMPTIONS.md`, `.skillgrid/change-specs/2026-11-08-mnemonic-mcp-split/adr.md`
- **Size:** ~120 (S)
- **Blocks:** none (ship gate)
- **Blocked by:** TICKET-06
- **Reversibility:** additive (a superseding ADR; the superseded one is marked, not deleted).
- **Fails-when:** the ADR number collides, the supersession pointer is missing, or the final suite is red.

## Gate Summary

| Gate | Where proven |
|---|---|
| Door check (rename mechanical) | TICKET-01 acceptance = the whole suite at the new path |
| Serial stdio (mutation-sensitive) | `TestSerialToolCalls` + `TestDeepConcurrencyStaysSerial` + the named mutation gate |
| Explore renderer (mutation-sensitive) | `TestExploreResponseShape` + the named mutation gate |
| Parallel safety (no store changes) | `TestTwoServersOneStoreParallel` (lease + no-lock) |
| No stale references | `TestNoStalePathReferences` (boundary-tolerant, exclusion set) |
| L4 final | full suite `-race` both modules + mutation sampling + Trivy advisory (TICKET-07) |
