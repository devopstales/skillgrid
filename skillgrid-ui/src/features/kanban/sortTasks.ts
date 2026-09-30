import { BOARD_COLUMNS, type BoardColumn, type UnifiedTask } from './types'

export type SortKey =
  | 'id'
  | 'title'
  | 'status'
  | 'board'
  | 'priority'
  | 'assignee'
  | 'milestone'
export type SortDir = 'asc' | 'desc'

// Ascending puts the least urgent first: low < medium < high < none.
const PRIORITY_RANK: Record<string, number> = { low: 0, medium: 1, high: 2 }

// Board pipeline index: position in the board column order, unknown boards last.
function boardIndex(t: UnifiedTask): number {
  const i = BOARD_COLUMNS.indexOf(t.board)
  return i === -1 ? BOARD_COLUMNS.length : i
}

// Provider-native status → board column, so a task's `status` sorts by the
// same pipeline order as its board. Unknown statuses fall back to the board.
const STATUS_TO_BOARD: Record<string, BoardColumn> = {
  'needs-triage': 'todo',
  'ready-for-agent': 'ready',
  'ready-for-human': 'ready',
  'in-progress': 'in_progress',
  blocked: 'blocked',
  done: 'done',
  wontfix: 'done',
}
function statusIndex(t: UnifiedTask): number {
  const b = STATUS_TO_BOARD[t.status] ?? t.board
  const i = BOARD_COLUMNS.indexOf(b)
  return i === -1 ? BOARD_COLUMNS.length : i
}

// Sort value per key. Empties (undefined priority/assignee/milestone) get a
// sentinel that sorts last in ascending order; the comparator handles the
// descending flip explicitly so empties end up first in desc.
function sortValue(t: UnifiedTask, key: SortKey): { num: number; str: string } {
  switch (key) {
    case 'id':
      return { num: 0, str: t.id }
    case 'title':
      return { num: 0, str: t.title.toLowerCase() }
    case 'status':
      // Sort by the task's status mapped to its board pipeline position.
      return { num: statusIndex(t), str: '' }
    case 'board':
      return { num: boardIndex(t), str: '' }
    case 'priority': {
      // Empties/unknowns have "no rank" and sort before every real rank in
      // both directions (handled by the sentinel below, not by the flip).
      const p = t.priority?.toLowerCase()
      const rank = p === undefined ? -1 : (PRIORITY_RANK[p] ?? -1)
      return { num: rank, str: '' }
    }
    case 'assignee': {
      const a = t.assignees?.[0]
      return { num: a === undefined ? 1 : 0, str: (a ?? '').toLowerCase() }
    }
    case 'milestone': {
      const m = t.milestone
      return { num: m === undefined ? 1 : 0, str: (m ?? '').toLowerCase() }
    }
  }
}

// Compare two tasks for one key in ASCENDING order (empties last). The stable
// tiebreak is always the natural id sort, regardless of direction.
function compareAsc(a: UnifiedTask, b: UnifiedTask, key: SortKey): number {
  const av = sortValue(a, key)
  const bv = sortValue(b, key)
  if (av.num !== bv.num) return av.num - bv.num
  if (key !== 'status' && key !== 'board') {
    const c = av.str.localeCompare(bv.str, undefined, {
      numeric: key === 'id',
      sensitivity: 'base',
    })
    if (c !== 0) return c
  }
  return a.id.localeCompare(b.id, undefined, { numeric: true, sensitivity: 'base' })
}

// Merge sort: deterministic and stable, and safe in worker realms where the
// native Array#sort comparator can be ignored.
function mergeSort(
  arr: UnifiedTask[],
  key: SortKey,
  dir: SortDir,
): UnifiedTask[] {
  if (arr.length < 2) return arr
  const cmp = dir === 'asc'
    ? (a: UnifiedTask, b: UnifiedTask) => compareAsc(a, b, key)
    : (a: UnifiedTask, b: UnifiedTask) => {
        const av = sortValue(a, key)
        const bv = sortValue(b, key)
        // Desc: reverse the value comparison, empties (num=1) first, but keep
        // the natural id tiebreak ascending for determinism.
        if (av.num !== bv.num) return bv.num - av.num
        if (key !== 'status' && key !== 'board') {
          const c = bv.str.localeCompare(av.str, undefined, {
            numeric: key === 'id',
            sensitivity: 'base',
          })
          if (c !== 0) return c
        }
        return a.id.localeCompare(b.id, undefined, { numeric: true, sensitivity: 'base' })
      }
  const mid = arr.length >> 1
  const left = mergeSort(arr.slice(0, mid), key, dir)
  const right = mergeSort(arr.slice(mid), key, dir)
  const out: UnifiedTask[] = []
  let li = 0
  let ri = 0
  while (li < left.length && ri < right.length) {
    if (cmp(left[li], right[ri]) <= 0) out.push(left[li++])
    else out.push(right[ri++])
  }
  while (li < left.length) out.push(left[li++])
  while (ri < right.length) out.push(right[ri++])
  return out
}

// Pure: returns a new array, never mutates `tasks`.
export function sortTasks(
  tasks: UnifiedTask[],
  key: SortKey,
  dir: SortDir,
): UnifiedTask[] {
  return mergeSort([...tasks], key, dir)
}
