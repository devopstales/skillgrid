Feature: Replace opencode + Kilo shell hooks with native TS plugins
  The shell-era hook system for opencode and Kilo is retired and replaced by
  5 native TypeScript plugins: skillgrid-compaction, skillgrid-events,
  skillgrid-squad, mnemonic-memory, and mnemonic-codeindex. The Go installer
  wires all 5 for both harnesses, deletes the retired shell scripts and
  checkpoint plugin copies, and preserves the shared workers still used by
  Cursor and the git-hook Stop gates. Facts HTTP routes expose the existing
  facts.Store over the mnemonic HTTP server.

  Background:
    Given a clean temp home directory
    And a repo root containing the 5 plugin source files under plugins/opencode/
    And the shared worker scripts tool-call-capture.js, stop-tests.js, and gate-stop.js under hooks/

  # --- Req 1: opencode installer wires 5 plugins ---

  Scenario: happy path SetupOpenCode installs all 5 plugins
    Given the repo root contains plugins/opencode/skillgrid-compaction.ts
    And the repo root contains plugins/opencode/skillgrid-events.ts
    And the repo root contains plugins/opencode/skillgrid-squad.ts
    And the repo root contains plugins/opencode/mnemonic-memory.ts
    And the repo root contains plugins/opencode/mnemonic-codeindex.ts
    When I run SetupOpenCode
    Then ~/.config/opencode/plugin/skillgrid-compaction.ts exists
    And ~/.config/opencode/plugin/skillgrid-events.ts exists
    And ~/.config/opencode/plugin/skillgrid-squad.ts exists
    And ~/.config/opencode/plugin/mnemonic-memory.ts exists
    And ~/.config/opencode/plugin/mnemonic-codeindex.ts exists
    And the opencode config lists ./plugin/skillgrid-compaction.ts
    And the opencode config lists ./plugin/skillgrid-events.ts
    And the opencode config lists ./plugin/skillgrid-squad.ts
    And the opencode config lists ./plugin/mnemonic-memory.ts
    And the opencode config lists ./plugin/mnemonic-codeindex.ts

  Scenario: happy path SetupOpenCode no longer installs hooks.yaml
    When I run SetupOpenCode
    Then ~/.config/opencode/plugin/hooks.yaml does not exist
    And ~/.config/opencode/hook/hooks.yaml does not exist
    And the opencode config does not list opencode-yaml-hooks

  Scenario: happy path SetupOpenCode no longer installs skillgrid-checkpoint
    When I run SetupOpenCode
    Then ~/.config/opencode/plugin/skillgrid-checkpoint.ts does not exist
    And the opencode config does not list skillgrid-checkpoint.ts

  # --- Req 2: kilocode installer wires 5 plugins ---

  Scenario: happy path SetupKiloCode installs all 5 plugins from opencode source
    Given the repo root contains plugins/opencode/skillgrid-compaction.ts
    And the repo root contains plugins/opencode/skillgrid-events.ts
    And the repo root contains plugins/opencode/skillgrid-squad.ts
    And the repo root contains plugins/opencode/mnemonic-memory.ts
    And the repo root contains plugins/opencode/mnemonic-codeindex.ts
    When I run SetupKiloCode
    Then ~/.config/kilo/plugin/skillgrid-compaction.ts exists
    And ~/.config/kilo/plugin/skillgrid-events.ts exists
    And ~/.config/kilo/plugin/skillgrid-squad.ts exists
    And ~/.config/kilo/plugin/mnemonic-memory.ts exists
    And ~/.config/kilo/plugin/mnemonic-codeindex.ts exists
    And the kilo config lists ./plugin/skillgrid-compaction.ts
    And the kilo config lists ./plugin/skillgrid-events.ts
    And the kilo config lists ./plugin/skillgrid-squad.ts
    And the kilo config lists ./plugin/mnemonic-memory.ts
    And the kilo config lists ./plugin/mnemonic-codeindex.ts

  Scenario: happy path SetupKiloCode no longer installs hooks.yaml
    When I run SetupKiloCode
    Then ~/.config/kilo/plugin/hooks.yaml does not exist
    And ~/.config/kilo/hook/hooks.yaml does not exist
    And the kilo config does not list opencode-yaml-hooks

  Scenario: happy path SetupKiloCode no longer installs skillgrid-checkpoint
    When I run SetupKiloCode
    Then ~/.config/kilo/plugin/skillgrid-checkpoint.ts does not exist
    And the kilo config does not list skillgrid-checkpoint.ts

  # --- Req 3: retired shell hooks deleted from repo ---

  Scenario: happy path retired opencode shell scripts are removed
    When the change is applied to the repo
    Then hooks/opencode-session-start.sh does not exist
    And hooks/opencode-session-end.sh does not exist
    And hooks/opencode-policy.sh does not exist
    And hooks/opencode-tool-capture.sh does not exist

  Scenario: happy path retired hooks.yaml files are removed
    When the change is applied to the repo
    Then plugins/opencode/hooks.yaml does not exist
    And plugins/kilo/hooks.yaml does not exist

  Scenario: happy path retired checkpoint plugins are removed
    When the change is applied to the repo
    Then plugins/opencode/skillgrid-checkpoint.ts does not exist
    And plugins/kilo/skillgrid-checkpoint.ts does not exist

  # --- Req 4: shared workers preserved ---

  Scenario: happy path shared worker tool-call-capture.js is kept
    When the change is applied to the repo
    Then hooks/tool-call-capture.js exists
    And the SetupOpenCode run installs hooks/tool-call-capture.js to ~/.skillgrid/hooks/
    And the SetupKiloCode run installs hooks/tool-call-capture.js to ~/.skillgrid/hooks/

  Scenario: happy path git-hook Stop gates are kept
    When the change is applied to the repo
    Then hooks/stop-tests.js exists
    And hooks/gate-stop.js exists
    And the SetupOpenCode run installs stop-tests.js to ~/.skillgrid/hooks/
    And the SetupKiloCode run installs gate-stop.js to ~/.skillgrid/hooks/

  # --- Req 5: facts HTTP routes ---

  Scenario: happy path POST /facts adds a fact
    Given a facts store with no facts
    When I POST /facts with body {"content": "use bcrypt not md5"}
    Then the response is 200
    And the response body has field id that is a positive integer

  Scenario: happy path POST /facts/search finds a fact
    Given a facts store with one fact "use bcrypt not md5"
    When I POST /facts/search with body {"query": "bcrypt"}
    Then the response is 200
    And the response body has field facts that is an array with at least 1 item
    And the first fact content contains "use bcrypt not md5"

  Scenario: happy path POST /facts/{id}/forget soft-deletes a fact
    Given a facts store with one fact and id 1
    When I POST /facts/1/forget
    Then the response is 204
    When I POST /facts/search with body {"query": "bcrypt"}
    Then the response body facts array is empty

  Scenario: happy path POST /facts/{id}/decay applies decay
    Given a facts store with one fact and id 1 and importance_score 1.0
    When I POST /facts/1/decay
    Then the response is 200
    And the response body has field score that is less than 1.0

  Scenario: happy path POST /facts/decay-all decays and purges all facts
    Given a facts store with 3 facts, one below the default threshold 0.5
    When I POST /facts/decay-all
    Then the response is 200
    And the response body has field decayed equal to 3
    And the response body has field purged equal to 1

  Scenario: error path POST /facts with empty content returns 400
    Given a facts store
    When I POST /facts with body {"content": ""}
    Then the response is 400

  Scenario: error path POST /facts/999/forget with unknown id returns 404
    Given a facts store with no facts
    When I POST /facts/999/forget
    Then the response is 404

  # --- Req 6: FindRepoRoot uses new plugin rels ---

  Scenario: happy path FindRepoRoot finds repo root via new plugin files
    Given a repo root containing plugins/opencode/skillgrid-compaction.ts
    When I call FindRepoRoot from a subdirectory
    Then it returns the repo root path

  Scenario: error path FindRepoRoot fails when no new plugin files exist
    Given a repo root containing only retired hooks.yaml
    When I call FindRepoRoot from a subdirectory
    Then it returns an error

  # --- Req 7: dropRetiredPlugins handles new retired keys ---

  Scenario: happy path dropRetiredPlugins removes opencode-yaml-hooks and checkpoint
    Given an opencode config that lists opencode-yaml-hooks and skillgrid-checkpoint.ts
    When I run SetupOpenCode
    Then the opencode config does not list opencode-yaml-hooks
    And the opencode config does not list skillgrid-checkpoint.ts
    And the opencode config does list ./plugin/skillgrid-compaction.ts
    And the opencode config does list ./plugin/skillgrid-events.ts
    And the opencode config does list ./plugin/skillgrid-squad.ts
    And the opencode config does list ./plugin/mnemonic-memory.ts
    And the opencode config does list ./plugin/mnemonic-codeindex.ts

  # --- Req 8: test suite passes ---

  Scenario: happy path Go tests for setup and http packages pass
    Given the code changes are applied
    When I run go test ./mnemonic/internal/setup/... ./mnemonic/internal/http/...
    Then all tests pass

  Scenario: happy path plugin tests pass
    Given the code changes are applied
    When I run pnpm test
    Then all tests pass

  Scenario: happy path lint and typecheck pass
    Given the code changes are applied
    When I run pnpm lint
    Then the command exits 0
    When I run pnpm typecheck
    Then the command exits 0
