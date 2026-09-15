---
name: sketch
description: Build throwaway interactive UI mockups to answer a "does this layout/interaction feel right?" question. Produces 2-3 dramatically different variants the user can switch between, a marked winner, and the constraints for the real build. Use when the design has 2+ meaningfully different layout or interaction options and the answer depends on *feeling* it, not reading a description.
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  based_on: gsd-core:gsd-sketch + mattpocock-skills:prototype (UI branch — prefer adjusting an existing page)
---

# Sketch

Answer a design question by **building interactive mockups the user can switch
between and feel.** The output is a **marked winner + rationale + constraints** — not
a description of what you would build. A sketch's value is that the user *experiences*
the trade-off (does the sidebar feel cramped? does the top nav disappear on mobile?)
rather than imagining it from prose.

The mockups are **throwaway by design.** They are built to be felt and compared, not
to ship. The one things that survive are: the **winner's key CSS/structure** (what the
real build must preserve) and the **constraints** (what the real build must not forget).

**Announce at start:** "I'm using the skillgrid:sketch skill to mock this up so you
can feel the difference."

**When a sketch is the wrong tool:** a single obvious layout does not need a variant
comparison — that's just a design decision. A sketch is for when there are **2+
meaningfully different options** and the choice depends on *feeling* the interaction.
A question that is purely about a *technical* feasibility ("can we stream this?") is
`skillgrid:spike`, not a sketch.

## Overview

This skill answers "does this layout/interaction feel right?" by **building
throwaway interactive mockups** the user can switch between and *feel* — never by
describing the trade-off in prose. It produces **2-3 dramatically different
variants**, a **marked winner + rationale**, and the **constraints** the real build
must honor. The mockups are throwaway by design; only the winner's key structure and
the constraints survive into the build.

## When to Use

- A new UI page/layout needs to be explored before building it for real, and there
  are **2+ meaningfully different** layout or interaction options.
- The visual direction is unclear and a cheap sketch beats guessing — the answer
  depends on *feeling* the interaction, not reading a description.

**When NOT to use:** for a one-line UI tweak or a fully-specified component where
the layout is already obvious — a sketch is for exploring, not confirming.

## Config

Read `.skillgrid/config.yaml` before starting. Use `conventions.specs_root` (default
`.skillgrid/specs`) as the base. Sketch artifacts live under `sketch.dir` (default
`{specs_root}/{topic}/sketches/`); shared themes live under `sketch.dir/themes/`; the
consolidated findings file comes from `sketch.findings` (default
`{specs_root}/{topic}/findings.md`). If the `mnemonic` block is enabled, the findings
file you write is cited by `skillgrid:writing-blueprints` downstream.

## Ground in Real Data Shapes (if spikes ran)

If the topic has a `.skillgrid/specs/YYYY-MM-DD-<topic>/findings.md` with spike
sections, **read it first** and ground the mockups in what the spikes proved: real
field names, real data shapes, real interaction states (streaming / loading / error /
empty). A sketch that shows placeholder "lorem ipsum" data when a spike already proved
the actual data shape is throwing away information. Use the spike's actual outputs
where they exist; fake only what the spikes did not cover.

## Mood Intake (before any code)

Before building, establish the **mood** in a short conversation. Ask **one question
at a time** (not a form), and stop once you have enough to differentiate the variants:

