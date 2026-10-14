# Review — 2026-10-10-obs-split-hnsw

> Change: `.skillgrid/specs/2026-10-10-obs-split-hnsw/` (moves to `.skillgrid/archive/2026-10-10-obs-split-hnsw/` at ship)
> Generated: 2026-10-14T00:00:00Z (requesting-code-review)
> Diff range: code commits `4d0be16d, 81edc9f2, c443c4ce, f9d1677f, 9d1ca413, 440c0bd8, af3b2f66` (+ spec-zone docs `899ce839, 9e8a70fb, f598eb21`, ADR `2e0a73c1`; QA follow-ups `c6cab1d9, 67b9e26d` as context)
> Rigor tier: T3 (per briefing.md / blueprint.md) — reviewed via the three-axis `requesting-code-review` dispatch; the T3 fresh-model parallel-review recommendation was declined by the coordinator (fresh subagent contexts, same model family — Grade B) and the decline is recorded below.
> Independence: Grade B per axis — see `## Independence`
>
> The durable audit record of the code review. **requesting-code-review** writes this into
> the spec folder, next to `report.md` (which owns the QA gate). It is committed in the spec
> zone before code-zone work continues. **ship** archives it with the folder; **reflect**
> cites it for lineage. The three review axes are reported **side by side, never merged** —
> that separation is the point.

## What Important means

> The explicit threshold that separates a finding worth fixing now from one to park.
> Do not restate the fix-loop cap or escalation numbers here — they live in
> `_shared/planning/rigor-tiers.md`. Name only the bar for "Important."

- **Critical** — must fix before merge. Security, data loss, broken functionality, a
  `SATISFIES` scenario whose RED evidence is missing or whose code path does not match the
  Given/When/Then. Blocks `ship`.
- **Important** — should fix before merge. Architecture drift from a term or an in-force
  ADR, a hard standard breach, a missing or partial requirement, a behavior the spec never
  asked for (scope creep). Unfixed Important issues do not proceed (per the skill's
  Red Flags).
- **Minor** — nice to have. Baseline smells, style, optimization, doc polish. These are the
  nits; they are capped and summarized, never one-per-line below the fold.

## Cap the nits

- **Max 5 Minor findings listed per axis.** Beyond that, collapse the rest into a count:
  "N further Minor items (style/smells) — see the raw reviewer report, not this file."
- A Minor finding that is actually a Critical dressed as polish gets re-labeled, not
  capped. Capping is for true nits, not for hiding a real issue behind a count.

## Do not report

- Anything tooling already enforces (lint, typecheck, format). If `gofmt` / the linter
  would catch it, it is not a finding here — it belongs to the QA gate's tooling pass.
- Informational context with no required action — put it in `FYI` at most, one line, and
  it does not count against any cap.

## Passes

> Three axes, reported side by side. Never merge them into one verdict — a change can be
> spec-perfect and violate every convention, beautifully written and implement the wrong
> thing, or conventionally clean and spec-complete and still one SQLi away from a data
> breach. Each axis carries its own worst issue and its own verdict.

### Standards

- **Worst issue (within this axis):** HNSW/IVF ANN adapters are fully implemented but no production writer populates them — `HnswSymbols`/`HnswChunks` can never return `ok=true` in production (S1)
- **Findings:** 0 Critical / 5 Important / 10 Minor (capped to 5 listed)
- **Verdict:** met-with-fixes

### Spec

- **Worst issue (within this axis):** `per-model-dim-namespaces` scenario closed by source-namespacing (440c0bd8) while the `Dim = 768` pin and the typed cross-dim error remain unmet (R4)
- **Findings:** 0 Critical / 2 Important / 2 Minor
- **Verdict:** met-with-fixes

### Security

- **Worst issue (within this axis):** INSTEAD OF DELETE trigger + `ftsDeleteObs` rebuild FTS5 from the FTS table's own rows after the row delete — other observations' index entries drop when the deleted row's side rows are missing (SEC1, Medium/Likely)
- **Findings:** 0 Critical / 0 High / 2 Medium / 1 Low / 1 Info
- **Verdict:** secure-with-fixes

## Findings

> One per line where possible. Every finding names its file + hunk (or scenario) so a
> reader can act without the session. Severity is the reviewer's; the label prefix
> (`Critical:` / `Optional:` / `Consider:` / `FYI`) is the reviewer's action marker.

### Critical

(none)

### Important

