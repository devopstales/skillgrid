# Findings — embed-visual-companion

Consolidated evidence for the topic. The blueprint cites this file for every design
decision that rests on a researched fact.

## Reference: grill-with-ui (github.com/jasonku09/grill-with-ui)

Plain Node (no deps, no build) + one HTML file. A "grilling" design interview moved out of
the terminal onto a local browser page.

- **Transport: polling, not WebSocket.** The page polls `GET /state` every 1s
  (`setTimeout(poll, 1000)` in `page.html`). No WS anywhere.
- **Two files, split ownership.** `state.json` is written **only by the agent** (questions,
  recommendations, thread replies, statuses, `agent.status`); `events.jsonl` is appended
  **only by the web app**, one line per Send. No locks, no half-read JSON. The server serves
  the last-good parsed `state.json`.
- **Server IS the monitor.** The serve process's stdout (or a blocking `wait --after N`
  command) is what wakes the agent (Claude's Monitor tool; `wait` mode for agents without it).
- **Staging + one Send.** The user stages actions (pick option, free text, thread message,
  defer, reopen) in the browser; they survive a reload; one "Send to Agent" (⌘↩) ships them
  as ONE event line = ONE agent turn. Send is disabled while the agent is working.
- **Per-question discussion threads** in a right panel, labeled by question id, never lose
  session context (one Claude session under the hood).
- **Visualize.** The agent (via a subagent, per `visual-brief.md`) writes `visual.html`; the
  page shows it in `<iframe sandbox="allow-scripts">` (NO `allow-same-origin`); only a
  version bump reloads the iframe; marked "Out of date" when decisions change; each redraw
  has a version number + one-line change note.
- **Resume.** `pending --session` replays every `events.jsonl` line with `seq > agent.handled`;
  `agent.handled` is written each turn. `serve` remembers its port in `server.json` so an open
  tab resumes polling after a restart (falls back to ephemeral).
- **State lives under the user's home** (keyed by the git common root so all worktrees share
  it), never in the repo.
- **Design doc output.** Finish writes an exhaustive `docs/<slug>-design.md` with a Terms
  section (_Avoid_ lists) and "Locked decisions" (durable, three gates) vs "Routine choices".

**Adopted:** polling transport; structured question model (question + lettered options + a
recommended option + rationale); durable + versioned decision trail; durable/resumable state;
sandboxed visual iframe (deferred to a follow-up in this blueprint).

## Reference: oh-my-opencode-slim `/interview` (github.com/alvinunreal/oh-my-opencode-slim)

`/interview` opens a localhost browser UI to refine a feature idea inside the same OpenCode
session, and writes a markdown spec in the repo.

- **Transport: polling, explicitly "not realtime."** Runtime interview state is in-memory;
  the **markdown file is the durable artifact**.
- **Two modes.** *Per-session* (default — each process runs its own server, random port,
  in-memory state) and *dashboard* (opt-in — a **dumb aggregator** server on a fixed port,
  e.g. 43211, that receives state pushes from all sessions, serves the UI, and relays answers
  back to the right session). **Auto-failover:** if the dashboard dies, the next process claims
  the port and **rebuilds state from the `.md` files on disk** using YAML frontmatter
  (`sessionID`, `baseMessageCount`, `updatedAt`).
- **Sessions are smart, the dashboard is dumb.** Sessions drive the LLM locally and write the
  `.md`; the dashboard is a zero-cross-process-SDK aggregator.
- **Output:** `interview/<slug>.md` with `Current spec` (rewritten as it clarifies) +
  `Q&A history` (append-only). Suggested answers are clearly marked recommended; keyboard-driven
  selection; freeform custom answers. `/goal from <slug>` promotes the spec into the session goal.
- **Remote access:** 127.0.0.1 + Tailscale/Cloudflare/SSH tunnel.

**Adopted:** the "dumb dashboard that owns the port + rebuilds from durable on-disk state"
shape — in this blueprint the **Mnemonic store is the durable state** and the long-lived
`skillgrid serve` process **is** the dashboard, so it never "dies" the way a per-session Node
process does. Suggested-answers-marked-recommended is the core UX.

