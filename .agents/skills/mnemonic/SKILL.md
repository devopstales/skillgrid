---
name: mnemonic
description: "Use when persisting, recalling, or structuring project memory, code orientation, or web research via the mnemonic MCP tools."
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  based_on: skillgrid-v2:mnemonic
---

# Mnemonic

Operate Mnemonic, skillgrid's local-first persistent memory (SQLite + FTS5, single `skillgrid` binary).
Three things it does that a bare agent cannot:
1. **Persistent memory** — decisions, bugfixes, and discoveries in `~/.skillgrid/mnemonic/<project>.sqlite` survive across sessions and compactions. Recall weeks later without re-reading everything.
2. **Code index** — per-project chunk/symbol/edge graph with freshness tracking. Orient without dumping whole trees.
3. **Web cache** — Context7/Exa/DeepWiki/WebFetch snapshots with TTLs, so online research is reused instead of re-fetched.

**Announce at start:** "I'm using the skillgrid:mnemonic skill for persistent memory, code orientation, and web research."

## Overview

Operates Mnemonic — skillgrid's local-first persistent memory (SQLite + FTS5, one `skillgrid` binary) — to persist, recall, and structure decisions, bugfixes, and discoveries that survive across sessions and compactions; to orient on code without dumping whole trees; and to cache web research so it is reused instead of re-fetched. The discipline is non-negotiable: **save when you learn, recall before you re-derive, close the session with a full summary** — skip any of the three and the next session starts blind.

## When to Use

- When you need to save or recall persistent memory/learnings (decisions, bugfixes, discoveries, conventions, preferences).
- When starting a session and you need to recall prior context before re-deriving it.
- When orienting on code or reusing prior web research via the cache.

**When NOT to use:** for transient within-session notes that won't survive a session — use the session state file, not `mem_save`.

## Config

Read `.skillgrid/config.yaml` before starting. This skill reads:

| Config key | Used for |
|------------|----------|
| `mnemonic.enabled` | If `false`, skip this skill entirely |
| `mnemonic.data_dir` | Override the default `~/.skillgrid/mnemonic` store location |

If `mnemonic.enabled: false`, stop — memory, code index, and web cache are all inactive.

## Tools

- Core (loaded automatically at session start by the hook or the `mnemonic` MCP — no manual search needed):
  `mem_save`, `mem_search`, `mem_context`, `mem_session_start`, `mem_session_end`, `mem_session_summary`,
  `mem_save_prompt`, `mem_get_observation`, `mem_suggest_topic_key`,
  `code_status`, `code_index`, `code_search`, `code_read`,
  `web_cache_lookup`, `web_cache_save`, `web_cache_search`, `web_cache_get`
- Deferred (fetch on demand): `mem_stats`, `mem_delete`, `mem_timeline`, `mem_pin`, `mem_review`, `mem_capture_passive`

**Fallback**: if tools are unexpectedly unavailable, run `skillgrid setup` again and restart the agent. Setup repairs the durable MCP config and permissions allowlist.

**Degradation contract** (per `../_shared/conventions/hybrid-degradation.md`): if Mnemonic is unavailable mid-session (MCP disconnected, binary missing), do NOT fail silently. Continue with filesystem-only persistence and emit a degradation line in your return envelope: `Degraded: mnemonic — {reason}`. Do not retry more than once per session. The filesystem copy in `.skillgrid/specs/` is the recovery source of truth.

## Usage

```
/mnemonic                                             # session status: mem_context + doctor + index status
/mnemonic <path>                                      # orient on a path: index status → map → memory search
/mnemonic save "<title>" --type decision --content "..."   # curated save (needs active session)
/mnemonic recall "<question>"                         # mem_context → mem_search → mem_get_observation
/mnemonic index [--dir .] [--watch]                   # incremental code index (opt-in live watcher)
/mnemonic search "<query>" [--corpus code|grep|mem|symbols|hybrid|semantic]
/mnemonic code <subcommand>                           # orient/graph: map, get-symbol, impact, callers/callees...
/mnemonic serve | mcp | setup | export | trail        # daemon, agent wiring, export, trails
```

