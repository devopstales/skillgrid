// @ts-nocheck
// OpenCode V2 bootstrap for the skillgrid native plugins (ADR-0032 V2 port).
//
// V1 plugins relied on two module-load globals that V2 no longer injects:
// `tool` (schema builder + tool wrapper) and `globalThis.opencode` (event bus).
// This module provides the V2-side replacements so the V1 tool/hook files stay
// byte-compatible:
//
//   - `makeTool()` — a JSON-Schema builder + `wrap()`. V1 files call a `tool`
//     global at module load; in V2 each file's V2 entrypoint does
//     `const { tool } = makeTool()` (the V1 named export is untouched). The
//     builder emits JSON Schema (V2's validation shape); `wrap` turns a V1
//     `{description, args, execute}` into a V2 `{name, description, input,
//     execute}` with a stringified result (V1's JSON.stringify contract).
//   - `globalThis.opencode.event.on` — a deferred event bus. Handlers register
//     at module load; dispatch defers until `setEventStream` wires the bus to
//     the V2 server event stream, then routes each event to the V1
//     `(input, output)` shape.
//
// Every failure is swallowed: a bootstrap problem degrades the plugin, it
// never breaks OpenCode's load.

// ── JSON-Schema builder (V2 `input` shape) ───────────────────────────────────
function schemaNode(kind, opts = {}) {
  const node = { type: kind, ...opts }
  node.describe = function (d) {
    if (typeof d === "string") node.description = d
    return node
  }
  node.optional = function () {
    node.optional = true
    return node
  }
  return node
}

/** Build the `tool` object a V1 file's module scope needs. Pure — no globals. */
export function makeTool() {
  return {
    schema: {
      string: (o) => schemaNode("string", o),
      number: (o) => schemaNode("number", o),
      boolean: (o) => schemaNode("boolean", o),
      array: (o) => schemaNode("array", o),
      object: (o) => schemaNode("object", o),
    },
    // V1 tool() wrapper: {description, args, execute} → V2 tool definition.
    // `name` is supplied by the caller (the map key). execute() stringifies
    // the handler's return so V2 sees a plain text result, matching V1's
    // JSON.stringify contract.
    wrap(definition) {
      const properties = {}
      const required = []
      for (const [key, value] of Object.entries(definition.args || {})) {
        const clean = { ...value }
        delete clean.describe
        delete clean.optional
        properties[key] = clean
        if (value.optional !== true) required.push(key)
      }
      return {
        name: definition.name,
        description: definition.description,
        input: {
          type: "object",
          properties,
          ...(required.length ? { required } : {}),
          additionalProperties: false,
        },
        async execute(input) {
          return JSON.stringify(await definition.execute(input))
        },
      }
    },
  }
}

// ── Deferred event bus ───────────────────────────────────────────────────────
const handlers = new Map()
let subscribeFn = null

function sessionIDOf(event) {
  return (
    event?.properties?.sessionID ||
    event?.sessionID ||
    event?.info?.sessionID ||
    ""
  )
}

async function dispatch(event) {
  const name = event?.type
  if (!name) return
  const list = handlers.get(name)
  if (!list || list.length === 0) return
  const input = event.properties || {}
  const output = { ...input }
  const sessionID = sessionIDOf(event)
  if (sessionID) output.sessionID = sessionID
  for (const handler of list) {
    try {
      await handler(input, output)
    } catch {
      // fail-open: one broken handler never blocks the stream
    }
  }
}

let controller = null

/** Wire the bus to the V2 server event stream. Returns the cleanup fn. */
export function setEventStream(subscribe) {
  subscribeFn = subscribe
  if (controller) controller.abort()
  controller = new AbortController()
  void (async () => {
    try {
      for await (const event of subscribe({ signal: controller.signal })) {
        await dispatch(event)
      }
    } catch {
      // stream closed or errored — cleanup aborts the controller
    }
  })()
  return () => controller?.abort()
}

globalThis.opencode = globalThis.opencode || {
  event: {
    on(name, handler) {
      if (!handlers.has(name)) handlers.set(name, [])
      handlers.get(name).push(handler)
      return () => {
        const list = handlers.get(name)
        if (list) {
          const i = list.indexOf(handler)
          if (i >= 0) list.splice(i, 1)
        }
      }
    },
  },
}
