# Blueprint: mnemonic memory checkpoint

**Status:** approved
**Tier:** T2
**Build shape:** Tracer thread — the door check is whether a Cursor `stop` follow-up produced by a server-gated claim actually makes the host agent write typed observations and a summary. Task 5 closes that path end-to-end (claim route → hook → follow-up) before the index, privacy, plugin, and UI work widen it.

**Goal:** Make the host agent the memory observer. Every few tool calls the server decides a checkpoint is due, hands the agent a bounded prompt, and the agent writes typed observations and a session summary through the existing `mem_save` / `mem_session_summary` tools. Session start gets a lean memory index (recent summaries + observation titles) instead of only the last step. `<private>` text is never stored. The Sessions view shows observations next to tool calls, live.

**Architecture:** Go HTTP server (`skillgrid serve`, 127.0.0.1:7438) owns gating, prompt rendering, privacy stripping, and project resolution. A new pure package `internal/mnemonic/checkpoint` holds the decision and rendering logic. Harness hooks (Cursor `hooks.json` → `~/.skillgrid/hooks/*.sh|*.js`; OpenCode/Kilo plugin) are thin transports that fail open. The React UI consumes the existing SSE `activity` frame.

**Tech Stack:** Go 1.22+, SQLite (modernc, in-repo migrations), Node hook scripts (no npm deps), TypeScript plugin for OpenCode/Kilo (loaded by the harness, no build step), React + Vitest.

**Spec:** `.skillgrid/specs/2026-10-02-mnemonic-memory-checkpoint/briefing.md`, `acceptance.feature`, `adr.md` (ADR-0022).

## Terms

- **Memory Checkpoint** — a moment when the host agent is asked to compress recent tool events into typed observations and a summary.
- **Checkpoint Claim** — the server-side gate: `POST /sessions/{id}/checkpoint/claim` returns `due` plus the rendered prompt; a claim resets the cooldown for that session.
- **Memory Index** — the session-start block listing recent session summaries and observation titles under a token cap.
- **Private Span** — text between `<private>` and `</private>`; stripped before any write.
- **Digest** — the structured list of new tool events (action, tool, path, command) placed inside the checkpoint prompt. Never contains tool output or previews.

## Must-Haves

### Truths (observable after the change)

- A Cursor session that records five tool calls while `skillgrid serve` runs receives one follow-up message at the next `stop`; the agent then writes observations and a summary, and no further follow-up arrives for ten minutes.
- With no server running, every hook exits 0 within its timeout and the agent is never blocked.
- `skillgrid prime` in this repository prints a `## Memory` section whose session ids match the ones `GET /sessions?directory=<repo>` returns.
- Text inside `<private>…</private>` cannot be found in `observations`, `sessions.summary`, `prompts`, or `session_events.payload`.
- The Sessions view shows a new observation row within one SSE poll of `mem_save` without a page reload.

### Artifacts

- `skillgrid-cli/internal/mnemonic/checkpoint/{checkpoint.go,digest.go,prompt.go}` + tests
- `skillgrid-cli/internal/mnemonic/store/migrations/049_session_checkpoint.sql`
- `skillgrid-cli/internal/mnemonic/http/checkpoint.go` (+ `checkpoint_test.go`)
- `skillgrid-cli/internal/mnemonic/memory/privacy.go` (+ test)
- `skillgrid-cli/internal/mnemonic/session_inject/index.go` (+ test)
- `hooks/tool-call-capture.js` (`checkpoint` mode), `hooks/cursor-session-end.sh`, `plugins/cursor/hooks-cursor.json`
- `plugins/opencode/skillgrid-checkpoint.ts`, `plugins/kilo/skillgrid-checkpoint.ts`
- `skillgrid-ui/src/features/sessions/ToolTimeline.tsx`, `api.ts`, `skillgrid-ui/src/features/mnemonic/SessionsPage.tsx`
- `skillgrid-cli/internal/mnemonic/http/openapi.yaml`, `docs/user-guide/04-hooks.md`, `docs/user-guide/05-memory.md`

### Key links

- `handleToolCallCreate` → `EnsureSession` → `RunHook(HookPostToolUse)` already runs on every tool call; checkpoint state is read from the same `sessions` row, so no new write path is added on the hot path.
- `memory.Service.SaveWithAction` and `SessionSummary` bump `sessions.last_memory_write_at`; `Decide` counts `session_events` newer than that timestamp. Without this link the claim never resets.
- `projectFromRequest` must resolve `directory` through `project.Resolve` (the same function `skillgrid prime` uses). Without this link hooks and prime keep writing to different SQLite files.
- `tool-call-capture.js checkpoint` is invoked from `cursor-session-end.sh` only when `$1 == stop`; the Cursor `stop` entry carries `loop_limit: 2`. Without the limit the follow-up could loop.

### One-way doors

- ⚠ one-way: migration `049_session_checkpoint.sql` adds two nullable columns to `sessions` (`checkpoint_claimed_at`, `last_memory_write_at`). Additive, no data rewrite, but it is a schema migration and executes on first open. Named in the approved briefing (2026-10-02); the executor confirms with the user before running Task 2's step 4 the first time against a real store.
- No dependency additions. No deletions of public routes or tools.

