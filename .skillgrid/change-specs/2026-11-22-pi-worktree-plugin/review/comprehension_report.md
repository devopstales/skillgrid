# Comprehension Review — blueprint.md

Harness-isolation attestation: omp task subagent, fresh context (parts 1 & 4 enforced by construction); no network/MCP used; parts 2 (read-only) and 3 (no project instructions) enforced by reviewer discipline — no file was edited by me other than this report.

Mode: REVIEW · Stage 6 of 5 (Comprehension Test) · Persona: naive reader who just joined the project.

## What this proposes

The blueprint builds a new npm package `@devopstales/pi-worktree` — one Pi coding-agent plugin made of two extension files plus a pure (Pi-import-free) core. The scratch side captures, at the end of every mutating turn, a non-committing git ref under `refs/skillgrid/` that records branch, HEAD, index tree, and worktree tree, with a safe restore that refuses cross-branch restores and never deletes pre-existing untracked, ignored, or oversized files. The worktrunk side wraps the `wt` CLI with the internal directive protocol (a temp file that tells Pi which new directory the worktree landed in), moves the Pi session into a linked session on worktree moves, recovers without replay when a session's working directory disappears, and enforces three skillgrid invariants: observing the actual worktree root at first use, rejecting disallowed branch names at `switch --create` time in a worktree context, and prepending `--no-squash --no-commit` to `merge` calls. Everything is specified as 11 tasks with concrete failing-test-first steps, a must-have list, a validation table per requirement, a threat matrix, and an in-repo doc-edit task that makes the contract constants machine-readable so a drift-guard test can pin the plugin's constants to them.

## How the existing system works, per this document

The skillgrid worktree standard lives in `_shared/work-unit-commits/README.md`: it defines tooling tiers (harness-native tool wins, then worktrunk `wt` when present with worktrees in `<projectroot>/.worktree/`, then plain `git worktree add`), turn-level scratch checkpoints as prunable `refs/skillgrid/` refs (temp-index capture, tree-SHA dedup, 50-refs-per-session cap, cross-session prune at startup, before-restore snapshot, safe-clean rules), and worktree lifecycle safety (allow-listed branch names, no-added-changes branch deletion, force-remove only with a backup ref, fan-out reconciliation via 🤖/💬 markers). Two community Pi extensions cover pieces but neither is skillgrid-aware: pi-worktrunk runs `wt` in-Pi with linked-session moves, no-replay cwd-removal recovery, markers, and approval dialogs; pi-rewind/checkpoint-pi provide git-ref turn checkpoints with temp-index mechanics, dedup, and pruning in their own ref namespace. The gap this blueprint closes: the harness tier-1 slot ("harness-native tool owns session state") has no implementation for the Pi coding agent, so a skillgrid repo running Pi falls back to git and loses session-follows-worktree behavior.

---

## Finding 1

1. **Severity:** Major
2. **Affected Scope:** Task 6, Step 1 test block ("checkBranchAllowList: worktree context rejects with nearest suggestion"), lines ~674–680
3. **Discrepancy Description & Rationale:** Task 6's Interfaces declares `checkBranchAllowList(branch: string, inWorktree: boolean): Promise<{ok: boolean; suggestion?: string}>` — a Promise — but the Step 1 test calls a function named `checkBranchAllowListSync` (four times) that is never declared or defined anywhere in the blueprint. A fresh executor reading only this document must guess that the test names a sync variant of the declared Promise function, or that the declaration is wrong. The Step 1 code is explicitly "the failing test you write first," so a name that exists nowhere else is an ambiguity that will surface at the TDD step, not at design time. Rename the test calls to the declared name (with `await` if it really is a Promise) or declare the `Sync` variant.

## Finding 2

1. **Severity:** Major
2. **Affected Scope:** Task 4, Step 1, first prune test ("51st capture prunes the oldest, keeping 50"), lines ~537–545
3. **Discrepancy Description & Rationale:** The test reads refs via `git(["for-each-ref", `refs/skillguard/`.replace("skillguard","skillgrid")], ...)`. The `.replace` exists only because the ref name is mistyped as `skillguard` — the expression evaluates to the correct `refs/skillgrid/`, so the test passes, but a naive reader (and a future maintainer) will conclude the scratch namespace is `refs/skillguard/`, directly contradicting OD-1, Global Constraints, and the drift guard. It looks like a deliberate obfuscation or a typo that got fossilized into the spec. It should just be the string `"refs/skillgrid/"`.

## Finding 3

