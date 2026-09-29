# Toolbar

Every sketch carries a small floating toolbar in the bottom-right corner. It gives the
user the controls they need to judge the variants without the agent being asked to
rebuild the mockup for each tweak: switch themes, preview at different viewports, and
inspect the actual computed styles on hover.

The toolbar is **unobtrusive by design** — small, dark, semi-transparent, in the
corner, never competing with the sketch for attention.

## Location and Look

Fixed bottom-right, above any other floating content. A compact pill that expands on
hover to reveal the controls.

```css
.sketch-toolbar {
  position: fixed;
  bottom: var(--space-4);
  right: var(--space-4);
  z-index: 9999;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: var(--space-1);
  font-family: var(--font-mono);
  font-size: var(--text-xs);
}
.sketch-toolbar__handle {
  width: 2.5rem;
  height: 2.5rem;
  border-radius: var(--radius-full);
  background: rgba(20, 20, 20, 0.75);
  color: #fff;
  border: 1px solid rgba(255, 255, 255, 0.15);
  backdrop-filter: blur(8px);
  cursor: pointer;
  display: grid;
  place-items: center;
  transition: opacity var(--transition);
}
.sketch-toolbar__panel {
  display: none;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-3);
  background: rgba(20, 20, 20, 0.85);
  border-radius: var(--radius-md);
  border: 1px solid rgba(255, 255, 255, 0.15);
  backdrop-filter: blur(8px);
  color: #fff;
  width: 11rem;
}
.sketch-toolbar:hover .sketch-toolbar__panel { display: flex; }
.sketch-toolbar:hover .sketch-toolbar__handle { opacity: 0.3; }
```

The handle is a `⚙` or `☰` glyph. On hover, the panel appears above it.

## Control 1: Theme Switcher

A `<select>` that swaps the theme file at runtime (see
[theme-system.md](theme-system.md)). Each `<option>` is a theme file in
`sketches/themes/`.

```html
<label class="toolbar-field">
  Theme
  <select data-theme-select>
    <option value="default" selected>Default</option>
    <option value="dark">Dark</option>
    <option value="warm-minimal">Warm Minimal</option>
  </select>
</label>
```

```js
document.querySelector('[data-theme-select]').addEventListener('change', (e) => {
  document.querySelector('link[data-theme]').href =
    `../themes/${e.target.value}.css`;
});
```

The options are generated from whatever files exist in `themes/` — if you create a new
theme file, add it to the select. (For a throwaway sketch you can hardcode the
options; you are not building a theme manager.)

## Control 2: Viewport Preview

Buttons that constrain the sketch's content to a phone / tablet / desktop width, so
the user can judge responsiveness without resizing the browser. The toolbar applies a
`max-width` + `margin: 0 auto` to the sketch's content wrapper — it does **not**
resize the browser window.

```html
<div class="toolbar-field toolbar-field--group" data-viewport-group>
  <span>Viewport</span>
  <button data-viewport="375">375</button>
  <button data-viewport="768">768</button>
  <button data-viewport="1280" class="active">1280</button>
</div>
```

```js
const content = document.querySelector('[data-sketch-content]');
document.querySelectorAll('[data-viewport]').forEach(btn => {
  btn.addEventListener('click', () => {
    document.querySelectorAll('[data-viewport]').forEach(b =>
      b.classList.toggle('active', b === btn));
    const w = btn.dataset.viewport;
    content.style.maxWidth = w === '1280' ? '1280px' : w + 'px';
    content.style.margin = w === '1280' ? '' : '0 auto';
  });
});
```

The default (active) viewport is the widest — the sketch renders full-width until the
user narrows it. This is a *preview* aid; the real responsive behavior is tested by
the CSS media queries, and the viewport buttons just make it faster to see.

## Control 3: Annotation Mode

A toggle that, when on, shows a computed-style overlay on any element the user hovers:
its background color, font size/family/weight, padding, margin, and border radius.
This answers "what is the actual spacing here?" without the user opening devtools.

```js
const annOverlay = document.createElement('div');
annOverlay.className = 'annotation-overlay';
annOverlay.style.display = 'none';
document.body.appendChild(annOverlay);

let annotating = false;
document.querySelector('[data-annotation-toggle]').addEventListener('change', (e) => {
  annotating = e.target.checked;
  document.body.classList.toggle('annotating', annotating);
  if (!annotating) annOverlay.style.display = 'none';
});

document.addEventListener('mouseover', (e) => {
  if (!annotating) return;
  const el = e.target.closest('[data-sketch-content] *');
  if (!el) return;
  const cs = getComputedStyle(el);
  annOverlay.innerHTML = [
    `bg ${cs.backgroundColor}`,
    `color ${cs.color}`,
    `font ${cs.fontSize} ${cs.fontWeight} ${cs.fontFamily.split(',')[0]}`,
    `pad ${cs.padding}`,
    `radius ${cs.borderRadius}`,
  ].join(' · ');
  const r = el.getBoundingClientRect();
  annOverlay.style.display = 'block';
  annOverlay.style.left = r.left + 'px';
  annOverlay.style.top = Math.max(0, r.top - 28) + 'px';
});
document.addEventListener('mouseout', (e) => {
  if (annotating && !e.target.closest('[data-sketch-content] *'))
    annOverlay.style.display = 'none';
});
```

```css
.annotation-overlay {
  position: fixed;
  z-index: 10000;
  padding: 0.25rem 0.5rem;
  background: rgba(0, 0, 0, 0.9);
  color: #fff;
  font-family: var(--font-mono);
  font-size: 0.7rem;
  border-radius: var(--radius-sm);
  pointer-events: none;
  white-space: nowrap;
}
body.annotating [data-sketch-content] *:hover {
  outline: 1px dashed var(--primary);
}
```

The overlay is `pointer-events: none` so it never blocks the hover it is describing.
The dashed outline on hover tells the user which element the annotation is for.

## The Toolbar Is Not Part of the Sketch

The toolbar is a **viewing aid**, not part of the design being tested. It is excluded
from the "what's liftable" in the findings — the real build does not ship the toolbar.
If a user likes a *control* the toolbar demonstrates (e.g., "I want the annotation
overlay in the real app's dev mode"), that is a separate feature request, noted in the
findings as a constraint or a follow-up, not silently included.

## Building It

The toolbar markup, CSS, and JS are self-contained. The starter at
[templates/sketch.html](../templates/sketch.html) includes a working toolbar with all
three controls — copy it rather than re-deriving it. When you add a sketch, the only
thing you may need to change is the theme `<select>` options (to match the actual
theme files present).