## Global Constraints

- Fail open (ADR-0021): every hook and plugin exits 0 / returns silently on any error; checkpoint timeouts default to 1500 ms (reuse `SKILLGRID_POLICY_TIMEOUT_MS`).
- No external LLM. The agent that owns the session does the compression. `ExtractionLLM`, `DedupLLM`, `DreamLLM`, `layer.LLM`, `AskLLM` stay unwired.
- No new Go module or npm dependency.
- Spec-zone commits before code-zone commits; one work unit per commit, no AI attribution trailers; zone guard forbids mixing `.skillgrid/specs/**` with code in one commit.
- Prompt-injection boundary: the Digest is built from structured columns only (`action_type`, `tool_name`, `path`, `command` truncated to 120 chars, `result_status`). `payload` / `content_preview` text never enters a prompt.
- Token budget: index ≤ `mnemonic.inject.max_tokens` (800) using the existing `session_inject` token estimator; prompt digest ≤ 1500 chars.
- Existing heuristics (`promote.go`, `compact`) keep running unchanged — the checkpoint is additive.
- Go `flag` stops at the first positional: any new CLI flag is parsed after the positional lift already in `runSetup`.

## File Structure

```
skillgrid-cli/
  internal/mnemonic/
    checkpoint/
      checkpoint.go          Decide(state, cfg, now) (due, reason)
      checkpoint_test.go
      digest.go              BuildDigest(events, maxChars) string
      digest_test.go
      prompt.go              RenderPrompt(PromptInput) string
      prompt_test.go
    config/
      load.go                Checkpoint + Inject sections (+ yaml structs, defaults, validation)
      load_test.go           TestCheckpointConfig
    memory/
      privacy.go             StripPrivate(string) string
      privacy_test.go        TestStripPrivate
      service.go             CheckpointState, ClaimCheckpoint, last_memory_write_at bumps, StripPrivate at write sites
      service_checkpoint_test.go
    store/migrations/
      049_session_checkpoint.sql
    session_inject/
      index.go               RenderIndex(summaries, observations, IndexConfig) string
      index_test.go          TestRenderIndex
    http/
      checkpoint.go          handleCheckpointClaim
      checkpoint_test.go     TestCheckpointClaim_*, TestCheckpointPrompt_Content
      server.go              route + projectFromRequest directory fallback
      toolcalls.go           StripPrivate on preview; TestToolCalls_ResolvesProjectFromDirectory, TestToolCalls_PrivateSpan
      toolevents.go          observations[] in GET /sessions/{id}/events
      mnemonic_files.go      observations count in sessions list
      openapi.yaml
    setup/
      cursor.go              loop_limit on stop entry
      opencode.go, kilocode.go   install + register skillgrid-checkpoint plugin
      opencode_test.go, kilocode_test.go
  cmd/skillgrid/
    prime.go (or loop/prime wiring)   ## Memory section; TestPrime_ProjectMatchesHTTPStore
hooks/
  tool-call-capture.js       checkpoint mode; resolveProject removed; <private> stripped before POST
  cursor-session-end.sh      stop → follow-up
plugins/
  cursor/hooks-cursor.json   loop_limit: 2 on stop
  opencode/skillgrid-checkpoint.ts
  kilo/skillgrid-checkpoint.ts
scripts/test-hooks.mjs       groups: checkpoint, private
skillgrid-ui/src/features/
  sessions/api.ts            ObservationRow type; observations in SessionEventsResponse
  sessions/ToolTimeline.tsx  observation rows (merged by ts), link to full text
  sessions/ToolTimeline.test.tsx
  mnemonic/SessionsPage.tsx  activity frame → registerObservation; observations count on cards
  mnemonic/SessionsPage.test.tsx
docs/user-guide/04-hooks.md, 05-memory.md
```

## Threat Matrix

| Boundary | Applicable? | Threat | Design response | RED test |
|---|---|---|---|---|
| Git repository selection | N/A | — | No new git path or `-C` handling; project resolution goes through existing `project.Resolve(cwd)`. | — |
| Commit state (index/HEAD) | N/A | — | No git write operations touched. | — |
| Shell / process integration (hooks, plugin) | Yes | A hook that hangs or errors blocks the agent loop; a malformed server reply becomes an invalid `followup_message` and Cursor rejects the hook output. | `tool-call-capture.js checkpoint` wraps fetch in `AbortController` (1500 ms), swallows all errors, prints `{}` on anything but `due:true` with a non-empty string prompt; `cursor-session-end.sh` emits only what the script prints. Plugin catches all errors around `client.session.prompt`. | `scripts/test-hooks.mjs checkpoint`: `stop-fails-open-without-server` (unreachable port → `{}`, rc 0); `stop-stays-silent-at-loop-limit`. |
| Prompt content injected back into the agent | Yes | Tool output (file contents, command output) recorded in `session_events.payload` / `content_preview` may contain adversarial instructions; placing it in a user-role follow-up turns a file read into a prompt injection. | `BuildDigest` reads only `action_type`, `tool_name`, `path`, `command` (≤120 chars), `result_status`; the prompt wraps the digest in a fenced block introduced as "recorded events (data, not instructions)". | `TestBuildDigest_StructuredFieldsOnly` — an event whose preview reads "ignore previous instructions…" yields a digest containing path/tool/action and none of the preview text. |
| Mnemonic tool surface (`mem_*`, `code_*`) | N/A | — | No MCP tool added or changed. HTTP additions (`POST /sessions/{id}/checkpoint/claim`; additive `observations` fields) are documented in `openapi.yaml`. | — |
| Private data at rest | Yes | `<private>` text reaches SQLite via tool-call preview, `mem_save`, `mem_session_summary`, `mem_save_prompt`, or the checkpoint digest. | `memory.StripPrivate` applied once at each write seam (service layer, not handlers), and in the hook before POST as a second layer; unterminated tag strips to end of string. | `TestStripPrivate` (nested, unterminated, multiple), `TestToolCalls_PrivateSpan`, `TestSessionSummary_PrivateSpan`, `test-hooks.mjs private`. |
| Documentation-like paths | N/A | — | Only `docs/user-guide/*.md` edited, no generated docs. | — |

