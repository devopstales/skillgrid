# 005 — Transport: CLI

**Status:** DRAFT · **Owner:** memory change
**Depends on:** `000-architecture`, `002-schema`
**Reference:** mnemonic v1 `cmd/mnemonic` dispatch + Engram CLI.

---

## 1. Capability

The CLI is the **human/ops interface**. It is a thin wrapper over the same `Store` /
`intelligence` / MCP-handler layer the MCP server uses — the CLI and MCP never diverge in
behavior (one handler, two surfaces). v2 ships **MCP + CLI only** (no HTTP dashboard, no
SPA — that decision is global to v2).

## 2. Command surface (memory-relevant)

```
mnemonic <command> [flags]

  mcp           Run the MCP stdio server (starts autosync when sync.auto=true)
  serve         (deprecated in v2 — the local HTTP *dashboard* is dropped; retained as
                 a no-op alias that prints a pointer to `mcp`/CLI until v2.1.)

  # memory
  mem save      Save an observation (--title --type --content --scope --topic-key)
  mem search    Search (--type --scope --limit --match-mode --all-projects)
  mem get       Full content by id (--id, --include-history)
  mem lookup    The wiki-first fallback: exact topic_key / id lookup
  mem context   Recent context (--budget, --older-history)
  mem review    list overdue / mark_reviewed
  mem timeline  Before/after in a session
  mem pin / unpin
  mem delete    Soft/hard delete
  mem session   start|end|summary|finalize
  mem stats     Per-type/project stats
  mem doctor    Functional health check (--strict for CI)

  # sync (full sync in memory)
  sync status   Per-target lifecycle, last seq, backoff
  sync push     Push pending mutations to the remote
  sync pull     Pull remote mutations since a cursor
  sync export   Git-friendly export: manifest + compressed chunks (.jsonl.gz)
  sync import   Import a chunk archive
  sync enroll   Enroll/unenroll a project for remote sync

  # infra
  index         (delegates to code-index change)
  trail         Inspect retrieval trails (recent|show)
  migrate       Export v1 observations as seed imports
  doctor        Whole-system health
  setup <agent> (delegates to the plugin changes: opencode|pi)
```

## 3. Conventions

- **Project selection is explicit for reads** (Engram): omitted selectors resolve the
  current project (explicit → `MNEMONIC_PROJECT` → cwd). `--all` reads every project.
  Don't combine an explicit project with `--all`.
- **Local server auth** (when `serve` alias or HTTP mirror is used): `MNEMONIC_HTTP_TOKEN`
  Bearer for mutating routes.
- **Output:** human-readable by default; `--json` for machine. JSON shape matches the MCP
  tool responses (one contract).
- **`--datasource`** overrides the store dialect (SQLite default → Postgres) for the
  invocation (`001-storage` §6, OPEN-4).

## 4. The sync CLI is first-class

Because "full sync in memory" is the decision, `sync *` is a **top-level command group**,
not buried. It mirrors the autosync manager's operations so an operator can push/pull/
export/import manually when autosync is off. The remote is the team's self-hosted
Postgres (one schema per project, `01-memory/006-sync` §2) — `sync push`/`pull` issue
direct SQL to `schema_<project>.mutations`; there is no cloud server to talk to. The
git-friendly export/import (`sync export`/`import`) is Engram's `.engram/manifest.json` +
`chunks/*.jsonl.gz` pattern (chunk id = first 8 hex of SHA-256 of payload), made
dialect-neutral.

## 5. Invariants

INV-2 (dual-mode, `--datasource`), INV-4 (local truth — sync never writes local from
remote without applying through the store). The CLI also **exposes** INV-5 audit trails
via `trail` (the *recording* of trails is enforced by the retrieval layer,
`01-memory/005-lookup` §7 — the CLI only inspects them).

## 6. Open items

None.
