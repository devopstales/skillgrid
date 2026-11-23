# Adopt the `.agents/` shared layout, the adversarial-authoring pattern, and the minimal grill-me skill from intent-driven-template

---
status: accepted
date: 2026-11-22
supersedes: none
---

## Context and Problem Statement

The `intent-driven-template` repo (Fission-AI / intent-driven-dev, OpenSpec 2.0
based) solves three problems skillgrid also has, in a shape worth borrowing:

1. **Role briefs are duplicated per harness.** The template keeps four role
   files (`senior-dev`, `senior-qa`, `adversarial-author`,
   `adversarial-reviewer`) once in `.agents/agents/`; each platform dir
   (`.opencode/agents/`, `.codex/agents/`, `.github/agents/`,
   `.claude/agents/`) holds a 3-line pointer. skillgrid's role briefs live
   under `.agents/agents/` (the Goldfish protocol personas), but the four
   *work roles* the `execute` skill dispatches (`senior-dev`, `senior-qa`,
   `adversarial-author`, `adversarial-reviewer`) have no canonical home —
   the dispatch prompts in `execute`'s `references/implementer-prompt.md`
   and `task-reviewer-prompt.md` restate the role each time.

2. **No cross-model author/review round.** The template's
   `adversarial-authoring` skill runs a two-stage subagent round (author →
   reviewer → reconciliation) and writes a `*.council.md` audit trail beside
   the artifact. skillgrid has the Goldfish 7-lens shootdown (brainstorm)
   and the 5-stage plan review (blueprint), but neither is a *two-stage
   author/review of a single artifact draft* with a persistent audit trail —
   the Goldfish lenses review the final artifact, not a draft before it is
   written.

3. **No 5-line skill shape.** The template's `grill-me` skill is five
   lines of prose: "Interview relentlessly, one question at a time,
   codebase-explore over asking, each question with a recommended
   answer." skillgrid's `brainstorm` skill (396 lines) covers the full
   interview, but there is no minimal skill for *grilling an existing
   draft* (a briefing, a blueprint, a design) without the brainstorm
   machinery.

## Considered Options

- **Adopt all three** (shared layout + adversarial-authoring + grill-me):
  the three are independent and each solves a real gap.
- **Adopt the shared layout only** (rename/relocate role briefs): the
  cheapest option; leaves the council-notes convention and the grill-me
  skill unborrowed.
- **Adopt adversarial-authoring only**: the highest-leverage single
  option; the shared layout and grill-me can wait.
- **Adopt none, document the comparison**: the "Not adopted" option —
  record what was looked at and why, without changing skillgrid.

## Decision Outcome

Chosen option: **Adopt all three**, because each solves a real gap and
the three are independent — a failure of one does not block the other
two. The shared layout is the cheapest and the most load-bearing (the
other two roles reference it); adversarial-authoring is the highest
leverage (a new capability, not a relocation); grill-me is the smallest
(a five-line skill that gives `brainstorm` a lightweight alternative for
the "grill this existing draft" case).

Adopted, per `_shared/rules/adoption-decisions.md` (the method, not the
plugin):

1. **The `.agents/agents/` shared layout** for the four work roles.
   New files: `.agents/agents/senior-dev.md`,
   `.agents/agents/senior-qa.md`,
   `.agents/agents/adversarial-author.md`,
   `.agents/agents/adversarial-reviewer.md`. The `execute` skill's
   dispatch templates (`references/implementer-prompt.md`,
   `references/task-reviewer-prompt.md`) will name the role file in the
   dispatch prompt instead of restating the role (a follow-up change —
   this adoption creates the files; the dispatch-template sweep is a
   separate commit).

   *Superseded (2026-11-23):* the one-file-per-role layout was
   consolidated into two files — `.agents/agents/roles.md` (work roles:
   `senior-dev`, `senior-qa`, `adversarial-author`, `adversarial-reviewer`)
   and `.agents/agents/goldfish.md` (protocol + the Goldfish lens and
   synthesizer briefs, formerly `.agents/agents/elephant-goldfish/`).
   Dispatch prompts name the section, not the file.

2. **The `adversarial-authoring` skill** at
   `.agents/skills/adversarial-authoring/SKILL.md`. New skill, v1.0,
   adapted to skillgrid conventions (the `part-of: skillgrid`
   frontmatter, the `_shared/` cross-references, the council-notes
   naming aligned to skillgrid's artifact names — `briefing.md` →
   `briefing.council.md`, etc.). The completion gates and the
   primary-only fallback are carried over verbatim from the template;
   they are the load-bearing part.

3. **The `grill-me` skill** at `.agents/skills/grill-me/SKILL.md`. New
   skill, v1.0, five lines of prose. It is the lightweight
   alternative to `brainstorm` for the "grill this existing draft"
   case — the output is the resolved decision tree, which feeds
   `brainstorm` (new plan) or the `briefing.md` interview log
   (existing plan).

### Consequences

- Good, because the role briefs have a canonical home; the dispatch
  prompts stop restating the role and instead name the file.
- Good, because the adversarial-authoring round gives a persistent
  audit trail (the `*.council.md`) for the draft stage — the Goldfish
  lenses cover the final artifact, the council notes cover the draft.
- Good, because grill-me gives `brainstorm` a 5-line sibling for the
  "grill this existing draft" case; the full interview machinery is
  not the right tool for a 20-minute design stress-test.
- Bad, because the four new role files + two new skills + one
  `INSTALL_TEMPLATE.md` + one ADR is a 8-file change; the
  dispatch-template sweep (making `execute`'s prompts reference the
  role files) is a separate commit and until it lands, the role files
  are unreferenced by the dispatch templates.
