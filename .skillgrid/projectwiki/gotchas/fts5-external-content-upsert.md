---
type: Gotcha
title: "FTS5 external-content tables desync on the upsert/revision path"
status: draft
sources:
  - ".skillgrid/change-specs/2026-11-23-mnemonic-core/briefing.md Shootdown (2026-11-23)"
---

# FTS5 external-content tables desync on the upsert/revision path

FTS5 external-content indexes (the `obs_fts` table over
`observations(title, content)`) are maintained by triggers. The classic trap:
triggers are written for INSERT and DELETE and **missed on the UPDATE path** —
exactly the path the `topic_key` upsert uses (same `topic_key` + `scope` ⇒
UPDATE + `revision_count++`, not a new row). After a revised save, `obs_fts`
still ranks the old text and the new text is unsearchable.

## The check

- Triggers must fire on all three: `AFTER INSERT`, `AFTER UPDATE` (of
  `title`/`content`), `AFTER DELETE` — the UPDATE trigger must delete-then-
  reinsert the FTS row.
- Regression test (required, from the briefing's testing decisions):
  save → upsert-revise → delete, asserting `obs_fts` matches the base table
  after each step. An upsert that only appends rows, or a revision that only
  UPDATEs the base row, is the failure mode this test exists to catch.

## Generalizes to

Any future external-content FTS table in the design (`chunks_fts`,
`web_fts` for the deferred `code`/`web` tools) inherits the same trap —
especially anything with an upsert or revision semantic.

## Related

- Supersedes: none.
- Consumed by: [[syntheses/mnemonic-core]] (testing decisions),
  [[architecture/mnemonic-memory-core]] (the `obs_fts` table).
- Depends on: [[entities/mnemonic]] (the system the trap lives in),
  [[concepts/in-file-vector-layer]] (same table, sibling vector column).
