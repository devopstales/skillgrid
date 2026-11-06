# Mnemonic-v2 Agent Plugins (memory / code-index / knowledge) — OpenCode + Pi agent plugins
#
#The OpenCode and Pi plugins that expose the mnemonic-v2 engine, ported from
#the engram plugin baseline with full tool-surface parity, consumed over the
#v2 MCP stdio surface; the memory save pipeline emits the SaveEvent consumed
#by the brain-side llm-wiki. Amended 2026-11-05: three plugin families per
#host — memory (mem_*), code-index (code_*), knowledge (kb_*, gated on the
#KB function ship). Parity source of truth: the v2 module's registered tool
#sets; the v1 module surface (HTTP :7438, fact_*) is legacy, out of parity.
#Amended 2026-11-05: pi-research fold R6 (capability gating, lifecycle,
#described degradation, consumer queue, read guard, skill+CLI distribution) —
#memory + code-index families, both hosts; the knowledge-base plugin is
#excluded (user direction).
#
#Source: .skillgrid/change-specs/2026-11-04-mnemonic-v2-memory-plugins/briefing.md
#Trace: briefing.md Requirements R1-R6; blueprint tasks SATISFIES lines.
#
# Format rules:
# - Steps in the ```gherkin fence are indented 6 spaces (verbatim — the extractor does NOT re-indent)
# - No Gherkin tags in the spec — selection/trace is via the SATISFIES field in blueprint/tasks
# - One blank line between the heading and the fence is fine
# - Gates: each requirement carries a `#### Gates` block — the runnable shadow of
#   its happy-path and failure scenarios. Author it BEFORE implementing.

## Requirements

### Requirement: Full agent-facing memory tool-surface parity

The system SHALL expose the full registered v2 memory tool surface (28 `mem_*`:
22 agent-profile + 6 admin, per `mnemonic-v2/internal/transport/mcp/contract_test.go`)
through the OpenCode and Pi memory plugins, backed by the mnemonic-v2 engine
(MCP stdio). The engram baseline (27-tool MCP surface, 23-tool `agent`
profile) is the behavior floor, not the ceiling. The v1 `fact_*` family is
out of the parity target (v1-HTTP legacy).

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
# CHECK: go test ./mnemonic-v2/... -count=1
# EXPECT: ok
# EVIDENCE: pending
````
````
# G2: plugin fail-open holds for the failure scenario
# CHECK: go test ./mnemonic-v2/... -count=1
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
# CHECK: go test ./mnemonic-v2/... -count=1
# EXPECT: ok
# EVIDENCE: pending
````
````
# G4: retry/self-heal behavior port holds
# CHECK: go test ./mnemonic-v2/... -count=1
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
# CHECK: go test ./mnemonic-v2/... -count=1
# EXPECT: ok
# EVIDENCE: pending
````
````
# G6: wiki ingestion failure does not roll back the memory save
# CHECK: go test ./mnemonic-v2/... -count=1
# EXPECT: ok
# EVIDENCE: pending
````

### Requirement: Code-index plugin parity

The system SHALL expose the full v2 `code_*` tool surface (25 tools, per the
registered set of the `2026-10-31-mnemonic-v2-code-index` change) through the
OpenCode and Pi code-index plugins.

#### Scenario: Happy path — agent searches code through the code-index plugin

```gherkin
      Given an agent with the code-index plugin enabled on an indexed project
      When the agent calls the code search tool for an unknown area
      Then the search returns ranked indexed hits
      And the agent reads the narrowed source range through the code read tool
```

#### Scenario: Edge case — stale index refresh workflow

```gherkin
      Given an agent whose code index is stale after a branch switch
      When the code status tool reports stale
      Then the agent refreshes through the code index tool before searching
      And the subsequent search reflects the current tree
```

#### Scenario: Failure state — memory server unreachable through the code-index plugin

```gherkin
      Given an agent with the code-index plugin enabled
      When a code tool is invoked while the mnemonic server is unreachable
      Then the plugin fails open and the agent continues
```

#### Gates
````
# G7: full code_* tool-surface parity holds (both hosts; registered set as source of truth)
# CHECK: go test ./mnemonic-v2/... -count=1
# EXPECT: ok
# EVIDENCE: pending
````
````
# G8: code-index plugin fail-open holds for the failure scenario
# CHECK: go test ./mnemonic-v2/... -count=1 && cd plugins/opencode && npm test
# EXPECT: ok
# EVIDENCE: pending
````

### Requirement: Knowledge-base plugin

The system SHALL expose the `kb_*` tool surface (12 tools per
`2026-10-31-mnemonic-v2-knowledge-base` `specs/008-transport.md`) through the
OpenCode and Pi knowledge-base plugins.