- Bad, because the `adversarial-authoring` skill's council-notes
  convention (`<basename>.council.md`) is new to skillgrid; the
  `pre-commit` zone guard does not know about it yet, and a
  `briefing.council.md` committed alongside a `briefing.md` in the
  spec zone is fine, but a `briefing.council.md` committed alongside a
  code file is a zone mix — the guard will catch it, but the skill
  should name it (it does, in the "Council Notes" section).

## Not adopted (with revisit conditions)

Per `_shared/rules/adoption-decisions.md`, the Not-adopted list is the
committed answer to "why didn't we use X?" for a later session.

- **The multi-CLI parity matrix** (the template's per-harness
  "Adding Another Agent" + per-harness model/effort sections in
  `README.md`). skillgrid is single-harness (Pi) at present; the
  parity matrix is worth ~200 lines of README per harness added.
  Revisit when a second harness (Claude Code, OpenCode, Codex CLI)
  is supported — the adoption is then a README section, not a
  skill change.
- **The `spec-as-source` layer** (the template's opt-in layer that
  uses fenced Gherkin in `spec.md` as the source for generating
  acceptance tests, plus the BDD zone rules, plus the
  `acceptance-test-authoring` skill, plus the `stack:` config line).
  skillgrid's `acceptance.feature` is the traceability oracle (the
  BDD gate in `execute`); the `spec-as-source` layer would make
  `acceptance.feature` *generated* from `spec.md` fenced Gherkin
  rather than authored. The BDD zone rule (specs-zone vs code-zone
  never mixed, `tasks.md` zone-neutral) is already enforced by the
  `pre-commit` zone guard in `_shared/work-unit-commits/README.md` —
  no change needed. Revisit when the Gherkin-in-spec pattern is
  wanted for a specific change; the zone rule is already there.
- **The `openspec-git-discipline` skill** (the template's
  proposal-reaches-main-before-apply + merge-before-archive
  discipline). skillgrid's phase model is different (the spec zone
  is `.skillgrid/change-specs/`, the archive move is `ship` Step 6,
  the base is per `chain.strategy`); a direct port would fight the
  phase model. Revisit as a `git-discipline` skill (state table +
  gates + red flags + phrasing) when the phase-transition git state
  gates are wanted as a first-class skill — the shape is worth
  borrowing, the content is skillgrid-specific.
- **The `c4-diagrams` skill** (the template's C4-model ASCII/Mermaid
  diagrams). skillgrid's `blueprint` skill uses Mermaid directly
  (per `_shared/rules/repo-mapping.md`); a C4-specific skill would
  duplicate the Mermaid conventions. Revisit when C4's four levels
  (Context / Container / Component / Code) are wanted as a
  first-class artifact type.
- **The `glossary` skill** (the template's opt-in glossary
  maintenance). skillgrid's glossary is the
  `projectwiki/glossary/01-business-terms.md` +
  `02-technical-terms.md` pair, maintained by the `projectwiki`
  skill (one page per term, never appended to a growing
  dictionary). Revisit when the template's "companion glossary
  references for artifacts" pattern is wanted.
- **The `gherkin-authoring` skill** (the template's Gherkin
  authoring). skillgrid's Gherkin is authored in `blueprint`
  Phase 5 (the acceptance spec) per the `acceptance-test-authoring`
  contract folded into `blueprint`. Revisit when the
  `spec-as-source` layer is adopted (above).
- **The `architectural-decision-records` skill** (the template's
  ADR authoring). skillgrid's ADR contract is
  `_shared/adr-records/` (cited, not restated); the skill was
  deleted and its contract lives in the shared bundle. Revisit
  never — the contract is already home.
- **The `test-driven-development` skill** (the template's TDD
  skill). skillgrid has `test-driven-development` (RED → GREEN →
  TRIANGULATE → REFACTOR) and `test-driven-verification` (evidence
  before "done"). Revisit never — the skills exist.
- **The `openspec-*` lifecycle skills** (`openspec-new-change`,
  `openspec-propose`, `openspec-continue-change`, etc.). These are
  generated from the OpenSpec CLI; skillgrid's phase model is
  skillgrid's own. Revisit never — different pipeline.
- **The `INSTALL_TEMPLATE.md` pattern** (the template's
  agent-readable install doc). Adopted in this change as
  `INSTALL_TEMPLATE.md` at the repo root — the agent-readable
  install doc for adopting skillgrid into an existing repo. The
  shape (inspect the target, copy missing, merge these specific
  files, verify, summarize exactly what changed) is the load-
  bearing part; the content is skillgrid-specific.
- **The `AGENTS.md` as 619 bytes** (the template's minimal
  `AGENTS.md` that points to the shared skills). skillgrid's
  `AGENTS.md` is the full preamble (Key Directories, Security
  Boundaries, etc.) written by onboarding; a 619-byte `AGENTS.md`
  would lose the repo-specific context. Revisit never — the
  preamble is the onboarding deliverable.

## Related

- Supersedes: none (first ADR in this repo's `decisions/` folder).
- Depends on: `_shared/adr-records/README.md` (the ADR contract —
  the IRON RULE, the numbering, the `supersedes` field),
  `_shared/rules/adoption-decisions.md` (the adoption method —
  the method-not-plugin test, the Not-adopted list, the smoke
  test, the rename sweep), `_shared/work-unit-commits/README.md`
  (the zone rule the council notes must respect).
- Consumed by: `execute` (the dispatch templates will name the role
  files — a follow-up change), `brainstorm` (the grill-me skill is
  the lightweight alternative), the onboarding flow (the
  `INSTALL_TEMPLATE.md` is the install doc).
