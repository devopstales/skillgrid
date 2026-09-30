import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { TaskCard } from './TaskCard'
import { type UnifiedTask } from './types'

function task(over: Partial<UnifiedTask>): UnifiedTask {
  return {
    id: 'task-7',
    title: 'Wire up the tracker board',
    status: 'ready-for-agent',
    board: 'todo',
    provider: 'backlogmd',
    ...over,
  }
}

describe('TaskCard', () => {
  it('renders the real AC progress 2/5 when ac_completed=2, ac_total=5', () => {
    render(<TaskCard task={task({ ac_completed: 2, ac_total: 5 })} onClick={vi.fn()} />)
    expect(screen.getByText(/2\/5/)).toBeTruthy()
  })

  it('hides the AC badge when ac_total is 0/absent', () => {
    render(<TaskCard task={task({ ac_total: 0 })} onClick={vi.fn()} />)
    expect(screen.queryByText(/AC/)).toBeNull()
  })

  it('opens the task on click', () => {
    const onClick = vi.fn()
    render(<TaskCard task={task({})} onClick={onClick} />)
    screen.getByRole('button').click()
    expect(onClick).toHaveBeenCalledTimes(1)
  })
})
