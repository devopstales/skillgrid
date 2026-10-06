import { useState } from 'react'
import {
  DndContext,
  KeyboardSensor,
  PointerSensor,
  closestCorners,
  useSensor,
  useSensors,
  type DragEndEvent,
  type UniqueIdentifier,
} from '@dnd-kit/core'
import { SortableContext, rectSortingStrategy, useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import {
  BOARD_COLUMNS,
  COLUMN_COLOR,
  COLUMN_LABEL,
  type BoardColumn,
  type UnifiedTask,
} from './types'
import { TaskCard } from './TaskCard'
import { groupColumnTasks, groupTasksByMilestone } from './epicTree'

function SortableCard({
  task,
  onClick,
  disabled,
}: {
  task: UnifiedTask
  onClick: () => void
  disabled: boolean
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } =
    useSortable({ id: task.id, disabled })
  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  }
  return (
    <div
      ref={setNodeRef}
      style={style}
      {...(disabled ? {} : attributes)}
      {...(disabled ? {} : listeners)}
    >
      <TaskCard task={task} onClick={onClick} dragging={isDragging} />
    </div>
  )
}

export function BoardView({
  tasks,
  onOpenTask,
  onMove,
  disabled,
}: {
  tasks: UnifiedTask[]
  onOpenTask: (id: string) => void
  onMove: (id: string, to: BoardColumn) => void
  disabled: boolean
}) {
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
    useSensor(KeyboardSensor),
  )
  const [collapsed, setCollapsed] = useState<Set<string>>(new Set())

  const byColumn: Record<BoardColumn, UnifiedTask[]> = {
    todo: [],
    ready: [],
    in_progress: [],
    blocked: [],
    done: [],
  }
  for (const t of tasks) byColumn[t.board]?.push(t)

  const milestoneGroups = groupTasksByMilestone(tasks)

  function toggle(name: string) {
    setCollapsed((prev) => {
      const next = new Set(prev)
      if (next.has(name)) next.delete(name)
      else next.add(name)
      return next
    })
  }

  function handleDragEnd(event: DragEndEvent) {
    const { active, over } = event
    if (!over) return
    const overId = String(over.id)
    let target: BoardColumn | null = null
    if (BOARD_COLUMNS.includes(overId as BoardColumn)) {
      target = overId as BoardColumn
    } else {
      const t = tasks.find((x) => x.id === overId)
      if (t) target = t.board
    }
    if (target) onMove(String(active.id), target)
  }

  function renderCards(tasks: UnifiedTask[]) {
    return groupColumnTasks(tasks).map((group) =>
      'parent' in group ? (
        <div key={group.parent.id} className="flex flex-col gap-2">
          <SortableCard
            task={group.parent}
            onClick={() => onOpenTask(group.parent.id)}
            disabled={disabled}
          />
          <div className="ml-3 flex flex-col gap-2 border-l border-edge/70 pl-2">
            <div className="font-mono text-[10px] uppercase tracking-[0.12em] text-ink-6">
              {group.parent.id} · {group.children.length}{' '}
              {group.children.length === 1 ? 'child' : 'children'}
            </div>
            {group.children.map((c) => (
              <SortableCard
                key={c.id}
                task={c}
                onClick={() => onOpenTask(c.id)}
                disabled={disabled}
              />
            ))}
          </div>
        </div>
      ) : (
        <SortableCard
          key={group.task.id}
          task={group.task}
          onClick={() => onOpenTask(group.task.id)}
          disabled={disabled}
        />
      ),
    )
  }

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={closestCorners}
      onDragEnd={handleDragEnd}
    >
      <div className="flex flex-col gap-2 p-4">
        {/* Column header bar */}
        <div className="flex gap-3">
          {BOARD_COLUMNS.map((col) => (
            <div
              key={col}
              className="flex flex-1 min-w-45 items-center gap-2 rounded bg-edge/30 px-3 py-2"
            >
              <span className={`h-2 w-2 rounded-full ${COLUMN_COLOR[col]}`} />
              <span className="font-mono text-[11px] font-semibold uppercase tracking-[0.1em] text-ink-3">
                {COLUMN_LABEL[col]}
              </span>
              <span className="ml-auto text-[10px] text-ink-5">
                {byColumn[col].length}
              </span>
            </div>
          ))}
        </div>

        {/* Milestone rows */}
        {milestoneGroups.map((group) => {
          const isCollapsed = collapsed.has(group.name)
          const total = group.tasks.length
          const done = group.tasks.filter((t) => t.board === 'done').length
          const pct = total > 0 ? Math.round((done / total) * 100) : 0
          return (
            <div
              key={group.name || '__none'}
              className="overflow-hidden rounded-md border border-edge bg-card"
            >
              <button
                type="button"
                onClick={() => toggle(group.name)}
                className="flex w-full items-center gap-2 px-3 py-2.5 transition-colors hover:bg-edge/10"
              >
                <svg
                  className={`h-3.5 w-3.5 shrink-0 text-ink-5 transition-transform ${isCollapsed ? '' : 'rotate-90'}`}
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth={2.5}
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <path d="M9 18l6-6-6-6" />
                </svg>
                <span
                  className={`truncate font-mono text-[12px] font-semibold uppercase tracking-[0.08em] ${group.name ? 'text-accent' : 'text-ink-5'}`}
                >
                  {group.name || 'Unassigned'}
                </span>
                <div className="ml-auto flex shrink-0 items-center gap-2">
                  <div className="h-1 w-20 overflow-hidden rounded-full bg-edge/50">
                    <div
                      className={`h-full rounded-full transition-all duration-300 ${pct === 100 ? 'bg-ok' : 'bg-accent'}`}
                      style={{ width: `${pct}%` }}
                    />
                  </div>
                  <span className="text-[10px] text-ink-5">
                    {done}/{total}
                  </span>
                  <span className="rounded bg-edge/60 px-1.5 text-[10px] text-ink-4">
                    {total}
                  </span>
                </div>
              </button>
              {!isCollapsed && (
                <div className="border-t border-edge-soft p-2">
                  <div className="flex gap-2">
                    {BOARD_COLUMNS.map((col) => {
                      const colTasks = group.tasks.filter((t) => t.board === col)
                      return (
                        <div
                          key={col}
                          className="flex min-h-12 flex-1 min-w-45 flex-col gap-1.5 rounded border border-edge-soft bg-surface-900/40 p-1.5"
                        >
                          {colTasks.length === 0 ? (
                            <span className="flex items-center justify-center py-2 font-mono text-[11px] text-ink-6">
                              —
                            </span>
                          ) : (
                            <SortableContext
                              items={colTasks.map((t) => t.id as UniqueIdentifier)}
                              strategy={rectSortingStrategy}
                            >
                              {renderCards(colTasks)}
                            </SortableContext>
                          )}
                        </div>
                      )
                    })}
                  </div>
                </div>
              )}
            </div>
          )
        })}
      </div>
    </DndContext>
  )
}
