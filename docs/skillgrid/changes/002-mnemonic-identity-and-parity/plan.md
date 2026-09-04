# Plan: 002 — Mnemonic Project Identity & Engram-Parity

## Technical Approach
Fix Mnemonic's unstable, fragmented project identity and close the feature gaps versus Engram. The change has four parts: (1) clone-private identity binding with child auto-promote, bounded ambiguity, bounded config walk, and seed aliases; (2) cross-store recall and alias unification; (3) observation lifecycle parity; (4) optional embedding recall. Each part builds on the prior.

## Architecture Decisions

### Decision: Clone-private identity binding
**Module / Interface / Seam / Adapter / Depth**: project identity; the binding is the sole source of the project id; no alternative adapter.
**Choice**: Bind the project to its clone on first resolve; every later call reads the binding.
**Alternatives considered**: Deriving the id from mutable git state on each call.
**Rationale**: A stable, clone-private id eliminates the path-bound fragmentation and the silent misses.

### Decision: Child auto-promote and bounded ambiguity
**Module / Interface / Seam / Adapter / Depth**: child repos; auto-promotion with a soft warning; ambiguity returns the candidate list.
**Choice**: Exactly one child repo auto-promotes; more than one returns ambiguity with candidates.
**Alternatives considered**: A blind directory-hash fallback.
**Rationale**: Removes the opaque fallback at parent directories and surfaces the candidates.

### Decision: Bounded config walk
**Module / Interface / Seam / Adapter / Depth**: config lookup; bounded to the enclosing repo root.
**Choice**: Stop walking past the enclosing repo root.
**Alternatives considered**: Walking to the filesystem root.
**Rationale**: Kills the ancestor-bleed risk.

### Decision: Seed aliases
**Module / Interface / Seam / Adapter / Depth**: alias routing; prior keys route to the new canonical id.
**Choice**: Auto-insert aliases when a binding is created and an existing store is present.
**Alternatives considered**: No alias seeding.
**Rationale**: Routes prior observations to the new canonical id with no new dispatch.

### Decision: Cross-store recall and alias unification
**Module / Interface / Seam / Adapter / Depth**: all stores; merged, re-ranked recall.
**Choice**: Search every store and merge the results.
**Alternatives considered**: Per-store only.
**Rationale**: Makes the fragmented stores one logical index.

### Decision: Observation lifecycle parity
**Module / Interface / Seam / Adapter / Depth**: pinning, expiry, duplicate count, recency.
**Choice**: Honour the new lifecycle columns.
**Alternatives considered**: No lifecycle.
**Rationale**: Brings recall quality in line with the reference.

### Decision: Optional embedding recall
**Module / Interface / Seam / Adapter / Depth**: vector recall behind a flag.
**Choice**: Off by default; when on, fuse FTS5 and cosine-over-embeddings.
**Alternatives considered**: FTS5 only.
**Rationale**: Adds semantic recall while keeping FTS5 the floor.

## Data Flow
    cwd ──▶ resolve (binding / children / ambiguity / config) ──▶ store id
                                                        │
                          ┌─────────────────────────────┤
                          ▼                             ▼
                     per-store                      all stores
                     (lifecycle, tools)             (cross-store recall)

## Impacted Files Map
| File | Action | Step | Description |
|------|--------|------|-------------|
| `internal/mnemonic/project/resolve.go` | Modify | 01 | New resolution semantics (binding, children, ambiguity, config) |
| `internal/mnemonic/store/store.go` | Modify | 01 | Store open/idempotency under new identity |
| `internal/mnemonic/store/migrations/` | New | 01,02,03,04 | `008_obs_lifecycle.sql` and subsequent migrations |
| `internal/mnemonic/service/service.go` | Modify | 01,02,03,04 | New service methods (pin/unpin, unify, tool_name) |
| `internal/mnemonic/mcp/` | Modify/New | 01,02,03,04 | New and updated MCP tools |
| `internal/mnemonic/http/server.go` | Modify | 01,02,03,04 | New and updated routes |
| `docs/skillgrid/agents/glossary/` | Modify | 01,02,03,04 | Add/update domain terms |

## Step WHAT

### Step 01-identity-binding — What it delivers
- The resolver binds the project to its clone and never re-derives it from mutable git state.
- A single child repo auto-promotes; more than one returns ambiguity with the candidate list.
- The config walk is bounded to the enclosing repo root.
- Aliases are seeded so prior keys route to the new canonical id.

### Step 02-cross-store-recall — What it delivers
- Recall spans every store, merged and re-ranked.
- Fragmented stores become one logical index.

### Step 03-lifecycle-parity — What it delivers
- Pinning, expiry, duplicate count, and recency are honoured.

### Step 04-embedding-recall — What it delivers
- Vector recall is available behind the flag, fused with FTS5.

## Interfaces / Contracts
- The project id is the single stable key for all stores and tools.
- All new SQL is additive; no existing schema changes.
- MCP tools and HTTP routes follow the existing patterns.

## Mnemonic Integration
- New and updated Mnemonic tools and routes for the four parts.
- No change to the existing Mnemonic save shape.

## Threat Matrix
- Routing: applicable (resolver, store, tools, routes).
- Shell/subprocess: not applicable.
- VCS/PR automation: not applicable.
- Executable-file classification: not applicable.
- Process integration: not applicable.
- Mnemonic tool contract: applicable (new/updated tools).
- Shared conventions: not applicable.

## Migration / Rollout
- Phased rollout; each part builds on the prior.
- No feature flags except the embedding flag.
- No data migration beyond the new columns.

## Open Questions
- None.