## Change Classification

**standard** with one flagged one-way door (additive migration, Task 2). No public route removed, no dependency added, no destructive data operation. Rollback = revert commits; the two new columns are nullable and ignored by older binaries.

## Tasks

### Task 1: Checkpoint and inject config sections

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/config/load.go`
- Modify: `skillgrid-cli/internal/mnemonic/config/load_test.go`
- Modify: `docs/user-guide/05-memory.md` (config keys table)

**Interfaces**

- Consumes: existing `Config` struct and `yaml` section structs in `load.go` (`Extraction`, `Dedup`, `Improvement` pattern).
- Produces:
  ```go
  // Checkpoint is the mnemonic.checkpoint section (ADR-0022).
  type Checkpoint struct {
      Enabled         bool          // default true
      MinEvents       int           // default 5, must be >= 1
      Cooldown        time.Duration // default 10m, must be > 0
      MaxObservations int           // default 5, must be >= 1
  }
  // Inject is the mnemonic.inject section (session-start Memory Index).
  type Inject struct {
      Summaries    int // default 5, >= 0
      Observations int // default 20, >= 0
      MaxTokens    int // default 800, >= 100
  }
  // Config gains: Checkpoint Checkpoint; Inject Inject
  ```
  Yaml: `mnemonic.checkpoint: {enabled, min_events, cooldown_minutes, max_observations}`, `mnemonic.inject: {summaries, observations, max_tokens}`.
- Seam: `config.Load` is already called by `serve` and `prime`; both read the new sections from the same struct.
- Deletion test: nothing deleted.
- Adapters: none.

**SATISFIES:** checkpoint-and-index-config / defaults-apply-without-keys, invalid-value.

**Steps**

1. Write `TestCheckpointConfig` in `load_test.go`: (a) a config with no `checkpoint`/`inject` keys yields the defaults above; (b) `min_events: 0` and `cooldown_minutes: -1` return an error naming the key (`mnemonic.checkpoint.min_events`). Run `go test ./internal/mnemonic/config -run TestCheckpointConfig` → fails to compile.
2. Add the two structs, yaml section structs, defaults in the loader, and validation returning `fmt.Errorf("mnemonic.checkpoint.min_events must be >= 1 (got %d)", v)`.
3. Run the test → passes. Run `go test ./internal/mnemonic/config` → all pass.
4. Add the keys to the config table in `docs/user-guide/05-memory.md` with defaults.
5. Commit: `feat(mnemonic): add checkpoint and inject config sections`.

### Task 2: Session checkpoint state in the store ⚠ one-way

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/store/migrations/049_session_checkpoint.sql`
- Modify: `skillgrid-cli/internal/mnemonic/memory/service.go`
- Create: `skillgrid-cli/internal/mnemonic/memory/service_checkpoint_test.go`

**Interfaces**

- Consumes: `sessions` table (048 is the latest migration), `session_events(session_id, timestamp)`, existing `SaveWithAction(ctx, SaveInput) (SaveResult, error)`, `SessionSummary(ctx, sessionID, summary string) error`.
- Produces:
  ```sql
  -- 049_session_checkpoint.sql
  ALTER TABLE sessions ADD COLUMN checkpoint_claimed_at TEXT;
  ALTER TABLE sessions ADD COLUMN last_memory_write_at TEXT;
  ```
  ```go
  type CheckpointState struct {
      SessionID       string
      Exists          bool
      EventsSinceWrite int       // session_events rows with timestamp > last_memory_write_at (all rows when NULL)
      LastClaimedAt   time.Time  // zero when NULL
      LastWriteAt     time.Time  // zero when NULL
  }
  func (s *Service) CheckpointState(ctx context.Context, sessionID string) (CheckpointState, error)
  func (s *Service) ClaimCheckpoint(ctx context.Context, sessionID string, at time.Time) error
  // internal: touchMemoryWrite(ctx, sessionID, now) called by SaveWithAction (when in.SessionID != "") and SessionSummary.
  ```
