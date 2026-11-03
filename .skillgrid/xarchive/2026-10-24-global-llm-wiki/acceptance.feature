# Global Operational Memory + Skill Improvement Loop

<!--
Source: .skillgrid/specs/2026-10-24-global-llm-wiki/
Trace: change briefing ## Definition of Done; blueprint tasks SATISFIES lines.
Format rules:
- Steps in the ```gherkin fence are indented 6 spaces (verbatim — the extractor does NOT re-indent)
- No Gherkin tags in the spec — selection/trace is via the SATISFIES field in blueprint/tasks
- One blank line between the heading and the fence is fine
- Gates: each requirement carries a `#### Gates` block — the runnable shadow of
  its happy-path and failure scenarios. Author it BEFORE implementing.
-->

## Requirements

### Requirement: Global store bootstrap

The system SHALL create a cross-project fact store at `~/.skillgrid/mnemonic/__global__.sqlite` that is invisible to project listing and TTL expiry, and that supports the same facts schema as per-project stores.

#### Scenario: happy path — global store is created and opened

```gherkin
      Given a fresh data directory with no existing stores
      When the global store is opened
      Then a file named `__global__.sqlite` exists in the data directory
      And the facts table in the global store has a `category` column
      And the category column defaults to `memory` for existing rows
```

#### Scenario: edge — global store is excluded from project listing

```gherkin
      Given a project store for `proj-a` exists
      And the global store exists
      When the project list is requested
      Then `proj-a` appears in the list
      And `__global__` does not appear in the list
```

#### Scenario: edge — global store is excluded from TTL expiry

```gherkin
      Given the global store contains facts with expired TTL
      When the TTL expiry sweep runs
      Then the global store's facts are not purged
      And the per-project stores' expired facts are purged normally
```

#### Scenario: failure — global store file name is not mangled by normalization

```gherkin
      Given the data directory is empty
      When the global store is opened with the pseudo-project id `__global__`
      Then the created file is named exactly `__global__.sqlite`
      And the project id is not altered by normalization
```

#### Gates

- happy path: global store created, category column present
- edge: excluded from project list AND TTL expiry
- failure: file name exact, no normalization mangling

### Requirement: Scope routing on fact tools

The system SHALL route fact operations to the correct store based on scope, and trail session events to the caller's project store for global writes.

#### Scenario: happy path — global fact add writes to global store with project trail

```gherkin
      Given a project session is active in `proj-a`
      When a fact is added with scope `global` and target `memory`
      Then the fact row exists in the global store
      And the fact has category `memory`
      And a session event trail row exists in `proj-a`'s store
      And no session event trail row exists in the global store
```

#### Scenario: happy path — project fact add writes to project store (unchanged)

```gherkin
      Given a project session is active in `proj-a`
      When a fact is added with scope `project` and target `memory`
      Then the fact row exists in `proj-a`'s store
      And no fact row is created in the global store
```

#### Scenario: edge — global fact add with target `user` uses user category

```gherkin
      Given a project session is active in `proj-a`
      When a fact is added with scope `global` and target `user`
      Then the fact row in the global store has category `user`
      And the user char limit (1375) applies, not the memory limit (2200)
```

#### Scenario: failure — global fact add with unknown session fails cleanly

```gherkin
      Given no session exists in `proj-a`
      When a fact is added with scope `global` for a nonexistent session
      Then the operation returns an error
      And no fact row is created in the global store
      And the error message names the missing session
```

#### Gates

- happy path: global add → global store + project trail; project add → project store only
- edge: target `user` uses user category + user limit
- failure: unknown session → clean error, no partial write

### Requirement: Fused search across stores

The system SHALL search both the global and project stores when scope is `all`, returning results interleaved by rank and labeled with scope and category.

#### Scenario: happy path — fused search returns hits from both stores

```gherkin
      Given the global store contains a fact matching `responses`
      And the project store contains a fact matching `responses`
      When a fact search is performed with scope `all` and query `responses`
      Then both hits are returned
      And each hit is labeled with its scope (`global` or `project`)
      And each hit is labeled with its category
      And the results are interleaved by each store's own rank
```

#### Scenario: edge — fused search with no matches in one store

```gherkin
      Given the global store contains no facts matching `Golang`
      And the project store contains one fact matching `Golang`
      When a fact search is performed with scope `all` and query `Golang`
      Then only the project hit is returned
      And it is labeled with scope `project`
```

#### Scenario: edge — fused search trail goes to project store only

```gherkin
      Given a project session is active in `proj-a`
      When a fused search is performed with scope `all`
      Then a fact_search trail row exists in `proj-a`'s store
      And no fact_search trail row exists in the global store
```

#### Scenario: failure — fused search with empty stores returns empty

