// Phase 2 Kanban — shared types mirroring the Go tracker.UnifiedTask DTO.

export const BOARD_COLUMNS = ['todo', 'in_progress', 'blocked', 'done'] as const
export type BoardColumn = (typeof BOARD_COLUMNS)[number]

export const COLUMN_LABEL: Record<BoardColumn, string> = {
  todo: 'Todo',
  in_progress: 'In Progress',
  blocked: 'Blocked',
  done: 'Done',
}

export const COLUMN_COLOR: Record<BoardColumn, string> = {
  todo: 'bg-zinc-500',
  in_progress: 'bg-accent',
  blocked: 'bg-red-500',
  done: 'bg-emerald-500',
}

// The four tracker providers the dashboard can render.
export const PROVIDERS = ['backlogmd', 'github', 'gitlab', 'jira'] as const
export type ProviderName = (typeof PROVIDERS)[number]

export const PROVIDER_LABEL: Record<ProviderName, string> = {
  backlogmd: 'Backlog.md',
  github: 'GitHub',
  gitlab: 'GitLab',
  jira: 'Jira',
}

export interface UnifiedTask {
  id: string
  title: string
  status: string
  status_detail?: string
  description?: string
  type?: string
  priority?: string
  assignees?: string[]
  labels?: string[]
  doc_refs?: string[]
  board: BoardColumn
  dependencies?: string[]
  milestone?: string
  parent?: string
  due_date?: string
  ac_completed?: number
  ac_total?: number
  is_ready?: boolean | null
  created_at?: string
  updated_at?: string
  provider: string
}

export interface TrackerDeps {
  task_id: string
  deps_in: string[]
  deps_out: string[]
}

export interface ProvidersInfo {
  provider: string
  connected: boolean
  reason?: string
  cli?: string
}

export interface TaskListResponse {
  tasks: UnifiedTask[]
  provider: string
}

// A provider's board load state.
export type LoadState =
  | { status: 'loading' }
  | { status: 'ready'; tasks: UnifiedTask[]; provider: string }
  | { status: 'degraded'; provider: string; reason: string; httpStatus: number }
  | { status: 'error'; message: string }

export interface KanbanFilters {
  query: string
  assignee: string
  label: string
  milestone: string
  priority: string
}

export const EMPTY_FILTERS: KanbanFilters = {
  query: '',
  assignee: '',
  label: '',
  milestone: '',
  priority: '',
}
