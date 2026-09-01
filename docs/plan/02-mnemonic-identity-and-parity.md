# Mnemonic — Project Identity & Engram-Parity Proposal

Status: **Proposal** (v1, 2026-09-01). Decision needed on P1 scope before spec.

This proposal targets `skillgrid-cli/internal/mnemonic`. It is grounded in the live
state of both memory systems on this machine and a line-by-line read of the two
project-resolution implementations:

- Mnemonic: `skillgrid-cli/internal/mnemonic/project/resolve.go`, `store/store.go`, `store/migrations/*`
- Engram (reference): `engram/internal/project/{detect,resolution,identity}.go` (`~/.engram/engram.db`)

Goal: make Mnemonic the stronger local-first engine by (a) fixing the single biggest
real failure — **unstable, fragmented project identity** — and (b) closing the
high-value feature gaps versus Engram without growing surface area arbitrarily.

---

## 1. Evidence (measured 2026-09-01)

### 1.1 Engram (the reference, `~/.engram/engram.db`)

- One shared SQLite DB. **919 observations, 87 sessions, 391 prompts**, FTS5 intact (`fts_integrity_ok`).
- Projects present: `skillgrid` (22 real obs), `.agents` (29 obs), `mam-test` (877 obs —
  bulk benchmark data, mostly empty titles).
- Schema extras Mnemonic lacks: `pinned`, `expires_at`, `duplicate_count`, `last_seen_at`,
  `embedding`/`embedding_model`/`embedding_created_at` (vector search), `tool_name`,
  and the full `sync_*` suite (cloud).
- Indexes: `idx_obs_dedupe ON (normalized_hash, project, scope, type, title, created_at DESC)`,
  `idx_obs_topic ON (topic_key, project, scope, updated_at DESC)`.
- **Project detection is a 6-case algorithm** (`detect.go:128`): config → **clone-private
  identity binding** → single-child auto-promote → ambiguous (error + candidate list) → basename.
- The binding (`identity.go:34`) is written once into `.git/engram-project-identity.json`
  (`{version, id, project}`) and *reused on every later call, across all clones and linked
  worktrees*. Rename/move/remote-change never changes the memory bucket.

### 1.2 Mnemonic (skillgrid-cli, `~/.skillgrid/mnemonic/*.sqlite`)

- **8 separate SQLite files, 7 of them empty** (0 obs). Only `skillgrid.sqlite`
  (11 obs, 661 index chunks) and `kubedash-skillgrid-test` (6 obs) hold anything.
- At the current cwd `/data/git/AI` — a *parent* of `ai/` containing 23 repos, itself not a repo —
  resolution falls to the **`directory-hash` fallback** → store `ai-ba52c523.sqlite` (0 obs, 0 ses).
  That opaque ID is the *basename `ai` + first-8-hex of `sha256(absPath)`* (`resolve.go:151`).
- Resolution chain (`resolve.go:36`): `.skillgrid/config.json "project"` → `git remote origin`
  basename → `{basename}-{hash8}`. **No child-scan, no clone binding, no ambiguity signal.**
- What Mnemonic already has: `memory_relations` + `mem_judge`, `project_aliases` + `project_migrations`
  (drift routing, migrations 006/007), `review_after` cycle (005), `prompts` (004), code index
  (files+chunks+BM25), web cache (TTL).

### 1.3 The concrete failure this proposal fixes

Because identity is **path-bound, not repo-bound**, memories scatter across mutually
invisible stores keyed by *which exact cwd you happened to be in*. Consequences seen
directly on this box:

1. **Fragmentation** — 8 stores for what is logically ≤3 projects; the agent at
   `/data/git/AI` cannot see the `skillgrid` memories that live one directory up.
2. **Non-stable identity** — move the folder OR rename the checkout OR `git remote set-url`
   → new hash / new name → prior memories stranded with no alias to the new key.
3. **Silent miss** — no `available_projects`/ambiguity result, so a write goes to the
   wrong bucket with no signal the agent can surface to the user (Engram returns a
   `judgment_required`-style ambiguity + recovery token; Mnemonic just picks `ai-ba52c523`).
4. **Config bleed risk** — `configProject` (`resolve.go:50`) walks *up to the filesystem
   root*; an ancestor `.skillgrid/config.json` can claim a repo that has its own identity.
   Engram bounds its config walk to the enclosing repo root (`detect.go:239`).

Parity gaps of note: no `pinned`, no `expires_at` on observations, no `duplicate_count`/
`last_seen_at` lifecycle, no vector/embedding recall, no `tool_name` provenance.

---

## 2. Prioritized proposals

| Pri | Item | Why it's worth it | Rough effort |
|-----|------|-------------------|--------------|
| **P1** | Stable clone-private project identity + bounded ambiguity contract | Fixes §1.3 — the actual daily failure | M |
| **P2** | Cross-store recall & alias unification | Makes existing fragmented stores queryable/merge-able | S |
| **P3** | Observation lifecycle parity: `pinned`, `expires_at`, `duplicate_count`/`last_seen_at` | High recall quality, low surface | S |
| **P4** | Optional embedding recall (behind flag) | Semantic recall on top of FTS5 | L |
| skip  | Cloud sync | Out of scope for local-first v2; revisit later | XL |

Rationale for the order: P1 is root-cause. P2 reaps existing data. P3 is cheap quality.
P4 is the largest, most speculative change, so it last and gateable.

---

## P1 — Stable project identity + bounded ambiguity (the core change)

**Adopt Engram's proven model** in `internal/mnemonic/project/`:

