import type { AuditEntry } from './api'

// AuditTab — the hash-chained memory change log (observation_versions) with a
// chain-validity badge. Demoted from the default Sessions view to a secondary
// tab in sessions-activity-unification (kept for verification, not day-to-day).
export function AuditTab({
  entries,
  chainValid,
}: {
  entries: AuditEntry[]
  chainValid: boolean
}) {
  return (
    <div className="flex h-full flex-col overflow-y-auto">
      <div className="flex items-center gap-2 border-b border-edge px-6 py-2">
        <h2 className="text-xs font-semibold uppercase tracking-wide text-zinc-500">Audit trail</h2>
        <span className="text-xs text-zinc-600">
          {entries.length} entries · hash-chained · {chainValid ? 'valid' : 'broken'}
        </span>
        <span className="ml-auto">
          <ChainBadge valid={chainValid} />
        </span>
      </div>
      {entries.length === 0 ? (
        <div className="p-8 text-center text-sm text-zinc-500">
          No memory changes recorded yet.
        </div>
      ) : (
        <AuditTable entries={entries} />
      )}
    </div>
  )
}

function ChainBadge({ valid }: { valid: boolean }) {
  return valid ? (
    <span className="rounded-md border border-emerald-500/30 bg-emerald-500/10 px-2 py-0.5 text-emerald-300">
      chain valid
    </span>
  ) : (
    <span className="rounded-md border border-red-500/30 bg-red-500/10 px-2 py-0.5 text-red-300">
      chain broken
    </span>
  )
}

function AuditTable({ entries }: { entries: AuditEntry[] }) {
  return (
    <table className="w-full text-left text-xs">
      <thead className="sticky top-0 bg-background">
        <tr className="text-[10px] uppercase tracking-wide text-zinc-600">
          <th className="px-4 py-2">seq</th>
          <th className="px-4 py-2">obs</th>
          <th className="px-4 py-2">rev</th>
          <th className="px-4 py-2">time</th>
          <th className="px-4 py-2">hash</th>
        </tr>
      </thead>
      <tbody className="divide-y divide-edge">
        {entries.map((e) => (
          <tr key={e.seq} className="font-mono text-[11px] text-zinc-400 hover:bg-card">
            <td className="px-4 py-1.5 text-zinc-600">{e.seq}</td>
            <td className="px-4 py-1.5">{e.observation_id}</td>
            <td className="px-4 py-1.5">{e.revision}</td>
            <td className="px-4 py-1.5">{e.created_at}</td>
            <td className="px-4 py-1.5">
              <span className="text-emerald-400/80">{e.hash.slice(0, 10)}</span>
              <span className="text-zinc-700">…</span>
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}
