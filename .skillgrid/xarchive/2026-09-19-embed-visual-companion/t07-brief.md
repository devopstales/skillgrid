### TICKET-07 — openapi + docs + MCP round-trip guard (from tasks.md)
### TICKET-07 — openapi + docs + MCP round-trip guard
- **Tracker ID:** TASK-013

- **Scope:** Add `/mnemonic/decisions`, `/mnemonic/decisions/{id}/answer`, `/prototype/{id...}` to `ui/openapi.yaml`. Add `docs/user-guide/10-decision-companion.md` (content schema + poll loop + `.skillgrid/prototype/` convention). Add the round-trip guard test: `GET /mnemonic/decisions` lists exactly the seeded decision; `mem_search("state pending")` is consistent with the list; the answer round-trip is visible to a fresh MCP `mem_search`.
- **Acceptance:** openapi.yaml validates (or the existing openapi test passes); the docs file exists and is referenced; `go test -count=1 ./internal/mnemonic/http/... -run 'TestDecisionRoundTrip'` passes (list ↔ `mem_search` consistency + answer visible to fresh MCP `mem_search`).
- **SATISFIES:** Scenario "the MCP round-trip is guard-tested" (GREEN).
- **Files:** `internal/mnemonic/http/ui/openapi.yaml`, `docs/user-guide/10-decision-companion.md` (new), `internal/mnemonic/http/decisions_test.go` (add round-trip test).
- **Size:** ~200 (S)
- **Blocks:** none
- **Blocked by:** TICKET-06 (openapi documents all 3 routes incl. `/prototype` from T06; round-trip test needs the answer route T02 + the stable route set). Run after T06 so the openapi reflects the final route set.
- **Fails-when:** `go test -count=1 ./internal/mnemonic/http/...` exits non-zero or the openapi test fails.

### Blueprint Task 4 (full)
### Task 4: openapi + docs convention + MCP round-trip guard

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/http/ui/openapi.yaml` (add `/mnemonic/decisions` + `/mnemonic/decisions/{id}/answer`)
- Create: `docs/user-guide/10-decision-companion.md` (content schema + the poll loop)
- Create: `skillgrid-cli/internal/mnemonic/http/usermanual_decisions_test.go` (doc-existence guard, mirror `usermanual_phase7_test.go`)
- Create: `skillgrid-cli/internal/mnemonic/mcp/decision_roundtrip_test.go` (MCP round-trip guard)

**Interfaces:**
- Consumes: existing `mem_save`/`mem_search` MCP tools (frozen).
- Produces: openapi entries; `TestDecisionsUserManual`; `TestDecisionMCPRoundTrip`.

**SATISFIES:** scenarios `decision-docs-exist` + `decision-mcp-roundtrip`.

- [ ] **Step 1: Write the failing tests**

```go
// usermanual_decisions_test.go
func TestDecisionsUserManual(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "docs", "user-guide", "10-decision-companion.md"))
	if err != nil { t.Fatalf("missing decision-companion doc: %v", err) }
	for _, marker := range []string{"type: decision", "state: pending", "recommended", "mem_save", "GET /mnemonic/decisions"} {
		if !strings.Contains(string(data), marker) { t.Errorf("doc missing marker %q", marker) }
	}
}

