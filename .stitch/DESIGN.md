---
name: Agent Control
colors:
  background: '#10141b'
  card: '#151b25'
  popover: '#151b25'
  foreground: '#dce2eb'
  on-card: '#dce2eb'
  primary: '#54c2d8'
  on-primary: '#161c24'
  primary-container: 'rgba(84, 194, 216, 0.10)'
  secondary: '#1f2632'
  on-secondary: '#dce2eb'
  muted: '#1c2330'
  muted-foreground: '#97a3b3'
  accent: '#1e2c33'
  on-accent: '#e3e9f1'
  destructive: '#d4593f'
  border: 'rgba(255, 255, 255, 0.09)'
  input: 'rgba(255, 255, 255, 0.12)'
  ring: '#54c2d8'
  chart-1: '#54c2d8'
  chart-2: '#4bc98a'
  chart-3: '#d9bd54'
  chart-4: '#e08a54'
  chart-5: '#cc6fd4'
  sidebar: '#0c1016'
  sidebar-foreground: '#ccd4df'
  sidebar-accent: '#1b222e'
  sidebar-border: 'rgba(255, 255, 255, 0.08)'
  surface-dim: '#0a0e13'
  surface-container-lowest: '#1a212d'
  surface-container-low: '#1f2632'
  surface-container: '#1c2330'
  surface-container-high: '#232c3b'
  surface-container-highest: '#2a3446'
  on-background: '#dce2eb'
  on-surface-variant: '#97a3b3'
  outline: '#97a3b3'
  outline-variant: 'rgba(255, 255, 255, 0.09)'
  inverse-surface: '#dce2eb'
  inverse-on-surface: '#161c24'
  primary-fixed: '#b8ecf6'
  primary-fixed-dim: '#54c2d8'
  on-primary-fixed: '#0a2b33'
  on-primary-fixed-variant: '#123841'
  secondary-fixed: '#2a3342'
  secondary-fixed-dim: '#1f2632'
  on-secondary-fixed: '#e8edf4'
  on-secondary-fixed-variant: '#3a4557'
  error: '#d4593f'
  on-error: '#ffffff'
  error-container: '#3a1d17'
  on-error-container: '#f3b3a3'
  background-alt: '#0b0f16'
typography:
  display-lg:
    fontFamily: Geist
    fontSize: 30px
    fontWeight: '600'
    lineHeight: 38px
    letterSpacing: -0.01em
  headline-md:
    fontFamily: Geist
    fontSize: 20px
    fontWeight: '600'
    lineHeight: 28px
    letterSpacing: -0.005em
  headline-sm:
    fontFamily: Geist
    fontSize: 18px
    fontWeight: '600'
    lineHeight: 26px
    letterSpacing: '0'
  body-lg:
    fontFamily: Geist
    fontSize: 16px
    fontWeight: '400'
    lineHeight: 26px
    letterSpacing: '0'
  body-md:
    fontFamily: Geist
    fontSize: 14px
    fontWeight: '400'
    lineHeight: 22px
    letterSpacing: '0'
  body-bold:
    fontFamily: Geist
    fontSize: 14px
    fontWeight: '500'
    lineHeight: 22px
    letterSpacing: '0'
  label-caps:
    fontFamily: 'Geist Mono'
    fontSize: 11px
    fontWeight: '500'
    lineHeight: 16px
    letterSpacing: 0.05em
  label-xs:
    fontFamily: 'Geist Mono'
    fontSize: 10px
    fontWeight: '500'
    lineHeight: 14px
    letterSpacing: 0.04em
  stat-lg:
    fontFamily: 'Geist Mono'
    fontSize: 28px
    fontWeight: '600'
    lineHeight: 36px
    letterSpacing: -0.02em
  code:
    fontFamily: 'Geist Mono'
    fontSize: 12.5px
    fontWeight: '400'
    lineHeight: 20px
    letterSpacing: '0'
rounded:
  sm: 0.3rem
  md: 0.4rem
  DEFAULT: 0.5rem
  lg: 0.75rem
  xl: 1rem
  full: 9999px
