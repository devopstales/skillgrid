# Mnemonic — Family-First CLI + Split MCP Servers — Implementation Blueprint (steps, files, verification)

**Status:** PROPOSED
**Goal:** ship the split surface: renamed module, family-first CLI, three single-purpose baseline-modeled MCP servers (one per client), strictly serial stdio, and a code-family response contract at least as good as the CodeGraph/gitnexus baselines — with no store changes.
**Architecture:** single Go binary, three family groups under `mnemonic` (`memory`/`code`/`projectwiki`), each with its verbs + an `mcp` verb that starts a stdio server exposing only that family (mcp-go v1.5.0 `MCPServer`, `WithWorkerPoolSize(1)`). All families share the one existing `handler.Handler` + store (SQLite, `MaxOpenConns=1` per process, `busy_timeout(5000)` + `_txlock=immediate` + WAL on the write DSN); the autosync lease (`004_sync_lease.sql`) already arbitrates N parallel MCP processes. The code family gains an enriched `code_search` (per-hit blast-radius + snippet) and a revived `code_explore` composite (query → ranked symbols + call paths + verbatim line-numbered source + blast radius, CodeGraph response format + v2 EXTRACTED/INFERRED provenance).
**Tech Stack:** Go (repo floor 1.22+, workspace go 1.26), `github.com/mark3labs/mcp-go v1.5.0`, existing SQLite store; no new dependencies.
**Spec:** [briefing.md](./briefing.md) (R1–R9, settled decisions 1–6).
**Findings:** baseline MCP probes of 2026-11-08 (engram 3.7.0 / openwiki 0.9.0 / gitnexus 1.6.12 / CodeGraph 1.6.0) — transcript + instructions + tool-response inventory in the briefing's *Baseline artifacts* (re-fetchable; /tmp is ephemeral).
**Tier:** T2
**Build shape:** tracer thread — T1–T3 make one family (`memory`) real end-to-end at its new path (rename → CLI → `mnemonic memory mcp` with baseline initialize + serial stdio); T4–T5 thicken the code family's response contract; T6–T7 close docs + ADR.
**Classification:** high-risk (public CLI + MCP contract cutover; no data migration) → L4 verification (full suite + build + coverage + mutation sampling + Trivy advisory).
**Critic:** complete (2026-11-08 scout pass, 32m — verdict NEEDS-FIXES: 4 MAJOR + 7 MINOR + 5 NIT, all resolved in this revision; user decisions: F-3 installer patch lands in T6 (declared scope expansion over `skillgrid-cli`), F-4 user-guide 05 gets an in-place v2 rewrite + legacy marking)
**Decision note (Owed-Decision Gate):** O1 resolved by default — the code server sends **no** `instructions` (gitnexus baseline; guidance in tool descriptions). Presented to the user as open item O1 in the briefing; no objection before this blueprint. Every other build value has a named source (briefing settled decisions / R1–R9).

## Hypothesis

```
**Claim:** splitting the surface into three single-purpose servers under a
family-first CLI, with serial stdio and a CodeGraph/gitnexus-grade code
response contract, gives each client a strictly smaller tool budget, strictly
serial complete responses, and code search/explore results at least as good as
the baselines — without any store change.
**Right condition:** probing `mnemonic memory mcp` returns only mem_* (22 in
agent mode) with the engram-shaped initialize; `mnemonic code mcp` answers two
concurrent tools/call strictly one at a time; `code_explore` for
"autosync lease acquisition runCycle" on this repo returns every CodeGraph
baseline element (header, counts, blast radius with callers + test files,
provenance note, line-numbered source per file) plus EXTRACTED/INFERRED tags;
`go build ./...` + `go test ./...` green for both workspace modules.
**Wrong condition:** any family server lists another family's tool; a stdout
frame is partial or interleaved; `code_explore` output is missing a baseline
element or contains duplicate edge entries; build or suite red after any task.
**Thinnest MVP:** T1–T3 — the rename + `mnemonic memory mcp` fully working at
its new path (baseline initialize, family-isolated tools, serial stdio).
**Door check:** T1 — build + full suite green at the renamed module path. If
the rename is not mechanical, stop; the rest of the blueprint is void.
```

