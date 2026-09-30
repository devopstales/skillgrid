import { describe, it, expect } from 'vitest'
import {
  BOARD_COLUMNS,
  COLUMN_LABEL,
  COLUMN_COLOR,
  type BoardColumn,
} from './types'

describe('board columns', () => {
  it('has five columns with ready at index 1', () => {
    expect(BOARD_COLUMNS).toEqual([
      'todo',
      'ready',
      'in_progress',
      'blocked',
      'done',
    ])
    expect(BOARD_COLUMNS.length).toBe(5)
    expect(BOARD_COLUMNS[1]).toBe('ready')
  })

  it('labels and colors the ready column', () => {
    expect(COLUMN_LABEL['ready' as BoardColumn]).toBe('Ready')
    expect(COLUMN_COLOR['ready' as BoardColumn]).toBe('bg-amber-500')
    expect(COLUMN_LABEL.todo).toBe('Todo')
    expect(COLUMN_LABEL.in_progress).toBe('In Progress')
    expect(COLUMN_LABEL.blocked).toBe('Blocked')
    expect(COLUMN_LABEL.done).toBe('Done')
    expect(COLUMN_COLOR.todo).toBe('bg-zinc-500')
    expect(COLUMN_COLOR.in_progress).toBe('bg-accent')
    expect(COLUMN_COLOR.blocked).toBe('bg-red-500')
    expect(COLUMN_COLOR.done).toBe('bg-emerald-500')
  })
})