1. **Severity:** Major
2. **Affected Scope:** Task 5, Step 3 code block (`src/scratch.ts` shape), lines ~626–642
3. **Discrepancy Description & Rationale:** The shape code calls `restoreCheckpoint(repoRoot(ctx.cwd), toplevelOf(ctx.cwd), p.id)` — three arguments, including a helper `toplevelOf` that appears nowhere else in the document — while Task 3 declares `restoreCheckpoint(root: string, id: string)` (two arguments) and every other call site (Task 3 tests, Task 5's own Consumes list) uses the two-argument form. A naive reader cannot tell whether the extra argument is a vestige of the retired OD-2 worktree-root half (which would make sense) or a real third parameter; either way the document is self-contradictory at a concrete code line. Pick one signature and make the call match it.

## Finding 4

1. **Severity:** Major
2. **Affected Scope:** Task 9, Step 1 test code (`tests/contract-drift.test.ts`), lines ~805–813
3. **Discrepancy Description & Rationale:** The default contract path is hardcoded to `/data/git/AI/skillgrid-skills/.agents/skills/_shared/work-unit-commits/README.md` — a machine-specific absolute path that is the author's workstation layout, while the Interfaces section of the same task says the env override defaults to "the in-repo path when developed inside skillgrid-skills." The File Structure section also states the contract path exists only "in this repo" (skillgrid-skills), so inside the new plugin repo there is no in-repo copy for the default to resolve to; the test as written fails with an ENOENT on any machine that is not the one where it was drafted, and the failure message ("contract fence not found") will mislead rather than name the missing file. The default should be a path derived from repo-relative resolution (or the env var should be made mandatory with a clear error), not a literal absolute path from one developer's box.

## Finding 5

1. **Severity:** Major
2. **Affected Scope:** Task 11, Step 2 (the `skillgrid-constants` fence to append to `work-unit-commits/README.md`), lines ~876–888
3. **Discrepancy Description & Rationale:** The fence block to be appended to the contract markdown is shown nested inside an outer ` ```markdown ` code fence in the blueprint. In the rendered blueprint, the inner ` ```skillgrid-constants ` line terminates the outer fence, so the remaining lines (`REF_BASE: ...` through the closing ` ``` `) fall outside any code fence and the very next line (`` ``` ``) opens a new one — the displayed "block to append" is visually broken, and a naive executor copying what they see cannot be sure which lines belong to the fence. Worse, if the inner fence is later indented when appended to the real contract doc, the drift guard's "one parse" of a ` ```skillgrid-constants ` line-anchored fence will not match. Show the fence in a form that survives rendering (e.g. an indented code block or a differently-delimited outer fence) and state explicitly whether the appended fence must start in column 0.

## Finding 6

1. **Severity:** Nit
2. **Affected Scope:** Global Constraints → "Zero runtime dependencies", line ~113
3. **Discrepancy Description & Rationale:** The constraint says dependencies are "beyond `@earendil-works/pi-coding-agent`, `@earendil-works/pi-tui`, `typebox` (all provided by the Pi ecosystem)." Task 1's `package.json` has none of these as `devDependencies`/`peerDependencies` at all (only `bun-types`), which is consistent with "provided by the Pi ecosystem" but the constraint sentence reads as if the package would depend on them. A naive reader will hunt for where these appear in `package.json`. One clause ("provided by the host Pi install; declared peer-only or not at all") would remove the hunt.

## Finding 7

1. **Severity:** Nit
2. **Affected Scope:** Task 3, Step 4 ("Expected: PASS (10 tests total)"), line ~501
3. **Discrepancy Description & Rationale:** The cumulative test count is wrong: Task 2 ends at 4 tests, Task 3 adds 5 (matching-branch restore, before-restore checkpoint, branch-mismatch refusal, missing-ref refusal, detached-HEAD refusal) = 9, not 10; the document's own Task 4 then expects "11 tests" after adding 2 prune tests (9 + 2 = 11), which confirms the Task 3 figure is off by one. For a document whose Self-Review section promises "every task has concrete files, test code, and commands," a miscounted expected pass total in the run step is exactly the kind of internal inconsistency a fresh reader checks.

## Finding 8

1. **Severity:** Nit
2. **Affected Scope:** Task 7, Step 1 test block (recovery group), lines ~729–735
3. **Discrepancy Description & Rationale:** The test comment says "temp repos: main + agent-demo worktree" but the code uses `mainWorktree` and `agentDemoWorktree` as bare, undefined identifiers (no fixture setup shown) — unlike Task 2, whose tests come with a full `beforeEach` fixture, and Task 7's Step 2, which only says "`wt.ts` + recovery missing." A naive reader cannot tell whether these are meant to be module-level fixtures to be written, variables the executor should invent, or a copy-paste gap. The other recovery test in the same block has the same shape (comment-only body). At minimum, name the fixture the test assumes (a `wt`-initialized temp repo with a default-branch checkout and one `agent-*` worktree) so the executor knows what to build before the test can run.

## Finding 9

1. **Severity:** Nit
2. **Affected Scope:** Task 10, Step 1 (`tests/e2e.sh` description), line ~835
3. **Discrepancy Description & Rationale:** The e2e script is required to assert "marker sequence" and "allow-list rejection on `feature-e2e`", but the Task 10 step lists neither a `wt config` setup (marker state needs worktrunk configured for the temp repo) nor where the allow-list context comes from (allow-list rejection only fires "in a worktree context," i.e. the `feature-e2e` attempt must happen after the `agent-e2e` move — the step implies the ordering but never states it). A naive executor scripting `e2e.sh` from this single bullet will either skip the marker assertion or invent the setup. The same self-containedness the other tasks enjoy (explicit fixture + explicit command + explicit expectation) is missing here; the bullet should name the setup steps or point at the Task 6/Task 8 fixtures it reuses.
