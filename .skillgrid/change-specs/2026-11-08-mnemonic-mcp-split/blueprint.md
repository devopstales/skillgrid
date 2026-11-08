# Mnemonic — Family-First CLI + Split MCP Servers — Implementation Blueprint (steps, files, verification)

**Status:** PROPOSED
**Goal:** ship the split surface: renamed module, family-first CLI, three single-purpose baseline-modeled MCP servers (one per client), strictly serial stdio, and a code-family response contract at least as good as the CodeGraph/gitnexus baselines — with no store changes.
**Architecture:** single Go binary, three family groups under `mnemonic` (`memory`/`code`/`projectwiki`), each with its verbs + an `mcp` verb that starts a stdio server exposing only that family (mcp-go v1.5.0 `MCPServer`, `WithWorkerPoolSize(1)`). All families share the one existing `handler.Handler` + store (SQLite, `MaxOpenConns=1` per process, `busy_timeout(1000)` + `_txlock=immediate`); the autosync lease (`004_sync_lease.sql`) already arbitrates N parallel MCP processes. The code family gains an enriched `code_search` (per-hit blast-radius + snippet) and a revived `code_explore` composite (query → ranked symbols + call paths + verbatim line-numbered source + blast radius, CodeGraph response format + v2 EXTRACTED/INFERRED provenance).
**Tech Stack:** Go (repo floor 1.22+, workspace go 1.26), `github.com/mark3labs/mcp-go v1.5.0`, existing SQLite store; no new dependencies.
**Spec:** [briefing.md](./briefing.md) (R1–R9, settled decisions 1–6).
**Findings:** baseline MCP probes of 2026-11-08 (engram 3.7.0 / openwiki 0.9.0 / gitnexus 1.6.12 / CodeGraph 1.6.0) — transcript + instructions + tool-response inventory in the briefing's *Baseline artifacts* (re-fetchable; /tmp is ephemeral).
**Tier:** T2
**Build shape:** tracer thread — T1–T3 make one family (`memory`) real end-to-end at its new path (rename → CLI → `mnemonic memory mcp` with baseline initialize + serial stdio); T4–T5 thicken the code family's response contract; T6–T7 close docs + ADR.
**Classification:** high-risk (public CLI + MCP contract cutover; no data migration) → L4 verification (full suite + build + coverage + mutation sampling + Trivy advisory).
**Critic:** pending (T2 + one-way doors → mandatory)
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
- **T-6** each server's `tools/list` returns only its family: agent profile → memory 22, code 16, projectwiki per existing profile table; all profile → 28 / 26 / 12. No cross-family leak in any mode.
- **T-7** two concurrent `tools/call`s to one server: the second is not processed until the first's response is written (one tool call in flight at a time). `backstop` (held-out timing test).
- **T-8** after serve starts, stdout carries only protocol frames (one complete JSON-RPC message per write); all diagnostics go to stderr. `backstop` (seam test: a post-startup diagnostic lands in the stderr buffer, stdout buffer gains exactly the protocol frames).
- **T-9** `code_search` hits for indexed symbols carry a blast-radius line (caller count + test files) + snippet; unresolved hits degrade gracefully (field omitted, no error); edge lists contain no duplicates.
- **T-10** `code_explore` (new composite tool, agent profile) returns in one call: `Exploration: <query>` header, symbol/file counts, blast-radius section (callers + test files per symbol), provenance note, verbatim line-numbered source per file with an edge-summary line, EXTRACTED/INFERRED per-edge tags, `degraded` flag when a leg is absent. `backstop` (quality vs. the captured CodeGraph response on this repo — comparison test, not byte-identity).
- **T-11** two `mnemonic <family> mcp` processes on one store run in parallel: no sync double-push (existing lease test passes unmodified) and no `database is locked` under the standard test load. `backstop` (existing two-process test, re-run at the new paths).
- **T-12** no live skill/doc/template references the old paths: grep over `.agents/skills` + `docs` + `.skillgrid/projectwiki` for `mnemonic mcp `, `mnemonic mem `, `mnemonic index `, `mnemonic serve`, `mnemonic-v2/` (excluding `change-archive/` + historical `change-specs/2026-*-mnemonic-v2-*` folders) returns zero hits.
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
| `mnemonic/go.mod`, `go.work` | module line + `use` entry |
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
| `docs/user-guide/05-memory-and-indexing.md` (v2 section) | three servers, initialize shapes, command lines, env table |
| `.agents/skills/onboarding/templates/agents-mnemonic.md` | MCP command lines → `mnemonic <family> mcp --profile agent` |
| `.agents/skills/_shared/rules/mnemonic-*.md`, `.agents/skills/mnemonic/SKILL.md` | v2 path references only (v1 `skillgrid-cli` references untouched) |
| `mnemonic/.../*_test.go` (server/cli/handler) | the RED tests per task below |
| `.skillgrid/projectwiki/decisions/04-adr-NNNN-mcp-family-split.md`, `projectwiki/ASSUMPTIONS.md`, `adr.md` (this change) | ADR + index + manifest (T7) |

