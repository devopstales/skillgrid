import type { DocsContent } from './types'

// Frontmatter chips: status / author / updated + any other key/value.
export function FrontmatterChips({ content }: { content: DocsContent }) {
  const fm = content.frontmatter
  if (!fm || Object.keys(fm).length === 0) return null

  const order: [string, string][] = []
  for (const key of ['status', 'priority', 'author', 'assignee', 'type', 'milestone', 'id']) {
    if (fm[key]) order.push([key, fm[key]])
  }
  // include remaining keys not already shown
  for (const [k, v] of Object.entries(fm)) {
    if (!order.some(([kk]) => kk === k) && v) order.push([k, v])
  }
  if (order.length === 0) return null

  return (
    <div className="flex flex-wrap items-center gap-1.5">
      {order.map(([k, v]) => (
        <span
          key={k}
          className="inline-flex items-center gap-1 rounded-md border border-edge bg-black/20 px-2 py-0.5 text-[11px]"
        >
          <span className="text-zinc-500">{k}:</span>
          <span className="font-medium text-zinc-200">{v}</span>
        </span>
      ))}
    </div>
  )
}
