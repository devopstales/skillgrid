---
name: product-ui
description: "Designs product and application UIs — dashboards, admin panels, settings, forms, data tables, editors, onboarding flows — with an Operate-mode bias toward scanability, consistency, and native affordances. Use when building or redesigning app UI, SaaS dashboards, admin consoles, settings pages, forms, data tables, or multi-step product flows. Not for marketing landing pages, portfolios, or brand/aesthetic-only work."
license: MIT
metadata:
  author: devopstales
  version: "2.0"
  part-of: skillgrid
  based_on: original (no upstream fold) — derived from the Operate-mode surface-class bias; cross-references `skillgrid:ui-ux-pro-max`, `skillgrid:ui-styling`, `skillgrid:design-system`, `skillgrid:design-taste-frontend`
---

# Product UI

## Contents
- Overview / When to Use / The Workflow
- Design Principles (Operate bias) / Common Rationalizations / Red Flags
- Verification / References / Cross-skill

**Announce at start:** "I'm using the skillgrid:product-ui skill to design this product UI."

## Overview

Design application UIs the way users actually operate them: scanable,
consistent, and built on native affordances. The visitor's success is
*completing a task*, not being impressed — so the interface recedes and the
work leads.

**Iron law:**

```
DESIGN FOR THE TASK, NOT THE AESTHETIC
```

A product UI that looks beautiful but costs a step per action is a failure.
Every layout, density, and state decision must trace back to a user's task
path. When the brief and the task conflict, the task wins.

**Persona-scope rule (reply language ≠ artifact language).** The persona this
skill adopts — Operate-mode, task-first, scanability-over-aesthetic — scopes
the *artifacts* it produces (the UI spec, the design, the layout, the
component notes), not the language of your reply to the user. Reply in the
language the user used; write the UI artifact in the language the artifact's
consumers use (the product's locale, or the user's instruction language). Never
let the persona leak into the reply tone: do not answer a German question in
English because the skill "thinks in product terms", and do not render a
product's UI labels in the reviewer's language when the product ships in
another. The persona is a lens on the *design*, not a setting on the *chat*.

## When to Use

- Designing a new dashboard, admin panel, settings page, or app shell
- Building or redesigning forms, data tables, or list views
- Designing multi-step flows: onboarding, checkout, configuration wizards
- Laying out a high-density data view (analytics, monitoring, telemetry)
- Establishing consistent interaction states across an existing app

**When NOT to use:**

- Marketing landing pages, portfolios, or brand sites — use
  `skillgrid:design-taste-frontend` (anti-slop) or a style skill such as
  `skillgrid:gpt-taste` / `skillgrid:high-end-visual-design` /
  `skillgrid:minimalist-ui` / `skillgrid:industrial-brutalist-ui`.
- Pure brand, logo, CIP, banner, or icon work — use `skillgrid:design`.
- Token architecture and component-spec systems — use `skillgrid:design-system`.
- shadcn/ui + Tailwind implementation mechanics — use `skillgrid:ui-styling`.

## The Workflow

```
1. Classify the surface        (Operate / Operate-dense / Flow / Onboard)
        │
2. Establish the layout spine   (structure before style)
        │
3. Define interaction states    (full cycle, every control)
        │
4. Calibrate density & spacing  (match the surface class)
        │
5. Handle the hard cases        (empty / loading / error / edge)
        │
6. Verify against the task path (evidence-gated exit)
```

Each step below is a decision gate. If a step's output is wrong, fix it before
advancing — a wrong spine cannot be saved by better states.

### Step 1 — Classify the surface

Name the surface before drawing anything. This single choice sets every later
default (density, motion, emphasis).

| Surface class | Examples | Primary bias |
|---|---|---|
| **Operate** | Settings, profile, CRUD admin | Clarity, low density, calm |
| **Operate-dense** | Dashboards, analytics, monitoring, data grids | Scanability, high density, data-first |
| **Flow** | Checkout, multi-step config, onboarding | Forward momentum, low distraction |
| **Onboard** | First-run, empty-state activation | Guidance, one action at a time |

Ask: *what does the visitor accomplish here, and in how many steps?* If the
answer is "a few clicks to a result," it is a **Flow**. If it is "scan and
act on data," it is **Operate-dense**. If it is "configure and persist," it is
**Operate**. When ambiguous, default to **Operate** and raise density only when
the data justifies it.

