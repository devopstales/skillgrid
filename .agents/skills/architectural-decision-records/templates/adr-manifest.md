# ADR Review Manifest

> Copy to `.skillgrid/specs/YYYY-MM-DD-<topic>/adr.md` — one per change,
> alongside `briefing.md` / `blueprint.md` / `tasks.md`. This is the change's
> ADR completion marker: it proves the in-force decisions were reviewed and
> records which new ADRs (if any) this change introduced. It holds *pointers*
> only — never duplicate a repo ADR's Context / Decision / Consequences here.

- **Change:** `YYYY-MM-DD-<topic>`
- **Status:** completed
- **Review date:** YYYY-MM-DD

## In-Force ADRs Reviewed

<!-- Every currently in-force ADR that constrains this change — derived by
     walking `supersedes` links across .skillgrid/adr/. An ADR is in force when
     its status is `accepted` and no later ADR's `supersedes` names it. List
     each as `.skillgrid/adr/NNNN-slug.md` with a one-line gist. If none are
     in force, say so. -->

- `.skillgrid/adr/NNNN-slug.md` — {one-line gist}
- (or) None in force.

## New Durable ADRs Created

<!-- Every repository-level ADR this change created, with its 4-digit sequence.
     These are pointers to the durable records in .skillgrid/adr/, not copies
     of their content. -->

- `.skillgrid/adr/NNNN-slug.md` — {decision, one line}
- (or) None — no major durable architectural decision was introduced by this
  change.

## Supersessions

<!-- If this change superseded a prior in-force ADR, name the pair. The prior
     file is untouched (IRON RULE); the new file's `supersedes` field points at
     it. -->

- `NNNN` supersedes `MMMM` — {why revisited, one line}
- (or) None.
