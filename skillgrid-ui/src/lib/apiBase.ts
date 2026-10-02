// apiOrigin — in Vite dev the SPA is on :5173 while the Go API is on :7438.
// Absolute API URLs are required for iframe embeds (Swagger) so HTML Accept
// bypass on the Vite proxy does not serve the SPA shell into the iframe.
// In production the UI is same-origin with the Go server, so relative paths win.
export function apiOrigin(): string {
  if (import.meta.env.DEV) return 'http://127.0.0.1:7438'
  return ''
}

export function apiUrl(path: string): string {
  const p = path.startsWith('/') ? path : `/${path}`
  return `${apiOrigin()}${p}`
}
