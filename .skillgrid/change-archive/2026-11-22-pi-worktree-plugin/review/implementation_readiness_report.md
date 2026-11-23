# Goldfish — Implementation Readiness (Stage 9, REVIEW) — @devopstales/pi-worktree

**Isolation attestation:** fresh context (harness-enforced by construction); read-only enforced by discipline — no file, shell-state, or other write was touched; no project instruction files (this prompt is the entire context); no network or MCP (harness-enforced).

---

## Finding 1

**Severity:** Major
**Affected Scope:** Task 5 Step 3 (`src/scratch.ts` shape snippet); `src/ui.ts` (File Structure + Task 8 "dialog half")

**Finding & Guidance:** Fails Zero-Decision Rule: the plan leaves the **shape of `ui.ts`** to the implementer. `updateFooter(ctx, created)` is called in the Task 5 snippet but never defined anywhere; `rebuildCache(ctx)` is called with a comment describing its behavior but never given a signature or storage contract (the Task 5 snippet's only "shape" for the tool's no-id path is a bare `listCheckpoints(...)` promise returned as the tool result — no return shape, no diff-preview mechanics, no truncation); `ctx.ui.select("Restore?", ["files-only", "cancel"])` and the `/checkpoint` handler's `/* same flow */` comment define the dialog's options but not its return contract or how a cancelled restore is reported; Task 8 adds "dialog half if not already present" on top. A fresh session writing Task 5 would have to invent the entire UI module — function signatures, footer widget construction, cache storage, and the tool-result shape for the list path (is it the raw `Checkpoint[]`? a truncated table? what fields?). Guidance: pin a `src/ui.ts` interface block — `footer(ctx, count: number)`, `restoreConfirm(ctx, id: string): Promise<"files-only" | "cancel">`, the tool's list-mode return shape (e.g. the `Checkpoint[]` fields surfaced, any truncation), and what `rebuildCache` loads into which in-memory structure consumed by the no-id tool path.

## Finding 2

**Severity:** Major
**Affected Scope:** Task 7 Step 3 (`wt.ts` tool body), Global Constraints ("wt requirement")

**Finding & Guidance:** Fails Zero-Decision Rule: the plan leaves **the tool-result shape and the error strategy for a failed/aborted `wt` invocation** to the implementer. `runDirectedWt` returns `{stdout, stderr, code, directive?}` (Task 6), but nothing says what `wt.ts` returns to the model from `execute` — the `stopped`-plus-restart-directory payload for print/json mode is named (MH-7) but its structure is not, the codemode `status: "queued"` payload is referenced in findings R1 without being restated in the plan, and the case "wt exited non-zero, no directive, no approval prompt" (error passthrough vs. normalized error object, which fields) is entirely open. Guidance: state the tool-result contract explicitly — e.g. on success `return {code: 0, stdout: <relevant tail>, directive, movedTo?, status}`; on non-zero with no directive `return {code, stderr}` (or the chosen error shape); in print/json mode with a directive `status: "stopped"` + restart path field named. One code block of the return statements suffices.

## Finding 3

**Severity:** Major
**Affected Scope:** Task 7 Step 1 (approval test) and Step 3 (approval dialog)

**Finding & Guidance:** Fails Zero-Decision Rule: the plan leaves **the approval-needed detection and the retry path** to the implementer. The text says "parse worktrunk's approval-needed output" — but findings R1 does not pin the output format (it describes behavior: "Pi shows every unapproved command in a confirmation dialog"), so the implementer must choose what marker/string identifies the condition, where in the output it lives, and how the command list is extracted. Equally open: what "retries once" means at the process level — a second `runDirectedWt` with the same argv (implied, but unstated), what the tool returns after the single retry (success payload, or error naming the still-unapproved commands), and the non-interactive-mode return shape ("fails and lists what to approve" — in which field, in what format). Guidance: either pin the detection rule (the exact `wt` stderr marker line and the command-list line format, verified against the Task 7 min-version pin), or make the approval-group test a behavior contract: stub `wt` that fails on first run with a documented marker line, assert the dialog input list equals the commands, assert `wt config approvals add` argv, assert exactly one retry, and assert the final return object.

## Finding 4

**Severity:** Major
**Affected Scope:** Task 6 Step 3 (`observeWorktreeRoot`); Task 7 (recovery `prefer the default branch's worktree`)

