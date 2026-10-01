// Overview landing page types (mirror the JSON the Go server emits for the
// KPI endpoints: /activity/stats, /code/status, /activity/events) plus the
// parsed shape of .skillgrid/state.yaml for the pipeline KPI.

export interface ActivityStats {
  project: string
  total: number
  byType: Record<string, number>
  activeSessions: number
}

export interface CodeStatus {
  file_count: number
  chunk_count: number
  last_indexed: string
  stale: boolean
}

export interface ActivityEvent {
  id: number
  ts: string
  type: string
  source: string
  actor?: string | null
  severity: string
  summary: string
  topicKey?: string | null
  sessionId: string
  relatedIds: number[]
}

export interface ActivityEventsResponse {
  project: string
  events: ActivityEvent[]
  limit: number
}

// The slice of .skillgrid/state.yaml the Overview page surfaces.
export interface PipelineState {
  current_phase: string
  current_change: string
  status: string
  completed_changes: number
}
