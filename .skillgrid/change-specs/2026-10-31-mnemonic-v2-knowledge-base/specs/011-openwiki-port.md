# 011 — OpenWiki Lifecycle Port (run checkpoint, no-op proof, claims, manifest)

**Status:** READY FOR EXECUTION · **Owner:** knowledge-base change
**Depends on:** `003-generate` (the pipeline contract this implements), `004-index`
(checkpoint co-design), `006-validate` (claims + the relocating span token),
`008-transport` (the surface this wires).
**Reference:** langchain-ai/openwiki `src/` (run-state.ts, page-jobs.ts,
page-manifest.ts, claims/brains/code/{paths,store,resolver}.ts, agent/utils.ts) —
the mechanism was reverse-engineered 2026-11-05 from a live init run on this repo
(runId 9e0389a6) and read against its source.

## 1. What this port is

The OpenWiki durable wiki-generation lifecycle, reproduced in the kb function per the
003/004/006 contracts: a **resumable run checkpoint**, a **page-job plan queue**,
**repo-grounded claims with relocating version tokens**, a **per-page baseline
manifest**, a **proven 4-layer no-op**, and a **finish** that leaves either a clean
idle row or an intact resumable checkpoint. The KB's generation source is memory facts
+ live code (003), not the whole-repo markdown walk OpenWiki uses; the durable
skeleton is identical.

## 2. Mechanism map (OpenWiki → kb)

| OpenWiki artifact | kb artifact | Notes |
|---|---|---|
| `openwiki/.run.json` | `kb_run_state` DB row (authoritative) | phases `idle/planning/generating`; `plan` JSON = ordered `PageJob[]` (`pending/skipped/complete`); `processed` JSON = `{sync_id: fact_hash}` (no seq — SaveEvent carries none) |
| `openwiki/.last-update.json` | `index/last-update.json` (file projection) | written by generate/index; **never read back** for decisions (003 §5.1) |
| `openwiki/.page-manifest.json` | `kb_page_baselines` + `index/.page-manifest.json` | baseline earned only when page bytes + pin-check are mutually consistent (003 §5.2) |
| `openwiki/.claims/<page>.json` | frontmatter `claims:`/`anchors:` (truth) → `kb_claims` (derived read side) | KB pages are files: frontmatter is truth, table is projection (KB-INV-1) — the sidecar direction is inverted vs OpenWiki |
| `repo://path#Lx-Ly` + resolver versions | `kb-lines-v1:sha256:<contentHash>:<base64url(meta)>` span tokens + `all_of`/`symbol` anchors | 006 §2.1 decision procedure: hint-hit → relocate (unique, context-anchored) → stale (unique, live value reported, token NOT rewritten) → unresolved (fail closed) |
| source fingerprint (content + git status + HEAD + ignore) | `kb-source-fingerprint-v1` = SHA-256 over format, head, project, fact records (`sync_id\0normalized_hash\0type\0topic_key`, sorted), code records (`path\0content_hash`, sorted) | **source-only**: excludes `index/**` and all `kb_*` rows (003 §3 layer-4 note) |
| noop = clean git ∧ no claim issues ∧ baseline coverage (+post-finalize re-check) | 4-layer proof: (1) processed-set clean, (2) read-only pin re-check, (3) baseline coverage, (4) double fingerprint re-check | 003 §3 |
| resume: drift → replan (plan deleted, manifest proves completions); skipped→pending | identical, plus `processed` survives drift (hash-keyed, changed facts are new) | 003 §5.1 |
| finish: deletions, finalize, manifest replace, `.run.json` deleted last | finish: reindex, baseline replace, clean → `phase=idle` + `plan` cleared + `last_full_run_at` (full runs); failure → checkpoint intact | 003 §5.1 |
| search: per-call FTS5 over H2 units, bm25 weights, term-coverage tiebreak | `kb_query` upgrade is a **separate ticket** (005) — out of scope here | |

## 3. Deliverables (this slice)