**Finding & Guidance:** Fails Zero-Decision Rule: the plan leaves **two "default branch" determinations** to the implementer, in two places. (a) `observeWorktreeRoot` returns a note "when it is not under `<projectroot>/.worktree/`" — but `git worktree list --porcelain` names the worktree containing `cwd` with a `branch` line, not a default branch; how the plugin distinguishes the primary checkout's worktree from others in order to compute `<projectroot>` is not written down (the branch recorded in the fixture is `agent-demo`, not `main`). (b) Recovery "prefer the default branch's worktree" — default branch meaning the remote's HEAD (`refs/remotes/origin/HEAD`), `refs/heads/main`/`refs/heads/master` by name, or `wt`'s configured base is unspecified, and the fallback when none of the recorded worktrees is the default branch's is unspecified (R1 only says "preferring main"). Guidance: name the exact rule once in Global Constraints and reference it from both tasks — e.g. "default branch = the branch of `git rev-parse --symbolic-ref refs/remotes/origin/HEAD`, else the first recorded worktree in `git worktree list` order", and "primary checkout = the worktree at `git rev-parse --git-common-dir`'s parent; `<projectroot>` = that path".

## Finding 5

**Severity:** Major
**Affected Scope:** Task 7 Step 1 (recovery test), Task 6 (mutex)

**Finding & Guidance:** Fails Zero-Decision Rule: the plan leaves **the storage and keying of the recorded-worktree list** to the implementer. `recordWorktrees(cwd)` has no signature beyond the name, no storage choice (module-level singleton, per-session `Map`, or persisted), no keying, and no reset on session/move; the recovery test calls `recordWorktrees(agentDemoWorktree)` at module scope while Task 6 requires an in-process mutex around directive-file lifecycle — whether recording participates in that mutex is unstated, and a second test in the same file (e.g. the mutex test) could corrupt the fixture if recording is not idempotent per cwd. Guidance: pin `recordWorktrees(root: string): void` as idempotent upsert into a module-level `Set<string>` (or `Map<root, meta>`), cleared at `session_start` in `wt.ts`, and state explicitly that it is not part of the directive mutex.

## Finding 6

**Severity:** Major
**Affected Scope:** Task 2 Step 3 (capture mechanics), MH-3

**Finding & Guidance:** Fails Zero-Decision Rule: the plan leaves **the `safeClean` file-listing mechanics** to the implementer. The comment says "delete files in the snapshot that are neither preexistingUntracked nor skippedLarge*" and R3 restates it, but what enumerates the snapshot's file set is not decided: `git ls-tree -r --name-only <worktreeTreeSha>` is the only listing that covers tracked **and** captured-untracked files in one call, while `git diff --name-only <headSha> <worktreeTreeSha>` would need the preexisting-untracked and large-skip lists folded in separately and does not cover captured untracked that also existed pre-checkpoint as different content. The correct-but-unwritten choice determines whether restored worktrees can be left with stale captured-untracked files (MH-3's "post-checkpoint untracked files are removed" fails). Guidance: name the listing command in the Task 3 restore body — `ls-tree -r --name-only` on the snapshot tree, minus `preexistingUntracked`, `skippedLargeFiles`, `skippedLargeDirs` (prefix), and any path under an `IGNORED_DIRS` component.

## Finding 7

**Severity:** Major
**Affected Scope:** Task 7 Step 3 (linked-session creation), Task 6 Step 1 (stub-wt tests), Global Constraints (`wt` requirement)

**Finding & Guidance:** Fails Zero-Decision Rule: two choices are open. (a) **Linked-session creation mechanism.** `pi-worktree` is a new repo with no access to pi-worktrunk's 1670-LOC `worktrunk.ts`; findings R1 documents the protocol and the Pi API surface but not the session-creation call. The plan says "create the linked 'Session recovered' session" and "linked-session move" without naming the `ctx.sessionManager` / `ctx.ui` mechanism, so the implementer must re-derive pi-worktrunk's move internals from upstream source (which is outside this repo and outside the reading list). (b) **Stub `wt` in tests.** Task 6's runner tests say "stub wt that writes B's path into the directive file" and "stub wt exits 1" — a stub that honors the `WORKTRUNK_DIRECTIVE_CD_FILE` env protocol. How the stub is provided (a temp executable on a PATH prefix, an env-var override of the wt binary path in `runDirectedWt`'s spawn, or a test-only seam) is not decided, and the Global Constraint "wt on PATH (real binary)" vs. "skipped with a named reason when wt is absent" does not cover which tests may use the stub. Guidance: state the session-move call sequence (the two or three Pi API calls, named) and the stub mechanism (e.g. `wt` binary path injectable via an env var the tests set, or PATH prepending of a fixture bin dir containing a one-line shell stub).

## Finding 8

**Severity:** Major
**Affected Scope:** Task 9 Step 1 + Step 3 (contract-drift guard, `evals/evals.json`)

