# Mnemonic — Family-First CLI + Split MCP Servers

Three single-purpose MCP servers under a family-first CLI (memory / code /
projectwiki), one mounted per client; renamed module (mnemonic-v2/ ->
mnemonic/, v0.2.0); strictly serial stdio (one tool call at a time, stdout
protocol-only); code-family responses at least as good as the CodeGraph and
gitnexus baselines (per-hit blast radius on search; a code_explore composite
returning source + call paths + blast radius in one call). No store changes:
the existing sync lease and DSN writer pragmas keep parallel servers safe.

Source: briefing.md (R1-R9, settled decisions 1-6) + blueprint.md
Must-Haves (T-1..T-13) + per-task SATISFIES lines.
Baselines: engram mcp 3.7.0 / openwiki mcp 0.9.0 / gitnexus mcp 1.6.12 /
CodeGraph serve --mcp 1.6.0 (probe inventory in the briefing).

## Requirements

### Requirement: the module rename is mechanical and complete (door check)

The system SHALL build and test green at the renamed module path
(`mnemonic/`, module `github.com/devopstales/skillgrid/mnemonic`,
workspace `use ./mnemonic`, Taskfile build entry updated), with no
`mnemonic-v2` references left in go.mod, go.work, or the Taskfile.

#### Scenario: rename-builds-and-tests-green-at-new-path

```gherkin
  Given the module renamed from mnemonic-v2 to mnemonic
  When I build and test the workspace
  Then every package builds and the full suite passes
  And the task build entry resolves the module at the new path
```

#### Scenario: rename-leaves-no-module-references

```gherkin
  Given the completed rename
  When I search go.mod and go.work and the Taskfile for mnemonic-v2
  Then no reference remains
```

#### Scenario: rename-keeps-v1-untouched

```gherkin
  Given the completed rename
  When I build the skillgrid-cli module
  Then it builds without importing the renamed module path
```

### Requirement: the CLI is family-first with a clean cutover

The system SHALL dispatch `mnemonic memory <verb>`, `mnemonic code <verb>`,
and `mnemonic projectwiki <verb>` with unchanged per-verb behavior, expose
`mcp` as a verb under each family, remove the top-level `mnemonic mcp` and
`mnemonic serve`, and answer removed paths with a non-zero exit and a
"did you mean" suggestion. No aliases.

#### Scenario: new-family-paths-dispatch

```gherkin
  Given the family-first CLI
  When I run mnemonic code search with a query
  And I run mnemonic memory save with a session
  And I run mnemonic projectwiki query with a term
  Then each command executes its existing behavior
```

#### Scenario: bare-code-defaults-to-index

```gherkin
  Given the family-first CLI
  When I run mnemonic code with no sub-verb
  Then the code index runs (the default verb is unchanged)
```

#### Scenario: old-paths-suggest-their-replacement

```gherkin
  Given the family-first CLI
  When I run mnemonic index
  And I run mnemonic mem save
  And I run mnemonic mcp
  Then each exits non-zero
  And each prints a did-you-mean suggestion naming the new path
```

#### Scenario: serve-is-gone

```gherkin
  Given the family-first CLI
  When I run mnemonic serve
  Then it exits non-zero with a did-you-mean suggestion
```

### Requirement: each family serves exactly its own tools (one server per client)

The system SHALL start a stdio MCP server per family via
`mnemonic <family> mcp --profile agent|all`; each server lists only its
family's tools (memory 22/28, code 19/38 before the composite lands,
projectwiki 8/12 for agent/all) and no other family's tool in any profile.

#### Scenario: memory-server-serves-memory-only

```gherkin
  Given the mnemonic memory mcp server in agent profile
  When I list its tools
  Then I see exactly the 22 agent memory tools
  And no code or projectwiki tool appears
```

#### Scenario: all-profile-adds-admin-tools-only

```gherkin
  Given the mnemonic memory mcp server in all profile
  When I list its tools
  Then I see the 28 memory tools (22 agent + 6 admin)
  And no tool from another family appears
```

