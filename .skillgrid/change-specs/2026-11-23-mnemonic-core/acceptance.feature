# mnemonic memory-core
> **STATUS:** `draft` (2026-11-23)
>
>
> Source: .skillgrid/change-specs/2026-11-23-mnemonic-core/
> Trace: change briefing R1–R9 (Definition of Done); blueprint tasks SATISFIES lines (truths T1–T14).
>
> Format rules:
> - Steps in the ```gherkin fence are indented 6 spaces (verbatim — the extractor does NOT re-indent)
> - No Gherkin tags in the spec — selection/trace is via the SATISFIES field in blueprint/tasks
> - One blank line between the heading and the fence is fine

## Requirements

### Requirement: Per-project storage

The system SHALL store each project's observations in one portable SQLite file under the data directory, so memory persists across processes and moves by copying that file.

#### Scenario: happy path — save in one process, search in a fresh process

```gherkin
      Given a project with no store yet
      When an observation is saved via the memory tool
      And a fresh process searches the same project for the observation's text
      Then the search returns the observation with its title and provenance
      And the project's single SQLite file exists under the data directory
```

#### Scenario: edge — nested project path yields a sanitized slug

```gherkin
      Given a project detected from the nested path "org/team/app"
      When an observation is saved
      Then the store file is created under a slug without path separators
      And two different projects never share a store file
```

#### Scenario: failure — data directory not writable

```gherkin
      Given a data directory that is not writable
      When an observation is saved
      Then the result carries the stable code for an unavailable store
      And the process exits with the handler-error code with an envelope on stdout
```

#### Gates
> G1: save in one process, search in a fresh process (mirrors happy path)
>   CHECK: node .skillgrid/acceptance-tests/gates/mnemonic-r1-persist.mjs
>   EXPECT: MNEMONIC_R1_PERSIST_OK
>   EVIDENCE: pending
> G2: data directory not writable (mirrors failure)
>   CHECK: node .skillgrid/acceptance-tests/gates/mnemonic-r1-unavailable.mjs
>   EXPECT: MNEMONIC_R1_UNAVAILABLE_OK
>   EVIDENCE: pending

### Requirement: In-file vectors

The system SHALL keep vectors inside the project's SQLite file — the BLOB as source of truth, an optional in-file ANN index — and SHALL NOT require an external vector database.

#### Scenario: happy path — vector leg contributes to recall

```gherkin
      Given observations stored with an active embedder
      When a search runs with both legs available
      Then the result reports per-leg provenance including the vector leg
      And the vector index lives inside the project's SQLite file
```

#### Scenario: edge — dimension change rebuilds the vector layer

```gherkin
      Given a store embedded at one dimension
      When the active embedder dimension changes and the store rebuilds its vectors
      Then search returns results with no stale-dimension vectors
      And every stored vector matches the current dimension
```

#### Scenario: failure — vector extension fails to load

```gherkin
      Given a build where the vector extension cannot load
      When a search runs
      Then brute-force ranking serves the vector leg
      And the envelope flags the result as degraded with a reason
```

#### Gates
> G3: vector leg contributes to recall (mirrors happy path)
>   CHECK: node .skillgrid/acceptance-tests/gates/mnemonic-r2-vector-leg.mjs
>   EXPECT: MNEMONIC_R2_VECTOR_LEG_OK
>   EVIDENCE: pending
> G4: vector extension fails to load (mirrors failure)
>   CHECK: node .skillgrid/acceptance-tests/gates/mnemonic-r2-no-ext.mjs
>   EXPECT: MNEMONIC_R2_NO_EXT_OK
>   EVIDENCE: pending

### Requirement: Embedder tiers

The system SHALL embed with a local ONNX model by default, an opt-in OpenAI-compatible API, and a deterministic hash floor, so recall works with zero network and zero model files.

#### Scenario: happy path — local model embeds with a semantic ranking

```gherkin
      Given the local model is present and verified
      When a paraphrase of a stored observation is searched
      Then the paraphrase ranks above an unrelated query
      And stored vectors are stamped with the model name and dimension
```

#### Scenario: edge — model file missing falls to the floor

```gherkin
      Given the local model file is absent
      When an observation is saved and searched
      Then the hash floor served the embeddings
      And the envelope flags the result as degraded with a reason
```

#### Scenario: failure — model load error never blocks save

```gherkin
      Given the local model fails to load
      When an observation is saved
      Then one bounded retry happens
      And the save succeeds on the hash floor with a degraded flag
