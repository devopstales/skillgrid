export function Stub({ title, note }: { title: string; note: string }) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-2">
      <h2 className="text-lg font-semibold text-zinc-100">{title}</h2>
      <p className="text-sm text-zinc-500">{note}</p>
    </div>
  )
}
