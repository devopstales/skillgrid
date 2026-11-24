# Contributing

Skillgrid is a set of behavior-shaping skills plus the hooks that enforce its invariants. The bar for changing a skill is high: skills are not documentation — they change how the agent acts. The bar for changing a hook is also high: a hook is a guarantee, and a wrong guarantee is worse than no guarantee.

## Quality bar

Every skill change must be:

- **Specific** — actionable steps, not vague advice. The agent follows the words exactly.
- **Verifiable** — clear exit criteria with evidence requirements (a command run, a gate met, an artifact written).
- **Battle-tested** — grounded in a real workflow, not a theoretical ideal.
- **Minimal** — only the content needed to guide the agent correctly. If a section wouldn't change behavior, it doesn't belong.

A good contribution names the behavior it fixes or improves and shows the before/after in the agent's actions, not just the diff.

## What not to do

- **Don't duplicate shared material.** Conventions, rules, the commit contract, and craft rules live in `_shared/`. Reference it; don't copy it.
- **Don't put process in the description.** The frontmatter `description` states what + when, never the step list (see `_shared/craft/skill-anatomy.md`).
- **Don't restructure a tuned skill to "look cleaner."** The `Common Rationalizations` tables, `Red Flags` lists, and gate language are load-bearing — the agent walks the spine top to bottom, and moving a section changes when a constraint bites. Change them only with a reason that changes behavior.
- **Don't commit on a protected ref or a detached HEAD.** The `precommit-guard` git hook denies it. Per-agent branches must match the allow-list regex (`SKILLGRID_AGENT_BRANCH_REGEX`).
- **Don't add AI attribution to commits.** Conventional commits only — no `Co-Authored-By`, `Generated-By`, or model trailers. The `commit-msg` hook blocks them.
- **Don't ship authoring residue.** Creation logs and pressure-test prompts stay out of the skill directory.

## Conventions

- **Provenance.** A skill derived from another source records it in `metadata.based_on` — a citation of the upstream skill(s) it was folded from, written once and **never edited after the fact**. Original work omits it.
- **Frontmatter.** Exact field set and order: `name`, `description`, `license`, `metadata`. `name` is kebab-case and equals the directory name; `description` is third person, trigger-specific, no workflow steps, ≤ 1024 chars; `metadata.version` is quoted `MAJOR.MINOR`; `metadata.part-of: skillgrid`. The mechanical check is `check-frontmatter.mjs` (below).
- **Description.** Trigger-based, third person, no process steps — the description is the only thing the model sees before loading the body, so a summary of the workflow creates a shortcut the agent will take.
- **Cross-references.** Use canonical `skillgrid:<name>` form, the name as the skill exists *now*. A renamed or folded skill is cited only in `based_on` and the merge note — never as a live reference. A rename is a whole-skill sweep: every occurrence in `SKILL.md` and every `references/*.md` changes in the same edit.
- **`_shared/` paths must resolve in this repo's `_shared/`** (subdirs: `craft/`, `rules/`, `references/`, `planning/`, `verification/`, `work-unit-commits/`, `adr-records/`, `templates/`, `schemas/`, `scripts/`). A `_shared/` path that resolves only elsewhere is a drift signal — copy the file here or reword the pointer.
- **Relative links.** Every link in a skill must resolve. Verify before committing.
- **Zone rule (BDD).** Commit `.skillgrid/change-specs/` changes before the code that satisfies them — never both in one commit. The `precommit-zone-guard` blocks mixed commits.
- **Relative references.** `SKILL.md` is a routing table, not a manual: one-line pointers to `references/` for branch-specific detail, one concern per reference file, one level deep (no `SKILL.md → a.md → b.md` chains).

## Layout

Skills are **flat**: one level deep, no group folders.

```
.agents/skills/<name>/SKILL.md      # the skill (flat; name == directory)
.agents/skills/<name>/references/   # docs loaded on demand
.agents/skills/<name>/scripts/      # executable code (no token cost when loaded)
.agents/skills/_shared/             # shared bundle (non-skill): craft/, rules/,
                                    #   references/, planning/, verification/,
                                    #   work-unit-commits/, adr-records/,
                                    #   templates/, schemas/, scripts/
.agents/agents/                     # subagent role files (dispatch briefs)
git/hooks/                          # git hook entrypoints (thin shims) + guard
                                    #   implementations (checkpoint-state.js
                                    #   dispatcher, precommit-*, ticketing-*, …)
omp/hooks/                          # agent tool-call hooks: gate-state.js,
                                    #   gate-stop.js, stop-tests.js (+ post/)
plugins/omp/skillgrid-guards/       # write/edit guards as an omp plugin:
                                    #   spec-zone fileset, STATUS banner
                                    #   ownership, strict artifact order
.agents/skills/skill-creator/       # the meta skill: audit + check scripts
docs/                               # the walkthrough (read in numeric order)
.github/workflows/                  # CI
```

Git hooks fire at commit time; the `skillgrid-guards` plugin fires at write/edit time (the moment of violation, not audit time); the `omp/hooks/` gate hooks enforce gate state at Stop. The git hooks are installed (copied) to `~/.skillgrid/git-hooks/` by `skillgrid install`, so the same implementation runs from the repo and from the user home.

## Running the checks

Run these before opening a PR (CI mirrors them — see `.github/workflows/`):

```bash
# frontmatter contract (field set + order, name == dir, description ≤ 1024,
# quoted MAJOR.MINOR version, part-of) — exit 0 when all skills conform
node .agents/skills/skill-creator/scripts/check-frontmatter.mjs check .

# SKILL.md size budget (byte ceiling per tier; see _shared/skill-size-budget.json)
node .agents/skills/skill-creator/scripts/skill-size-budget.mjs check .

# the write/edit guards: unit suites + the self-validation backstop
cd plugins/omp/skillgrid-guards && bun install && bun test && bun scripts/backstop.mjs
```

**The size budget is a ratchet.** `_shared/skill-size-budget.json` records each skill's last-known SKILL.md size and a per-tier ceiling (standard 22,000 / large 38,000 / xl 40,000 / external 90,000 bytes). The check measures SKILL.md only — `references/` is not counted, which is where growth should go. When a check fails, either trim the content or raise the skill's tier in the JSON and re-baseline with `--write`. A silent overage is a drift signal: the skill grew without a deliberate re-tier.

**A hook with no test is an unproven guarantee.** When you change a git hook or a guard, add a case to `scripts/test-hooks.mjs` (landing — see the repo backlog) so the guard is proven against throwaway repos. The guards plugin ships its own `bun test` suites plus `scripts/backstop.mjs` (negative half: a deliberately broken write must be blocked; positive half: canonical writes must pass) — run both when touching it.

**CI.** `.github/workflows/pi-worktree-ci.yml` builds and tests `plugins/omp/pi-worktree` (bun install/test + coverage) on push/PR for that path. Add a workflow for new guard plugins the same way — a guard without CI is a guarantee only on your machine.

## Auditing a skill

Before a skill ships (or after a significant edit), the 10-rule audit runs (four authoring phases — Trigger, Structure, Steering, Pruning — plus cross-cutting concerns). Findings are reported as a table: Rule / Status (PASS/WARN/FAIL) / Finding / Fix. The full checklist is in `skill-creator/references/ten-rule-audit.md`; the format contract every skill's SKILL.md must obey is `_shared/craft/skill-anatomy.md`.
