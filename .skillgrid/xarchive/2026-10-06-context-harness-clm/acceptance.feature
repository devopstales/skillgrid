# Context Harness (with CLM) — acceptance contract
#
# Source: .skillgrid/specs/2026-10-06-context-harness-clm/
# Trace: change briefing ## Requirements; blueprint tasks SATISFIES lines.
#
# Format rules:
# - Steps in the ```gherkin fence are indented 6 spaces (verbatim — the extractor does NOT re-indent)
# - No Gherkin tags in the spec — selection/trace is via the SATISFIES field in blueprint/tasks
# - Gates: each requirement carries a `#### Gates` block — the runnable shadow of
#   its happy-path and failure scenarios. Author it BEFORE implementing.

## Requirements

### Requirement: Capture gate (intercept-and-abstract)

The system SHALL store large tool output (> threshold, default ~4KB) to the session-scoped Sandbox Store and give the agent a summary plus a `ctx_search` pointer, while small output flows through unchanged and `SKILLGRID_CTX_BYPASS=1` forces the full text.

#### Scenario: large-output-captured

```gherkin
      Given a session with a tool that produces output over the capture threshold
      When the tool call is captured in the PostToolUse seam
      Then the full output is stored in the Sandbox Store
      And the agent receives a 200-char summary and a ctx_search pointer
```

#### Scenario: small-output-passes-through

```gherkin
      Given a session with a tool that produces output at or under the capture threshold
      When the tool call is captured
      Then no row is written to the Sandbox Store
      And the agent receives the original text unchanged
```

#### Scenario: bypass-forces-full-text

```gherkin
      Given SKILLGRID_CTX_BYPASS is set to 1
      And a tool that produces output over the capture threshold
      When the tool call is captured
      Then the agent receives the full text
      And no row is written to the Sandbox Store
```

#### Gates
# G1: large output is captured and the agent gets a summary + pointer (mirrors large-output-captured)
#   CHECK: node acceptance-tests/ctx-capture-gate.mjs --case large
#   EXPECT: CAPTURE_GATE_OK
#   EVIDENCE: pending
# G2: small output produces no sandbox row (mirrors small-output-passes-through)
#   CHECK: node acceptance-tests/ctx-capture-gate.mjs --case small
#   EXPECT: CAPTURE_GATE_OK
#   EVIDENCE: pending
# G3: bypass forces the full text (mirrors bypass-forces-full-text)
#   CHECK: node acceptance-tests/ctx-capture-gate.mjs --case bypass
#   EXPECT: CAPTURE_GATE_OK
#   EVIDENCE: pending

### Requirement: Structured query (ctx_query)

The system SHALL answer deterministic counts, lists, and existence checks over the code index with no JS execution and no line ranges in v1.

#### Scenario: ctx-query-known-symbol

```gherkin
      Given a code index containing a known symbol
      When the agent runs ctx_query for that symbol's kind and name
      Then the result returns the correct count and list
      And the query executes without running JavaScript
```

#### Scenario: ctx-query-unknown-symbol

```gherkin
      Given a code index that does not contain the queried symbol
      When the agent runs ctx_query for it
      Then the result is empty or falsy
      And no error is raised
```

#### Gates
# G4: a known symbol resolves deterministically (mirrors ctx-query-known-symbol)
#   CHECK: node acceptance-tests/ctx-query.mjs --case known
#   EXPECT: CTX_QUERY_OK
#   EVIDENCE: pending
# G5: an unknown symbol is empty, not an error (mirrors ctx-query-unknown-symbol)
#   CHECK: node acceptance-tests/ctx-query.mjs --case unknown
#   EXPECT: CTX_QUERY_OK
#   EVIDENCE: pending

### Requirement: Proactive index (ctx index + indexed_files)

The system SHALL index a file or directory into the project-scoped `indexed_files` store reusing the observations schema shape, upsert by source path + chunk, and bound chunks per call.

#### Scenario: ctx-index-file

```gherkin
      Given a repository file with several chunks
      When the agent runs ctx index on that file
      Then rows are written to indexed_files for each chunk
      And re-running ctx index upserts without duplicate rows
```

#### Scenario: ctx-index-bounded

