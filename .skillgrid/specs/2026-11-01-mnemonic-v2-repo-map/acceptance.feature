# Mnemonic v2 — Repo-Map Function
#
# A single read-only, ranked (PageRank+IDF+context-prior), coverage-labeled,
# hub-flagged, token-budgeted navigation view over the project's code structure
# (the code-index graph) and its accumulated memory (the observations), auto-
# injected as the agent's first-minute context via the MCP session-start hook.
# The map orients; the agent drills (code_search/code_read/mem_lookup are separate
# tools). No new tables, no new migration, no Store extension (ADR-0047).
#
# Source: .skillgrid/specs/2026-11-01-mnemonic-v2-repo-map/specs/03-repo-map/000-overview.md
# Trace: blueprint.md Must-Haves (T1–T10) + per-task SATISFIES lines. The scenarios
# below mirror the blueprint's SATISFIES names 1:1.

## Requirements

### Requirement: the ranking engine and the thin end-to-end read (door check)

The system SHALL rank the file reference graph by deterministic PageRank (damping
0.85, max 100 iters, ε=1e-6) with edges weighted by reference count × IDF(symbol
name), apply the chat-context prior (the `context` param) and the `--focus` boost/
neighborhood, emit a token-budgeted signature-line tree, and — as the door check —
read the two legs (a seeded code-index graph + a seeded memory store) and emit the
coverage envelope.

#### Scenario: door-check-repo-map

```gherkin
      Given a seeded code-index graph with files symbols and edges
      And a seeded memory store with an observation mentioning a file path
      When I run repo map
      Then the ranked file tree is emitted under the token budget
      And every row carries the full signature line of its file
      And the coverage envelope is versioned and present
```

#### Scenario: ranking-deterministic

```gherkin
      Given the same seeded file reference graph
      When I run the PageRank twice with no context prior
      Then both runs return an identical file order
```

#### Scenario: ranking-idf-downweights-common

```gherkin
      Given a graph where a common symbol name and a rare name have equal reference count
      When I run the PageRank
      Then the file referencing the rare name ranks above the file referencing the common name
```

#### Scenario: ranking-context-prior-raises

```gherkin
      Given a seeded graph and a context prior naming internal/memory/pipeline.go
      When I run the PageRank with the prior
      Then pipeline.go and its direct dependents rank above the no-prior run
      And the rank delta is greater than zero
```

#### Scenario: signature-line-output

```gherkin
      Given a Go file with a method that has parameters and a return type
      When I run repo map over that file
      Then the emitted row contains the full func signature line
      And the row is not a bare identifier
```

#### Scenario: token-budget-walk-stops

```gherkin
      Given a ranked file list that exceeds the token budget
      When I render the map at the 2048 token default
      Then the walk stops at the budget
      And no file past the budget is emitted
```

#### Scenario: coverage-complete

```gherkin
      Given both legs are present and fresh
      When I run repo map
      Then the coverage status is complete
      And every source is authoritative
      And there are no issues
```

#### Scenario: coverage-partial-timed-out-leg

```gherkin
      Given the code-index leg timed out and the memory leg is present
      When I run repo map
      Then the coverage status is partial
      And the issues name the stale subtree
      And the map does not read as having no dependencies
```

#### Scenario: coverage-unavailable-both-missing

```gherkin
      Given the code-index is absent and the memory is empty
      When I run repo map
      Then the coverage status is unavailable
      And the counts are reported
      And the result is an empty graph with provenance, not a hard error
```

#### Gates
````
# G1: door check + ranking (deterministic/IDF/prior) + signature lines + token budget + coverage
# CHECK: cd mnemonic-v2 && go test ./internal/repomap/ -run 'TestDoorCheckRepoMapReadsAndRanks|TestRankingIDFDownweightsCommon|TestRankingContextPriorRaises|TestCoveragePartialTimedOutLeg|TestCoverageUnavailableBothMissing' -count=1
# EXPECT: ok  (go test success-only marker)
# EVIDENCE: pending
````

### Requirement: the hub flag and the memory join

The system SHALL flag files with fan-in ≥ hub_threshold (default 3) with a HUB tag
and importer count in the orientation view, and join the memory's topic hubs + per-
file developer considerations (saved observations FTS-matched to path/package) onto
the map. A file with no matching observation gets no consideration line.