Engine (`mnemonic-v2/internal/kb/`), all over the existing `store.Store` seam
(`Exec`/`Query`/`ExecDDL`; no new Store interface methods):

1. `frontmatter.go` — real YAML frontmatter parse/emit (yaml.v3): `title/type/project/
   aliases/anchors/claims/sources/status/stale_after/generated/verified`; engine-owned
   fields (`status/stale_after/generated/verified`) are never agent-authored; `generated`
   advances on body change only.
2. `fingerprint.go` — `SourceFingerprint(ctx, st, project) (string, error)` per §2.
3. `runstate.go` — `Begin` (init/update/resume incl. drift replan + manifest promotion +
   skipped reset), `SetPlan`, job transitions, `Finish(clean)`, file projection writer.
4. `manifest.go` — `RecordBaseline` (only when consistent), `PageIsCurrent`, projection.
5. `pins.go` — span token build/resolve (006 §2.1 exactly), `all_of` token check,
   `symbol` check via the code graph (`SymbolByName`/`EdgesByNameOnlyToName`; graph
   absent → ripgrep-degraded occurrence check, `degraded: true`), claim literal check,
   per-page `ValidatePage` result shape (006 §3).
6. `index_inc.go` — incremental `Index` (body_hash diff; soft-delete removed; parse
   `anchors:`/`claims:` into `kb_claims`; compute wikilink `in_degree`); `Reindex` kept
   as the full path.
7. `generate.go` — graduate-rule (deterministic: importance gate + type map + novelty),
   entity-targeting (sources citing sync_id, entity/topic_key match), `Synthesizer` seam
   (`Synthesize(ctx, SynthInput) (string, error)`; nil/failed → append-only degraded
   floor + deterministic output guard), the 4-layer no-op proof, baseline recording,
   `kb_events` emission, finish.
8. `validate.go` — upgrade to file-based `ValidatePage` (frontmatter anchors/claims vs
   live code), keeping the stored-baseline fast path for the no-op proof.

Surface:

9. CLI (`cmd/mnemonic/kb.go`) — real arg parsing per 008 §6: `generate [--full]
   [--json]`, `index`, `reindex`, `validate [rel_path…] [--strict]`, `query <terms…>
   [--limit N]`, `status`, `bootstrap`, `lint`, `cite-source <rel_path>`, `eval`.
   Exit codes: 0 ok · 1 drift under `--strict` · 2 usage/operational.
10. MCP `kb_*` tools (12, 008 §3) over the shared handler + CLI↔MCP parity test —
    **separate ticket** (KB-T2) so the core lands first.

## 4. Tickets

| # | Ticket | Scope |
|---|---|---|
| KB-T1 | frontmatter + fingerprint | 1, 2 |
| KB-T2 | run checkpoint + manifest + finish | 3, 4 |
| KB-T3 | pins (span token, token/symbol/claim checks) + validate upgrade | 5, 8 |
| KB-T4 | incremental index (claims table, in_degree) | 6 |
| KB-T5 | generate (graduate, target, synthesize floor, 4-layer no-op, finish, events) | 7 |
| KB-T6 | CLI arg wiring + exit codes | 9 |
| KB-T7 | MCP `kb_*` surface + parity test | 10 |

## 5. Invariants (restated)

KB-INV-1 files are truth (frontmatter is the claims sidecar); KB-INV-3 `index/**` +
`kb_*` rows written only by the engine; KB-INV-4 clean run = proven no-op; INV-3
degraded is explicit (append-only, graph-absent, stale index); INV-5 `kb_events` trail
on every page write; the fingerprint is source-only (the run's own writes can never
perturb it).

## 6. Verification floor

- `go test ./internal/kb/...` green, including: span-token relocate/stale/unresolved
  matrix, 4-layer no-op (each layer failing independently), resume-after-drift,
  manifest-promotes-pending, append-only floor, incremental index diff + soft-delete.
- Smoke: `mnemonic kb index` + `kb generate --full` + `kb validate` against a temp
  vault + temp store (test), then one real run on this repo's vault.
