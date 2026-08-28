// Mnemonic OpenCode plugin — session lifecycle + memory-protocol injection.
// Copied into ~/.config/opencode/plugins/ by `skillgrid setup opencode`.
//
// Responsibilities (per openspec/changes/mnemonic/specs/agent-plugins/spec.md):
//   1. Memory Protocol injected via chat.system.transform at every chat turn.
//   2. Session auto-start: POST /sessions with the workspace directory on
//      session.created event.
//   3. Stale-index nudge: warn (never auto-run) when the code index is stale.
//   4. Auto-start: spawn `skillgrid serve` in background if GET /health fails.
//   5. Compaction recovery: GET /context and inject on
//      experimental.session.compacting.

// Overrideable base URL + token so dev can point the plugin at a non-default
// port without re-running `skillgrid setup`.
const BASE_URL = (
  process.env.SKILLGRID_MNEMONIC_HTTP_URL ?? "http://127.0.0.1:7438"
).replace(/\/$/, "");
const TOKEN = process.env.SKILLGRID_MNEMONIC_HTTP_TOKEN ?? "";

function req(method: string, path: string, body?: unknown): Promise<any> {
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (TOKEN) headers["Authorization"] = "Bearer " + TOKEN;
  return fetch(BASE_URL + path, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  })
    .then((res) => res.text())
    .then((text) => JSON.parse(text));
}

// The shared protocol markdown ships in the skillgrid repo and is mirrored into
// the user home by `skillgrid setup`. Read it from disk if present; fall back
// to a compact inline version so the plugin still works headless.
async function loadProtocol(): Promise<string> {
  try {
    const fs = (await import("node:fs/promises")).default;
    const path = (await import("node:path")).default;
    const os = (await import("node:os")).default;
    const candidates = [
      path.join(os.homedir(), ".config", "kilo", "shared", "memory-protocol.md"),
      path.join(os.homedir(), ".config", "opencode", "shared", "memory-protocol.md"),
    ];
    for (const c of candidates) {
      try {
        const text = await fs.readFile(c, "utf8");
        if (text.trim()) return text;
      } catch {
        /* try next */
      }
    }
  } catch {
    /* use inline */
  }
  return INLINE_PROTOCOL;
}

const INLINE_PROTOCOL = `# Mnemonic Memory Protocol

Mnemonic is skillgrid's local-first memory (SQLite + FTS5): \`mem_*\`, \`code_*\`, \`web_*\` tools over MCP and HTTP.

## When to Save (mandatory)
Call \`mem_save\` immediately after: a bug fix, an architecture/design decision, a non-obvious discovery, a config change, a new pattern, a learned user preference, or a user correction.

## Save Format
- title: verb + what (short, searchable)
- type: standing | preference | convention | decision | architecture | bugfix | pattern | config | correction | discovery | learning | lesson | session_log
- scope: project (default) | user | global
- topic_key (optional): stable key for upserts, e.g. architecture/auth-model
- content: **What** / **Why** / **Where** / **Learned** sections

## Session Protocol
1. \`mem_session_start\` on open
2. \`mem_save\` during work (dedups by hash within 24h, upserts by topic_key)
3. \`mem_search\` before starting work that may overlap prior sessions
4. \`mem_session_summary\` + \`mem_session_end\` before closing

## Code Index Ladder
\`code_status\` → \`code_index\` (if stale) → \`code_search\` → \`code_read\`.

## Web Cache
\`web_cache_lookup\` before remote research MCPs; \`web_cache_save\` immediately after Context7/Exa/DeepWiki/WebFetch returns; \`web_cache_search\` over cached research.`;

async function ensureServer() {
  try {
    await req("GET", "/health");
    return;
  } catch {
    /* not up */
  }
  const cp = await import("node:child_process");
  (cp as any).spawn?.("skillgrid", ["serve"], { detached: true, stdio: "ignore" });
}

export default async function MnemonicPlugin({
  directory,
  worktree,
}: {
  directory: string;
  worktree?: string | null;
}) {
  const dir = worktree || directory || process.cwd();
  const protocol = await loadProtocol();
  let sessionStarted = false;

  // Fire-and-forget: on session creation, ensure the HTTP server is up and
  // create a session record. Non-fatal — the MCP server is the primary
  // transport and does not depend on this HTTP record.
  async function onSessionCreated() {
    if (sessionStarted) return;
    sessionStarted = true;
    await ensureServer();
    try {
      await req(
        "POST",
        "/sessions?directory=" + encodeURIComponent(dir),
      );
    } catch {
      /* non-fatal */
    }
  }

  return {
    "chat.system.transform": (_input: any, output: any) => {
      const current: string = output.system ?? "";
      output.system =
        current +
        "\n\n" +
        protocol +
        "\n\n" +
        "If the code index is stale (code_status returns stale=true), prefer " +
        "code_search over repo-wide grep. Do NOT start a full re-index " +
        "automatically; ask first." +
        "\n";
    },

    event: async ({ event }: { event: { type: string } }) => {
      if (event.type === "session.created") {
        onSessionCreated();
      }
    },

    "experimental.session.compacting": async (_input: any, output: any) => {
      // Inject recent session context (from the HTTP API) into the
      // compaction prompt so nothing important is lost across summarisation.
      try {
        const ctx = await req("GET", "/context?limit=5");
        if (ctx && Array.isArray(ctx.sessions) && ctx.sessions.length) {
          const lines = ctx.sessions
            .map((s: any) => `- ${s.summary ?? s.id ?? "(no summary)"}`)
            .join("\n");
          (output.context ??= []).push(
            "## Mnemonic: recent sessions\n" + lines,
          );
        }
      } catch {
        /* non-fatal */
      }
    },
  };
}
