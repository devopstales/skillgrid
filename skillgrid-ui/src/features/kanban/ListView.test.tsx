import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { ListView } from './ListView'
import { type UnifiedTask } from './types'

function task(over: Partial<UnifiedTask> & Pick<UnifiedTask, 'id'>): UnifiedTask {
  return {
    title: 'Untitled',
    status: 'needs-triage',
    board: 'todo',
    provider: 'backlogmd',
    ...over,
  }
}

// Read the on-screen row order from the ID column (first cell of each data row).
function rowIds(container: HTMLElement): string[] {
  return Array.from(container.querySelectorAll('tbody tr td:first-child')).map(
    (td) => td.textContent ?? '',
  )
}

const byPriority = [
  task({ id: 'task-1', priority: 'low' }),
  task({ id: 'task-2', priority: 'high' }),
  task({ id: 'task-3' }),
  task({ id: 'task-4', priority: 'medium' }),
]

describe('ListView', () => {
  it('renders all column headers', () => {
    const { container } = render(<ListView tasks={[]} onOpenTask={vi.fn()} />)
    for (const label of [
      'ID',
      'Title',
      'Status',
      'Board',
      'Priority',
      'Assignee',
      'Milestone',
    ]) {
      expect(screen.getByRole('columnheader', { name: new RegExp(label, 'i') })).toBeTruthy()
    }
    expect(container.querySelector('table')).toBeTruthy()
  })

  it('clicking the Priority header sorts rows by priority rank (asc: none first)', () => {
    const { container } = render(<ListView tasks={byPriority} onOpenTask={vi.fn()} />)
    fireEvent.click(screen.getByRole('columnheader', { name: /priority/i }))
    // asc = lowest rank first: (none), low, medium, high
    expect(rowIds(container)).toEqual(['task-3', 'task-1', 'task-4', 'task-2'])
  })

  it('clicking the same header again flips to desc', () => {
    const { container } = render(<ListView tasks={byPriority} onOpenTask={vi.fn()} />)
    const header = screen.getByRole('columnheader', { name: /priority/i })
    fireEvent.click(header)
    expect(rowIds(container)).toEqual(['task-3', 'task-1', 'task-4', 'task-2'])
    fireEvent.click(header)
    // desc = highest rank first: high, medium, low, (none)
    expect(rowIds(container)).toEqual(['task-2', 'task-4', 'task-1', 'task-3'])
  })

  it('marks the active header with aria-sort and a direction arrow', () => {
    render(<ListView tasks={byPriority} onOpenTask={vi.fn()} />)
    const header = screen.getByRole('columnheader', { name: /priority/i })
    fireEvent.click(header)
    expect(header.getAttribute('aria-sort')).toBe('ascending')
    expect(header.textContent).toContain('▲')
    fireEvent.click(header)
    expect(header.getAttribute('aria-sort')).toBe('descending')
    expect(header.textContent).toContain('▼')
  })

  it('keeps a row click opening the task (row click, not header click)', () => {
    const onOpenTask = vi.fn()
    const { container } = render(<ListView tasks={byPriority} onOpenTask={onOpenTask} />)
    const firstRow = container.querySelector('tbody tr')
    expect(firstRow).toBeTruthy()
    fireEvent.click(firstRow!)
    expect(onOpenTask).toHaveBeenCalledWith('task-1')
  })

  it('shows the milestone title from milestoneTitles, not the raw ID', () => {
    const { container } = render(
      <ListView
        tasks={[task({ id: 'task-1', milestone: 'm-1' })]}
        onOpenTask={vi.fn()}
        milestoneTitles={{ 'm-1': 'Kanban milestones' }}
      />,
    )
    const row = container.querySelector('tbody tr')!
    expect(row.textContent).toContain('Kanban milestones')
    // The raw ID must not be shown as a standalone cell (it's the title now).
    expect(row.textContent).not.toContain('m-1')
  })

  it('falls back to the raw milestone ID when no title matches', () => {
    const { container } = render(
      <ListView
        tasks={[task({ id: 'task-1', milestone: 'm-1' })]}
        onOpenTask={vi.fn()}
        milestoneTitles={{ 'm-2': 'Other' }}
      />,
    )
    const row = container.querySelector('tbody tr')!
    expect(row.textContent).toContain('m-1')
  })
})
