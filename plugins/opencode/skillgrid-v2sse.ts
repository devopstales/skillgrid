// @ts-nocheck
// v2SSE — shared glue for the dual (v1/v2) skillgrid plugins. v1 plugins use the
// OpenCode v1 hook API; v2 (opencode >= 2.0) has no event bus or tool.execute.*
// hooks on PluginContext, so these plugins subscribe to the SDK SSE event stream
// inside their `setup` instead. This helper resolves the server base URL the
// same way the plugins already do (env overrides, falling back to the running
// server's own port, else the conventional 7438), and returns an async
// generator that yields raw SSE event objects.
//
// Import as:  import { v2SSE } from "./skillgrid-v2sse"
//   (the `./` keeps it relative, so it resolves under both the repo plugins/
//   dir and the installed ~/.config/opencode/plugins/ dir — bare specifiers
//   would require a package.json "imports" map we don't have.)

const BASE_URL =
  process.env.SKILLGRID_CHECKPOINT_URL ||
  process.env.SKILLGRID_MNEMONIC_HTTP_URL ||
  "http://127.0.0.1:7438"

// opencode sets these on spawned child processes (plugins/tools run in-process
// of the server); the conventional mnemonic HTTP port is the fallback.
function sseBaseUrl() {
  if (process.env.SKILLGRID_CHECKPOINT_URL) return process.env.SKILLGRID_CHECKPOINT_URL
  if (process.env.SKILLGRID_MNEMONIC_HTTP_URL) return process.env.SKILLGRID_MNEMONIC_HTTP_URL
  const port = process.env.OPENCODE_SERVER_PORT || process.env.OPENCODE_PORT
  if (port) return "http://127.0.0.1:" + port
  return BASE_URL
}

// v2SSE() returns an async generator yielding one event object per SSE message.
// It re-subscribes with a backoff if the stream ends unexpectedly, and stops
// when the AbortSignal is aborted. Fail-open: any connect error is swallowed
// (the plugins treat the stream as advisory).
export async function* v2SSE(signal) {
  const { createOpencodeClient } = await import("@opencode-ai/sdk/v2")
  const client = createOpencodeClient({
    baseUrl: sseBaseUrl(),
    throwOnError: false,
  })
  let delay = 500
  while (!signal.aborted) {
    try {
      const { stream } = await client.event.subscribe({ signal })
      for await (const event of stream) {
        if (signal.aborted) return
        yield event
      }
      // stream ended cleanly — fall through and reconnect
      delay = 500
    } catch (e) {
      if (signal.aborted) return
      await sleep(delay)
      delay = Math.min(delay * 2, 15000)
    }
  }
}

function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms))
}
