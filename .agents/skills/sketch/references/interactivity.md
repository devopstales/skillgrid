# Interactivity

A static mockup is a screenshot. The point of a sketch is that the user *feels* the
interaction — does the nav feel responsive, does the form give good feedback, does the
list handle its states. Every interactive element must respond. This file is the
minimum bar for "feels alive."

## Required Interactions

Every element type that appears in a sketch must behave:

| Element | Must do |
|---------|---------|
| Button | `:hover` state, `:active`/press state, a click handler with visible feedback |
| Form field | focus state, validation on submit, an error message, a success state |
| List | **empty** state AND **populated** state (both reachable) |
| Toggle/switch | actually toggles, and updates whatever it controls |
| Tab | switches the target content (already covered in variant-patterns) |
| Link | `:hover` state; if it "navigates," show the destination (even a stub) |
| Modal/overlay | opens and closes, with a visible close affordance |
| Dropdown | opens, selects, reflects the selection |

A button that does nothing on click is a broken sketch, not a subtle detail. If the
element has no obvious behavior, give it a behavior (a button that shows a toast, a
field that echoes its value on submit) — *something* that responds.

## Fake the Backend

There is no real backend in a sketch. Fake it convincingly:

```js
function fakeSubmit(form, { delay = 600 } = {}) {
  const btn = form.querySelector('button[type=submit]');
  const original = btn.textContent;
  btn.disabled = true;
  btn.textContent = 'Saving…';
  btn.classList.add('is-loading');
  setTimeout(() => {
    btn.disabled = false;
    btn.textContent = original;
    btn.classList.remove('is-loading');
    showToast('Saved ✓');
  }, delay);
}
```

- **Loading state** — a spinner or "Saving…" label, the button disabled
- **Success state** — a toast, a checkmark, a confirmation message
- **Error state** — for a field with a likely failure, a way to trigger it (e.g.,
  leave it empty to see the validation error)

The user is judging the *experience of the interaction*, and that experience includes
the waiting and the confirmation. A form that instantly "saves" with no feedback is
telling the user nothing about how the real thing will feel.

## State Cycling

When a design has multiple states (empty / loading / populated / error / partial), the
sketch must let the user **reach every state**. Add a small "state" control (a select
or a row of buttons) that cycles the relevant container through its states:

```js
const states = ['empty', 'loading', 'populated', 'error'];
const stateSelect = document.querySelector('[data-state-select]');
stateSelect.addEventListener('change', () => {
  renderList(container, stateSelect.value);
});
```

Without state cycling, the user only sees whatever state the mockup happens to be in
— usually "populated" — and never learns how the empty or error states look. For a
data-heavy screen, the empty and error states are often the ones with the most
design decisions (what does the user do when there is nothing? what does the error
tell them?).

## Transitions

A baseline transition makes the sketch feel polished rather than janky:

```css
* {
  transition: background-color var(--transition),
              color var(--transition),
              box-shadow var(--transition),
              transform var(--transition);
}
```

`--transition` is `0.15s ease` (from the theme). This is a floor, not a ceiling — a
specific interaction (a modal opening, a drawer sliding) may need its own duration and
easing. But *some* transition on state changes is the default; an instant,
unanimated jump reads as unfinished.

## The Vanilla-JS Rule

Inline `<script>` tags, vanilla JS. No frameworks, no build step, no module imports
that require a bundler. The sketch must open by double-clicking the HTML file (or
`serving it statically`) and work. A sketch that requires `npm run dev` to see has
already become a project, not a sketch.

If a variant genuinely needs a library to test (e.g., "does chart.js render this
data shape at all?"), load it from a CDN in a `<script>` tag — that is still
static-openable. But the sketch's *own* logic (tabs, state cycling, fake backend) is
vanilla.

## What "Alive" Is Not

- **Alive ≠ animated.** A subtle transition is the bar; a bouncing logo is not.
- **Alive ≠ realistic data.** The data should be *representative* (right shape, right
  volume, right edge cases) but does not need to be real. If a spike produced real
  data shapes, use them; otherwise, plausible fake data.
- **Alive ≠ complete.** The sketch tests the *interaction and layout*, not every
  feature. A list does not need pagination, search, and filtering all working — but
  it does need to look right when it has 0, 1, 10, and 1000 items (state cycling).