```gherkin
      Given the global store is empty
      And the project store is empty
      When a fact search is performed with scope `all`
      Then the result set is empty
      And no error is returned
```

#### Gates

- happy path: both stores searched, hits labeled, interleaved by rank
- edge: one-store match, trail to project only
- failure: empty stores → empty result, no error

### Requirement: Batch operations with atomic char limit

The system SHALL enforce a character limit on the global store, checked only on the batch's final result, and support substring matching for patch and remove operations.

#### Scenario: happy path — batch frees space and adds new fact

```gherkin
      Given the global memory store is at 2100 of 2200 characters
      When a batch operation removes 200 characters and adds 250 characters
      Then the batch succeeds
      And the final usage is 2150 of 2200 characters
```

#### Scenario: edge — overflow returns structured error with current entries

```gherkin
      Given the global memory store is at 2150 of 2200 characters
      When a fact of 100 characters is added
      Then the operation returns an overflow error
      And the error includes the current entries
      And the error includes the usage string `2,150/2,200`
      And no fact row is created
```

#### Scenario: edge — exact duplicate is rejected

```gherkin
      Given the global store contains a fact with content `user prefers dark mode`
      When a fact with content `user prefers dark mode` is added
      Then the operation succeeds with a no-duplicate-added message
      And no second fact row is created
```

#### Scenario: edge — patch uses substring matching

```gherkin
      Given the global store contains a fact `User prefers dark mode in all editors`
      When a patch is performed with old_text `dark mode in all editors` and content `light mode in VS Code`
      Then the fact is updated to `User prefers light mode in VS Code`
```

#### Scenario: failure — ambiguous substring match returns error

```gherkin
      Given the global store contains two facts both containing the text `dark mode`
      When a remove is performed with old_text `dark mode`
      Then the operation returns an ambiguous-match error
      And no fact is removed
```

#### Scenario: failure — no-match substring returns error with closest entries

```gherkin
      Given the global store contains a fact `user prefers dark mode`
      When a patch is performed with old_text `light mode in terminal`
      Then the operation returns a no-match error
      And the error includes the closest matching entries
```

#### Gates

- happy path: batch free+add succeeds within limit
- edge: overflow error with entries+usage, dedup, substring patch
- failure: ambiguous match, no-match with closest entries

### Requirement: Prefetch injection (frozen snapshot)

The system SHALL render a frozen snapshot of global facts into the system prompt at session start, with two labeled sections separated by section signs.

#### Scenario: happy path — prefetch renders two labeled sections

```gherkin
      Given the global store contains 2 facts with category `memory`
      And the global store contains 1 fact with category `user`
      When the prefetch snapshot is rendered
      Then the output contains a `GLOBAL MEMORY` section header
      And the output contains a `USER PROFILE` section header
      And the memory section shows usage percentage and character count
      And the user section shows usage percentage and character count
      And entries within each section are separated by section signs
      And entries are sorted by importance descending
```

#### Scenario: edge — prefetch is frozen mid-session

```gherkin
      Given a session has started and the prefetch snapshot was rendered
      When a new fact is added to the global store
      And the prefetch snapshot is requested again for the same session
      Then the second render returns the cached result from session start
      And the new fact is not included in the cached result
```

#### Scenario: failure — prefetch with empty store renders empty sections

```gherkin
      Given the global store is empty
      When the prefetch snapshot is rendered
      Then the output contains both section headers
      And both sections show 0 percent usage
      And no entries are listed
```

#### Gates

- happy path: two sections, § separators, usage headers, sorted by importance
- edge: frozen (cached) mid-session
- failure: empty store → empty sections, no error

### Requirement: Write approval for passive writes

The system SHALL stage passive (distill) fact writes in a durable pending queue when write approval is enabled, and apply them only on explicit approval.

#### Scenario: happy path — passive write is staged and approved

```gherkin
      Given write approval is enabled
      And a session ends with a distill pass that extracts one fact
      When the distill pass attempts to write the fact to the global store
      Then the fact is stored in the pending queue
      And the fact is not in the global facts table
      When the pending fact is approved
      Then the fact is in the global facts table
      And the fact is removed from the pending queue
```

#### Scenario: edge — pending queue survives server restart

```gherkin
      Given write approval is enabled
      And a distill pass has staged one fact in the pending queue
      When the server restarts (new service instance)
      And the pending list is requested
      Then the staged fact is still in the pending queue
```

#### Scenario: edge — explicit writes bypass the approval gate

```gherkin
      Given write approval is enabled
      When an agent explicitly adds a fact with scope `global` during a session
      Then the fact is written directly to the global store
      And the fact is not staged in the pending queue
```

#### Scenario: failure — rejected pending fact is discarded

```gherkin
      Given write approval is enabled
      And a distill pass has staged one fact in the pending queue
      When the pending fact is rejected
      Then the fact is removed from the pending queue
      And the fact is not in the global facts table
```

