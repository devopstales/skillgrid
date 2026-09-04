# Plan: 003 — Mnemonic Self-Evolving Context Database

## Technical Approach
Add **Tiered Storage**, **Semantic Search**, compaction into **Long-term Memory**, and **Retrieval Trail** on Go MCP + SQLite+FS. After **002** (reuse vector helpers / `MNEMONIC_EMBED`). No **001** tables yet: generic `tiered_contents` + content-write **Seam**; wire teams briefs when 001 lands. SQL at `010_*` (001 keeps `009_*`). No OpenCode plugin; Pure Go embeddings; UAT behavioral.

## Architecture Decisions

### Decision: Generic tier registry
**Module / Interface / Seam / Adapter / Depth**: `tiered`; Generate/Read/Register; content-write seam; FS+SQL; deep.
**Choice**: `tiered_contents` + `{path}.abstract` / `.overview` sidecars.
**Alternatives considered**: `ALTER tasks` now; tier every `mem_save`.
**Rationale**: 001 absent; intent limits compaction to team-task + tier-eligible seam.

### Decision: Pure Go embedder behind flag
**Module / Interface / Seam / Adapter / Depth**: `embedder.Embedder`; Embed; real seam; Local + RemoteAPI; deep.
**Choice**: Option A; `path_embeddings`; reuse 002 blob/cosine.
**Alternatives considered**: Python service; always-on remote.
**Rationale**: Intent; title/L0 floor when flag off.

### Decision: Explicit commit + async tiering
**Module / Interface / Seam / Adapter / Depth**: compaction; commit; 001 completion-hook seam; stub/heuristic Summarizer; deep.
**Choice**: Explicit `mnemonic_commit`; auto only as 001 hook; non-blocking `GenerateTiers`.
**Alternatives considered**: Every session end; sync LLM inline.
**Rationale**: Locked assumptions; latency via bg + `migrate --tier`.

### Decision: L1-default search with trail
**Module / Interface / Seam / Adapter / Depth**: retrieval tools; search/load; trail logger; SQLite; deep.
**Choice**: Ranked L1 (+ L0); L2 only via `load_full_details`; log **Retrieval Trail**.
**Alternatives considered**: Return L2; defer trails.
**Rationale**: Tokens + operator debug.

## Data Flow

```
tier write ──▶ FS L2 ──▶ (bg) GenerateTiers → L0/L1 + SQL
mnemonic_commit ──▶ memory.md + tiers ──▶ long_term_memories
semantic_search ──▶ embed ──▶ path_embeddings/L0 ──▶ L1 + trail
load_full_details ──▶ L2
trail CLI ──▶ retrieval_trails
```

## Impacted Files Map
| File | Action | Step | Description |
|------|--------|------|-------------|
| `skillgrid-cli/internal/mnemonic/store/migrations/010_tiered_context.sql` | Create | 01 | tiered_contents, long_term_memories, retrieval_trails, path_embeddings |
| `docs/skillgrid/agents/glossary/technical.md` | Modify | 01 | New technical terms |
| `docs/skillgrid/agents/glossary/business.md` | Modify | 01 | Long-term Memory |
| `skillgrid-cli/internal/mnemonic/tiered/tiered.go` | Create | 02 | Generate/read L0/L1/L2 |
| `skillgrid-cli/internal/mnemonic/tiered/summarizer.go` | Create | 02 | Summarizer **Interface** + adapters |
| `skillgrid-cli/internal/mnemonic/tiered/hook.go` | Create | 02 | Non-blocking content-write **Seam** |
| `skillgrid-cli/cmd/skillgrid/migrate.go` | Create | 02 | `runMigrate` (`--tier`) |
| `skillgrid-cli/internal/mnemonic/embedder/embedder.go` | Create | 03 | Embedder + local/remote adapters |
| `skillgrid-cli/internal/mnemonic/memory/embedding.go` | Modify | 03 | Share vector helpers |
| `skillgrid-cli/internal/mnemonic/service/service.go` | Modify | 03 | Search/load/trail facade |
| `skillgrid-cli/internal/mnemonic/mcp/tools_retrieval.go` | Create | 03 | semantic_search, load_full_details |
| `skillgrid-cli/internal/mnemonic/mcp/tools_compaction.go` | Create | 04 | mnemonic_commit |
| `skillgrid-cli/internal/mnemonic/service/compaction.go` | Create | 04 | Commit → long_term_memories |
| `skillgrid-cli/internal/mnemonic/mcp/server.go` | Modify | 04 | Register retrieval + compaction |
| `skillgrid-cli/cmd/skillgrid/trail.go` | Create | 05 | trail show\|recent |
| `skillgrid-cli/cmd/skillgrid/main.go` | Modify | 05 | Dispatch migrate + trail |

