Feature: skill-pi foundation
  An opinionated, non-fork distribution of Pi that runs with the skillgrid
  standards, skills, memory (mnemonic MCP), code index, events, squad,
  compaction, a permission extension, ticketing, a context extension, a
  custom TUI, and the /memory-sync command — all from skill-pi's own config
  paths, never touching a user's plain pi install.

  Background:
    Given a skillgrid repo at the working directory
    And the skill-pi module is installed at the pinned pi version
    And the mnemonic MCP server is reachable at the loopback URL

  # --- Task 1: module skeleton + dependency ADR (R12) ---

  Scenario: happy path module-skeleton
    Given the skill-pi module exists at the repo root
    When I read skill-pi/package.json
    Then the piConfig block sets name "skill-pi" and configDir ".skill-pi"
    And the bin block maps skill-pi to ./bin/skill-pi.js
    And the @earendil-works/pi-coding-agent dependency is an exact version with no caret or tilde
    And pi-version.txt content equals the package.json dependency string
    And README.md names a minimum pi version for piConfig.configDir
    And ADR 0042 is indexed in ASSUMPTIONS.md LOCKED section

  # --- Task 2: launcher CLI + cwd shim (R1, R2, R3, R14, R15, R18, R20, R22) ---

  Scenario: happy path launcher
    Given a fresh repo with no .skill-pi directory
    When I run skill-pi in the repo
    Then PI_CODING_AGENT_DIR is set to ~/.skill-pi in the spawned process
    And ~/.pi/ is byte-identical before and after the run
    And the shim at ~/.skill-pi/cwd/<project>/ is materialized from scratch
    And the shim package.json carries piConfig.configDir ".skill-pi"
    And pi reads its global config from ~/.skill-pi/
    And pi reads its project config through the shim to <project>/.skill-pi/

  Scenario: happy path cwd-shim
    Given a repo with an existing .skill-pi directory
    When I add a new file to the real project root
    And I run skill-pi again
    Then the new file is visible through the shim without a manual sync
    And <project>/.pi/ is byte-identical before and after the run

  Scenario: edge spawn-respawn-loop
    Given the spawned pi process exits with code 75
    When I observe the launcher
    Then the launcher re-spawns pi
    And the shim is re-materialized before the re-spawn
    And a second exit with code 0 does not re-spawn
    And eleven consecutive exits with code 75 stop the loop

  Scenario: happy path owned-subcommands
    When I run skill-pi version
    Then the pinned pi version is printed without entering the TUI
    When I run skill-pi --help
    Then pi's help is shown, not the launcher's
    When I run skill-pi with a pi flag like --session
    Then the flag is forwarded to pi and honored

  Scenario: happy path where-json
    When I run skill-pi --where
    Then it exits with code 0
    And it emits a parseable JSON object with keys skillPiRoot, piCli, agentDir, stateDir, shimDir, projectRoot, configDir, executable
    And configDir equals <project-root>/.skill-pi
    And agentDir equals ~/.skill-pi

  Scenario: happy path trust-first-run
    Given a fresh repo with no trust entry in ~/.skill-pi/trust.json
    When I run skill-pi for the first time
    Then the trust prompt is shown
    When I accept the trust prompt
    Then ~/.skill-pi/trust.json contains a key equal to the shim's realpath
    When I run skill-pi again
    Then no trust prompt is shown

  Scenario: edge quiet-startup
    When I run skill-pi
    Then PI_SKIP_VERSION_CHECK is set to 1 in the spawn env
    And --no-skills --no-prompt-templates --no-themes --extension <skill-pi-root> are in the argv
    And pi does not print its version banner
    And pi does not auto-discover ~/.pi/agent/skills/

  # --- Task 3: global config + system prompt (R16, R17) ---

  Scenario: happy path global-config-no-url
    When I grep skill-pi/settings.json and skill-pi/mcp.json for a provider base URL
    Then no remote https URL with a /v1 or /api path is found
    And setting SKILLGRID_MNEMONIC_HTTP_URL changes the effective base URL the extensions resolve

  Scenario: happy path system-delta-only
    When I grep skill-pi/system/SYSTEM.md for the gentle-ai persona or engram protocol blocks
    Then the grep count is zero
    And the zone discipline section is present
    And the commit contract section is present
    And a pointer to AGENTS.md for the persona and engram blocks is present

  # --- Task 4: five opencode-behavior extension ports (R5) ---

  Scenario: happy path extension-ports-register
    Given a pi session run under skill-pi
    When I inspect the tool registry
    Then mem_save, mem_search, mem_get_observation are registered
    And code_search, code_read, code_status, code_index are registered
    And event capture tools are registered
    And the compaction summary tool is registered
    When I call mem_save from the pi session
    Then the observation lands in the mnemonic store

  # --- Task 5: four new extensions — permission, tickets, context, mcp-bridge (R6, R7, R8, R19) ---

  Scenario: happy path permission-gate
    Given defaultMode is "ask" and no matching rule exists
    When I call the edit tool from a pi session
    Then the approval prompt is shown with Allow once, Allow always, Allow for session, Reject, Reject with feedback
    Given a deny rule on the bash pattern "rm -rf *"
    When I call the bash tool with "rm -rf /tmp/test"
    Then the call is blocked with Permission denied and no prompt
    Given the command "git status | grep error"
    When the permission extension parses it with tree-sitter
    Then both "git status" and "grep error" match their respective rules
    When I choose Allow always on the prompt
    Then the rule is persisted to the local tier permission.settings.local.json

  Scenario: happy path ticket-tool
    When I call the ticket tool with action ready
    Then ready-for-agent Backlog.md tasks are returned
    When I call the ticket tool with action create and a title
    Then a .backlog/tasks/task-NNN.md file is created
    And no .tickets/ directory is created
    When the ticket list exceeds 40000 characters
    Then the output is truncated to 40000 characters
    And the ready results are line-clamped to 20 lines by default

  Scenario: happy path context-command
    Given a pi session run under skill-pi
    When I run /skillgrid context usage
    Then a usage map is rendered with a Mnemonic category
    When I run /skillgrid context injections
    Then the hidden injection parts are shown
    When the compaction threshold is crossed
    Then a hint is emitted or /compact is triggered

  Scenario: happy path mcp-bridge-registry
    Given a pi session run under skill-pi
    When I inspect the tool registry
    Then mcp__ entries are present for the mnemonic, backlog, fluxcd, codescene, and trivy servers
    And the permission extension's mcp__* glob matches them

  # --- Task 6: TUI via @earendil-works/pi-tui (R9) ---

  Scenario: happy path tui-breakpoint
    Given a terminal at 140 columns or wider
    When the TUI renders
    Then the full layout is shown with header, transcript, and right rail
    Given a terminal at fewer than 140 columns
    When the TUI renders
    Then the compact layout is shown with bottom bar and agents widget

  Scenario: edge tui-gauge-tone
    Given the context gauge is at 79 percent
    When the TUI renders
    Then the gauge tone is normal
    Given the context gauge is at 80 percent
    When the TUI renders
    Then the gauge tone escalates to warning
    Given the context gauge is at 95 percent
    When the TUI renders
    Then the gauge tone escalates to critical

  # --- Task 7: /memory-sync command + skills sync (R10, R11) ---

  Scenario: happy path memory-sync-jsonl
    Given a mnemonic store with existing observations
    When I run /memory-sync
    Then <project>/.skill-pi/memory/facts.jsonl exists and is valid JSONL
    When I run /memory-sync a second time
    Then no rows are duplicated in facts.jsonl
    And the SQLite DB file is gitignored

  Scenario: happy path skills-sync
    Given ~/.agents/skills/ contains a representative skill with a SKILL.md
    When the installer sync step runs
    Then the skill is present in the pi skills location
    And a pi session lists it in its available skills
    And ~/.agents/skills/ remains the single source of truth with no second catalog in the repo

  # --- Task 8: installer extension + --where + trust persistence (R13, R15, R20) ---

  Scenario: happy path installer-npm-ci
    When I run skillgrid-cli install in a repo
    Then ~/.skill-pi/node_modules/@earendil-works/pi-coding-agent exists at the pinned version
    And skill-pi is on PATH
    And <project-root>/.skill-pi/ is materialized as a real directory

  # --- Task 9: chronos injection + version-pair + supply-chain (R12, R21) ---

  Scenario: happy path chronos-injection
    Given a pi session run under skill-pi
    When I inspect the session context
    Then the injected date/time is present
    And the value matches the system clock

  Scenario: happy path version-pair-equality
    When I read skill-pi/pi-version.txt and skill-pi/package.json
    Then the @earendil-works/pi-coding-agent dependency string equals the pi-version.txt content
    And the version is exact with no caret, tilde, or range

  # --- Task 10: foundation e2e tracer (R1–R22 proof) ---

  Scenario: happy path foundation-e2e
    Given a fresh skillgrid repo
    When I run skill-pi in the repo
    Then ~/.pi/ is byte-identical before and after
    And <project>/.pi/ is byte-identical before and after
    And <project>/.skill-pi/ exists as a real gitignored directory
    And a pi session exposes the memory, code index, event, squad, and compaction tools
    And the permission gate is active with defaultMode ask
    And the ticket tool is registered
    And /skillgrid context usage renders the usage map
    And the TUI renders in the alt-screen terminal
    And /memory-sync writes valid JSONL
    And the mcp-bridge registers mcp__ entries
    And the chronos date/time is injected
    And the launcher re-spawns on exit code 75
    And skill-pi --where emits the 8-key JSON
    And skill-pi version prints the pinned version without entering the TUI