## What You Must Do When Invoked

If no path was given, use `.` (current directory). Do not ask the user for a path.

Follow these steps in order. Do not skip steps.

### Step 1 — Ensure the binary and store exist

```bash
command -v skillgrid >/dev/null || go build -o /tmp/skillgrid-op ./skillgrid-cli 2>&1 | tail -3
skillgrid --help 2>&1 | head -20
ls "${SKILLGRID_MNEMONIC_DATA_DIR:-$HOME/.skillgrid/mnemonic}" 2>/dev/null || echo "no mnemonic store yet (created on first index/save)"
```

Facts (verified against `skillgrid-cli/`):
- Data dir: `~/.skillgrid/mnemonic`, override `SKILLGRID_MNEMONIC_DATA_DIR`. Per-project store `<project>.sqlite`.
- Daemons: `skillgrid serve` → HTTP API `http://127.0.0.1:7438` (viewer `/`, Swagger `/swagger-ui`); `skillgrid mcp` → stdio server `skillgrid-mnemonic` with `mem_*`, `code_*`, `web_*`, `team_*` tools.
- Config: `skillgrid-cli/config.d/indexing.yaml` (include go/ts/tsx/md; chunk 80 lines / overlap 10; web-cache cap 256 KB; TTLs context7 720h, exa 168h, deepwiki 336h, fetch 168h).
- If the binary is missing, say so and stop — do not invent memory contents.

### Step 2 — Open or recover the session

Session tools require a live session id (`sid`). `mem_save` without `session_id` fails.

```
mem_session_start(title: "skillgrid/<goal>")   # once per agent session, reuse sid for every mem_save
```

Recovery (never ask what you can search):
```
mem_context(limit: 5)                          # fast: recent session summaries
→ if empty: mem_search(query: "<phase keywords>") → mem_get_observation(id)  # previews are truncated; get_observation is the only full-content path
```

Skillgrid naming (when working a change `{YYYY-MM-DD-<topic>}`): `title` == `topic_key` == `skillgrid/{YYYY-MM-DD-<topic>}/{artifact}` (`briefing|blueprint|findings|tasks|execution-progress|qa-report|ship|report`), `type: architecture`, `scope: project`. Same `topic_key` + `scope` → UPDATE, not INSERT.

### Step 3 — Recall before doing (mandatory before new work)

```
1. mem_context — checks recent history (cheap)
2. mem_search(query, match_mode: any|all, scope: project|user|global, all_projects: bool) — FTS5 over observations
3. mem_get_observation(id) — full untruncated body
```

Also search proactively when starting work that may overlap prior sessions, or when the user says any variant of "remember / recall / what did we do / how did we solve". For Skillgrid changes, group searches first (STEP A), then retrievals (STEP B).

### Step 4 — Save during work (mandatory — do NOT wait to be asked)

Call `mem_save` IMMEDIATELY and WITHOUT BEING ASKED after:

- **Decisions/conventions**: architecture or design decision made, team convention documented, workflow change agreed, tool/library choice made with tradeoffs
- **Completed work**: bug fix (include root cause), feature with non-obvious approach, artifact created with significant content, config/environment change
- **Discoveries**: non-obvious codebase finding, gotcha or edge case, pattern established, user preference or constraint learned
- **User confirmation/rejection**: user confirms a recommendation, rejects an approach, expresses a preference, or a discussion concludes with a chosen direction — even if the agent proposed it

Self-check after EVERY task: "Did I or the user just make a decision, confirm a recommendation, express a preference, fix a bug, learn something non-obvious, or establish a convention? If yes, call `mem_save` NOW."

