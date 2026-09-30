export function Stub({ title, note }: { title: string; note: string }) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-2 font-mono">
      <h2 className="text-[15px] font-semibold text-ink-2">{title}</h2>
      <p className="text-[13px] text-ink-5">{note}</p>
    </div>
  )
}
