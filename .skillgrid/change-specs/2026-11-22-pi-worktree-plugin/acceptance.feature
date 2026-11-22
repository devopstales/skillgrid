# @devopstales/pi-worktree — Acceptance Spec

Source: .skillgrid/change-specs/2026-11-22-pi-worktree-plugin/
Trace: change briefing ## Definition of Done; blueprint tasks SATISFIES lines.

## Requirements

### Requirement: capture-after-mutating-turn

The system SHALL produce exactly one new checkpoint ref under `refs/skillgrid/` for a turn that mutated the worktree, with the full metadata block and the real git index byte-identical before and after capture.

#### Scenario: checkpoint-after-mutating-turn

```gherkin
      Given a git repository with one committed file and a live Pi session
      When the agent turn edits a tracked file and the turn ends
      Then exactly one new ref exists under refs/skillgrid/ for the session
      And the ref metadata names the branch, head, index-tree, worktree-tree, session, turn, and label
      And the real git index is byte-identical to its pre-capture state
```

#### Scenario: readonly-turn-no-ref

```gherkin
      Given a git repository with one committed file and a live Pi session
      When the agent turn only reads files and the turn ends
      Then no new ref exists under refs/skillgrid/ for the session
```

#### Scenario: unborn-head-capture

```gherkin
      Given a freshly-initialized git repository with an untracked file and no commits
      When the agent turn ends
      Then one checkpoint ref is created with a zero-SHA head in its metadata
```

#### Gates
Runnable shadow of this requirement's happy-path and failure scenarios.
Author BEFORE implementing. One entry per happy-path and failure scenario
(name the scenario it mirrors); edge scenarios may be left to the tests.
G1: checkpoint-after-mutating-turn — one ref, full metadata, index untouched
  CHECK: bun test tests/scratch.test.ts -t "mutating turn"
  EXPECT: PASS
  EVIDENCE: pending
G2: unborn-head-capture — zero-SHA head in metadata
  CHECK: bun test tests/scratch.test.ts -t "unborn HEAD"
  EXPECT: PASS
  EVIDENCE: pending

### Requirement: dedup-on-unchanged-tree

The system SHALL not create a new checkpoint ref when the worktree tree is unchanged from the previous capture for the same session.

#### Scenario: dedup-on-unchanged-turn

```gherkin
      Given a session that already has one checkpoint for its current tree
      When a subsequent turn makes no worktree changes and ends
      Then createCheckpoint returns skipped with reason unchanged
      And the ref count for the session is unchanged
```

#### Scenario: ignored-only-change-no-ref

```gherkin
      Given a session that already has one checkpoint for its current tree
      When a turn only touches files inside ignored directories and ends
      Then the worktree tree SHA is unchanged and no new ref is created
```

#### Scenario: resume-unchanged-turn-no-duplicate

```gherkin
      Given a session that was interrupted after a checkpoint and is resumed
      When the first turn after resume makes no worktree changes and ends
      Then no duplicate ref is created for the unchanged tree
```

#### Gates
G1: readonly-turn-no-ref — no ref for an unchanged turn
  CHECK: bun test tests/scratch.test.ts -t "read-only turn"
  EXPECT: PASS
  EVIDENCE: pending
G2: resume-unchanged-turn-no-duplicate — dedup seeded on resume
  CHECK: bun test tests/scratch.test.ts -t "resume"
  EXPECT: PASS
  EVIDENCE: pending

### Requirement: safe-restore

The system SHALL restore a dirty tree on a matching branch — deleted tracked files return, post-checkpoint untracked files are removed, pre-existing untracked files survive, ignored/large skipped items are never deleted — and always take a before-restore checkpoint first.

#### Scenario: restore-on-matching-branch

```gherkin
      Given a checkpoint on branch main with a pre-existing untracked file
      When the branch advances, a file is deleted, and a new untracked file appears
      And restore is invoked with the checkpoint id on branch main
      Then the deleted tracked file returns with its original content
      And the post-checkpoint untracked file is removed
      And the pre-existing untracked file survives
      And a before-restore checkpoint ref exists
```

#### Scenario: restore-refused-cross-branch

