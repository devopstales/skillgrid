# Terminal Ops — Design System

**Owner:** skillgrid dashboard
**Status:** selected (prototype 002 winner) — v1.0
**Interactive spec:** `./design-system.html` (token tables auto-generated from `:root`)
**Build target:** port the token block into `../../src/index.css` under `:root`.

A dense monospace console on an OLED slate canvas. One run-green accent carries
everything that is live or passing. An 8px rhythm, a 12-column grid, hairline
borders, and no shadows — the layout reads like a terminal because the type is
small, the grid is fine-grained, and status is encoded in color and tiny chips
rather than big cards.

---

## 1. Principles

1. **Density is the feature.** 11px labels, 12–13px body, 12px card padding.
   Information per square inch is the goal, not breathing room.
2. **One accent, three states.** Run-green is the only brand hue — it carries
   active, ok, live, and success. Amber is the single warning hue, red is
   reserved for failure. Do not introduce a new brand color.
3. **Separation by line, not by elevation.** No box-shadows. The 1px
   `--line` border plus a one-step surface change (`--bg` → `--surface-1`)
   is the only depth in the system.
4. **Status lives in chips and rows.** Big state is a colored word
   (`PASS`, `17`); detail is a hairline row. Never a colored background block
   for status.
5. **Mono-first type.** The whole UI is set in the mono stack. A system sans is
   available for long prose only.

---

## 2. Color

### Surfaces (one-step elevation only)

| Token | Value | Role |
|-------|-------|------|
| `--bg` | `#0f172a` | app canvas (OLED slate) |
| `--surface-1` | `#1b2336` | card |
| `--surface-2` | `#0d1526` | sidebar / recessed |
| `--surface-inset` | `#0b1120` | inputs, tracks, pill fill |

### Lines

| Token | Value | Role |
|-------|-------|------|
| `--line` | `#272f42` | hairline: card & row borders |
| `--line-soft` | `#1e293b` | structural: panel & topbar borders |

### Text (6-step scale)

| Token | Value | Role |
|-------|-------|------|
| `--text` | `#f8fafc` | primary: headings, KPI, active nav |
| `--text-2` | `#e2e8f0` | secondary: values, secondary titles |
| `--text-3` | `#cbd5e1` | body: list names, prose |
| `--text-4` | `#94a3b8` | muted: labels, descriptions |
| `--text-5` | `#64748b` | faint: card headers, hints |
| `--text-6` | `#475569` | faintest: timestamps, section labels |

### Brand & status

| Token | Value | Role |
|-------|-------|------|
| `--accent` | `#22c55e` | run-green: active, ok, live, success |
| `--accent-ink` | `#14532d` | accent-tinted border (chips, toggle on) |
| `--accent-soft` | `rgba(34,197,94,.07)` | nav active, primary button fill |
| `--accent-tint` | `rgba(34,197,94,.12)` | chart area fill, hover |
| `--ok` | `#22c55e` | success (alias of accent) |
| `--warn` | `#f59e0b` | in-progress, at-risk, pin |
| `--warn-ink` | `#78350f` | warn chip border |
| `--warn-soft` | `rgba(245,158,11,.08)` | active chip fill |
| `--danger` | `#ef4444` | failure, blocked |
| `--danger-ink` | `#7f1d1d` | danger chip border |
| `--danger-soft` | `rgba(239,68,68,.08)` | blocked chip fill |
| `--info` | `#38bdf8` | link / code / user scope |
| `--violet` | `#a78bfa` | global scope |

