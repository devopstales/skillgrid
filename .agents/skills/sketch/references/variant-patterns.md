# Variant Patterns

A sketch is a set of variants the user switches between. The patterns below govern how
variants are structured, marked, and compared.

## Tab-Based Switching (the default)

All variants live in **one HTML file**, toggled by a tab bar. The user sees one
variant at a time, switches by clicking tabs, and the winner is marked on the tab.

```html
<nav class="variant-tabs">
  <button class="variant-tab active" data-variant="a">A — Sidebar</button>
  <button class="variant-tab" data-variant="b">B — Top Nav</button>
  <button class="variant-tab" data-variant="c">C — Floating</button>
</nav>

<main>
  <section class="variant" data-variant="a"> <!-- Variant A markup --> </section>
  <section class="variant" data-variant="b" hidden> <!-- Variant B markup --> </section>
  <section class="variant" data-variant="c" hidden> <!-- Variant C markup --> </section>
</main>

<script>
  document.querySelectorAll('.variant-tab').forEach(tab => {
    tab.addEventListener('click', () => {
      const target = tab.dataset.variant;
      document.querySelectorAll('.variant-tab').forEach(t =>
        t.classList.toggle('active', t === tab));
      document.querySelectorAll('.variant').forEach(v =>
        v.hidden = v.dataset.variant !== target);
    });
  });
</script>
```

Each `.variant` section is a complete, self-contained layout. Shared chrome (the tab
bar, the toolbar) lives *outside* the variants so it does not reset on switch.

## Meaningful Difference (the core rule)

First-round variants must differ in **structure**, not skin. A structure difference is
one of:

- **Layout skeleton** — sidebar vs top-nav vs floating panels vs single-column
- **Information hierarchy** — what is primary vs secondary vs hidden
- **Primary interaction** — how the core action is reached (one-tap nav vs a form vs
  a command palette)

Three color palettes on one layout skeleton is **not** a variant comparison — it is a
theme comparison (use the theme switcher for that). If your three variants share the
same skeleton, they are not meaningfully different; restructure them.

## Variant Count

- **2-3** is the normal range
- **Never more than 4** — more overwhelms the user; they are choosing, not cataloging
- If you have 5+ genuinely different options, **narrow to 3 before showing**: pick the
  3 most promising, note the 2 you cut and why in the findings. The user's job is to
  choose among the presented set, not to triage a long list.

## Refinement Rounds

After a first-round verdict, the user may want **refinements within the chosen
direction** rather than a different direction. Refinement rounds build 2-3 variants
that share the chosen skeleton but vary one axis (density, spacing, a single
component's treatment). The tab labels reflect the refinement axis:
`"Dense"`, `"Medium"`, `"Airy"`.

## Synthesis (cherry-pick) is a First-Class Outcome

The user often does not pick a single variant — they **combine** elements: "A's
layout, C's palette, B's nav." This is a valid and common verdict. Do not force a
single-choice pick. Instead:

1. Build a **labeled synthesis variant** (e.g., `S — A-layout + C-palette + B-nav`)
2. Add it to the tab bar alongside the originals
3. Re-present; the user confirms the synthesis or adjusts it

The synthesis variant is what gets marked as the winner, not one of the originals.

## Winner Marking

When the user picks (or confirms a synthesis), mark the winning tab with a ★ and an
`aria-label`. **Keep all variants navigable** — the losers are the record of what was
considered and may be reversed.

```js
function markWinner(variant) {
  document.querySelectorAll('.variant-tab').forEach(t => {
    t.classList.toggle('winner', t.dataset.variant === variant);
    t.setAttribute('aria-label',
      t.dataset.variant === variant
        ? t.textContent + ' (selected)'
        : t.textContent);
  });
}
```

```css
.variant-tab.winner::after { content: " ★"; }
```

## Side-by-Side (small elements only)

For comparing a **small element** (a button style, a card, a form field) rather than a
full layout, a side-by-side grid is clearer than tabs — the elements are small enough
to see together:

```html
<div class="side-by-side">
  <figure><figcaption>Button A</figcaption><!-- button --></figure>
  <figure><figcaption>Button B</figcaption><!-- button --></figure>
</div>
```

Use side-by-side only for elements that fit comfortably in a grid (under ~400px wide).
Full layouts use tabs — side-by-side full layouts require constant horizontal
scrolling and defeat the comparison.

## What the Findings Must Capture

The findings section for a sketch records:

- **Winner** (variant letter + name, or the synthesis label)
- **Rationale** — the user's words for *why* (this is the design decision; the build
  needs the reasoning, not just the choice)
- **What's liftable** — the winner's markup path and the theme file(s) it uses
- **Constraints for the build** — responsive breakpoints, max-width, core-action
  reach, anything the real build must preserve
- **Mode** — `standalone` (new HTML file) or `diff-on-existing-page` (the sketch is a
  `?variant=` toggle on a real page)
