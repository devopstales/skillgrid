# Locked decisions are ADR files; ASSUMPTIONS.md holds the path

---
status: "accepted"
supersedes: none
date: 2026-10-02
---

## Context and Problem Statement

On 2026-09-29 a consolidation was written into `.skillgrid/ASSUMPTIONS.md` under the heading `### ADR-0012`. That number already belonged to `.skillgrid/artifacts/04-adr-0012-sqlite-as-second-brain.md`. The inline text said:

> Consolidate `.skillgrid/artifacts/00-prd.md`, the ADR files (`04-adr-*.md`), `03-adr-index.md`, and `05-locked-constraints.md` into `.skillgrid/ASSUMPTIONS.md`. The ADR records are inlined. The source files are `git mv`'d to `.skillgrid/archive/`.

The file grew to 52 KB. Every session reads it. The ADR bodies were the growth, and 0001–0012 plus 0016 already existed as files, so the bodies were stored twice. Number 0012 in the in-force table pointed at the consolidation text while the glossary and ADR-0016 cite 0012 as the SQLite store decision.

## Considered Options

- Keep inlining every ADR body in `ASSUMPTIONS.md`
- Keep both the file and the inline body
- One file per locked decision; `ASSUMPTIONS.md` stores only the path

## Decision Outcome

Chosen option: "One file per locked decision; `ASSUMPTIONS.md` stores only the path", because the in-force set is what a session needs, and the record is what you open when the decision is in scope.

Each locked decision is `.skillgrid/artifacts/04-adr-NNNN-slug.md`. The `### In-force set` table in `.skillgrid/ASSUMPTIONS.md` has one row per decision and a Record column that is the path. The row carries number, title, status, supersedes, date, and in-force. It does not carry the body. Supersede by adding a new file and flipping the prior row to `no`. Never delete a file or a row.

Number 0012 remains `04-adr-0012-sqlite-as-second-brain.md`. The 2026-09-29 consolidation text is preserved in this record and is no longer in force.

### Consequences

- Good, because `ASSUMPTIONS.md` stays an index a session can read, and each decision has one file.
- Bad, because a reader follows a path to see Context and Consequences, and `GET /docs/adrs` reads the files rather than headings inside `ASSUMPTIONS.md`.
