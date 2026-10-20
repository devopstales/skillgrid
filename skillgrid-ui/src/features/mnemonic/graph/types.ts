export interface GraphNode {
  id: number
  uid: string
  label: string
  type: string
  language: string
  path: string
  degree: number
  community: number
}

export interface GraphEdge {
  id: string
  source: number
  target: number
  type: string
  weight: number
  directed: boolean
  confidence?: string
  /** TICKET-12: the source line in the FROM file where the edge was observed
   *  (served by the stored edges.line; additive — old payloads omit it). */
  line?: number
}

export interface GraphResponse {
  nodes: GraphNode[]
  edges: GraphEdge[]
  truncated: boolean
  degraded: boolean
  files?: string[]
  project: string
}

export interface GraphNodesResponse {
  nodes: GraphNode[]
  truncated: boolean
  total: number
}
