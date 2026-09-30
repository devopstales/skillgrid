import { describe, it, expect } from 'vitest'
import { groupColumnTasks } from './epicTree'
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
