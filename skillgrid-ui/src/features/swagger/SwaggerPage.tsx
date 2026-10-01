// SwaggerPage — embeds the Go-served Swagger UI bundle (/swagger/) in an
// iframe inside the app layout. The thin header bar keeps the page labelled
// and offers a full-screen escape hatch.
export function SwaggerPage() {
  return (
    <div className="flex h-full flex-col">
      <div className="flex shrink-0 items-center gap-3 border-b border-edge-soft bg-bg px-4 py-2">
        <h1 className="text-[13px] font-semibold text-ink">API Documentation</h1>
        <a
          href="/swagger/"
          target="_blank"
          rel="noopener noreferrer"
          className="ml-auto text-[12px] text-info hover:underline"
        >
          Open in new tab ↗
        </a>
      </div>
      <iframe
        src="/swagger/"
        title="Skillgrid API Documentation (Swagger UI)"
        className="w-full flex-1 border-none"
      />
    </div>
  )
}
