import type { ChangeSnapshot, Checkpoint } from './api'

// ChangesPane — the change log with checkpoint markers overlaid. Moved from the
// former Handoff Hub `changes` tab (sessions-activity-unification). Data comes
// from fetchSnapshots + fetchHubStatus; live updates arrive via the snapshot
// SSE (wired by the parent).
export function ChangesPane({
  snapshots,
  checkpointByCommit,
  openCheckpoints,
  latest,
}: {
  snapshots: ChangeSnapshot[]
  checkpointByCommit: Map<string, Checkpoint>
  openCheckpoints: Checkpoint[]
  latest: ChangeSnapshot | null
}) {
  return (
    <div className="flex flex-col gap-4 overflow-y-auto p-6">
      {latest && (
        <div className="rounded-lg border p-4">
          <div className="text-xs uppercase text-muted-foreground">Latest commit</div>
          <div className="mt-1 flex items-center gap-2 font-mono text-sm">
            <span className="rounded bg-muted px-1.5 py-0.5">{latest.commitShort}</span>
            <span>{latest.subject}</span>
            <span className="text-muted-foreground">({latest.branch})</span>
          </div>
          {latest.context && (
            <div className="mt-3 grid gap-2 text-sm sm:grid-cols-2">
              {latest.context.task && <Ctx label="Task" value={latest.context.task} />}
              {latest.context.decisions && <Ctx label="Decisions" value={latest.context.decisions} />}
              {latest.context.remaining && <Ctx label="Remaining" value={latest.context.remaining} />}
              {latest.context.tried && <Ctx label="Tried" value={latest.context.tried} />}
            </div>
          )}
        </div>
      )}

      {openCheckpoints.length > 0 && (
        <div className="rounded-lg border border-amber-300 bg-amber-50 p-3">
          <div className="text-xs font-medium text-amber-800">
            Open checkpoints ({openCheckpoints.length})
          </div>
          <ul className="mt-1 space-y-0.5 text-sm text-amber-900">
            {openCheckpoints.map((c) => (
              <li key={c.id} className="font-mono">
                {c.name} <span className="text-amber-700">[{c.status}]</span> @ {c.commit.slice(0, 7)}
                {c.specDir ? <span className="text-amber-700"> · {c.specDir}</span> : null}
              </li>
            ))}
          </ul>
        </div>
      )}

      <div className="overflow-hidden rounded-lg border">
        <table className="w-full text-sm">
          <thead className="bg-muted/50 text-left text-xs uppercase text-muted-foreground">
            <tr>
              <th className="px-3 py-2">Commit</th>
              <th className="px-3 py-2">Subject</th>
              <th className="px-3 py-2">Branch</th>
              <th className="px-3 py-2">Context</th>
              <th className="px-3 py-2">Checkpoint</th>
            </tr>
          </thead>
          <tbody>
            {snapshots.map((s) => {
              const cp = checkpointByCommit.get(s.commit)
              return (
                <tr key={s.commit} className="border-t hover:bg-muted/30">
                  <td className="px-3 py-2 font-mono text-xs">{s.commitShort}</td>
                  <td className="px-3 py-2">{s.subject}</td>
                  <td className="px-3 py-2 text-muted-foreground">{s.branch}</td>
                  <td className="px-3 py-2 text-xs text-muted-foreground">
                    {s.context?.task ? s.context.task : '—'}
                  </td>
                  <td className="px-3 py-2">
                    {cp ? (
                      <span className="rounded bg-amber-100 px-1.5 py-0.5 text-xs text-amber-800">
                        {cp.name}
                      </span>
                    ) : (
                      <span className="text-xs text-muted-foreground">—</span>
                    )}
                  </td>
                </tr>
              )
            })}
            {snapshots.length === 0 && (
              <tr>
                <td colSpan={5} className="px-3 py-6 text-center text-muted-foreground">
                  No snapshots recorded yet. Run `skillgrid handoff record` or commit to populate
                  the log.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function Ctx({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <span className="text-xs font-medium text-muted-foreground">{label}</span>
      <div className="text-sm">{value}</div>
    </div>
  )
}