Out of the edit scope: `mnemonic/internal/core/**` (store unchanged), `mnemonic/internal/memory/autosync.go` (lease unchanged), `skillgrid-cli/**`, `mnemonic/internal/codeindex/search/hybrid.go` (search package stays pure; enrichment lives in the handler).

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| Documentation-like paths | N/A: no file-classification logic touched by this change | — | — |
| Git repository selection | N/A: cwd/git-toplevel authority unchanged; the code family's existing root resolution is reused as-is | — | — |
| Commit state | N/A: no VCS automation added | — | — |
| Push state | N/A: no push logic | — | — |
| PR commands | N/A: no PR logic | — | — |
| **Mnemonic tool surface** (`mem_*`/`code_*`/`projectwiki_*`) | Applicable: new tool `code_explore`; new return shape for `code_search` hits (added blast-radius + snippet fields); server identities/capabilities change; per-server profile counts | Explicit contract delta: tool names unchanged; `code_search` gains fields (additive); `code_explore` is new; `serverInfo.name`/`capabilities`/`instructions` per family per R4 | One test per changed contract: `TestInitializeShapePerFamily` (3 servers), `TestToolIsolationPerFamily` (agent+all × 3), `TestSearchHitBlastRadiusShape`, `TestExploreResponseShape`, `TestSerialToolCalls` |
| **Shared-convention drift** | Applicable: `_shared/rules/mnemonic-*.md` + onboarding template + user-guide name the old command paths/env | One-line impact: these files are the source of truth for how agents wire the MCP + which CLI paths exist — any skill that still names `mnemonic mcp`/`mnemonic index`/`mnemonic mem` is bound to a dead command | `TestNoStalePathReferences` — grep over live skills/docs/vault (excl. archive + historical folders) for `mnemonic mcp `, `mnemonic mem `, `mnemonic index `, `mnemonic serve`, `mnemonic-v2/`; zero hits |

## Tasks

Build shape: tracer thread. D1–D4 are one-way doors (tagged on their task; stop for explicit approval before executing that task).

### T1 — Rename module `mnemonic-v2/` → `mnemonic/` (door check)

> ⚠ one-way: D1 (module path + directory rename)

- **Files:** `mnemonic/` (whole tree via `git mv`), `mnemonic/go.mod`, `go.work`
- **Steps:**
  1. `git mv mnemonic-v2 mnemonic`.
  2. `go.mod`: module line → `github.com/devopstales/skillgrid/mnemonic`; `go.work`: `use ./mnemonic`.
  3. Update the 115 in-module import paths (mechanical replace of the module path prefix).
  4. Verify: `go build ./...` + `go test ./...` in `mnemonic/` **and** the root workspace (both modules); `grep -r 'mnemonic-v2' go.work mnemonic/go.mod` → 0 hits.
- **Test:** build + full suite green at the new path = the acceptance (the door check for the whole blueprint).
- **SATISFIES:** T-1

### T2 — Family-first CLI tree

> ⚠ one-way: D2 (CLI cutover, no aliases) · D3 (top-level `mcp` + `serve` removed)

- **Files:** `mnemonic/cmd/mnemonic/{main,mem,index,projectwiki}.go`, `cli_test.go`
- **Steps:**
  1. RED: `cli_test.go` — `mnemonic memory save`, `mnemonic code index`, `mnemonic code search`, `mnemonic projectwiki query`, `mnemonic memory mcp --profile all` (flag parse only, no serve) dispatch to the right runners; `mnemonic index`, `mnemonic mem save`, `mnemonic mcp`, `mnemonic serve` exit non-zero, stderr contains "did you mean" + the new path.
  2. GREEN: `main.go` dispatch — `case "mem"` → `"memory"`, `case "index"` → `"code"`; remove `case "mcp"` / `case "serve"`; unknown-verb handler gains a suggestion list (the three family groups + their verbs) and prints `did you mean: <closest>`; `mem.go`/`index.go` group names + `case "mcp"` in each of the three family runners calling `runMCP(fl, rest, family)`.
  3. `mnemonic code` (bare) still runs the index (default-verb behavior preserved from `case "index", ""`).
- **SATISFIES:** T-2
- **Note:** `runMCP` here takes the family param but T3 reworks its body; T2 only needs the dispatch + flag parse to compile — T2's `mcp` cases call the new signature with the family constant, and `runMCP` keeps today's body (all three families registered) until T3 switches it. This keeps T2 independently green.

### T3 — Per-family MCP servers + serial stdio (the tracer reaches end-to-end)

> ⚠ one-way: D4 (server identity split)

