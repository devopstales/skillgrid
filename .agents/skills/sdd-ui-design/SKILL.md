---
name: sdd-ui-design
description: >
  Place UI design decisions, previews, and implementation constraints into durable SDD artifacts.
  Trigger: UI/UX direction work during SDD explore, brainstorm, design, plan, or verify phases.
license: Apache-2.0
metadata:
  author: devopstales
  version: "2.0"
  depends_on: [sdd-design, engram-ui-elements, engram-visual-language, engram-project-structure]
---

## When to Use

Use this skill when UI decisions must be captured as durable project artifacts, especially when `/sdd-design-ui`, `/sdd-brainstorm`, or `/sdd-explore` influences interface behavior.

## Responsibilities

This skill owns artifact placement and decision durability, not visual taste generation itself.

Primary artifact targets:

- root `DESIGN.md`
- `.skillgrid/preview/`
- PRD success criteria
- OpenSpec `design.md`
- handoff and research files under `.skillgrid/tasks/`

Use visual-quality skills separately when needed:

- `huashu-design`
- `superdesign`
- `impeccable`
- `frontend-design`
- `frontend-ui-engineering`

## Artifact Placement Rules

- Stable design-system rules belong in `DESIGN.md`.
- Draft comparisons and exploratory options belong in `.skillgrid/preview/`.
- User-facing acceptance expectations belong in PRD criteria.
- Implementation constraints belong in OpenSpec `design.md`.
- External references and deep comparisons belong in `.skillgrid/tasks/research/<change-id>/`.

## Design-It-Twice Rule For Interface Decisions

When the UI choice also defines an interface shape (component props, routes, module boundaries, API-to-UI contracts, shared primitives), generate at least three distinct options before convergence.

Before generating options, collect:

- problem to solve
- callers and users (human users, components, modules, tests, external consumers)
- required operations and states
- constraints (accessibility, performance, compatibility, existing patterns)
- what must stay hidden internally vs exposed

For each option, include:

- interface or surface shape
- usage example
- hidden complexity
- tradeoffs and likely failure modes

Compare options using:

- simplicity of surface
- implementation cost and performance
- ease of correct use vs misuse
- responsive behavior
- accessibility
- fit with existing design language

Only promote the accepted direction to implementation guidance. Keep rejected options in preview or research artifacts.

## Preview Discipline

For browser-viewable UI options, scaffold preview files with:

```bash
.skillgrid/scripts/preview.sh <change-id>-options
```

For text-only comparisons, use:

```bash
.skillgrid/scripts/preview.sh --md <change-id>-options
```

Recommended naming:

- `.skillgrid/preview/<change-id>-options.html`
- `.skillgrid/preview/<change-id>-option-notes.md`

Link preview artifacts from PRD blocks or handoff files whenever possible so they are discoverable.

## OpenSpec Design Insert Template

When a UI decision affects implementation, insert this section into the change `design.md`:

```markdown
## UI / UX constraints

- **Design source:** `.skillgrid/preview/<change-id>-options.html`
- **Tokens:** follow root `DESIGN.md`
- **States required:** default, loading, empty, error, success
- **Accessibility:** keyboard reachable, visible focus, reduced motion, contrast compliant
- **Responsive behavior:** <breakpoints or layout rules>
- **Interface shape:** <accepted surface contract and hidden internals>
```

## Delivery Checklist

1. Ask at most one clarifying question if the target surface is ambiguous.
2. Produce differentiated options when uncertainty exists.
3. Document accepted direction and key rejected alternatives.
4. Capture accessibility, responsive, and testing implications.
5. Update durable artifacts and include exact preview file paths.

## Return

- Load `skills/_shared/sdd-phase-common.md` for executor protocol.
- **Return:** the standard SDD envelope per `skills/_shared/sdd-return-envelope.md`.
- Put selected direction and rationale in `executive_summary.overview`; options/tradeoffs in `detailed_report` (may use `loaded_required_skills`, `loaded_optional_skills`, `missing_skills` per contract phase extensions).