```
mem_save(
  title: "Verb + what",                        # short, searchable, e.g. "Fixed N+1 query in UserList"
  type: decision|architecture|bugfix|pattern|config|discovery|learning|preference|convention,
  content: "**What**: one sentence\n**Why**: motive\n**Where**: files/paths\n**Learned**: gotchas (omit if none)",
  session_id: sid,                              # REQUIRED
  scope: project,                               # default; user|global when appropriate
  topic_key: "stable/key",                      # recommended for evolving topics (upsert)
)
```

Rules:
- Different topics must not share a `topic_key`. If unsure, call `mem_suggest_topic_key` first, then reuse it.
- Read current content with `mem_get_observation` before rewriting an evolved topic.
- Wrap secrets/tokens/PII in `<private>…</private>` — stripped before storage.
- Safety net: end task responses with a `## Key Learnings:` numbered list; passive capture persists it even if `mem_save` was missed.

### Step 5 — Code retrieval: Orientation Ladder + Search Router

Never dump whole directory trees. Prefer:

```
code_status → code_map → symbols / code_outline → code_related → code_read
```

```bash
skillgrid index --dir .                                   # reindex when status is not fresh
skillgrid search "AuthService"                            # intent router: daily path auto-picked, stamped route+provenance
skillgrid search --corpus symbols "AuthService"           # escape hatch: code|grep|mem|symbols|hybrid|semantic
skillgrid code map [--path-prefix DIR]
skillgrid code get-symbol --name Name
skillgrid code impact --name Name                          # blast-radius (additive; does not replace code_get_*)
skillgrid code get-callers|get-callees|get-dependents|get-implementors|get-tests-for|get-type-hierarchy --name Name
skillgrid code index-status
```

- **Freshness**: `fresh` | `lag` | `empty` | `unknown`. Compat `stale=true` means empty-only, not lag. Reindex when not `fresh`.
- **Router**: omit `--corpus` for the daily path (identifier→symbols, `func $NAME`→grep, decision/remember→mem, else code). `--corpus` is the advanced escape hatch, not six equal peers.
- **Provenance**: mem and code are never silently fused — mem results carry `observations`, code-family results never mix mem observations unlabeled.
- **Semantic** sets `degraded=true` explicitly when the Local Code Embedder is unavailable (never silent).
- **Details (folded from code-index):** extractors (Python tree-sitter needs `-tags treesitter`, CGo stays in Extractor Adapters), Matcher dialect v2 (`struct`/`method`/`const`/`enum` + v1; invalid pattern → abort), `code_impact` is additive blast-radius (never replaces `code_get_*`), stable tools (`code_status`, `code_index`, `code_search`, `code_read`, `code_get_*`; lexical baseline `chunks`/`chunks_fts`), export/serve (`skillgrid export --project ID --out DIR` under allowed root; `skillgrid serve` dashboard viz read-only, graph mutate → 405). Full ladder/router/conventions: [references/code-indexing.md](references/code-indexing.md).

### Step 6 — Web research cache (before/after every remote lookup)

```
web_cache_lookup(source: context7|exa|deepwiki|fetch|manual, ...)  # BEFORE the remote call
→ miss: call Context7 / Exa / DeepWiki / WebFetch
→ web_cache_save(source, content, ...) within the same turn        # cap 256 KB — summarize first
"what did we find about X?" → web_cache_search(query) → web_cache_get(id)
```

### Step 7 — Close the session (mandatory before saying "done")

```
1. mem_session_summary(session_id: sid, summary: <structure below>)
2. mem_session_end(session_id: sid, summary: "one-line outcome")
```

`mem_session_summary` structure (required):

```
## Goal
[What we were working on this session]

## Instructions
[User preferences or constraints discovered — skip if none]

## Discoveries
- [Technical findings, gotchas, non-obvious learnings]

## Accomplished
- [Completed items with key details]

## Next Steps
- [What remains to be done — for the next session]

## Relevant Files
- path/to/file — [what it does or what changed]
```

This is NOT optional — if you skip it, the next session starts blind.

