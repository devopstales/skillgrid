# 008 — Transport (the `projectwiki_*` MCP surface, the CLI, the event subscription)

**Status:** DRAFT · **Owner:** project-wiki change
**Depends on:** `005-query`, `003-generate`, `006-validate`, `007-lint`, `004-index`,
`001-structure`, the memory change's `004-transport-mcp` (the shared MCP server + the
`SaveEvent`), `005-transport-cli` (the shared CLI dispatch).
**Reference:** Engram's MCP server conventions (profiles, write queue, degradation
envelope), pi-llm-wiki's `wiki_*` tool surface.

---

## 1. Capability

This spec is the **surface**: the `projectwiki_*` MCP tools, the `projectwiki` CLI command group,
and the event-bus subscription that makes the write path eager. It is a **thin adapter**
— it resolves the project, validates the contract, and calls the ProjectWiki's store/pipeline.
It does not reimplement persistence or generation rules.

The ProjectWiki joins the **one MCP server, three tool families** pattern
(`00-foundation/004-transport-mcp` §1 of the memory change): `mem_*` (memory), `code_*`
(code-index), `projectwiki_*` (this change). The plugins (`opencode`, `pi`) consume the `projectwiki_*`
family through the same MCP stdio server.

## 2. Profiles

Same two profiles as the memory change:

- **`agent`** — the tools an agent may call during work (the default for plugins).
- **`all`** — `agent` + admin tools. For ops/CLI.

The ProjectWiki's `agent` profile is the working set (query, read, generate, validate, lint,
status, cite; + the three opt-in trajectory tools when `config.json.trajectories` is
true, 012). `projectwiki_status` is in `agent` because it is a **read-only view of derived
state** (page counts, coverage, freshness) the agent needs to decide when to save /
re-verify — it is *admin-shaped* only in that it reads the derived layer, never writes
it. The write/admin tools (`projectwiki_index`, `projectwiki_reindex`, `projectwiki_bootstrap`, `projectwiki_rebuild_meta`)
are in `all`.

## 3. The `projectwiki_*` tool surface

