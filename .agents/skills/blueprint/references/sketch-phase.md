# Sketch (UI/interaction variants — inline capability)

Answer "does this layout/interaction feel right?" by **building throwaway
interactive mockups the user can switch between and *feel*** — never by
describing the trade-off in prose. The output is a **marked winner + rationale
+ constraints**, written as a `## Sketch: NNN-name` section in the change's
`findings.md` (per `_shared/rules/sdd-structure.md` — the `sketch` role:
"UI/interaction variants → `findings.md` (Sketch section)").

A sketch is an **inline capability of this skill**, not a numbered phase — it
runs wherever the draft (Phase 2) meets a layout or interaction decision with
2+ meaningfully different options, just as Phase 1 runs research and Phase 4
runs a prototype. The sketch's value is that the user *experiences* the
trade-off (does the sidebar feel cramped? does the top nav disappear on
mobile?) rather than imagining it from prose. The mockups are **throwaway by
design** — built to be felt and compared, not to ship. The only things that
survive are the **winner's key structure** (what the real build must
preserve) and the **constraints** (what the real build must not forget) — both
captured in the findings section.

**One-line boundary with `skillgrid:ui-design`:** sketch is the *variant*
mechanic — build 2+ options, get a verdict, pick; `skillgrid:ui-design`
finishes the *chosen* design (surface class, five interaction states, density
calibration, hard cases) into the deliverable.

## When to Use

- A new UI page/layout needs to be explored before building it for real, and
  there are **2+ meaningfully different** layout or interaction options.
- The visual direction is unclear and a cheap sketch beats guessing — the
  answer depends on *feeling* the interaction, not reading a description.

**When the sketch is the wrong tool:**

- A single obvious layout does not need a variant comparison — that is just a
  design decision (use `skillgrid:ui-design` to finish it).
