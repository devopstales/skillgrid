# projectwiki MERMAID

- Mermaid for flow/sequence/state/dependency structure; terminal renders ASCII.
- Genuinely un-drawable (layout, mockup, geometry) → SVG via `currentColor` + theme vars; no scripts/animation/background/foreignObject/external href.
- Static, self-contained only.

# projectwiki PAGES

- One page per concept; frontmatter `title`/`type`/`status`; `rel_path` = wikilink target.
- Soft-delete: `projectwiki_pages.deleted_at`; deprecate: `status = 'deprecated'`.

# projectwiki CLAIMS

- Claims: must-hold tokens (subject + literal + anchor).
- `projectwiki validate` pin-check: fresh/stale/unresolved.

# projectwiki CONVENTIONS

- Concepts in `decisions/`/`invariants/`/`workflows/`/`glossary/`; one concept per why.
- Same-edit rule: page edits + `projectwiki index` in one commit (`index/` is engine-written; there is no catalog/log to update by hand).
- Decisions via `skillgrid:architectural-decision-records`.