1. **Feel** — what should this feel like? (clinical / warm / playful / dense /
   airy — the user's words, not a list you impose)
2. **References** — any existing product or page that captures part of the feel?
   (one or two is enough; "nothing specific" is a valid answer)
3. **Core action** — what is the single thing a user does on this screen most often?
   (the variants must all make *that* action work; a variant that breaks the core
   action is not a real variant)

Three questions, one at a time, is usually enough. If the user says "just build
something," pick a reasonable mood from the project's existing aesthetic and note the
assumption in the findings.

## Prefer Adjusting an Existing Page (the mattpocock rule)

**Before creating a throwaway standalone file, check if the page already exists in the
app.** If it does, the sketch is a *diff on that page* — a `?variant=` query param or a
dev-only toggle that swaps the layout — not a new file. This has two advantages: the
user sees the variant in *context* (with real surrounding chrome, real routing, real
data), and the winner is closer to the real implementation (less "lift" required).

Only when there is **no existing page** (greenfield, or a genuinely new screen) do you
create a standalone HTML sketch file. State which mode you used in the findings
(`diff-on-existing-page` vs `standalone`) so the build knows how much lift is needed.

## The Process

### Step 1: Set up the sketch directory

Create `{specs_root}/YYYY-MM-DD-<topic>/sketches/NNN-descriptive-name/`. The `NNN` is
the next free number in that topic's sketches directory (three digits, zero-padded).

On the **first** sketch for a topic, also create `sketches/themes/` with a
`default.css` (see [references/theme-system.md](references/theme-system.md)) — all
subsequent sketches in the topic link to the same theme, so design decisions compound.

```
sketches/
├── themes/
│   └── default.css     # shared CSS-variable theme (created on first sketch)
└── NNN-name/
    └── index.html      # the mockup (2-3 variants, tab-switchable)
```

Use the starter at [templates/sketch.html](templates/sketch.html) as the scaffold — it
has the theme link, the variant tab bar, the toolbar, and the variant-switching
script.

### Step 2: Build 2-3 dramatically different variants

Build **2-3 variants in the same HTML file**, switchable via a tab bar. The variants
must be **meaningfully different** in their first round — not the same layout with
different colors. Different means: different layout structure (sidebar vs top-nav vs
floating panels), different information hierarchy, different primary interaction
pattern.

Rules from [references/variant-patterns.md](references/variant-patterns.md):

- **First round:** 2-3 dramatically different approaches
- **Refinement rounds:** 2-3 subtle variations *within* the chosen direction
- **Never more than 4** — more than that overwhelms; narrow before showing
- **Synthesis is a first-class outcome** — if the user cherry-picks ("A's layout,
  C's palette"), build a labeled synthesis variant, don't force a choice

### Step 3: Make it feel alive

Static mockups are barely better than screenshots. Every interactive element must
respond. Follow [references/interactivity.md](references/interactivity.md):

- Buttons have click handlers with visible feedback
- Forms validate and show a success state
- Lists have empty AND populated states
- Toggles actually toggle
- The "Save" button shows a loading state then success (fake the backend)
- If the design has multiple states (empty/loading/populated/error), include buttons
  to cycle through them

Use vanilla JS in inline `<script>` tags. No frameworks, no build step.

### Step 4: Add the toolbar

Every sketch gets the floating toolbar from
[references/toolbar.md](references/toolbar.md): a theme switcher (swap the CSS file at
runtime), viewport preview buttons (375px / 768px / 1280px), and an annotation toggle
(overlay spacing/color/font-size on hover). The toolbar is unobtrusive — small, dark,
semi-transparent, never competing with the sketch.

### Step 5: Present and get the verdict

Open the sketch for the user. Ask: **which one feels right, and why?** Then:

- **A clear winner** → mark it (★ Selected on the tab), keep all variants navigable
- **Cherry-pick** ("A's layout, C's palette") → build a synthesis variant, re-present
- **Refinement round** ("all too dense, make them airier") → build 2-3 variants
  *within* the chosen direction, re-present
- **None work** → go back to mood intake; the variants were not different enough, or
  the mood was wrong

### Step 6: Write the findings

Append a `## Sketch: NNN-name` section to the topic's
`.skillgrid/specs/YYYY-MM-DD-<topic>/findings.md` (create it with a
`# Findings — <topic>` header if it does not exist). The section is compact:

```markdown
## Sketch: 001-dashboard-layout

- **Winner:** Variant B (Top Nav) — synthesis with C's palette
- **Rationale:** "The sidebar felt cramped on tablet; top nav gave the content
  room. C's warmer palette matched the feel we wanted."
- **What's liftable:** the top-nav component structure at
  `sketches/001-dashboard-layout/index.html` (variant B's markup), and the
  `themes/warm-minimal.css` palette
- **Constraints for the build:** the nav must collapse to a hamburger below 768px;
  the content area must not exceed 1280px; the primary action (create ticket) must
  be one tap from the top nav
- **Mode:** standalone (no existing page to diff against)
```

Commit the sketch directory + the findings file.

### Step 7: Report

Report: the winner, the one-line rationale, what's liftable, the constraints, and the
path to the sketch. The blueprint (`skillgrid:writing-blueprints`) reads the findings
file; it does not re-open the mockup.

## References

- [theme-system.md](references/theme-system.md) — the shared CSS-variable theme, how
  to create and switch themes
- [variant-patterns.md](references/variant-patterns.md) — tab-based variants, winner
  marking, side-by-side for small elements, synthesis variants, variant count rules
- [interactivity.md](references/interactivity.md) — required interactions, fake
  backend, state cycling, transitions
- [toolbar.md](references/toolbar.md) — the floating toolbar: theme switcher,
  viewport preview, annotation mode

## Common Rationalizations

| Excuse | Reality |
|--------|---------|
| "I'll just describe the layout in the spec" | A sketch's value is that the user *feels* the trade-off. A description is what you get when you have not felt it. Build the mockup. |
| "The layout is obvious, no need for variants" | If it's obvious, it's not a design decision — it's an implementation detail. Don't sketch it. If it's not obvious, there are 2+ options, and you need to feel them. |
| "I'll make 5 variants to be thorough" | More than 4 overwhelms. If you have 5+ options, narrow to the 3 most promising *before* showing. The user is choosing, not cataloging. |
| "I'll make all three variants the same layout, different colors" | Variants must be meaningfully *different* — different structure, not different skin. Three color palettes on one layout is not a variant comparison. |
| "I'll build a production page instead of a sketch" | A sketch is throwaway and fast. A production page is the build. Build the mockup, get the verdict, *then* build the page to the winner. |
| "The existing page is close enough, I'll skip the diff" | If the page exists, diff it. A standalone mockup throws away the context (real chrome, real routing, real data) that makes the verdict trustworthy. |
| "The sketch is done, I'll delete the losers" | Keep all variants navigable. The winner is highlighted, not the only option — the user may reverse course, and the losers are the record of what was considered. |

## Red Flags

**Never:**

- Ship more than 4 variants in a single sketch
- Make first-round variants that differ only in color (they must differ in structure)
- Build a standalone mockup when the page already exists (diff it instead)
- Delete losing variants after a winner is marked (keep them navigable)
- Write the findings without the rationale (the *why* is as important as the *what*)
- Skip the interactivity (a static mockup is a screenshot, not a sketch)

## Verification

- [ ] A sketch artifact was produced — `sketches/NNN-name/index.html` exists with
  2-3 switchable variants (file saved).
- [ ] Mood intake was done before any code — the Findings/notes record the feel,
  references, and core action established in conversation.
- [ ] The sketch is grounded in real data shapes (or spike verdicts) where a
  `findings.md` with spike sections exists — no "lorem ipsum" where a spike proved
  the actual shape.
- [ ] The adjust-an-existing-page vs standalone decision was made and recorded in
  the findings (`diff-on-existing-page` vs `standalone`).
- [ ] Findings written — a `## Sketch: NNN-name` section in `findings.md` with
  winner, rationale, liftable parts, build constraints, and mode.
