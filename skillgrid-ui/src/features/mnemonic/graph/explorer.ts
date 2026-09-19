// buildPathTree builds a nested tree from a flat list of file paths (split on
// '/'). Pure + deterministic so it's unit-testable. Duplicate paths are
// deduped. `maxDepth` caps nesting (deeper segments fold into the deepest
// directory node so no file is lost).
//
// A node is either a directory (has `children`) or a file leaf (has `file`
// set to the full path).
export interface PathTreeNode {
  name: string
  // Full path when this is a file leaf; undefined for directories.
  file?: string
  // Present only for directories.
  children?: PathTreeNode[]
}

export function buildPathTree(paths: string[], maxDepth = 8): PathTreeNode {
  const root: PathTreeNode = { name: '' }
  root.children = []
  const seen = new Set<string>()

  for (const raw of paths) {
    const p = raw.replace(/^\/+/, '') // strip a leading slash
    if (!p || seen.has(p)) continue
    seen.add(p)
    const parts = p.split('/')
    // If the path is deeper than maxDepth, fold the overflow into the last
    // directory segment (its name becomes the tail) so the file isn't dropped.
    const segs = parts.length > maxDepth
      ? [...parts.slice(0, maxDepth - 1), parts.slice(maxDepth - 1).join('/')]
      : parts

    let cur = root
    for (let i = 0; i < segs.length; i++) {
      const seg = segs[i]
      const isLast = i === segs.length - 1
      if (isLast) {
        // file leaf
        cur.children!.push({ name: seg, file: p })
      } else {
        // directory — find or create
        let dir = cur.children!.find((c) => c.name === seg && !c.file)
        if (!dir) {
          dir = { name: seg, children: [] }
          cur.children!.push(dir)
        }
        cur = dir
      }
    }
  }

  // sort: directories first, then files, both alphabetical — stable, scannable
  const sortNode = (n: PathTreeNode) => {
    if (!n.children) return
    n.children.sort((a, b) => {
      const aDir = !!a.children
      const bDir = !!b.children
      if (aDir !== bDir) return aDir ? -1 : 1
      return a.name.localeCompare(b.name)
    })
    n.children.forEach(sortNode)
  }
  sortNode(root)
  return root
}

// agentForPath returns which agent a config file belongs to (client-side
// path-prefix heuristic), or null for a plain source file. There is NO
// per-agent dimension on the backend, so this is an approximation:
//   - .cursor/ or *.cursorrules            → 'cursor'
//   - kilo/ or plugins/kilo                → 'kilo'
//   - .opencode/ or plugins/opencode       → 'opencode'
//   - shared agent config (AGENTS.md, mcp.yaml, indexing.yaml, tools.yaml,
//     skills-lock.json, config.d/)          → 'all'
//   - anything else                          → null
export function agentForPath(p: string): string | null {
  const path = p.replace(/^\/+/, '')
  const segs = path.split('/')
  const base = segs[segs.length - 1]
  const has = (re: RegExp) => segs.some((s) => re.test(s)) || re.test(base)

  if (has(/^\.cursor$/) || /\.cursorrules$/.test(base)) return 'cursor'
  if (has(/^kilo$/) || (segs.includes('plugins') && has(/^kilo$/))) return 'kilo'
  if (has(/^\.opencode$/) || (segs.includes('plugins') && has(/^opencode$/))) return 'opencode'
  if (
    base === 'AGENTS.md' ||
    base === 'mcp.yaml' ||
    base === 'indexing.yaml' ||
    base === 'tools.yaml' ||
    base === 'skills-lock.json' ||
    has(/^config\.d$/)
  )
    return 'all'
  return null
}