```gherkin
      Given a file that would produce more chunks than the per-call bound
      When the agent runs ctx index on it
      Then the number of rows is bounded to the per-call limit
```

#### Gates
# G6: indexing a file creates rows and upserts (mirrors ctx-index-file)
#   CHECK: node acceptance-tests/ctx-index.mjs --case file
#   EXPECT: CTX_INDEX_OK
#   EVIDENCE: pending
# G7: a large file is bounded (mirrors ctx-index-bounded)
#   CHECK: node acceptance-tests/ctx-index.mjs --case bound
#   EXPECT: CTX_INDEX_OK
#   EVIDENCE: pending

### Requirement: Fused search (ctx_search)

The system SHALL fuse the session-scoped Sandbox Store and the project-scoped index via RRF and return per-leg provenance.

#### Scenario: ctx-search-fused

```gherkin
      Given a query that matches both a captured output and an indexed file
      When the agent runs ctx search
      Then both results are returned fused by reciprocal rank fusion
      And each result carries its source leg
```

#### Scenario: ctx-search-single-leg

```gherkin
      Given a query that matches only the Sandbox Store
      When the agent runs ctx search
      Then only the Sandbox Store leg is returned
```

#### Gates
# G8: a both-leg query returns fused, provenanced results (mirrors ctx-search-fused)
#   CHECK: node acceptance-tests/ctx-search.mjs --case both
#   EXPECT: CTX_SEARCH_OK
#   EVIDENCE: pending
# G9: a single-leg query returns that leg (mirrors ctx-search-single-leg)
#   CHECK: node acceptance-tests/ctx-search.mjs --case single
#   EXPECT: CTX_SEARCH_OK
#   EVIDENCE: pending

### Requirement: ctx CLI

The system SHALL expose `ctx stats`, `ctx index <path>`, `ctx search <query>`, and `ctx purge` from the command line, with `ctx purge` clearing the session-scoped sandbox and revisions.

#### Scenario: ctx-purge-clears-sandbox

```gherkin
      Given a session with rows in the Sandbox Store and context revisions
      When the operator runs ctx purge
      Then the Sandbox Store is empty for the session
      And the context revisions are empty for the session
```

#### Scenario: ctx-stats-reports-counts

```gherkin
      Given a session with captured output and indexed files
      When the operator runs ctx stats
      Then the output reports row counts for the Sandbox Store and indexed_files
```

#### Gates
# G10: ctx purge empties sandbox + revisions (mirrors ctx-purge-clears-sandbox)
#   CHECK: node acceptance-tests/ctx-cli.mjs --case purge
#   EXPECT: CTX_CLI_OK
#   EVIDENCE: pending
# G11: ctx stats reports counts (mirrors ctx-stats-reports-counts)
#   CHECK: node acceptance-tests/ctx-cli.mjs --case stats
#   EXPECT: CTX_CLI_OK
#   EVIDENCE: pending

### Requirement: Context Routing block

The system SHALL inject a Context Routing block from `skillgrid prime` that maps intent to the right tool, advisory and non-blocking.

#### Scenario: prime-routing-block

```gherkin
      Given a session start that runs skillgrid prime
      When the prime output is rendered
      Then it contains the Context Routing block
      And the block maps counts/lists to ctx_query, retrieve to ctx_search, index to ctx index, and memory to mem_*
      And the session is not blocked
```

#### Gates
# G12: prime output carries the routing mappings and does not block (mirrors prime-routing-block)
#   CHECK: node acceptance-tests/ctx-routing.mjs
#   EXPECT: CTX_ROUTING_OK
#   EVIDENCE: pending

### Requirement: CLM opt-in (default off)

The system SHALL keep the Context Language Model off by default; when off, no mirror is rendered, no overflow guard runs, and no context revisions are written.

#### Scenario: clm-off-by-default

```gherkin
      Given a session with default configuration
      When a model request is made
      Then no mirror file is rendered
      And no context revision is written
      And no budget note appears
```

#### Scenario: clm-on-renders-mirror

```gherkin
      Given clm.enabled is true in configuration
      When a model request is made
      Then a mirror file is rendered before the request
```

