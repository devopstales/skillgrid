# Stage 8 — Intermediate Zoom Out (target: full `blueprint.md`, post-Stage 6+7)

Persona: `zoom-out.md` re-dispatched as a fresh pass (Stage-8 instance, not the
Stage-5 one). Mode: REVIEW.
Fold state (stated for the record): in the Consolidated run the parent folds
Stage 6+7 findings inline before this stage; this report was produced in the same
round as Stage 7, so it audits (a) the document as written for pre-existing
altitude drift, and (b) the *expected* fold of the round-1 findings for the
drift/creep those folds would introduce.
Harness isolation: part 1 enforced by fresh subagent dispatch (separate pass from
Stage 5); part 2 enforced (writes confined to `review/`); part 3 partial —
per-project instruction files not loaded as instructions (stated, not silent);
part 4 enforced (local only).

## Findings

NO FINDINGS

## Fold-drift assessment (vector 5: altitude drift + scope creep from fixes)

- **Event→action mapping table (Stage 7, Major)** — the fold adds a mapping table.
  Altitude-safe if it lands where the seam is pinned: the envelope section
  (contract-line correction) + Task 1 Step 1 table-test rows. Drift risk if it
  grows into a new top-level section with prose explanation — keep it a table +
  one sentence. No scope creep: the table ports briefing R1's mappings, it is not
  a new feature.
- **`**`-aware matcher (Stage 7, Major)** — the fold names a small helper in
  `internal/obs` with table cases. Weeds-level only if it grows a spec of its own
  (quoting rules, case-sensitivity matrices); one pinned signature + 3-4 table
  cases is the right altitude. No scope creep.
- **Exit-1-for-usage-on-`_ingest` (Stage 7, Major)** — a two-line edit to the
  envelope exit-code table + Task 3 Step 1. No drift.
- **Built-in-copy mechanism (Stage 7, Major)** — the fold adds one mechanism
  sentence + one Files-list entry (the `//go:embed` directive location). Watch
  item: do not let the fix balloon into a packaging decision (version stamping,
  sync-repo layout) — the briefing's open item is only "where does install read
  the four source files from". Pin one sentence; leave packaging to the
  memory-plugins change if it surfaces.
- **Remaining Stage 7 Minors/Nits + Stage 6 Nit** — all are in-place one-to-three-
  line corrections (env var rename, Event field, sessions derivation, `all`
  semantics, cursor bound, K1/K6 expansion, banned word). None can lower the
  document's altitude.

Pre-existing-drift check on the document as written: the header, Must-Haves, and
File Structure sit at interface/data-flow altitude; all loop-level detail is
confined to task steps (the correct place for a blueprint). No section reads lower
than the rest; no premature-optimization structures; the two "Assessed, not
one-way" entries are altitude-appropriate assessments, not scope.
