Feature: mem fs browses the code index (OpenViking-style)

  Background:
    Given a project store with an existing code index
    And two indexed files: "src/auth/login.go" (1 symbol "Handler") and "src/util.go" (0 symbols)

  Scenario: code-index-listing (ls on a directory lists files/subdirs)
    When I run `mem fs ls <project-root>`
    Then the listing contains the "src" directory
    And no memory observations are returned

  Scenario: code-index-listing-symbols (ls on a file lists its symbols)
    When I run `mem fs ls src/auth/login.go`
    Then the listing contains exactly one entry with kind "symbol"
    And the entry name is "Handler" with signature "func Handler() error" and line range 1-10

  Scenario: code-index-tree (tree renders the repo tree with symbol counts)
    When I run `mem fs tree <project-root>`
    Then the tree shows "src" and "login.go" annotated "1 symbols"
    And the tree shows "util.go" annotated "0 symbols"

  Scenario: code-index-find (find globs paths and symbol names)
    When I run `mem fs find *.go`
    Then two file entries are returned
    When I run `mem fs find Handler`
    Then one symbol entry named "Handler" is returned
    When I run `mem fs find *.go src/auth/`
    Then only the "login.go" file is returned (scope narrows the search)

  Scenario: code-index-cat (cat returns node source)
    When I run `mem fs cat src/auth/login.go::Handler`
    Then the output contains "func Handler() error"
    When I run `mem fs cat src/auth/login.go`
    Then the output contains the full file text
    When I run `mem fs cat src/auth/login.go::Nope`
    Then the command exits non-zero with a clear error

  Scenario: empty-store-no-index (unindexed store notes, does not error)
    Given a project store with no code index (zero files)
    When I run `mem fs ls <project-root>`
    Then the command prints a "no code index" note and exits 0
    When I run `mem fs tree <project-root>`
    Then the output contains "no code index"

  Scenario: memory-unchanged (mem search / MCP still work)
    When I save a memory observation and run `mem search`
    Then the observation is still found (mem fs re-point did not touch memory)
