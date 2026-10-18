// Knowledge concept graph payload — served by GET /knowledge/graph, derived
// per call by the server from the project's current knowledge bundle (never
// stored). One .md concept file per node; catalogs (index/log/reverse-index)
// are not nodes.

export interface KnowledgeConceptNode {
  id: string
  title: string
  type: string
  description: string
  tags: string[]
  status: string
  orphan: boolean
  deprecated: boolean
}

export interface KnowledgeConceptEdge {
  source: string
  target: string
}

export interface KnowledgeGraphResponse {
  nodes: KnowledgeConceptNode[]
  edges: KnowledgeConceptEdge[]
  backlinks: Record<string, string[]>
}
