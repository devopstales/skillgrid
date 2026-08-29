## Why

GitNexus provides relationship intelligence (callers, blast radius, shortest
path, taint) that Mnemonic does not ship. Mnemonic provides persistent memory
(`mem_*`), BM25 code search (`code_*`), and a web-research cache (`web_*`)
that GitNexus does not ship. The two are orthogonal: GitNexus holds no history,
Mnemonic holds no graph.

This change has two parts:

1. **Skill-level bridge (done)** — copy the six `gentleman-ai` GitNexus skills
   into `.agents/skills/`, add a "Mnemonic Integration" section to each that
   maps one GitNexus workflow step to one Mnemonic call (`mem_save`,
   `mem_search`, `mem_context`, `web_cache_*`, `code_status`). The bridge is
   intentionally 1:1 — no tool overlap, just a disciplined join point.
2. **Mnemonic tool additions (new)** — three net-new MCP tools that close the
   gaps the skills exposed:
   - `change_summary` — git-diff digest for pre-commit + session-close glue.
   - `graph_stale` signal on `code_status` — one staleness read instead of two.
   - `session_diff_summary` — pair `detect_changes` with `mem_session_summary`.
   `graph_search` and `graph_cypher` ride on the in-flight `mnemonic-graph`
   and `mnemonic-cypher` changes; they are not repeated here.

## What Changes

- Six GitNexus skills now live in `.agents/skills/gitnexus-*` and reference
  Mnemonic as the memory layer (already merged).
- `.agents/AGENTS.md` GitNexus table points at `.agents/skills/` and lists
  `mnemonic-memory` / `mnemonic-memory-protocol` (already merged).
- Three new Mnemonic MCP tools + one extended tool response:
  - `change_summary` — one call returns the git-diff-affected files, symbols,
    and a one-line natural-language summary.
  - `code_status` response gains `graph_stale: bool` — a single staleness
    signal the agent can pair with GitNexus `status`.
  - `session_diff_summary` — at session close, pull the working-tree diff and
    write it into the session summary body.
- No changes to `mem_save`, `mem_search`, `code_search`, `web_cache_*` —
  those already serve the bridge at the skill level.

## Capabilities

### New Capabilities

- `mnemonic-bridge-skills`: Six GitNexus skills with Mnemonic integration sections (done).
- `change-summary`: New `change_summary` MCP tool returning a git-diff digest.
- `graph-stale-signal`: `code_status` response includes `graph_stale: bool`.
- `session-diff-summary`: New `session_diff_summary` MCP tool pairing diff with session summary.

### Modified Capabilities

- `code-status`: Response schema gains `graph_stale` field.

## Impact

- **Skills**: `.agents/skills/gitnexus-{cli,debugging,exploring,guide,impact-analysis,refactoring}/SKILL.md` (new + Mnemonic Integration sections).
- **MCP tools**: `change_summary`, `session_diff_summary` added; `code_status` response extended.
- **HTTP API**: `skillgrid serve` gains `GET /changes/summary` and extends `GET /code/status`.
- **Code**: `skillgrid-cli/internal/mnemonic/{mcp,http,service}` — new handlers.
- **Docs**: `docs/14-gitnexus-mnemonic-bridge.md` becomes a pointer to this change.
- **No breaking changes**: existing `mem_*`, `code_*`, `web_*` tools and routes unchanged.