#### Gates

- happy path: stage → approve → in store
- edge: survives restart, explicit writes bypass
- failure: reject → discarded

### Requirement: Background distill (one pass, scope-classifying)

The system SHALL run one LLM pass at session end to extract durable facts, classifying each as project or global scope.

#### Scenario: happy path — distill extracts and routes facts by scope

```gherkin
      Given a session ended with a conversation containing both project and global facts
      When the distill pass runs
      Then one LLM call is made (not two)
      And the project-scoped fact is written to the project store
      And the global-scoped fact is written to the global store
```

#### Scenario: edge — distill coalescing (latest snapshot wins)

```gherkin
      Given two distill triggers fire for the same session
      When both triggers are processed
      Then only one LLM call is made
      And the later snapshot's facts are the ones written
```

#### Scenario: failure — distill failure does not break session close

```gherkin
      Given the LLM client is unreachable
      When the distill pass runs at session end
      Then the session closes normally
      And an error is logged
      And no facts are written
```

#### Gates

- happy path: one LLM call, facts routed by scope
- edge: coalescing (one call, latest wins)
- failure: LLM down → session closes, no facts written

### Requirement: Skill improvement loop (observe, inspect, amend, evaluate)

The system SHALL log skill execution outcomes (including failures), detect recurring failure patterns, propose amendments to skill scripts, and roll back amendments that do not improve outcomes.

#### Scenario: happy path — skill failure is logged with outcome and stderr

```gherkin
      Given a skill script that exits with code 1 and writes `connection timeout` to stderr
      When the skill is executed
      Then a usage row is created with outcome `failure`
      And the usage row captures `connection timeout` in the stderr field
```

#### Scenario: happy path — inspect detects recurring failure pattern

```gherkin
      Given a skill has 5 recorded executions
      And 3 of them failed with stderr `connection timeout`
      And 2 of them succeeded
      When the skill is inspected
      Then the evidence summary shows 5 total runs
      And the evidence summary shows 40 percent success rate
      And the evidence summary lists `connection timeout` as the top failure pattern
```

#### Scenario: happy path — amend writes a new file without modifying the original

```gherkin
      Given a skill with a recorded failure pattern
      And the LLM returns a proposed patch
      When the skill is amended
      Then a file named `<original>.amended` is created
      And the original skill file is unchanged
```

#### Scenario: edge — evaluate detects regression and rolls back

```gherkin
      Given a skill amendment was applied at time T
      And the success rate before T was 80 percent
      And the success rate after T is 50 percent
      When the amendment is evaluated
      Then the result is `regression`
      And the `.amended` file is discarded
      And the original skill file is restored
```

#### Scenario: edge — evaluate detects improvement and applies

```gherkin
      Given a skill amendment was applied at time T
      And the success rate before T was 50 percent
      And the success rate after T is 80 percent
      When the amendment is evaluated
      Then the result is `improvement`
      And the `.amended` file replaces the original
      And a git commit is created
```

#### Scenario: failure — skill timeout is logged as error outcome

```gherkin
      Given a skill script that exceeds its timeout
      When the skill is executed
      Then a usage row is created with outcome `error`
      And the usage row captures the timeout in the stderr field
```

#### Gates

- happy path: failure logged, pattern detected, amend writes new file
- edge: regression → rollback, improvement → apply + commit
- failure: timeout → error outcome

### Requirement: CLI surface for global memory

The system SHALL provide CLI commands for global memory operations including scope, target, and pending queue management.

#### Scenario: happy path — CLI adds a global fact with scope and target

```gherkin
      Given the global store exists
      When the CLI command `memory fact add --scope global --target memory` is run with content `test fact`
      Then a fact row with category `memory` exists in the global store
      And the CLI outputs the new fact id
```

#### Scenario: happy path — CLI lists and approves pending facts

```gherkin
      Given write approval is enabled
      And one fact is staged in the pending queue
      When the CLI command `memory global pending` is run
      Then the pending fact is listed with its content and target
      When the CLI command `memory global approve all` is run
      Then the fact is in the global store
      And the pending queue is empty
```

#### Scenario: edge — CLI prefetch prints the frozen snapshot

```gherkin
      Given the global store contains facts
      When the CLI command `memory global prefetch` is run
      Then the output contains the frozen snapshot format
      And the output contains both section headers
```

#### Scenario: failure — CLI with invalid scope returns usage error

```gherkin
      Given the global store exists
      When the CLI command `memory fact add --scope invalid` is run
      Then the CLI exits with a non-zero code
      And the error message lists the valid scope values
```

#### Gates

- happy path: add with scope+target, pending list + approve
- edge: prefetch prints snapshot
- failure: invalid scope → usage error