```gherkin
      Given a checkpoint on branch main
      When the current branch is agent-other and restore is invoked
      Then restore is refused with reason branch
      And the refusal names both branch main and agent-other
```

#### Scenario: restore-refused-missing-checkpoint

```gherkin
      Given a repository with no checkpoint for the given id
      When restore is invoked with that id
      Then restore is refused with reason missing
```

#### Gates
G1: restore-on-matching-branch — files restored correctly
  CHECK: bun test tests/scratch.test.ts -t "restore on matching branch"
  EXPECT: PASS
  EVIDENCE: pending
G2: restore-refused-cross-branch — named refusal
  CHECK: bun test tests/scratch.test.ts -t "refused: branch mismatch"
  EXPECT: PASS
  EVIDENCE: pending

### Requirement: bounded-refs

The system SHALL bound refs: the 51st capture in a session prunes the oldest (50 max), and at session start, other sessions' refs older than the liveness threshold are pruned in the background while fresh sibling refs survive.

#### Scenario: fifty-cap-prunes-oldest

```gherkin
      Given a repository with 51 checkpoint refs for one session
      When the 51st capture completes
      Then the oldest ref is pruned and 50 refs remain
```

#### Scenario: cross-session-prune-keeps-fresh-sibling

```gherkin
      Given a repository with a fresh ref from a sibling session and an old ref from a dead session
      When the current session starts and the background prune runs with a clock past the liveness threshold
      Then the dead session's ref is pruned
      And the fresh sibling session's ref survives
```

#### Scenario: exactly-50-no-prune

```gherkin
      Given a repository with exactly 50 checkpoint refs for one session
      When the 50th capture completes
      Then no ref is pruned and 50 refs remain
```

#### Gates
G1: session-start-prune — 50-cap enforced
  CHECK: bun test tests/scratch.test.ts -t "51st capture"
  EXPECT: PASS
  EVIDENCE: pending
G2: cross-session-prune-keeps-fresh-sibling — liveness threshold
  CHECK: bun test tests/scratch.test.ts -t "liveness threshold"
  EXPECT: PASS
  EVIDENCE: pending

### Requirement: directive-follow

The system SHALL move the Pi session to a linked session when a `wt` invocation's directive names a directory inside the original repository; the source session stays resumable; in print/json mode the result is `stopped` + restart directory, no move.

#### Scenario: session-moves-to-worktree

```gherkin
      Given a live Pi TUI session in the primary checkout and wt on PATH
      When the worktrunk tool runs wt switch --create agent-e2e
      And the directive names the new worktree's path
      Then the Pi session moves to a linked session in the new worktree
      And the source session remains resumable
```

#### Scenario: print-mode-stopped

```gherkin
      Given a Pi print-mode session in the primary checkout and wt on PATH
      When the worktrunk tool runs wt switch --create agent-e2e
      And the directive names the new worktree's path
      Then the result is stopped with the restart directory
      And no session move occurs
```

#### Scenario: directive-outside-repo-rejected

```gherkin
      Given a live Pi TUI session and a wt invocation whose directive names a path outside the original repository
      When the directive is read
      Then the session stays put
      And the report names the rejected path
```

#### Gates
G1: session-follows-worktree — linked-session move
  CHECK: bash tests/e2e.sh (the session-move assertion)
  EXPECT: PASS
  EVIDENCE: pending
G2: directive-outside-repo-rejected — stay put
  CHECK: bun test tests/wt-core.test.ts -t "directive naming a path outside"
  EXPECT: PASS
  EVIDENCE: pending

### Requirement: no-replay-recovery

