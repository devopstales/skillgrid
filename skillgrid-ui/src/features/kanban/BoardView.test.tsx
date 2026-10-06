import { render, screen, fireEvent } from '@testing-library/react'
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

describe('BoardView milestone rows', () => {
  it('shows milestone title from titleMap instead of raw ID', () => {
    const tasks = [
      task({ id: 't1', title: 'Task One', milestone: 'm-1', board: 'todo' }),
      task({ id: 't2', title: 'Task Two', milestone: 'm-1', board: 'done' }),
    ]
    render(
      <BoardView
        tasks={tasks}
        onOpenTask={vi.fn()}
        onMove={vi.fn()}
        disabled={false}
        milestoneTitles={{ 'm-1': 'mnemonic-memory-improvements' }}
      />,
    )
    // The milestone row + card badge both show the human-readable title
    expect(screen.getAllByText('mnemonic-memory-improvements').length).toBeGreaterThanOrEqual(1)
    // The raw ID is not shown as the group label
    expect(screen.queryByText(/^m-1$/)).toBeNull()
  })

  it('falls back to raw ID when no title matches', () => {
    const tasks = [task({ id: 't1', title: 'Task One', milestone: 'unknown-ms' })]
    render(<BoardView tasks={tasks} onOpenTask={vi.fn()} onMove={vi.fn()} disabled={false} milestoneTitles={{}} />)
    // Row label + card badge both show the raw ID
    expect(screen.getAllByText('unknown-ms').length).toBeGreaterThanOrEqual(1)
  })

  it('shows Unassigned for tasks without a milestone', () => {
    const tasks = [task({ id: 't1', title: 'No Milestone' })]
    render(<BoardView tasks={tasks} onOpenTask={vi.fn()} onMove={vi.fn()} disabled={false} />)
    expect(screen.getByText('Unassigned')).toBeTruthy()
  })

  it('shows milestone title in task card badge', () => {
    const tasks = [task({ id: 't1', title: 'Task One', milestone: 'm-1' })]
    render(
      <BoardView
        tasks={tasks}
        onOpenTask={vi.fn()}
        onMove={vi.fn()}
        disabled={false}
        milestoneTitles={{ 'm-1': 'memory-improvements' }}
      />,
    )
    // The card badge shows the title (row label + badge)
    expect(screen.getAllByText('memory-improvements').length).toBeGreaterThanOrEqual(1)
  })

  it('shows correct done/total progress in milestone row', () => {
    const tasks = [
      task({ id: 't1', title: 'Done One', milestone: 'm-1', board: 'done' }),
      task({ id: 't2', title: 'Done Two', milestone: 'm-1', board: 'done' }),
      task({ id: 't3', title: 'Todo One', milestone: 'm-1', board: 'todo' }),
      task({ id: 't4', title: 'Todo Two', milestone: 'm-1', board: 'todo' }),
    ]
    render(
      <BoardView
        tasks={tasks}
        onOpenTask={vi.fn()}
        onMove={vi.fn()}
        disabled={false}
        milestoneTitles={{ 'm-1': 'progress-ms' }}
      />,
    )
    // Progress text shows 2/4
    expect(screen.getByText('2/4')).toBeTruthy()
  })

  it('shows 0/total when no tasks are done', () => {
    const tasks = [
      task({ id: 't1', title: 'Todo One', milestone: 'm-1', board: 'todo' }),
      task({ id: 't2', title: 'Todo Two', milestone: 'm-1', board: 'ready' }),
    ]
    render(
      <BoardView
        tasks={tasks}
        onOpenTask={vi.fn()}
        onMove={vi.fn()}
        disabled={false}
        milestoneTitles={{ 'm-1': 'zero-ms' }}
      />,
    )
    expect(screen.getByText('0/2')).toBeTruthy()
  })

  it('shows correct per-column counts in header bar', () => {
    const tasks = [
      task({ id: 't1', title: 'Todo', milestone: 'm-1', board: 'todo' }),
      task({ id: 't2', title: 'Todo2', milestone: 'm-1', board: 'todo' }),
      task({ id: 't3', title: 'Ready', milestone: 'm-1', board: 'ready' }),
      task({ id: 't4', title: 'Done', milestone: 'm-1', board: 'done' }),
      task({ id: 't5', title: 'Done2', milestone: 'm-1', board: 'done' }),
      task({ id: 't6', title: 'Done3', milestone: 'm-1', board: 'done' }),
    ]
    render(
      <BoardView
        tasks={tasks}
        onOpenTask={vi.fn()}
        onMove={vi.fn()}
        disabled={false}
        milestoneTitles={{ 'm-1': 'count-ms' }}
      />,
    )
    // Header bar shows per-column counts: todo=2, ready=1, in_progress=0, blocked=0, done=3
    // Find the count spans in the header (they're bare number text)
    const headerCounts = screen.getAllByText(/^(\d+)$/)
    const counts = headerCounts.map((el) => el.textContent)
    // The header has 5 column counts; find them among all numeric text
    expect(counts).toContain('2') // todo
    expect(counts).toContain('1') // ready
    expect(counts).toContain('3') // done
  })

  it('collapses milestone row on click, hiding task cards', () => {
    const tasks = [
      task({ id: 't1', title: 'Visible Card', milestone: 'm-1', board: 'todo' }),
    ]
    render(
      <BoardView
        tasks={tasks}
        onOpenTask={vi.fn()}
        onMove={vi.fn()}
        disabled={false}
        milestoneTitles={{ 'm-1': 'collapse-ms' }}
      />,
    )
    // Card is visible initially
    expect(screen.getByText('Visible Card')).toBeTruthy()
    // Click the milestone row toggle button to collapse (title appears in row label + card badge)
    const toggles = screen.getAllByRole('button', { name: /collapse-ms/i })
    fireEvent.click(toggles[0])
    // Card should be hidden after collapse
    expect(screen.queryByText('Visible Card')).toBeNull()
  })

  it('expands a collapsed milestone row on second click', () => {
    const tasks = [
      task({ id: 't1', title: 'Toggle Card', milestone: 'm-1', board: 'todo' }),
    ]
    render(
      <BoardView
        tasks={tasks}
        onOpenTask={vi.fn()}
        onMove={vi.fn()}
        disabled={false}
        milestoneTitles={{ 'm-1': 'expand-ms' }}
      />,
    )
    const toggles = screen.getAllByRole('button', { name: /expand-ms/i })
    // Collapse
    fireEvent.click(toggles[0])
    expect(screen.queryByText('Toggle Card')).toBeNull()
    // Expand
    fireEvent.click(toggles[0])
    expect(screen.getByText('Toggle Card')).toBeTruthy()
  })

  it('renders multiple milestone rows independently', () => {
    const tasks = [
      task({ id: 't1', title: 'Alpha Task', milestone: 'alpha', board: 'todo' }),
      task({ id: 't2', title: 'Beta Task', milestone: 'beta', board: 'done' }),
    ]
    render(
      <BoardView
        tasks={tasks}
        onOpenTask={vi.fn()}
        onMove={vi.fn()}
        disabled={false}
        milestoneTitles={{ alpha: 'Alpha MS', beta: 'Beta MS' }}
      />,
    )
    // Both milestone rows render (title appears in row label + card badge)
    expect(screen.getAllByText('Alpha MS').length).toBeGreaterThanOrEqual(1)
    expect(screen.getAllByText('Beta MS').length).toBeGreaterThanOrEqual(1)
    // Both tasks are visible (both rows expanded by default)
    expect(screen.getByText('Alpha Task')).toBeTruthy()
    expect(screen.getByText('Beta Task')).toBeTruthy()
  })

  it('hides column body but keeps row header when collapsed', () => {
    const tasks = [
      task({ id: 't1', title: 'Hidden Card', milestone: 'm-1', board: 'todo' }),
    ]
    render(
      <BoardView
        tasks={tasks}
        onOpenTask={vi.fn()}
        onMove={vi.fn()}
        disabled={false}
        milestoneTitles={{ 'm-1': 'header-ms' }}
      />,
    )
    const toggles = screen.getAllByRole('button', { name: /header-ms/i })
    fireEvent.click(toggles[0])
    // Row header still visible (title in row label; card badge is hidden with the body)
    expect(screen.getAllByText('header-ms').length).toBeGreaterThanOrEqual(1)
    // Card hidden
    expect(screen.queryByText('Hidden Card')).toBeNull()
  })
})