## Step WHAT

### Step 01-schema-extensions — What it delivers
- As operator: store open creates tier/memory/trail/embedding tables without rewriting existing rows.
- Given existing DB, when migrate runs, observations/FTS/code index stay intact.
- Edge: schema 008 → 010 once; re-open idempotent.

### Step 02-tiered-storage — What it delivers
- As writer via content-write seam: L2 save later yields L0/L1 sidecars + path columns without blocking.
- Given L2 files, `skillgrid migrate --tier` backfills L0/L1; L2 unchanged.
- Edge: summarizer failure leaves L2 intact; error logged.

### Step 03-semantic-retrieval — What it delivers
- As agent: `semantic_search` returns ranked L1 (with abstracts), not L2.
- Given a result path, `load_full_details` returns L2 markdown.
- Edge: embeddings off/empty → title/L0 fallback; trail still recorded.

### Step 04-session-compaction — What it delivers
- As agent: `mnemonic_commit` persists **Long-term Memory** with L0/L1/L2 and optional source link.
- Session end alone does not auto-commit.
- Edge: missing sources → clear error; no partial write.

### Step 05-trail-observability — What it delivers
- As operator: `trail recent` / `trail show <id>` show query, directories, files, result path.
- Empty store → empty list, not error.
- Edge: unknown id → not-found.

## Interfaces / Contracts

```go
type Summarizer interface { Abstract(string) (string, error); Overview(string) (string, error) }
type Embedder interface { Embed(context.Context, string) (memory.Vector, error); Model() string }
type ContentWriteHook interface { AfterContentWrite(ctx context.Context, fullPath, kind, sourceID string) }
// semantic_search → {results:[{overview,abstract,full_path}], trail_id}
// load_full_details{path} → {content}
// mnemonic_commit{task_id?,lessons_learned?,title?} → {memory_id,paths}
```

## Mnemonic Integration
New tools: `semantic_search`, `load_full_details`, `mnemonic_commit`. Existing `mem_*`/`code_*`/`web_*` unchanged. Topic key: `sdd/003-mnemonic-self-evolving-context-database/plan`.

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED tests | Owner step |
|---|---|---|---|---|
| Documentation-like paths | N/A: no executable classification | — | — | — |
| Git repository selection | N/A: no git cwd authority | — | — | — |
| Commit state | N/A: no git commit automation | — | — | — |
| Push state | N/A: no push automation | — | — | — |
| PR commands | N/A: no PR CLI | — | — | — |
| **Mnemonic tool surface** | Applicable — new tools/shapes | Additive only; `mem_*` unchanged; JSON-only helper | `semantic_search` body is L1-only; L2 needs `load_full_details`; `mem_save` still registered | 03, 04 |
| **Shared-convention drift** | N/A: no `_shared/conventions/*` edits this phase | — | — | — |

## Migration / Rollout
Additive `010_*`; leave `009_*` for 001. `MNEMONIC_EMBED` (+ optional remote endpoint); local preferred. Backfill via `migrate --tier`; hooks for new writes. Prefer after 002; teams hook after 001.

## Open Questions
- [ ] Local embedder library (onnxruntime-go vs hash-embedder stub) — pick in step 03; does not block schema/FS.
