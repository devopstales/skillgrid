// skillgrid memindex — OpenCode plugin (task 7-2)
//
// Responsibilities (per specs/http-api "Plugin usage pattern"):
//   1. auto-start "skillgrid serve" in the background if GET /health fails
//   2. POST /sessions on session start (workspace directory)
//   3. inject the Memory Protocol via chat.system.transform
//   4. on compaction: GET /context and inject the result before continuing
//   5. nudge (not auto-run) when the code index is stale
import { existsSync, spawn } from "node:fs";
import { join } from "node:path";

const BASE = process.env.SKILLGRID_MEMINDEX_HTTP ?? "http://127.0.0.1:7438";

async function health(): Promise<boolean> {
  try {
    const r = await fetch(BASE + "/health", { signal: AbortSignal.timeout(750) });
    return r.ok;
  } catch {
    return false;
  }
}

function ensureServer(): void {
  // Reuse a healthy server; only spawn when the health check fails.
  try {
    const child = spawn("skillgrid", ["serve"], { detached: true, stdio: "ignore" });
    child.unref();
  } catch {
    // skillgrid not on PATH — the agent can fall back to the MCP server.
  }
}

const MEMORY_PROTOCOL = `## MemIndex Memory Protocol (skillgrid-memindex)

You have local-first persistent memory backed by skillgrid (mem_* tools).

Memory:
- Session start/resume: mem_context first; mem_search when a topic recurs;
  mem_get_observation for full untruncated content.
- Save IMMEDIATELY after: bug fixes, decisions, non-obvious discoveries,
  config changes, established patterns, learned user preferences (mem_save,
  type: standing|preference|convention|decision|bugfix|lesson|correction|discovery|session_log).
- Reuse topic_key to upsert an evolving topic — do not create duplicate rows.
- Before ending a session: mem_session_summary with
  Goal / Discoveries / Accomplished / Next Steps / Relevant Files.

Code search ladder: code_status → code_search → code_read
(PREFER over grep/rg in unknown territory of a large repo; grep for exact identifiers.)

Web research cache:
- web_cache_lookup BEFORE Context7/Exa/DeepWiki/web fetch.
- web_cache_save IMMEDIATELY after the remote call returns (source + cache_key).
- stale:true → re-fetch and re-save.
`;

export default async (client) => {
  // 1. Plugin usage pattern: reuse healthy server, else auto-start.
  if (!(await health())) {
    ensureServer();
    // Give the server a moment; the first session call may 404 once.
    await new Promise((r) => setTimeout(r, 250));
  }

  client.on("session.start", async (evt) => {
    const dir = evt?.directory ?? process.cwd();
    // 2. Session created on start.
    try {
      const r = await fetch(BASE + "/sessions", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Workspace-Dir": dir,
        },
        body: JSON.stringify({ directory: dir }),
      });
      const s = await r.json();
      client.sessionID = s?.id ?? client.sessionID;
    } catch {
      // Non-fatal: memory tools still work via MCP.
    }
    // 5. Index nudge (not an auto-run full index).
    try {
      const r = await fetch(BASE + "/code/status?dir=" + encodeURIComponent(dir));
      if (r.ok) {
        const st = await r.json();
        if (st?.stale_hint) {
          client.messages?.append?.({
            role: "system",
            content: "MemIndex index hint: " + st.stale_hint,
          });
        }
      }
    } catch {
      // ignore
    }
  });

  // 3. Memory Protocol injected via chat.system.transform.
  client.messages?.transform?.((msg) => {
    if (msg?.role === "system") {
      return { ...msg, content: MEMORY_PROTOCOL + "\n\n" + (msg.content ?? "") };
    }
    return msg;
  });

  // 4. Compaction recovery: GET /context, inject before the agent continues.
  client.on("session.compact", async () => {
    try {
      const r = await fetch(BASE + "/context?limit=5");
      if (!r.ok) return;
      const data = await r.json();
      const sessions = data?.sessions ?? [];
      if (!sessions.length) return;
      const lines = sessions.map(
        (s) => `- ${s.id} (${s.status}) ${s.summary ?? "(no summary)"}`.trim()
      );
      client.messages?.append?.({
        role: "system",
        content:
          "MemIndex: context recovered after compaction (memindex sessions):\n" +
          lines.join("\n") +
          "\nUse mem_search / mem_get_observation to drill into these.",
      });
    } catch {
      // ignore
    }
  });

  client.on("session.end", async () => {
    if (client.sessionID) {
      try {
        await fetch(BASE + "/sessions/" + client.sessionID, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ operation: "end" }),
        });
      } catch {
        // ignore
      }
    }
  });
};