#### Scenario: hub-flag-3-importers

```gherkin
      Given a file imported by three files and a file imported by two
      When I run repo map
      Then the three-importer file carries a HUB tag and the importer count three
      And the two-importer file carries no HUB tag
```

#### Scenario: hub-threshold-config

```gherkin
      Given a file imported by two files and hub_threshold set to two
      When I run repo map
      Then the two-importer file carries a HUB tag
```

#### Scenario: memory-join-surfaces-seeded-observation

```gherkin
      Given a seeded observation that mentions internal/memory/pipeline.go
      When I run repo map over pipeline.go
      Then the observation surfaces as the file's developer consideration
```

#### Scenario: memory-topic-hubs-by-importance

```gherkin
      Given observations across several topic keys with varying importance
      When I run repo map with kind topics
      Then the topic hubs are returned top-N ordered by importance
```

#### Scenario: memory-join-absent-not-fabricated

```gherkin
      Given an empty memory store and a file in the code graph
      When I run repo map over that file
      Then the file carries no developer consideration line
      And no consideration is fabricated
```

#### Gates
````
# G2: hub flag (3 importers, config threshold) + memory join (seeded surfaces, topics, absent not fabricated)
# CHECK: cd mnemonic-v2 && go test ./internal/repomap/ -run 'TestHubFlag3Importers|TestHubThresholdConfig|TestMemoryJoinSurfacesSeededObservation|TestMemoryTopicHubsByImportance|TestMemoryJoinAbsentNotFabricated' -count=1
# EXPECT: ok
# EVIDENCE: pending
````

### Requirement: the cross-graph kinds and the focus-mode disambiguation

The system SHALL serve the cross-graph kinds (focus/neighborhood, blast-radius,
temporal) that join the code graph and the memory in one call, and disambiguate a
`--focus <symbol>` (neighborhood) from a `--focus <path>` (boost) — the MCP tool
takes an explicit `mode` to remove the ambiguity.

#### Scenario: kind-blast-radius-cross-graph

```gherkin
      Given a symbol with code dependents and an observation that references it
      When I run repo map with kind blast-radius on the symbol
      Then the code reverse-BFS set is returned
      And the observations that reference the symbol are returned
```

#### Scenario: kind-temporal-weekly-bins

```gherkin
      Given code re-indexes and memory growth spanning two weeks
      When I run repo map with kind temporal
      Then weekly bins are returned for both the code re-indexes and the memory growth
```

#### Scenario: focus-mode-disambiguation

```gherkin
      Given a focus argument that is a symbol name and a focus argument that is a path
      When I run repo map with mode neighborhood on the symbol and mode boost on the path
      Then the symbol focus returns a cross-graph neighborhood
      And the path focus returns a ranking boost
      And the two are distinct results
```

#### Gates
````
# G3: cross-graph kinds (blast-radius, temporal) + focus-mode disambiguation
# CHECK: cd mnemonic-v2 && go test ./internal/repomap/ -run 'TestKindBlastRadiusCrossGraph|TestKindTemporalWeeklyBins|TestFocusModeDisambiguation' -count=1
# EXPECT: ok
# EVIDENCE: pending
````

### Requirement: the context envelope and the per-turn re-rank

The system SHALL return ONE context envelope on the session-start hook call (the
map + coverage + hub warnings + memory summary + intent prior + budget), auto-
injected when the config flag is on, and re-rank per turn in under 100ms when the
conversation context shifts. The cold-start inject budget is ~2× the mid-session
budget.

#### Scenario: context-envelope-one-call

```gherkin
      Given a session-start hook call with an intent prior
      When I build the context envelope
      Then one envelope is returned containing the map the coverage the hubs and the memory summary
      And the intent prior is carried
```

#### Scenario: session-start-auto-inject-flag

```gherkin
      Given the auto-inject config flag is off
      When the session-start hook runs
      Then the envelope is not injected
      Given the auto-inject config flag is on
      When the session-start hook runs
      Then the envelope is injected
```

#### Scenario: per-turn-rerank-under-100ms

```gherkin
      Given a warm map and a conversation context that adds a newly read file
      When I re-rank for the next turn
      Then the re-rank completes in under 100 milliseconds
      And the newly read file's rank rises
```

#### Scenario: dynamic-cold-start-budget