- Seam: `memory.Service` methods, called by the HTTP handler in Task 4.
- Deletion test: nothing deleted.
- Adapters: none; older binaries ignore the new columns.

**SATISFIES:** checkpoint-claim-gating / claim-resets-after-summary (store half), claim-for-unknown-session (store half).

**Steps**

1. Write `TestCheckpointState_CountsEventsSinceLastWrite` and `TestCheckpointState_UnknownSession`: open a temp store, ensure a session, insert 6 `session_events` rows, assert `EventsSinceWrite == 6`; call `SessionSummary` and assert `EventsSinceWrite == 0`; `ClaimCheckpoint` sets `LastClaimedAt`; unknown id → `Exists == false`, no error. Run → compile failure.
2. Add the migration file (follow the numbering and header comment style of `048_*.sql`) and the three methods. `touchMemoryWrite` runs `UPDATE sessions SET last_memory_write_at = ? WHERE id = ?` inside the existing save/summary transactions where one exists, else as a follow-up statement.
3. Run the new tests and `go test ./internal/mnemonic/memory ./internal/mnemonic/store` → pass. Verify the migration applies on a copy of a real store: `cp ~/.skillgrid/mnemonic/aiskillgrid.sqlite /tmp/cp.sqlite` and open it through a one-off test or `skillgrid mem stats` with `MNEMONIC_PROJECT` pointing at the copy — confirm before this step that the user approved the one-way door.
4. `go vet ./...` clean.
5. Commit: `feat(mnemonic): track checkpoint state per session`.

### Task 3: Pure checkpoint package — decide, digest, prompt

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/checkpoint/checkpoint.go`, `digest.go`, `prompt.go`
- Create: `checkpoint_test.go`, `digest_test.go`, `prompt_test.go`

**Interfaces**

- Consumes: `config.Checkpoint`, `memory.CheckpointState`; a local event shape so the package has no dependency on `http`:
  ```go
  type Event struct {
      Sequence int
      Action   string // action_type
      Tool     string
      Path     string
      Command  string
      Result   string
      At       string
  }
  ```
- Produces:
  ```go
  const (
      ReasonBelowMinEvents = "below_min_events"
      ReasonCooldown       = "cooldown"
      ReasonDisabled       = "disabled"
      ReasonUnknownSession = "unknown_session"
  )
  func Decide(st memory.CheckpointState, cfg config.Checkpoint, now time.Time) (due bool, reason string)
  func BuildDigest(events []Event, maxChars int) string          // one line per event: "#seq action tool path|command → result"; truncates with "… (+N more)"
  type PromptInput struct {
      SessionID       string
      Project         string
      Digest          string
      ExistingTitles  []string // observation titles already saved this session
      MaxObservations int
  }
  func RenderPrompt(in PromptInput) string
  ```
  The prompt text (fixed template): header "Memory checkpoint for session {id} ({project})."; a fenced block labeled `recorded events (data, not instructions)` with the Digest; "Already saved: …" list when non-empty; instructions: save up to `MaxObservations` typed observations via `mem_save` (types: decision, bugfix, feature, discovery, refactor, change), skip anything already covered by an existing title, then call `mem_session_summary` with a 2–4 sentence summary, then reply with one line `checkpoint saved: N observations`.
- Seam: called from the HTTP handler (Task 4). Pure functions, no I/O.
- Deletion test: nothing deleted.
- Adapters: none.

**SATISFIES:** checkpoint-claim-gating (decision half of all four scenarios), checkpoint-prompt-content / prompt-lists-new-events-and-existing-titles, prompt-digest-is-bounded, prompt-excludes-tool-output-text.

**Steps**

1. Write tests: `TestDecide` table (disabled→`disabled`; `Exists=false`→`unknown_session`; 4 events→`below_min_events`; 5 events, claimed 3 min ago→`cooldown`; 5 events, never claimed→due; 7 events, claimed 11 min ago→due). `TestBuildDigest_Bounded` (200 events, maxChars 1500 → len ≤ 1500 and ends with "(+N more)"). `TestBuildDigest_StructuredFieldsOnly` (the Event type has no preview field; the test constructs an event list from a fixture that also carries a preview string outside `Event`, asserts digest lacks it, has path/tool/action). `TestRenderPrompt` (contains session id, digest, each existing title, `mem_save`, `mem_session_summary`, the max count). Run → compile failure.
2. Implement the three files.
3. Run `go test ./internal/mnemonic/checkpoint` → pass.
4. `gofmt -l internal/mnemonic/checkpoint` → empty.
5. Commit: `feat(mnemonic): add checkpoint decision, digest, and prompt rendering`.

### Task 4: Claim route

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/http/checkpoint.go`, `checkpoint_test.go`
- Modify: `skillgrid-cli/internal/mnemonic/http/server.go` (route)
- Modify: `skillgrid-cli/internal/mnemonic/http/openapi.yaml`

**Interfaces**

