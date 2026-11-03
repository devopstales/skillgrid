# Theme System

Sketches share a single CSS-variable theme so that design decisions compound across
the topic: change the palette once, and every sketch in the topic updates. A sketch
that hardcodes colors cannot be re-themed without touching every file — the theme
system exists to prevent that.

## Location

`{specs_root}/YYYY-MM-DD-<topic>/sketches/themes/default.css`

Created on the **first** sketch for a topic. Every sketch in the topic links to it:

```html
<link rel="stylesheet" href="../themes/default.css">
```

(Adjust the relative path if the sketch is nested deeper than `sketches/NNN/`.)

## Shape

A theme is a set of CSS custom properties, not a set of styles. It declares the
**visual vocabulary** — colors, typography, spacing, shapes, shadows, motion. The
sketch's own CSS uses these variables; it does not hardcode values.

```css
/* themes/default.css */
:root {
  /* Color */
  --bg: #fafafa;
  --bg-elevated: #ffffff;
  --surface: #f0f0f0;
  --text: #1a1a1a;
  --text-muted: #6b6b6b;
  --primary: #2563eb;
  --primary-hover: #1d4ed8;
  --danger: #dc2626;
  --success: #16a34a;
  --border: #e0e0e0;

  /* Typography */
  --font-sans: system-ui, -apple-system, "Segoe UI", Roboto, sans-serif;
  --font-mono: ui-monospace, "SF Mono", "Cascadia Code", monospace;
  --text-xs: 0.75rem;
  --text-sm: 0.875rem;
  --text-md: 1rem;
  --text-lg: 1.25rem;
  --text-xl: 1.5rem;

  /* Spacing */
  --space-1: 0.25rem;
  --space-2: 0.5rem;
  --space-3: 0.75rem;
  --space-4: 1rem;
  --space-6: 1.5rem;
  --space-8: 2rem;

  /* Shape */
  --radius-sm: 4px;
  --radius-md: 8px;
  --radius-lg: 12px;
  --radius-full: 9999px;

  /* Shadow */
  --shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.05);
  --shadow-md: 0 4px 6px rgba(0, 0, 0, 0.07);
  --shadow-lg: 0 10px 15px rgba(0, 0, 0, 0.1);

  /* Motion */
  --transition: 0.15s ease;
}
```

## Rules

- **Variables only.** The theme file declares `:root { ... }` custom properties. It
  does not style elements. Element styles live in the sketch's own CSS and reference
  the variables: `background: var(--bg-elevated);`.
- **No component classes.** The theme is the vocabulary, not the components. A sketch
  that needs a `.card` class puts it in its own `<style>` block, using theme variables
  for every value.
- **One theme per topic by default.** Multiple themes (light/dark, brand variants) go
  in separate files in `themes/` (`light.css`, `dark.css`, `warm-minimal.css`); the
  toolbar's theme switcher swaps which file is linked.
- **The winner's theme is liftable.** If a specific theme file is what the user chose,
  it goes in the findings as "what's liftable" — the real build links the same file.

## Switching themes at runtime

The toolbar's theme switcher works by swapping the `<link>`'s `href`:

```js
function setTheme(name) {
  document.querySelector('link[data-theme]').href = `../themes/${name}.css`;
}
```

The sketch's `<link rel="stylesheet" data-theme href="../themes/default.css">` is the
single swap point. No JS re-render is needed — CSS variables cascade, so every element
updates on the next frame.

## Seeding from the project

If the project already has a design system (CSS variables, a Tailwind config, a
`DESIGN.md`), seed the theme from it rather than inventing new values. The sketch is
testing a *layout/interaction* — it should not also be testing a new color palette the
project has already decided on. Read the project's existing tokens and use them; only
introduce new tokens for things the project has not decided (e.g., a new component's
spacing).