```

#### Gates
> G5: local model embeds with a semantic ranking (mirrors happy path)
>   CHECK: node .skillgrid/acceptance-tests/gates/mnemonic-r3-semantic.mjs
>   EXPECT: MNEMONIC_R3_SEMANTIC_OK
>   EVIDENCE: pending
> G6: model load error never blocks save (mirrors failure)
>   CHECK: node .skillgrid/acceptance-tests/gates/mnemonic-r3-load-error.mjs
>   EXPECT: MNEMONIC_R3_LOAD_ERROR_OK
>   EVIDENCE: pending

### Requirement: Memory model

The system SHALL store observations as upserts keyed by topic and scope, deduplicate exact repeats, strip private spans, and bind saves to a live session.

#### Scenario: happy path — re-save revises, never duplicates

```gherkin
      Given an observation saved with a topic key and project scope
      When the same topic key and scope is saved again with new content
      Then the observation is revised in place with an incremented revision count
      And no second row exists for that topic key and scope
```

#### Scenario: edge — private spans stripped and exact repeats counted

```gherkin
      Given content containing a private span
      When it is saved
      Then the stored content does not contain the private span's text
      And an exact repeat without a topic key increments the duplicate count instead of storing a new row
```

#### Scenario: failure — save without a live session

```gherkin
      Given no live session
      When an observation is saved
      Then the result fails with the stable code for a missing session
```

#### Gates
> G7: re-save revises, never duplicates (mirrors happy path)
>   CHECK: node .skillgrid/acceptance-tests/gates/mnemonic-r4-upsert.mjs
>   EXPECT: MNEMONIC_R4_UPSERT_OK
>   EVIDENCE: pending
> G8: save without a live session (mirrors failure)
>   CHECK: node .skillgrid/acceptance-tests/gates/mnemonic-r4-no-session.mjs
>   EXPECT: MNEMONIC_R4_NO_SESSION_OK
>   EVIDENCE: pending

### Requirement: Hybrid recall

The system SHALL recall by fusing the full-text and vector legs into one ranking with provenance, honour scope isolation, and write a trail row for every retrieval.

#### Scenario: happy path — fused ranking with provenance and trail

```gherkin
      Given observations stored with embeddings
      When a search runs with both legs available
      Then the results are fused into one ranking with per-leg provenance
      And a trail row records the query, route and hits
```

#### Scenario: edge — query text containing full-text operators is literal

```gherkin
      Given an observation whose title contains the phrase "fixed the cache"
      When a search uses that phrase as the query
      Then the search succeeds
      And the phrase is treated as literal text, not as full-text syntax
```

#### Scenario: failure — vector leg absent still returns full-text results

```gherkin
      Given the vector leg is unavailable
      When a search with matching full-text terms runs
      Then the full-text matches are returned
      And the envelope flags the result as degraded
```

#### Gates
> G9: fused ranking with provenance and trail (mirrors happy path)
>   CHECK: node .skillgrid/acceptance-tests/gates/mnemonic-r5-fusion.mjs
>   EXPECT: MNEMONIC_R5_FUSION_OK
>   EVIDENCE: pending
> G10: vector leg absent still returns full-text results (mirrors failure)
>   CHECK: node .skillgrid/acceptance-tests/gates/mnemonic-r5-fts-only.mjs
>   EXPECT: MNEMONIC_R5_FTS_ONLY_OK
>   EVIDENCE: pending

### Requirement: MCP interface

The system SHALL expose the memory tool as the stdio MCP server `mnemonic-memory` with the full tool surface, every tool returning the shared envelope.

#### Scenario: happy path — full tool surface listed and callable

```gherkin
      Given the MCP server is running
      When the tool list is requested
      Then all twelve memory tools are listed
      And calling the search tool returns the shared envelope
```

#### Scenario: edge — bad argument yields a stable error

```gherkin
      Given the MCP server is running
      When a tool is called with a missing required argument
      Then an MCP error is returned carrying the stable error code
```

#### Scenario: failure — store busy returns an error envelope

```gherkin
      Given the store is held by another writer past the busy timeout
      When a saving tool is called
      Then the result is an error envelope with the stable code for a busy store
```

#### Gates
> G11: full tool surface listed and callable (mirrors happy path)
>   CHECK: node .skillgrid/acceptance-tests/gates/mnemonic-r6-tools.mjs
>   EXPECT: MNEMONIC_R6_TOOLS_OK
>   EVIDENCE: pending
> G12: store busy returns an error envelope (mirrors failure)
>   CHECK: node .skillgrid/acceptance-tests/gates/mnemonic-r6-busy.mjs
>   EXPECT: MNEMONIC_R6_BUSY_OK
>   EVIDENCE: pending

### Requirement: CLI

The system SHALL expose the memory tool as `mnemonic memory <verb>`, printing one envelope JSON per invocation with the 0/1/2 exit-code contract.

#### Scenario: happy path — search prints an envelope and exits zero

```gherkin
      Given an observation in the project store
      When the CLI runs a memory search
      Then stdout is exactly one envelope JSON with the results
      And the exit code is zero
```

#### Scenario: edge — unknown flag is a usage error

```gherkin
      Given any store state
      When the CLI runs a memory verb with an unknown flag
      Then usage is printed to stderr
      And stdout carries no envelope
      And the exit code is the usage code
