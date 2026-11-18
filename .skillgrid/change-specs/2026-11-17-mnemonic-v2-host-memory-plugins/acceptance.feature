# Mnemonic-v2 5-Host Memory Plugins — acceptance scenarios (BDD)
# Companion to blueprint.md. CHECK/EXPECT are runnable gates; EVIDENCE lines
# are filled at execution/QA (the repo pattern). Scenario names are the
# blueprint tasks' SATISFIES targets.

Feature: 5-host mnemonic-v2 memory surface (one installer, one new verb)
  As an operator of pi, omp, opencode, kilo, and cursor
  I want the mnemonic-v2 memory surface (mem_* tools, prompt capture, session
  registration, compaction recovery) on every host
  So I can keep agent memory across sessions without a daemon.

  # ── save-prompt verb contract (threat: Mnemonic tool surface, delta 1) ──

  Scenario: verb-save-prompt-plain
    Given a project store resolvable from the current directory
    When I pipe plain text "first prompt" to `mnemonic memory save-prompt --id s1`
    Then exit code is 0 and stdout is empty
    And a user_prompts row exists for session s1 with content "first prompt"
    # CHECK: printf 'first prompt' | mnemonic memory save-prompt --id s1
    # EXPECT: exit 0, no stdout; SELECT count(*) FROM user_prompts = 1

  Scenario: verb-idempotent-refire
    Given a prompt row already stored for session s1 within the 10-minute window
    When I re-fire `memory save-prompt --id s1` with identical content
    Then no second user_prompts row is created
    And a re-fire with different content stores a second row
    # CHECK: fire twice with identical stdin; SELECT count(*) = 1
    # EXPECT: 1 row after re-fire, 2 rows after a distinct-content fire

  Scenario: verb-cursor-json-payload
    Given the cursor beforeSubmitPrompt envelope {"prompt":"hello cursor","conversation_id":"c1"}
    When I pipe it to `mnemonic memory save-prompt --id cursor`
    Then the "prompt" field is stored (not the envelope) and exit code is 0
    And a malformed envelope "{not json" is stored as raw text, not an error
    # CHECK: printf '{"prompt":"hello cursor"}' | mnemonic memory save-prompt --id cursor
    # EXPECT: row content = "hello cursor"

  Scenario: verb-stdin-cap
    Given a 300 KiB payload on stdin
    When I run `mnemonic memory save-prompt`
    Then the verb reads exactly 256 KiB (excess discarded) and exits 0
    # CHECK: unit: readStdinCapped over a 300 KiB reader
    # EXPECT: returned bytes == the first 256 KiB, len == 256<<10

  Scenario: verb-resolves-cwd-store-like-tools
    Given the project-resolution chain (explicit > env > git > basename)
    When I run the verb with the project env var set to a worktree project
    Then the prompt lands in that project's store and no other project's store is touched
    And the session-start probe and the verb resolve the same store for the same cwd
    # threat: Git repository selection (worktree/subdirectory must hit the same store)
    # CHECK: t.Setenv(<EnvVar>,"wt"); save-prompt; assert wt.sqlite has 1 row, p.sqlite has 0
    # EXPECT: per-project store files are distinct (dataDir/<project>.sqlite)

  Scenario: verb-auto-registers-missing-session
    Given no sessions row for id s2
    When I run `mnemonic memory save-prompt --id s2`
    Then exit code is 0 (fail-open: the store's auto-register creates the row)
    And the prompt row is stored under s2
    # CHECK: save-prompt with a fresh id; SELECT id, ownership FROM sessions WHERE id='s2'
    # EXPECT: exit 0; one session row (shared ownership), one prompt row

  # ── Live backstops (EVIDENCE at execution/QA) ───────────────────────────

  Scenario: live-omp-door-check
    Given omp with the extension manually installed and the mnemonic-memory MCP entry
    When a live omp session runs N prompts and one task subagent
    Then exactly N prompt rows exist for the conversation's session
    And the sessions table holds exactly 1 row for the conversation
    And the subagent adds 0 session rows and 0 prompt rows
    And `mem_save` then `mem_search` round-trips against the session cwd's store
    # EVIDENCE: (filled at execution — Task 2 door check)

  Scenario: live-opencode-kilo-session
    Given opencode and kilo with the plugin installed
    When a live session saves memories and calls `mem_save` with a foreign or absent session_id
    Then the row is stored under the adapter's session (the tool.execute.before rewrite fired)
    And N prompts produce exactly N prompt rows and 1 session row
    # EVIDENCE: (filled at QA — Task 11)

  Scenario: live-compaction-notice
    Given a pi/omp or opencode/kilo session with stored memory context
    When the session is compacted
    Then the agent's context receives the FIRST ACTION REQUIRED memory re-injection notice
    # EVIDENCE: (filled at QA — Task 11)

  Scenario: live-cursor-session
    Given cursor with the installed hooks + rule
    When a live session starts and submits prompts
    Then sessionStart created exactly one `cursor` session row
    And each beforeSubmitPrompt created a prompt row
    And the installed rule contains `session_id: "cursor"` and zero occurrences of `web_cache_`, `fact_`, or `serve`
    # EVIDENCE: (filled at QA — Task 11)

  # ── Fail-open and probe ─────────────────────────────────────────────────

  Scenario: fail-open-drop-counter
    Given the store busy or the verb failing or timing out
    When the extension fires `save-prompt`
    Then no error surfaces to the host session
    And the per-project drop counter increments
    And `skillgrid status --family memory` reports "N captures dropped (store busy)"
    # CHECK: unit over the capture module (injected failing spawn)
    # EXPECT: host callback gets no error; counter file N+1; status text matches

  Scenario: probe-missing-binary-repair
    Given the mnemonic binary unresolvable at session start (the PATH case)
    When the pi/omp extension runs its session-start probe
    Then a warning naming the cause and the exact repair command is injected into context
    And the per-project scratch record is written and the session continues
    And `skillgrid status --family memory` surfaces the record
    # CHECK: unit over the shell module (injected spawn → ENOENT)
    # EXPECT: notice contains cause + repair command; scratch file written; no throw

  # ── Installer (threat: manifest tracked-set semantics, D-2) ─────────────

  Scenario: installer-roundtrip-fixture
    Given each of the 5 fixture HOMEs (omp, pi, opencode, kilo, cursor) with foreign MCP entries
    When I run `skillgrid install --family memory all`
    Then each HOME gains exactly the tracked file set + entries, every command an absolute binary path
    And the kilo plugin is a byte-identical copy of the opencode-kilo source
    And a re-run is byte-identical (idempotent no-op)
    And foreign MCP entries and files are byte-untouched
    And `skillgrid uninstall --family memory all` returns each HOME byte-identical to pre-install
    # CHECK: fixture-HOME test over install/uninstall (per-host round-trip)
    # EXPECT: tracked set exact; kilo diff == empty; foreign bytes unchanged; HOME diff == empty

  Scenario: engram-coexistence-warning
    Given a kilo fixture HOME with an engram file present
    When I run `skillgrid install --family memory kilo`
    Then the coexistence warning is printed
    And `skillgrid status --family memory` lists the engram file
    # CHECK: install kilo fixture with engram file; capture stdout + status
    # EXPECT: warning text present; status lists the file

  # ── Cutover (one-way D-1) ───────────────────────────────────────────────

  Scenario: cutover-rewrites-stale-entries
    Given fixture HOMEs carrying the three stale v1 shapes (omp string-command, opencode/kilo JSONC arrays, cursor `mnemonic` key)
    When I run `skillgrid install --family memory all`
    Then each stale entry is rewritten to [<abs-bin>,"memory","mcp","--profile","agent"] with the host dialect preserved (comments, sibling entries, enabled/type fields)
    And the post-cutover installed set contains no functional reference to `:7438`, `skillgrid mcp`, `web_cache_`, or `fact_`
    # CHECK: cutover fixture tests (one per dialect)
    # EXPECT: entry rewritten exactly once; dialect bytes preserved; grep clean

  Scenario: skillmd-v2-grep-gate
    Given the cutover commit
    When I grep `.agents/skills/` for `:7438`, `skillgrid mcp`, `web_cache_`, `fact_*`, `code_index`
    Then no live reference remains (change-archive and ADR references excluded)
    # threat: shared-convention drift (SKILL.md rewritten in the same commit)
    # CHECK: grep -rnE ':7438|skillgrid mcp|web_cache_|fact_|code_index' .agents/skills/
    # EXPECT: 0 hits (excluding the change-specs/decisions allowlist)

  # ── Protocol single source (threat: tool surface deltas 2/3) ────────────

  Scenario: protocol-single-source
    Given the agent-profile MCP instructions rendered in Go
    When I run the rune guard test
    Then the rendered instructions are at most 2048 runes
    And the cursor rule's body section is diff-equal to the Go source
    And a smoke session shows the protocol body exactly once
    # CHECK: go test ./internal/transport/mcp/ -run 'TestInstructionsRuneGuard|TestCursorRuleBodyDiff'
    # EXPECT: both pass; EVIDENCE: (body-once filled at QA)

  # ── v1 data import ──────────────────────────────────────────────────────

  Scenario: v1-import-script
    Given a fixture v1 store with observations and sessions
    When I run `scripts/mnemonic-v1-import.sh --from <v1-store> --to <v2-store>`
    Then `mnemonic memory search` finds the v1 content in the v2 store
    And a re-run adds 0 rows (content-keyed idempotency)
    And `--dry-run` prints the would-import counts and writes nothing
    # CHECK: run script on fixtures; mnemonic memory search <v1 term>; re-run; --dry-run
    # EXPECT: hits on v1 content; row count unchanged on re-run; dry-run byte-identical stores