**Contrast note:** all six text steps sit on `--surface-1` (#1b2336).
`--text-6` (#475569) is decorative-only (timestamps, section labels) and is
intentionally below WCAG AA for body use — never set body copy in it.

---

## 3. Typography

Monospace-first. Numerals use tabular figures for column alignment.

| Token | Size / Weight | Use |
|-------|---------------|-----|
| `--fs-h1` | 30px / 700, ls -.02em | page title, QA gate banner |
| `--fs-kpi` | 26px / 700 | KPI numbers |
| `--fs-title` | 13px / 600 | row titles (observations) |
| `--fs-body` | 13px / 400 | default UI text |
| `--fs-sub` | 12px / 400 | secondary, list metadata |
| `--fs-meta` | 11px / 600, .1em, uppercase | card headers |
| `--fs-tag` | 10px / 400 | type/scope tags |

| Token | Stack |
|-------|-------|
| `--font-mono` | `ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace` |
| `--font-sans` | `system-ui, -apple-system, "Segoe UI", sans-serif` (prose only) |

Rules:
- Default to `--font-mono` for every UI string.
- Use `--font-sans` only inside long-form prose (memory observation bodies).
- Card headers are always `--fs-meta` (uppercase, .1em tracking, `--text-5`).

---

## 4. Spacing & radius

8px base rhythm.

| Token | Value | Token | Value |
|-------|-------|-------|-------|
| `--space-1` | 4px | `--space-5` | 20px |
| `--space-2` | 8px | `--space-6` | 24px |
| `--space-3` | 12px | `--grid-gap` | 12px |
| `--space-4` | 16px | `--card-pad` | 12px |

| Token | Value | Use |
|-------|-------|-----|
| `--r-1` | 3px | tags, status chips |
| `--r-2` | 4px | inputs, pills |
| `--r-3` | 6px | cards |

No radius larger than 6px exists in this system.

---

## 5. Layout

- **Shell:** fixed 220px sidebar (`--side-w`) + fluid content. Topbar is 44px
  (`--topbar-h`) with a breadcrumb on the left and status pills on the right.
- **Grid:** 12 columns (`--cols`), 12px gap (`--grid-gap`), `align-content: start`.
- **Spans:** KPI strip = four × `span 3`; wide panels = `span 8` paired with a
  `span 4` side panel; full-width list pages use `span 12`.
- **Responsive:** below 1100px all partial spans collapse to full width.

```
+--------+----------------------------------------+
| side   |  topbar (breadcrumb .... status pills) |
| 220px  +----------------------------------------+
|  nav   |  12-col grid                            |
|  groups|   [3][3][3][3]   <- KPI strip           |
|  OBSERVE| [8][4]        <- trend + feed           |
|  PIPE  | [12]          <- full-width list         |
|  SYSTEM|                                          |
+--------+----------------------------------------+
```

Nav groups: **OBSERVE** (overview, memory, sessions, code graph),
**PIPELINE** (changes, qa gate, adr), **SYSTEM** (settings). These map 1:1 to
the `skillgrid-ui` feature set and route tree.

---

## 6. Components

### Card
- `--surface-1` fill, `--line` border, `--r-3` radius, `--card-pad` body padding.
- Header: `--fs-meta` uppercase label in `--text-5`, hairline divider, optional
  right-aligned status. No shadow.

### KPI
- Number `--fs-kpi` in `--text`; label `--fs-sub` in `--text-5`; delta
  `--fs-sub` in `--accent` (or `--warn` when at-risk).

### Button
- **Default:** `--surface-inset` fill, `--line` border, `--text-2` label.
- **Primary:** `--accent-soft` fill, `--accent-ink` border, `--accent` label.
  Hover deepens to `--accent-tint`.
- **Disabled:** opacity .4, no pointer.

### Input
- `--surface-inset` fill, `--line` border, `--r-2`. Focus: border → `--accent`.

### Toggle
- 30×16 track, `--r-2`/8px pill, `#334155` off. On: track `--accent-ink`, knob
  `--accent`. 150ms transition.

### Pill (status, topbar)
- `--surface-inset` fill, `--line-soft` border, `--r-2`. Optional 7px
  `--accent` dot for "online".

### Status chip
- `--r-1`, `--fs-sub`. `pass` = accent text/border/soft; `active` = warn;
  `blocked` = danger. Never a solid fill.

### Tag (type/scope)
- `--r-1`, `--fs-tag`, `--line` border, `--text-4`. Tinted variant
  (`.t`) = `--accent` text + `--accent-ink` border.

---

## 7. Charts & data

Three idioms, all resolving from accent + line tokens:

1. **Streaming area** — index health / trends. Stroke `--accent` 1.5px, fill
   `--accent-tint`, grid lines `--line`. No axis labels; the data is the point.
2. **Bullet chart** — KPI vs target. Bar `--accent` on `--surface-inset` track;
   a 2px white `target` tick marks the threshold. Amber bar for at-risk values.
3. **Segmented ratio** — memory scope. project = `--accent`, user = `--info`,
   global = `--violet`; track = `--surface-inset`.

---

## 8. Page patterns

| Page | Pattern |
|------|---------|
| overview | KPI strip + streaming area + event feed + bullet charts + pipeline state + scope ratio |
| memory | search bar + observation rows (type tag, scope tag, pin, time, title, body) |
| sessions | row list: id (green) · name · project · time/live |
| code graph | symbol list → click → definition/callers/callees/blast-radius detail |
| changes | row list with phase + status chip |
| qa gate | green PASS banner + verification ladder (✓ / ! rows) |
| adr | table: number · title · date · status (in force / draft / superseded) |
| settings | toggle list + engine key/value block |

Event feed rows: `<time> <tool> <message>` — time `--text-6`, tool `--accent`,
message `--text-3`.

---

## 9. Do / Don't

**Do**
- Reuse the tokens; every value is a `:root` custom property.
- Keep status as chips/colored words, not colored blocks.
- Use `--accent` for any "good/active/live" signal.
- Use tabular numerals for aligned number columns.

**Don't**
- Introduce gradients, glass, or box-shadows.
- Add a second brand hue (amber/red/info/violet are scoped, not brand).
- Set body copy in `--text-6`; it is decorative-only.
- Exceed 6px radius or 8px spacing steps.
- Use `--font-sans` for UI chrome.