spacing:
  unit: 4px
  xs: 4px
  sm: 8px
  md: 16px
  lg: 24px
  xl: 32px
  gutter: 16px
  margin-mobile: 24px
  margin-desktop: 40px
---

# Design System: Agent Control (Skillgrid UI)

**Project:** Web admin dashboard for a skills/rules/hooks-backed AI agent workflow. Next.js 16 + Tailwind v4 + shadcn (base-ui) components, dark-first.

## 1. Visual Theme & Atmosphere

Agent Control is a dark, technical control room for operators who watch an
AI agent's state — backlog, docs, memory, code index, sessions. The mood is
that of a terminal: a near-black blue-gray canvas (`#10141b`) with an even
darker sidebar (`#0c1016`), hairline white borders at ~9% opacity, and a single
glowing cyan accent (`#54c2d8`) that marks everything live and active. The
topbar's `~/backlog` path crumb in monospace and the pulsing "connected" dot
make the interface read as a connected instrument panel, not a marketing site.

The design philosophy is **restrained density**: a 4/8px baseline rhythm,
compact 32px rows, and monospace labels for machine data (ids, timestamps,
versions, statuses) while human prose stays in Geist. Depth is flat — no heavy
shadows, only border contrast and subtle translucent fills
(`bg-background/40`, `bg-primary/10`). The result is legible at low light,
high information density without noise, and a consistent "dev tool" voice.

## 2. Color Palette & Roles

### Primary Foundation

- **Ink Canvas** `#101416`-range (`oklch(0.155 0.012 255) ≈ #10141b`) — the
  root page background.
- **Panel Card** `#151b25` — card, popover, and drawer surfaces; one step
  lighter than the canvas so panels read as raised without shadows.
- **Deep Sidebar** `#0c1016` — the navigation rail, the darkest surface in the
  system, creating a frame around the content.
- **Well** `#0a0e13` — code blocks and inset wells inside cards.

### Accent & Interactive

- **Cyan Signal** `#54c2d8` (`oklch(0.72 0.13 195)`) — the single primary
  accent: active nav, primary buttons, links, focus rings, live indicators.
  Never used for body text.
- **Cyan Wash** `rgba(84,194,216,0.10)` — soft fills for active states, icon
  plates, and "current" chips.
- **Accent Teal Surface** `#1e2c33` — hover/selected surfaces in lists and nav.

### Typography & Text Hierarchy

- **Signal White** `#dce2eb` — primary text; slightly blue, never pure white.
- **Dimmed** `#97a3b3` — secondary text, placeholders, timestamps, muted
  labels.
- **Faint** `rgba(255,255,255,0.09)` / `0.08` — all borders and dividers.

### Functional States

- **Coral Alert** `#d4593f` — destructive actions, blocked tasks, error
  banners (10–20% washes for backgrounds, 30–40% for borders).
- **Chart Green** `#4bc98a` — done/complete status dots.
- **Chart Amber** `#d9bd54` — high-priority badges.
- **Chart Orange** `#e08a54` — PUT operations, warm warnings.
- **Chart Magenta** `#cc6fd4` — fifth chart series only.

## 3. Typography Rules

### Hierarchy & Weights

Two families only. **Geist** (sans) for all human-readable UI text; **Geist
Mono** for machine data — task ids, file paths, timestamps, versions, status
labels, section eyebrows, badges. This dual-voice split *is* the design
language: monospace = data, sans = prose.

- **Display** (page titles): Geist 30px / 600 / -0.01em, e.g. "Agent Control".
- **Headlines**: Geist 18–20px / 600. Markdown h1 = 20px with a bottom hairline
  rule; h2 = 18px with a 60%-opacity rule; h3 = 16px, no rule.
- **Body**: Geist 14px / 400 / 22px line-height; lead paragraphs 16px/26px.
- **UI labels**: Geist 14px / 500 for buttons and card titles.
- **Eyebrows & badges**: Geist Mono 10–11px, 500, uppercased, +0.04–0.05em
  tracking — "ROADMAP", "DOCUMENTS", priority chips, the `~/backlog` crumb.
