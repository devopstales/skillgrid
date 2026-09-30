interface Props {
  value: number
  min?: number
  max?: number
  onChange: (depth: number) => void
}

// DepthSlider: bounds the neighborhood radius (hops) when a center node is
// focused. At 0 only the center node shows.
export function DepthSlider({ value, min = 0, max = 6, onChange }: Props) {
  return (
    <div className="rounded border border-edge bg-surface-2/90 p-2">
      <div className="mb-1 flex items-center justify-between text-[11px] text-ink-5">
        <span className="font-semibold uppercase tracking-[0.1em]">Depth</span>
        <span className="text-accent">{value} hops</span>
      </div>
      <input
        type="range"
        min={min}
        max={max}
        step={1}
        value={value}
        onChange={(e) => onChange(Number(e.target.value))}
        className="w-40 accent-[#22c55e]"
      />
    </div>
  )
}
