# Decision companion — the Mnemonic decision bridge

The decision companion is a **view over `type: decision` Mnemonic observations**.
The agent posts interview questions; the dashboard renders them; the user
answers; the agent reads the answer back. No WebSocket, no second process, no
new store — **polling is the transport**. Everything is durable in the Mnemonic
SQLite store (see [Serve — the web dashboard](./09-serve-dashboard.md) for the
UI that renders it).

This page is the **agent-side convention**: the content schema, the topic-key
and visibility rules, the poll loop, the answer semantics, and the
throwaway-prototype serving convention.

---

## The decision content schema (JSON in `observations.content`)

A decision is an ordinary observation whose `content` column is a structured
JSON payload. The schema:

```json
{
  "question": "Which layout for the homepage?",
  "options": [
    { "id": "a", "label": "Single column", "rationale": "cleanest reading" },
    { "id": "b", "label": "Two column" }
  ],
  "recommended": "a",
  "state": "pending"
}
```

Fields:

| Field | Required | Meaning |
|-------|----------|---------|
| `question` | yes | The question, in domain language. A missing/blank question makes the row malformed (flagged `parseError`, never a 500). |
| `options[]` | yes (≥1) | Each: `id` (stable, `a`/`b`/…), `label`, optional `rationale`. |
| `recommended` | no | The id of the option the agent recommends (highlighted in the UI). |
| `state` | yes | `pending` \| `answered` \| `superseded`. **This is a content field, not the governance `status`** — `status` is reserved for `active` \| `superseded` \| `archived`. |
| `answeredOption` | set by answer | The id of the option the user chose. |
| `answerNote` | set by answer | The user's answer note. |
| `updatedBy` | set by answer | `agent`, or `user:<name>` after an answer. |
| `visual` | no | A prototype id `<topic>/<variant>.html` served from `.skillgrid/prototype/` (see [Prototypes](#prototypes-the-visual-function)). When present, the decision card renders it in a sandboxed iframe. |

After a user answers, the content carries the answer inline:

```json
{
  "question": "Which layout for the homepage?",
  "options": [ { "id": "a", "label": "Single column" }, { "id": "b", "label": "Two column" } ],
  "recommended": "a",
  "state": "answered",
  "answeredOption": "b",
  "answerNote": "two column reads better",
  "updatedBy": "user:paladm"
}
```

---

## The topic-key and visibility convention

- **`type: decision`** — the observation type. The dashboard and the list route
  select on this.
- **`topic_key: interview/<slug>/<question-id>`** — a stable, hierarchical key.
  `<slug>` is the interview (e.g. `homepage-layout`), `<question-id>` is one
  question within it (e.g. `layout-1`). Reusing the same `topic_key` **upserts**
  the row (an in-place question update; it bumps the revision and appends a
  version).
- **`visibility: team`** — the companion's default. A fresh `mem_save` is
  `private` (governance default); the frozen `mem_save` tool has no visibility
  argument, so the agent widens it with `mem_share` (target `team`) after
  saving. The dashboard reader is a team-scoped identity: **private decisions
  are never listed**.

---

## The loop (polling, not realtime)

1. **Agent → user.** `mem_save` the decision (`type: decision`,
   `topic_key: interview/<slug>/<question-id>`, `content: {… state: "pending"}`),
   then `mem_share` it to `team`. Re-saving the same `topic_key` updates the
   question in place.
2. **Dashboard.** Polls `GET /mnemonic/decisions?state=pending` and renders the
   cards (question, options, the recommended highlight, and the visual when
   present).
3. **User → agent.** Click an option (optionally with a note) →
   `POST /mnemonic/decisions/{id}/answer` with body `{"optionId","note"}`. This
   is **write-gated** (Bearer `SKILLGRID_HTTP_TOKEN`); it sets `state: "answered"`,
   records the choice, and **appends a version** (durable, auditable).
4. **Agent.** Polls `mem_search` (its own frozen MCP surface) on a query
   consistent with the decision — a word of the question or the content — and
   proceeds when the returned content has `state == "answered"`.

There is no push: both sides poll. The round trip is guard-tested
(`TestDecisionRoundTrip` in `skillgrid-cli/internal/mnemonic/http/`): the
dashboard list and the agent's `mem_search` agree on the seeded decision, and
a *fresh* `mem_search` (a new service, simulating the agent polling later in a
resumed session) sees the answered state and the chosen option.

---

## Answer semantics

- **Pending → answered.** The first answer flips the convention `state` and
  records `answeredOption` / `answerNote` / `updatedBy` (`user:<name>`).
- **Re-answer appends a version.** Answering an already-answered decision is a
  `200`: the state stays `answered`, the *new* answer is recorded, and another
  `observation_versions` row is appended. The history is **append-only** — the
  **latest content carries the latest answer**, and prior answers are recoverable
  from the version history (governance). This is the same version-append path
  `mem_update` uses.
- **Superseded is agent-owned.** The `pending → superseded` transition is made by
  the agent (e.g. the question was answered elsewhere). The answer gate does not
  apply to a superseded decision.

Error contract for `POST /mnemonic/decisions/{id}/answer`:

| Status | When |
|--------|------|
| `200` | Answered (or re-answered). Body: `{"ok": true, "state": "answered"}`. |
| `400` | Missing/invalid `project` or `id` (non-integer), malformed body, or missing `optionId`. |
| `401` | Write-gate: no Bearer token while `SKILLGRID_HTTP_TOKEN` is set. |
| `404` | Unknown id (no such decision row in the project). |
| `409` | Content has no valid `state` (missing, malformed, or unrecognized), or the decision is `superseded`. |
| `422` | `optionId` is not in the decision's `options`. |

`GET /mnemonic/decisions` takes `?state=pending|answered|superseded|all` (default
all) and `?topic=` (a `topic_key` prefix filter). It never 500s on a malformed
row — the row is returned with `parseError: true`.

---

## Prototypes (the visual function)

When a question is clearer **shown** than told, the agent writes throwaway
standalone interactive HTML to

```
.skillgrid/prototype/<topic>/<variant>.html
```

and sets `content.visual` to `<topic>/<variant>.html`. The dashboard serves it at
`GET /prototype/<topic>/<variant>.html` (sandboxed to `.skillgrid/prototype/`,
traversal-guarded) and renders it in a **sandboxed iframe**
(`sandbox="allow-scripts"`, no `allow-same-origin`) inside the decision card.

The prototype is **throwaway by design** — the durable record is the decision
observation and its version history, not the HTML. It is a **distinct subtree**
from the Phase 7 `.stitch/` prototypes gallery (`GET /prototypes/{id}`):

| | Decision companion | Phase 7 gallery |
|---|--------------------|-----------------|
| Served at | `GET /prototype/{id...}` | `GET /prototypes/{id}` |
| Sandbox root | `.skillgrid/prototype/` | `.stitch/` |
| Purpose | Throwaway interview visuals | Reusable design prototypes |

The `sketch` skill's default output location is `sketch.dir`
(`{specs_root}/{topic}/sketches/`). For the companion flow, **point `sketch` at
`.skillgrid/prototype/<topic>/`** (set `sketch.dir` in `.skillgrid/config.yaml`)
so the variants land where the dashboard can serve them; the companion then
reads the variant id back into `content.visual`.

---

## Durability

Everything is in the Mnemonic SQLite store: it survives a server restart, is
versioned (`observation_versions`), owned, and governed (`visibility`,
`acl_grants`). A resumed session reads the same decisions — the agent's poll
(`mem_search`) and the dashboard's poll (`GET /mnemonic/decisions`) see the same
rows, so the companion survives a process boundary with no extra state.

---

## Next step

- [Serve — the web dashboard](./09-serve-dashboard.md) — the UI that renders the decision cards.
- [Memory and indexing](./05-memory-and-indexing.md) — the store the bridge reads and writes.
