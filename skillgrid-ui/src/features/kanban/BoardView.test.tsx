import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { BoardView } from './BoardView'
import { type UnifiedTask } from './types'

function task(over: Partial<UnifiedTask> & Pick<UnifiedTask, 'id'>): UnifiedTask {
  return { title: over.title ?? over.id, status: 'needs-triage', board: 'todo', provider: 'backlogmd', ...over }
}

describe('BoardView epic tree', () => {
  it('groups subtasks under their parent epic in a column', () => {
    const tasks = [
      task({ id: 'task-50', title: 'Child One', board: 'todo', parent: 'task-10' }),
      task({ id: 'task-10', title: 'Epic Parent', board: 'todo' }),
      task({ id: 'task-51', title: 'Child Two', board: 'todo', parent: 'task-10' }),
      task({ id: 'task-7', title: 'Standalone', board: 'todo' }),
    ]
    render(<BoardView tasks={tasks} onOpenTask={vi.fn()} onMove={vi.fn()} disabled={false} />)
    // The epic header is labelled with the parent id + its child count.
    const epic = screen.getByText(/task-10.*2 children/i)
    expect(epic).toBeTruthy()
    // Children render as cards under the epic.
    expect(screen.getByText('Child One')).toBeTruthy()
    expect(screen.getByText('Child Two')).toBeTruthy()
  })

  it('renders tasks without parents as plain cards (no epic grouping)', () => {
    const tasks = [task({ id: 'task-1', title: 'Solo' })]
    render(<BoardView tasks={tasks} onOpenTask={vi.fn()} onMove={vi.fn()} disabled={false} />)
    expect(screen.getByText('Solo')).toBeTruthy()
    expect(screen.queryByText(/children/i)).toBeNull()
  })

  it('does not double-count a parent that is itself a child elsewhere', () => {
    const tasks = [
      task({ id: 'epic-a', title: 'Epic A', board: 'todo' }),
      task({ id: 'task-1', title: 'Mid', board: 'todo', parent: 'epic-a' }),
      task({ id: 'task-2', title: 'Leaf', board: 'todo', parent: 'task-1' }),
    ]
    render(<BoardView tasks={tasks} onOpenTask={vi.fn()} onMove={vi.fn()} disabled={false} />)
    // task-1 has one child (task-2) and is itself a child of epic-a.
    expect(screen.getByText(/task-1.*1 child/i)).toBeTruthy()
    expect(screen.getByText(/epic-a.*1 child/i)).toBeTruthy()
  })
})