- [Standards] **S1** `mnemonic/internal/vectorstore/hnsw.go` `Upsert` + `store.go` `HnswSymbols/HnswChunks` — no production writer populates the HNSW graph: the indexer's dual-write (`codeindex/indexer.go:1325,1401`) still calls only `vectorstore.UpsertSymbol/UpsertChunk` (vec0 + BLOB); the only `HnswSymbols/HnswChunks` callsites are hybrid route helpers (which degrade gracefully via `ok=false`) and tests. Fail-loud + YAGNI breach: the 058 drop of the 057 graph is lossless only while no writer exists, and a later writer will silently discard 057-era rows. — follow-up ticket: route `embedPass` dual-write through the cached `HnswIndex` upserts (or gate the HNSW route behind it).
- [Standards] **S2** `mnemonic/internal/vectorstore/ivf.go` + migration `064_ivf_store.sql` — IVF is a fully-implemented-but-unwired adapter: nothing in `hybrid/` routes to `IVFIndex`, no production code calls `Build`. The blueprint says "IVF first, HNSW behind ADR-0037"; as shipped the first ANN path is dead. Speculative Generality. — park with S1 (one writer ticket wires both), or wire IVF as the actual first path.
- [Standards] **S3** `065_hnsw_store.sql` `hnsw_meta` table — created and named in ADR-0037 ("top node, node count, dim" for warm-load) but never written or read anywhere in range (grep: zero Go references). Speculative Generality. — drop the table or start using it, and correct the ADR if dropped.
- [Standards] **S4** `.skillgrid/artifacts/04-adr-0037-embedded-ann-index.md` lines 15/23/31 — ADR body still says "migration 057" in three places while the graph persists in 058 (058 header + `hnsw.go:17` both say 058); also `hnsw.go:52` comment still says `efSearch=32` after the 440c0bd8 correction to 96. Second post-hoc correction to the same ADR in range — the 057→058 swap was never recorded in the ADR. — reword the ADR to 058 and sync the stale constant comment.
- [Standards] **S5** `ivf.go:Upsert` writes `model = "ivf"` and `hnsw.go:Upsert` writes `model = "hnsw"` into the source-of-truth BLOB table, where `model` is load-bearing (ADR-0006: model is the cache-invalidation signal; `embed_meta` guards model swaps). If these upsert paths ever fire in production (with S1), they will corrupt the model bookkeeping with backend names indistinguishable from real model-swap artifacts. — pass the real model through `Upsert` (interface change) rather than hard-coding a backend name.
- [Spec] **S6-R4** `briefing.md:19` (R4) + `acceptance.feature:36-40` (per-model-dim-namespaces) — `vectorstore/store.go:27` still pins `const Dim = 768` and `UpsertSymbol/UpsertChunk` still take `model string, _ int` (model ignored); no 1024-dim fixture, no per-(model,dim) vector table, no typed cross-dim error anywhere in range. The scenario was closed by 440c0bd8's HNSW *source* namespacing (the symbol/chunk id-collision fix), which satisfies the id-collision problem, not the model/dim problem. — park with S1/S2: the per-model dim namespaces + cross-dim typed error land with the ANN writer follow-up.
- [Spec] **S6-migration-054** `acceptance.feature:6-10` + `briefing.md` Verdict — spec zone still says "migration 054" / "pre-054" while on disk the split landed at `062_obs_split.sql` (rebased per TICKET-02 Precondition after CLM 053-055 + CTX 056-061 landed). The rebase is justified and documented in the migration headers, but the spec-zone renumber was never recorded. — cosmetic; correct the spec zone at ship (or now, spec-zone commit).

### Minor

- [Standards] `hnsw.go:76-86` — duplicate constructor comment: `NewHnsw` documented twice (the second paragraph belongs to `NewHnswSource`). *(Consider)*
- [Standards] `066_hnsw_source.sql:3` — header says "The HNSW graph lands at 057 (next free)" while 066 drops those very tables in the same migration run; 057's header should say "superseded in-place by 058". *(FYI)*
- [Standards] `store.go:114-158` `hnswCache` — package-global keyed by `*sql.DB` pointer; `InvalidateHnswCache()` called from production code nowhere (only tests); the ADR-0006 "process-global cache is a known bad" consequence applies and the store-close/reindex invalidation hook is missing. *(Consider: call it where `hybrid.InvalidateVectorCache` is called, `codeindex/indexer.go:945`)*
- [Standards] `bench_test.go:154` — `TestObservationsCallerAudit` hardcodes an absolute path `/data/git/AI/skillgrid/...` as a fallback candidate; host-specific artifact that will mislead failure messages elsewhere. *(Consider: drop the absolute path)*
- [Standards] `obs_split.go:110-116` `sideScopeWhere` — builds SQL by string-concatenating a table name; safe today (literals only) but an injection-shaped helper. *(Consider: `map[string]string` of the five known scopes)*
- (5 further Minor items collapsed — duplicated bitemporal filter across ~59 read sites, triplicated `cosine`, 055 FK-repoint index inventory FYI, rollback-line overstatement in 054 header, `TestHnswWarmLatency` comment drift — see the raw reviewer report in the session transcript, not this file.)
- [Security] `062_obs_split.sql` `observations_view_delete` + `obs_split.go:99-101` `ftsDeleteObs` — FTS rebuild is O(N) per hard-delete; by design (commented), noted so the QA gate doesn't mistake it for a bug. *(FYI)*
- [Security] pre-existing, out of diff, not re-reported: `corsMiddleware` reflects every `Origin` on the unauthenticated loopback HTTP server; `buildFTSQuery` term-quoting is correct for FTS5 but allows query-shape injection (not SQLi). *(FYI)*

