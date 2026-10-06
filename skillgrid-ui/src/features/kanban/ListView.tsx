import { useMemo, useState } from 'react'
import {
  COLUMN_COLOR,
  COLUMN_LABEL,
  type UnifiedTask,
} from './types'
import { sortTasks, type SortKey, type SortDir } from './sortTasks'

const COLUMNS: { key: SortKey; label: string }[] = [
  { key: 'id', label: 'ID' },
  { key: 'title', label: 'Title' },
  { key: 'status', label: 'Status' },
  { key: 'board', label: 'Board' },
  { key: 'priority', label: 'Priority' },
  { key: 'assignee', label: 'Assignee' },
  { key: 'milestone', label: 'Milestone' },
]

export function ListView({
  tasks,
  onOpenTask,
  milestoneTitles,
}: {
  tasks: UnifiedTask[]
  onOpenTask: (id: string) => void
  milestoneTitles?: Record<string, string>
}) {
  const [sort, setSort] = useState<{ key: SortKey; dir: SortDir } | null>(null)

  const sorted = useMemo(() => {
    if (!sort) return tasks
    return sortTasks(tasks, sort.key, sort.dir)
  }, [tasks, sort])

  function toggleSort(key: SortKey) {
    setSort((prev) => {
      if (!prev || prev.key !== key) return { key, dir: 'asc' }
      return { key, dir: prev.dir === 'asc' ? 'desc' : 'asc' }
    })
  }

  return (
    <div className="p-4 font-mono">
      <table className="w-full text-[13px]">
        <thead>
          <tr className="border-b border-edge text-left text-[11px] uppercase tracking-[0.1em] text-ink-6">
            {COLUMNS.map(({ key, label }) => {
              const active = sort?.key === key
              return (
                <th
                  key={key}
                  onClick={() => toggleSort(key)}
                  aria-sort={
                    active ? (sort.dir === 'asc' ? 'ascending' : 'descending') : undefined
                  }
                  className="cursor-pointer select-none px-3 py-2 font-medium hover:text-ink-2"
                >
                  <span className="inline-flex items-center gap-1">
                    {label}
                    {active && (
                      <span className="text-[9px]">{sort.dir === 'asc' ? '▲' : '▼'}</span>
                    )}
                  </span>
                </th>
              )
            })}
          </tr>
        </thead>
        <tbody>
          {sorted.map((t) => (
            <tr
              key={t.id}
              onClick={() => onOpenTask(t.id)}
              className="cursor-pointer border-b border-edge/50 hover:bg-edge/15"
            >
              <td className="px-3 py-2 text-[12px] text-ink-5">{t.id}</td>
              <td className="px-3 py-2 text-ink-2">{t.title}</td>
              <td className="px-3 py-2 text-ink-4">{t.status}</td>
              <td className="px-3 py-2">
                <span className="flex items-center gap-1.5 text-ink-4">
                  <span
                    className={`h-2 w-2 rounded-full ${COLUMN_COLOR[t.board]}`}
                  />
                  {COLUMN_LABEL[t.board]}
                </span>
              </td>
              <td className="px-3 py-2 text-ink-4">{t.priority ?? '—'}</td>
              <td className="px-3 py-2 text-ink-4">
                {t.assignees?.length ? t.assignees.join(', ') : '—'}
              </td>
              <td className="px-3 py-2 text-ink-4">
                {t.milestone ? (milestoneTitles?.[t.milestone] || t.milestone) : '—'}
              </td>
            </tr>
          ))}
          {sorted.length === 0 && (
            <tr>
              <td colSpan={7} className="px-3 py-8 text-center text-ink-6">
                No tasks
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  )
}
