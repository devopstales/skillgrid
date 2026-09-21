import type { Checkpoint, HandoffRef } from './api'

// HandoffsPane — checkpoints + handoff refs (session/team handoffs). Moved from
// the former Handoff Hub `handoffs` tab (sessions-activity-unification).
export function HandoffsPane({
  checkpoints,
  refs,
}: {
  checkpoints: Checkpoint[]
  refs: HandoffRef[]
}) {
  return (
    <div className="flex flex-col gap-4 overflow-y-auto p-6">
      <section>
        <h2 className="mb-2 text-sm font-semibold text-zinc-200">Checkpoints ({checkpoints.length})</h2>
        <div className="overflow-hidden rounded-lg border border-edge">
          <table className="w-full text-sm">
            <thead className="bg-card text-left text-xs uppercase text-zinc-500">
              <tr>
                <th className="px-3 py-2">Name</th>
                <th className="px-3 py-2">Status</th>
                <th className="px-3 py-2">Commit</th>
                <th className="px-3 py-2">Spec</th>
                <th className="px-3 py-2">Evidence</th>
              </tr>
            </thead>
            <tbody>
              {checkpoints.map((c) => (
                <tr key={c.id} className="border-t border-edge">
                  <td className="px-3 py-2 font-mono text-xs text-zinc-300">{c.name}</td>
                  <td className="px-3 py-2">
                    <StatusPill status={c.status} />
                  </td>
                  <td className="px-3 py-2 font-mono text-xs text-zinc-400">{c.commit.slice(0, 7)}</td>
                  <td className="px-3 py-2 text-xs text-zinc-500">{c.specDir || '—'}</td>
                  <td className="px-3 py-2 text-xs text-zinc-500">{c.evidence || '—'}</td>
                </tr>
              ))}
              {checkpoints.length === 0 && (
                <tr>
                  <td colSpan={5} className="px-3 py-6 text-center text-zinc-600">
                    No checkpoints placed.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </section>

      <section>
        <h2 className="mb-2 text-sm font-semibold text-zinc-200">Handoff refs ({refs.length})</h2>
        <div className="overflow-hidden rounded-lg border border-edge">
          <table className="w-full text-sm">
            <thead className="bg-card text-left text-xs uppercase text-zinc-500">
              <tr>
                <th className="px-3 py-2">Handoff</th>
                <th className="px-3 py-2">Type</th>
                <th className="px-3 py-2">Commits</th>
                <th className="px-3 py-2">Spec</th>
              </tr>
            </thead>
            <tbody>
              {refs.map((r) => (
                <tr key={r.handoffId + r.handoffType} className="border-t border-edge">
                  <td className="px-3 py-2 font-mono text-xs text-zinc-300">{r.handoffId}</td>
                  <td className="px-3 py-2 text-xs text-zinc-400">{r.handoffType}</td>
                  <td className="px-3 py-2 font-mono text-xs text-zinc-400">
                    {(r.fromCommit || '…').slice(0, 7)}..{(r.toCommit || '…').slice(0, 7)}
                  </td>
                  <td className="px-3 py-2 text-xs text-zinc-500">{r.specDir || '—'}</td>
                </tr>
              ))}
              {refs.length === 0 && (
                <tr>
                  <td colSpan={4} className="px-3 py-6 text-center text-zinc-600">
                    No handoff refs yet (session/team handoffs populate this).
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  )
}

function StatusPill({ status }: { status: Checkpoint['status'] }) {
  const color =
    status === 'verified'
      ? 'bg-emerald-500/15 text-emerald-300'
      : status === 'stale'
        ? 'bg-amber-500/15 text-amber-300'
        : status === 'archived'
          ? 'bg-zinc-500/15 text-zinc-400'
          : 'bg-blue-500/15 text-blue-300'
  return <span className={`rounded px-1.5 py-0.5 text-xs ${color}`}>{status}</span>
}
