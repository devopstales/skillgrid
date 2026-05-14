---
name: sdd-adr
id: sdd-adr
category: Workflow
description: Author or review architectural decision records (ADRs) using hub templates
usage: "/sdd-adr <decision-topic-or-path>"
---

You are executing `/sdd-adr` for the SDD workflow.

## Skill Loading Contract

Load and follow first:
- `.agents/skills/architectural-decision-records/SKILL.md`
- `.agents/skills/architectural-decision-records/preferences.md` (create defaults if missing per skill)

## Workflow

1. From `$ARGUMENTS`, infer the single architectural decision, target ADR path, or review request.
2. Follow the skill: one decision per ADR, honor `preferred-style` in `preferences.md`, pick templates from `templates/`.
3. Draft or revise using only known facts; mark unknowns explicitly.
4. If superseding, link or reference prior ADRs without rewriting history.

## Return Contract

Return a structured result with:
- `status`: `completed | blocked | failed`
- `executive_summary`
- `detailed_report` (template used, files touched or proposed)
- `artifacts` (ADR paths)
- `next_recommended`
- `risks`
