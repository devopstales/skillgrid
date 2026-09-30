import { useMemo } from 'react'
import type { ActivityEvent } from './api'

export interface ActivityFilters {
  type: string
  source: string
  severity: string
  actor: string
}

export const EMPTY_FILTERS: ActivityFilters = { type: '', source: '', severity: '', actor: '' }

// Filters — type / source / severity / actor dropdowns. Distinct values are
// derived from the loaded events so the filter options always match what's
// present.
export function Filters({
  events,
  value,
  onChange,
}: {
  events: ActivityEvent[]
  value: ActivityFilters
  onChange: (f: ActivityFilters) => void
}) {
  const opts = useMemo(() => {
    const uniq = (arr: (string | undefined | null)[]) =>
      Array.from(new Set(arr.filter((x): x is string => !!x && x !== '').sort()))
    return {
      type: uniq(events.map((e) => e.type)),
      source: uniq(events.map((e) => e.source)),
      severity: uniq(events.map((e) => e.severity)),
      actor: uniq(events.map((e) => e.actor)),
    }
  }, [events])

  const set = (k: keyof ActivityFilters, v: string) => onChange({ ...value, [k]: v })
  const active = Object.values(value).some((v) => v !== '')

  return (
    <div className="flex flex-wrap items-center gap-2">
      <Select label="type" value={value.type} options={opts.type} onChange={(v) => set('type', v)} />
      <Select label="source" value={value.source} options={opts.source} onChange={(v) => set('source', v)} />
      <Select label="severity" value={value.severity} options={opts.severity} onChange={(v) => set('severity', v)} />
      <Select label="actor" value={value.actor} options={opts.actor} onChange={(v) => set('actor', v)} />
      {active && (
        <button
          type="button"
          onClick={() => onChange(EMPTY_FILTERS)}
          className="rounded border border-edge-soft px-2 py-1 font-mono text-[12px] text-ink-4 hover:text-ink-2"
        >
          Clear
        </button>
      )}
    </div>
  )
}

function Select({
  label,
  value,
  options,
  onChange,
}: {
  label: string
  value: string
  options: string[]
  onChange: (v: string) => void
}) {
  return (
    <label className="flex items-center gap-1 font-mono text-[11px] text-ink-6">
      {label}
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="rounded border border-edge-soft bg-inset px-2 py-1 text-[12px] text-ink-3 focus:border-accent focus:outline-none"
      >
        <option value="">all</option>
        {options.map((o) => (
          <option key={o} value={o}>
            {o}
          </option>
        ))}
      </select>
    </label>
  )
}
