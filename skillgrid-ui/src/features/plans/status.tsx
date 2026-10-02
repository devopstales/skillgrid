// Status → chip classification for the Changes (plans) view. The backend
// surfaces free-form SDD status tokens (`done`, `in-progress`, `approved`,
// `PENDING`, …); the prototype's row list reduces them to a 4-state gate chip:
// pass (shipped/done), active (executing), blocked (failed), pending (rest).

export type ChipKind = 'pass' | 'active' | 'blocked' | 'pending'

const PASS = ['done', 'complete', 'shipped', 'archived', 'reflected']
const ACTIVE = ['progress', 'executing', 'active', 'approved', 'qa', 'review']
const BLOCKED = ['blocked', 'fail', 'rejected']

export function chipKind(status: string | undefined | null): ChipKind {
  const s = (status ?? '').toLowerCase()
  if (BLOCKED.some((k) => s.includes(k))) return 'blocked'
  if (PASS.some((k) => s.includes(k))) return 'pass'
  if (ACTIVE.some((k) => s.includes(k))) return 'active'
  return 'pending'
}

// Status chip: --r-1 radius, --fs-sub, text + hairline border + soft fill in
// the state hue. Never a solid fill (DESIGN.md §6).
export const CHIP_CLASS: Record<ChipKind, string> = {
  pass: 'border-ok/40 bg-ok/[0.08] text-ok',
  active: 'border-warn-ink bg-warn/[0.08] text-warn',
  blocked: 'border-danger-ink bg-danger/[0.08] text-danger',
  pending: 'border-edge bg-inset text-ink-5',
}

// Dot colour for the pipeline graph nodes (same hue as the chip text).
export const DOT_CLASS: Record<ChipKind, string> = {
  pass: 'border-ok bg-ok/30',
  active: 'border-warn bg-warn/30',
  blocked: 'border-danger bg-danger/30',
  pending: 'border-ink-6 bg-inset',
}

// Pipeline order: live work first, then queued, then shipped.
export const KIND_ORDER: Record<ChipKind, number> = {
  active: 0,
  blocked: 1,
  pending: 2,
  pass: 3,
}

export function StatusChip({ status, className = '' }: { status: string; className?: string }) {
  const kind = chipKind(status)
  return (
    <span
      data-testid="plan-status-chip"
      data-kind={kind}
      className={`inline-block rounded-[3px] border px-2 py-px font-mono text-[11px] leading-[16px] ${CHIP_CLASS[kind]} ${className}`}
    >
      {kind}
    </span>
  )
}
