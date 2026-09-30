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

  const byColumn: Record<BoardColumn, UnifiedTask[]> = {
    todo: [],
    in_progress: [],
    blocked: [],
    done: [],
  }
  for (const t of tasks) byColumn[t.board]?.push(t)

  function handleDragEnd(event: DragEndEvent) {
    const { active, over } = event
    if (!over) return
    const overId = String(over.id)
    // Over a column id or a task id (the task's column is the drop target).
    let target: BoardColumn | null = null
    if (BOARD_COLUMNS.includes(overId as BoardColumn)) {
      target = overId as BoardColumn
    } else {
      const t = tasks.find((x) => x.id === overId)
      if (t) target = t.board
    }
    if (target && target !== byColumn[overId as BoardColumn]?.find((x) => x.id === String(active.id))?.board) {
      onMove(String(active.id), target)
    }
  }

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={closestCorners}
      onDragEnd={handleDragEnd}
    >
      <div className="flex gap-3 overflow-x-auto p-4">
        {BOARD_COLUMNS.map((col) => (
          <div
            key={col}
            className="flex w-72 shrink-0 flex-col rounded-md border border-edge bg-surface-2/60"
          >
            <div className="flex items-center gap-2 border-b border-edge px-3 py-2.5">
              <span className={`h-2 w-2 rounded-full ${COLUMN_COLOR[col]}`} />
              <span className="font-mono text-[12px] font-semibold uppercase tracking-[0.12em] text-ink-3">
                {COLUMN_LABEL[col]}
              </span>
              <span className="ml-auto rounded bg-edge/60 px-1.5 text-[11px] text-ink-5">
                {byColumn[col].length}
              </span>
            </div>
            <SortableContext
              items={byColumn[col].map((t) => t.id as UniqueIdentifier)}
              strategy={rectSortingStrategy}
            >
              <div className="flex flex-col gap-2 p-2">
                {byColumn[col].map((t) => (
                  <SortableCard
                    key={t.id}
                    task={t}
                    onClick={() => onOpenTask(t.id)}
                    disabled={disabled}
                  />
                ))}
                {byColumn[col].length === 0 && (
                  <div className="rounded border border-dashed border-edge/70 p-4 text-center font-mono text-[11px] text-ink-6">
                    {disabled ? '—' : 'No tasks'}
                  </div>
                )}
              </div>
            </SortableContext>
          </div>
        ))}
      </div>
    </DndContext>
  )
}
