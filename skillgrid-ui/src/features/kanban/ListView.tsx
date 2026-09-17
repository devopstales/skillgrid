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
    <div className="p-4">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-edge text-left text-xs uppercase tracking-wide text-zinc-500">
            <th className="px-3 py-2">ID</th>
            <th className="px-3 py-2">Title</th>
            <th className="px-3 py-2">Status</th>
            <th className="px-3 py-2">Board</th>
            <th className="px-3 py-2">Priority</th>
            <th className="px-3 py-2">Assignee</th>
            <th className="px-3 py-2">Milestone</th>
          </tr>
        </thead>
        <tbody>
          {tasks.map((t) => (
            <tr
              key={t.id}
              onClick={() => onOpenTask(t.id)}
              className="cursor-pointer border-b border-edge/40 hover:bg-edge/20"
            >
              <td className="px-3 py-2 font-mono text-xs text-zinc-500">{t.id}</td>
              <td className="px-3 py-2 text-zinc-200">{t.title}</td>
              <td className="px-3 py-2 text-zinc-400">{t.status}</td>
              <td className="px-3 py-2">
                <span className="flex items-center gap-1.5 text-zinc-400">
                  <span
                    className={`h-2 w-2 rounded-full ${COLUMN_COLOR[t.board]}`}
                  />
                  {COLUMN_LABEL[t.board]}
                </span>
              </td>
              <td className="px-3 py-2 text-zinc-400">{t.priority ?? '—'}</td>
              <td className="px-3 py-2 text-zinc-400">
                {t.assignees?.length ? t.assignees.join(', ') : '—'}
              </td>
              <td className="px-3 py-2 text-zinc-400">{t.milestone ?? '—'}</td>
            </tr>
          ))}
          {tasks.length === 0 && (
            <tr>
              <td colSpan={7} className="px-3 py-8 text-center text-zinc-600">
                No tasks
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  )
}