**SDD session close:** when a change ends its full pipeline, **`reflect` is the single owner** of `mem_session_summary` + `mem_session_end` — it closes the session after writing `report.md`. Sub-agents in earlier phases (execution, qa, review, ship) emit `## Key Learnings` only and do **not** close the session, so they never double-close it. Outside an SDD change, the active agent closes the session as above.

After compaction / "FIRST ACTION REQUIRED": FIRST call `mem_session_summary` with the compacted content, then `mem_context`, only then continue.

## Subcommands (CLI reference)

| Command | Purpose |
|---|---|
| `skillgrid mcp [--debug]` | stdio MCP server (`mem_*`, `code_*`, `web_*`, `team_*`) |
| `skillgrid serve [--port 7438] [--bind 127.0.0.1] [--dir DIR]` | HTTP API + dashboard (Memory Visualization and Code Graph are read-only; mutate via viz is rejected) |
| `skillgrid index [--dir .] [--watch] [--embeddings]` | incremental code index; `--watch` opts into debounced live indexer |
| `skillgrid search [--corpus …] [--limit 20] [--dir .] <query>` | centralized retrieval with intent routing; JSON with `route` + `provenance` |
| `skillgrid code <get-symbol\|get-signature\|symbols-in-file\|map\|list-projects\|index-status\|impact\|get-callers\|…>` | orient/graph (search lives under `skillgrid search`) |
| `skillgrid setup <opencode\|kilocode\|cursor> [--dry-run]` | install `skillgrid-mnemonic` MCP + agent plugins |
| `skillgrid migrate [--tier]` | backfill tier sidecars |
| `skillgrid trail <recent\|show>` | inspect retrieval trails |
| `skillgrid export --project ID --out DIR` | Obsidian Markdown + viz JSON (allowed-root enforced) |

Full protocol refs: [references/memory.md](references/memory.md) (Skillgrid artifact naming, two-step recovery, upserts), [references/code-indexing.md](references/code-indexing.md) (code index + search router — former `mnemonic-code-index` skill, now folded here).

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "I'll remember it in my head" | The head is the session — it dies at compaction or close. `mem_save` NOW is the only thing that survives; "remembered" means lost. |
| "The session summary is optional, skip it" | Step 7 says it is NOT optional — "if you skip it, the next session starts blind." Goal/Instructions/Discoveries/Accomplished/Next Steps/Relevant Files must all be filled. |
| "I'll search memory later if I need it" | Step 3 is recall *before doing*, mandatory. Re-reading code to find something you already saved is re-derivation, not recall. |
| "This discovery is too small to save" | Small non-obvious findings are exactly what a fresh session cannot re-derive cheaply. If it surprised you, it's worth a `mem_save`. |

## Red Flags

- A session ended without a `mem_session_summary` (or with any of the 5 sections empty).
- A decision, bugfix, or non-obvious discovery was made but never `mem_save`'d.
- A recall was done by re-reading code instead of `mem_context` → `mem_search` → `mem_get_observation`.
- `mem_save` called without a live `session_id` (it fails) or a fresh session started mid-session instead of reusing the sid.
- A remote lookup (Context7/Exa/WebFetch) made without a `web_cache_lookup` first, or not followed by `web_cache_save`.

## Verification

- [ ] `mem_session_start` was called once at session open and its `sid` reused for every `mem_save`.
- [ ] Before new work, `mem_context` (and `mem_search`/`mem_get_observation` when needed) was used to recall prior context instead of re-deriving it.
- [ ] Significant decisions/bugfixes/discoveries were `mem_save`'d with `**What**`/`**Why**`/`**Where**`/`**Learned**` and a live `session_id`.
- [ ] `mem_session_summary` was called at session end with all 5 sections (Goal, Instructions, Discoveries, Accomplished, Next Steps) plus Relevant Files non-empty.
- [ ] Each remote web lookup was bracketed by `web_cache_lookup` (before) and `web_cache_save` (after).
