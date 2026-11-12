# projectwiki PAGES

- Author pages inside the SDD lifecycle; never as drive-by edits.
- Ten types, ten folders: `decisions/` `invariants/` `workflows/` `architecture/` `gotchas/` `glossary/` `sources/` `concepts/` `syntheses/` `preferences/`. A write outside its folder is rejected at review.
- One file per thing: no nested page directories, no growing monolith pages (split past ~200 lines).
- Every page needs 2+ outgoing `[[type/slug]]` links. In `decisions/`, that is a `## Related` block: what it supersedes/amends, and two pages it depends on.
- Revise via sparse decisions: confirm / revise-same-id / retract; carry unaffected content forward.
- A vault-root doc is not a page: `ARCHITECTURE.md` / `ASSUMPTIONS.md` / `WINDOWS.md` are unindexed. Durable knowledge becomes a typed page.

# projectwiki CLAIMS

- Claims are must-hold tokens tied to a subject + literal.
- `projectwiki validate` runs the pin-check before edits; `projectwiki eval` guards retrieval regressions.

# projectwiki MERMAID

- Flow/sequence/state/dependency structure only; terminal renders ASCII.
- Genuinely un-drawable → SVG via `currentColor` + theme vars; no scripts/animation/background.