| Tool | Profile | Purpose |
|---|---|---|
| `projectwiki_query` | agent | **Stage-1 deep retrieval** (wiki-first): ranked links + 1-line snippets; filters `type/project/limit/match_mode/grounded_only`; `thin` flag + `fallback: mem_lookup` hint. Optional `body` field on rows per the `005` §2.3 inline carve-out (`config.json` `recall:`). |
| `projectwiki_read` | agent | **Stage-2:** the full page (or a `##` section); `include_backlinks`; for grounded pages, the pin-check state. |
| `projectwiki_search` | agent | The raw FTS over `projectwiki_pages_fts` (the power path; more rows, longer snippets). |
| `projectwiki_generate` | agent | Run generation: `full: bool` (default incremental, entity-targeted from the latest `SaveEvent`); optional `model` (`provider/id` per-run override, `003` §4 — precedence override > config > session, unknown degrades to the append-only floor). Returns `changed: N`. |
| `projectwiki_validate` | agent | **The pin-check** (`006-validate`): validate one or all grounded pages against live code. `strict: bool` (fail on drift). Returns per-page `status` + `checks`. |
| `projectwiki_lint` | agent | **Deterministic health** (`007-lint`): orphans, broken links, dup aliases, coverage, broken citations, ungrounded, stale index. `fix: bool` (auto-repair the safe ones). |
| `projectwiki_status` | agent | Vault health: page counts by type, coverage, last `projectwiki_generate`/`projectwiki_index`/`projectwiki_lint` times, index freshness, `strict` setting. 012: + stale-page count (`updated_at` vs `stale_after` / 30-day default), 7-day activity by `event_type` (`projectwiki_events`), zero-backlink count — all pure reads, no LLM (the reference's `wiki_dashboard`). |
| `projectwiki_cite_source` | agent | Record an external citation (`external:<name>` + `version` + `source_url`; computes `source_commit` + `build_fingerprint`) on a page's `sources:`. Does not fetch/store the body. |
| `projectwiki_capture_trajectory` (opt-in, 012) | agent | **Write.** Capture the completed task's tool-call sequence into an immutable packet (`trajectories/TRJ-YYYY-MM-DD-NNN/`: `manifest.json`, `packet.json`, `extracted.md`) + a `projectwiki_trajectories` row. One lightweight call, no skeleton page; no session tool log → explicit error, never a silent empty packet. Registered only when `config.json.trajectories` is true. |
| `projectwiki_distill_skills` (opt-in, 012) | agent | **Write.** List undistilled trajectories (`projectwiki_trajectories.distilled_at IS NULL`) for the agent to generalize into `skill`/`case` pages citing `trajectory:TRJ-*`. Distilled promotion is derived: `projectwiki_index` sets `distilled_at` when a citing page is indexed. Registered only when enabled. |
| `projectwiki_recall_skill` (opt-in, 012) | agent | **Read.** "Have I done this before?" — `projectwiki_query` filtered to `type ∈ {skill, case}` + the `005` §2.3 inline carve-out (short bodies inlined, agents skip bare links). Call at task start. Registered only when enabled. |
| `projectwiki_reindex` | admin | **Full** rebuild of `projectwiki_pages`/`projectwiki_pages_fts`/`projectwiki_claims` from the markdown (`004-index`). Idempotent. |
| `projectwiki_index` | admin | **Incremental** reindex (changed pages only); advances the checkpoint. |
| `projectwiki_bootstrap` | admin | Scaffold a new vault (the sentinel `config.json`, `PROJECTWIKI_SCHEMA.md`, `templates/`, the page folders, `.ignore`) and write/refresh a `### ProjectWiki` routing block in the repo root's `AGENTS.md` (marker-delimited; in-place refresh — never duplicated). Idempotent no-op on an existing vault (the AGENTS.md block is still ensured). |
| `projectwiki_rebuild_meta` | admin | Rebuild the derived metadata that is *not* the FTS index — the `projectwiki_run_state` checkpoint + the backlink in-degree cache + the `index/` human projections (`catalog.md` / `log.md` / `reverse-index.md` / `events.jsonl`, 001 §2 — the reference's `meta/` pattern; also rewritten by `projectwiki_index`/`projectwiki_reindex` as they run). (Distinct from `projectwiki_reindex`, which is the FTS.) |

> The tool list is the final set (the plan's draft is confirmed here, minus
> `projectwiki_render` — there is no `outputs/` to render, KB-9): **12 core tools + 3 opt-in
> trajectory tools (012, agent profile, registered only when `config.json.trajectories`
> is true — when off they are absent from `tools/list` entirely: zero prompt cost)**.
> `projectwiki_generate` is in the `agent`
> profile because the agent triggers it proactively (session discipline, `009-skill`);
> it is the ProjectWiki's "save" equivalent.

## 4. Shared server conventions (inherited from the memory change)

All of `00-foundation/004-transport-mcp` §4 apply to `projectwiki_*`:

- **Project resolution at tool-call time** (cwd → config → git binding → basename).
  Write tools (`projectwiki_generate`, `projectwiki_cite_source`; + `projectwiki_capture_trajectory`,
  `projectwiki_distill_skills` when enabled, 012) take a mandatory `expected_project`
  ownership assertion.
- **Write queue.** ProjectWiki writes (`projectwiki_generate`, `projectwiki_cite_source`; + the two trajectory
  writes when enabled, 012) are serialized through a
  shared ProjectWiki write queue (distinct from the memory write queue — the ProjectWiki and memory are
  different functions with different write sets; both are queued, neither blocks the
  other).
- **Degradation in the envelope (INV-3).** When a tool ran degraded (LLM absent →
  append-only generation; graph absent → ripgrep-only pin-check; stale index →
  `degraded: stale_index`), the response carries `degraded: true` + the reason. Never a
  silent floor.
- **Identity for the ProjectWiki is `rel_path`, not `sync_id`.** Unlike memory (where `sync_id`
  is the cross-machine identity, INV-6), the ProjectWiki's stable identity is the **page
  `rel_path`** (a page is a git-tracked file, not a row). `kbpg-…` is a **local row id**
  for intra-store FKs only — it is *not* a cross-machine identity and is never exposed to
  the agent. (The ProjectWiki syncs as *files*; `rel_path` is what makes two machines' vaults
  line up, `004-index` §2.)
- **HTTP mirror.** Every `projectwiki_*` tool has an HTTP mirror (POST/GET routes) for the CLI
  and non-MCP clients. Same contract, same write queue. **Write tools are async:** a
  POST that starts a run (`projectwiki_generate`) returns a `run_id` + `status` immediately; the
  run completes on the ProjectWiki write queue. A repeat POST for an already-active run returns
  `409` (do not double-start) — the client polls or is notified on completion. Read
  tools (`projectwiki_query`/`projectwiki_read`/`projectwiki_status`) are synchronous.

## 5. The event subscription (the eager write path)

The ProjectWiki **subscribes** to the memory `SaveEvent` on the event bus (the boundary,
`004-transport-mcp` §5 of the memory change). The subscription:

- **Is in-process** (same Go binary, event bus — the ProjectWiki does not import the memory
  package, and memory does not import the ProjectWiki).
- **Is async with respect to the save.** The `mem_save` returns before the ProjectWiki's
  generation runs (the two-phase handoff, OPEN-11). A ProjectWiki generation failure never fails
  a `mem_save` (failure isolation, `002-sources` §2.1).
- **Bounded:** the ProjectWiki's regeneration is incremental + entity-targeted (`003-generate`),
  so a single save triggers only the affected pages. The memory side only guarantees the
  event is emitted exactly once per committed save.
- **Retry:** if the ProjectWiki consumer errors, the event is logged + queued for wiki-side retry
  (the ProjectWiki's own concern). The checkpoint (`projectwiki_run_state.processed`, `004-index`) records
  the processed `{sync_id, fact_hash}` set, so a retry is idempotent (it does not
  reprocess a handled event).

> **The `SaveEvent` payload this change consumes** (the dependency on the memory spec
> made checkable — defined in the memory change's `004-transport-mcp` §5): the ProjectWiki reads
> `sync_id` (the fact identity, no `seq`), `project`, `scope`, `type`, `topic_key` +
> `entity` (entity-targeting, `003-generate` §3 step 3), and `importance` (the
> graduate-rule gate, `003-generate` §2). If the memory `SaveEvent` ever drops
> `entity`/`topic_key`/`importance`, `003`'s targeting + graduate-rule silently break —
> this enumeration is the contract the two changes must keep in sync.

> **The event is the only cross-function trigger.** The ProjectWiki is not pulled by the agent
> to "update" — it is *pushed* by every `mem_save`. The agent's role is the *read* path
> (`projectwiki_query`) and the *judgment* (the graduate-rule runs on the event, not on an agent
> poll). This is what "eager-per-save" means concretely.

## 6. The CLI (`projectwiki` command group)

The CLI is a thin wrapper over the same handlers (the CLI and MCP never diverge). The
`projectwiki` group (in `005-transport-cli`'s dispatch):

```
mnemonic projectwiki <subcommand> [flags]

  # read
  query       Stage-1 deep retrieval (--type --project --limit --grounded-only --json)
  read        Stage-2 full page (--path, --section, --backlinks)
  search      Raw FTS (--type --project --limit --match-mode)

  # write / maintain
  generate    Run generation (--full for a full re-derive)
  validate    The pin-check (--strict to fail on drift; --paths to scope to changed files)
  lint        Deterministic health (--fix to auto-repair the safe findings; --strict)
  status      Vault health (page counts, coverage, freshness)
  cite        Record an external citation (--page --url --version)

  # admin (reindex / bootstrap)
  index       Incremental reindex
  reindex     Full rebuild of the FTS index from the markdown
  bootstrap   Scaffold a new vault (idempotent)
  rebuild-meta  Rebuild the checkpoint + backlink cache
```

**Conventions** (inherited, `005-transport-cli`): project selection explicit for reads
(omitted → current project); `--json` for machine output (matches the MCP contract);
`--datasource` overrides the store dialect (SQLite → PG) for the invocation. The
`knowledge validate --strict` and `knowledge lint --strict` are the CI / pre-commit
entry points (`006-validate` §5).

## 7. Invariants

INV-2 (dual-mode, `--datasource`), INV-3 (degradation envelope on every `projectwiki_*` tool),
INV-5 (trails recorded by the query path), INV-7 (the agent curates — `projectwiki_generate` is
the agent's proactive save, the graduate-rule is the filter), PW-INV-3 (the derived
*index* is written only by the engine's reindex path — `projectwiki_index`/`projectwiki_reindex` and
`projectwiki_generate`'s step 5; the agent never hand-edits `index/**` or the `projectwiki_*` tables).

> **PW-INV-3 and the agent profile (the "which is it?" resolved).** Profiles restrict
> **direct agent calls**, not internal pipeline composition. `projectwiki_generate` is
> agent-profile (the agent's sanctioned save path — it writes page markdown *and*,
> internally, the derived index via its step 5); that is *allowed* because `projectwiki_generate`
> **is** the engine's write path, not a bypass of it. `projectwiki_lint(fix=true)` may call
> `projectwiki_index` internally for the stale-index repair. What PW-INV-3 forbids is the agent
> (or a user) editing `index/**` files or the `projectwiki_*` rows *directly* — the derived layer
> is always a projection of the markdown, never a hand-edited source.

## 8. Open items

None. The tool surface, the profiles, the event subscription, and the CLI group are
fixed. The one carried tunable is the `thin` threshold (`005-query`) and the
graduate-rule importance cutoffs (`003-generate`) — both defaulted in `config.json`,
refined in the review.