// decision_roundtrip_test.go (in mcp pkg) — save a decision via mem_save, search it back, parse content.
func TestDecisionMCPRoundTrip(t *testing.T) {
	// (adapt to the package's existing MCP test harness — see single_open_test.go / e2e_memory_ext_test.go)
	// 1. mem_save{type:decision, topic_key:interview/demo/rt, visibility:team, content:`{"question":"Q","options":[{"id":"a","label":"A"}],"recommended":"a","state":"pending"}`}
	// 2. mem_search{query or topic_key} → find the row
	// 3. parse content JSON → assert question/options/recommended/state present
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/http/ -run TestDecisionsUserManual -v && go test ./internal/mnemonic/mcp/ -run TestDecisionMCPRoundTrip -v`
Expected: FAIL — doc missing, MCP test not written.

- [ ] **Step 3: Write implementation (doc + openapi + MCP test)**

`docs/user-guide/10-decision-companion.md` — the convention:

````markdown
# Decisions — the Mnemonic decision companion

The companion is a **view over `type=decision` Mnemonic observations**. The agent
posts interview questions; the dashboard renders them; the user answers; the agent
reads the answer back. No WebSocket, no second process, no new store — polling is the
transport.

## The decision content schema (JSON in `observations.content`)

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
- `question` (required) — the question, in domain language.
- `options[]` (required, ≥1) — `id` (stable, `a`/`b`/…), `label`, optional `rationale`.
- `recommended` (optional) — the id of the option the agent recommends (highlighted in the UI).
- `state` (required) — `pending` | `answered` | `superseded`. **This is a content field, not the governance `status`** (`status` is reserved for `active|superseded|archived`).
- `answeredOption`, `answerNote` — set by the user's answer (the dashboard writes these).
- `updatedBy` — `agent` or `user`.
- `visual` (optional) — the prototype id `<topic>/<variant>.html`, served from `.skillgrid/prototype/` (see below). When present, the decision card renders it in a sandboxed iframe.

## The loop (polling, not realtime)

1. **Agent → user:** `mem_save` an observation: `type: decision`, `topic_key: interview/<slug>/<question-id>`, `visibility: team`, `content: {…state:pending}`. Reusing the same `topic_key` **upserts** the row (in-place question update; bumps the version history).
2. **Dashboard** polls `GET /mnemonic/decisions?state=pending` and renders the cards.
3. **User → agent:** click an option (or add a note) → `POST /mnemonic/decisions/{id}/answer {optionId, note}` → sets `state: answered`, records the choice, **appends a version** (durable, auditable trail).
4. **Agent** polls `mem_search` on the decision's `topic_key` (or the activity feed) and proceeds when `content.state == answered`.

## Prototypes (the visual function)

When a question is clearer *shown* than told, the agent writes throwaway standalone
interactive HTML to `.skillgrid/prototype/<topic>/<variant>.html` (via the existing `sketch`
skill, pointed at this dir for the companion flow) and sets `content.visual` to
`<topic>/<variant>.html`. The dashboard serves it at `GET /prototype/<topic>/<variant>.html`
(sandboxed to `.skillgrid/prototype/`, traversal-guarded) and renders it in a sandboxed
iframe (`sandbox="allow-scripts"`, no `allow-same-origin`) inside the decision card. The
prototype is throwaway by design; the durable record is the decision + its version history.
`sketch.dir` (the skill's default `{specs_root}/{topic}/sketches/`) is for standalone sketch
work; the companion uses `.skillgrid/prototype/` so the dashboard can serve it.

## Durability

Everything is in the Mnemonic SQLite store: it survives a server restart, is
versioned (`observation_versions`), owned, and governed (`visibility`, `acl_grants`).
A resumed session reads the same decisions.
````

Add the two openapi paths (mirror the `/mnemonic/memories/{id}/status` shape) and write the MCP round-trip test.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/http/ -run 'TestDecision' -v && go test ./internal/mnemonic/mcp/ -run TestDecisionMCPRoundTrip -v`
Expected: PASS.

- [ ] **Step 5: Run full suite + build + UI build:check**

Run: `cd skillgrid-cli && go build ./... && go test ./... 2>&1 | tail -20 && cd ../skillgrid-ui && npm run build:check`
Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/http/ui/openapi.yaml skillgrid-cli/internal/mnemonic/http/usermanual_decisions_test.go skillgrid-cli/internal/mnemonic/mcp/decision_roundtrip_test.go docs/user-guide/10-decision-companion.md
git commit -m "docs(decisions): openapi + user-guide convention + MCP round-trip guard"
```

---

