# 004 — Transport: MCP stdio

**Status:** DRAFT · **Owner:** memory change
**Depends on:** `000-architecture`, `002-schema`, `003-intelligence`
**Reference:** Engram's 27-tool MCP surface + 23-tool `agent` profile
(`internal/mcp/agent_contract.go`, `docs/contracts/agent-contract-v1.json`).

---

## 1. Capability

The MCP stdio server is the **primary agent interface**. It exposes the memory function
as a set of tools. It is a **thin adapter** — it resolves the project, validates the
contract, and calls the `Store` / `intelligence` seams. It does not reimplement
persistence rules (Engram's memory-core invariant).

**One MCP server, three tool families** (the plugins in `04`/`05` consume this):
`mem_*` (memory — this change), `code_*` (code-index), `wiki_*` (llm-wiki). This spec
covers the `mem_*` family and the shared server conventions.

## 2. Profiles

Two profiles (Engram's pattern):

- **`agent`** — the tools an agent may call during work. This is the default for
  plugins. (≈23 memory tools.)
- **`all`** — `agent` + admin tools. For ops/CLI.

Admin-only (in `all`, not `agent`): `mem_delete`, `mem_stats`, `mem_timeline`,
`mem_merge_projects`, `mem_review`.

## 3. The `mem_*` tool surface (this change)

| Tool | Profile | Purpose |
|---|---|---|
| `mem_save` | agent | Save a structured observation. Dedupe, topic upsert, `capture_prompt` side-effect (**tri-state, amended 2026-11-01**: `true`/`false`/absent — absent defaults to `true`; the handler surfaces it as `*bool` so "absent" is never confused with `false`), **conflict-candidate detection** returning `candidates[]` + `judgment_required`. Emits the **save event** for the eager wiki handoff (§5). |
| `mem_search` | agent | FTS (+ vector when armed) search; filters `type/project/scope/limit/match_mode/all_projects`; compact rows with relation annotations. |
| `mem_get_observation` | agent | Full untruncated content by id; `include_history`/`history_cursor` for version paging. |
| `mem_lookup` | agent | **The wiki-first fallback** (INV from `000`): a narrow raw-fact lookup by id or exact `topic_key`. This is the "memory half" of wiki-first/memory-fallback. |
| `mem_search_prompts` | agent | Search stored user prompts. |
| `mem_context` | agent | Recent context (sessions/prompts/observations) with a byte budget (16–64 KiB), pinned ≤20, `older_history`. |
| `mem_save_prompt` | agent | Store a user prompt; feeds process-local context for `capture_prompt`. |
| `mem_capture_passive` | agent | Extract `## Key Learnings:` items from text into observations. |
| `mem_update` | agent | Partial update by id; mandatory `expected_project` ownership assertion. |
| `mem_judge` | agent | Record a verdict on a pending `judgment_id` (relation/reason/evidence/confidence). |
| `mem_judge_many` | agent | Batch 1–32 judgments in one queued write. |
| `mem_compare` | agent | Proactive semantic comparison → creates a judged relation directly (idempotent per pair). |
| `mem_consolidate` | agent | Preview/confirm merge of ≥2 observations (fingerprint-gated, destructive). |
| `mem_suggest_topic_key` | agent | Family-heuristic topic key suggestion. |
| `mem_session_start` | agent | Register a session (`{id, project, directory, ownership_mode, resume, isolated}`). |
| `mem_session_end` | agent | End a session (optional free-text summary). |
| `mem_session_summary` | agent | Persist the six-section structured summary (Goal/Instructions/Discoveries/Accomplished/Next Steps/Relevant Files). |
| `mem_session_finalize` | agent | **Atomic** summary + close + idempotent retry journal. |
| `mem_current_project` | agent | Project discovery. |
| `mem_list_projects` | agent | List known projects. |
| `mem_pin` / `mem_unpin` | agent | Local-only pin (search boost + context ordering). |
| `mem_delete` | admin | Soft/hard delete; mandatory `expected_project`. |
| `mem_review` | admin | `list` overdue / `mark_reviewed` (restart decay clock). |
| `mem_timeline` | admin | Before/after in a session. |
| `mem_merge_projects` | admin | Merge project spelling variants. |
| `mem_doctor` | admin | Diagnostics (embed round-trip, capabilities). |
| `mem_stats` | admin | Per-type/project statistics. |

> The tool count differs from Engram's 27 because v2 adds `mem_lookup` (the
> wiki-fallback) and `mem_consolidate`/`mem_compare` are agent-profile in v2 (the LLM
> is first-class, so the agent does more semantic work in-profile).

## 4. Shared server conventions

- **Project resolution at tool-call time** (Engram): cwd → config → git binding →
  basename. Update/delete tools take a mandatory `expected_project` ownership assertion
  (a consistency check, not authn/authz).
- **Write queue.** A shared memory write queue serializes MCP writes (Engram
  `internal/mcp/write_queue.go`). `mem_judge_many` is one queued write.
- **Degradation in the envelope (INV-3).** When a tool ran degraded, the response
  carries `degraded: true` + the reason. Never a silent floor.
- **`sync_id` is the id the agent sees** (INV-6) where cross-machine identity matters;
  the local numeric id is internal. `mem_get_observation` accepts both.
- **HTTP mirror.** Every tool has an HTTP mirror (POST/GET routes) for the CLI and
  non-MCP clients. Same contract, same write queue.

## 5. The save event (the eager wiki handoff)

`mem_save` is the **only** tool that emits a save event. After the durable fact write
commits, it emits (in-process, event bus — *not* a dependency on the wiki package):

```go
type SaveEvent struct {
    SyncID     string   // the new/updated observation
    Project    string   // resolved canonical project
    Scope      string
    Type       string
    TopicKey   string   // the entity hint for entity-targeted regeneration
    Entity     string   // best-effort entity name (from topic_key family / title)
    Importance float64
    Degraded   bool
}
```

The llm-wiki function subscribes to this event and runs its **incremental,
entity-targeted** regeneration. The memory function **does not import** the wiki
package — the event bus is the boundary. This is how "eager-per-save" (OPEN-11) is
realized without inverting the dependency: memory emits, wiki consumes.

**Failure isolation:** if the wiki consumer is absent or errors, the save still succeeds
and the event is logged + queued for retry (the wiki's own concern). A wiki failure never
fails a `mem_save`.

## 6. Invariants

INV-3 (degradation envelope), INV-6 (sync_id), INV-7 (agent saves, engine stores).

## 7. Open items

None.
