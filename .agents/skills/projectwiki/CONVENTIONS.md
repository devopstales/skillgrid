# projectwiki Conventions

- Vault layout: `decisions/`, `invariants/`, `workflows/`, `glossary/`, `reference/`.
- Page format: frontmatter (`title`, `type`, `status`) + markdown body; pins via baselines.
- Claims: must-hold tokens live in `invariants/` with claims[].
- Mermaid: decision rationale diagrams where the why is structural.

# projectwiki Pages

- One page per concept; `rel_path` is the wikilink target.
- Frontmatter drives retrieval filters (`type`, `status`, `grounded`).
- Soft-delete via `projectwiki_pages.deleted_at`; deprecate via `status = 'deprecated'`.

# projectwiki Claims

- Claims are must-hold assertions bound to a subject + literal.
- Pin-check (`projectwiki validate`) gates reliance on a grounded page.

# projectwiki Mermaid

- Use Mermaid only for genuine flow/sequence/state structure; never for trivia.
