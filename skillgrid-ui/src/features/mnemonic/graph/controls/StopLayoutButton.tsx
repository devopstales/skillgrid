// StopLayoutButton interrupts the running force simulation, freezing the graph
// in place. Shown only while the force layout is still optimizing.
interface Props {
  visible: boolean
  onStop: () => void
}

export function StopLayoutButton({ visible, onStop }: Props) {
  if (!visible) return null
  return (
    <button
      type="button"
      onClick={onStop}
      className="flex items-center gap-1.5 rounded-lg border border-amber-500/40 bg-amber-500/15 px-3 py-1 text-xs font-medium text-amber-200 hover:bg-amber-500/25"
      title="Stop the layout animation and freeze the graph in place"
    >
      <span className="inline-block h-2.5 w-2.5 rounded-sm bg-amber-300" aria-hidden />
      Stop Layout
    </button>
  )
}
