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
    <div className="flex h-full min-h-0 flex-col font-mono">
      <div className="flex items-center gap-2 border-b border-edge-soft px-4 py-2">
        <h1 className="text-[14px] font-semibold text-ink">Memories</h1>
        <span className="text-[12px] text-ink-5">{total} observations · pinned first</span>
      </div>

      {error && (
        <div className="border-b border-danger-ink bg-danger/10 px-4 py-2 text-[12px] text-danger">
          {error}
        </div>
      )}

      <div className="flex min-h-0 flex-1">
        {/* Grid */}
        <aside className="flex w-96 shrink-0 flex-col border-r border-edge-soft">
          <div className="min-h-0 flex-1 overflow-y-auto">
            {loading ? (
              <div className="p-4 text-[12px] text-ink-5">Loading…</div>
            ) : memories.length === 0 ? (
              <div className="p-4 text-[12px] text-ink-6">No memories yet.</div>
            ) : (
              <ul className="divide-y divide-edge">
                {memories.map((m) => (
                  <MemoryRow key={m.id} m={m} active={m.id === selectedId} onSelect={setSelectedId} />
                ))}
              </ul>
            )}
          </div>
          <div className="flex items-center justify-between border-t border-edge px-3 py-2 text-[12px] text-ink-5">
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
              <span className="text-[13px] text-ink-5">Select a memory to view its details and governance.</span>
            </div>
          )}
          {selectedId != null && detailError && (
            <div className="flex h-full items-center justify-center p-8">
              <span className="rounded border border-danger-ink bg-danger/10 px-4 py-2 text-[13px] text-danger">
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
          'flex w-full items-start gap-2 border-l-2 px-3 py-2.5 text-left',
          active ? 'border-accent bg-accent/10' : 'border-transparent hover:bg-card',
        ].join(' ')}
      >
        {m.pinned && <PinIcon />}
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5">
            <span className="truncate text-[13px] font-medium text-ink-2">{m.title || '(untitled)'}</span>
          </div>
          <div className="mt-0.5 flex items-center gap-1.5 text-[11px] text-ink-5">
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
    <span className="rounded bg-edge/60 px-1 py-0.5 text-[10px] text-ink-4">{type}</span>
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
    <div className="px-6 py-5 font-mono">
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <h1 className="text-[19px] font-semibold text-ink">{detail.title || '(untitled)'}</h1>
        <TypeChip type={detail.type} />
        {detail.topic_key && (
          <span className="text-[12px] text-ink-5">{detail.topic_key}</span>
        )}
        {detail.pinned && <PinIcon />}
      </div>

      {/* Governance facts */}
      <div className="mb-4 grid grid-cols-2 gap-2 text-[12px] sm:grid-cols-4">
        <Fact label="Visibility" value={detail.visibility || '—'} />
        <Fact label="Status" value={detail.status || '—'} />
        <Fact label="Owner" value={detail.owner || '—'} />
        <Fact label="Revisions" value={String(detail.revision_count)} />
      </div>

      {editError && (
        <div className="mb-3 rounded border border-danger-ink bg-danger/10 px-3 py-2 text-[12px] text-danger">
          {editError}
        </div>
      )}
      {msg && (
        <div className="mb-3 rounded border border-accent-ink bg-accent/10 px-3 py-2 text-[12px] text-accent">
          {msg}
        </div>
      )}

      {/* Editable content */}
      <textarea
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        rows={10}
        className="w-full rounded border border-edge bg-surface-2/60 p-3 text-[12px] leading-relaxed text-ink-3 focus:border-accent/60 focus:outline-none"
      />
      <div className="mt-2 flex items-center gap-2">
        <button
          type="button"
          disabled={busy || draft === detail.content}
          onClick={() => run(() => onEdit(draft))}
          className="rounded border border-accent bg-accent/15 px-3 py-1.5 text-[12px] font-medium text-accent hover:bg-accent/25 disabled:opacity-40"
        >
          Save content
        </button>
        <div className="ml-auto flex items-center gap-1.5">
          <span className="text-[12px] text-ink-5">Set visibility</span>
          <select
            value={detail.visibility || ''}
            disabled={busy}
            onChange={(e) => run(() => onShare(e.target.value))}
            className="rounded border border-edge-soft bg-inset px-2 py-1 text-[12px] text-ink-3"
          >
            <option value="">—</option>
            <option value="private">private</option>
            <option value="team">team</option>
            <option value="restricted">restricted</option>
            <option value="agent">agent</option>
          </select>
          <span className="text-[12px] text-ink-5">Status</span>
          <select
            value={detail.status || ''}
            disabled={busy}
            onChange={(e) => run(() => onStatus(e.target.value))}
            className="rounded border border-edge-soft bg-inset px-2 py-1 text-[12px] text-ink-3"
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
          <h4 className="mb-2 text-[11px] font-semibold uppercase tracking-[0.12em] text-ink-6">
            Version history ({detail.versions.length})
          </h4>
          <ul className="space-y-1">
            {detail.versions.map((v) => (
              <li key={v.revision} className="flex items-center gap-2 text-[12px] text-ink-4">
                <span className="rounded bg-edge/60 px-1.5 py-0.5 text-[10px] text-ink-5">
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
          <h4 className="mb-2 text-[11px] font-semibold uppercase tracking-[0.12em] text-ink-6">
            Grants ({detail.grants.length})
          </h4>
          <ul className="space-y-1">
            {detail.grants.map((g) => (
              <li key={g.grantee} className="text-[12px] text-ink-4">
                {g.grantee} <span className="text-ink-6">({g.grant_type})</span>
              </li>
            ))}
          </ul>
        </section>
      )}

      {/* Provenance footer */}
      <div className="mt-6 flex flex-wrap gap-x-4 gap-y-1 border-t border-edge pt-3 text-[11px] text-ink-6">
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
    <div className="rounded border border-edge bg-surface-2/60 px-2.5 py-1.5">
      <div className="text-[10px] uppercase tracking-[0.1em] text-ink-6">{label}</div>
      <div className="truncate text-[13px] text-ink-3">{value}</div>
    </div>
  )
}

function PinIcon() {
  return (
    <svg viewBox="0 0 16 16" className="h-3.5 w-3.5 shrink-0 text-warn" fill="currentColor">
      <path d="M9.5 2 14 6.5l-2 .5L9 12l-1.5-1.5L4 14l-1-1 3.5-3.5L5 8z" />
    </svg>
  )
}
