# Session events layer

Source: .skillgrid/specs/2026-09-21-session-events-layer/
Trace: change briefing requirements; blueprint tasks SATISFIES lines.

Format rules (plain prose — lines starting with `# ` would false-trigger the
extractor's single-H1 rule, so header notes stay untagged):
- Steps in the ```gherkin fence are indented 6 spaces (verbatim — the extractor does NOT re-indent)
- No Gherkin tags in the spec — selection/trace is via the SATISFIES field in blueprint/tasks
- One blank line between the heading and the fence is fine
- Gates: each requirement carries a `#### Gates` block — the runnable shadow of
  its happy-path and failure scenarios. Author it BEFORE implementing.

## Requirements

### Requirement: session-lifecycle

The system SHALL record every agent session with its start commit and end commit, so the session's change range is always known.

#### Scenario: session-lifecycle

```gherkin
      Given a git repository at commit "abc1234"
      When an agent session starts in that repository
      Then the session record carries start commit "abc1234" and status active
      And the session event stream opens with a session start entry at position zero
      When the session ends at commit "def5678"
      Then the session record carries end commit "def5678" and an end time
      And the session event stream closes with a session end entry last
```

#### Scenario: session-lifecycle-outside-repo

```gherkin
      Given a working directory that is not a git repository
      When an agent session starts there
      Then the session starts normally with an empty change range and no error
```

#### Scenario: session-lifecycle-unknown-session

```gherkin
      Given no session with id "no-such-session"
      When the session is ended
      Then the end is rejected with a session-not-found error
```

#### Gates
G1: session-lifecycle happy path
   CHECK: cd skillgrid-cli && go test ./internal/mnemonic/memory/ -run TestSessionStartEndEvents
   EXPECT: --- PASS
   EVIDENCE: pending
G2: session-lifecycle-unknown-session failure path
   CHECK: cd skillgrid-cli && go test ./internal/mnemonic/memory/ -run TestSessionStartEndEvents
   EXPECT: --- PASS
   EVIDENCE: pending

### Requirement: tool-call-stream

The system SHALL append one ordered event per tool call and keep per-session counters, so a session's activity is replayable in order.

#### Scenario: tool-call-stream

```gherkin
      Given an active agent session
      When a file write completes and then a shell command completes
      Then the session event stream holds a file write entry at position 1 and a shell execution entry at position 2
      And the session counters show one file written and one command executed
```

#### Scenario: tool-call-stream-concurrent

```gherkin
      Given an active agent session with parallel subagents reporting tool calls
      When the subagents record their tool calls concurrently
      Then every recorded entry has a unique position and the positions are gap-free in order
```

#### Scenario: tool-call-stream-rejected

```gherkin
      Given an active agent session
      When a tool call event arrives with an unknown hook type
      Then the event is rejected with an error and no entry is appended
```

#### Gates
G1: tool-call-stream happy path
   CHECK: cd skillgrid-cli && go test ./internal/mnemonic/memory/ -run TestPostToolUseAppendsOrderedEvents
   EXPECT: --- PASS
   EVIDENCE: pending
G2: tool-call-stream-rejected failure path
   CHECK: cd skillgrid-cli && go test ./internal/mnemonic/memory/ -run TestPostToolUseAppendsOrderedEvents
   EXPECT: --- PASS
   EVIDENCE: pending

### Requirement: sensitive-redaction

The system SHALL flag sensitive file activity and never store sensitive content, so secrets that cross a tool call are not persisted.

#### Scenario: sensitive-redaction

```gherkin
      Given an active agent session
      When a file write targets "config/.env" with content "SECRET=abc123"
      Then the entry is flagged sensitive and the sensitive-actions counter is 1
      And the stored detail holds a content hash and a masked preview but not "SECRET=abc123"
```

#### Scenario: sensitive-redaction-clean-path

```gherkin
      Given an active agent session
      When a file write targets "src/main.go"
      Then the entry is not flagged sensitive and the sensitive-actions counter stays 0
```

#### Scenario: sensitive-redaction-absence

```gherkin
      Given a flagged sensitive entry with a stored hash and preview
      When the stored detail is inspected
      Then the full secret value appears nowhere in the stored detail
```

#### Gates
G1: sensitive-redaction happy path (positive control: hash present, preview masked)
   CHECK: cd skillgrid-cli && go test ./internal/mnemonic/memory/ -run TestSensitiveWriteRedacted
   EXPECT: --- PASS
   EVIDENCE: pending
G2: sensitive-redaction-absence (absence oracle with positive control above)
   CHECK: cd skillgrid-cli && go test ./internal/mnemonic/memory/ -run TestSensitiveWriteRedacted
   EXPECT: --- PASS
   EVIDENCE: pending

### Requirement: commit-events

The system SHALL record every work-unit commit as a session event carrying its parsed context block, so commits are queryable per session.

#### Scenario: commit-events

```gherkin
      Given an active agent session in a git repository
      When a work-unit commit lands carrying a context block with task, decisions, and remaining work
      Then a commit entry holds the commit id and the parsed task, decisions, and remaining work
```

#### Scenario: commit-events-without-block

```gherkin
      Given an active agent session in a git repository
      When a commit lands with no context block
      Then a commit entry is still recorded with empty parsed detail and no error
```

#### Scenario: commit-events-no-repo

```gherkin
      Given a directory with no commits yet
      When the commit record path runs there
      Then it reports no commit to record with a clean error and no entry
```

#### Gates
G1: commit-events happy path
   CHECK: cd skillgrid-cli && go test ./internal/mnemonic/memory/ -run TestCommitEventParsesContextBlock
   EXPECT: --- PASS
   EVIDENCE: pending
G2: commit-events-no-repo failure path
   CHECK: cd skillgrid-cli && go test ./internal/mnemonic/memory/ -run TestCommitEventParsesContextBlock
   EXPECT: --- PASS
   EVIDENCE: pending

### Requirement: resume-from-events

The system SHALL answer what changed in a session from its event stream plus the net commit range, so a fresh session can resume without the old checkpoint file or hub.

#### Scenario: resume-from-events

```gherkin
      Given a session with a start entry, tool entries, and commit entries
      When the session changes are requested for that session
      Then every entry returns in position order with the net start-to-end change summary
```

#### Scenario: resume-from-events-quiet-session

```gherkin
      Given a session with a start entry and an end entry but no tool entries
      When the session changes are requested for that session
      Then only the start and end entries return with an empty change summary and no error
```

#### Scenario: resume-from-events-unknown-session

```gherkin
      Given no session with id "no-such-session"
      When the session changes are requested for that id
      Then the request fails with a session-not-found error
```

#### Gates
G1: resume-from-events happy path (CLI)
   CHECK: cd skillgrid-cli && go test ./cmd/skillgrid/ -run TestSessionShow
   EXPECT: --- PASS
   EVIDENCE: pending
G2: resume-from-events happy path (MCP tool)
   CHECK: cd skillgrid-cli && go test ./internal/mnemonic/mcp/ -run TestSessionChangesTool
   EXPECT: --- PASS
   EVIDENCE: pending
G3: resume-from-events-unknown-session failure path
   CHECK: cd skillgrid-cli && go test ./cmd/skillgrid/ -run TestSessionShow
   EXPECT: --- PASS
   EVIDENCE: pending

### Requirement: old-surfaces-removed

The system SHALL NOT offer the checkpoint file, the handoff command, the hub/relay memory tools, or the checkpoint script subcommands, so only the events layer remains.

#### Scenario: old-surfaces-removed

```gherkin
      Given the installed skillgrid command set and memory tool list
      When the handoff command is invoked
      Then it reports an unknown subcommand
      And the memory tool list contains no handoff, session handoff, session resume, session status, or knowledge refresh tools
      And the checkpoint script reports an unknown subcommand for snapshot and restore
      And no checkpoint file is written after a work-unit commit
```

#### Scenario: old-surfaces-removed-stale-file

```gherkin
      Given a stale checkpoint file left on disk from before the consolidation
      When a fresh session resumes
      Then the stale file is ignored and the session position comes from the event stream
```

#### Scenario: old-surfaces-removed-tool-call

```gherkin
      Given a client calling a removed handoff memory tool
      When the call is dispatched
      Then it fails with a method-not-found error and no side effect
```

#### Gates
G1: old-surfaces-removed happy path (CLI surface gone — positive control: help lists session show)
   CHECK: cd skillgrid-cli && go test ./cmd/skillgrid/ -run TestHandoffCommandGone
   EXPECT: --- PASS
   EVIDENCE: pending
G2: old-surfaces-removed-tool-call failure path
   CHECK: cd skillgrid-cli && go test ./internal/mnemonic/mcp/ -run TestRemovedToolsAbsent
   EXPECT: --- PASS
   EVIDENCE: pending

### Requirement: per-plan-workspace

The system SHALL keep one scratch workspace per plan with a plan-naming ledger, so a follow-up plan never reads another plan's progress.

#### Scenario: per-plan-workspace

```gherkin
      Given two different plan files
      When each plan's workspace is resolved
      Then the two workspaces are distinct directories
      And each holds its own ledger, task briefs, and review packages
      And each ledger names its own plan file on its first line
```

#### Scenario: per-plan-workspace-foreign-ledger

```gherkin
      Given a resumed controller on plan B with plan A's workspace present
      When the controller reads its workspace ledger
      Then it starts plan B at its first task and leaves plan A's ledger untouched
```

#### Scenario: per-plan-workspace-missing-plan

```gherkin
      Given no plan file at the given path
      When the plan workspace is resolved
      Then the resolution fails with a missing-plan error and no directory is created
```

#### Gates
G1: per-plan-workspace happy path
   CHECK: bash scripts/test-sdd-workspace.sh
   EXPECT: Results: 0 failed
   EVIDENCE: pending
G2: per-plan-workspace-missing-plan failure path
   CHECK: bash scripts/test-sdd-workspace.sh
   EXPECT: Results: 0 failed
   EVIDENCE: pending

Rules (plain prose — see header note on the single-H1 rule):
- ≥1 happy + edge + failure scenario per requirement
- Every briefing requirement → a Rule: (### Requirement:)
- Scenario names unique and referenceable from blueprint tasks SATISFIES lines
- Domain language only: use .skillgrid/glossary terms, never implementation jargon
- Then steps state observable outcomes, never internal state
- Steps: 6-space indent in the fence (verbatim — the extractor does NOT re-indent)
- Gates: every requirement's happy-path scenario MUST have a G<n> entry (runnable
  CHECK+EXPECT, or a manual/ABANDON entry with a reason); a happy path with none
  is a traceability gap
- Gates: CHECK must be a repo-owned command; EXPECT is a success-only marker,
  never a copied number; a negative/absence oracle names a positive control fixture
- Gates: ABANDON is terminal and non-successful — it surfaces as a handoff, never
  a pass; EVIDENCE is bound to the gate's current CHECK+EXPECT
