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
    <div className="flex gap-1 rounded-lg border border-slate-700 bg-slate-900/80 p-1">
      {OPTIONS.map((o) => (
        <button
          key={o.kind}
          type="button"
          onClick={() => onChange(o.kind)}
          className={
            'rounded-md px-3 py-1 text-xs font-medium transition-colors ' +
            (value === o.kind
              ? 'bg-accent text-white'
              : 'text-slate-300 hover:bg-slate-800')
          }
        >
          {o.label}
        </button>
      ))}
    </div>
  )
}
