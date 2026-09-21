import { useEffect, useMemo, useState } from 'react'
import {
  fetchSnapshots,
  fetchHubStatus,
  openSnapshotStream,
  HandoffError,
  type ChangeSnapshot,
  type Checkpoint,
  type HandoffRef,
} from './api'
import { fetchActivityEvents, type ActivityEvent } from '../activity/api'

type Tab = 'changes' | 'activity' | 'handoffs'

// Handoff Hub (change 015-handoff-hub): one page combining the change-tracking
// snapshot log, live activity, and team/agent handoffs. Change log is backed
// by the git-derived change_snapshots; checkpoints overlay as markers; the
// live snapshot stream reuses /activity/stream (event: snapshot).

export function HandoffHubPage() {
  const [tab, setTab] = useState<Tab>('changes')
  const [snapshots, setSnapshots] = useState<ChangeSnapshot[]>([])
  const [checkpoints, setCheckpoints] = useState<Checkpoint[]>([])
  const [refs, setRefs] = useState<HandoffRef[]>([])
  const [events, setEvents] = useState<ActivityEvent[]>([])
  const [error, setError] = useState('')
  const [live, setLive] = useState(false)
  const [latest, setLatest] = useState<ChangeSnapshot | null>(null)

  const load = () => {
    Promise.all([fetchSnapshots(200), fetchHubStatus()])
      .then(([snaps, status]) => {
        setSnapshots(snaps.snapshots ?? [])
        setCheckpoints(status.checkpoints ?? [])
        setRefs(status.handoff_refs ?? [])
        setLatest(status.latest_snapshot ?? null)
      })
      .catch((e) => {
        setError(e instanceof HandoffError ? e.message : 'failed to load handoff hub')
      })
    fetchActivityEvents(100)
      .then((r) => setEvents(r.events ?? []))
      .catch(() => {})
  }

  useEffect(() => {
    setError('')
    load()
    // Live change snapshots (event: snapshot) — prepend, de-dup by commit.
    const close = openSnapshotStream({
      onReady: () => setLive(true),
      onSnapshot: (s) => {
        setSnapshots((prev) => {
          if (prev.some((p) => p.commit === s.commit)) return prev
          const ns: ChangeSnapshot = {
            id: Date.now(),
            branch: s.branch,
            commit: s.commit,
            commitShort: s.commitShort,
            subject: s.subject,
            author: s.author,
            committedAt: s.committedAt,
            changedFiles: '',
          }
          setLatest(ns)
          return [ns, ...prev].slice(0, 200)
        })
      },
      onError: () => setLive(false),
    })
    return close
  }, [])

  const checkpointByCommit = useMemo(() => {
    const m = new Map<string, Checkpoint>()
    for (const c of checkpoints) m.set(c.commit, c)
    return m
  }, [checkpoints])

  const openCheckpoints = useMemo(
    () => checkpoints.filter((c) => c.status === 'open' || c.status === 'stale'),
    [checkpoints],
  )

  return (
    <div className="flex h-full flex-col gap-4 overflow-y-auto p-8">
      <header className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Handoff Hub</h1>
          <p className="text-sm text-muted-foreground">
            Change tracking, live activity, and team/agent handoffs — one engine-backed view.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <span className={live ? 'text-xs text-emerald-500' : 'text-xs text-muted-foreground'}>
            {live ? '● live' : '○ offline'}
          </span>
          <nav className="flex gap-1 rounded-lg border p-1">
            {(['changes', 'activity', 'handoffs'] as Tab[]).map((t) => (
              <button
                key={t}
                onClick={() => setTab(t)}
                className={`rounded-md px-3 py-1 text-sm capitalize ${
                  tab === t ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:bg-muted'
                }`}
              >
                {t === 'handoffs' ? 'Handoffs' : t}
              </button>
            ))}
          </nav>
        </div>
      </header>

      {error && (
        <div className="rounded-md border border-red-300 bg-red-50 p-3 text-sm text-red-700">{error}</div>
      )}

      {tab === 'changes' && (
        <ChangesTab
          snapshots={snapshots}
          checkpointByCommit={checkpointByCommit}
          openCheckpoints={openCheckpoints}
          latest={latest}
        />
      )}
      {tab === 'activity' && <ActivityTab events={events} />}
      {tab === 'handoffs' && <HandoffsTab checkpoints={checkpoints} refs={refs} />}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Changes tab — the change log with checkpoint markers overlaid.
// ---------------------------------------------------------------------------

function ChangesTab({
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
    <div className="flex flex-col gap-4">
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
          <div className="text-xs font-medium text-amber-800">Open checkpoints ({openCheckpoints.length})</div>
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
                      <span className="rounded bg-amber-100 px-1.5 py-0.5 text-xs text-amber-800">{cp.name}</span>
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
                  No snapshots recorded yet. Run `skillgrid handoff record` or commit to populate the log.
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

// ---------------------------------------------------------------------------
// Activity tab — reuses the activity feed (observations).
// ---------------------------------------------------------------------------

function ActivityTab({ events }: { events: ActivityEvent[] }) {
  return (
    <div className="flex flex-col gap-2">
      {events.slice(0, 50).map((e) => (
        <div key={e.id} className="flex items-start gap-3 rounded-md border p-3">
          <span
            className={`mt-1 h-2 w-2 shrink-0 rounded-full ${
              e.severity === 'high' ? 'bg-red-500' : e.severity === 'medium' ? 'bg-amber-500' : 'bg-slate-400'
            }`}
          />
          <div className="min-w-0">
            <div className="truncate text-sm font-medium">{e.summary}</div>
            <div className="text-xs text-muted-foreground">
              {e.type} · {e.source} {e.actor ? `· ${e.actor}` : ''} · {new Date(e.ts).toLocaleString()}
            </div>
          </div>
        </div>
      ))}
      {events.length === 0 && (
        <div className="rounded-md border p-6 text-center text-sm text-muted-foreground">No activity yet.</div>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Handoffs tab — checkpoints + handoff refs (session/team handoffs).
// ---------------------------------------------------------------------------

function HandoffsTab({ checkpoints, refs }: { checkpoints: Checkpoint[]; refs: HandoffRef[] }) {
  return (
    <div className="flex flex-col gap-4">
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
                  <td colSpan={5} className="px-3 py-6 text-center text-muted-foreground">No checkpoints placed.</td>
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