1. **Clone-private identity binding.** On first resolve of a git repo, persist
   `{version:1, id:<uuidv4 hex>, project:<canonical>}` into
   `$(git common-dir)/skillgrid-mnemonic-identity.json` (atomic write, 0600, like
   `identity.go:58`). `project` seeded from `git remote origin` name, else repo-root
   basename, else existing store name. **Every later call reads the binding first** and
   never re-derives from mutable git state. Linked worktrees share one binding via
   `common-dir`, exactly like Engram (`detect.go:354`).
2. **Child auto-promote + ambiguity.** Replicate `scanChildren` (`detect.go:420`): exactly
   one child repo → auto-promote (with a soft warning); `>1` → return
   `ErrAmbiguousProject` with `AvailableProjects`. This directly replaces the blind
   `ai-ba52c523` fallback at parent dirs.
3. **Bounded config walk.** Stop walking past the enclosing repo root (or, outside git,
   only the cwd). Kill the ancestor-bleed risk in `configProject`.
4. **Recovery token.** Mirror Engram's `user_selected_after_ambiguous_project` +
   `ENGRAM_PROJECT`-equivalent (`MNEMONIC_PROJECT`) so the agent/CLI can pick from
   `AvailableProjects` instead of guessing.
5. **Seed aliases.** When a binding is created *and* an existing store already exists under
   the old dir-hash/remote key, auto-`INSERT INTO project_aliases (alias→canonical)` so
   prior observations resolve to the new canonical id. Reuse the existing
   `canonicalForAlias` routing (`service.go:445`) with zero new dispatch.

**Interface sketch** (drops cleanly into `ResolveDetailed`):

```go
type Resolution struct {
    ID                string
    Source            ResolveSource          // config | identity | child | ambiguous | dir
    CommonDir         string                 // set for git/identity/child
    AvailableProjects []string               // non-empty iff ambiguous
    Warning           string                 // e.g. "auto-promoted child: skillgrid"
    Err               error
}
func ResolveDetailed(cwd string) (Resolution, error)
```

**Acceptance (verifiable now):**
- From `/data/git/AI` the resolver surfaces `AvailableProjects=[…repos…]` (not `ai-ba52c523`).
- Resolve from `/data/git/AI/skillgrid` twice, after `git remote set-url origin X` and after
  copying the checkout to a sibling path → **identical** store id.
- A worktree of `skillgrid` resolves to the same store id as the main checkout.
- `kubedash-skillgrid-test` and a fresh dir-hash store are linked via `project_aliases`
  so a single `mem_search(all_projects)` returns both.

**Pros:** one root fix; reuses 3 existing tables; matches the reference implementation we
already trust. **Cons:** writes into `.git/` (needs the same permission caveats Engram
documents); must keep `store.Open` idempotent when two cwds now map to one id.

---

## P2 — Cross-store recall & alias unification

- Add `mem_search(all_projects=true)`: open every store under
  `~/.skillgrid/mnemonic/`, run the same FTS5 query, merge + re-rank by `revision_count`,
  `last_seen_at`, recency. This lets the 8 files become one logical index.
- Expose a `mem_unify` admin tool wrapping the existing `project_migrations`/`aliases` path
  (idempotent, records `migrated_at`) so a user can fold `ai-ba52c523` + `-8a5edab2` +
  `mam-test-e8114cfe` into `skillgrid` or a chosen canonical key in one call.

**Effort S, payoff high** — immediately rescues today's orphaned rows.

---

## P3 — Observation lifecycle parity

Add, via migration `008_obs_lifecycle.sql`:
`pinned INT DEFAULT 0`, `expires_at TEXT`, `duplicate_count INT DEFAULT 1`,
`last_seen_at TEXT`; index `idx_obs_dedupe` identical to Engram. Then:

- `mem_pin` / `mem_unpin` — pinned rows ordered first in context and `mem_search`.
- `mem_review` already reads `review_after`; extend it to auto-bump `last_seen_at` /
  `duplicate_count` on hits (recency-weighted recall) and honour `expires_at` (expire →
  soft-delete, like `deleted_at`).
- Store `tool_name` on save for provenance (which tool surfaced/created the memory).

**Effort S, no new transport surface** (all ride existing `mem_*` tools).

---

## P4 — Optional embedding recall (gateable)

- Column trio matches Engram: `embedding BLOB`, `embedding_model TEXT`,
  `embedding_created_at TEXT`.
- Off by default (`MNEMONIC_EMBED=1`). When on, `mem_search` runs FTS5 **and** cosine
  over embeddings, Reciprocal-Rank-Fusion the two rankings. FTS5 stays the floor so the
  feature degrades gracefully when the embedder is absent.
- Keep the local-first promise: embedder is a pluggable function (local ONNX or a
  `MNEMONIC_EMBED_ENDPOINT`), never a required dependency.

**Effort L — last, and only after P1–P3 land and the identity contract is stable** (the
embedding rows are per-project, so they inherit P1's correctness for free).

---

## Explicit non-goals (this pass)

- Cloud sync (Engram `sync_*`) — keep Mnemonic local-first; revisit once identity is stable.
- Rewriting the code index or web cache (both already exceed Engram, which has neither).
- Changing FTS5 → another engine.

---

## Proposed rollout

1. **P1 + migration 008 + P2** in one change (identity + recall unlock existing data).
2. **P3** tools on top of the new columns.
3. **P4** as a separate flag-gated change.

Open question for decision: **Should the identity file live in `.git/` (Engram parity,
survives worktrees) or in the store dir `~/.skillgrid/mnemonic/` (no `.git` writes)?**
Engram uses `.git/`; I recommend matching it for cross-checkout stability, accepting the
`.git` write permission caveat.