- **Stats**: Geist Mono 28px / 600, tight tracking, for big numbers.
- **Code**: Geist Mono 12.5px; inline code sits on a secondary wash with a
  hairline border; block code sits on the Well surface in a bordered, rounded
  container.

### Spacing Principles

Letter-spacing is used as a semantic device: tight (-0.01em) on display
titles for strength, positive (+0.04–0.05em) only on uppercased monospace
labels for the "instrument panel" read. Body text never gets tracking.
Line-height is relaxed (1.5–1.6) for prose in docs, compact (1.25–1.3) for
card titles and dense board rows.

## 4. Component Stylings

### Buttons

Compact by default (h-8) — this is a dense tool, not a marketing page.
Rounded-lg (0.75rem). Variants:

- **Primary**: solid Cyan Signal fill, near-black text (`#161c24`), 80% fill on
  hover.
- **Secondary**: Panel/secondary surface fill, light text, 80% on hover.
- **Outline**: hairline border, transparent → muted wash on hover.
- **Ghost**: no border; muted-wash hover only.
- **Destructive**: 10–20% Coral wash, coral text — never a solid red fill.
- **Sizes**: xs h-6, sm h-7, default h-8, lg h-9; icon buttons square at the
  same heights. Focus = 3px ring at 50% cyan.

### Cards & Containers

Cards are **flat and bordered**, not shadowed. Corner radius lg (0.75rem).
Hairline border (`rgba(255,255,255,0.09)`) over Panel Card `#151b25`. Internal
padding 16px. Interactive cards (e.g. task cards) shift to a 40% cyan border
and a 40% accent wash on hover. Disabled/coming-soon cards use a **dashed**
hairline border over a 40%-opacity card surface. Empty states are dashed
border, centered, generous vertical padding, monospace "empty" hint.

### Navigation

- **Sidebar** (240px, hidden below md): Deep Sidebar background, hairline right
  border. Brand block is a 28px rounded-md cyan plate with a terminal glyph,
  name in Geist 14px/600, version in 10px mono dimmed. Section eyebrows in 10px
  mono uppercase. Nav rows: 16px vertical rhythm, icon + label, muted →
  full-foreground on hover with a 60% accent wash; active rows get a solid
  accent wash + medium weight. Footer shows a live data-source chip (cyan
  radio glyph + mono label) on a 40% accent wash.
- **Topbar** (56px): translucent canvas background with backdrop blur, hairline
  bottom border. Left: `~/section` mono breadcrumb with cyan `~`. Right:
  pill-shaped "connected" chip — hairline border, 6px cyan dot, 11px mono.

### Inputs & Forms

Inputs: rounded-md, 12% white fill (`bg-input/40`), hairline border, 14px text,
left icon inset 10px. Focus = 2px cyan ring, no border color change. Segment
controls (provider switcher, priority filter) are bordered pills containing
rounded-md tabs, p-1 container; active tab = solid cyan (provider) or secondary
wash (priority filter) with mono 10px uppercase labels.

### Domain-Specific Components

- **Board columns** (backlog): four-column grid (xl) / 2 (md) / 1 (mobile).
  Each column is a rounded-lg section on a 40%-opacity canvas with a hairline
  border; header has a mono uppercase eyebrow + a 20px square count chip on a
  secondary wash.
- **Task cards**: rounded-md, Panel surface, 12px padding. Row 1: mono task id
  (11px dimmed) + priority badge (mono 10px uppercase, colored wash: critical
  = coral, high = amber, medium = cyan, low = neutral). Title in Geist
  14px/500. Label chips in mono 10px on secondary wash. Footer row: assignee
  initials in a 20px mono circle, doc-ref count, relative timestamp — all
  10px mono dimmed.
- **Task detail drawer**: right-side sheet, max-width 28rem, Panel surface with
  a left hairline border, over a 70% canvas backdrop with light blur. Header
  = status dot + mono id, with "open" external link and close button. Body uses
  a 3-column definition grid (label / value), a bordered pre for the
  description, and a footer with created/updated mono timestamps. Closes on
  Escape.
