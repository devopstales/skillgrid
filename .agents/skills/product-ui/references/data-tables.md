# Data Tables & Dense Grids

Conventions for **Operate-dense** surfaces: dashboards, analytics, monitoring,
telemetry, and any grid where the user scans and acts on data. Load this when
the surface class is Operate-dense.

## Column strategy

- **Fewer, wider columns beat many narrow ones.** Start with 4–6 columns. If
  the data needs more, group related columns under a header or move secondary
  data into a detail pane / row expansion.
- **One primary column.** The column the user sorts/filters on most is the
  primary. It gets default sort and the most width.
- **Right-align numbers.** Tabular figures (`font-variant-numeric:
  tabular-nums`) so values line up vertically. Use a monospace or tabular
  face for data-heavy columns.
- **Sticky the first column** when horizontal scroll is unavoidable, so the
  row identity (name / ID) stays visible.

## Sort, filter, search

- **Sort** — single-click asc, second-click desc, third-click clears. Show a
  sort indicator on the active column. Never sort two columns silently.
- **Filter** — put filters in a dedicated filter bar above the grid, not
  buried in a per-column dropdown, unless the column count is small. Label
  each filter with the field it applies to.
- **Search** — a single search box that searches the primary column (and any
  clearly-named secondary) is enough for most grids. Debounce it.
- **Reset** — a single "clear all" that resets sort + filter + search
  together. Partial resets confuse.

## Virtualization & pagination

- **Virtualize** lists past ~200–300 visible rows. Keep a stable row height so
  the scroll position does not jump as rows mount/unmount.
- **Pagination** for bounded datasets where "how many total" matters (billing,
  audit logs). **Infinite scroll** for feeds where the user just wants the
  next batch. Do not mix both on one grid.
- **Stable scroll anchor** — when a new item is inserted or a row updates, the
  viewport must not jump. Anchor to the top visible row.

## Keyboard navigation (mandatory)

A dense grid is unusable without a keyboard path:

- `↑` / `↓` — move the row cursor.
- `←` / `→` or `Tab` — move across columns (where cells are editable).
- `Enter` — open / confirm the selected row.
- `Space` — toggle selection.
- `Shift + ↑/↓` — extend selection.
- `/` — focus the search box (a common, cheap, high-value shortcut).
- `Esc` — clear selection / close the detail pane.

Selection must be visible (row highlight) and announced to screen readers.

## Row states

- **Selected** — distinct from hover. Selection survives sort/filter changes
  (by row ID, not index).
- **Updated / fresh** — a brief, non-intrusive highlight when a row's data
  changes live; must respect `prefers-reduced-motion` and must not scroll the
  viewport.
- **Error / failed row** — inline, per-row; do not dismiss the whole grid
  because one row failed to load.

## Density calibration

For Operate-dense:

- Row height 32–40px.
- Spacing scale 8 / 12 / 16 / 24.
- No card boxes around rows — 1px dividers or zebra striping at most.
- Numbers in monospace / tabular figures.
- Hide decorative elements (avatars can be initials, drop the drop shadow).
