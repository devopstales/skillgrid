---
id: TASK-040
title: Adopt skillgrid-specific improvements from mattpocock-skills v1.3
status: ready-for-human
assignee: []
created_date: '2026-10-05 12:45'
updated_date: '2026-10-05 13:07'
labels:
  - skills
  - planning
  - inspiration
dependencies: []
priority: medium
type: enhancement
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Plan to selectively port the most valuable, skillgrid-specific ideas from mattpocock/skills v1.3.0/1.3.1. Each item is adapted to skillgrid's own conventions (`.skillgrid/` artifacts, `skillgrid:{name}` refs, `skill-anatomy.md` contract) rather than copied verbatim. Source of inspiration: ~/git/ai-test/mattpocock-skills (v1.3.1).

## In scope (4 items)
1. New skill `environment-retro` (lifecycle) — retrospective that improves the agent ENVIRONMENT, not the code.
2. Invocation invariant in skill-anatomy.md (user-invoked vs model-invoked + no-skill-calls-user-invoked rule).
3. "Invoke the Skill tool with `skillgrid:{name}`" for operative cross-skill references.
4. Merge-danger (door + blast radius) in document PR body.

## Out of scope (noted, deferred)
- Harness-neutrality (agents/openai.yaml, AGENTS->CLAUDE symlink) — we are OpenCode-centric.
- GLOSSARY-MAP.md multi-context glossaries — single-context today.
- chief-of-staff long-horizon coordinator.
- Renaming terms files to GLOSSARY.md — our 01-/02-terms files already dodge the context-window collision.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 environment-retro skill exists under .agents/skills/lifecycle/, conforms to skill-anatomy.md, and is wired into reflect's handoff + sdd-structure.md
- [x] #2 skill-anatomy.md documents user-invoked vs model-invoked + the invariant that a skill can never invoke a user-invoked skill
- [x] #3 cross-skill reference convention distinguishes operative (invoke Skill tool) from passive (name) references
- [x] #4 document skill's pr type enforces a merge-danger section (one-way/two-way door + blast radius)
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Adopted the four most valuable, skillgrid-specific ideas from mattpocock/skills v1.3.0/1.3.1 (adapted, not ported).

1. New skill `.agents/skills/lifecycle/environment-retro/SKILL.md` (based_on mattpocock-skills:retro): sweeps the agent ENVIRONMENT, never the code, across six categories (navigation, automated checks, coding standards, steering no-ops, tool economy, information access). Core rule: a mechanical mistake gets a deterministic check; the standards file is for judgement calls only; a repo with no guardrail is itself a finding. Wired in as a reflect sub-phase (Step 3.6) via operative cross-reference, added to the sdd-structure phase table, and a `## Environment Retro` section added to the shared report template (between Surprises and Acceptance Verdict).

2. skill-anatomy.md: new "Invocation class" section (user-invoked vs model-invoked, dual description) + the invariant that a skill can never invoke a user-invoked skill. Tagged the five human gates with disable-model-invocation: true (using-skillgrid, onboarding, brainstorming, interviewing, reflect). Added a new-skill checklist row and an audit note for craft/skill-write + skill-creator (prose says user-invoked but frontmatter lacks the flag).

3. skill-anatomy.md: "Cross-skill references" now splits operative (Invoke the Skill tool with `skillgrid:{name}`, one skill per call, model-invoked targets only) from passive (plain backticked name in router prose).

4. document: PR template gains a `## Merge Danger` section (one-way vs two-way door + one-word blast radius) and `## What` now leads with the smallest visual (pseudocode/call tree/file tree/Mermaid/diff sketch); SKILL.md Step 3 names the rule.

Verification: skill-size-budget.mjs check passes for all touched skills (reflect promoted to the large tier to match ship/qa weight); environment-retro is 111 lines / 8545 bytes, in-bounds. The onboarding size overage is pre-existing (budget ledger was stale at 27088 vs real 43260), not introduced here. Working tree also carries unrelated parallel churn from the 2026-10-02-skillgrid-workflow-updates spec — review the diffs in that context.
<!-- SECTION:FINAL_SUMMARY:END -->