### Step 2 — Establish the layout spine

Fix structure before style. The spine is the skeleton every screen shares.

- **Navigation model.** Choose one primary pattern and stick to it: top nav,
  left sidebar, or command palette. Do not mix two primary nav patterns on one
  surface. A dashboard that defaults to a left sidebar is fine; do not add a
  full top nav on top of it.
- **Content grid.** One content container, constrained width
  (`max-w` ~1200–1440px for Operate; full-bleed allowed for Operate-dense
  grids). A single max-width, never per-section.
- **Region roles.** Name each region's job (header / filter bar / data region /
  action bar / detail pane). A region with two jobs is two regions.
- **Hierarchy by placement, not decoration.** The primary action and the
  primary data get the most visual weight by position and contrast — not by a
  heavier shadow or a brighter color than the secondary content.

For Operate-dense surfaces, load
[references/data-tables.md](references/data-tables.md) for grid, column,
virtualization, and sort/filter conventions.

### Step 3 — Define interaction states

LLMs default to the single "successful" state. A product UI is the *full
cycle*. For every interactive control, define all five states — do not ship a
control that only works when it succeeds:

| State | Requirement |
|---|---|
| **Default** | Clear, labeled, discoverable |
| **Hover / focus** | Visible change; focus ring is mandatory (a11y) |
| **Active / pressed** | Tactile feedback (`scale(0.98)` / `translateY(1px)`), no layout shift |
| **Loading** | Skeleton matching the layout shape; never a generic spinner for a region |
| **Disabled** | Reduced emphasis, no action, clearly non-interactive |

Plus the three content states for any data region:

- **Empty** — composed, shows how to populate; never a blank void.
- **Error** — clear, inline for forms; contextual for transient failures.
- **Partial / degraded** — what the user sees when data is incomplete or a
  section is unavailable.

For forms, load [references/forms.md](references/forms.md) for validation,
label, and error-placement conventions.

### Step 4 — Calibrate density & spacing

Density is a surface-class decision, not a taste decision.

| Surface | Density | Spacing scale |
|---|---|---|
| Operate | Low–mid | 16 / 24 / 32 / 48 |
| Operate-dense | High | 8 / 12 / 16 / 24 |
| Flow | Low | 24 / 32 / 48 |
| Onboard | Low | 32 / 48 / 64 |

Rules:

- **One spacing scale per project.** Pick the scale for the dominant surface
  and use it everywhere. Mixing 8px and 24px rhythms in the same app reads as
  broken.
- **Density is consistent within a surface.** A dashboard does not get one
  ultra-dense table and one airy card grid on the same screen without a
  deliberate, stated reason.
- **Whitespace is not wasted space** in Operate/Flow surfaces. Density is for
  data, not for chrome.

### Step 5 — Handle the hard cases

The hard cases are where product UIs actually break. Design them explicitly,
not as an afterthought:

- **Empty states** for every list, table, and detail pane.
- **Long / overflow content** — truncation with `title`, ellipsis, or a "show
  more"; never a clipped word with no escape.
- **Very long lists / tables** — pagination, virtualization, or infinite scroll
  with a stable scroll anchor (see data-tables reference).
- **Concurrent / async actions** — optimistic updates with rollback on
  failure; never a full-page reload for a single-row change.
- **Keyboard path** — the primary task must be completable without a mouse:
  tab order, arrow-key nav in grids, `Enter` to confirm, `Esc` to cancel.
- **i18n headroom** — labels and buttons must not overflow when the string
  grows 30–50% (the German case). Reserve width or wrap, never truncate
  action labels.

### Step 6 — Verify against the task path

Before declaring the design done, walk the *actual* primary task end to end
and confirm each of the boxes in Verification. This is the evidence gate —
"seems usable" is not a checkbox.

## Design Principles (Operate bias)

These are the standing biases that separate a product UI from a marketing
page. They override the aesthetic defaults most models reach for.

1. **Consistency over novelty.** The same action looks and behaves the same
   everywhere in the app. A "delete" is the same color, position, and
   confirmation pattern on every screen. Novelty is for brand, not for
   operations.
2. **Scanability first.** The eye path is: status → primary data → primary
   action. A user should be able to answer "what's the state and what do I do
   next" without reading body copy.
