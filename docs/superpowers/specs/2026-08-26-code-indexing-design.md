# Code indexing tool integration — design

> **STATUS: DRAFT (2026-08-26)**

**Plan:** *(not yet written — derive from decisions below)*

**Related:** [04-mcp-servers.md](../../04-mcp-servers.md), [2026-08-26-add-mcp-integration-design.md](2026-08-26-add-mcp-integration-design.md), [Engram](https://github.com/Gentleman-Programming/engram), [NOTE.md](../../NOTE.md)

## Summary

Integrate a **single primary code-indexing tool** into skillgrid so agents query structure, flows, and blast radius before editing — not just file contents. **`config.d/` declares the tool**; install wires MCP + CLI on PATH; **project indexing** runs per repo (`analyze` / `index`).

**Proposed primary tool:** [GitNexus](https://github.com/abhigyanpatwari/GitNexus) — demote overlapping entries (`codegraph`, optional `ccc`) from the default bundle.

## Problem

Agents without a code index:

- grep and read files linearly — miss cross-module callers and execution flows
- refactor in isolation — break dependents they never opened
- duplicate work Engram already solved for *memory*, but not for *structure*

skillgrid today ships **three** overlapping local indexers in `config.d/mcp.yaml`:

| MCP id | Command | Role |
|--------|---------|------|
| `codegraph` | `codegraph mcp` | Local knowledge graph |
| `gitnexus` | `gitnexus mcp` | Local knowledge graph |
| `ccc` | `ccc mcp` | AST semantic search (CocoIndex Code) |

Plus **remote** repo understanding (`deepwiki`, `context7`) — complementary, not substitutes for a local index.

Having two graph indexers active confuses agents (“which tool do I call?”) and doubles index storage/maintenance.

## Goal

After `skillgrid install` + project setup:

1. One **primary** local indexer is on PATH and registered in agent MCP configs.
2. Operators have a documented **index lifecycle**: init → analyze/index → refresh on change → query via MCP.
3. Indexing pairs cleanly with **Engram** (session memory) and docs tools (Context7, DeepWiki).
4. No runtime `npx` — binaries from `tools.yaml` + `npm --prefix ~/.skillgrid/npm` (same as add-mcp policy).

## Non-goals

- Running two graph indexers in the default bundle (GitNexus **or** CodeGraph, not both)
- Cloud code upload or SaaS indexing as default
- Replacing IDE-native indexing (Cursor, etc.) — skillgrid adds a portable layer
- `skillgrid init` v1 (document manual/`gitnexus analyze` first; CLI subcommand later)
- Indexing `docs/` superpowers artifacts (code zone only)

---

## Tool evaluation & recommendation

### What “code indexing” means here

| Capability | Graph index (GitNexus, CodeGraph) | Semantic search (CocoIndex Code) | Remote docs (DeepWiki) |
|------------|-----------------------------------|----------------------------------|------------------------|
| Call graph / callers / callees | ✓ | partial via search hits | ✗ |
| Blast radius / impact | ✓ | ✗ | ✗ |
| Execution flows / processes | ✓ (GitNexus) | ✗ | ✗ |
| Coordinated rename | ✓ (GitNexus) | ✗ | ✗ |
| Git-diff impact | ✓ (GitNexus) | ✗ | ✗ |
| Natural-language “find code about X” | via `query` | ✓ **`search`** | ✓ (repo docs) |
| 100% local / offline | ✓ | ✓ | ✗ |
| Multi-repo registry | ✓ (GitNexus) | per project | N/A |

Agents doing **IDD/BDD/TDD apply** need graph capabilities most (impact before refactor, trace failures). Semantic search is a nice add-on, not a replacement.

### Candidates compared

| Criterion | **GitNexus** | **CodeGraph** | **CocoIndex Code (`ccc`)** |
|-----------|--------------|---------------|----------------------------|
| **Model** | Knowledge graph (KuzuDB + Tree-sitter) | Knowledge graph (SQLite + Tree-sitter) | Vector index (AST chunks + embeddings) |
| **MCP tools** | `query`, `context`, `impact`, `detect_changes`, `rename`, `cypher`, `list_repos` | `codegraph_explore` (+ optional specialized tools) | `search` only |
| **Agent fit** | Kilo, OpenCode, Cursor, Codex, Claude Code ([setup](https://github.com/abhigyanpatwari/GitNexus)) | Same family via `codegraph install` | Claude, Codex, Cursor, OpenCode via `ccc mcp` |
| **Skills ecosystem** | Upstream + user already has `gitnexus-*` skills | Installer writes agent instructions | `ccc` skill via marketplace |
| **Index command** | `gitnexus analyze` | `codegraph init -i` + `codegraph index` | `ccc index` |
| **Install** | npm package `gitnexus` | npm `@colbymchenry/codegraph` (already in `tools.yaml`) | pip/npm `cocoindex-code` → `ccc` binary |
| **skillgrid today** | `mcp.yaml` entry | `tools.yaml` + `mcp.yaml` | `mcp.yaml` as `ccc` |
| **Overlap with Engram** | Low — structure vs memory | Low | Low |
| **Overlap with each other** | High vs CodeGraph | High vs GitNexus | Different layer (search) |
| **Maturity / scope** | Broad (graph + MCP + skills + hooks on Claude) | Focused MCP + explore | Search-first, lighter |
| **License** | ISC | MIT | Apache 2.0 |

**Remote alternatives (keep, not primary index):**

| Tool | Use |
|------|-----|
| [DeepWiki MCP](https://mcp.deepwiki.com/mcp) | Q&A over public GitHub repos — no local index |
| Context7 | Library docs, not your repo structure |

### Recommendation

| Role | Tool | Rationale |
|------|------|-----------|
| **Primary (v1)** | **GitNexus** | Richest MCP surface for apply workflows (impact, `detect_changes`, rename); matches existing `gitnexus-*` skills; [local-first graph](https://github.com/abhigyanpatwari/GitNexus); OpenCode/Kilo documented |
| **Remove from default bundle** | **CodeGraph** | Redundant graph with GitNexus; two graphs → agent confusion and double index maintenance |
| **Defer (optional v2)** | **CocoIndex Code (`ccc`)** | Add only if users need semantic `search` *in addition to* graph — not as primary |
| **Keep as remote** | DeepWiki, Context7 | Docs and upstream libraries, not repo structure |

**Decision (proposed, lock after spike):**

```
Primary indexer:  gitnexus
Default mcp.yaml: gitnexus yes | codegraph no | ccc no
Engram pairing:   Engram = memory, GitNexus = code map
```

**Why not CodeGraph despite already being in `tools.yaml`?**

CodeGraph is strong and local-first ([docs](https://colbymchenry.github.io/codegraph/)), but skillgrid’s direction is **one graph, one MCP namespace**. GitNexus wins on impact/rename/diff tooling and existing skill investment. CodeGraph remains a valid fork choice — document swap procedure in plan, don’t ship both.

**Why not CocoIndex as primary?**

[CocoIndex Code](https://github.com/cocoindex-io/cocoindex-code) excels at semantic retrieval (`search` with AST chunks, incremental index) but does not expose blast-radius or multi-file rename. Better as a **second tool** for “find by meaning” than as the sole indexer for strict TDD/refactor workflows ([intent-driven.dev TDD post](https://intent-driven.dev/blog/2026/08/23/tdd-bdd-spec-driven-development/) assumes impact-aware edits).

---

## Architecture (GitNexus primary)

```
config.d/
├── tools.yaml          gitnexus (pinned npm)
├── mcp.yaml            gitnexus MCP entry only (graph slot)
└── indexing.yaml       NEW — index lifecycle defaults (optional v1)

skillgrid install
├── npm install gitnexus --prefix ~/.skillgrid/npm
├── MCP via add-mcp or sync script → gitnexus mcp
└── (optional) gitnexus setup -c opencode,kilo-code  — once per machine

Per project (operator or future skillgrid init)
└── gitnexus analyze    → ~/.gitnexus/ + project graph
    └── agents call MCP: query | context | impact | detect_changes
```

### Pairing with other skillgrid tools

| Tool | Layer | When agent uses it |
|------|-------|-------------------|
| **GitNexus** | Code structure | Before refactor, debug, impact analysis |
| **Engram** | Cross-session memory | Decisions, conventions, past session facts |
| **Context7** | Library API docs | External dependency usage |
| **DeepWiki** | Public repo docs | Upstream source exploration |
| **Gryph** (planned) | Audit trail | After tool calls — not indexing |

Rule for agents: **GitNexus for “how is this code wired?”; Engram for “what did we decide last time?”**

### MCP registration

Align with [add-mcp integration](2026-08-26-add-mcp-integration-design.md):

```yaml
# config.d/mcp.yaml (graph slot — single entry)
gitnexus:
  type: local
  command:
    - gitnexus
    - mcp
```

Install `gitnexus` via `tools.yaml`; binary on `~/.skillgrid/npm/node_modules/.bin/`. **No `npx -y gitnexus@…`.**

### Project index lifecycle

| Phase | Command | When |
|-------|---------|------|
| First open | `gitnexus analyze` | After clone or when graph missing/stale |
| Pre-refactor | MCP `impact` or `context` | During IDD apply / strict TDD |
| Pre-commit | MCP `detect_changes` | Optional; aligns with zone-guard commits |
| Stale index | `gitnexus analyze` | When MCP context reports staleness |

Index storage: GitNexus global registry under `~/.gitnexus/` (tool-managed). Not committed to git.

### Agent workflow hook (documentation)

Add to [02-usage.md](../../02-usage.md) / project `AGENTS.md` template:

```
Before structural edits:
1. Read gitnexus://repo/{name}/context — check freshness
2. Use impact/context for symbol under change
3. After significant commits, re-run analyze if detect_changes warns
```

Maps to existing `gitnexus-guide` skill workflow.

---

## config.d deliverables (proposed)

| File | Change |
|------|--------|
| `tools.yaml` | Add/pin `gitnexus`; **remove** `@colbymchenry/codegraph` from default |
| `mcp.yaml` | Keep `gitnexus`; **remove** `codegraph` and `ccc` from default |
| `indexing.yaml` | Optional: `primary: gitnexus`, `analyze_on_init: false`, `stale_policy: warn` |
| `AGENTS.md` marker | Short block: primary indexer + analyze command |

---

## Install step (proposed)

New gated step after MCP sync (or combined with MCP):

1. `hasTool(tools, "gitnexus")` gate
2. Verify `gitnexus` on PATH (`~/.skillgrid/npm/bin` first)
3. Dry-run: log `gitnexus setup -c opencode,kilo-code` (optional, idempotent)
4. Warn if `gitnexus` missing — continue install

**Do not** auto-run `gitnexus analyze` on every install (project-specific, slow).

---

## Migration from current bundle

| Current | Action |
|---------|--------|
| `codegraph` in tools.yaml + mcp.yaml | Remove from defaults; document opt-in for forks |
| `ccc` in mcp.yaml | Remove from defaults; v2 optional semantic search |
| `gitnexus` in mcp.yaml | Keep; ensure binary via tools.yaml not bare PATH hope |

Existing users: re-run install; manually remove duplicate MCP entries from agent configs or run `add-mcp` sync when available.

---

## Success criteria

1. Exactly **one** graph indexer in default `mcp.yaml`.
2. `gitnexus` installed via managed npm prefix — no runtime `npx`.
3. Docs describe analyze → query lifecycle and Engram pairing.
4. Spike confirms Kilo + OpenCode load `gitnexus mcp` after install.
5. Agent skills (`gitnexus-*`) and MCP tools align on same repo index.

---

## Open questions

1. **Spike:** `gitnexus setup` vs add-mcp-only for Kilo/OpenCode — which is idempotent enough for install step?
2. **CodeGraph opt-in:** separate `config.d/indexing.codegraph.yaml` overlay for teams that prefer it?
3. **CocoIndex v2:** add `ccc` back as `indexing.semantic_search: true` flag?
4. **`skillgrid init`:** run `gitnexus analyze` automatically when subcommand lands?

---

## References

- [GitNexus](https://github.com/abhigyanpatwari/GitNexus) — recommended primary
- [CodeGraph](https://colbymchenry.github.io/codegraph/) — alternative graph (not default)
- [CocoIndex Code](https://github.com/cocoindex-io/cocoindex-code) — optional semantic search
- [add-mcp integration design](2026-08-26-add-mcp-integration-design.md)
- [DeepWiki MCP](https://mcp.deepwiki.com/mcp) — remote, complementary