## Must-Haves

### Truths

- **T-1** `go build ./...` + `go test ./...` green at the `mnemonic/` path; `go.mod` module line = `github.com/devopstales/skillgrid/mnemonic`; `go.work` uses `./mnemonic`; no `mnemonic-v2` references in go.mod/go.work.
- **T-2** `mnemonic memory save|search`, `mnemonic code index|search|status`, `mnemonic projectwiki query` dispatch at the new paths with unchanged behavior; `mnemonic index`, `mnemonic mem save`, bare `mnemonic mcp`, `mnemonic serve` exit non-zero with a "did you mean" suggestion naming the new path. `backstop` (runtime probe).
- **T-3** `mnemonic memory mcp` initialize → `serverInfo {mnemonic-memory, 0.2.0}`, capabilities `{"tools":{"listChanged":true}}`, instructions = engram-derived block with the agent-profile tool list in agent mode. `backstop` (stdio handshake probe).
- **T-4** `mnemonic projectwiki mcp` initialize mirrors openwiki's shape (serverInfo `mnemonic-projectwiki` 0.2.0, capabilities `{"tools":{"listChanged":true}}`, instructions = openwiki-derived block with the lifecycle section rewritten for `projectwiki_generate`/`validate`/`lint`). `backstop`.
- **T-5** `mnemonic code mcp` initialize mirrors gitnexus's shape (serverInfo `mnemonic-code` 0.2.0, capabilities `{"tools":{},"resources":{},"prompts":{}}`, **no** instructions). `backstop`.
- **T-6** each server's `tools/list` returns only its family, exact name sets pinned from `codeToolSet()`/`codeAdminOnlyTools`/the mem+wiki profile tables: agent profile → memory 22, code 19, projectwiki 8; all profile → memory 28, code 38, projectwiki 12; after T5 adds `code_explore` (agent profile) → code 20 / 39. No cross-family leak in any mode.
- **T-7** two concurrent `tools/call`s to one server: the second is not processed until the first's response is written (one tool call in flight at a time). `backstop` (held-out timing test).
- **T-8** after serve starts, stdout carries only protocol frames (one complete JSON-RPC message per write); all diagnostics go to stderr. `backstop` (seam test: a post-startup diagnostic lands in the stderr buffer, stdout buffer gains exactly the protocol frames).
- **T-9** `code_search` hits for indexed symbols carry a blast-radius line (caller count + test files) + snippet; unresolved hits degrade gracefully (field omitted, no error); edge lists contain no duplicates.
- **T-10** `code_explore` (new composite tool, agent profile) returns in one call: `Exploration: <query>` header, symbol/file counts, blast-radius section (callers + test files per symbol), provenance note, verbatim line-numbered source per file with an edge-summary line, EXTRACTED/INFERRED per-edge tags, per-hit fusion-leg provenance (from `Hit.Provenance`), `degraded` flag when a leg is absent. `backstop` (quality vs. the captured CodeGraph response on this repo — comparison test, not byte-identity).
- **T-11** two `mnemonic <family> mcp` processes on one store (process pair tested: `mnemonic memory mcp` + `mnemonic code mcp`) run in parallel: no sync double-push (existing lease test passes unmodified) and no `database is locked` under the standard test load. `backstop` (existing two-process test, re-run at the new paths).
- **T-12** no live skill/doc/template references the old paths: grep over `.agents/skills` + `docs` + `.skillgrid/projectwiki` with boundary-tolerant patterns (`mnemonic (mcp|mem |index |serve)([ \t]|$|\`)`, `mnemonic-v2/`) returns zero hits **outside the exclusion set**: `change-archive/`, historical `change-specs/2026-*-mnemonic-v2-*` folders, and `projectwiki/decisions/04-adr-*.md` (historical decision records — immutable once shipped; the vault's live pages, notably `ARCHITECTURE.md`, are NOT excluded and must be rewritten by T6).
- **T-13** ADR exists at `projectwiki/decisions/04-adr-NNNN-mcp-family-split.md` (NNNN = next free number), indexed in `ASSUMPTIONS.md` § LOCKED, listed in this change's `adr.md` manifest; it supersedes the combined-server decision (T16 of `2026-10-31-mnemonic-v2-code-index`) and records decisions 1–6 of the briefing.

### One-way doors

- **D1 (T1):** module path + directory rename — external importers of `github.com/devopstales/skillgrid/mnemonic-v2` break (in-repo only today; `skillgrid-cli` does not import it).
- **D2 (T2):** CLI cutover without aliases — every old command path dies in one commit.
- **D3 (T2):** top-level `mnemonic mcp` and `mnemonic serve` removed.
- **D4 (T3):** server identity split — clients keyed on `serverInfo.name = mnemonic-v2` break (in-repo onboarding template + the user's agent configs, both under this repo's control).

### Artifacts

- The renamed `mnemonic/` module building + testing green (T1).
- Three `mcp` verbs (per family) producing the three baseline-shaped servers (T3).
- Enriched `code_search` response + `code_explore` composite tool + format renderer (T4, T5).
- Updated docs/config: the user-guide page(s) describing the v2 MCP surface, `onboarding/templates/agents-mnemonic.md` (MCP command lines), `_shared/rules/mnemonic-*.md` path references, `mnemonic/SKILL.md` where it names v2 paths (T6).
- ADR + index row + manifest entry (T7).

### Key links

- family verb → `runMCP(fl, args, family)` → `mcp.NewServer(h, profile, family)` → only that family's registrations + instructions (T2/T3).
- `CodeHandler.Search` → graph in-edge join → enriched hit (callers + test files + snippet) (T4).
- `code_explore` tool → `CodeHandler.Explore` → hybrid ranking + `code_read` source + blast radius → CodeGraph-format renderer with provenance tags (T5).

## File Structure

| File | Responsibility (delta) |
|---|---|
| `mnemonic/` (whole tree, from `mnemonic-v2/`) | renamed module; import paths updated |
| `mnemonic/go.mod`, `go.work`, `Taskfile.yml` (`MEM2_DIR: mnemonic-v2` → `mnemonic`) | module line + `use` entry + build/install entry (F-5: `task build/install/cross-build` break after the rename otherwise) |
| `mnemonic/cmd/mnemonic/main.go` | dispatch: family groups `memory`/`code`/`projectwiki`; top-level `mcp`/`serve` cases removed; "did you mean" suggestion helper |
| `mnemonic/cmd/mnemonic/mem.go` | group name `memory`; `case "mcp"` → `runMCP(fl, rest, FamilyMemory)` |
| `mnemonic/cmd/mnemonic/index.go` | group name `code` (default verb = index run, unchanged); `case "mcp"` → `runMCP(fl, rest, FamilyCode)` |
| `mnemonic/cmd/mnemonic/projectwiki.go` | `case "mcp"` → `runMCP(fl, rest, FamilyProjectWiki)` |
| `mnemonic/cmd/mnemonic/mcp.go` | `runMCP(fl, args, family)`: family-selected registration, `outW()` seam → stderr after `openRuntime`, `Start(ctx, family)` |
| `mnemonic/internal/transport/mcp/server.go` | `Family` type + `NewServer(h, profile, family)`; per-family name/instructions; `codeInstructions` split out of `combinedInstructions`; `Start` passes `server.WithWorkerPoolSize(1)` |
| `mnemonic/internal/transport/mcp/tools_mem.go` | description text synced to the engram baseline |
| `mnemonic/internal/transport/mcp/tools_code.go` | description text in gitnexus `WHEN TO USE:` style; new `code_explore` tool entry (agent profile) |
| `mnemonic/internal/transport/mcp/tools_projectwiki.go` | description text synced to the openwiki baseline |
| `mnemonic/internal/transport/handler/handler_code.go` | `Search` enrichment (blast-radius join + snippet, dedup); `Explore(ctx, query)` composite |
| `mnemonic/internal/transport/handler/explore_format.go` (new) | CodeGraph-baseline response renderer (header/counts/blast-radius/provenance/source blocks, EXTRACTED/INFERRED tags) |
| `docs/user-guide/05-memory-and-indexing.md` | in-place v2 rewrite of the MCP/CLI/env sections (three servers, initialize shapes, family command lines); v1-only content (dashboard, web cache, fact memory, L0–L3) marked legacy with a pointer (user decision F-4) |
| `skillgrid-cli/internal/setup/setup.go` (+ `hermes_test.go`, `protocol_test.go`, `cursor_test.go`) | **declared scope expansion** (user decision F-3): installer MCP entries write `["mnemonic","<family>","mcp","--profile","agent"]` (memory family = default) instead of the dead `["mnemonic","mcp"]` |
| `.skillgrid/projectwiki/ARCHITECTURE.md` | live vault page: module paths, CLI dispatch table, `runMCP` signature anchor rewritten for the new tree; `projectwiki index` same-edit (F-2) |
| `.agents/skills/onboarding/templates/agents-mnemonic.md` | v1-menu ladder (SKILLGRID_MCP_CODE_TOOLS, code_map/…) → the v2 family surface; MCP command lines per family |
| `.agents/skills/_shared/rules/mnemonic-*.md`, `.agents/skills/mnemonic/SKILL.md` (v2 refs), `.agents/skills/projectwiki/SKILL.md` (bare `mnemonic-v2` ref), `.agents/skills/qa/scripts/qa-gate.mjs` (`--anticipated "mnemonic-v2/"`) | path references only (v1 `skillgrid-cli` command references untouched) |
| `mnemonic/.../*_test.go` (server/cli/handler), `mnemonic/cmd/mnemonic/{parity_test.go,index_test.go}` | the RED tests per task below (parity/index tests call old CLI paths) |
| `.skillgrid/projectwiki/decisions/04-adr-NNNN-mcp-family-split.md`, `projectwiki/ASSUMPTIONS.md`, `adr.md` (this change) | ADR + index + manifest (T7) |

Out of the edit scope: `mnemonic/internal/core/**` (store unchanged), `mnemonic/internal/memory/autosync.go` (lease unchanged), `skillgrid-cli/**` **except** the installer MCP entries in `internal/setup/` (T6, user-declared expansion), `mnemonic/internal/codeindex/search/hybrid.go` (search package stays pure; enrichment lives in the handler).

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| Documentation-like paths | N/A: no file-classification logic touched by this change | — | — |
| Git repository selection | N/A: cwd/git-toplevel authority unchanged; the code family's existing root resolution is reused as-is | — | — |
| Commit state | N/A: no VCS automation added | — | — |
| Push state | N/A: no push logic | — | — |
| PR commands | N/A: no PR logic | — | — |
| **Mnemonic tool surface** (`mem_*`/`code_*`/`projectwiki_*`) | Applicable: new tool `code_explore`; new return shape for `code_search` hits (added blast-radius + snippet fields); server identities/capabilities change; per-server profile counts | Explicit contract delta: tool names unchanged; `code_search` gains fields (additive); `code_explore` is new; `serverInfo.name`/`capabilities`/`instructions` per family per R4 | One test per changed contract: `TestInitializeShapePerFamily` (3 servers), `TestToolIsolationPerFamily` (agent+all × 3), `TestSearchHitBlastRadiusShape`, `TestExploreResponseShape`, `TestSerialToolCalls` |
| **Shared-convention drift** | Applicable: `_shared/rules/mnemonic-*.md` + onboarding template + user-guide 05 + vault `ARCHITECTURE.md` + the `skillgrid-cli` installer (which writes the client MCP command lines) name the old command paths/env | One-line impact: these files are the source of truth for how agents wire the MCP + which CLI paths exist — any skill that still names `mnemonic mcp`/`mnemonic index`/`mnemonic mem` is bound to a dead command, and any fresh install from the unpatched installer writes a dead command | `TestNoStalePathReferences` — grep over live skills/docs/vault (excl. T-12's exclusion set) with boundary-tolerant patterns; zero hits. Installer side: the three setup tests assert the new `["mnemonic","<family>","mcp","--profile","agent"]` args |

## Tasks

Build shape: tracer thread. D1–D4 are one-way doors (tagged on their task; stop for explicit approval before executing that task).

### T1 — Rename module `mnemonic-v2/` → `mnemonic/` (door check)

> ⚠ one-way: D1 (module path + directory rename)

- **Files:** `mnemonic/` (whole tree via `git mv`), `mnemonic/go.mod`, `go.work`, `Taskfile.yml`
- **Steps:**
  1. `git mv mnemonic-v2 mnemonic`.
  2. `go.mod`: module line → `github.com/devopstales/skillgrid/mnemonic`; `go.work`: `use ./mnemonic`; `Taskfile.yml`: `MEM2_DIR: mnemonic-v2` → `mnemonic` (build/install/cross-build entry).
  3. Update the 115 in-module import paths (mechanical replace of the module path prefix).
  4. Verify: `go build ./...` + `go test ./...` in `mnemonic/` **and** the root workspace (both modules); `task build` (or the `go build` it wraps) succeeds at the new path; `grep -r 'mnemonic-v2' go.work mnemonic/go.mod Taskfile.yml` → 0 hits.
- **Test:** build + full suite green at the new path = the acceptance (the door check for the whole blueprint).
- **SATISFIES:** T-1

### T2 — Family-first CLI tree

> ⚠ one-way: D2 (CLI cutover, no aliases) · D3 (top-level `mcp` + `serve` removed)

- **Files:** `mnemonic/cmd/mnemonic/{main,mem,index,projectwiki}.go`, `cli_test.go`, `parity_test.go`, `index_test.go`; `mnemonic/internal/transport/mcp/server.go` gains the `Family` type + `FamilyMemory`/`FamilyCode`/`FamilyProjectWiki` constants (T2 needs them before T3; T3 consumes them)
- **Steps:**
  1. RED: `cli_test.go` — `mnemonic memory save`, `mnemonic code index`, `mnemonic code search`, `mnemonic projectwiki query`, `mnemonic memory mcp --profile all` (flag parse only, no serve) dispatch to the right runners; `mnemonic index`, `mnemonic mem save`, `mnemonic mcp`, `mnemonic serve` exit non-zero, stderr contains "did you mean" + the new path.
  2. GREEN: `main.go` dispatch — `case "mem"` → `"memory"`, `case "index"` → `"code"`; remove `case "mcp"` / `case "serve"`; unknown-verb handler gains a suggestion list (the three family groups + their verbs) and prints `did you mean: <closest>`; `mem.go`/`index.go` group names + `case "mcp"` in each of the three family runners calling `runMCP(fl, rest, family)`. Update the old-path invocations in `parity_test.go` / `index_test.go` to the new paths (they are T2's RED test's neighbors, not T3's).
  3. `mnemonic code` (bare) still runs the index (default-verb behavior preserved from `case "index", ""`).
- **SATISFIES:** T-2
- **Note:** `runMCP` here takes the family param but T3 reworks its body; T2 only needs the dispatch + flag parse to compile — T2's `mcp` cases call the new signature with the family constant, and `runMCP` keeps today's body (all three families registered) until T3 switches it. This keeps T2 independently green.

### T3 — Per-family MCP servers + serial stdio (the tracer reaches end-to-end)

> ⚠ one-way: D4 (server identity split)

- **Files:** `mnemonic/internal/transport/mcp/{server,tools_mem,tools_code,tools_projectwiki}.go`, `mnemonic/cmd/mnemonic/mcp.go`, `server_test.go`
- **Steps:**
  1. RED: `TestInitializeShapePerFamily` — stdio handshake (initialize → initialized) against each family server, assert `serverInfo`/`capabilities`/`instructions` per T-3/T-4/T-5 (memory: engram-derived block, agent mode lists only agent tools; projectwiki: openwiki-derived with lifecycle rewritten for generate/validate/lint; code: gitnexus shape, **no** instructions).
  2. RED: `TestToolIsolationPerFamily` — `tools/list` per family × {agent, all}: exact name sets pinned from the registration literals — memory 22/28, code 19/38 (the 38-entry `codeToolSet()` minus the 19 `codeAdminOnlyTools`), projectwiki 8/12; zero cross-family names.
  3. RED: `TestSerialToolCalls` — handler with an artificial delay; fire two `tools/call`s concurrently; assert the second's handler start is after the first's response write (the held-out T-7 proof).
  4. RED: `TestStdoutProtocolOnly` — with the `outW()`/`stderr` test seams, a post-startup diagnostic lands in the stderr buffer; the stdout buffer contains exactly the protocol frames (T-8).
  5. GREEN: `server.go` — `NewServer(h, profile, family)` selects **name + version per family (const `serverVersion` → `0.2.0`)** + instructions + which registrations run (existing `RegisterCodeTools`/`RegisterProjectWikiTools` become family-gated); the memory family's instructions render the CORE/DEFERRED tool lists **per profile** (agent mode: agent tools only; all mode: + admin list); split `codeInstructions` out of `combinedInstructions`; capabilities per family: memory/projectwiki keep the default `{"tools":{"listChanged":true}}`, the code family passes `server.WithToolCapabilities(false)` + `server.WithResourceCapabilities(false,false)` + `server.WithPromptCapabilities(false)` to yield the pinned `{"tools":{},"resources":{},"prompts":{}}` (explicit false wins over implicit AddTool registration; omitempty then renders the empty objects); `Start(ctx)` → `server.ServeStdio(s.mcp, server.WithWorkerPoolSize(1), server.WithQueueSize(1))` (queue size 1 closes mcp-go's >100-in-flight synchronous-fallback exception to "one at a time"). `mcp.go` — `runMCP(fl, args, family)` registers only the selected family's handler; flips the `outW()` seam to stderr after `openRuntime`; startup warnings (autosync) stay pre-serve on stderr. **Autosync decision:** all three family servers start the autosync manager when `sync.auto=true` (unchanged from today's body); the 60s lease elects one sync worker per data dir across the parallel processes — no per-family gating.
  6. Sync the three families' tool descriptions to their baselines (engram `mem_*` text; gitnexus `WHEN TO USE:` style for `code_*`; openwiki compact text for `projectwiki_*`).
- **SATISFIES:** T-3, T-4, T-5, T-6, T-7, T-8

### T4 — `code_search` enrichment (per-hit blast-radius + snippet)

- **Files:** `mnemonic/internal/transport/handler/{handler_code.go,*_test.go}`
- **Steps:**
  1. RED: `TestSearchHitBlastRadiusShape` — indexed fixture with a symbol that has callers + a `_test.go` referent: `Search` returns the hit with caller count + test file paths + snippet (first line of the symbol span); a hit whose symbol did not resolve: blast-radius fields omitted, no error; a symbol with two distinct caller edges listed twice in the fixture: result contains each edge exactly once (dedup floor).
  2. GREEN: in `CodeHandler.Search`, after the hybrid result, join each resolved hit's symbol to the graph's in-edges: caller count = distinct caller symbols; test files = distinct `*_test.go` files among callers (existing test-file detection, same rule `code_affected` uses); snippet via the existing `code_read` line fetch (one line, the span start); dedup edges by (caller uid, target uid).
- **SATISFIES:** T-9

### T5 — `code_explore` composite (primary tool, CodeGraph-format response)

- **Files:** `mnemonic/internal/transport/handler/{handler_code.go,explore_format.go,*_test.go}`, `mnemonic/internal/transport/mcp/tools_code.go`
- **Steps:**
  1. RED: `TestExploreResponseShape` — fixture graph: `Explore("autosync lease runCycle")` output contains, in order: `**Exploration: <query>**` header, `Found N symbols across M files.`, `**Blast radius**` section with per-symbol `file:line — N callers; tests: <files>` lines, the provenance note, one `**\`file\`**` source block per file with line-numbered verbatim source + the edge-summary line; EXTRACTED edges untagged, INFERRED edges tagged `(inferred)`; per-hit fusion-leg provenance rendered from `Hit.Provenance`; `degraded` line present iff a leg reported degradation; no duplicate edge lines. Second case (empty index / non-git dir, root falls back to cwd): header + `Found 0 symbols across 0 files.` + no source blocks + no `degraded` line + no error.
  2. GREEN: `explore_format.go` — the renderer (pure function over a typed `ExploreResult`; the CodeGraph baseline layout from `/tmp/base-codegraph-explore-response.txt` is the reference, re-fetched at execution). `handler_code.go` — `Explore(ctx, query)`: rank via the existing hybrid search (T4 path), fetch source per file via the existing read path (gap-eliding, same as `code_read`), blast radius via the existing impact join, edge provenance from the graph's EXTRACTED/INFERRED labels. Register `code_explore` in `tools_code.go` (agent profile) with the CodeGraph-style description ("PRIMARY TOOL — call FIRST…", one-capped-call, treat source as already Read).
- **SATISFIES:** T-10

### T6 — Docs / templates / convention files

- **Files:** `docs/user-guide/05-memory-and-indexing.md`, `.skillgrid/projectwiki/ARCHITECTURE.md`, `skillgrid-cli/internal/setup/{setup.go,hermes_test.go,protocol_test.go,cursor_test.go}`, `.agents/skills/onboarding/templates/agents-mnemonic.md`, `.agents/skills/_shared/rules/mnemonic-{memory,code-indexing,artifacts}.md`, `.agents/skills/mnemonic/SKILL.md` (v2 refs), `.agents/skills/projectwiki/SKILL.md` (bare `mnemonic-v2`), `.agents/skills/qa/scripts/qa-gate.mjs` (`--anticipated "mnemonic-v2/"`)
- **Steps:**
  1. RED: `TestNoStalePathReferences` (shared-convention drift boundary): grep over `.agents/skills` + `docs` + `.skillgrid/projectwiki` with T-12's boundary-tolerant patterns and T-12's exclusion set (archive + historical change folders + `projectwiki/decisions/04-adr-*.md`) — fails before (ARCHITECTURE.md + the convention files), passes after.
  2. GREEN — **user-guide 05 in-place v2 rewrite (user decision F-4):** rewrite 05's MCP/CLI/env sections for the v2 surface (three servers, per-family initialize shapes, `mnemonic <family> <verb>` command tables, env table); mark the v1-only content (HTTP dashboard :7438, web cache, fact memory, L0–L3 layered memory) as legacy/v1 with a pointer to `change-archive/` records. Adjacent `09-serve-dashboard.md` stays (v1 dashboard, out of scope — named here so nobody re-scopes it).
  3. GREEN — **`ARCHITECTURE.md` (live vault page, F-2):** module paths, the CLI dispatch table, and the `runMCP` signature anchor rewritten for the new tree; run `mnemonic projectwiki index` in the same edit (same-edit rule).
  4. GREEN — **installer (declared scope expansion, user decision F-3):** `skillgrid-cli/internal/setup/setup.go` MCP entries write `["mnemonic","<family>","mcp","--profile","agent"]` with the memory family as the default (the family the behavior blocks target); update the three setup tests (`hermes_test.go`, `protocol_test.go`, `cursor_test.go`).
  5. GREEN — convention files: onboarding template ladder (v1 menu: `SKILLGRID_MCP_CODE_TOOLS`, `code_map`/`code_search_symbols`/…) → the v2 family surface + per-family MCP command lines; `_shared/rules/mnemonic-*.md` path references; `mnemonic/SKILL.md` + `projectwiki/SKILL.md` v2 references; `qa-gate.mjs` `--anticipated "mnemonic/"`. v1 `skillgrid` command references stay untouched.
- **SATISFIES:** T-12 (and R7's client-config contract: fresh installs get live commands)

### T7 — ADR (last, per user execution order: code first, ADR after)

- **Files:** `.skillgrid/projectwiki/decisions/04-adr-NNNN-mcp-family-split.md` (NNNN = next free number, resolved at execution: `ls .skillgrid/projectwiki/decisions/ | sort | tail -1`), `.skillgrid/projectwiki/ASSUMPTIONS.md` (§ LOCKED row), `adr.md` (this change's manifest)
- **Steps:**
  1. Author the ADR: supersedes the combined-server decision (T16 of `2026-10-31-mnemonic-v2-code-index`, located via `ASSUMPTIONS.md` § LOCKED at execution); records briefing decisions 1–6 (rename, family-first tree, three baseline-modeled servers + one-per-client, serial stdio, code response contract, store unchanged) with the baseline evidence (probe inventory from the briefing).
  2. Add the `ASSUMPTIONS.md` LOCKED pointer row; add the manifest entry to this change's `adr.md`; run `mnemonic projectwiki index` (same-edit rule).
- **SATISFIES:** T-13

## Verification (per task + final)

- Per task: its RED test runs red→green; `go build ./...` + `go test ./...` (both modules) green before the next task.
- Final (L4, per classification): full suite `-race`; mutation sampling on `explore_format.go` + the serial-queue option (kill `WithWorkerPoolSize(1)` → `TestSerialToolCalls` must catch it); Trivy `fs . --scanners vuln` (advisory-only per AGENTS.md); the three stdio handshake probes re-run manually against the built binary and diffed by eye against the briefing's baseline inventory; two-server parallel run on one store (lease + no-lock check, T-11).
- Zone order: spec zone (this file) committed before any code-zone commit; T7's ADR is spec-zone and commits after the code it records (the user's stated order — code first, ADR last — overrides the default spec-before-code for this single artifact).

## Self-Review

1. **Spec coverage:** R1→T1 · R2→T2 · R3→T3 · R4→T3 · R5→T3 · R6→T4+T5 · R7→T6 (installer + docs) · R8→T6 · R9→T7. No gaps. (Critic F-3: R7's carrier was the out-of-scope installer — resolved by the user-declared T6 expansion.)
2. **Must-haves coverage:** every R maps to ≥1 truth (R3/R4 → T-3..T-6; R5 → T-7/T-8; R6 → T-9/T-10; R8 → T-12; R9 → T-13). T-11 covers the parallel-profiles claim (process pair named).
3. **One-way doors:** D1–D4 listed + tagged on T1/T2/T3.
4. **Placeholder scan:** no TBD/"implement later"/"similar to Task N" — T7's ADR number is resolved by a named command at execution (not a placeholder; the vault's "Highest sequence in use" line in `ASSUMPTIONS.md` is the reliable source, `ls|sort|tail -1` cross-check); T2's runMCP body deferral is a named interim state, not a hole.
5. **Type consistency:** `runMCP(fl, args, family)` used in T2 (new signature, old body) and T3 (new body) — consistent; `Family`/`FamilyMemory`/`FamilyCode`/`FamilyProjectWiki` defined in T2 (server.go), consumed by T2's runners and T3; `mcp.NewServer(h, profile, family)` named once, used in T3; `Explore(ctx, query)` / `ExploreResult` / `explore_format.go` consistent between T5 steps.
6. **Critic-round fixes applied:** F-1 counts (22/28, 19/38 → 20/39, 8/12) · F-2 T-12 exclusions + ARCHITECTURE.md in T6 · F-3 installer in T6 (user) · F-4 05 in-place rewrite (user) · F-5 Taskfile + qa-gate.mjs + projectwiki/SKILL.md · F-6 busy_timeout 5000 · F-7 WithQueueSize(1) · F-8 With*Capabilities(false) named · F-9 fusion provenance in T-10 + T5 · F-10 empty-index RED case · F-11 autosync decision + T-11 pair · F-12 parity/index tests · F-13 Family home in T2 · F-14 version const · F-15 boundary-tolerant patterns.