- **Files:** `mnemonic/internal/transport/mcp/{server,tools_mem,tools_code,tools_projectwiki}.go`, `mnemonic/cmd/mnemonic/mcp.go`, `server_test.go`
- **Steps:**
  1. RED: `TestInitializeShapePerFamily` — stdio handshake (initialize → initialized) against each family server, assert `serverInfo`/`capabilities`/`instructions` per T-3/T-4/T-5 (memory: engram-derived block, agent mode lists only agent tools; projectwiki: openwiki-derived with lifecycle rewritten for generate/validate/lint; code: gitnexus shape, **no** instructions).
  2. RED: `TestToolIsolationPerFamily` — `tools/list` per family × {agent, all}: exact name sets (22/28, 16/26, N/12 per existing profile tables); zero cross-family names.
  3. RED: `TestSerialToolCalls` — handler with an artificial delay; fire two `tools/call`s concurrently; assert the second's handler start is after the first's response write (the held-out T-7 proof).
  4. RED: `TestStdoutProtocolOnly` — with the `outW()`/`stderr` test seams, a post-startup diagnostic lands in the stderr buffer; the stdout buffer contains exactly the protocol frames (T-8).
  5. GREEN: `server.go` — `type Family` (`memory`/`code`/`projectwiki`); `NewServer(h, profile, family)` selects name + instructions + which registrations run (existing `RegisterCodeTools`/`RegisterProjectWikiTools` become family-gated); split `codeInstructions` out of `combinedInstructions`; `Start(ctx)` → `server.ServeStdio(s.mcp, server.WithWorkerPoolSize(1))`. `mcp.go` — `runMCP(fl, args, family)` registers only the selected family's handler; flips the `outW()` seam to stderr after `openRuntime`; startup warnings (autosync) stay pre-serve on stderr.
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
  1. RED: `TestExploreResponseShape` — fixture graph: `Explore("autosync lease runCycle")` output contains, in order: `**Exploration: <query>**` header, `Found N symbols across M files.`, `**Blast radius**` section with per-symbol `file:line — N callers; tests: <files>` lines, the provenance note, one `**\`file\`**` source block per file with line-numbered verbatim source + the edge-summary line; EXTRACTED edges untagged, INFERRED edges tagged `(inferred)`; `degraded` line present iff a leg reported degradation; no duplicate edge lines.
  2. GREEN: `explore_format.go` — the renderer (pure function over a typed `ExploreResult`; the CodeGraph baseline layout from `/tmp/base-codegraph-explore-response.txt` is the reference, re-fetched at execution). `handler_code.go` — `Explore(ctx, query)`: rank via the existing hybrid search (T4 path), fetch source per file via the existing read path (gap-eliding, same as `code_read`), blast radius via the existing impact join, edge provenance from the graph's EXTRACTED/INFERRED labels. Register `code_explore` in `tools_code.go` (agent profile) with the CodeGraph-style description ("PRIMARY TOOL — call FIRST…", one-capped-call, treat source as already Read).
- **SATISFIES:** T-10

### T6 — Docs / templates / convention files

- **Files:** `docs/user-guide/05-memory-and-indexing.md` (v2 section), `.agents/skills/onboarding/templates/agents-mnemonic.md`, `.agents/skills/_shared/rules/mnemonic-{memory,code-indexing,artifacts}.md`, `.agents/skills/mnemonic/SKILL.md` (v2 path references only)
- **Steps:**
  1. RED: `TestNoStalePathReferences` (shared-convention drift boundary): grep over `.agents/skills` + `docs` + `.skillgrid/projectwiki` (excluding `change-archive/` and historical `change-specs/2026-*-mnemonic-v2-*` folders) for `mnemonic mcp `, `mnemonic mem `, `mnemonic index `, `mnemonic serve`, `mnemonic-v2/` — fails before, passes after.
  2. GREEN: rewrite the v2 MCP surface docs (three servers, initialize shapes, per-family command lines, env table), onboarding template MCP entries → `mnemonic <family> mcp --profile agent`, and every v2 path reference in the convention files. v1 (`skillgrid-cli`) references stay untouched.
- **SATISFIES:** T-12

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

1. **Spec coverage:** R1→T1 · R2→T2 · R3→T3 · R4→T3 · R5→T3 · R6→T4+T5 · R7→T6 (contract text) · R8→T6 · R9→T7. No gaps.
2. **Must-haves coverage:** every R maps to ≥1 truth (R3/R4 → T-3..T-6; R5 → T-7/T-8; R6 → T-9/T-10; R8 → T-12; R9 → T-13). T-11 covers the parallel-profiles claim.
3. **One-way doors:** D1–D4 listed + tagged on T1/T2/T3.
4. **Placeholder scan:** no TBD/"implement later"/"similar to Task N" — T7's ADR number is resolved by a named command at execution (not a placeholder); T2's runMCP body deferral is a named interim state, not a hole.
5. **Type consistency:** `runMCP(fl, args, family)` used in T2 (new signature, old body) and T3 (new body) — consistent; `mcp.NewServer(h, profile, family)` named once, used in T3; `Explore(ctx, query)` / `ExploreResult` / `explore_format.go` consistent between T5 steps.
