import { useEffect, useRef, useState } from 'react'

const TOKEN_KEY = 'skillgrid.httpToken'

// SettingsPage — the system settings view: the HTTP write token (stored in
// localStorage, attached by the decisions/tracker write routes) and a static
// About block. The token input is controlled; Save persists it and shows a
// brief confirmation.
export function SettingsPage() {
  const [token, setToken] = useState('')
  const [saved, setSaved] = useState(false)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Populate the field from localStorage on mount (the SPA cannot read the
  // server's env, so the value the user saved here is the only source).
  useEffect(() => {
    setToken(localStorage.getItem(TOKEN_KEY) ?? '')
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current)
    }
  }, [])

  function save() {
    localStorage.setItem(TOKEN_KEY, token)
    setSaved(true)
    if (timerRef.current) clearTimeout(timerRef.current)
    timerRef.current = setTimeout(() => setSaved(false), 2000)
  }

  return (
    <div className="flex h-full flex-col gap-4 overflow-y-auto p-8 font-mono">
      <div>
        <h1 className="text-[14px] font-semibold text-ink">Settings</h1>
        <p className="mt-1 text-[13px] text-ink-5">Client-side preferences for this dashboard</p>
      </div>

      <section className="max-w-xl rounded-md border border-edge bg-card p-4">
        <h2 className="text-[13px] font-semibold text-ink">HTTP Token</h2>
        <p className="mt-1 text-[12px] text-ink-4">
          Used for API authentication to the Skillgrid backend.
        </p>
        <div className="mt-3 flex items-center gap-2">
          <input
            type="password"
            aria-label="HTTP Token"
            value={token}
            onChange={(e) => setToken(e.target.value)}
            className="min-w-0 flex-1 rounded border border-edge bg-inset px-3 py-1.5 text-[13px] text-ink-2 outline-none focus:border-accent/60"
          />
          <button
            type="button"
            onClick={save}
            className="shrink-0 rounded border border-accent-ink bg-accent/10 px-3 py-1.5 text-[12px] font-medium text-accent hover:bg-accent/20"
          >
            Save
          </button>
        </div>
        {saved && (
          <p role="status" className="mt-2 text-[12px] text-accent">
            Token saved
          </p>
        )}
      </section>

      <section className="max-w-xl rounded-md border border-edge bg-card p-4">
        <h2 className="text-[13px] font-semibold text-ink">About</h2>
        <ul className="mt-1 list-none space-y-0.5 text-[12px] text-ink-4">
          <li>Skillgrid Dashboard</li>
          <li>Terminal Ops design variant A</li>
          <li>Powered by Skillgrid</li>
        </ul>
      </section>
    </div>
  )
}
