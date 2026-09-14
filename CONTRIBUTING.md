# Contributing

AISkillGrid is a set of behavior-shaping skills plus the hooks that enforce its invariants. The bar for changing a skill is high: skills are not documentation, they change how the agent acts. The bar for changing a hook is also high: a hook is a guarantee, and a wrong guarantee is worse than no guarantee.

## Quality bar

Every skill change must be:

- **Specific** — actionable steps, not vague advice. The agent follows the words exactly.
- **Verifiable** — clear exit criteria with evidence requirements (a command run, a gate met, an artifact written).
- **Battle-tested** — grounded in a real workflow, not a theoretical ideal.
- **Minimal** — only the content needed to guide the agent correctly. If a section wouldn't change behavior, it doesn't belong.

A good contribution names the behavior it fixes or improves and shows the before/after in the agent's actions, not just the diff.

## What not to do

- **Don't duplicate shared material.** Conventions, the TDD cycle, and the threat matrix live in `_shared/`. Reference it; don't copy it.
- **Don't put process in the description.** The frontmatter `description` states what + when, never the step list.
- **Don't restructure a tuned skill to "look cleaner."** The `Common Rationalizations` tables, `Red Flags` lists, and gate language are load-bearing. Change them only with a reason that changes behavior.
- **Don't commit on a protected branch.** Work on a per-agent branch (`agent-*`); the git hooks enforce it.
- **Don't add AI attribution to commits.** Conventional commits only — no `Co-Authored-By`, `Generated-By`, or model trailers. The `commit-msg` hook blocks them.
- **Don't ship authoring residue.** Creation logs and pressure-test prompts stay out of the skill directory.

## Conventions

- **Provenance.** A skill derived from another source carries a `# based on <source>` comment after the frontmatter. Original work may omit it.
- **Description.** Trigger-based, third person, no process steps (see [docs/skill-anatomy.md](docs/skill-anatomy.md)).
- **Relative links.** Every `../` link in a skill must resolve. Verify before committing.
- **Zone rule (BDD).** Commit `.skillgrid/specs/` changes before the code that satisfies them — never both in one commit. The `precommit-zone-guard` blocks mixed commits.
- **Test pyramid.** `qa` picks the highest test layer that fits each behavior and guards against duplicate coverage. Don't add a test at a higher layer when a cheaper layer covers the same behavior — see [`.agents/skills/qa/references/test-strategy.md`](.agents/skills/qa/references/test-strategy.md).

## Layout

```
.agents/skills/<name>/SKILL.md   # the skill
.agents/hooks/                   # guard implementations
.agents/git-hooks/               # thin shims over .agents/hooks/
.github/prompts/  .github/agents/# canonical command + agent assets (IDE mirror source)
scripts/                         # install-hooks, sync-ide-assets, test-hooks
docs/skill-anatomy.md            # the skill contract
docs/user-guide/                 # the walkthrough
```

## Running the checks

CI runs two checks; run them locally before opening a PR:

```bash
./scripts/sync-ide-assets.sh --check   # canonical prompts/agents intact, IDE mirrors in sync
./scripts/test-hooks.sh                # guard hooks against throwaway repos
```

`test-hooks.sh` needs `git` and `awk` only. Add a case there when you change a hook — a hook with no test is an unproven guarantee.
