import { useCallback, useEffect, useState } from 'react'
import { apiFetch } from '../../lib/api'
import { ErrorState, LoadingState, PageHeader, SectionTitle } from '../../components/ui/Badges'

interface Finding {
  id: string
  severity: string
  package: string
  installed_version: string
  fixed_version?: string
  title?: string
  target: string
}

interface TrivyResult {
  available?: boolean
  version?: string
  scanned_at?: string
  duration_ms?: number
  critical: number
  high: number
  medium: number
  low: number
  total: number
  fail_on?: string
  verdict: string
  findings?: Finding[]
  error_message?: string
}

function SevBadge({ level, count }: { level: string; count: number }) {
  const colors: Record<string, string> = {
    CRITICAL: 'bg-red-900/60 text-red-300 border-red-700',
    HIGH: 'bg-orange-900/60 text-orange-300 border-orange-700',
    MEDIUM: 'bg-yellow-900/60 text-yellow-300 border-yellow-700',
    LOW: 'bg-blue-900/60 text-blue-300 border-blue-700',
  }
  const cls = colors[level] || 'bg-surface-700 text-ink-4 border-edge'
  return (
    <div className={`rounded-lg border px-3 py-2 ${cls}`}>
      <div className="text-xl font-bold">{count}</div>
      <div className="text-[10px] uppercase tracking-wider">{level}</div>
    </div>
  )
}

export function SecurityPage() {
  const [data, setData] = useState<TrivyResult | null>(null)
  const [error, setError] = useState('')
  const [scanning, setScanning] = useState(false)
  const [expanded, setExpanded] = useState<Set<string>>(new Set())

  const load = useCallback(async () => {
    setScanning(true)
    setError('')
    try {
      const d = await apiFetch<TrivyResult>('/security/trivy')
      setData(d)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setScanning(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  const toggle = (id: string) => {
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  if (error) {
    return (
      <div className="p-4">
        <div className="kpi-card">
          <div className="text-sm font-semibold">Security</div>
          <div className="mt-2 text-xs text-red-400">{error}</div>
          <button
            type="button"
            onClick={() => void load()}
            className="mt-3 rounded bg-surface-700 px-3 py-1 text-xs hover:bg-surface-600"
          >
            Retry
          </button>
        </div>
      </div>
    )
  }

  if (!data) {
    return (
      <div className="p-4">
        <div className="kpi-card">
          <div className="text-sm font-semibold">Security</div>
          <div className="mt-2 text-xs text-ink-4">{scanning ? 'Scanning…' : 'Loading…'}</div>
          <LoadingState />
        </div>
      </div>
    )
  }

  if (data.error_message && !data.available) {
    return (
      <div className="p-4">
        <ErrorState error={data.error_message} />
      </div>
    )
  }

  const verdictCls =
    data.verdict === 'PASS'
      ? 'bg-green-900/60 text-green-300 border-green-700'
      : data.verdict === 'FAIL'
        ? 'bg-red-900/60 text-red-300 border-red-700'
        : 'bg-surface-700 text-ink-4 border-edge'

  return (
    <div className="h-full space-y-4 overflow-y-auto p-4">
      <div className="flex items-center justify-between">
        <PageHeader
          title="Security"
          subtitle={`Trivy ${data.version || ''} · scanned ${data.scanned_at?.slice(0, 19) || '—'}`}
        />
        <button
          type="button"
          onClick={() => void load()}
          disabled={scanning}
          className="rounded bg-surface-700 px-3 py-1.5 text-xs hover:bg-surface-600 disabled:opacity-50"
        >
          {scanning ? 'Scanning…' : 'Re-scan'}
        </button>
      </div>

      <div className={`flex items-center justify-between rounded-lg border px-4 py-3 ${verdictCls}`}>
        <div>
          <span className="text-lg font-bold">{data.verdict}</span>
          {data.fail_on && <span className="ml-2 text-xs opacity-70">fail_on: {data.fail_on}</span>}
        </div>
        <span className="text-xs opacity-70">
          {data.total} finding{data.total !== 1 ? 's' : ''}
        </span>
      </div>

      <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
        <SevBadge level="CRITICAL" count={data.critical} />
        <SevBadge level="HIGH" count={data.high} />
        <SevBadge level="MEDIUM" count={data.medium} />
        <SevBadge level="LOW" count={data.low} />
      </div>

      {data.findings && data.findings.length > 0 && (
        <div className="kpi-card">
          <SectionTitle>Findings</SectionTitle>
          <div className="space-y-2">
            {data.findings.map((f, i) => {
              const key = `${f.id}-${i}`
              const isOpen = expanded.has(key)
              const sevCls: Record<string, string> = {
                CRITICAL: 'text-red-400',
                HIGH: 'text-orange-400',
                MEDIUM: 'text-yellow-400',
                LOW: 'text-blue-400',
              }
              return (
                <div key={key} className="overflow-hidden rounded-lg border border-edge">
                  <button
                    type="button"
                    onClick={() => toggle(key)}
                    className="flex w-full items-center gap-3 px-3 py-2 text-left hover:bg-surface-700/50"
                  >
                    <span className={`w-20 shrink-0 font-mono text-xs font-bold ${sevCls[f.severity] || 'text-ink-4'}`}>
                      {f.severity}
                    </span>
                    <span className="shrink-0 font-mono text-xs text-ink-4">{f.id}</span>
                    <span className="flex-1 truncate text-xs">
                      {f.package} {f.installed_version}
                    </span>
                    <span className="text-[10px] text-ink-4">{isOpen ? '−' : '+'}</span>
                  </button>
                  {isOpen && (
                    <div className="space-y-1 px-3 pb-3 text-xs">
                      <div>
                        <span className="text-ink-4">Title:</span> {f.title || '—'}
                      </div>
                      <div>
                        <span className="text-ink-4">Package:</span>{' '}
                        <span className="font-mono">{f.package}</span>{' '}
                        <span className="text-ink-4">({f.installed_version})</span>
                      </div>
                      {f.fixed_version && (
                        <div>
                          <span className="text-ink-4">Fixed in:</span>{' '}
                          <span className="font-mono text-green-400">{f.fixed_version}</span>
                        </div>
                      )}
                      <div>
                        <span className="text-ink-4">Target:</span>{' '}
                        <span className="font-mono text-ink-4">{f.target}</span>
                      </div>
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        </div>
      )}

      {(!data.findings || data.findings.length === 0) && (
        <div className="kpi-card py-6 text-center">
          <div className="text-sm font-semibold text-green-400">No vulnerabilities found</div>
          <div className="mt-1 text-xs text-ink-4">Trivy scan completed in {data.duration_ms}ms</div>
        </div>
      )}

      <div className="text-[10px] text-ink-4">
        Scan duration: {data.duration_ms}ms · Advisory only — findings are reported, never blocking.
      </div>
    </div>
  )
}