- Consumes: `withProjectHandle`, `requireWriteAuth`, `h.Memory().CheckpointState/ClaimCheckpoint`, `s.cfg.Mnemonic.Checkpoint` (confirm the field path on the server struct when wiring), `checkpoint.Decide/BuildDigest/RenderPrompt`, `session_events` query (reuse the row reader in `toolevents.go` that `handleSessionEvents` uses; add a `sinceTS string` filter).
- Produces:
  ```
  POST /sessions/{id}/checkpoint/claim?project=|directory=   (body may also carry {"project"} / {"directory"})
  200 {"due": bool, "reason": "" | "below_min_events" | "cooldown" | "disabled" | "unknown_session", "prompt": string}
  ```
  `due:true` → the handler calls `ClaimCheckpoint(now)` before responding; `prompt` is non-empty only when due. Unknown session → 200 with `due:false, reason:"unknown_session"` (not 404: the hook must stay silent, not log an error).
- Seam: `s.mux.HandleFunc("POST /sessions/{id}/checkpoint/claim", s.requireWriteAuth(s.handleCheckpointClaim))` next to the other session routes.
- Deletion test: nothing deleted.
- Adapters: none.

**SATISFIES:** checkpoint-claim-gating / all four scenarios (HTTP level), checkpoint-prompt-content / all scenarios (HTTP level), checkpoint-and-index-config / disabled-checkpoint.

**Steps**

1. Write `TestCheckpointClaim_DueAndCooldown` (POST tool-calls ×5 through the existing route, claim → `due:true` + prompt listing them; claim again → `cooldown`; `SessionSummary` then 5 more calls after advancing the server clock — inject a `now func() time.Time` on the server for tests — → due again), `TestCheckpointClaim_UnknownSession`, `TestCheckpointClaim_Disabled`, `TestCheckpointPrompt_Content` (prompt contains existing observation titles saved via `Save`, and not the `content_preview` of a tool call). Run → fail.
2. Implement `handleCheckpointClaim`; add `now` field defaulting to `time.Now`.
3. Run `go test ./internal/mnemonic/http -run 'TestCheckpointClaim|TestCheckpointPrompt'` → pass; full `go test ./internal/mnemonic/http` → pass.
4. Document the route and response schema in `openapi.yaml`.
5. Commit: `feat(mnemonic): serve checkpoint claims for sessions`.

### Task 5: Cursor stop follow-up (door check)

**Files:**
- Modify: `hooks/tool-call-capture.js` (add `checkpoint` mode)
- Modify: `hooks/cursor-session-end.sh`
- Modify: `plugins/cursor/hooks-cursor.json`, `skillgrid-cli/internal/mnemonic/setup/cursor.go` (+ `cursor_test.go`)
- Modify: `scripts/test-hooks.mjs` (new `checkpoint` group)
- Modify: `docs/user-guide/04-hooks.md`

**Interfaces**

- Consumes: Cursor `stop` stdin `{conversation_id, generation_id, model, status, loop_count}`; claim route (Task 4); existing `SKILLGRID_MNEMONIC_URL` / port constants and `SKILLGRID_POLICY_TIMEOUT_MS` in the script.
- Produces:
  - `node tool-call-capture.js checkpoint` reads stdin, POSTs `/sessions/{conversation_id}/checkpoint/claim` with `{directory: cwd}` when `loop_count < 2`; prints `{"followup_message": prompt}` when `due === true && typeof prompt === "string" && prompt`; otherwise prints `{}`. Any error/timeout → `{}` and exit 0. Honors `SKILLGRID_CHECKPOINT_URL` override (used by tests to point at a fake server).
  - `cursor-session-end.sh stop` pipes stdin to the script in `checkpoint` mode and echoes its stdout; `cursor-session-end.sh sessionEnd` keeps the current behavior and never emits a follow-up.
  - Cursor `stop` hook entry gains `"loop_limit": 2` in both `hooks-cursor.json` and `cursorHookScripts` upsert in `setup/cursor.go` (the merge must update an existing skillgrid entry lacking the key).
- Seam: hook stdin/stdout contract; fake HTTP server inside `test-hooks.mjs` (Node `http.createServer`) returning scripted claim replies.
- Deletion test: nothing deleted.
- Adapters: none.

**SATISFIES:** cursor-stop-followup / stop-returns-followup-when-due, stop-stays-silent-at-loop-limit, stop-fails-open-without-server.

**Steps**

1. Add `caseCheckpoint()` to `test-hooks.mjs` (filter `checkpoint`): start a fake server on an ephemeral port that replies `{due:true, prompt:"Memory checkpoint…"}`; run the script with stdin `{conversation_id:"s1", loop_count:0}` and `SKILLGRID_CHECKPOINT_URL` set → stdout parses to `{followup_message}`, rc 0. Same with `loop_count:2` → `{}`. Server replying `{due:false, reason:"cooldown"}` → `{}`. URL pointing at a closed port → `{}` within 2 s, rc 0. Register `caseCheckpoint()` in the run-all list. Run `node scripts/test-hooks.mjs checkpoint` → fails.
2. Implement the `checkpoint` branch in `main()`, update `cursor-session-end.sh` (`case "$1" in stop) …`), add `loop_limit` in JSON and Go, extend `TestUpsertCursorHooks` to assert the stop entry has `loop_limit: 2` and that an existing entry without it is updated.
3. Run `node scripts/test-hooks.mjs checkpoint` → `Results: … 0 failed`; `go test ./internal/mnemonic/setup` → pass. Door check by hand: `skillgrid setup cursor`, start `skillgrid serve`, open a Cursor session in this repo, make five tool calls, stop → confirm the follow-up appears and observations land (`GET /sessions/{id}/events`). Record the result in `acceptance.feature` G4/G5 EVIDENCE.
4. Update `docs/user-guide/04-hooks.md`: "Memory checkpoints" subsection (what the follow-up is, `loop_limit`, how to disable via `mnemonic.checkpoint.enabled: false`).
5. Commit: `feat(hooks): ask the agent for a memory checkpoint at stop`.