#### Gates
# G13: default config yields no mirror and no revision (mirrors clm-off-by-default)
#   CHECK: node acceptance-tests/clm-optin.mjs --case off
#   EXPECT: CLM_OPTIN_OK
#   EVIDENCE: pending
# G14: clm on renders the mirror (mirrors clm-on-renders-mirror)
#   CHECK: node acceptance-tests/clm-optin.mjs --case on
#   EXPECT: CLM_OPTIN_OK
#   EVIDENCE: pending

### Requirement: CLM mirror render + model edit

The system SHALL render the effective context to a 0600 mirror file before each request when CLM is on, using nonce-bound ids, and allow the agent to edit it with ordinary file tools.

#### Scenario: clm-mirror-rendered

```gherkin
      Given clm.enabled is true
      When the context hook renders the effective context
      Then a mirror file exists with mode 0600
      And the file contains a LIVE_CONTEXT header and CTX_TURN blocks
```

#### Scenario: clm-model-edits-mirror

```gherkin
      Given a rendered mirror file
      When the agent edits the file with a file tool
      Then the edit is observable at turn-end
```

#### Gates
# G15: the mirror is rendered 0600 with the expected format (mirrors clm-mirror-rendered)
#   CHECK: node acceptance-tests/clm-mirror.mjs --case render
#   EXPECT: CLM_MIRROR_OK
#   EVIDENCE: pending
# G16: a model edit is observable at turn-end (mirrors clm-model-edits-mirror)
#   CHECK: node acceptance-tests/clm-mirror.mjs --case edit
#   EXPECT: CLM_MIRROR_OK
#   EVIDENCE: pending

### Requirement: Revision capture + validation

The system SHALL read the model's mirror edit at turn-end, validate it, persist a context revision, and activate it for the next request; an unchanged mirror writes no new revision and a nonce mismatch discards the edit.

#### Scenario: clm-revision-captured

```gherkin
      Given a changed mirror at turn-end
      When the checkpoint capture path reads and posts the edit
      Then a context revision row is persisted whose anchor matches the raw prefix
      And the revision is active on the next request
```

#### Scenario: clm-unchanged-mirror-skips

```gherkin
      Given an unchanged mirror at turn-end
      When the checkpoint capture path reads it
      Then no new context revision row is written
```

#### Scenario: clm-nonce-mismatch-discards

```gherkin
      Given a mirror edit whose nonce does not match
      When the capture path posts it
      Then the edit is discarded and no revision is activated
```

#### Gates
# G17: a changed mirror persists + activates a revision (mirrors clm-revision-captured)
#   CHECK: node acceptance-tests/clm-revision.mjs --case changed
#   EXPECT: CLM_REVISION_OK
#   EVIDENCE: pending
# G18: an unchanged mirror writes no row (mirrors clm-unchanged-mirror-skips)
#   CHECK: node acceptance-tests/clm-revision.mjs --case unchanged
#   EXPECT: CLM_REVISION_OK
#   EVIDENCE: pending
# G19: a nonce mismatch discards the edit (mirrors clm-nonce-mismatch-discards)
#   CHECK: node acceptance-tests/clm-revision.mjs --case nonce
#   EXPECT: CLM_REVISION_OK
#   EVIDENCE: pending

### Requirement: Overflow guard

The system SHALL withhold the oldest tool results when the calibrated estimate of the request exceeds `budget − reserve`, and withhold nothing when under budget.

#### Scenario: clm-overflow-withholds

```gherkin
      Given a request whose estimated size exceeds budget minus reserve
      When the overflow guard runs
      Then the oldest tool results are swapped for one-line notes
      And the withhold decision is stored on the revision
```

#### Scenario: clm-overflow-under-budget

```gherkin
      Given a request whose estimated size is under budget minus reserve
      When the overflow guard runs
      Then no tool result is withheld
```

#### Gates
# G20: over-budget withholds the oldest tool results (mirrors clm-overflow-withholds)
#   CHECK: node acceptance-tests/clm-overflow.mjs --case over
#   EXPECT: CLM_OVERFLOW_OK
#   EVIDENCE: pending
# G21: under-budget withholds nothing (mirrors clm-overflow-under-budget)
#   CHECK: node acceptance-tests/clm-overflow.mjs --case under
#   EXPECT: CLM_OVERFLOW_OK
#   EVIDENCE: pending