Feature: skill-pi swarm slice
  The main pi lead runs a goal-loop swarm over the existing mnemonic team
  plane, spawning Pi SDK subagent sessions as worker/reviewer, enforcing the
  two-review done gate, with opt-in per-task worktree isolation and a goals
  container so the dashboard shows goal progress.

  Background:
    Given the skill-pi foundation is installed and proven
    And the mnemonic team plane is running at the loopback URL

  # --- Task 11: goals table + tasks.goal_id migration (ADR-0039) ---

  Scenario: happy path goal-container-migration
    Given an empty mnemonic team plane database
    When the migration runs
    Then a goals table exists with id, title, created_at columns
    And the tasks table has a goal_id foreign key column
    And re-running the migration changes nothing

  # --- Task 12: SpawnTask + team_spawn_task accept goal_id (ADR-0039, defect A) ---

  Scenario: happy path spawn-task-by-goal
    Given a goal with id "goal-1" exists
    When I call team_spawn_task with goal_id "goal-1" and a title
    Then the created task has goal_id "goal-1" set
    And the task is retrievable by goal id
    When I call GET /teams/tasks/<task-id>/reviews
    Then the reviews endpoint returns the task's review list

  # --- Task 13: ListTasksByGoal with per-status counts (ADR-0039) ---

  Scenario: happy path goal-progress-query
    Given a goal with 3 tasks: 1 pending, 1 in-progress, 1 done
    When I call ListTasksByGoal for that goal
    Then the result includes all 3 tasks
    And the per-status counts show pending 1, in-progress 1, done 1

  # --- Task 14: pi extension scaffold — 6 team-plane HTTP tools (swarm R1) ---

  Scenario: happy path squad-http-adapter
    Given a pi session run under skill-pi
    When I inspect the tool registry
    Then team_plane tools are registered for the 6 team-plane HTTP operations
    And a squad_reviews read tool is registered for the two-review gate
    And the tools communicate with the team plane over HTTP at the loopback URL

  # --- Task 15: role discovery — roles.ts (swarm R3, ADR-0041) ---

  Scenario: happy path role-discovery
    Given agents/pi/roles/worker.md and agents/pi/roles/reviewer.md exist
    When the role discovery runs
    Then worker and reviewer are discovered as roles
    And dropping a new agents/pi/roles/foo.md makes foo a role with no code change

  # --- Task 16: worktree isolation — worktree.ts + branch naming (swarm R5, ADR-0040) ---

  Scenario: happy path worktree-isolation
    Given a task with worktree true in its brief
    When the worktree setup runs for that task
    Then a git worktree is created under .worktrees/swarm-<task-id>
    And the branch is named worktree-agent-swarm-<task-id>
    And the branch name is recorded in the task's worktree_branch column
    When the task completes
    Then the work is committed to the worktree branch

  # --- Task 17: subagent spawner — spawn.ts (swarm R2, R8) + role prompts ---

  Scenario: happy path swarm-spawn-subagent
    Given a task in pending status
    When I spawn a worker subagent for that task
    Then the subagent's memory owner is set to the task id
    And the subagent's cwd is the repo root
    And the task status transitions to in-progress
    Given a task with worktree true
    When I spawn a worker subagent for that task
    Then the subagent's cwd is the worktree path

  Scenario: edge serial-by-default
    Given one worker subagent is active for a task
    When I attempt to spawn a second worker subagent
    Then the second spawn waits until the first task is done

  # --- Task 18: checkpoint — goal.ts + two-review gate (swarm R7, R4) ---

  Scenario: happy path checkpoint-roundtrip
    Given a lead session is running
    When the lead writes a checkpoint with current task and status
    Then .skill-pi/goal/progress.json contains the checkpoint
    When the lead reads the checkpoint back
    Then the current task and status are returned

  Scenario: happy path two-review-gate
    Given a task in review_spec status
    When a spec_compliance review is submitted with passed true
    And a code_quality review is submitted with passed true
    Then the task transitions to done
    Given a task in review_spec status
    When a spec_compliance review is submitted with passed false
    Then the task is routed back to the worker for re-review
    And the task does not transition to done

  # --- Task 19: dashboard goal-progress view (swarm R6) ---

  Scenario: happy path dashboard-goal-progress
    Given a goal with tasks in various statuses
    When the dashboard renders the goal-progress view
    Then the goal's tasks are listed with their statuses
    And the per-status counts are shown

  # --- Task 20: end-to-end tracer (swarm build-shape proof) ---

  Scenario: happy path e2e-goal-loop
    Given a skillgrid repo with the skill-pi foundation installed
    And a goal with 2 tasks is created on the team plane
    When the pi lead runs the goal loop
    Then both tasks are spawned as worker subagents serially
    And each task goes through spec_compliance and code_quality review
    And both tasks reach done status
    And the dashboard shows the goal at 100 percent progress
    And the lead's checkpoint in progress.json reflects the completed goal
