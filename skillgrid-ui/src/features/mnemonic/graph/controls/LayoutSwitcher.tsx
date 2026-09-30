import type { LayoutKind } from '../layouts'

const OPTIONS: { kind: LayoutKind; label: string }[] = [
  { kind: 'force', label: 'Force' },
  { kind: 'tree', label: 'Tree' },
  { kind: 'circles', label: 'Circles' },
]

interface Props {
  value: LayoutKind
  onChange: (k: LayoutKind) => void
}

export function LayoutSwitcher({ value, onChange }: Props) {
  return (
    <div className="flex gap-1 rounded border border-edge bg-surface-2/90 p-1">
      {OPTIONS.map((o) => (
        <button
          key={o.kind}
          type="button"
          onClick={() => onChange(o.kind)}
          className={
            'rounded px-3 py-1 text-[12px] font-medium transition-colors ' +
            (value === o.kind
              ? 'bg-accent/15 text-accent'
              : 'text-ink-4 hover:bg-surface-2')
          }
        >
          {o.label}
        </button>
      ))}
    </div>
  )
}
