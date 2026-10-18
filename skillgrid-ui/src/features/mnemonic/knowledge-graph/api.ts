import type { KnowledgeGraphResponse } from './types'
import { apiGet } from '../../../lib/api'

// fetchKnowledgeGraph loads the read-only knowledge concept graph for the
// current project. The server derives it per call from the current bundle
// (no store), so a re-fetch always reflects the latest Markdown.
export function fetchKnowledgeGraph(): Promise<KnowledgeGraphResponse> {
  return apiGet<KnowledgeGraphResponse>('/knowledge/graph')
}
