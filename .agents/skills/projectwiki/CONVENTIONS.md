# projectwiki Conventions

- Vault layout: `decisions/`, `invariants/`, `workflows/`, `architecture/`, `gotchas/`, `glossary/`, `sources/`, `concepts/`, `syntheses/`, `preferences/` (the ten type folders); `templates/` and `index/` are scaffolding.
- `index/` is engine-written: `last-update.json` today. There is no `catalog.md`, `log.md`, `reverse-index.md`, or `events.jsonl` — never author them.
- Vault-root docs (`ARCHITECTURE.md`, `ASSUMPTIONS.md`, `WINDOWS.md`, `PROJECTWIKI_SCHEMA.md`) are outside index scope: readable by path, invisible to `query`/`lint`. No `reference/` folder exists.
- Page format: frontmatter (`title`, `type`, `status`, optional `aliases`/`sources`/`anchors`/`claims`) + markdown body; pins via anchors.
- Claims: must-hold tokens live in `invariants/` with `claims[]`.
- Mermaid: decision rationale diagrams where the why is structural.

# projectwiki Pages

- One page per thing; `rel_path` is the wikilink target and pages never nest (`<type-folder>/<name>.md`).
- Every page carries 2+ outgoing `[[type/slug]]` links; an ADR carries a `## Related` block (supersedes/amends + 2 dependencies).
- One long page is a smell: split past ~200 lines (the technical glossary is the standing example — per-term pages, not a dictionary).
- Frontmatter drives retrieval filters (`type`, `status`, `grounded`). `description` is not parsed — put the summary in the H1 and first paragraph.
- Soft-delete via `projectwiki_pages.deleted_at`; deprecate via `status = 'deprecated'`.

# projectwiki Claims

- Claims are must-hold assertions bound to a subject + literal.
- Pin-check (`projectwiki validate`) gates reliance on a grounded page.

# projectwiki Mermaid

- Use Mermaid only for genuine flow/sequence/state structure; never for trivia.
