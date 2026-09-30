import type { KanbanFilters, UnifiedTask } from './types'

// FilterBar narrows the board by query (title/id/label), assignee, label,
// milestone, and priority. The distinct values are derived from the loaded
// tasks, so the selects only offer real options (never invented).
export function FilterBar({
  filters,
  onChange,
  tasks,
}: {
  filters: KanbanFilters
  onChange: (f: KanbanFilters) => void
  tasks: UnifiedTask[]
}) {
  const distinct = (pick: (t: UnifiedTask) => string[] | string | undefined) => {
    const set = new Set<string>()
    for (const t of tasks) {
      const v = pick(t)
      if (Array.isArray(v)) v.forEach((x) => set.add(x))
      else if (v) set.add(v)
    }
    return Array.from(set).sort()
  }
  const assignees = distinct((t) => t.assignees)
  const labels = distinct((t) => t.labels)
  const milestones = distinct((t) => t.milestone)
  const priorities = distinct((t) => t.priority)

  const select = (
    value: string,
    options: string[],
    key: keyof KanbanFilters,
    placeholder: string,
  ) => (
    <select
      value={value}
      onChange={(e) => onChange({ ...filters, [key]: e.target.value })}
      className="rounded border border-edge-soft bg-inset px-2 py-1.5 text-[12px] text-ink-3 focus:border-accent focus:outline-none"
    >
      <option value="">{placeholder}</option>
      {options.map((o) => (
        <option key={o} value={o}>
          {o}
        </option>
      ))}
    </select>
  )

  return (
    <div className="flex flex-wrap items-center gap-2 border-b border-edge-soft px-4 py-3 font-mono">
      <input
        type="text"
        placeholder="filter (title, id, label)…"
        value={filters.query}
        onChange={(e) => onChange({ ...filters, query: e.target.value })}
        className="w-56 rounded border border-edge-soft bg-inset px-2.5 py-1.5 text-[12px] text-ink-3 placeholder:text-ink-6 focus:border-accent focus:outline-none"
      />
      {select(filters.assignee, assignees, 'assignee', 'All assignees')}
      {select(filters.label, labels, 'label', 'All labels')}
      {select(filters.milestone, milestones, 'milestone', 'All milestones')}
      {select(filters.priority, priorities, 'priority', 'All priorities')}
      {(filters.query || filters.assignee || filters.label || filters.milestone || filters.priority) && (
        <button
          type="button"
          onClick={() =>
            onChange({ query: '', assignee: '', label: '', milestone: '', priority: '' })
          }
          className="rounded px-2 py-1.5 text-[12px] text-ink-5 hover:text-ink-2"
        >
          Clear
        </button>
      )}
    </div>
  )
}

export function applyFilters(
  tasks: UnifiedTask[],
  f: KanbanFilters,
): UnifiedTask[] {
  const q = f.query.trim().toLowerCase()
  return tasks.filter((t) => {
    if (q) {
      const hay = [
        t.id,
        t.title,
        ...(t.labels ?? []),
      ]
        .join(' ')
        .toLowerCase()
      if (!hay.includes(q)) return false
    }
    if (f.assignee && !(t.assignees ?? []).includes(f.assignee)) return false
    if (f.label && !(t.labels ?? []).includes(f.label)) return false
    if (f.milestone && t.milestone !== f.milestone) return false
    if (f.priority && t.priority !== f.priority) return false
    return true
  })
}
