# ADR Review Manifest

> Copy to `.skillgrid/specs/YYYY-MM-DD-<topic>/adr.md` — one per change,
> alongside `briefing.md` / `blueprint.md` / `tasks.md`. This is the change's
> ADR completion marker: it proves the in-force decisions were reviewed and
> records which new ADR entries (if any) this change introduced. It holds *pointers*
> only — never duplicate an ADR entry's Context / Decision / Consequences here.

- **Change:** `YYYY-MM-DD-<topic>`
- **Status:** completed
- **Review date:** YYYY-MM-DD

## In-Force ADRs Reviewed

<!-- Every currently in-force ADR that constrains this change — derived by
     walking `supersedes` links across the `### In-force set` table in
     .skillgrid/ASSUMPTIONS.md. An ADR is in force when its status is `accepted`
     and no later ADR's `supersedes` names it. List each as
     `ASSUMPTIONS.md § ### ADR-NNNN` with a one-line gist. If none are in force,
     say so. -->

- `ASSUMPTIONS.md § ### ADR-NNNN` — {one-line gist}
- (or) None in force.

## New Durable ADRs Created

<!-- Every ADR entry this change created, with its 4-digit sequence. These are
     pointers to the durable entries in .skillgrid/ASSUMPTIONS.md, not copies of
     their content. -->

- `ASSUMPTIONS.md § ### ADR-NNNN` — {decision, one line}
- (or) None — no major durable architectural decision was introduced by this
  change.

## Supersessions

<!-- If this change superseded a prior in-force ADR, name the pair. The prior
     entry is untouched (IRON RULE); the new entry's `**Supersedes.**` field
     points at it. -->

- `NNNN` supersedes `MMMM` — {why revisited, one line}
- (or) None.