- A question that is purely about *technical* feasibility ("can we stream
  this?") is Phase 4 (prototype), not a sketch.
- A one-line UI tweak or a fully-specified component where the layout is
  already obvious — a sketch is for exploring, not confirming.

## Ground in Real Data Shapes (if research or prototype ran)

If the topic's `findings.md` has `## Research:` or `## Prototype:` sections,
**read them first** and ground the mockups in what they proved: real field
names, real data shapes, real interaction states (streaming / loading / error
/ empty). A sketch that shows placeholder "lorem ipsum" data when a research
pass or prototype already proved the actual data shape is throwing away
information. Use the actual outputs where they exist; fake only what they did
not cover.

## Mood Intake (before any code)

Before building, establish the **mood** in a short conversation. Ask **one
question at a time** (not a form), and stop once you have enough to
differentiate the variants:

1. **Feel** — what should this feel like? (clinical / warm / playful / dense /
   airy — the user's words, not a list you impose)
2. **References** — any existing product or page that captures part of the
   feel? (one or two is enough; "nothing specific" is a valid answer)
3. **Core action** — what is the single thing a user does on this screen most
   often? (the variants must all make *that* action work; a variant that
   breaks the core action is not a real variant)

Three questions, one at a time, is usually enough. If the user says "just
build something," pick a reasonable mood from the project's existing
aesthetic and note the assumption in the findings.

## Prefer Adjusting an Existing Page

**Before creating a throwaway standalone file, check if the page already
exists in the app.** If it does, the sketch is a *diff on that page* — a
`?variant=` query param or a dev-only toggle that swaps the layout — not a new
file. The user sees the variant in *context* (real surrounding chrome, real
routing, real data), and the winner is closer to the real implementation
(less "lift" required).

Only when there is **no existing page** (greenfield, or a genuinely new
screen) do you create a standalone HTML sketch file. State which mode you used
in the findings (`diff-on-existing-page` vs `standalone`) so the build knows
how much lift is needed.

## The Process

### 1. Set up the sketch directory

Create `{specs_root}/YYYY-MM-DD-<topic>/sketches/NNN-descriptive-name/` (the
`sketches/` subdirectory is a canonical location under the change folder, per
`_shared/rules/sdd-structure.md` § Directory Structure). `NNN` is the next
free number in that topic's `sketches/` directory (three digits, zero-padded).

On the **first** sketch for a topic, also create `sketches/themes/default.css`
—a CSS-variable theme (colors, typography, spacing, shapes, shadows, motion as
custom properties). Every later sketch in the topic links to the same theme,
so design decisions compound: change the palette once, every sketch updates.

```
{specs_root}/YYYY-MM-DD-<topic>/sketches/
├── themes/default.css   # shared CSS-variable theme (created on first sketch)
└── NNN-name/index.html  # the mockup (2-3 variants, tab-switchable)
```

### 2. Build 2–3 dramatically different variants

Build **2–3 variants in the same HTML file**, switchable via a tab bar. The
user sees one variant at a time and switches by clicking tabs.

```html
<nav class="variant-tabs">
  <button class="variant-tab active" data-variant="a">A — Sidebar</button>
  <button class="variant-tab" data-variant="b">B — Top Nav</button>
  <button class="variant-tab" data-variant="c">C — Floating</button>
</nav>
<main>
  <section class="variant" data-variant="a"></section>
  <section class="variant" data-variant="b" hidden></section>
  <section class="variant" data-variant="c" hidden></section>
</main>
<script>
  document.querySelectorAll('.variant-tab').forEach(tab =>
    tab.addEventListener('click', () => {
      document.querySelectorAll('.variant-tab').forEach(t => t.classList.toggle('active', t === tab));
      document.querySelectorAll('.variant').forEach(v => v.hidden = v.dataset.variant !== tab.dataset.variant);
    }));
</script>
```

Each `.variant` section is a complete, self-contained layout. Shared chrome
(the tab bar, the toolbar) lives *outside* the variants so it does not reset
on switch.

**The core rule — meaningful difference:** first-round variants must differ in
**structure**, not skin. A structure difference is one of:

- **Layout skeleton** — sidebar vs top-nav vs floating panels vs single-column
- **Information hierarchy** — what is primary vs secondary vs hidden
- **Primary interaction** — how the core action is reached (one-tap nav vs a
  form vs a command palette)

Three color palettes on one layout skeleton is **not** a variant comparison —
it is a theme comparison (that is what the theme switcher is for). If your
three variants share the same skeleton, restructure them.

**Variant count:**

- **2–3** is the normal range; **never more than 4** — more overwhelms the
  user; they are choosing, not cataloging.
- If you have 5+ genuinely different options, **narrow to 3 before showing**:
  pick the 3 most promising, note the ones you cut and why in the findings.

**Refinement rounds:** after a first-round verdict, the user may want
refinements *within* the chosen direction rather than a different direction.
Build 2–3 variants that share the chosen skeleton but vary one axis (density,
spacing, a single component's treatment). Tab labels reflect the refinement
axis: `"Dense"`, `"Medium"`, `"Airy"`.

**Synthesis (cherry-pick) is a first-class outcome.** The user often does not
pick a single variant — they *combine* elements: "A's layout, C's palette,
B's nav." Do not force a single-choice pick. Build a **labeled synthesis
variant** (e.g. `S — A-layout + C-palette + B-nav`), add it to the tab bar
alongside the originals, re-present; the user confirms the synthesis or
adjusts it. The synthesis variant — not one of the originals — is what gets
marked as the winner.

### 3. Make it feel alive

A static mockup is a screenshot. Every interactive element must respond:

| Element | Must do |
|---------|---------|
| Button | `:hover` + `:active` states, a click handler with visible feedback |
| Form field | focus state, validation on submit, an error message, a success state |
| List | **empty** state AND **populated** state (both reachable) |
| Toggle/switch | actually toggles, and updates whatever it controls |
| Link | `:hover` state; if it "navigates," show the destination (even a stub) |
| Modal/overlay | opens and closes, with a visible close affordance |
| Dropdown | opens, selects, reflects the selection |

**Fake the backend.** No real backend exists: the "Save" button shows a
loading state ("Saving…", disabled) then success (a toast or checkmark); a
field with a likely failure has a way to trigger its error state. If the
design has multiple states (empty / loading / populated / error), include
buttons to cycle through them. Vanilla JS in inline `<script>` tags — no
frameworks, no build step.

### 4. Add the toolbar

Every sketch carries a small floating toolbar (bottom-right, compact, dark,
semi-transparent — never competing with the sketch):

- **Theme switcher** — swap the CSS file at runtime
- **Viewport preview** — **375px / 768px / 1280px** (phone / tablet /
  desktop) so the user judges the responsive behavior without a device
- **Annotation toggle** — overlay spacing / color / font-size on hover so the
  user can judge the actual computed values

### 5. Present and get the verdict

Open the sketch for the user and ask: **which one feels right, and why?** Then:

- **A clear winner** → mark the winning tab (★, plus an `aria-label`), keep
  all variants navigable
- **Cherry-pick** ("A's layout, C's palette") → build the labeled synthesis
  variant, re-present
- **Refinement round** ("all too dense, make them airier") → build 2–3
  variants *within* the chosen direction, re-present
- **None work** → back to mood intake; the variants were not different enough,
  or the mood was wrong

**Winner marking:** the winner is highlighted, never the only option — the
losers are the record of what was considered, and the user may reverse course.

```js
/* CSS: .variant-tab.winner::after { content: " ★"; } */
function markWinner(variant) {
  document.querySelectorAll('.variant-tab').forEach(t => {
    t.classList.toggle('winner', t.dataset.variant === variant);
    t.setAttribute('aria-label', t.dataset.variant === variant ? t.textContent + ' (selected)' : t.textContent);
  });
}
```

**Side-by-side (small elements only):** for comparing a *small element* (a
button style, a card, a form field) rather than a full layout, a side-by-side
grid is clearer than tabs — the elements are small enough to see together.
Use it only for elements under ~400px wide; full layouts use tabs (side-by-side
full layouts require constant horizontal scrolling and defeat the comparison).

### 6. Write the findings

Append a `## Sketch: NNN-name` section to the topic's
`{specs_root}/YYYY-MM-DD-<topic>/findings.md` (create it with a
`# Findings — <topic>` header if it does not exist). The section is compact:

```markdown
## Sketch: 001-dashboard-layout

- **Winner:** Variant B (Top Nav) — synthesis with C's palette
- **Rationale:** "The sidebar felt cramped on tablet; top nav gave the content
  room. C's warmer palette matched the feel we wanted."
- **What's liftable:** the top-nav component structure at
  `sketches/001-dashboard-layout/index.html` (variant B's markup), and the
  `themes/default.css` palette
- **Constraints for the build:** the nav must collapse to a hamburger below
  768px; the content area must not exceed 1280px; the primary action (create
  ticket) must be one tap from the top nav
- **Mode:** standalone (no existing page to diff against)
```

**The rationale is the design decision** — the user's words for *why*. The
build needs the reasoning, not just the choice. Never write the winner line
without the rationale.

### 7. Report

Report: the winner, the one-line rationale, what's liftable, the constraints,
and the path to the sketch. The blueprint reads `findings.md`; it does not
re-open the mockup. Every blueprint design decision that rests on the chosen
layout cites the `## Sketch:` section (the blueprint header's **Findings**
line carries the path).

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
- Let the sketch finish the design: the picked variant goes to
  `skillgrid:ui-design` for surface class, interaction states, and the hard cases

## Verification

- [ ] A sketch artifact was produced — `sketches/NNN-name/index.html` exists
  with 2–3 switchable variants (file saved).
- [ ] Mood intake was done before any code — the findings record the feel,
  references, and core action established in conversation.
- [ ] The sketch is grounded in real data shapes where `findings.md` already
  has `## Research:` / `## Prototype:` sections — no "lorem ipsum" where a
  prior pass proved the actual shape.
- [ ] The adjust-an-existing-page vs standalone decision was made and recorded
  in the findings (`diff-on-existing-page` vs `standalone`).
- [ ] Findings written — a `## Sketch: NNN-name` section in `findings.md` with
  winner, rationale, liftable parts, build constraints, and mode.
