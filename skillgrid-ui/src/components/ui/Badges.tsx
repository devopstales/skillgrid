import type { ReactNode } from 'react'

export function TypeBadge({ type }: { type?: string }) {
  const t = type || 'unknown'
  return <span className={`badge badge-${t}`}>{t}</span>
}

export function StatusBadge({ status }: { status?: string }) {
  const s = status || 'idle'
  return <span className={`status-badge status-${s}`}>{s}</span>
}

export function LoadingState() {
  return (
    <div className="flex items-center justify-center py-8">
      <div className="h-4 w-4 animate-spin rounded-full border-2 border-edge border-t-accent" />
    </div>
  )
}

export function EmptyState({ message }: { message?: string }) {
  return <div className="py-8 text-center text-sm text-ink-4">{message || 'No data'}</div>
}

export function ErrorState({ error }: { error: string }) {
  return (
    <div className="rounded-lg border border-danger/40 bg-danger/10 p-3 text-sm text-red-300">{error}</div>
  )
}

export function KpiCard({
  label,
  value,
  sub,
}: {
  label: string
  value: string | number
  sub?: ReactNode
}) {
  return (
    <div className="kpi-card">
      <div className="kpi-value">{typeof value === 'number' ? value.toLocaleString() : value}</div>
      <div className="kpi-label">{label}</div>
      {sub && <div className="mt-1 text-xs text-ink-4">{sub}</div>}
    </div>
  )
}

export function PageHeader({ title, subtitle }: { title: string; subtitle?: string }) {
  return (
    <div className="mb-4">
      <h1 className="text-sm font-semibold text-ink">{title}</h1>
      {subtitle && <p className="text-xs text-ink-4">{subtitle}</p>}
    </div>
  )
}

export function SectionTitle({ children }: { children: ReactNode }) {
  return <div className="section-title mb-3">{children}</div>
}