3. **Affordance over decoration.** A control must look like what it does. If
   it's a button, it reads as a button. No decorative gradient on a submit
   button, no icon that requires a label to be understood.
4. **Progressive disclosure.** Show the common path by default; reveal the
   advanced path on demand. A settings page with 40 always-visible fields is a
   form, not a settings experience.
5. **Motion is functional.** In product UIs, motion communicates state
   (a drawer opening, a list reordering, an item being added). It is never
   ambient or ornamental. Respect `prefers-reduced-motion` always.
6. **One theme, locked.** Pick light, dark, or auto at the app level. Sections
   do not invert. Design for both modes from the start; do not ship light-only.
7. **The task path is the test.** If adding a visual element forces a user to
   look away from the task to understand it, the element is noise.

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "A left sidebar is the standard for dashboards, so I'll use one." | The standard is a starting hypothesis, not a rule. Ask what the task needs — a top nav or command palette can be better. The spine is a decision, not a default. |
| "I'll just add the loading state later." | A control with no loading state is a bug waiting to ship. The full cycle is part of the design, not a follow-up. |
| "This is a settings page, so one long form is fine." | 40 always-visible fields is a form, not a settings experience. Progressive disclosure is the product-UI answer. |
| "The design looks good, so the task path works." | Aesthetics and usability are different axes. A beautiful UI that costs an extra step per action fails the Iron law. Walk the task path. |
| "I'll handle i18n when we localize." | i18n headroom is a layout constraint you pay for now or clip strings later. Reserve width in the design. |
| "The empty state can just be blank." | A blank void tells the user nothing and invites "is it broken?" A composed empty state is part of the design. |

## Red Flags

- A control shipped with only a default + hover state (no active, loading, or
  disabled).
- A generic circular spinner standing in for a region's loading state.
- Two different spacing scales or two different nav patterns in one surface.
- A data region with no empty state, no error state, or no loading state.
- A primary action styled more prominently (bigger shadow / brighter) than the
  primary data it acts on.
- A label or button that will clip when the string is translated to German.
- A "delete" that looks or behaves differently on two screens.
- Motion that is ambient rather than state-driven, or that ignores
  `prefers-reduced-motion`.

## Verification

Every box is evidence-gated — provable by a screenshot, a rendered artifact, or
a stated task-path walkthrough. "Seems right" is not a checkbox.

- [ ] Surface class is named (Operate / Operate-dense / Flow / Onboard) and
  the density + spacing scale chosen for it are stated.
- [ ] One primary navigation pattern is in use; no competing second pattern.
- [ ] A single max-width content container is applied (Operate) or a deliberate
  full-bleed grid (Operate-dense).
- [ ] Every interactive control has all five states designed (default,
  hover/focus, active, loading, disabled) — confirmed by inspection of the
  rendered output.
- [ ] Every data region has empty, loading, and error states designed.
- [ ] The primary task is completable by keyboard alone (tab order, arrows,
  Enter, Esc) — confirmed by a keyboard walkthrough or a stated path.
- [ ] No label or action button overflows under a +40% string-length test
  (the i18n case).
- [ ] Motion is state-driven only and collapses under
  `prefers-reduced-motion` — confirmed in both reduced and normal settings.
- [ ] The design is reviewed in both light and dark mode (or the committed
  single mode), with WCAG AA contrast confirmed for body text.
- [ ] The primary task path was walked end to end and each step was confirmed
  against the surface class's goal.

## References

- [references/data-tables.md](references/data-tables.md) — Operate-dense grids:
  column strategy, sort/filter, virtualization, pagination, keyboard nav. Load
  when the surface class is Operate-dense.
- [references/forms.md](references/forms.md) — form architecture: labels,
  validation timing, error placement, multi-step flows, async submits. Load
  when the surface contains significant forms or flows.

## Cross-skill

- `skillgrid:ui-ux-pro-max` — searchable data for palettes, fonts, UX
  guidelines; use for *lookup*, not for the Operate bias this skill enforces.
- `skillgrid:ui-styling` — shadcn/ui + Tailwind implementation mechanics.
- `skillgrid:design-system` — token architecture and component specs.
- `skillgrid:design-taste-frontend` — anti-slop rules; the *counterpart* for
  marketing surfaces. Use it, not this skill, for landing pages.