### Requirement: Calibration

The system SHALL correct the size-estimate factor against the provider's token count for each request and store the corrected factor per session.

#### Scenario: clm-calibration-corrects

```gherkin
      Given a request whose provider token count differs from the estimate
      When the calibration runs
      Then the stored calibration factor reflects the correction
      And the next estimate uses the corrected factor
```

#### Gates
# G22: the provider count corrects the stored factor (mirrors clm-calibration-corrects)
#   CHECK: node acceptance-tests/clm-calibrate.mjs
#   EXPECT: CLM_CALIBRATE_OK
#   EVIDENCE: pending

### Requirement: Session-scoped purge + resume

The system SHALL purge `tool_outputs` and `context_revisions` at session end, and on resume reconstruct the active revision and validate its anchor, falling back to raw context on a mismatch.

#### Scenario: clm-resume-validates-anchor

```gherkin
      Given a resumed session whose raw prefix still matches the stored anchor
      When the plugin reconstructs the active revision
      Then the revision is restored and used
```

#### Scenario: clm-resume-broken-anchor-falls-back

```gherkin
      Given a resumed session whose raw history was compacted so the anchor no longer matches
      When the plugin reconstructs the active revision
      Then the revision is discarded
      And the session falls back to raw context
```

#### Gates
# G23: a valid anchor restores the revision (mirrors clm-resume-validates-anchor)
#   CHECK: node acceptance-tests/clm-resume.mjs --case valid
#   EXPECT: CLM_RESUME_OK
#   EVIDENCE: pending
# G24: a broken anchor falls back to raw context (mirrors clm-resume-broken-anchor-falls-back)
#   CHECK: node acceptance-tests/clm-resume.mjs --case broken
#   EXPECT: CLM_RESUME_OK
#   EVIDENCE: pending

### Requirement: CLM config

The system SHALL read all CLM settings from the `clm:` block with environment overrides, and yield all defaults (off) when the block is absent.

#### Scenario: clm-config-honored

```gherkin
      Given a clm block in configuration
      When the CLM reads its settings
      Then the enabled, budget, reserve, reminders, guard, and cap values are honored
```

#### Scenario: clm-env-override-wins

```gherkin
      Given a clm budget in configuration and a different SKILLGRID_CTX_CLM_BUDGET env value
      When the CLM reads its settings
      Then the env value wins over the config value
```

#### Scenario: clm-absent-block-defaults

```gherkin
      Given no clm block in configuration
      When the CLM reads its settings
      Then all settings are at their defaults and CLM is off
```

#### Gates
# G25: the clm block is honored (mirrors clm-config-honored)
#   CHECK: node acceptance-tests/clm-config.mjs --case block
#   EXPECT: CLM_CONFIG_OK
#   EVIDENCE: pending
# G26: an env override wins over config (mirrors clm-env-override-wins)
#   CHECK: node acceptance-tests/clm-config.mjs --case env
#   EXPECT: CLM_CONFIG_OK
#   EVIDENCE: pending
# G27: an absent block yields defaults/off (mirrors clm-absent-block-defaults)
#   CHECK: node acceptance-tests/clm-config.mjs --case absent
#   EXPECT: CLM_CONFIG_OK
#   EVIDENCE: pending

# Rules:
# - ≥1 happy + edge + failure scenario per requirement
# - Every briefing requirement → a Rule: (### Requirement:)
# - Scenario names unique and referenceable from blueprint tasks SATISFIES lines
# - Domain language only: use .skillgrid/artifacts/02-technical-terms.md terms, never implementation jargon
# - Then steps state observable outcomes, never internal state
# - Steps: 6-space indent in the fence (verbatim — the extractor does NOT re-indent)
# - Gates: every requirement's happy-path scenario MUST have a G<n> entry (runnable
#   CHECK+EXPECT, or a manual/ABANDON entry with a reason); a happy path with none
#   is a traceability gap
# - Gates: CHECK must be a repo-owned command (Node script preferred); EXPECT is a
#   success-only marker, never a copied number; a negative/absence oracle names a
#   positive control fixture
# - Gates: ABANDON is terminal and non-successful — it surfaces as a handoff, never
#   a pass; EVIDENCE is bound to the gate's current CHECK+EXPECT
