// Community coloring. The backend already computes Leiden communities (server
// side, via the community package) and tags each node with `community`. We map
// community ids to a stable categorical palette for rendering + the legend.
//
// community=-1 means "no community data" (the 005/008 soft dep is absent);
// those nodes get the neutral color.

export const COMMUNITY_PALETTE = [
  '#6366f1', // indigo
  '#22c55e', // green
  '#f59e0b', // amber
  '#ef4444', // red
  '#06b6d4', // cyan
  '#a855f7', // purple
  '#ec4899', // pink
  '#14b8a6', // teal
  '#eab308', // yellow
  '#3b82f6', // blue
  '#f97316', // orange
  '#84cc16', // lime
]

export const NEUTRAL_COLOR = '#94a3b8' // slate-400 (no community data)

// communityColor returns the palette color for a community id.
export function communityColor(community: number): string {
  if (community < 0) return NEUTRAL_COLOR
  return COMMUNITY_PALETTE[community % COMMUNITY_PALETTE.length]
}

// communityStats counts members per community (for the legend).
export function communityStats(
  nodes: { community: number }[],
): Map<number, number> {
  const m = new Map<number, number>()
  for (const n of nodes) {
    m.set(n.community, (m.get(n.community) ?? 0) + 1)
  }
  return m
}
