import { useCallback, useEffect, useState } from 'react'
import {
  editMemory,
  fetchMemories,
  fetchMemory,
  MnemonicError,
  shareMemory,
  setStatus,
  type MemoryDetail,
  type MemorySummary,
} from './api'

// MemoriesPage is the memory browser: a paginated grid (pinned first) on the
// left, and a detail pane with the 013 governance overlay (versions, grants,
// visibility, status) + write-gated edit/share/status controls on the right.

const PAGE = 30

export function MemoriesPage() {
  const [memories, setMemories] = useState<MemorySummary[]>([])
  const [total, setTotal] = useState(0)
  const [offset, setOffset] = useState(0)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [detail, setDetail] = useState<MemoryDetail | null>(null)
  const [detailError, setDetailError] = useState('')

  const loadPage = useCallback(async (off: number) => {
    setLoading(true)
    setError('')
    try {
      const res = await fetchMemories(PAGE, off)
      setMemories(res.memories)
      setTotal(res.total)
    } catch (e) {
      setError(e instanceof MnemonicError ? e.message : 'failed to load memories')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadPage(offset)
  }, [offset, loadPage])

  useEffect(() => {
    if (selectedId == null) {
      setDetail(null)
      return
    }
    let cancelled = false
    setDetailError('')
    fetchMemory(selectedId)
      .then((d) => {
        if (!cancelled) setDetail(d)
      })
      .catch((e) => {
        if (!cancelled) setDetailError(e instanceof MnemonicError ? e.message : 'failed to load memory')
      })
    return () => {
      cancelled = true
    }
  }, [selectedId])

  const onEdit = useCallback(
    async (content: string) => {
      if (!detail) return
      await editMemory(detail.id, { content })
      const d = await fetchMemory(detail.id)
      setDetail(d)
    },
    [detail],
  )

  const onShare = useCallback(
    async (visibility: string) => {
      if (!detail) return
      await shareMemory(detail.id, { visibility })
      setDetail(await fetchMemory(detail.id))
    },
    [detail],
  )

  const onStatus = useCallback(
    async (status: string) => {
      if (!detail) return
      await setStatus(detail.id, status)
      setDetail(await fetchMemory(detail.id))
    },
    [detail],
  )

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="flex items-center gap-2 border-b border-edge px-4 py-2">
        <h1 className="text-sm font-semibold text-zinc-100">Memories</h1>
        <span className="text-xs text-zinc-500">{total} observations · pinned first</span>
      </div>

      {error && (
        <div className="border-b border-red-500/30 bg-red-500/10 px-4 py-2 text-xs text-red-300">
          {error}
        </div>
      )}

      <div className="flex min-h-0 flex-1">
        {/* Grid */}
        <aside className="flex w-96 shrink-0 flex-col border-r border-edge">
          <div className="min-h-0 flex-1 overflow-y-auto">
            {loading ? (
              <div className="p-4 text-xs text-zinc-500">Loading…</div>
            ) : memories.length === 0 ? (
              <div className="p-4 text-xs text-zinc-600">No memories yet.</div>
            ) : (
              <ul className="divide-y divide-edge">
                {memories.map((m) => (
                  <MemoryRow key={m.id} m={m} active={m.id === selectedId} onSelect={setSelectedId} />
                ))}
              </ul>
            )}
          </div>
          <div className="flex items-center justify-between border-t border-edge px-3 py-2 text-xs text-zinc-500">
            <button
              type="button"
              disabled={offset === 0}
              onClick={() => setOffset((o) => Math.max(0, o - PAGE))}
              className="rounded px-2 py-1 hover:bg-card disabled:opacity-40"
            >
              ← Prev
            </button>
            <span>
              {Math.min(offset + 1, total)}–{Math.min(offset + PAGE, total)} of {total}
            </span>
            <button
              type="button"
              disabled={offset + PAGE >= total}
              onClick={() => setOffset((o) => o + PAGE)}
              className="rounded px-2 py-1 hover:bg-card disabled:opacity-40"
            >
              Next →
            </button>
          </div>
        </aside>

        {/* Detail */}
        <main className="min-h-0 flex-1 overflow-y-auto">
          {selectedId == null && (
            <div className="flex h-full items-center justify-center p-8 text-center">
              <span className="text-sm text-zinc-400">Select a memory to view its details and governance.</span>
            </div>
          )}
          {selectedId != null && detailError && (
            <div className="flex h-full items-center justify-center p-8">
              <span className="rounded-md border border-red-500/30 bg-red-500/10 px-4 py-2 text-sm text-red-300">
                {detailError}
              </span>
            </div>
          )}
          {selectedId != null && detail && (
            <MemoryDetailPane detail={detail} onEdit={onEdit} onShare={onShare} onStatus={onStatus} />
          )}
        </main>
      </div>
    </div>
  )
}

