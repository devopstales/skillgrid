// AdrEntry is one row of the ADR index table (03-adr-index.md).
export interface AdrEntry {
  id: string // e.g. "0001"
  title: string
  status: string // "accepted" | "superseded" | "proposed" | ...
  date: string // e.g. "2026-09-16"
  inForce: boolean
  path: string // full repo-relative path for /docs/content
  supersedes: string // "—" or a sequence like "0006"
}

// parseAdrIndex extracts the in-force table rows from the ADR index markdown.
// The index is a GFM table: | # | Title | Status | Supersedes | Date | In force | Record |
// The Record column carries the path to the full record file, either as a
// markdown link relative to .skillgrid/artifacts/ (legacy 03-adr-index.md) or
// as a backticked repo-relative path (ASSUMPTIONS.md § In-force set, ADR-0019).
export function parseAdrIndex(markdown: string): AdrEntry[] {
  const lines = markdown.split('\n')
  const entries: AdrEntry[] = []

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    if (!line.trim().startsWith('|')) continue

    const cells = splitTableRow(line)
    if (cells.length < 7) continue

    // Skip the header row and the separator row (---|---|...).
    const isSeparator = cells.every((c) => /^:?-+:?$/.test(c.trim()))
    if (isSeparator) continue

    // The first cell of the header is "#"; the first cell of a data row is the
    // numeric sequence (e.g. "0001"). Use that to skip the header.
    const id = cells[0].trim()
    if (!/^\d+$/.test(id)) continue

    entries.push({
      id,
      title: cells[1].trim(),
      status: cells[2].trim(),
      supersedes: cells[3].trim(),
      date: cells[4].trim(),
      inForce: /yes/i.test(cells[5]),
      path: recordPath(cells[6]),
    })
  }

  return entries
}

// recordPath resolves the Record cell to a repo-relative path for /docs/content.
// Accepts `[label](rel.md)`, `` `path.md` ``, or a bare `path.md`. A path that
// already starts with `.skillgrid/` is kept; anything else is treated as
// relative to `.skillgrid/artifacts/`.
function recordPath(cell: string): string {
  const link = /\(([^)]+\.md)\)/.exec(cell)
  const raw = link ? link[1] : (/([^\s`()[\]]+\.md)/.exec(cell)?.[1] ?? '')
  const trimmed = raw.trim()
  if (!trimmed) return ''
  if (trimmed.startsWith('.skillgrid/')) return trimmed
  return `.skillgrid/artifacts/${trimmed.replace(/^[./]+/, '')}`
}

// splitTableRow splits a markdown table row into trimmed cells, dropping the
// leading/trailing empty cells produced by the outer pipes.
function splitTableRow(line: string): string[] {
  const trimmed = line.trim().replace(/^\|/, '').replace(/\|$/, '')
  return trimmed.split('|').map((c) => c.trim())
}