### Task 6: One project resolver

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/http/server.go` (`projectFromRequest`)
- Modify: `skillgrid-cli/internal/mnemonic/http/toolcalls.go` (+ `toolcalls_test.go`)
- Modify: `hooks/tool-call-capture.js` (remove `resolveProject` and the active-project pin read; always send `directory`)
- Modify: `skillgrid-cli/cmd/skillgrid/prime*.go` test file (`TestPrime_ProjectMatchesHTTPStore`)

**Interfaces**

- Consumes: `project.Resolve(cwd)`, `s.svc.OpenForDirectory(directory)` / `openHandleForDir`.
- Produces: `projectFromRequest(r)` order: explicit `project` (query/body) → `directory` (query/body) resolved via `project.Resolve` → error "project or directory is required". Response bodies that echo `project` now show the resolved id.
- Seam: every handler already calls `projectFromRequest`; `tool-call-capture.js` sends `directory: process.cwd()` and no `project`.
- Deletion test: `resolveProject()` and the `active-project` pin read are deleted from `tool-call-capture.js`; `rg -n 'active-project|resolveProject' hooks/tool-call-capture.js` → no matches. The `prime` write of `active-project` stays (other readers may exist) — note in docs that hooks no longer depend on it.
- Adapters: none.

**SATISFIES:** one-project-resolver / prime-and-hooks-share-one-project, directory-without-project-param, prime-in-another-repository.

**Steps**

1. Write `TestToolCalls_ResolvesProjectFromDirectory` (POST with only `directory` of a temp git repo → 201 and `project` equals `project.Resolve(dir)`), and `TestPrime_ProjectMatchesHTTPStore` (run prime's project resolution for a temp repo and compare to the HTTP response `project` for the same directory). Run → fail.
2. Implement the fallback; delete `resolveProject` from the hook script; send `directory`.
3. Run the two tests plus `go test ./internal/mnemonic/http ./cmd/skillgrid` → pass. Manual: run a tool call from this repo with `skillgrid serve` running; `sqlite3 ~/.skillgrid/mnemonic/aiskillgrid.sqlite 'select count(*) from session_events'` grows while `skillgrid.sqlite` does not.
4. `node scripts/test-hooks.mjs` (all groups) → 0 failed.
5. Commit: `fix(mnemonic): resolve the project server-side for hook requests`.

### Task 7: Private spans never stored

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/memory/privacy.go`, `privacy_test.go` (check first whether `session_inject/privacy.go` already exports a stripper; if so, move/alias it here so there is one implementation)
- Modify: `skillgrid-cli/internal/mnemonic/memory/service.go` (`SaveWithAction`, `SessionSummary`, `SavePrompt`), `http/toolcalls.go`, `http/checkpoint.go` (digest input)
- Modify: `hooks/tool-call-capture.js`, `scripts/test-hooks.mjs` (`private` group)

**Interfaces**

- Produces:
  ```go
  // StripPrivate removes every <private>…</private> span (case-insensitive, nested-safe, unterminated → strip to end) and collapses the surrounding whitespace.
  func StripPrivate(s string) string
  ```
- Seam: called on `SaveInput.Title/Content`, the summary string, `PromptInput.Content`, `toolCallBody.ContentPreview` and `Command` before any write; JS strips preview/command before POST.
- Deletion test: nothing deleted.
- Adapters: none.

**SATISFIES:** private-spans-never-stored / tool-call-with-private-span, unterminated-private-tag, private-span-in-summary; checkpoint-prompt-content / prompt-omits-private-spans.

**Steps**

1. Write `TestStripPrivate` (table: single span, two spans, nested, unterminated, mixed case, no tag → unchanged), `TestToolCalls_PrivateSpan` (POST preview with a span → stored `payload` lacks the inner text), `TestSessionSummary_PrivateSpan`; add `casePrivate()` to `test-hooks.mjs` running the script in capture mode against the fake server from Task 5 and asserting the request body lacks the inner text. Run → fail.
2. Implement `StripPrivate` and call it at each seam; add the JS mirror (regex `/<private>[\s\S]*?(<\/private>|$)/gi`).
3. Run `go test ./internal/mnemonic/memory ./internal/mnemonic/http` and `node scripts/test-hooks.mjs private` → pass.
4. `rg -n 'StripPrivate' skillgrid-cli/internal` lists all four write seams plus the digest.
5. Commit: `feat(mnemonic): strip private spans before any memory write`.

