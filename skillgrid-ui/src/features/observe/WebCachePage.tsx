import { useEffect, useState } from 'react'
import { apiGet } from '../../lib/api'
import {
  EmptyState,
  ErrorState,
  KpiCard,
  LoadingState,
  PageHeader,
  SectionTitle,
  TypeBadge,
} from '../../components/ui/Badges'

interface WebStatus {
  total_entries?: number
  by_source?: Record<string, number>
}

interface WebEntry {
  id: string
  source?: string
  title?: string
  query?: string
  url?: string
  cached_at?: string
}

const SOURCES = ['all', 'context7', 'exa', 'deepwiki', 'fetch', 'manual']

export function WebCachePage() {
  const [status, setStatus] = useState<WebStatus | null>(null)
  const [entries, setEntries] = useState<WebEntry[] | null>(null)
  const [source, setSource] = useState('all')
  const [error, setError] = useState<string | null>(null)
  const [searchError, setSearchError] = useState<string | null>(null)

  useEffect(() => {
    apiGet<WebStatus>('/web/status')
      .then(setStatus)
      .catch((err: Error) => setError(err.message))
  }, [])

  useEffect(() => {
    if (!status) return
    const params = source !== 'all' ? { source, q: '' } : { q: '' }
    setSearchError(null)
    apiGet<{ results?: WebEntry[] }>('/web/search', params)
      .then((d) => setEntries(d.results || []))
      .catch((err: Error) => setSearchError(err.message))
  }, [status, source])

  if (error) {
    return (
      <div className="p-6">
        <ErrorState error={error} />
      </div>
    )
  }
  if (!status) {
    return (
      <div className="p-6">
        <LoadingState />
      </div>
    )
  }

  return (
    <div className="p-6">
      <div className="mb-4 flex items-center justify-between">
        <PageHeader title="Web Cache" subtitle="Cached web research snapshots" />
        <select
          value={source}
          onChange={(e) => setSource(e.target.value)}
          className="rounded border border-edge bg-surface-700 px-3 py-1.5 text-xs focus:border-accent focus:outline-none"
        >
          {SOURCES.map((s) => (
            <option key={s} value={s}>
              {s}
            </option>
          ))}
        </select>
      </div>

      <div className="mb-4 grid grid-cols-2 gap-4 md:grid-cols-4">
        <KpiCard label="Total Entries" value={status.total_entries || 0} />
        <KpiCard label="Context7" value={status.by_source?.context7 || 0} />
        <KpiCard label="Exa" value={status.by_source?.exa || 0} />
        <KpiCard label="Fetch" value={status.by_source?.fetch || 0} />
      </div>

      <div className="kpi-card">
        <SectionTitle>Entries</SectionTitle>
        {searchError ? (
          <ErrorState error={searchError} />
        ) : entries ? (
          entries.length > 0 ? (
            <div className="space-y-1">
              {entries.map((e) => (
                <div key={e.id} className="table-row flex items-center gap-3 px-1 py-2 text-sm">
                  <TypeBadge type={e.source} />
                  <span className="flex-1 truncate text-ink">{e.title || e.query || 'untitled'}</span>
                  {e.url && (
                    <span className="max-w-[300px] truncate font-mono text-xs text-ink-4">{e.url}</span>
                  )}
                  {e.cached_at && (
                    <span className="shrink-0 font-mono text-xs text-ink-4">
                      {e.cached_at.slice(0, 10)}
                    </span>
                  )}
                </div>
              ))}
            </div>
          ) : (
            <EmptyState
              message={
                status.total_entries === 0
                  ? 'Cache is empty — no web research cached yet'
                  : 'No entries match filter'
              }
            />
          )
        ) : (
          <LoadingState />
        )}
      </div>
    </div>
  )
}
