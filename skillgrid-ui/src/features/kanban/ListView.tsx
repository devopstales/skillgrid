import {
  COLUMN_COLOR,
  COLUMN_LABEL,
  type UnifiedTask,
} from './types'

export function ListView({
  tasks,
  onOpenTask,
}: {
  tasks: UnifiedTask[]
  onOpenTask: (id: string) => void
}) {
  return (
    <div className="p-4 font-mono">
      <table className="w-full text-[13px]">
        <thead>
          <tr className="border-b border-edge text-left text-[11px] uppercase tracking-[0.1em] text-ink-6">
            <th className="px-3 py-2 font-medium">ID</th>
            <th className="px-3 py-2 font-medium">Title</th>
            <th className="px-3 py-2 font-medium">Status</th>
            <th className="px-3 py-2 font-medium">Board</th>
            <th className="px-3 py-2 font-medium">Priority</th>
            <th className="px-3 py-2 font-medium">Assignee</th>
            <th className="px-3 py-2 font-medium">Milestone</th>
          </tr>
        </thead>
        <tbody>
          {tasks.map((t) => (
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
              <td className="px-3 py-2 text-ink-4">{t.milestone ?? '—'}</td>
            </tr>
          ))}
          {tasks.length === 0 && (
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
