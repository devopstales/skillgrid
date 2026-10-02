# ADR Review Manifest

- **Change:** `2026-09-24-mnemonic-memory-improvements`
- **Status:** completed
- **Review date:** 2026-10-02

## In-Force ADRs Reviewed

- `ASSUMPTIONS.md § ### ADR-0005` — MCP process is the trust boundary; search must not widen who can read a private observation.
- `ASSUMPTIONS.md § ### ADR-0006` — vector recall stays in-memory cosine; this change reuses `SearchByVector`, it does not add a SQLite vector extension.
- `ASSUMPTIONS.md § ### ADR-0011` — live search still excludes rows whose `invalid_at` has passed; the new blend keeps that predicate.
- `ASSUMPTIONS.md § ### ADR-0013` — repo artifacts win over session memory.
- `ASSUMPTIONS.md § ### ADR-0016` — `mem_ask` already uses `BlendedSearch` and a `matched_via` idea on the cited floor; `mem_search` grows its own additive signals and does not change `mem_ask`.
- `ASSUMPTIONS.md § ### ADR-0017` — D3 graph decision; unrelated, number collision avoided.

## New Durable ADRs Created

- `ASSUMPTIONS.md § ### ADR-0018` — owner-scoped RRF plus additive `signals` / `matched_via` on `mem_search`; reinforcement decay is query-time and config-gated.

## Supersessions

- None.
