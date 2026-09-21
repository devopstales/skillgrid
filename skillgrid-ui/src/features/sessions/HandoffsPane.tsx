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
        <h2 className="mb-2 text-sm font-semibold">Checkpoints ({checkpoints.length})</h2>
        <div className="overflow-hidden rounded-lg border">
          <table className="w-full text-sm">
            <thead className="bg-muted/50 text-left text-xs uppercase text-muted-foreground">
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
                <tr key={c.id} className="border-t">
                  <td className="px-3 py-2 font-mono text-xs">{c.name}</td>
                  <td className="px-3 py-2">
                    <StatusPill status={c.status} />
                  </td>
                  <td className="px-3 py-2 font-mono text-xs">{c.commit.slice(0, 7)}</td>
                  <td className="px-3 py-2 text-xs text-muted-foreground">{c.specDir || '—'}</td>
                  <td className="px-3 py-2 text-xs text-muted-foreground">{c.evidence || '—'}</td>
                </tr>
              ))}
              {checkpoints.length === 0 && (
                <tr>
                  <td colSpan={5} className="px-3 py-6 text-center text-muted-foreground">
                    No checkpoints placed.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </section>

      <section>
        <h2 className="mb-2 text-sm font-semibold">Handoff refs ({refs.length})</h2>
        <div className="overflow-hidden rounded-lg border">
          <table className="w-full text-sm">
            <thead className="bg-muted/50 text-left text-xs uppercase text-muted-foreground">
              <tr>
                <th className="px-3 py-2">Handoff</th>
                <th className="px-3 py-2">Type</th>
                <th className="px-3 py-2">Commits</th>
                <th className="px-3 py-2">Spec</th>
              </tr>
            </thead>
            <tbody>
              {refs.map((r) => (
                <tr key={r.handoffId + r.handoffType} className="border-t">
                  <td className="px-3 py-2 font-mono text-xs">{r.handoffId}</td>
                  <td className="px-3 py-2 text-xs">{r.handoffType}</td>
                  <td className="px-3 py-2 font-mono text-xs">
                    {(r.fromCommit || '…').slice(0, 7)}..{(r.toCommit || '…').slice(0, 7)}
                  </td>
                  <td className="px-3 py-2 text-xs text-muted-foreground">{r.specDir || '—'}</td>
                </tr>
              ))}
              {refs.length === 0 && (
                <tr>
                  <td colSpan={4} className="px-3 py-6 text-center text-muted-foreground">
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
      ? 'bg-emerald-100 text-emerald-800'
      : status === 'stale'
        ? 'bg-amber-100 text-amber-800'
        : status === 'archived'
          ? 'bg-slate-100 text-slate-600'
          : 'bg-blue-100 text-blue-800'
  return <span className={`rounded px-1.5 py-0.5 text-xs ${color}`}>{status}</span>
}
