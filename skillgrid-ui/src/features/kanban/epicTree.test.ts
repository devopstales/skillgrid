import { describe, it, expect } from 'vitest'
import { groupColumnTasks, groupTasksByMilestone } from './epicTree'
import { type UnifiedTask } from './types'

function t(over: Partial<UnifiedTask> & Pick<UnifiedTask, 'id'>): UnifiedTask {
  return { title: over.title ?? over.id, status: 'needs-triage', board: 'todo', provider: 'backlogmd', ...over }
}

describe('groupColumnTasks', () => {
  it('groups children under their parent and keeps standalone tasks flat', () => {
    const groups = groupColumnTasks([
      t({ id: 'c1', parent: 'epic' }),
      t({ id: 'epic' }),
      t({ id: 'c2', parent: 'epic' }),
      t({ id: 'solo' }),
    ])
    const epic = groups.find((g) => 'parent' in g) as { parent: UnifiedTask; children: UnifiedTask[] }
    expect(epic).toBeTruthy()
    expect(epic.parent.id).toBe('epic')
    expect(epic.children.map((c) => c.id)).toEqual(['c1', 'c2'])
    const flats = groups.filter((g) => 'task' in g).map((g) => (g as { task: UnifiedTask }).task.id)
    expect(flats).toEqual(['solo'])
  })

  it('keeps a mid-level task a child of its parent, not a top-level group', () => {
    const groups = groupColumnTasks([
      t({ id: 'epic-a' }),
      t({ id: 'mid', parent: 'epic-a' }),
      t({ id: 'leaf', parent: 'mid' }),
    ])
    const a = groups.find((g) => 'parent' in g && (g as { parent: UnifiedTask }).parent.id === 'epic-a') as {
      parent: UnifiedTask
      children: UnifiedTask[]
    }
    expect(a.children.map((c) => c.id)).toEqual(['mid'])
    const mid = groups.find((g) => 'parent' in g && (g as { parent: UnifiedTask }).parent.id === 'mid') as {
      parent: UnifiedTask
      children: UnifiedTask[]
    }
    expect(mid.children.map((c) => c.id)).toEqual(['leaf'])
    // epic-a must not be duplicated as a flat card.
    const flats = groups.filter((g) => 'task' in g).map((g) => (g as { task: UnifiedTask }).task.id)
    expect(flats).toEqual([])
  })

  it('treats a task whose parent is in another column as flat', () => {
    const groups = groupColumnTasks([
      t({ id: 'orphan', parent: 'elsewhere' }),
    ])
    expect(groups).toHaveLength(1)
    expect('task' in groups[0]).toBe(true)
  })
})

describe('groupTasksByMilestone', () => {
  it('groups tasks by milestone value', () => {
    const groups = groupTasksByMilestone([
      t({ id: 't1', milestone: 'm-1' }),
      t({ id: 't2', milestone: 'm-1' }),
      t({ id: 't3', milestone: 'm-2' }),
    ])
    expect(groups).toHaveLength(2)
    const m1 = groups.find((g) => g.id === 'm-1')
    expect(m1).toBeTruthy()
    expect(m1!.tasks.map((x) => x.id)).toEqual(['t1', 't2'])
    const m2 = groups.find((g) => g.id === 'm-2')
    expect(m2!.tasks.map((x) => x.id)).toEqual(['t3'])
  })

  it('groups tasks without a milestone under empty string', () => {
    const groups = groupTasksByMilestone([
      t({ id: 't1' }),
      t({ id: 't2' }),
    ])
    expect(groups).toHaveLength(1)
    expect(groups[0].id).toBe('')
    expect(groups[0].title).toBe('')
    expect(groups[0].tasks).toHaveLength(2)
  })

  it('sorts alphabetically by title, unassigned last', () => {
    const groups = groupTasksByMilestone([
      t({ id: 't1', milestone: 'zeta' }),
      t({ id: 't2', milestone: 'alpha' }),
      t({ id: 't3' }),
      t({ id: 't4', milestone: 'mid' }),
    ])
    expect(groups.map((g) => g.id)).toEqual(['alpha', 'mid', 'zeta', ''])
  })

  it('resolves milestone titles from titleMap', () => {
    const groups = groupTasksByMilestone(
      [t({ id: 't1', milestone: 'm-1' })],
      { 'm-1': 'My Milestone' },
    )
    expect(groups[0].id).toBe('m-1')
    expect(groups[0].title).toBe('My Milestone')
  })

  it('falls back to raw ID when no title matches', () => {
    const groups = groupTasksByMilestone(
      [t({ id: 't1', milestone: 'unknown' })],
      {},
    )
    expect(groups[0].title).toBe('unknown')
  })

  it('sorts by resolved title, not raw ID', () => {
    const groups = groupTasksByMilestone(
      [
        t({ id: 't1', milestone: 'z' }),
        t({ id: 't2', milestone: 'a' }),
      ],
      { z: 'Alpha Title', a: 'Zeta Title' },
    )
    // Sorted by title: "Alpha Title" < "Zeta Title"
    expect(groups.map((g) => g.id)).toEqual(['z', 'a'])
  })

  it('handles a single milestone', () => {
    const groups = groupTasksByMilestone([
      t({ id: 't1', milestone: 'm-1' }),
      t({ id: 't2', milestone: 'm-1' }),
    ])
    expect(groups).toHaveLength(1)
    expect(groups[0].id).toBe('m-1')
    expect(groups[0].tasks).toHaveLength(2)
  })

  it('handles all-unassigned', () => {
    const groups = groupTasksByMilestone([
      t({ id: 't1' }),
      t({ id: 't2' }),
      t({ id: 't3' }),
    ])
    expect(groups).toHaveLength(1)
    expect(groups[0].id).toBe('')
    expect(groups[0].tasks).toHaveLength(3)
  })

  it('returns empty groups for empty task list', () => {
    const groups = groupTasksByMilestone([])
    expect(groups).toHaveLength(0)
  })
})
