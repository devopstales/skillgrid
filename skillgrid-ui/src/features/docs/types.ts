// Docs view types (mirror the /docs/* JSON the Go server emits).

export interface DocsTreeNode {
  dir: boolean
  name: string
  path: string
  title?: string
  status?: string
  updated?: string
  children?: DocsTreeNode[]
}

export interface DocsMermaidCfg {
  securityLevel: string
}

export interface DocsContent {
  path: string
  title?: string
  frontmatter?: Record<string, string>
  body: string
  relatedPlans?: string[]
  updatedAt?: string
  mermaid?: DocsMermaidCfg
}

export interface DocsSearchHit {
  path: string
  title?: string
  snippet?: string
  root?: string
}

export interface DocsTreeResponse {
  root: string
  nodes: DocsTreeNode[]
}

export interface DocsSearchResponse {
  q: string
  results: DocsSearchHit[]
}

export const DOCS_ROOTS = ['all', 'sdd', 'openspec', 'backlog', 'docs'] as const
export type DocsRoot = (typeof DOCS_ROOTS)[number]

export const ROOT_LABEL: Record<DocsRoot, string> = {
  all: 'All',
  sdd: 'SDD Plans',
  openspec: 'OpenSpec',
  backlog: 'Backlog',
  docs: 'Docs',
}
