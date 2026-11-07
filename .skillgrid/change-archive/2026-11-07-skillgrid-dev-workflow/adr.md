# ADR Review Manifest

- **Change:** `2026-11-07-skillgrid-dev-workflow`
- **Status:** completed
- **Review date:** 2026-11-07

## In-Force ADRs Reviewed

- `.skillgrid/projectwiki/decisions/04-adr-0013-repo-source-of-truth.md` — the repo is the source of truth; the workflow must be a repo artifact, not a session-local plan.
- `.skillgrid/projectwiki/decisions/04-adr-0019-decisions-are-files.md` — ADRs are files; the per-change manifest holds pointers only.
- `.skillgrid/projectwiki/decisions/04-adr-0048-hybrid-sot.md` — the `.skillgrid/` Markdown bundle is the SoT for the project wiki; workflows live in `workflows/` (the placement zone for this change).
- `.skillgrid/projectwiki/decisions/04-adr-0051-agent-style-code-style-group.md` — agent-style v0.9.0 prose standard applies to the page the change writes.

## New Durable ADRs Created

- None — no major durable architectural decision was introduced by this change (doc placement is trivially reversible and follows ADR-0048 directly).

## Supersessions

- None.
