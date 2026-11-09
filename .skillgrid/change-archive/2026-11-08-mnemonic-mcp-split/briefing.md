# Mnemonic — Family-First CLI + Split MCP Servers (requirements & intent)

> **STATUS:** `draft` (2026-11-08)

> **Role:** the *what* — requirements, intent, success criteria. The companion
> `blueprint.md` is the *how*. Planning session of 2026-11-08 (this file
> consolidates it); baseline MCP probes are referenced under *Baseline artifacts*.

## Problem / Intent

One `mnemonic mcp` stdio server currently serves three families — `mem_*` (28),
`code_*` (26), `projectwiki_*` (12) — to every client, with one combined
instructions block (the ProjectWiki section alone is ~4.3KB). Any agent mounting
it pays the full tool budget and all three instruction blocks, whether or not it
uses the family. Meanwhile the CLI is not family-consistent: `mem` and
`projectwiki` are family groups but the code family is named after its default
verb (`mnemonic index`), and the transport sits at top level
(`mnemonic mcp --profile …`) instead of under the family it serves. The module
directory `mnemonic-v2/` still carries a dev-stage suffix although the product
is version 0.2.0. Finally, mcp-go v1.5.0 serves tool calls on a worker pool of
5, so concurrent calls answer in completion order — the requirement is strictly
one response at a time per server.

This change splits the surface into three single-purpose MCP servers under a
family-first CLI, renames the module, makes responses strictly serial, and
raises the code-family tool responses to the quality bar of two external
reference servers (gitnexus, CodeGraph).

## Settled decisions (planning session, 2026-11-08)

1. **Rename** `mnemonic-v2/` → `mnemonic/` (module
   `github.com/devopstales/skillgrid/mnemonic`; server version `0.2.0`).
2. **Family-first CLI tree** — the family is the top-level axis:
   - `mem` group → `memory`; `index` group → `code` (`mnemonic index` →
     `mnemonic code index`; `mnemonic index search` → `mnemonic code search`;
     all other sub-verbs move unchanged); `projectwiki` unchanged.
   - `mcp` becomes a verb under each family: `mnemonic memory mcp`,
     `mnemonic code mcp`, `mnemonic projectwiki mcp`; `--profile agent|all`
     stays a flag on those verbs.
   - `serve` (deprecated no-op) removed; bare `mnemonic mcp` removed.
   - Clean cutover: **no aliases**; unknown-verb errors carry "did you mean"
     suggestions.
   - MCP tool prefixes unchanged: `mem_*`, `code_*`, `projectwiki_*` (the
     protocol surface is not renamed with the CLI).
3. **Three single-purpose MCP servers**, one mounted per client:
   - `mnemonic memory mcp` — baselined on `~/go/bin/engram mcp` (3.7.0)
   - `mnemonic projectwiki mcp` — baselined on `openwiki mcp` (0.9.0),
     **including the initialize response**
   - `mnemonic code mcp` — baselined on `gitnexus mcp` (1.6.12) **and**
     `CodeGraph serve --mcp` (1.6.0), tool responses included
   - Server identities: `mnemonic-memory` / `mnemonic-projectwiki` /
     `mnemonic-code`, version `0.2.0`.
   - Per-family instructions only (no combined block):
     - memory ← engram's block adapted (positioning + CORE TOOLS one-liners =
       agent-profile set + DEFERRED TOOLS = admin set + PROACTIVE SAVE RULE +
       DELIVERY GUARANTEE + CONFLICT SURFACING), tool lists profile-aware
     - projectwiki ← openwiki's text as the literal base (retrieval discipline,
       trust rules, claims/stale rules); lifecycle section rewritten for
       `projectwiki_generate`/`validate`/`lint`; workspace/link paragraphs
       dropped (single-project vault)
     - code ← **none** (gitnexus sends none; guidance lives in tool
       descriptions — open item O1, default: none)
