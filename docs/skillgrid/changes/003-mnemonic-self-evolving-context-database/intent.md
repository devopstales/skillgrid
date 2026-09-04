# Intent: 003 — Mnemonic Self-Evolving Context Database

## Business Problem
Mnemonic is flat: full-token loads, fragmented search, no durable compaction, opaque retrieval. Upgrade to tiered L0/L1/L2, semantic retrieval, session compaction, and trail observability — without rewriting Go MCP + SQLite+FS.

## Target Users & Situations
- **Agent** — overview-first recall; full details on demand.
- **Operator** — inspect directories/files when recall fails.
- Urgency: High; builds on **002** (embeddings) and **001** (teams).

## Business Rules
- Modules only; keep Go MCP + SQLite+FS hybrid.
- Pure Go embeddings (Option A); no Python service. Feature-flag provider — offline local preferred, external API optional.
- Additive SQL; content on FS with path columns.
- Default search returns L1; L2 only via `load_full_details`.
- Compaction: team-task briefs/outputs (001) + tier-eligible seam — not every `mem_save`.
- `mnemonic_commit` explicit first; auto only as 001 completion hook — not every session end.
- Auto-tiering must not block writes.
- OpenCode plugin deferred; UAT behavioral (no hard token-%); glossary terms defined in design.

## Success Criteria (UAT-level)
- [ ] Tier-eligible writes create `.abstract` (L0) + `.overview` (L1) and path columns.
- [ ] `skillgrid migrate --tier` backfills L0/L1 without corrupting L2.
- [ ] `semantic_search` returns ranked L1 and records a retrieval trail.
- [ ] `load_full_details` returns L2 markdown for a search path.
- [ ] `mnemonic_commit` writes long-term memory with L0/L1/L2 and optional source link.
- [ ] `skillgrid trail show|recent` shows query, directories, files, result path.
- [ ] `go test ./...` passes for touched packages.

## Scope

### In Scope
- Schema: tier paths, `long_term_memories`, `retrieval_trails`, embeddings.
- Tiered FS + `migrate --tier` + background content-write hooks.
- MCP: `semantic_search`, `load_full_details`, `mnemonic_commit`.
- Trail logging + `skillgrid trail` CLI.

### Out of Scope
- OpenCode plugin; Python embedding service (Option B).
- Rewriting FTS5, code index, or web cache; cloud sync.

## Step Blueprint
- `01-schema-extensions`: Additive SQL for tier paths, long-term memories, trails, embeddings.
- `02-tiered-storage`: L0/L1/L2 FS, tiered module, `migrate --tier`, background hooks.
- `03-semantic-retrieval`: Pure Go embeddings + search/load tools + trail logging.
- `04-session-compaction`: `mnemonic_commit` → long-term memories with tiers.
- `05-trail-observability`: `skillgrid trail show|recent` CLI.

## Affected Areas
| Area | Impact | Description |
|------|--------|-------------|
| `skillgrid-cli/internal/mnemonic/store/migrations/` | New | Additive schema (≥009+) |
| `skillgrid-cli/internal/mnemonic/` (tiered) | New | Generate/read L0/L1/L2 |
| `skillgrid-cli/internal/mnemonic/memory/` | Modified | Embeddings, compaction |
| `skillgrid-cli/internal/mnemonic/mcp/` | Modified | Retrieval/commit tools |
| `skillgrid-cli/` CLI | Modified | `migrate --tier`, `trail` |

## Risks
| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Summarization cost/latency | High | Background hooks; migrate; test stubs |
| Couples to teams (001) | Med | Content-write seam; wire when present |
| Embedding quality | Med | Feature-flag; title+L0 fallback |

## Rollback Plan
Remove new migrations, MCP tools, CLI, tiered helper. L2 stays; drop `.abstract`/`.overview` if needed.

## Dependencies
- Prefer after **002**; teams hooks after **001**.
- Source: `docs/plan/03-mnemonic-self-evolving-context-database.md`.
- Intent approved by user 2026-09-04.
