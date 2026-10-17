import { useEffect, useState } from 'react'
import { ErrorState, LoadingState, PageHeader, SectionTitle } from '../../components/ui/Badges'
import { fetchReport, type ReportJSON } from './api'

function CvePanel({ findings }: { findings: ReportJSON['static'][number]['findings'] }) {
  const sevCls: Record<string, string> = {
    CRITICAL: 'text-red-400',
    HIGH: 'text-orange-400',
    MEDIUM: 'text-yellow-400',
    LOW: 'text-blue-400',
  }
  return (
    <div className="kpi-card">
      <SectionTitle>CVE (Trivy)</SectionTitle>
      {findings.length === 0 ? (
        <div className="py-4 text-center text-sm text-green-400">No vulnerabilities found</div>
      ) : (
        <div className="space-y-2">
          {findings.map((f, i) => (
            <div key={`${f.cve_id || f.rule_id}-${i}`} className="rounded-lg border border-edge px-3 py-2">
              <div className="flex items-center gap-3">
                <span className={`w-20 shrink-0 font-mono text-xs font-bold ${sevCls[f.severity] || 'text-ink-4'}`}>
                  {f.severity}
                </span>
                <span className="shrink-0 font-mono text-xs text-ink-4">{f.cve_id || f.rule_id}</span>
                <span className="flex-1 truncate text-xs" data-testid="cve-package">
                  {f.package}
                </span>
                <span className="shrink-0 font-mono text-[10px] text-ink-4">{f.version}</span>
              </div>
              <div className="mt-1 text-xs">
                <span className="text-ink-4">{f.title}</span>
                {f.fixed_version && (
                  <span className="ml-2 font-mono text-green-400">fixed in {f.fixed_version}</span>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

export function ReportsPage() {
  const [rep, setRep] = useState<ReportJSON | null>(null)
  const [err, setErr] = useState('')

  useEffect(() => {
    fetchReport().then(setRep).catch((e) => setErr(e instanceof Error ? e.message : String(e)))
  }, [])

  if (err) return <ErrorState error={err} />
  if (!rep) return <LoadingState />
  const trivy = rep.static.find((s) => s.tool === 'trivy')
  return (
    <div className="h-full space-y-4 overflow-y-auto p-4">
      <PageHeader
        title="Quality Report"
        subtitle={`generated ${rep.generated_at ? rep.generated_at.slice(0, 19) : '—'}`}
      />
      {trivy?.status === 'error' && (
        <div className="kpi-card py-3 text-xs text-red-400" data-testid="trivy-source-error">
          Trivy source failed — see scans for details.
        </div>
      )}
      <CvePanel findings={trivy?.findings ?? []} />
    </div>
  )
}