4. **Serial stdio:** `server.WithWorkerPoolSize(1)` — one tool call in flight
   at a time, one complete response at a time. Protocol frames already
   mutex-serialized in mcp-go. Post-startup, nothing but protocol frames may
   reach stdout (the `outW()` seam flips to stderr after `openRuntime` in the
   mcp path).
5. **Code tool response contract (equivalent-or-better):** `mnemonic code`
   tool responses must be at least as good as CodeGraph's `codegraph_explore`
   response (query header, symbol/file counts, blast radius with callers +
   test files per symbol, provenance note, verbatim line-numbered source per
   file with edge summary) and gitnexus's `context`/`query` responses
   (structured: status / symbol / epistemic / incoming / outgoing):
   - `code_search` is enriched: every hit gains a blast-radius line
     (caller count + test files) + a snippet — self-contained per result.
   - `code_explore` is revived on the v2 engine as the code server's primary
     composite tool (v1 had it; v2 dropped it; CodeGraph's single-primary-tool
     model is the reference): query → ranked symbols + call paths + verbatim
     line-numbered source per file + blast radius, in one call. Response
     format = CodeGraph baseline plus the v2 extras that make it strictly
     better: per-edge EXTRACTED/INFERRED provenance tags, `degraded` flag,
     fusion provenance.
   - Floor: no duplicate entries in edge lists (gitnexus `context` returns
     `run` twice in `incoming.calls` — dedup is the minimum bar).
6. **Store unchanged; parallel profiles are safe** (verified, not reworked):
   `004_sync_lease.sql`'s 60s-TTL lease already elects one sync worker across N
   MCP processes; the DSN carries `busy_timeout(1000)` + `_txlock=immediate`
   for cross-process writers. Pre-existing, unchanged edge: the code-index
   409 active-run guard is per-process; the fingerprint gate + SQLite locks
   backstop it.

## Requirements

- **R1 (rename):** `git mv mnemonic-v2 mnemonic`; module path
  `github.com/devopstales/skillgrid/mnemonic`; all in-module imports,
  `go.work`, and build entries follow. Historical
  `.skillgrid/change-specs/2026-*-mnemonic-v2-*` folder names are kept (they
  are the record of that milestone). `skillgrid-cli` (v1) is untouched.
- **R2 (CLI tree):** `mnemonic memory <verb>` / `mnemonic code <verb>` /
  `mnemonic projectwiki <verb>` as the family axis; sub-verb sets unchanged
  per family; `mcp` as a per-family verb; `serve` removed; bare `mnemonic mcp`
  removed; no aliases; "did you mean" on unknown verbs.
- **R3 (three servers):** each `mnemonic <family> mcp` starts a stdio MCP
  server exposing **only** that family's tools (profile-filtered as today:
  agent ⊂ all), under the family's server identity + version 0.2.0.
- **R4 (baseline-modeled responses):** the `initialize` response and
  `tools/list` descriptions of each server follow its baseline (memory←engram,
  projectwiki←openwiki, code←gitnexus+CodeGraph) per the settled decisions'
  table — capabilities: memory/projectwiki `{"tools":{"listChanged":true}}`,
  code `{"tools":{},"resources":{},"prompts":{}}`.
- **R5 (serial stdio):** at most one tool call is processed at a time; stdout
  carries exactly one complete JSON-RPC frame per write; all diagnostics go to
  stderr after serve starts.
- **R6 (code response contract):** per settled decision 5 — `code_search`
  per-hit blast-radius line + snippet; `code_explore` composite primary tool;
  dedup floor on edge lists.
- **R7 (one server per client):** the client-config contract states a client
  session mounts at most one mnemonic server (the family it needs); the
  in-repo onboarding template + user-guide encode it.
- **R8 (docs/config):** user-guide 05 (MCP section, tool tables, env table),
  onboarding template `agents-mnemonic.md` (MCP command lines →
  `mnemonic <family> mcp --profile agent`), `mnemonic/SKILL.md` command block,
  and any v2 skill/CLI reference updated to the new command paths.
