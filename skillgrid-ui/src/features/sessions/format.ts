// Cost is null when the session's model has no pricing entry.
export function formatCost(v?: number | null): string {
  if (v === null || v === undefined) return 'n/a'
  return v < 0.01 ? `$${v.toFixed(4)}` : `$${v.toFixed(2)}`
}

export function formatTokens(n?: number): string {
  if (!n) return '0'
  return n >= 1_000_000 ? `${(n / 1_000_000).toFixed(1)}M` : n >= 1000 ? `${(n / 1000).toFixed(1)}k` : String(n)
}
