# Compact search output
#
# Source: .skillgrid/specs/2026-09-17-compact-search-output/
# Trace: change briefing ## Definition of Done; blueprint tasks SATISFIES lines.
#
# Format rules:
# - Steps in the ```gherkin fence are indented 6 spaces (verbatim — the extractor does NOT re-indent)
# - No Gherkin tags in the spec — selection/trace is via the SATISFIES field in blueprint/tasks
# - One blank line between the heading and the fence is fine
# - Gates: each requirement carries a `#### Gates` block — the runnable shadow of
#   its happy-path and failure scenarios. Author it BEFORE implementing.

## Requirements

### Requirement: compact-format-symbol

The system SHALL render a symbol search hit as a single compact line carrying the symbol name, the file, the line range, and the signature, with no source body.

#### Scenario: compact-format-symbol

```gherkin
      Given a hybrid search result containing symbol hits
      When the compact formatter is applied to the result
      Then each symbol hit is a single line of the form name (file:line_start-line_end) — signature
```

#### Scenario: compact-no-body

```gherkin
      Given a hybrid search result containing a symbol hit whose snippet is a multi-line source body
      When the compact formatter is applied to the result
      Then the compact line contains only the name, file, line range, and signature and no body line
```

#### Gates
# G1: compact-format-symbol happy path
#   CHECK: cd skillgrid-cli && go test ./internal/mnemonic/hybrid/ -run TestCompactFormat
#   EXPECT: --- PASS
#   EVIDENCE: pending
# G2: compact-no-body happy path
#   CHECK: cd skillgrid-cli && go test ./internal/mnemonic/hybrid/ -run TestCompactFormat
#   EXPECT: --- PASS
#   EVIDENCE: pending

### Requirement: compact-format-chunk

The system SHALL render a chunk search hit as a single compact line carrying the file, the line range, and a preview of at most 80 characters of the first line, with no full source body.

#### Scenario: compact-format-chunk

```gherkin
      Given a hybrid search result containing chunk hits
      When the compact formatter is applied to the result
      Then each chunk hit is a single line of the form file:line_start-line_end — preview and the preview is at most 80 characters
```

#### Scenario: compact-token-budget

```gherkin
      Given a hybrid search result of ten mixed hits
      When the compact formatter is applied to the result
      Then the compact output estimates to at most 250 tokens
```

#### Gates
# G1: compact-format-chunk happy path
#   CHECK: cd skillgrid-cli && go test ./internal/mnemonic/hybrid/ -run TestCompactFormat
#   EXPECT: --- PASS
#   EVIDENCE: pending
# G2: compact-token-budget happy path
#   CHECK: cd skillgrid-cli && go test ./internal/mnemonic/hybrid/ -run TestCompactFormat
#   EXPECT: --- PASS
#   EVIDENCE: pending

### Requirement: compact-format-deterministic

The system SHALL produce byte-identical compact output for identical input so the language-model prompt cache is reusable across repeated queries.

#### Scenario: compact-deterministic

```gherkin
      Given a hybrid search result
      When the compact formatter is applied twice to the identical result
      Then the two compact outputs are byte-identical
```

#### Gates
# G1: compact-deterministic happy path
#   CHECK: cd skillgrid-cli && go test ./internal/mnemonic/hybrid/ -run TestCompactFormat
#   EXPECT: --- PASS
#   EVIDENCE: pending

### Requirement: compact-search-tools

The system SHALL let an agent request a compact answer from the hybrid and full-text search tools via an opt-in context flag, leaving the default full answer unchanged, and SHALL expand named files back to full source within the compact answer.

#### Scenario: compact-search-hybrid

```gherkin
      Given a code index with searchable files
      When the hybrid search tool is called with the context flag set to true
      Then the response is marked compact and returns a single string of compact lines
```

#### Scenario: full-search-unchanged

```gherkin
      Given a code index with searchable files
      When the hybrid search tool is called without the context flag
      Then the response is the unchanged full format with hits as a list
```

#### Scenario: compact-search-fts

```gherkin
      Given a code index with searchable files
      When the full-text search tool is called with the context flag set to true
      Then the response is marked compact and returns a single string of compact lines
```

#### Scenario: compact-token-budget-mcp

```gherkin
      Given a code index with a small result set
      When the hybrid search tool is called with the context flag set to true
      Then the compact string estimates to at most 250 tokens
```

#### Scenario: compact-unfold-single

```gherkin
      Given a code index where a search hits several files
      When the hybrid search tool is called with the context flag and one file name in the unfold list
      Then the response contains the full source for that file and compact lines for every other hit
```

#### Scenario: compact-unfold-glob

```gherkin
      Given a code index where a search hits several files including a test file
      When the hybrid search tool is called with the context flag and a file-name glob in the unfold list
      Then the response contains the full source for the matched test file and compact lines for every other hit
```

#### Gates
# G1: compact-search-hybrid happy path
#   CHECK: cd skillgrid-cli && go test ./internal/mnemonic/mcp/ -run TestCompactSearch
#   EXPECT: --- PASS
#   EVIDENCE: pending
# G2: full-search-unchanged (regression)
#   CHECK: cd skillgrid-cli && go test ./internal/mnemonic/mcp/ -run TestCompactSearch
#   EXPECT: --- PASS
#   EVIDENCE: pending
# G3: compact-search-fts + compact-token-budget-mcp happy path
#   CHECK: cd skillgrid-cli && go test ./internal/mnemonic/mcp/ -run TestCompactSearch
#   EXPECT: --- PASS
#   EVIDENCE: pending
# G4: compact-unfold-single + compact-unfold-glob happy path
#   CHECK: cd skillgrid-cli && go test ./internal/mnemonic/mcp/ -run TestCompactSearchUnfold
#   EXPECT: --- PASS
#   EVIDENCE: pending

# Rules:
# - ≥1 happy + edge + failure scenario per requirement
# - Every briefing requirement → a Rule: (### Requirement:)
# - Scenario names unique and referenceable from blueprint tasks SATISFIES lines
# - Domain language only: use .skillgrid/glossary terms, never implementation jargon
# - Then steps state observable outcomes, never internal state
# - Steps: 6-space indent in the fence (verbatim — the extractor does NOT re-indent)
# - Gates: every requirement's happy-path scenario MUST have a G<n> entry (runnable
#   CHECK+EXPECT, or a manual/ABANDON entry with a reason); a happy path with none
#   is a traceability gap
# - Gates: CHECK must be a repo-owned command; EXPECT is a success-only marker,
#   never a copied number; a negative/absence oracle names a positive control fixture
# - Gates: ABANDON is terminal and non-successful — it surfaces as a handoff, never
#   a pass; EVIDENCE is bound to the gate's current CHECK+EXPECT
