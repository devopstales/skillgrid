import { describe, it, expect } from 'vitest'
import { sortTasks, type SortKey, type SortDir } from './sortTasks'
import { type UnifiedTask } from './types'

function t(over: Partial<UnifiedTask> & Pick<UnifiedTask, 'id'>): UnifiedTask {
  return {
    title: 'Untitled',
    status: 'needs-triage',
    board: 'todo',
    provider: 'backlogmd',
    ...over,
  }
}

function ids(tasks: UnifiedTask[]): string[] {
  return tasks.map((x) => x.id)
}

function order(tasks: UnifiedTask[], key: SortKey, dir: SortDir): string[] {
  return ids(sortTasks(tasks, key, dir))
}

describe('sortTasks', () => {
  it('sorts id naturally so task-2 < task-10 (asc)', () => {
    const tasks = [t({ id: 'task-10' }), t({ id: 'task-2' }), t({ id: 'task-1' })]
    expect(order(tasks, 'id', 'asc')).toEqual(['task-1', 'task-2', 'task-10'])
  })

  it('sorts board by column index, not alphabetically (asc)', () => {
    const tasks = [
      t({ id: 'b1', board: 'blocked' }),
      t({ id: 'b2', board: 'todo' }),
      t({ id: 'b3', board: 'done' }),
      t({ id: 'b4', board: 'ready' }),
      t({ id: 'b5', board: 'in_progress' }),
    ]
    expect(order(tasks, 'board', 'asc')).toEqual(['b2', 'b4', 'b5', 'b1', 'b3'])
  })

  it('sorts priority by rank high>medium>low>none (asc = lowest rank first)', () => {
    const tasks = [
      t({ id: 'p-low', priority: 'low' }),
      t({ id: 'p-high', priority: 'high' }),
      t({ id: 'p-med', priority: 'medium' }),
      t({ id: 'p-none' }),
    ]
    expect(order(tasks, 'priority', 'asc')).toEqual(['p-none', 'p-low', 'p-med', 'p-high'])
  })

  it('puts unassigned tasks last on assignee asc', () => {
    const tasks = [
      t({ id: 'a-zed', assignees: ['zed'] }),
      t({ id: 'a-none' }),
      t({ id: 'a-alice', assignees: ['alice'] }),
    ]
    expect(order(tasks, 'assignee', 'asc')).toEqual(['a-alice', 'a-zed', 'a-none'])
  })

  it('sorts title case-insensitively (asc)', () => {
    const tasks = [
      t({ id: 't-b', title: 'beta' }),
      t({ id: 't-a', title: 'Alpha' }),
      t({ id: 't-c', title: 'ALPHA-2' }),
    ]
    expect(order(tasks, 'title', 'asc')).toEqual(['t-a', 't-c', 't-b'])
  })

  it('sorts status by board pipeline index, not raw string (asc)', () => {
    const tasks = [
      t({ id: 'st-blocked', status: 'blocked' }),
      t({ id: 'st-done', status: 'done' }),
      t({ id: 'st-ready', status: 'ready-for-agent' }),
      t({ id: 'st-triage', status: 'needs-triage' }),
    ]
    expect(order(tasks, 'status', 'asc')).toEqual(['st-triage', 'st-ready', 'st-blocked', 'st-done'])
  })

  it('triangulates: equal keys tiebreak by natural id order', () => {
    const tasks = [
      t({ id: 's-10', title: 'Same' }),
      t({ id: 's-2', title: 'Same' }),
      t({ id: 's-1', title: 'Same' }),
    ]
    expect(order(tasks, 'title', 'asc')).toEqual(['s-1', 's-2', 's-10'])
  })

  it('returns an empty array for empty input', () => {
    expect(sortTasks([], 'id', 'asc')).toEqual([])
  })

  it('triangulates: desc reverses id natural order', () => {
    const tasks = [t({ id: 'task-10' }), t({ id: 'task-2' }), t({ id: 'task-1' })]
    expect(order(tasks, 'id', 'desc')).toEqual(['task-10', 'task-2', 'task-1'])
  })

  it('triangulates: desc reverses board column order', () => {
    const tasks = [
      t({ id: 'b1', board: 'blocked' }),
      t({ id: 'b2', board: 'todo' }),
      t({ id: 'b3', board: 'done' }),
      t({ id: 'b4', board: 'ready' }),
      t({ id: 'b5', board: 'in_progress' }),
    ]
    expect(order(tasks, 'board', 'desc')).toEqual(['b3', 'b1', 'b5', 'b4', 'b2'])
  })

  it('triangulates: desc reverses priority rank', () => {
    const tasks = [
      t({ id: 'p-low', priority: 'low' }),
      t({ id: 'p-high', priority: 'high' }),
      t({ id: 'p-med', priority: 'medium' }),
      t({ id: 'p-none' }),
    ]
    expect(order(tasks, 'priority', 'desc')).toEqual(['p-high', 'p-med', 'p-low', 'p-none'])
  })

  it('triangulates: assignee desc puts unassigned first', () => {
    const tasks = [
      t({ id: 'a-zed', assignees: ['zed'] }),
      t({ id: 'a-none' }),
      t({ id: 'a-alice', assignees: ['alice'] }),
    ]
    expect(order(tasks, 'assignee', 'desc')).toEqual(['a-none', 'a-zed', 'a-alice'])
  })

  it('does not mutate the input array', () => {
    const tasks = [t({ id: 'task-10' }), t({ id: 'task-2' }), t({ id: 'task-1' })]
    const snapshot = ids(tasks)
    sortTasks(tasks, 'id', 'asc')
    expect(ids(tasks)).toEqual(snapshot)
  })

  it('sorts milestone alphabetically (asc)', () => {
    const tasks = [
      t({ id: 'm-c', milestone: 'charlie' }),
      t({ id: 'm-a', milestone: 'alpha' }),
      t({ id: 'm-b', milestone: 'bravo' }),
    ]
    expect(order(tasks, 'milestone', 'asc')).toEqual(['m-a', 'm-b', 'm-c'])
  })

  it('puts tasks without a milestone last on milestone asc', () => {
    const tasks = [
      t({ id: 'm-none' }),
      t({ id: 'm-a', milestone: 'alpha' }),
      t({ id: 'm-b', milestone: 'bravo' }),
    ]
    expect(order(tasks, 'milestone', 'asc')).toEqual(['m-a', 'm-b', 'm-none'])
  })

  it('reverses milestone order on desc', () => {
    const tasks = [
      t({ id: 'm-a', milestone: 'alpha' }),
      t({ id: 'm-b', milestone: 'bravo' }),
      t({ id: 'm-c', milestone: 'charlie' }),
    ]
    expect(order(tasks, 'milestone', 'desc')).toEqual(['m-c', 'm-b', 'm-a'])
  })

  it('puts tasks without a milestone first on milestone desc', () => {
    const tasks = [
      t({ id: 'm-a', milestone: 'alpha' }),
      t({ id: 'm-none' }),
      t({ id: 'm-b', milestone: 'bravo' }),
    ]
    expect(order(tasks, 'milestone', 'desc')).toEqual(['m-none', 'm-b', 'm-a'])
  })
})
