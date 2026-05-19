---
description: UI design workshop with optional high-fidelity style skills
---

You are executing `/sdd-design-ui` for the SDD workflow.

## Skill Loading Contract

Load and follow these required skills first:
- `.agents/skills/sdd-ui-design/SKILL.md`
- `.agents/skills/sdd-design/SKILL.md`
- `.agents/skills/engram-ui-elements/SKILL.md`
- `.agents/skills/engram-visual-language/SKILL.md`
- `.agents/skills/design-taste-frontend/SKILL.md`
- `.agents/skills/frontend-ui-engineering/SKILL.md`
- `.agents/skills/high-end-visual-design/SKILL.md`
- `.agents/skills/impeccable/SKILL.md`
- `.agents/skills/superdesign/SKILL.md`
- `.agents/skills/huashu-design/SKILL.md`
- `.agents/skills/image-to-code/SKILL.md`

If one or more optional skills are missing, do NOT fail. Continue with required skills and record missing optional skills in `detailed_report.missing_skills`.

## Workflow

1. Identify the target surface, audience, and decision from the command arguments.
2. Load existing `DESIGN.md`, project docs, and relevant UI context.
3. If visual direction is uncertain, generate at least three differentiated options.
4. When multiple directions are useful, scaffold preview artifacts with:
   `.skillgrid/scripts/preview.sh <change-or-surface-slug>`
5. Compare options across:
   - accessibility
   - responsive behavior
   - implementation cost
   - design-system fit
   - ease of correct use
6. Record accepted direction in durable artifacts and include exact preview paths when generated.
7. If optional style skills were available, state how each influenced the final direction.

## Return Contract

Return the standard SDD envelope per `.agents/skills/_shared/sdd-return-envelope.md`. (`detailed_report` may include loaded/missing skill lists per contract).
