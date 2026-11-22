# ASSUMPTIONS

Durable project understanding. First file a reader opens. Tiers:
VERIFIED (observed facts), INFERRED (hypotheses, not yet verified),
LOCKED (ADR index — pointer rows only), Open Questions.

## VERIFIED

- The repo is `skillgrid-skills` (remote `git@github.com:devopstales/skillgrid.git`):
  a docs-first skill family — markdown skills under `.agents/skills/`,
  shared policy under `.agents/skills/_shared/`, repo docs under `docs/`.
- Skills are tested by `test-*.mjs` node scripts in each skill's
  `scripts/` dir; scenario evals live in `evals/`.
- `wt` (worktrunk) is the tier-2 worktree tool per
  `.agents/skills/_shared/work-unit-commits/README.md` § Tooling tiers;
  tier 1 is a harness-native tool, tier 3 is `git worktree add`.
- The worktree standard's constants (branch allow-list, `refs/skillgrid/`
  scratch namespace, 50/session cap, 10MiB / 200-file filters,
  `{{ repo_path }}/.worktree/{{ branch | sanitize }}` template) are the
  contract the `pi-worktree` plugin must implement (R13 contract-drift
  guard enforces the direction).
- Active change `2026-11-22-pi-worktree-plugin`: the plugin builds in
  `plugins/pi/pi-worktree` **inside this repo** — programmer override of
  the briefing's original "new sibling npm repo" assumption (2026-11-22).

## INFERRED (HYPOTHESIS)

- The `plugins/` family pattern (one dir per harness plugin, shared
  conventions) will be formalized as the first plugin (`pi-worktree`)
  ships; the manifest/README/changelog conventions it sets become the
  family precedent.
- `wt` is installed on the programmer's machine (stated 2026-11-22);
  it was not on PATH in this session — re-verify at execution setup.

## LOCKED

| Decision | Path |
|---|---|
| _none yet_ | _decisions/NNNN-slug.md_ |

## Open Questions

- Minimum `wt` version pin for the `WORKTRUNK_DIRECTIVE_CD_FILE`
  protocol (confirm exact release at build step 2, record in plugin
  README).
- Does the family want a shared `@devopstales/pi-*` manifest
  convention doc (first plugin sets the precedent)?
- Where does `wt` live? (not on PATH at spec time)