```gherkin
      Given the inject_tokens default of 1024
      When I build the envelope on cold start and on a mid-session turn
      Then the cold-start budget is approximately twice the mid-session budget
```

#### Gates
````
# G4: context envelope (one call, auto-inject flag) + per-turn re-rank (<100ms) + dynamic cold-start budget
# CHECK: cd mnemonic-v2 && go test ./internal/repomap/ -run 'TestContextEnvelopeOneCall|TestPerTurnRerankUnder100ms|TestDynamicColdStartBudget' -count=1
# EXPECT: ok
# EVIDENCE: pending
````

### Requirement: the emitted doc and the drift check

The system SHALL emit `.skillgrid/repo-map.md` with the per-file field contract
(Description, Developer Consideration, Maintenance Flag, Architectural Role,
Critical Dependencies) and a header stamp, gitignored by default with a commit
flag, and provide a CI-able `--drift` check (exit 0 fresh, exit 1 stale).

#### Scenario: emit-doc-per-file-fields

```gherkin
      Given a file with an observation and inbound and outbound edges
      When I emit the repo map doc
      Then the doc has all five per-file fields and the header stamp
```

#### Scenario: emit-gitignore-default-commit-flag

```gherkin
      Given the repo_map.commit flag is false
      When I emit the doc
      Then the doc path is gitignored
      Given the repo_map.commit flag is true
      When I emit the doc
      Then the doc path is not gitignored
```

#### Scenario: drift-exit-codes

```gherkin
      Given an emitted doc that matches the live state
      When I run the drift check
      Then the exit code is zero and the CI JSON is valid
      When the tree changes and I run the drift check
      Then the exit code is one
```

#### Gates
````
# G5: emit doc (per-file fields + stamp + gitignore/commit flag) + drift exit codes
# CHECK: cd mnemonic-v2 && go test ./internal/repomap/ -run 'TestEmitDocPerFileFields|TestDriftExitCodes' -count=1
# EXPECT: ok
# EVIDENCE: pending
````

### Requirement: the transport, the CLI, and the AGENTS.md trigger-table

The system SHALL expose the `repo_map` MCP tool (agent profile) + the session-start
hook + the `mnemonic repo map` CLI group (byte-identical to the MCP), add the lean
AGENTS.md trigger-table to block.md (≤6 lines, no new top-level section), and
re-emit the doc on the code-index fresh→lag flip.

#### Scenario: mcp-cli-byte-identical

```gherkin
      Given the same repo map args via the MCP tool and via the CLI
      When I compare the JSON output
      Then the MCP and CLI output are byte-identical
```

#### Scenario: agents-md-lean-trigger-table

```gherkin
      When I render the agent config block
      Then the Code Map sub-block is at most six lines
      And it is a when-to-command table plus one trust rule, not a prose paragraph
      And no new top-level section is added
```

#### Scenario: onboarding-reemit-on-lag

```gherkin
      Given an emitted doc and the code-index freshness that flips fresh to lag
      When onboarding runs
      Then the doc is re-emitted with a fresh stamp
```

#### Gates
````
# G6: transport + CLI + AGENTS.md trigger-table + onboarding re-emit
# CHECK: cd mnemonic-v2 && go test ./internal/transport/... ./cmd/... -run 'TestRepoMapMcpCliParity|TestAgentsMdLeanTriggerTable|TestOnboardingReemitOnLag|TestSessionStartAutoInjectFlag' -count=1
# EXPECT: ok
# EVIDENCE: pending
````

### Requirement: the doctor map-health and the QA measurability

The system SHALL report the map health (coverage status + leg status) on the
doctor surface, and — as the QA measurability scenario — surface a seeded
observation for a seeded file in the map's consideration join.

#### Scenario: doctor-map-health

```gherkin
      Given a project with both legs present
      When I run the doctor
      Then the map health reports the coverage status and the code-index and memory leg status
```

#### Scenario: qa-consideration-join-seeded

```gherkin
      Given a seeded observation for a seeded file
      When I run repo map over that file
      Then the observation surfaces in the map output's consideration join
```

#### Gates
````
# G7: doctor map-health + QA measurability (seeded consideration join)
# CHECK: cd mnemonic-v2 && go test ./internal/repomap/ -run 'TestDoctorMapHealth|TestQAConsiderationJoinSeeded' -count=1
# EXPECT: ok
# EVIDENCE: pending
````
