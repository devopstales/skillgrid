# kb MERMAID

- Mermaid for flow/sequence/state/dependency structure; terminal renders ASCII.
- Genuinely un-drawable (layout, mockup, geometry) → SVG via `currentColor` + theme vars; no scripts/animation/background/foreignObject/external href.
- Static, self-contained only.

# kb PAGES

- One page per concept; frontmatter `title`/`type`/`status`; `rel_path` = wikilink target.
- Soft-delete: `kb_pages.deleted_at`; deprecate: `status = 'deprecated'`.

# kb CLAIMS

- Claims: must-hold tokens (subject + literal + anchor).
- `kb validate` pin-check: fresh/stale/unresolved.

# kb CONVENTIONS

- Concepts in `decisions/`/`invariants/`/`workflows/`/`glossary/`; one concept per why.
- Same-edit rule: concept + index.md + log.md + reverse-index.md in one commit.
- Decisions via `skillgrid:architectural-decision-records`.