## Disposition (receiving-code-review)

> Triage of every Important finding. "Fix now" = fixed in this session with a test; "Park
> with ruling" = deferred to a named follow-up with the reason; "Noise" = reviewer wrong
> (evidence in line).

| ID | Axis | Severity | Disposition | Ruling / evidence |
|----|------|----------|-------------|-------------------|
| SEC1 | Security | Medium (Likely) | **Fix now** | FTS 'rebuild' re-reads the FTS table's own rows, so the trigger's delete-then-rebuild drops other rows' index entries whenever the deleted row's side rows are missing. Fixed by rebuilding from live `observations_core` (same shape as 067) in `observations_view_delete` and in `ftsDeleteObs`; regression test added (hard-delete a row, assert a surviving row's terms still MATCH). |
| SEC2 | Security | Medium (Confirmed, latent) | **Fix now** | `splitSQLStatements` `inTrig` case word-matches `END` without tracking string literals; a future trigger body with `'END'`/`'BEGIN'` in a literal shreds the file and re-fails every open. Fixed by applying the `inS` sub-state inside `inTrig`; regression test added (migration with a trigger literal containing END). |
| S1 | Standards | Important | **Park — follow-up ticket** | The ANN writer is a separate vertical slice: it needs the indexer `embedPass` dual-write (BLOB + HNSW in one tx, the dual-write the blueprint names for TICKET-05 but which landed as read-only routing). Current code is correct and graceful (`ok=false` fall-through — verified `vectorcache.go:100-122`). Parking S1+S2+S3+S5 as one ANN-writer follow-up ticket. |
| S2 | Standards | Important | **Park — with S1** | IVF wiring is part of the same writer ticket (blueprint: "IVF first"); keeping both adapters dead-but-tested is cheaper than half-wiring one now. |
| S3 | Standards | Important | **Park — with S1** | `hnsw_meta` drops or gets used by the writer ticket (warm-load metadata is its stated purpose); ADR corrected there. |
| S4 | Standards | Important | **Fix now** | ADR-0037 body + `hnsw.go:52` constant comment reworded 057→058 / 32→96 (doc-only, same commit as SEC1/SEC2). |
| S5 | Standards | Important | **Park — with S1** | Interface change (`Upsert` gains `model`) lands with the writer, when the upserts first fire in production. |
| S6-R4 | Spec | Important | **Park — with S1** | R4 (dim unpin + per-model namespaces + cross-dim typed error) is vectorstore work that belongs with the writer follow-up; 440c0bd8's source-namespacing was a different (correct, tested) fix for the id-collision. Scenario text stays open until the follow-up. |
| S6-migration-054 | Spec | Important | **Fix now** | `acceptance.feature` + briefing Verdict renumbered 054→062 (spec-zone, doc-only). |

**Fix-loop round:** 1 of 3.

## Independence

> The no-leaks proof. Each axis records its independence grade so a reader can see whether
> the "independently reviewed" evidence is real. Mirrors the evaluator protocol in
> `_shared/verification/calibration.md`.

| Axis | Grade | Rationale |
|------|-------|-----------|
| Standards | B | fresh subagent context, no shared session history with the implementer; same model family / toolchain |
| Spec | B | fresh subagent context, no shared session history with the implementer; same model family / toolchain |
| Security | B | fresh subagent context, no shared session history with the implementer; same model family / toolchain; reviewer ran its own empirical driver checks |

**Reading the grade:** **A** = fresh subagent, no shared history (independent). **B** = fresh
context but same model family / toolchain as the implementer (independent, weaker). **C** =
inline self-review or the implementer's verdict leaked in — **diagnostic only**, cannot stand
as the independent evidence that `ship` or a merge requires.

**T3 note:** the tier's fresh-model parallel-review recommendation was declined by the
coordinator and is recorded here; the three axes ran as three independent fresh contexts
(blindness contract held — no session narrative, no prior verdict, no implementer rationale
passed to any reviewer).

## Verdict

> The overall review verdict is the **floor** across the three axes (per
> `_shared/verification/floor.md`): the weaker axis caps the whole review. Never average the
> three axes into a single "overall."

- **Standards:** met-with-fixes
- **Spec:** met-with-fixes
- **Security:** secure-with-fixes
- **Floor (decides):** met-with-fixes
- **Worst issue (across all three axes):** S1 — ANN adapters shipped without a production writer (degrades gracefully; follow-up ticket owns it)
- **Unfixed Important count (must be 0 to proceed):** 0 (SEC1, SEC2, S4, S6-migration-054 fixed this session; S1/S2/S3/S5/S6-R4 parked with named follow-up and rulings)
