# Evidence Claims (shared across all Skillgrid skills)

Rules for when a claim counts as evidence. Consumed by `debug`, `test`,
`drift`, and `ship`. A consumer cites this file; it never restates these rules.

## Claims vs. evidence

- **Self-reported "fixed" is a claim.** Reproduce the original report's
  exact scenario on a fresh build before closing anything. A fix verified
  only by a test you wrote for this session is not a closed report until the
  original trigger runs and passes.
- **Close-outs land against NAMED TESTS**, never promises or "should be
  fixed now." Each closure names the proving test: commit SHA + test file +
  test name.
- **A test written against the reported mechanism that PASSES on unmodified
  main: the report is right, the diagnosis is not.** The issue's stated
  mechanism is a hypothesis; only its symptom is evidence. Strengthen the
  test until it drives the real surface, then re-diagnose.
- **Green tests + broken behavior = suspect the test was taught to agree.**
  Check whether the commit that introduced the defect also updated a
  fixture, a test helper, or an assertion that pins the defect as intended
  behavior. Delete the pinning row; do not leave a test that asserts the
  bug.

## Baseline discipline

- A pre-existing test failure is a **baseline failure** only after the same
  command and environment reproduce it on a comparable isolated clean base.
  Otherwise report `baseline unverified`.
- Never use `git stash`/`git stash pop` for the baseline comparison — it
  disturbs the working tree. Use a worktree of the base commit (per the
  shared tooling tiers; the baseline worktree gets the same setup step —
  `execute` Step 0d — before the comparison command runs). Recovery path for
  in-flight work: a scratch checkpoint of the working state
  (`refs/skillgrid/`, per `_shared/work-unit-commits/README.md` § Scratch
  checkpoints) taken before leaving the worktree is the restore target when
  you come back.
- An attributed-to-base failure that was never proven on a clean base is a
  claim, not evidence.

## Fix-shape ranking (deletion-first)

Rank candidate fixes by what they DELETE:

1. Removes a mechanism so the failure class becomes impossible.
2. A static guard or ratchet that makes reintroduction a test failure.
3. A localized predicate fix behind a failing repro test.

New surface — verbs, flags, states, config keys, parallel representations of
existing truth — is last resort, deferred until re-verified evidence
demands it.

**The over-engineering test, before ANY fix:** does it add a state, a verb,
a config flag, a gate, or a second representation of existing truth? If
yes, redesign first. The correct fix usually deletes or relaxes something.

**N issues with one root = one fix at the root, closing all of them.**
N issues never justify N patches; a per-issue patch list is a red flag that
the root was never found.

## Exits

- Every refusal, block, or failure message must **name an executable
  exit** — a command or action the user can actually run next.
- **A WRONG exit outranks a MISSING one.** A dead end tells the user to
  stop; advice that does not work sends them in circles blaming themselves.
- Verify the named exit by executing the printed command before calling the
  message fixed. Never assume runnability.