## Reference: the raw brainstorming companion (`.agents/skills/brainstorming/scripts/`)

- **Transport: raw RFC-6455 WebSocket + `fs.watch`.** `server.cjs` is a Node HTTP server with a
  hand-rolled WebSocket (frame encode/decode, `?key=` session gate, cookie mirror), a file
  watcher over `content/*.html`, an owner-PID watchdog + 4h idle timeout, and a best-effort
  browser open. The agent writes an HTML file to `screen_dir`; the server serves the newest one
  (wrapped in `frame-template.html` with `helper.js` injected); the browser posts clicks back
  over the WS, which are appended to `state/events`.
- **`frame-template.html` + `helper.js`** are the presentation contract (frame CSS, option/card
  markup, selection + WS client with reconnect + tombstone).
- **Durability: none.** Per-screen, ephemeral; `/tmp` sessions are deleted on stop.

**Not adopted:** the WebSocket, `fs.watch`, `frame-template.html`/`helper.js` verbatim reuse,
the `?key=` gate, and the "agent pushes one HTML screen" model. This is the weakest of the
three on durability and the only one using WS.

## Synthesis — why the Mnemonic decision bridge

- **Two of the three references use polling, not WS.** Only the raw companion uses WS — and it
  is the weakest on durability. So the embedded companion drops WS.
- **grill gives the structure** (questions + recommended option + rationale + durable/versioned
  trail), **oh-my-opencode-slim gives the architecture** (a dashboard that owns the port and
  rebuilds from durable on-disk state, with suggested answers marked recommended).
- **The durable on-disk state is the Mnemonic store, and the dashboard is `skillgrid serve`.**
  The agent already reads/writes this store via `mem_save`/`mem_update`/`mem_search` MCP tools,
  and the store already has owner / `status` / `visibility` / `acl_grants` / an append-only
  `observation_versions` history (`migrations/017_layered_memory_governance.sql`). A decision is
  a `type=decision` observation with a structured JSON `content`; the approval gate is a
  `content.state` field (`pending`/`answered`/`superseded`) because the governance `status` is
  closed to `active|superseded|archived` (`memory/governance.go:174`).
- **No migration, no new table, no new process, no new transport.** Only thin HTTP routes
  (`GET /mnemonic/decisions`, `POST /mnemonic/decisions/{id}/answer`, `GET /prototype/{id...}`) + a
  React view + a content convention. Reversible.

## Decision: the prototype function lives on-disk at `.skillgrid/prototype/`

The brainstorming flow has a *second* visual mechanism besides the interactive companion: the
`sketch` skill builds **throwaway standalone interactive HTML** (2–3 switchable variants, a
marked winner + rationale + constraints). The user chose **on-disk storage** for the companion's
visuals, at the concrete path:

```
.skillgrid/prototype/<topic>/<variant>.html
```

- **Separate from `.stitch/`** (which holds Stitch-generated design-system prototypes, read by the
  existing `prototypes.go` → `GET /prototypes`) and **separate from `sketch.dir`**
  (`{specs_root}/{topic}/sketches/`, where the `sketch` skill currently drops its HTML). The
  companion flow points `sketch` at `.skillgrid/prototype/<topic>/` so the dashboard can serve it.
- **Served by a new `GET /prototype/{id...}`** that reuses the existing `stitchFile`
  path-traversal guard shape, root = `.skillgrid/prototype/` (reject absolute ids, `..`, and
  anything escaping the root after `filepath.Clean`). A distinct subtree from the plural
  `/prototypes` route.
- **Rendered in the existing sandboxed iframe** (`sandbox="allow-scripts"`, no
  `allow-same-origin`) — the identical trust rule as the `.stitch/` Prototypes view and grill's
  `visual.html`. The decision's `content.visual` carries the id; a decision without `visual`
  renders no iframe. Only a re-authored variant reloads it.
- **Throwaway by design.** The durable record is the decision observation + its `observation_versions`
  history; the prototype HTML is scratch under `.skillgrid/` (expected gitignored).
