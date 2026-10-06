import type { UnifiedTask } from './types'

export interface EpicGroup {
  parent: UnifiedTask
  children: UnifiedTask[]
}
export interface FlatGroup {
  task: UnifiedTask
}
export type ColumnGroup = EpicGroup | FlatGroup

export interface MilestoneGroup {
  name: string
  tasks: UnifiedTask[]
}

// Group all tasks by milestone across columns. Tasks without a milestone are
// grouped under the empty-string key, which renders as "Unassigned". Groups are
// sorted alphabetically; the unassigned group sorts last.
export function groupTasksByMilestone(tasks: UnifiedTask[]): MilestoneGroup[] {
  const map = new Map<string, UnifiedTask[]>()
  for (const t of tasks) {
    const key = t.milestone ?? ''
    const arr = map.get(key) ?? []
    arr.push(t)
    map.set(key, arr)
  }
  const groups: MilestoneGroup[] = []
  for (const [name, list] of map) {
    groups.push({ name, tasks: list })
  }
  groups.sort((a, b) => {
    if (!a.name) return 1
    if (!b.name) return -1
    return a.name.localeCompare(b.name)
  })
  return groups
}

// Split a column's tasks into display groups. Tasks with no children (and no
// parent, or a parent not in this column) render as flat cards; a task that has
// at least one child in the same column renders as an EpicGroup with those
// children. A task that is both a parent and someone else's child still
// appears once — as a child of its own parent — and never again as a top-level
// group (no double counting).
export function groupColumnTasks(tasks: UnifiedTask[]): ColumnGroup[] {
  const byParent = new Map<string, UnifiedTask[]>()
  for (const t of tasks) {
    if (t.parent) {
      const arr = byParent.get(t.parent) ?? []
      arr.push(t)
      byParent.set(t.parent, arr)
    }
  }
  const ids = new Set(tasks.map((t) => t.id))
  const used = new Set<string>()
  const groups: ColumnGroup[] = []
  for (const t of tasks) {
    const children = byParent.get(t.id)
    if (children && children.length > 0) {
      groups.push({ parent: t, children })
      used.add(t.id)
      for (const c of children) used.add(c.id)
    } else if (!t.parent || !ids.has(t.parent)) {
      // No parent, or parent not in this column → flat card.
      groups.push({ task: t })
      used.add(t.id)
    }
    // else: t is a child of a parent already emitted above → skip.
  }
  // Children render in original column order; parents keep column order too.
  void used
  return groups
}
