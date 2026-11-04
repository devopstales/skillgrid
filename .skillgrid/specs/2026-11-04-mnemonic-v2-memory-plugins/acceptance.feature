# Mnemonic-v2 Memory Plugins — OpenCode + Pi agent plugins
#
#The OpenCode and Pi plugins that expose the mnemonic-v2 memory function, ported
#from the engram plugin baseline with full tool-surface parity, backed by the
#mnemonic/mnemonic-v2 memory server; the memory save pipeline emits the SaveEvent
#consumed by the brain-side llm-wiki.
#
#Source: .skillgrid/specs/2026-11-04-mnemonic-v2-memory-plugins/briefing.md
#Trace: briefing.md Requirements R1-R3; blueprint tasks SATISFIES lines.
#
# Format rules:
# - Steps in the ```gherkin fence are indented 6 spaces (verbatim — the extractor does NOT re-indent)
# - No Gherkin tags in the spec — selection/trace is via the SATISFIES field in blueprint/tasks
# - One blank line between the heading and the fence is fine
# - Gates: each requirement carries a `#### Gates` block — the runnable shadow of
#   its happy-path and failure scenarios. Author it BEFORE implementing.

## Requirements

### Requirement: Full agent-facing memory tool-surface parity

The system SHALL expose the full engram-baseline agent-facing memory tool surface
through the OpenCode and Pi plugins, backed by the mnemonic/mnemonic-v2 memory
function.

#### Scenario: Happy path — agent saves a new memory through the plugin tool

```gherkin
      Given an OpenCode or Pi agent with the memory plugin enabled
      When the agent calls the memory save tool for a new topic
      Then the mnemonic/mnemonic-v2 memory function persists the observation
```

#### Scenario: Edge case — agent looks up an unknown topic through the plugin tool

```gherkin
      Given an agent invoking a memory lookup tool through the plugin
      When the lookup names a topic with no stored observation
      Then the lookup returns a miss without erroring
```

#### Scenario: Failure state — memory server unreachable through the plugin

```gherkin
      Given an agent with the memory plugin enabled
      When a memory tool is invoked while the mnemonic server is unreachable
      Then the plugin fails open and the agent continues
```

#### Gates
````
# G1: full memory tool-surface parity holds for the happy-path save scenario
# CHECK: go test ./mnemonic/... -count=1
# EXPECT: ok
# EVIDENCE: pending
````

# G2: plugin fail-open holds for the failure scenario
# CHECK: go test ./mnemonic/... -count=1
# EXPECT: ok
# EVIDENCE: pending
````

### Requirement: Engram-proven behavior port

The system SHALL port engram-proven plugin behaviors into the existing skillgrid
plugin packages without behavior drift.

#### Scenario: Happy path — compaction recovery persists prior work

```gherkin
      Given an agent that experienced context compaction
      When the compaction recovery path runs before_agent_start
      Then the prior work is persisted through the session summary tool
```

#### Scenario: Edge case — passive capture redaction

```gherkin
      Given a captured tool output containing a private span
      When the passive capture path evaluates the output
      Then the stored capture has no private span and honors the redaction policy
```

#### Scenario: Failure state — retry and self-heal after transient failure

```gherkin
      Given a memory tool dispatch that fails transiently
      When the plugin retries per its retry policy
      Then the retry succeeds without duplicating the write
```

#### Gates
````
# G3: compaction recovery behavior port holds
# CHECK: go test ./mnemonic/... -count=1
# EXPECT: ok
# EVIDENCE: pending
````

# G4: retry/self-heal behavior port holds
# CHECK: go test ./mnemonic/... -count=1
# EXPECT: ok
# EVIDENCE: pending
````

### Requirement: Memory→wiki handoff contract

The system SHALL emit the SaveEvent consumed by the brain-side wiki through the
memory save pipeline.

#### Scenario: Happy path — save emits a SaveEvent for the wiki

```gherkin
      Given a saved observation in the mnemonic memory function
      When the save pipeline completes
      Then a save event is emitted carrying the observation identity
      And the brain-side wiki can ingest it via its capture tool
```

#### Scenario: Edge case — wiki vault not initialized

```gherkin
      Given a memory save that completes while the wiki vault is not initialized
      When the save event is routed to the wiki
      Then the wiki handles the uninitialized vault without failing the save
```

#### Scenario: Failure state — wiki ingestion failure does not lose the memory save

```gherkin
      Given a memory save that completes
      When the brain-side wiki ingestion fails
      Then the memory save remains committed and the failure is surfaced
```

#### Gates
````
# G5: memory save emits a SaveEvent consumable by the wiki
# CHECK: go test ./mnemonic/... -count=1
# EXPECT: ok
# EVIDENCE: pending
````

# G6: wiki ingestion failure does not roll back the memory save
# CHECK: go test ./mnemonic/... -count=1
# EXPECT: ok
# EVIDENCE: pending
````

## Traceability

- Requirements R1-R3 map to scenarios above; blueprint tasks will carry `SATISFIES` lines.
- Design approval recorded 2026-11-04 ("look good"); implementation has not started.