### Task 8: Memory index at session start

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/session_inject/index.go`, `index_test.go`
- Modify: `skillgrid-cli/internal/mnemonic/loop/render.go` (`PrimeInput.MemoryIndex string`, rendered as `## Memory` when non-empty)
- Modify: `skillgrid-cli/cmd/skillgrid/prime.go` (fetch `RecentContext(ctx, cfg.Inject.Summaries)` and `RecentObservations(ctx, cfg.Inject.Observations)`, render)
- Modify: `docs/user-guide/05-memory.md`

**Interfaces**

- Consumes: `memory.Session{ID, Title, StartedAt, Summary}`, `memory.Observation{ID, Type, Title, CreatedAt, Pinned}`, `config.Inject`, the existing token estimator in `session_inject`.
- Produces:
  ```go
  type IndexConfig struct{ Summaries, Observations, MaxTokens int }
  // RenderIndex returns "" when both inputs are empty. Pinned observations first, then newest first.
  // Lines: "- #<id> <type> <title> · <YYYY-MM-DD> · ~<tok>" ; summaries: "- <session id short> <date>: <first sentence of summary>".
  // Footer: "Full text: mem_get_observation <id> · mem_timeline <id> · mem_search <query>".
  // Drops the oldest entries until the estimated token count <= MaxTokens.
  func RenderIndex(summaries []memory.Session, observations []memory.Observation, cfg IndexConfig) string
  ```
- Seam: `prime` already builds `PrimeInput`; `MemoryIndex` is one more field.
- Deletion test: nothing deleted.
- Adapters: none.

**SATISFIES:** memory-index-at-session-start / prime-lists-summaries-and-observation-index, index-respects-token-cap, index-absent-for-empty-project.

**Steps**

1. Write `TestRenderIndex` (pinned first; dates; footer; 60 observations with `MaxTokens: 200` → output under cap and newest retained; both empty → `""`), and extend the existing `RenderPrime` test so a non-empty `MemoryIndex` yields a `## Memory` section and an empty one yields none. Run → fail.
2. Implement `RenderIndex`, wire `PrimeInput.MemoryIndex`, fetch in `prime`.
3. Run `go test ./internal/mnemonic/session_inject ./internal/mnemonic/loop ./cmd/skillgrid` → pass. Manual: `skillgrid prime` in this repo prints `## Memory` with ids from `aiskillgrid`.
4. Document the section and the `mnemonic.inject` keys in `05-memory.md`.
5. Commit: `feat(mnemonic): inject a memory index at session start`.

### Task 9: OpenCode and Kilo checkpoint plugin

**Files:**
- Create: `plugins/opencode/skillgrid-checkpoint.ts`, `plugins/kilo/skillgrid-checkpoint.ts` (identical content; Kilo copy kept separate because `setup/kilocode.go` copies from `kiloPluginRel`)
- Modify: `skillgrid-cli/internal/mnemonic/setup/opencode.go`, `kilocode.go`, `opencode_test.go`, `kilocode_test.go`
- Modify: `docs/user-guide/04-hooks.md`

**Interfaces**

- Consumes: OpenCode plugin API (`export const SkillgridCheckpoint: Plugin = async ({ client, directory }) => ({ event: async ({ event }) => { … } })`), event `session.idle` with `event.properties.sessionID`; claim route.
- Produces: on `session.idle`, POST `/sessions/{sessionID}/checkpoint/claim` with `{directory}` (1500 ms timeout); when `due`, `client.session.prompt({ path: { id: sessionID }, body: { parts: [{ type: "text", text: prompt }] } })`. A per-session in-memory guard skips the claim while a prompted turn is still running (set on prompt, cleared on the next `session.idle`). All errors swallowed.
- Setup: `SetupOpenCode` copies `plugins/opencode/skillgrid-checkpoint.ts` to `~/.config/opencode/plugin/skillgrid-checkpoint.ts` and `SetupKiloCode` to `~/.config/kilo/plugin/skillgrid-checkpoint.ts` via `copyFromRepo`; `upsertPluginKey` is not needed for file-based plugins — confirm against the OpenCode docs during step 2 and, if a config entry is required, reuse `upsertPluginKey(cfgPath, "./plugin/skillgrid-checkpoint.ts", dryRun)`.
- Seam: file copy + harness plugin loading.
- Deletion test: nothing deleted.
- Adapters: none.

**SATISFIES:** opencode-kilo-idle-prompt / setup-installs-checkpoint-plugin, plugin-does-not-prompt-when-not-due, plugin-fails-open-without-server (manual G7).

**Steps**

1. Write `TestSetupOpenCode_CheckpointPlugin` and `TestSetupKilo_CheckpointPlugin` (temp HOME + repo root; after setup the plugin file exists; dry-run writes nothing). Run → fail.
2. Write the plugin and the copy calls.
3. Run `go test ./internal/mnemonic/setup` → pass. Manual G7: `skillgrid setup opencode`, run OpenCode in this repo with `skillgrid serve`, five tool calls, idle → prompt appears once; stop the server, idle again → nothing, no error in the OpenCode log. Record evidence.
4. Document the plugin in `04-hooks.md` (OpenCode/Kilo section).
5. Commit: `feat(setup): install the checkpoint plugin for OpenCode and Kilo`.

