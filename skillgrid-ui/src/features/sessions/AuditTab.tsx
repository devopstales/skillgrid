import type { AuditEntry } from './api'

// AuditTab — the hash-chained memory change log (observation_versions) with a
// chain-validity badge. Demoted from the default Sessions view to a secondary
// tab in sessions-activity-unification (kept for verification, not day-to-day).
export function AuditTab({
  entries,
  chainValid,
  error,
}: {
  entries: AuditEntry[]
  chainValid: boolean
  error?: string
}) {
  return (
    <div className="flex h-full flex-col overflow-y-auto">
      {error && (
        <div className="border-b border-danger-ink bg-danger/10 px-6 py-2 font-mono text-[12px] text-danger">
          {error}
        </div>
      )}
      <div className="flex items-center gap-2 border-b border-edge-soft px-6 py-2 font-mono">
        <h2 className="text-[11px] font-semibold uppercase tracking-[0.12em] text-ink-6">Audit trail</h2>
        <span className="text-[12px] text-ink-6">
          {entries.length} entries · hash-chained · {chainValid ? 'valid' : 'broken'}
        </span>
        <span className="ml-auto">
          <ChainBadge valid={chainValid} />
        </span>
      </div>
      {entries.length === 0 ? (
        <div className="p-8 text-center font-mono text-[13px] text-ink-5">
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
    <span className="rounded border border-accent/40 bg-accent/10 px-2 py-0.5 font-mono text-[11px] text-accent">
      chain valid
    </span>
  ) : (
    <span className="rounded border border-danger-ink bg-danger/10 px-2 py-0.5 font-mono text-[11px] text-danger">
      chain broken
    </span>
  )
}

function AuditTable({ entries }: { entries: AuditEntry[] }) {
  return (
    <table className="w-full text-left text-[12px]">
      <thead className="sticky top-0 bg-bg">
        <tr className="text-[10px] uppercase tracking-[0.1em] text-ink-6">
          <th className="px-4 py-2 font-medium">seq</th>
          <th className="px-4 py-2 font-medium">obs</th>
          <th className="px-4 py-2 font-medium">rev</th>
          <th className="px-4 py-2 font-medium">time</th>
          <th className="px-4 py-2 font-medium">hash</th>
        </tr>
      </thead>
      <tbody className="divide-y divide-edge">
        {entries.map((e) => (
          <tr key={e.seq} className="font-mono text-[11px] text-ink-4 hover:bg-card/60">
            <td className="px-4 py-1.5 text-ink-6">{e.seq}</td>
            <td className="px-4 py-1.5">{e.observation_id}</td>
            <td className="px-4 py-1.5">{e.revision}</td>
            <td className="px-4 py-1.5">{e.created_at}</td>
            <td className="px-4 py-1.5">
              <span className="text-accent/80">{e.hash.slice(0, 10)}</span>
              <span className="text-ink-6">…</span>
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}