**Finding & Guidance:** Fails Zero-Decision Rule: three format choices are open. (a) **Fence value grammar.** Task 11 writes the fence as `key: value` with JSON-ish values, but the Task 9 parser contract is only "key: value list, no prose regexes" — how a quoted regex string is distinguished from an unquoted number, and how the `IGNORED_DIRS` array literal is split, are left to the implementer (an implementer who writes the parser to split on the first `:` will misparse `AGENT_BRANCH_RE`'s regex, which contains no `:` but is quoted; an implementer who trims quotes asymmetrically fails the guard). (b) **`evals/evals.json` format.** The plan says "evals in the repo's `evals/evals.json` format" — but this is a *new* repo with no such format; the only format that exists is skillgrid-skills' `execute/evals/evals.json` (`{skill_name, evals: [{id, prompt, expected_output, files}]}`). Whether to copy that shape verbatim, or the parallel `scenarios.yaml` shape (the two ported scenarios live in `scenarios.yaml`), is a format choice the plan does not make. (c) **Scenario content.** "marker-lifecycle + approval scenarios" are named but their prompts, assertions, and `eval_type` are not written; the two ported ones are, by reference. Guidance: pin the fence grammar in one sentence (exact: `KEY: value` where value is either a bare integer, a double-quoted string with no escaping rules needed, or a JSON array of double-quoted strings), state that `evals.json` copies skillgrid-skills' execute format verbatim (one `evals.json`, the two ported scenarios re-expressed as prompt/expected_output), and either write the marker/approval scenarios or defer them to the changelog with an explicit "v0.1.0 ships the two ported scenarios only" decision.

## Finding 9

**Severity:** Major
**Affected Scope:** Task 8 Step 3 (marker write commands), Global Constraints (Markers)

**Finding & Guidance:** Fails Zero-Decision Rule: the plan leaves **the exact marker-state write commands** to the implementer. The stale-report text pins the *clear* command (`wt config state marker clear`), but `markWaiting(cwd)` / `markWorking(cwd)` — the set half of the lifecycle — are not given their `wt config state marker …` forms (set value, subcommand, whether the emoji is a literal argument). The Task 8 test group ("drives `wt` and reads `wt list`") depends on them, and an implementer who guesses the set command wrong will have a green `wt list` in a happy-path run that does not exercise the write. Guidance: state the two set commands in Task 8's Produces or Step 3 — e.g. `wt config state marker set <emoji>` (or the actual R1 marker-table forms — findings R1 names the lifecycle but not the argv) — or make Task 8 Step 3 the place where they are recorded against the Task 7 min-version pin, same as the README pin.

## Finding 10

**Severity:** Major
**Affected Scope:** Task 2 Step 3 (capture step 2)

**Finding & Guidance:** Fails Zero-Decision Rule: the plan leaves **which paths are `add --all`-ed into the temp index** to the implementer. Step 2 says "filtered untracked+modified via `git status --porcelain=v1 -uall` → apply IGNORED_DIRS (any path component), MAX_UNTRACKED_FILE_SIZE, MAX_UNTRACKED_DIR_FILES" — but the mechanics of "apply" are open: which porcelain status codes are selected (modified ` M`/`M `, staged `M `/`A `? deleted ` D`?), how a path whose *directory* exceeds `MAX_UNTRACKED_DIR_FILES` is detected from a per-file `status` stream (the 200-file cap is a per-directory count that `status -uall` does not precompute — the implementer must choose a pre-pass, e.g. counting untracked entries per top-level untracked dir, or a `find`-style walk), and whether tracked modified files go through the size cap at all (the contract's ">10MiB" is scoped to *untracked* files — the plan restates the cap unscoped in the step). These are exactly the kind of choices where two correct-looking implementations diverge on a repo with a 250-file `dist/`-like untracked dir. Guidance: pin the status-code filter, the per-directory count pre-pass (command + which directory level), and restate the caps with their contract scoping (untracked-only) in the step.

## Finding 11

**Severity:** Major
**Affected Scope:** Task 2 Step 1 (capture test), Task 2 Step 3 (`lastTree`), Task 3 tests (module state)

**Finding & Guidance:** Fails Zero-Decision Rule: the plan leaves **the module-level `lastTree` dedup state and its cross-test lifecycle** to the implementer. (a) The Step 1 capture tests never call `resetSessionState()` (Task 3's tests do), and the "read-only turn is skipped" test is the second `createCheckpoint` call in the `beforeEach` repo — but bun:test runs tests in a file against a fresh module instance only per file, so if any test in the capture group leaves `lastTree` populated for `s1` (they do — the first test creates a ref for `s1`), the dedup test's "created" expectation depends on test execution order or on a reset the plan does not write. (b) `resetSessionState()` is *produced* by Task 4 but *consumed* by Task 3's tests — a fresh session executing Task 3 will write tests that call a helper that does not exist until Task 4, contradicting the plan's own Step 2 "Expected: FAIL — `restoreCheckpoint is not defined`" (which would instead be a missing-helper error). Guidance: either move `resetSessionState()` into Task 2's Produces (it guards Task 2's own dedup map) and add the call to Task 2's tests, or restate that Task 3's tests are written *after* Task 4 lands — but the plan's sequential tracer order and the Step 1 test code as written both assume the opposite.

## Finding 12

**Severity:** Major
**Affected Scope:** Task 10 Step 1 (`tests/e2e.sh`), Task 9 Step 4 / MH-14 (`npm test`)

**Finding & Guidance:** Fails Zero-Decision Rule: the plan leaves **`e2e.sh`'s harness mechanics** to the implementer. The scenario sequence is fully specified (good), but *how* it is driven is not: what makes the Pi session "scripted" (TUI keystrokes via an expect-style tool? `pi print`/`-p` invocations? the Pi RPC/JSON mode?), how a single session spans the `wt switch --create` move (a move creates a *linked* session — does the script drive the new session, and how does it know its id/path), and how the "restore after a dirty turn" step is asserted (dialog confirmation in a non-interactive context). Separately, the plan's own package.json defines `"test": "bun test"` while MH-14 and the validation table (R12) require `npm test` to be green in a bare Node/Bun environment — a fresh session will write `e2e.sh` with whichever invocation Pi accepts locally and have no instruction on how the harness picks that up, so the R12 "npm test green" gate has no defined script surface for e2e. Guidance: state the driver (e.g. "Pi JSON/RPC mode via a Node script in `tests/e2e/` that `e2e.sh` invokes; linked session located via `ctx.sessionManager.getSessionFile()` from the move result; restore driven by invoking the `scratch-restore` tool through the same RPC session") and either add `"test": "bun test && bash tests/e2e.sh"` to Task 10's package.json edit or name e2e as manual-only with the R12 row restated to unit+bench only.

## Finding 13

**Severity:** Nit
**Affected Scope:** Task 9 Step 1 (contract-drift fence parse)

**Finding & Guidance:** Fails Zero-Decision Rule (narrow): the fence's closing line is not pinned. Task 11's fence is opened by `~~~skillgrid-constants` and closed by `~~~`; the Task 9 parser says "a line-anchored `~~~skillgrid-constants` block" — the terminator is not named, so an implementer who matches "to the next line starting with `~~~`" will break if a `~~~` ever appears inside the block, and one who hard-codes "7 lines after the opener" will break when a constant is added. One sentence closes it: "the block ends at the next line that is exactly `~~~`".

## Finding 14

**Severity:** Nit
**Affected Scope:** Task 1 Step 3 (`package.json`), Global Constraints (Package)

**Finding & Guidance:** The plan's package.json snippet carries `name/version/license/type/pi/engines/scripts/devDependencies` but not `files`, `main`, or `description` — for a published npm package whose Pi manifest references `./src/*.ts`, the tarball contents are an open choice (npm's default `files` inclusion vs. an explicit `"files": ["src", "references", "evals"]`). Not a behavior risk for the v0.1.0 smoke (Task 10 installs the packed tarball and `pi -e`-style loading of `./src/scratch.ts` will fail if `src/` is excluded by an explicit `files` list the implementer adds or omits inconsistently), but one field in the snippet removes the choice: add `"files": ["src", "references", "evals", "README.md", "changelog"]`.

---

## PASS/FAIL

**FAIL** against the Zero-Decision Rule.

Justification: the blueprint is strong on invariants, threat surfaces, and the pure-git core (Tasks 1–4 are close to executable as written — the capture/restore/prune mechanics, constants, and test assertions are pinned to the level where a competent engineer makes only loop-style choices). But the wt side and the wiring layer leave exactly the class of choices the rule forbids: the `wt.ts` tool-result/error contract, the approval detection-and-retry mechanics, the marker write argv, the recorded-worktree storage, the session-move call sequence, the `safeClean` listing, the fence/evals formats, and the e2e driver are all named only by behavior, not by the concrete shape an implementer must produce on the first attempt. Twelve Major open choices (Findings 1–12) each require the implementer to either re-derive pi-worktrunk internals from a source outside the reading list or pick between two or more defensible designs; each is fixable with one pinned sentence, one signature block, or one code snippet in the named task. Two Nits (fence terminator, tarball `files`) round out the list. Slicing should not proceed until Findings 1–12 are closed.
