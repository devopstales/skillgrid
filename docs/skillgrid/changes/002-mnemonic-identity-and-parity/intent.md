# Intent: 002 — Mnemonic Project Identity & Engram-Parity

## Business Problem
Mnemonic's project identity is unstable and fragmented: it is path-bound, not repo-bound, so memories scatter across eight mutually invisible SQLite stores keyed by whichever exact cwd the agent happened to be in. Moving the folder, renaming the checkout, or changing the git remote strands prior memories with no alias to the new key. There is no ambiguity signal, no pinning, no expiry, no duplicate-count/recency lifecycle, no vector/embedding recall, and no tool-name provenance. This is the single biggest real failure and the dominant feature gap versus Engram.

## Target Users & Situations
- **Agent (orchestrator/worker/reviewer)** — needs reliable, stable, cross-checkout project identity and full recall.
- **Any consumer of Mnemonic memory** — needs the corrected, unified, enriched data.
- Urgency: High — the fragmentation and instability affect every use of the memory system.

## Business Rules
- Adopt Engram's proven model: clone-private identity binding, child auto-promote, bounded ambiguity, bounded config walk, seed aliases.
- All new SQL is additive migrations; no existing schema changes.
- MCP tools follow the existing `s.AddTool(toolDef, handlerFunc)` pattern.
- HTTP routes follow the existing `mux.HandleFunc` pattern with bearer-token auth on writes.

## Success Criteria (UAT-level)
- [ ] From `/data/git/AI` the resolver surfaces `AvailableProjects` (not `ai-ba52c523`).
- [ ] Resolving from `/data/git/AI/skillgrid` after `git remote set-url origin X` and after copying the checkout to a sibling path yields the identical store id.
- [ ] A worktree of `skillgrid` resolves to the same store id as the main checkout.
- [ ] `kubedash-skillgrid-test` and a fresh dir-hash store are linked via `project_aliases` so a single `mem_search(all_projects)` returns both.
- [ ] `mem_pin`/`mem_unpin`, `mem_review` (with `expires_at` honoured), and `tool_name` provenance behave per the new columns.
- [ ] With `MNEMONIC_EMBED=1`, `mem_search` merges FTS5 and cosine-over-embeddings via reciprocal-rank fusion.
- [ ] All Go tests pass: `go test ./...`.

## Scope

### In Scope
- Clone-private identity binding, child auto-promote, ambiguity, bounded config walk, seed aliases.
- Cross-store recall and alias unification (`mem_search(all_projects=true)`, `mem_unify`).
- Observation lifecycle parity (`pinned`, `expires_at`, `duplicate_count`/`last_seen_at`).
- Optional embedding recall behind `MNEMONIC_EMBED`.

### Out of Scope
- Cloud sync (Engram `sync_*`) — keep Mnemonic local-first; revisit once identity is stable.
- Rewriting the code index or web cache (both already exceed Engram, which has neither).
- Changing FTS5 to another engine.

## Step Blueprint
- `01-identity-binding`: Establish clone-private identity binding, child auto-promote, ambiguity, bounded config walk, and seed aliases.
- `02-cross-store-recall`: Deliver cross-store recall and alias unification.
- `03-lifecycle-parity`: Deliver observation lifecycle parity.
- `04-embedding-recall`: Deliver optional embedding recall.

## Affected Areas
| Area | Impact | Description |
|------|--------|-------------|
| `internal/mnemonic/project/resolve.go` | Modified | New resolution semantics (binding, children, ambiguity, config) |
| `internal/mnemonic/store/store.go` | Modified | Store open/idempotency under new identity |
| `internal/mnemonic/store/migrations/` | New | `008_obs_lifecycle.sql` and subsequent migrations |
| `internal/mnemonic/service/service.go` | Modified | New service methods (pin/unpin, unify, tool_name) |
| `internal/mnemonic/mcp/` | Modified/New | New and updated MCP tools |
| `internal/mnemonic/http/server.go` | Modified | New and updated routes |
| `docs/skillgrid/agents/glossary/` | Modified | Add/update domain terms |

## Risks
| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Writes into `.git/` (permission caveats) | Med | Match Engram's documented caveats; keep `store.Open` idempotent |
| Two cwds mapping to one id | Med | Idempotent `store.Open`; seed aliases route prior keys |
| Large, multi-part change | Med | Staged rollout; each part builds on the prior |

## Rollback Plan
- All changes are additive; removing the new migrations, tools, and routes reverts to the prior state.
- Existing databases will not re-run removed migrations (migration system checks `index_meta`).

## Dependencies
- None external — all Go stdlib + existing deps (`modernc.org/sqlite`, `mcp-go`).
- `docs/plan/02-mnemonic-identity-and-parity.md` (proposal, already in repo).
