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