function MemoryRow({
  m,
  active,
  onSelect,
}: {
  m: MemorySummary
  active: boolean
  onSelect: (id: number) => void
}) {
  return (
    <li>
      <button
        type="button"
        onClick={() => onSelect(m.id)}
        className={[
          'flex w-full items-start gap-2 px-3 py-2.5 text-left',
          active ? 'bg-accent/15' : 'hover:bg-card',
        ].join(' ')}
      >
        {m.pinned && <PinIcon />}
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5">
            <span className="truncate text-sm font-medium text-zinc-200">{m.title || '(untitled)'}</span>
          </div>
          <div className="mt-0.5 flex items-center gap-1.5 text-[11px] text-zinc-500">
            <TypeChip type={m.type} />
            {m.topic_key && <span className="truncate">{m.topic_key}</span>}
            <span className="ml-auto shrink-0">{m.created_at.slice(0, 10)}</span>
          </div>
        </div>
      </button>
    </li>
  )
}

function TypeChip({ type }: { type: string }) {
  return (
    <span className="rounded bg-zinc-700/60 px-1 py-0.5 text-[10px] text-zinc-400">{type}</span>
  )
}

function MemoryDetailPane({
  detail,
  onEdit,
  onShare,
  onStatus,
}: {
  detail: MemoryDetail
  onEdit: (content: string) => Promise<void>
  onShare: (visibility: string) => Promise<void>
  onStatus: (status: string) => Promise<void>
}) {
  const [draft, setDraft] = useState(detail.content)
  const [busy, setBusy] = useState(false)
  const [msg, setMsg] = useState('')
  const [editError, setEditError] = useState('')

  useEffect(() => {
    setDraft(detail.content)
    setMsg('')
    setEditError('')
  }, [detail])

  const run = async (fn: () => Promise<unknown>) => {
    setBusy(true)
    setEditError('')
    setMsg('')
    try {
      await fn()
      setMsg('Saved.')
    } catch (e) {
      setEditError(e instanceof MnemonicError ? e.message : 'operation failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="px-6 py-5">
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <h1 className="text-xl font-semibold text-zinc-100">{detail.title || '(untitled)'}</h1>
        <TypeChip type={detail.type} />
        {detail.topic_key && (
          <span className="text-xs text-zinc-500">{detail.topic_key}</span>
        )}
        {detail.pinned && <PinIcon />}
      </div>

      {/* Governance facts */}
      <div className="mb-4 grid grid-cols-2 gap-2 text-xs sm:grid-cols-4">
        <Fact label="Visibility" value={detail.visibility || '—'} />
        <Fact label="Status" value={detail.status || '—'} />
        <Fact label="Owner" value={detail.owner || '—'} />
        <Fact label="Revisions" value={String(detail.revision_count)} />
      </div>

      {editError && (
        <div className="mb-3 rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-300">
          {editError}
        </div>
      )}
      {msg && (
        <div className="mb-3 rounded-md border border-emerald-500/30 bg-emerald-500/10 px-3 py-2 text-xs text-emerald-300">
          {msg}
        </div>
      )}

      {/* Editable content */}
      <textarea
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        rows={10}
        className="w-full rounded-md border border-edge bg-card/40 p-3 font-mono text-xs leading-relaxed text-zinc-300 focus:border-accent/60 focus:outline-none"
      />
      <div className="mt-2 flex items-center gap-2">
        <button
          type="button"
          disabled={busy || draft === detail.content}
          onClick={() => run(() => onEdit(draft))}
          className="rounded-md bg-accent/80 px-3 py-1.5 text-xs font-medium text-zinc-950 hover:bg-accent disabled:opacity-40"
        >
          Save content
        </button>
        <div className="ml-auto flex items-center gap-1.5">
          <span className="text-xs text-zinc-500">Set visibility</span>
          <select
            value={detail.visibility || ''}
            disabled={busy}
            onChange={(e) => run(() => onShare(e.target.value))}
            className="rounded-md border border-edge bg-card px-2 py-1 text-xs text-zinc-300"
          >
            <option value="">—</option>
            <option value="private">private</option>
            <option value="team">team</option>
            <option value="restricted">restricted</option>
            <option value="agent">agent</option>
          </select>
          <span className="text-xs text-zinc-500">Status</span>
          <select
            value={detail.status || ''}
            disabled={busy}
            onChange={(e) => run(() => onStatus(e.target.value))}
            className="rounded-md border border-edge bg-card px-2 py-1 text-xs text-zinc-300"
          >
            <option value="">—</option>
            <option value="active">active</option>
            <option value="superseded">superseded</option>
            <option value="archived">archived</option>
          </select>
        </div>
      </div>

      {/* Version history */}
      {detail.versions && detail.versions.length > 0 && (
        <section className="mt-6">
          <h4 className="mb-2 text-xs font-semibold uppercase tracking-wide text-zinc-500">
            Version history ({detail.versions.length})
          </h4>
          <ul className="space-y-1">
            {detail.versions.map((v) => (
              <li key={v.revision} className="flex items-center gap-2 text-xs text-zinc-400">
                <span className="rounded bg-zinc-700/60 px-1.5 py-0.5 text-[10px] text-zinc-500">
                  rev {v.revision}
                </span>
                <span>{v.created_at}</span>
              </li>
            ))}
          </ul>
        </section>
      )}

      {/* ACL grants */}
      {detail.grants && detail.grants.length > 0 && (
        <section className="mt-6">
          <h4 className="mb-2 text-xs font-semibold uppercase tracking-wide text-zinc-500">
            Grants ({detail.grants.length})
          </h4>
          <ul className="space-y-1">
            {detail.grants.map((g) => (
              <li key={g.grantee} className="text-xs text-zinc-400">
                {g.grantee} <span className="text-zinc-600">({g.grant_type})</span>
              </li>
            ))}
          </ul>
        </section>
      )}

      {/* Provenance footer */}
      <div className="mt-6 flex flex-wrap gap-x-4 gap-y-1 border-t border-edge pt-3 text-[11px] text-zinc-600">
        <span>id {detail.id}</span>
        <span>scope {detail.scope || '—'}</span>
        <span>source {detail.source || '—'}</span>
        <span>created {detail.created_at}</span>
        <span>updated {detail.updated_at}</span>
      </div>
    </div>
  )
}

function Fact({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-md border border-edge bg-card/40 px-2.5 py-1.5">
      <div className="text-[10px] uppercase tracking-wide text-zinc-600">{label}</div>
      <div className="truncate text-sm text-zinc-200">{value}</div>
    </div>
  )
}

function PinIcon() {
  return (
    <svg viewBox="0 0 16 16" className="h-3.5 w-3.5 shrink-0 text-amber-400" fill="currentColor">
      <path d="M9.5 2 14 6.5l-2 .5L9 12l-1.5-1.5L4 14l-1-1 3.5-3.5L5 8z" />
    </svg>
  )
}
