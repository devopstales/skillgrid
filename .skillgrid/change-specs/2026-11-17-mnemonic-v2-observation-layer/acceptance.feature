# Mnemonic-v2 Observation Layer — acceptance scenarios (BDD)
# Companion to blueprint.md. CHECK/EXPECT are runnable gates; EVIDENCE lines
# are filled at execution/QA (the repo pattern).

Feature: Observation layer (gryph-style agent event capture)
  As an operator of mnemonic-v2
  I want agent tool activity captured into the per-project store without a daemon
  So I can answer "what did the agent read, write, and run" from `mnemonic obs`.

  # ── Ingest contract ───────────────────────────────────────────────────────

  Scenario: obs-ingest-normalizes
    Given a project store with migration 300 applied
    When I pipe a tool_call envelope with input command "git status" to `mnemonic obs _ingest omp tool_call`
    Then exit code is 0
    And an obs_events row exists with tool_type "shell" and command "git status"
    # CHECK: printf '%s' '{"v":1,"session_id":"s1","cwd":"<repo>","agent":"omp","event":"tool_call","tool_name":"bash","input":{"command":"git status"}}' | mnemonic obs _ingest omp tool_call
    # EXPECT: exit 0; SELECT tool_type FROM obs_events = 'shell'

  Scenario: obs-privacy-contract
    Given an envelope whose output is "SECRET=1" with input file_path "/repo/.env"
    When I ingest it as a tool_result
    Then the row has path "/repo/.env"
    And input_preview and output_preview are empty (sensitive path)
    And content_hash is the sha256 of the full pre-truncation value
    And a prompt containing <private>x</private> stores no "x" in the prompt column
    # CHECK: ingest + SELECT path, input_preview, output_preview, content_hash, prompt
    # EXPECT: path set, previews NULL, hash = sha256("SECRET=1"), prompt lacks "x"

  Scenario: obs-ingest-atomically-registers-session
    Given an envelope with a never-seen session_id
    When I ingest it
    Then a sessions row for that id exists in the same store
    And a second ingest with the same session_id creates no duplicate session row

  Scenario: obs-ingest-exit-codes-fail-open
    Given a malformed JSON envelope on stdin
    When I run `mnemonic obs _ingest omp tool_call`
    Then exit code is 1 and a note is on stderr
    And no obs_events row was created
    And a valid envelope on a postgres-dialect store exits 1 ("sqlite dialect only")
    And an envelope with an unknown agent exits 1

  # ── CLI read surface ──────────────────────────────────────────────────────

  Scenario: obs-logs-and-query-read-stored-events
    Given a store with shell, file_read, and file_write events across two sessions
    When I run `mnemonic obs logs`
    Then the last-24h events are printed as a human table
    And `obs logs --format json` is a parseable JSON array
    And `obs query --action tool_call --command "git *"` returns exactly the git shell events
    And `obs query --file "src/**" --action file_write` glob-matches path

  Scenario: obs-sessions-session-stats-export
    Given a store with multiple sessions and events
    When I run `mnemonic obs sessions`
    Then each session lists with agent, title, event count, last-active
    And `obs session <id-prefix>` prints time-ordered detail
    And `obs stats --since 7d` counters equal a direct SQL GROUP BY for the same window
    And `obs export --since 1d` emits one parseable JSON object per line

  Scenario: obs-prune-scoped-to-obs-events
    Given obs_events rows 30 and 100 days old plus a fresh row
    When I run `mnemonic obs prune --retention 90d --dry-run`
    Then it reports the deletable count and deletes nothing
    And `mnemonic obs prune --retention 90d` deletes only the 100-day row
    And sessions rows and every other table are untouched

  # ── Install lifecycle ─────────────────────────────────────────────────────

  Scenario: obs-install-omp-and-pi
    Given a fixture HOME without an omp extension
    When I run `mnemonic obs install omp`
    Then ~/.omp/agent/extensions/mnemonic-obs.ts exists and contains `const AGENT = "omp"` (never "__AGENT__")
    And the manifest under the data dir records the file hash
    And re-running install changes nothing (idempotent no-op)
    And with SKILL_PI_AGENT_DIR set, `obs install pi` creates <dir>/extensions/mnemonic-obs/{mnemonic-obs.ts,package.json}
    And `obs uninstall omp` removes exactly the installed file (a foreign extension survives)

  Scenario: obs-install-opencode-kilo
    Given fixture opencode and kilo config trees
    When I run `mnemonic obs install opencode kilo`
    Then ~/.config/opencode/plugins/mnemonic-obs.ts contains `AGENT = "opencode"`
    And ~/.config/kilo/plugin/mnemonic-obs.ts contains `AGENT = "kilo"` plus the config registration entry (foreign config entries preserved)

  Scenario: obs-install-cursor-preserves-foreign-entries
    Given a ~/.cursor/hooks.json containing gryph, orca, and herdr entries
    When I run `mnemonic obs install cursor`
    Then all foreign entries are byte-identical
    And exactly the 20 obs entries exist, each once
    And a backup of the pre-image exists under the data dir
    And `obs uninstall cursor` removes exactly the 20 obs entries
    And a re-install adds nothing (idempotent)
    And a malformed hooks.json fails install with the file untouched

  # ── Live backstops (EVIDENCE at QA) ───────────────────────────────────────

  Scenario: obs-live-omp-session-produces-events
    Given obs installed for omp on a real host
    When an omp session runs at least 3 tool calls and quits
    Then `mnemonic obs logs --agent omp` shows session_start, tool_call+tool_result pairs, session_end
    # EVIDENCE: (filled at execution)

  Scenario: obs-fail-open-missing-binary
    Given the mnemonic binary renamed out of PATH
    When the agent runs a tool call
    Then the tool call proceeds and the hook exits 0 with a stderr note
    # EVIDENCE: (filled at execution)

  Scenario: obs-live-hosts
    Given obs installed for all detected hosts
    When one short real session runs per host
    Then `mnemonic obs logs --agent <h>` shows that host's events with the correct agent label
    # EVIDENCE: (filled at QA)