#### Scenario: code-and-wiki-servers-are-isolated

```gherkin
  Given the mnemonic code mcp and mnemonic projectwiki mcp servers
  When I list both servers' tools
  Then each lists only its own family (code 19, projectwiki 8 in agent profile)
  And neither lists a memory tool
```

#### Scenario: unknown-profile-refuses

```gherkin
  Given the family-first CLI
  When I run mnemonic memory mcp with an unknown profile
  Then it exits non-zero naming the valid profiles
```

### Requirement: each initialize response follows its baseline

The system SHALL answer `initialize` per family: `mnemonic-memory` 0.2.0 with
`{"tools":{"listChanged":true}}` and the engram-derived instructions (core
tool list rendered per profile); `mnemonic-projectwiki` 0.2.0 with
`{"tools":{"listChanged":true}}` and the openwiki-derived instructions
(lifecycle rewritten for generate/validate/lint); `mnemonic-code` 0.2.0 with
`{"tools":{},"resources":{},"prompts":{}}` and no instructions.

#### Scenario: memory-initialize-follows-engram

```gherkin
  Given the mnemonic memory mcp server in agent profile
  When I initialize it
  Then the server identity is mnemonic-memory version 0.2.0
  And the tools capability advertises list-changed
  And the instructions name only the agent-profile core tools with the deferred list
```

#### Scenario: projectwiki-initialize-follows-openwiki

```gherkin
  Given the mnemonic projectwiki mcp server
  When I initialize it
  Then the server identity is mnemonic-projectwiki version 0.2.0
  And the instructions carry the retrieval discipline and claims rules
  And the lifecycle section names projectwiki generate validate and lint
  And no workspace or link language remains
```

#### Scenario: code-initialize-follows-gitnexus

```gherkin
  Given the mnemonic code mcp server
  When I initialize it
  Then the server identity is mnemonic-code version 0.2.0
  And the capabilities are tools resources and prompts all empty
  And no instructions field is present
```

### Requirement: responses are strictly serial and stdout is protocol-only

The system SHALL process at most one tool call at a time (the second of two
concurrent calls is not processed until the first's response is written,
including beyond the queue), and after serve starts, stdout carries exactly
the protocol frames while all diagnostics go to stderr.

#### Scenario: concurrent-calls-answer-one-at-a-time

```gherkin
  Given a mnemonic memory mcp server whose save takes a measurable time
  When I send two tool calls concurrently
  Then the second call is not processed until the first response is written
  And each response is a single complete JSON-RPC frame
```

#### Scenario: deep-concurrency-stays-serial

```gherkin
  Given a mnemonic code mcp server
  When I send more than one hundred tool calls before any response
  Then no call is processed concurrently with another
```

#### Scenario: stdout-carries-only-protocol-frames

```gherkin
  Given a mnemonic memory mcp server that emits a post-startup diagnostic
  When I read its stdout after the diagnostic fires
  Then stdout contains exactly the protocol frames
  And the diagnostic appears on stderr
```

### Requirement: code search answers are self-contained per hit

The system SHALL return, for each `code_search` hit on an indexed symbol, a
blast-radius line (distinct caller count + test files) and a snippet;
unresolved hits degrade gracefully (fields omitted, no error); edge lists
never contain duplicates.

#### Scenario: search-hit-carries-blast-radius-and-snippet

```gherkin
  Given an indexed symbol with two distinct callers and one test file
  When I search for it via mnemonic code mcp
  Then the hit reports two callers and the test file
  And the hit carries the symbol's first source line as a snippet
```

#### Scenario: unresolved-hit-degrades-gracefully

```gherkin
  Given a search hit whose symbol did not resolve in the graph
  When I search for it
  Then the hit is returned with blast-radius fields omitted
  And no error is raised
```

#### Scenario: duplicate-edges-are-reported-once

```gherkin
  Given a symbol with the same caller edge recorded twice
  When I search for it
  Then the blast-radius line counts the caller exactly once
```

### Requirement: code_explore answers in one call at baseline quality