- **R9 (ADR, last):** an ADR superseding T16's combined-server decision,
  recording the rename, family-first tree, three baseline-modeled servers,
  serial stdio, one-server-per-client, and the code response contract. Written
  after the code is verified (user's execution order: code first, ADR after).

## Out of scope

- No new MCP families (skills/scans/deps/web-cache are not v2 MCP surface yet).
- No store, sync, or index-semantics changes (R6 is response-shape only, over
  the existing graph).
- No `skillgrid-cli` (v1) changes.
- No HTTP transport (v2's `serve` is a deprecated no-op; removed here).
- No MCP tool-name changes (`mem_*` / `code_*` / `projectwiki_*` prefixes stay).
- No alias/compat window for the old CLI paths (clean cutover; in-repo configs
  are templates under this repo's control).

## Success criteria

- `mnemonic memory mcp` initialize → `serverInfo.name = mnemonic-memory`,
  version `0.2.0`, capabilities `{"tools":{"listChanged":true}}`, instructions
  matching the engram-derived block (agent mode lists only its tools).
- `mnemonic projectwiki mcp` initialize mirrors openwiki's shape (serverInfo /
  capabilities / instructions) adapted per settled decision 3.
- `mnemonic code mcp` initialize mirrors gitnexus's shape; no instructions
  (unless O1 resolves to CodeGraph's block).
- Each server's `tools/list` returns only its family (agent profile:
  memory 22, code 16, projectwiki per-profile; all profile: 28 / 26 / 12).
- Two concurrent `tools/call`s to one server: the second is not processed
  until the first's response is written (held-out test).
- `code_search` hit for a symbol with callers returns the caller count + test
  files in the hit itself (held-out test against an indexed repo).
- `code_explore` returns, in one call: header, counts, blast-radius section,
  provenance note, and verbatim line-numbered source per file — and no
  duplicate edge entries.
- `mnemonic code index`, `mnemonic code search`, `mnemonic memory save`,
  `mnemonic projectwiki query` all work at their new paths; `mnemonic index`,
  `mnemonic mem save`, `mnemonic mcp`, `mnemonic serve` print the usage/"did
  you mean" error (or are gone).
- Two `mnemonic <family> mcp` processes on one store run in parallel without
  sync double-push (existing lease test still passes) and without
  `database is locked` under the standard test load.
- `go build ./...` + `go test ./...` green for both modules; user-guide 05,
  onboarding template, and SKILL.md command references show only new paths.

## Open items

- **O1:** code server instructions — none (gitnexus, default) vs CodeGraph's
  markdown block. Resolved in the blueprint gate.

## Baseline artifacts (probed 2026-11-08; /tmp is ephemeral — re-fetch per blueprint)

- `~/go/bin/engram mcp` — initialize + tools/list (27 `mem_*` tools);
  transcript `/tmp/mcp-probe-engram.out`; instructions
  `/tmp/base-engram-instructions.txt` (1959B); `mem_save` description
  `/tmp/base-desc-mem_save.txt` (1932B)
- `openwiki mcp` 0.9.0 — initialize (instructions 4278B) + tools/list (10
  tools); `/tmp/mcp-probe-openwiki.out`;
  `/tmp/base-openwiki-instructions.txt`; `openwiki_search` (772B) /
  `openwiki_begin` (621B) descriptions
- `gitnexus mcp` 1.6.12 — initialize + tools/list (17 tools);
  `/tmp/mcp-probe-gitnexus2.out`; `context` tool response (7033B, structured
  JSON) `/tmp/base-gitnexus-context-response.txt`; `query` (1462B) /
  `context` (5694B) / `impact` (15445B) / `detect_changes` (1706B) descriptions
- `CodeGraph serve --mcp` 1.6.0 — initialize (instructions block) + tools/list
  (1 tool: `codegraph_explore`); `/tmp/mcp-probe-codegraph*.out`;
  **live `codegraph_explore` response** against this repo's own index
  (23947B) `/tmp/base-codegraph-explore-response.txt`