### Task 10: Observations in the session event feed

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/http/toolevents.go` (+ test), `mnemonic_files.go` (sessions list), `openapi.yaml`

**Interfaces**

- Produces: `GET /sessions/{id}/events` → `{project, sessionId, events, observations: [{id, type, title, created_at, tokens, pinned}]}`; sessions list items gain `observations: <count>` (count of non-deleted observations for the session). Both additive.
- Seam: existing handlers; `h.Memory()` queries.
- Deletion test: nothing deleted.
- Adapters: UI treats missing fields as empty (older servers).

**SATISFIES:** live-observations-in-sessions-view / observation-row-links-to-full-text (data half).

**Steps**

1. Extend `TestSessionEvents_*` (existing test file for `toolevents.go`) with a saved observation → appears in `observations` with its type/title; sessions list shows `observations: 1`. Run → fail.
2. Implement the queries.
3. `go test ./internal/mnemonic/http` → pass.
4. Update `openapi.yaml`.
5. Commit: `feat(mnemonic): include observations in session event feeds`.

### Task 11: Live observation rows in the Sessions view

**Files:**
- Modify: `skillgrid-ui/src/features/sessions/api.ts` (`ObservationRow`, `SessionEventsResponse.observations`)
- Modify: `skillgrid-ui/src/features/sessions/ToolTimeline.tsx` (+ `ToolTimeline.test.tsx`)
- Modify: `skillgrid-ui/src/features/mnemonic/SessionsPage.tsx` (+ `SessionsPage.test.tsx`)

**Interfaces**

- Consumes: `openActivityStream({onActivity})` frames `{id, ts, type, source, actor, severity, summary, sessionId}`; Task 10 response.
- Produces: `ToolTimeline` accepts `registerObservation: (h: ((o: ObservationRow) => void) | null) => void`; renders observation rows (icon `◆`, type chip, title) merged chronologically with tool rows; clicking a row navigates to the existing observation detail route (`/mnemonic/observations/:id` — confirm the existing path in the router during step 2). `SessionsPage` wires the `activity` frame to the registered handler when `sessionId` matches the selected session (or any when none selected) and increments the card's `observations` count. A disconnected stream shows the existing "live paused" state; no reconnect logic added.
- Seam: props + SSE callbacks already in place.
- Deletion test: nothing deleted.
- Adapters: `observations ?? []`, `observations ?? 0`.

**SATISFIES:** live-observations-in-sessions-view / observation-appears-live, observation-row-links-to-full-text, stream-disconnected.

**Steps**

1. Write Vitest cases: `ToolTimeline` renders an observation row between two tool rows by timestamp and links to the detail route; `SessionsPage` dispatches a mocked `activity` frame → row appears and the card count increments; `onError` → "live paused" indicator visible. Run `npm test -- ToolTimeline SessionsPage` → fail.
2. Implement.
3. `npm test` and `npm run lint` in `skillgrid-ui` → pass; `npm run build` → pass.
4. Visual check in the browser against `skillgrid serve` + `mem_save` from a shell; screenshot kept out of the repo.
5. Commit: `feat(ui): show observations live in the sessions timeline`.

## Self-Review

- Every requirement in `acceptance.feature` maps to a task: gating (2,3,4), prompt (3,4,7), cursor-stop (5), opencode-kilo (9), index (8), one-project (6), private (7), live UI (10,11), config (1,4). Gates G1–G14 and G3b each have a named test in a task.
- Every task names exact files, signatures, a failing-first test, and a conventional commit subject. No placeholders remain.
- Interfaces match the seams read in the code (`projectFromRequest`, `withProjectHandle`, `SaveWithAction`, `SessionSummary`, `RecentContext`, `RecentObservations`, `PrimeInput`, `openActivityStream` callbacks, `copyFromRepo`).
- Threat matrix rows each have a design response and a RED test or a stated N/A reason.
- Docs are folded into the tasks whose deliverable they describe (1, 5, 8, 9); no trailing docs task.
- Build shape is a tracer thread; Task 5 is the door check and must be hand-verified before Tasks 6–11.
- Open questions from the briefing are unchanged: merging `skillgrid.sqlite` into `aiskillgrid.sqlite` is an operator action, not scheduled here; the Kilo plugin directory is assumed `~/.config/kilo/plugin/` and verified in Task 9 step 2.

## Owed-Decision Gate

- ⚠ one-way: migration 049 (Task 2). Approval owed before first run against a real store; the briefing naming the columns was approved on 2026-10-02, the executor re-confirms at Task 2 step 3.
- Serial-development override: recorded in `adr.md`; this change executes while webui-rewrite sits at QA, by user decision.
- No other owed decisions.

## Plan Review

Reviewed against: requirement coverage, file/signature concreteness, one-way-door tagging, fail-open constraint, prompt-injection boundary, zone-guard commit discipline. Ready for `skillgrid:slicing`.