The system SHALL expose `code_explore` (agent profile) returning in one call:
the query header, symbol/file counts, a blast-radius section (callers + test
files per symbol), the provenance note, and verbatim line-numbered source per
file with the edge-summary line; EXTRACTED/INFERRED per-edge tags; per-hit
fusion-leg provenance; and the degraded flag only when a leg degraded.

#### Scenario: explore-returns-the-full-baseline-shape

```gherkin
  Given an indexed repo with a symbol that has callers and a test file
  When I call code_explore naming that symbol
  Then the response opens with the query header and symbol and file counts
  And the blast-radius section lists the symbol with its callers and test file
  And each file's source appears verbatim with line numbers and an edge summary
  And no edge is listed twice
```

#### Scenario: explore-tags-inferred-edges

```gherkin
  Given a call edge recorded as inferred by the extractor
  When I call code_explore over it
  Then the edge is tagged as inferred
  And extracted edges carry no tag
```

#### Scenario: explore-on-empty-index-degrades-silently

```gherkin
  Given a directory with no code index and no git repository
  When I call code_explore
  Then the response reports zero symbols across zero files
  And no source blocks and no degraded flag appear
  And no error is raised
```

#### Scenario: explore-declares-a-degraded-leg

```gherkin
  Given an index where one search leg reports degradation
  When I call code_explore
  Then the response carries the degraded flag naming the leg
```

### Requirement: parallel family servers share one store safely

The system SHALL let two family MCP servers run in parallel on one store
without a sync double-push and without database-locked errors under standard
load.

#### Scenario: two-servers-one-store-parallel

```gherkin
  Given a store with autosync enabled
  When I run mnemonic memory mcp and mnemonic code mcp in parallel
  Then at most one process holds the sync lease at any time
  And neither process reports a database-is-locked error under standard load
```

### Requirement: every live reference and fresh install points at the new surface

The system SHALL leave no live skill/doc/vault reference to the old paths
(excluding the archive and historical decision records), and the installer
SHALL write live per-family MCP command lines into client configs.

#### Scenario: no-stale-path-references-remain

```gherkin
  Given the completed docs and convention updates
  When I search live skills docs and vault pages for the old command paths and mnemonic-v2
  Then no reference remains outside the archive and historical decision records
```

#### Scenario: fresh-install-writes-live-mcp-commands

```gherkin
  Given a fresh client install
  When the installer writes the MCP configuration
  Then each client entry uses a mnemonic family mcp command with the agent profile
  And no entry uses the removed bare mnemonic mcp form
```

#### Scenario: architecture-page-matches-the-new-tree

```gherkin
  Given the vault architecture page after the update
  When I validate the page's anchors against the working tree
  Then the module paths and CLI dispatch table name the family-first tree
  And the pinned server entrypoint signature matches the new code
```

### Requirement: the decision is recorded as an ADR

The system SHALL record the split as an ADR (next free number) superseding
the combined-server decision, indexed in the ADR index and this change's
manifest, with the vault reindexed in the same edit.

#### Scenario: adr-lands-and-indexes

```gherkin
  Given the verified split
  When the ADR is written and indexed
  Then the ADR file exists at the next free sequence number
  And the ADR index names it as superseding the combined-server decision
  And this change's manifest lists it
```

## Gates

### Requirement: the quality gate holds

The full suite and build hold for both workspace modules, with the
classification's L4 additions (race, mutation sampling on the renderer and
the serial option, Trivy advisory).

#### Scenario: full-suite-green-race

```gherkin
  Given the completed change
  When I run the full test suite with the race detector for both modules
  Then every package passes
```

#### Scenario: mutation-of-serial-option-is-caught

```gherkin
  Given the completed change
  When I remove the single-worker stdio option (mutation)
  Then the concurrent-calls scenario fails
```

#### Scenario: mutation-of-renderer-is-caught

```gherkin
  Given the completed change
  When I drop the blast-radius section from the explore renderer (mutation)
  Then the explore shape scenario fails
```