> **Gated:** scenarios and gates are verified once the KB function
> (`2026-10-31-mnemonic-v2-knowledge-base`) ships its `kb_*` routes. Until
> then, parity is asserted against the spec-pinned tool table.

#### Scenario: Happy path — agent queries the knowledge base through the plugin

```gherkin
      Given an agent with the knowledge-base plugin enabled on a bootstrapped vault
      When the agent calls the kb_query tool for a question
      Then the query returns ranked page links with one-line snippets
      And the agent reads a full page through the kb_read tool
```

#### Scenario: Edge case — async KB write does not double-start

```gherkin
      Given an agent that starts a kb_generate run through the plugin
      When the run returns a run_id and the agent repeats the start
      Then the repeated start is rejected as an active run
      And the agent polls the run status to completion
```

#### Scenario: Failure state — degraded KB run surfaces the envelope, never fails the agent

```gherkin
      Given a kb tool call that runs degraded (LLM or graph absent)
      When the plugin dispatches the call
      Then the response carries the degradation flag with its reason
      And the agent continues without a tool error
```

#### Gates
````
# G9: kb_* tool-surface parity holds (spec-pinned table until KB tools land; registered set after)
# CHECK: go test ./mnemonic-v2/... -count=1
# EXPECT: ok
# EVIDENCE: pending (gated on 2026-10-31-mnemonic-v2-knowledge-base ship)
````
````
# G10: degraded KB run surfaces the envelope without failing the agent
# CHECK: go test ./mnemonic-v2/... -count=1 && cd plugins/opencode && npm test
# EXPECT: ok
# EVIDENCE: pending (gated on 2026-10-31-mnemonic-v2-knowledge-base ship)
````

### Requirement: Plugin engineering — capability gating, lifecycle, described degradation (pi-research fold)

Validated by pi-research (2026-11-05). User direction: all folds land on the
**memory + code-index plugins, both hosts** — the knowledge-base plugin is
excluded from this requirement.

#### Scenario: Happy path — admin memory tools gated by profile without restart

```gherkin
      Given a Pi agent with the memory plugin enabled on the default agent profile
      When the six admin mem_* tools are inspected
      Then they are registered but hidden from the model
      And enabling the all profile re-exposes them without a pi restart
```

#### Scenario: Edge case — code-index tools absent on the server self-guard

```gherkin
      Given an agent with the code-index plugin enabled
      And the server capability probe reports no code-index family
      When a code_* tool is invoked
      Then the tools are hidden on Pi or return the self-guard unavailable envelope on OpenCode
      And the agent continues without a tool error
```

#### Scenario: Happy path — session shutdown drains the MCP child within budget

```gherkin
      Given a plugin session with a live mnemonic mcp child process
      When the host session shuts down with reason quit
      Then the child is drained within the bounded teardown budget
      And a hung drain trips the force-exit watchdog leaving no orphaned child
```

#### Scenario: Happy path — compaction recovery fires from the named compaction events

```gherkin
      Given a Pi session that triggers compaction
      When the session_before_compact and session_compact events fire
      Then the recovery path persists prior work through the session summary tool
```

#### Scenario: Edge case — subagent read of a secret path is denied

```gherkin
      Given a subagent context invoking the code read tool
      When the read targets a secret file or a path outside the project root
      Then the read is denied with a read-denied reason
      And an in-project read resolves normally
```

#### Scenario: Failure state — embedder unavailable degrades with a described reason

```gherkin
      Given a project whose index has the semantic leg unavailable
      When the agent calls the code search tool
      Then the search returns FTS-leg hits without failing
      And the response and the health surface name the missing embedder with its repair
      And the degradation is reported as disabled, not unhealthy
```

#### Gates
````
# G11: capability gating + self-guard + read guard hold (both hosts)
# CHECK: cd plugins/pi/mnemonic && npm test && cd plugins/opencode && npm test
# EXPECT: ok
# EVIDENCE: pending
````
````
# G12: shutdown drain/watchdog + compaction events + described-OFF hold (both hosts)
# CHECK: cd plugins/pi/mnemonic && npm test && cd plugins/pi/codeindex && npm test && cd plugins/opencode && npm test
# EXPECT: ok
# EVIDENCE: pending (codeindex half gated on 2026-10-31-mnemonic-v2-code-index ship)
````

## Traceability

- Requirements R1-R6 map to scenarios above; blueprint tasks carry `SATISFIES` lines.
- Design approval recorded 2026-11-04 ("look good"); 2026-11-05 amendment
  (3-family split, KB plugin gated on the brain function) per user direction;
  2026-11-05 amendment (pi-research fold R6 — scoped to the memory +
  code-index families on both hosts, knowledge-base plugin excluded) per user
  direction.
