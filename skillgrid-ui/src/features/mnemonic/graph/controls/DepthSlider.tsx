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
    <div className="rounded-lg border border-slate-700 bg-slate-900/80 p-2">
      <div className="mb-1 flex items-center justify-between text-[11px] text-slate-400">
        <span className="font-semibold uppercase tracking-wide">Depth</span>
        <span className="text-violet-400">{value} hops</span>
      </div>
      <input
        type="range"
        min={min}
        max={max}
        step={1}
        value={value}
        onChange={(e) => onChange(Number(e.target.value))}
        className="w-40 accent-violet-500"
      />
    </div>
  )
}