The system SHALL recover to a surviving worktree (repository identity re-verified, the repo's default branch preferred) when the session's cwd is gone, and NOT run a command requested after the directory died.

#### Scenario: recovery-no-replay

```gherkin
      Given a live Pi session in a worktree that is recorded in the worktree list
      When the worktree's directory is removed by another process
      And the next wt call carries a pending command requested after the removal
      Then the session recovers to a surviving worktree with re-verified repository identity
      And the pending command is not run
```

#### Scenario: no-surviving-worktree

```gherkin
      Given a live Pi session in the only worktree
      When the worktree's directory is removed and no other worktree survives
      And the next wt call is made
      Then no move occurs
      And the report names the original path as the restart location
```

#### Gates
G1: recovery-no-replay — move to survivor, no replay
  CHECK: bun test tests/wt-core.test.ts -t "recover to surviving worktree"
  EXPECT: PASS
  EVIDENCE: pending
G2: no-surviving-worktree — report restart location
  CHECK: bun test tests/wt-core.test.ts -t "no surviving worktree"
  EXPECT: PASS
  EVIDENCE: pending

### Requirement: standard-root-observed

The system SHALL observe the actual worktree root on first `wt` use and, when worktrees land outside `<projectroot>/.worktree/`, carry a one-line "observed root differs" note in the report — never silently following a foreign layout.

#### Scenario: standard-root-observed

```gherkin
      Given a repository with wt configured so worktrees land under .worktree/
      When the first wt use observes the worktree root
      Then no note is in the report
```

#### Scenario: foreign-root-noted

```gherkin
      Given a repository with wt configured so worktrees land in a foreign directory
      When the first wt use observes the worktree root
      Then the report carries a one-line observed-root-differs note naming the foreign path
```

#### Gates
G1: standard-root-observed — no note for the standard root
  CHECK: bun test tests/wt-core.test.ts -t "observeWorktreeRoot"
  EXPECT: PASS
  EVIDENCE: pending
G2: foreign-root-noted — one-line note
  CHECK: bun test tests/wt-core.test.ts -t "observeWorktreeRoot"
  EXPECT: PASS
  EVIDENCE: pending

### Requirement: branch-allow-list

The system SHALL reject `switch --create` with a branch outside `^(agent-|worktree-agent-|worktree-wf_)[A-Za-z0-9._/-]+$` in a worktree context, with the nearest allowed form suggested; the primary checkout is unconstrained.

#### Scenario: allow-list-at-creation

```gherkin
      Given a live Pi session in a worktree context
      When the worktrunk tool runs wt switch --create agent-x
      Then the branch is allowed
```

#### Scenario: allow-list-rejected-with-suggestion

```gherkin
      Given a live Pi session in a worktree context
      When the worktrunk tool runs wt switch --create feature-x
      Then the branch is rejected
      And the rejection names the nearest allowed form (agent-feature-x)
```

#### Scenario: primary-checkout-unconstrained

```gherkin
      Given a live Pi session in the primary checkout
      When the worktrunk tool runs wt switch --create feature-x
      Then the branch is allowed
```

#### Gates
G1: allow-list-at-creation — allowed in worktree context
  CHECK: bun test tests/wt-core.test.ts -t "checkBranchAllowList"
  EXPECT: PASS
  EVIDENCE: pending
G2: allow-list-rejected-with-suggestion — named rejection
  CHECK: bun test tests/wt-core.test.ts -t "checkBranchAllowList"
  EXPECT: PASS
  EVIDENCE: pending

### Requirement: merge-defaults

The system SHALL execute `merge` calls with `--no-squash --no-commit` prepended unless the caller explicitly passed either flag; the executed argv is observable in the result.

#### Scenario: merge-gets-default-flags

```gherkin
      Given a live Pi session and a wt merge command
      When the worktrunk tool runs wt merge main
      Then the executed argv includes --no-squash and --no-commit
```

#### Scenario: explicit-squash-not-rewritten

```gherkin
      Given a live Pi session and a wt merge command with an explicit --squash
      When the worktrunk tool runs wt merge --squash main
      Then the executed argv includes --no-commit but not --no-squash
```

#### Gates
G1: merge-defaults — both flags prepended
  CHECK: bun test tests/wt-core.test.ts -t "applyMergeDefaults"
  EXPECT: PASS
  EVIDENCE: pending
G2: explicit-squash-not-rewritten — no duplication
  CHECK: bun test tests/wt-core.test.ts -t "applyMergeDefaults"
  EXPECT: PASS
  EVIDENCE: pending

### Requirement: approval-passthrough

The system SHALL show the approval dialog when a `wt` call needs project-command approval; approving saves exactly those commands and retries once; no code path injects `-y`/`--yes`.

#### Scenario: approval-approved-and-retried

```gherkin
      Given a wt call that triggers an unapproved project-command hook
      When the approval dialog is shown and the user approves the listed commands
      Then exactly those commands are saved to the approvals file
      And the wt call is retried once and succeeds
```

#### Scenario: approval-declined

```gherkin
      Given a wt call that triggers an unapproved project-command hook
      When the approval dialog is shown and the user declines
      Then the command fails
      And the approvals file is unchanged
```

#### Gates
G1: approval-passthrough — save + retry
  CHECK: bun test tests/wt-core.test.ts -t "approval"
  EXPECT: PASS
  EVIDENCE: pending
G2: approval-declined — fail, approvals unchanged
  CHECK: bun test tests/wt-core.test.ts -t "approval"
  EXPECT: PASS
  EVIDENCE: pending

### Requirement: marker-lifecycle

The system SHALL follow the marker lifecycle (`session_start`→💬, `agent_start`→🤖, `agent_end`→💬, `session_shutdown`→clear) visible in `wt list`; a stale marker is reported and manually clearable.

#### Scenario: marker-follows-lifecycle

```gherkin
      Given a live Pi session in a worktree with wt configured
      When the session starts, the agent works, the agent ends, and the session shuts down
      Then wt list shows waiting, working, waiting, and no marker in that order
```

#### Scenario: stale-marker-reported

```gherkin
      Given a worktree whose session was killed before shutdown left a marker
      When the next session in that worktree starts
      Then the stale marker is reported
      And running wt config state marker clear removes it
```

#### Gates
G1: marker-lifecycle — the four-state sequence
  CHECK: bun test tests/markers.test.ts -t "lifecycle"
  EXPECT: PASS
  EVIDENCE: pending
G2: stale-marker-reported — report + clear
  CHECK: bun test tests/markers.test.ts -t "stale"
  EXPECT: PASS
  EVIDENCE: pending

### Requirement: core-tests-without-pi

The system SHALL pass `npm test` in a bare Node/Bun environment with only git + `wt` on PATH (no Pi install); an import-scan test fails the build if any `src/core/` file imports a Pi package.

#### Scenario: npm-test-passes-without-pi

```gherkin
      Given a bare Node/Bun environment with git and wt on PATH and no Pi install
      When npm test is run in the plugin repository
      Then all core tests pass
```

#### Scenario: import-scan-fails-on-pi-import

```gherkin
      Given a src/core/ file that imports @earendil-works/pi-coding-agent
      When npm test is run
      Then the import-scan test fails the build
```

#### Gates
G1: core-tests-without-pi — green without Pi
  CHECK: npm test
  EXPECT: PASS
  EVIDENCE: pending
G2: import-scan-fails-on-pi-import — core purity
  CHECK: bun test tests/import-scan.test.ts
  EXPECT: PASS
  EVIDENCE: pending

### Requirement: contract-drift-guard

The system SHALL fail the contract-drift guard test when any plugin constant diverges from the value in the `~~~skillgrid-constants` fence of the skillgrid-skills contract, naming the mismatch.

#### Scenario: drift-guard-passes-on-match

```gherkin
      Given the plugin constants match the contract fence
      When the drift guard test runs with SKILLGRID_CONTRACT_PATH set
      Then the test passes
```

#### Scenario: drift-named

```gherkin
      Given the plugin's MAX_PER_SESSION is 99 and the contract fence says 50
      When the drift guard test runs with SKILLGRID_CONTRACT_PATH set
      Then the test fails naming MAX_PER_SESSION and both values
```

#### Scenario: missing-fence

```gherkin
      Given a contract file without the ~~~skillgrid-constants fence
      When the drift guard test runs with SKILLGRID_CONTRACT_PATH set to that file
      Then the test fails with the fence-not-found message
```

#### Gates
G1: contract-drift-guard — pass on match
  CHECK: SKILLGRID_CONTRACT_PATH=<path> bun test tests/contract-drift.test.ts
  EXPECT: PASS
  EVIDENCE: pending
G2: drift-named — failure names the mismatch
  CHECK: SKILLGRID_CONTRACT_PATH=<path> bun test tests/contract-drift.test.ts (with a mutated constant)
  EXPECT: FAIL naming the field
  EVIDENCE: pending