- **Docs view**: left file nav (mono titles, active = accent wash) + right
  article card. Article header shows the file path in mono with a cyan doc
  glyph and a relative-updated stamp; "linked tasks" render as small bordered
  mono chips with an arrow-out glyph; markdown renders per the typography
  rules above (ruled headings, ruled tables, 60%-cyan blockquote bars).
- **Status dots**: 8px circles — todo = dimmed, in-progress = cyan, blocked =
  coral, done = chart green.

## 5. Layout Principles

### Grid & Structure

App shell is a full-viewport flex row: fixed 240px sidebar + fluid main column
(56px topbar + scrollable content). Content is centered at **max-width 56rem
(896px)** with 24px horizontal padding on desktop. The backlog board is the
exception — a full-width 4-column grid (12px gutters) for maximum density.

### Whitespace Strategy

Strict 4px base unit (Tailwind scale). Section gaps 16px, major section
separation 32–40px. Card internals 12–16px. The system is deliberately compact:
board rows, nav rows, and definition grids all target 32–40px line heights.
No decorative whitespace — every gap is structural.

### Alignment & Visual Balance

Left-aligned everywhere except empty/loading states (centered). The cyan
accent appears at most once per card/row (active dot, badge, or icon plate) to
keep the eye anchored. Machine data right-aligns its own micro-grid (ids,
counts, timestamps) while prose stays left.

### Responsive Behavior & Touch

Mobile-first collapse: sidebar disappears below md (nav becomes topbar-driven),
board goes 4 → 2 → 1 columns, toolbars stack vertically, task detail drawer
becomes full-width. Touch targets stay at a practical 28–32px (desktop-first
tool); interactive rows keep full-width hit areas.

## 6. Design System Notes for Stitch Generation

### Language to Use

"Dark technical control room", "near-black blue-gray canvas", "hairline white
borders at 9% opacity", "single glowing cyan accent", "flat panels, no heavy
shadows", "monospace for machine data, clean geometric sans for prose",
"compact instrument-panel density", "terminal aesthetic".

### Color References

Always reference colors by name + hex: Ink Canvas `#10141b`, Panel Card
`#151b25`, Deep Sidebar `#0c1016`, Cyan Signal `#54c2d8`, Signal White
`#dce2eb`, Dimmed Gray-Blue `#97a3b3`, Coral Alert `#d4593f`, Chart Green
`#4bc98a`, Chart Amber `#d9bd54`. Borders: white at 9% opacity. Active
highlights: cyan at 10% fill.

### Component Prompts

1. "A dark dashboard sidebar: near-black `#0c1016` rail, 28px rounded cyan
   `#54c2d8` icon plate with a terminal glyph, 'Agent Control' title in Geist
   semibold with a 10px monospace version number below, section eyebrows in
   uppercase monospace, nav rows with 16px icons, active row on a dark teal
   wash, hairline white 9% borders, a data-source status chip with a pulsing
   cyan dot at the bottom."
2. "A Kanban backlog board on a near-black blue-gray `#10141b` canvas: four
   flat columns with hairline borders on a 40%-opacity surface, uppercase
   monospace column headers with small square count chips, compact task cards
   with monospace task ids, colored 10px uppercase priority badges (coral
   critical, amber high, cyan medium), label chips, and dimmed monospace
   timestamps in the footer."
3. "A right-side task detail drawer: dark `#151b25` panel with a hairline left
   border over a blurred 70%-black backdrop, header with a status dot and
   monospace task id, a 3-column label/value metadata grid, a bordered
   monospace description block on a darker well, and a small monospace
   created/updated footer."

### Incremental Iteration

When editing existing screens, preserve the flat/bordered panel language —
never introduce drop shadows or gradient fills. Keep the cyan accent singular
per view; if a screen starts looking "colorful", move secondary meaning to the
chart palette (green/amber/orange) at low opacity washes. Keep all machine
data in Geist Mono 10–12px uppercase where it is a label, and Geist 14px for
values. Dashed borders are the reserved idiom for "not yet available" surfaces.
