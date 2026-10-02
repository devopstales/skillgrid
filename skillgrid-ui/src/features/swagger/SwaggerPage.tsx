import { apiUrl } from '../../lib/apiBase'

// SwaggerPage — embeds the Go-served Swagger UI bundle in an iframe.
// Dev must use an absolute API origin: relative /swagger/ is HTML-bypassed by
// the Vite proxy and would nest the whole SPA inside the iframe.
export function SwaggerPage() {
  const src = apiUrl('/swagger/')
  return (
    <div className="flex h-full flex-col">
      <div className="flex shrink-0 items-center gap-3 border-b border-edge-soft bg-bg px-4 py-2">
        <h1 className="text-[13px] font-semibold text-ink">API Documentation</h1>
        <a
          href={src}
          target="_blank"
          rel="noopener noreferrer"
          className="ml-auto text-[12px] text-info hover:underline"
        >
          Open in new tab ↗
        </a>
      </div>
      <iframe
        src={src}
        title="Skillgrid API Documentation (Swagger UI)"
        className="w-full flex-1 border-none bg-surface-800"
      />
    </div>
  )
}