```

#### Scenario: failure — handler error prints an envelope and exits one

```gherkin
      Given a save that will fail for a missing session
      When the CLI runs the memory save
      Then stdout is one envelope JSON with the stable error code
      And the exit code is one
```

#### Gates
> G13: search prints an envelope and exits zero (mirrors happy path)
>   CHECK: node .skillgrid/acceptance-tests/gates/mnemonic-r7-cli.mjs
>   EXPECT: MNEMONIC_R7_CLI_OK
>   EVIDENCE: pending
> G14: handler error prints an envelope and exits one (mirrors failure)
>   CHECK: node .skillgrid/acceptance-tests/gates/mnemonic-r7-exit1.mjs
>   EXPECT: MNEMONIC_R7_EXIT1_OK
>   EVIDENCE: pending

### Requirement: Single handler parity

The system SHALL run the CLI and MCP as two adapters over one shared handler, so identical inputs produce byte-identical envelopes.

#### Scenario: happy path — byte-identical envelopes across adapters

```gherkin
      Given identical search inputs
      When the same search runs once via the CLI and once via MCP
      Then the two envelopes are byte-identical
```

#### Scenario: edge — concurrent CLI and MCP on one project

```gherkin
      Given a project with a store
      When a CLI save and an MCP search run concurrently
      Then both complete without error
      And the saved observation is searchable
```

#### Scenario: failure — a rendering divergence is caught by the parity suite

```gherkin
      Given one adapter's envelope rendering is changed for a single method
      When the parity suite runs
      Then the suite fails naming the divergent method
```

#### Gates
> G15: byte-identical envelopes across adapters (mirrors happy path)
>   CHECK: node .skillgrid/acceptance-tests/gates/mnemonic-r8-parity.mjs
>   EXPECT: MNEMONIC_R8_PARITY_OK
>   EVIDENCE: pending
> G16: a rendering divergence is caught by the parity suite (mirrors failure)
>   CHECK: node .skillgrid/acceptance-tests/gates/mnemonic-r8-divergence.mjs
>   EXPECT: MNEMONIC_R8_DIVERGENCE_OK
>   EVIDENCE: pending

### Requirement: Offline by default

The system SHALL perform zero network I/O on the default save and search paths; only the one-time verified model download at init and an explicitly configured API tier may touch the network.

#### Scenario: happy path — save and search succeed with the network cut

```gherkin
      Given the default configuration with the network unavailable
      When an observation is saved and searched
      Then both succeed
      And no outbound connection is attempted
```

#### Scenario: edge — first run without a model does not download

```gherkin
      Given a fresh data directory and no model
      When a save and a search run
      Then the hash floor is used with a degraded flag
      And no download is attempted
```

#### Scenario: failure — configured API endpoint down falls to the floor

```gherkin
      Given the API embedder tier configured against an unreachable endpoint
      When an observation is saved
      Then the save succeeds on the hash floor
      And the envelope flags the result as degraded
```

#### Gates
> G17: save and search succeed with the network cut (mirrors happy path)
>   CHECK: node .skillgrid/acceptance-tests/gates/mnemonic-r9-offline.mjs
>   EXPECT: MNEMONIC_R9_OFFLINE_OK
>   EVIDENCE: pending
> G18: configured API endpoint down falls to the floor (mirrors failure)
>   CHECK: node .skillgrid/acceptance-tests/gates/mnemonic-r9-api-down.mjs
>   EXPECT: MNEMONIC_R9_API_DOWN_OK
>   EVIDENCE: pending

> Rules:
> - ≥1 happy + edge + failure scenario per requirement
> - Every briefing requirement → a Rule: (### Requirement:)
> - Scenario names unique and referenceable from blueprint tasks SATISFIES lines
> - Domain language only: use glossary terms, never implementation jargon
> - Then steps state observable outcomes, never internal state
> - Steps: 6-space indent in the fence (verbatim — the extractor does NOT re-indent)
> - Gates: every requirement's happy-path scenario MUST have a G<n> entry (runnable
>   CHECK+EXPECT, or a manual/ABANDON entry with a reason); a happy path with none
>   is a traceability gap
> - Gates: CHECK must be a repo-owned command (Node script preferred); EXPECT is a
>   success-only marker, never a copied number; a negative/absence oracle names a
>   positive control fixture
> - Gates: ABANDON is terminal and non-successful — it surfaces as a handoff, never
>   a pass; EVIDENCE is bound to the gate's current CHECK+EXPECT
> - Gates: a design task may carry a STRUCTURAL gate — an observable invariant about
>   the module's shape, in the same CHECK+EXPECT format (e.g. "the public interface
>   exposes ≤3 entry points", "tests cross the seam and assert on outcomes, not
>   internal state"). These are the deep-module invariants from
>   skillgrid:writing-blueprints → references/codebase-design.md (deletion test,
>   one-adapter rule). Assert the shape, not the internals.
