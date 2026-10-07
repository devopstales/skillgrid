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

<!-- Every currently in-force ADR that constrains this change — the rows in
     `.skillgrid/ASSUMPTIONS.md` § `## LOCKED` ADR index marked "In force: yes".
     List each as the linked file path with a one-line gist. If none are in force,
     say so. -->

- `.skillgrid/artifacts/04-adr-NNNN-slug.md` — {one-line gist}
- (or) None in force.

## New Durable ADRs Created

<!-- Every ADR file this change created. Pointers only — never copy the body. -->

- `.skillgrid/artifacts/04-adr-NNNN-slug.md` — {decision, one line}
- (or) None — no major durable architectural decision was introduced by this
  change.

## Supersessions

<!-- If this change superseded a prior in-force ADR, name the pair. The prior
     file is untouched (IRON RULE); the new file's `supersedes` frontmatter
     points at it, and the prior table row flips to "In force: no". -->

- `NNNN` supersedes `MMMM` — {why revisited, one line}
- (or) None.
