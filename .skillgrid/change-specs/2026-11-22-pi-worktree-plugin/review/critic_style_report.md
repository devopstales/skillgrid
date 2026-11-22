# Goldfish Stage 7 — Critic Report (prose / terminology / consistency)

**Target:** `blueprint.md` (968 lines, read in full)
**Mode:** REVIEW — Stage 7, style/consistency persona
**Harness-isolation attestation (R37):** fresh context (part 1 and part 4 — no network/MCP — enforced by the omp task subagent construction); no project instruction files were provided in this prompt; part 2 (read-only) I enforced by discipline: the only file I touched is this report (no target edits, no shell commands other than read-only `grep`/`sed`/`awk`); part 3 (no project instructions) holds because this brief plus the explicit reading list (findings.md, briefing.md, work-unit-commits/README.md) is my entire context.

Scope note: the OD-2 retirement is treated as settled (per the stage-7 context). Findings below do NOT re-raise the retired worktree-root half of the check or the other settled findings (verifyStandardRoot rename, detectForeignMarkerPlugin, generated catalog, tsconfig/statSync, prose-regex fence); several findings document *residue* of those retirements still present in the text.

---

## Blocker findings

1. **Severity:** Blocker
   **Affected Scope:** Self-Review → item 6 "Spec reconciliation (against briefing.md)" (around line 946)
   **Discrepancy Description & Rationale:** The spec reconciliation states "**Contradictions:** none." and asserts "Every briefing requirement R1–R13 maps to a task", but the document contradicts briefing R7 in at least three places:
   - briefing R7's requirement text: "on first `wt` use, the plugin **verifies** worktrunk's path template places worktrees in `<projectroot>/.worktree/`", acceptance scenario `` `happy path standard-root-verified` ``;
   - blueprint Global Constraints § Standard root: "the plugin **observes** the actual root at first use and notes a one-line 'observed root differs' … it **never verifies** the config string itself — R7";
   - blueprint Task 6's SATISFIES line cites `` `happy path standard-root-observed` `` — a scenario name that exists in neither the briefing's acceptance scenarios nor this blueprint's own Validation table (whose R7 row uses prose, not scenario names).
   The R7 mapping in item 6 ("R7→T6") is presented as satisfied, so "no contradictions" is false on the face of the two documents. The reconciliation is either missing the R7 verify→observe revision or it silently renamed the briefing's scenario without recording the rename. Either way an executor reading only the blueprint cannot tell which behavior (verify or observe, and which scenario id) is authoritative. This is the single most consequential inconsistency in the document: a requirement-level contradiction inside the section whose stated job is to detect them.

2. **Severity:** Blocker
   **Affected Scope:** Threat Matrix, row "Untrusted input" — Test column (around line 100)
   **Discrepancy Description & Rationale:** The Test column names a function that does not exist anywhere in the document: "ref id with `/` or `..` → rejected at `loadCheckpointFromRef`". The actual API defined in Task 3's Interfaces (Produces) is `loadCheckpoint(root: string, id: string): Promise<Checkpoint | null>`; `loadCheckpointFromRef` appears exactly once in the whole blueprint and only in this cell. The test it describes (ref-id validation on load) is otherwise unplaced in any task. An executor writing the test from the threat matrix either invents a second function name or cannot tell which function is the seam. Rename the reference to `loadCheckpoint`.

---

## Major findings

3. **Severity:** Major
   **Affected Scope:** Task 3, Step 3 implementation snippet (around line 489) vs Threat Matrix "Commit state" row (around line 102)
   **Discrepancy Description & Rationale:** The refusal-detail string for detached HEAD is stated two ways that cannot both be what `restoreCheckpoint` returns. Task 3's code shows the implementation:
   `return { status: "refused", reason: "branch", detail: `checkpoint on '${cp.branch}', current '${curBranch}'` }` — where, per the comment on the line above, `curBranch` is `"DETACHED"` (and the Task 3 test around line 467 asserts `res.detail).toContain("DETACHED")`).
   The threat matrix's test for the same case expects: `detached-HEAD restore refused with "branch mismatch: (detached)"` — a different format (`branch mismatch:` prefix, lowercase `detached` in parentheses) that the shown code never produces. This is a duplicated statement with different values: an executor reconciles by picking one, and the one the threat matrix names will make the matrix's own test fail against the shown implementation. Pick one exact string in one place (ideally the Task 3 code, with the matrix cell referring to it).

4. **Severity:** Major
   **Affected Scope:** Threat Matrix, row "Git repository selection" — Execution boundary and Test columns (around line 101)
   **Discrepancy Description & Rationale:** The row describes a check that no task implements and no test covers. Boundary: "the **pair check** compares `--show-toplevel` outputs (realpath'd), never user-typed paths". Test: "`tests/scratch.test.ts`: pair check from a subdirectory of the worktree (same toplevel → pass); from the primary checkout (different toplevel → refuse)".
   - After OD-2's retirement (stated in the document itself: "the branch check already implies root identity"), "pair check" is dead terminology — the check that exists is the **branch-keyed check**, which compares branch names, not `--show-toplevel` outputs. A branch check cannot "compare `--show-toplevel` outputs".
   - No task in Build Order, File Structure, or Tasks 1–11 creates or tests a toplevel comparison: Task 3 (restore + branch-keyed check) is the only restore task, and its test groups are restore/refusal/missing/detached. The threat matrix promises a test that the plan never schedules, in a test file whose groups are enumerated elsewhere (File Structure: "capture/dedup/restore/prune/branch-check/unborn-HEAD/detached groups").
   - The subdirectory half ("pair check from a subdirectory … same toplevel → pass") is also a different operation than the row's premise: the boundary text says git calls always pass `{cwd: root}` with root from `--show-toplevel`, so a "pair check from a subdirectory" has no defined inputs — the check would not see the subdirectory at all.
   Either retire the row to the branch check (and say what, if anything, protects the "runs from a subdirectory" case) or add the toplevel comparison to a task. As written, this row states a control the plan does not contain.

5. **Severity:** Major
   **Affected Scope:** Global Constraints § Pi floor (around line 112) vs Task 1, Step 3 package.json snippet (around line 232)
   **Discrepancy Description & Rationale:** The document says the Pi floor "is" stated in package.json's `engines`, but the one concrete package.json in the plan does not say it. Global Constraints: "**Pi floor: 0.99.1+** … `package.json` `engines` / README state it." Task 1's package.json snippet — which the same section explicitly designates as the authoritative shape ("see Task 1's snippet, which carries only `bun-types` as a devDependency") — has no `engines` field at all. The constraint's `engines` claim is contradicted by the only package.json the document specifies (the README half is covered by Task 9's README content list). Either add `engines` to Task 1's snippet or drop "`package.json` `engines`" from the constraint.

---

## Nit findings

6. **Severity:** Nit
   **Affected Scope:** Build Order & Dependencies, mermaid diagram node T3 (around line 163)
   **Discrepancy Description & Rationale:** The node is labeled `T3[Task 3: scratch-core restore + pair check]`. After OD-2's retirement the Task 3 heading itself is "restore + **branch-keyed check**" and the task's own prose calls it the branch-keyed check throughout; "pair check" survives only in the diagram. (The other "pair" occurrences — lines 101/102, 183, 964 — are in context that references the retired half or is a retirement record, which is defensible; this node is the one that reads as the current name.) Rename the node to match the Task 3 heading.

7. **Severity:** Nit
   **Affected Scope:** Task 3, Step 1 test snippet (around lines 423–447)
   **Discrepancy Description & Rationale:** The "restore on matching branch" test (around line 426) asserts `expect(r.status).toBe("created")` for sessionId `s1` turn 1 with no file changes since the capture-group test "read-only turn is skipped" (Task 2, around line 341) already captured the identical tree for sessionId `s1`. Per the Task 2 implementation, dedup keys on the module-level `lastTree` map, which persists across tests in the same file; `resetSessionState` is a Task 4 deliverable, not yet written at Task 3, and nothing in the shown `beforeEach` clears it. So the checkpoint this test needs would be `skipped`, and the test would fail against the very dedup mechanism it tests (the "restore takes a before-restore checkpoint first" test a few lines later fails the same way). Either each group calls `resetSessionState` in `beforeEach`, or the test file header needs a note that this is done per test. As written the snippet is internally inconsistent with the dedup mechanism it depends on.

8. **Severity:** Nit
   **Affected Scope:** Validation table, R13 row (around line 90)
   **Discrepancy Description & Rationale:** The Failure column says "contract file moved (**path fallback error**, not silent pass)". There is no path fallback in the drift-guard design: `SKILLGRID_CONTRACT_PATH` is the required env var, and its stated behaviors are (Task 9) unset → "set SKILLGRID_CONTRACT_PATH …", missing fence → "contract fence not found at <path> — Task 11 not applied?". A moved file is simply an unreadable path. "Path fallback error" is a third behavior that no task defines, and "fallback" implies a secondary path exists. Rename to the actual behavior, e.g. "unreadable path → hard failure, not silent pass".

9. **Severity:** Nit
   **Affected Scope:** Task 8, Step 3 (around line 785) vs findings.md R1
   **Discrepancy Description & Rationale:** The stale-marker report text is given as `run `wt <marker-clear>` to remove` — a literal placeholder in what is otherwise a fully pre-made string. Findings.md R1 names the actual command: `wt config state marker clear`. The Zero-Decision Rule (no unfilled slots in a blueprint) is breached by a `<…>` slot in a user-visible string. Replace with the real command.

10. **Severity:** Nit
    **Affected Scope:** Global Constraints, "Constants" bullet (around line 116)
    **Discrepancy Description & Rationale:** "`IGNORED_DIRS` = the **13** upstream dir names (`node_modules`, `.venv`, `venv`, `env`, `.env`, `dist`, `build`, `.pytest_cache`, `.mypy_cache`, `.cache`, `.tox`, `__pycache__`) + `.worktree`" — the parenthetical enumerates **12** names. (The "13" matches findings.md R2's "13 ignored dir patterns (… +1)", where the 13th is checkpoint-pi's unspecified pattern; the contract's own filter set, and the Task 1 constant, and the Task 11 fence, all contain 12 upstream + `.worktree` = 13 total.) Either drop the count ("the upstream dir names") or list the 13th, so the count and the enumeration agree within this bullet.

11. **Severity:** Nit
    **Affected Scope:** Task 7, Step 3 (around line 738)
    **Discrepancy Description & Rationale:** The sentence describing the approval dialog ends mid-thought: "on approve run `wt config approvals add` for exactly those…" — the snippet trail-off inside a prose step. The complete behavior (retry once, dialog re-shows on decline, non-interactive fails with a list) is in MH-12 and findings.md R1, so the step should cite them or close the sentence instead of ending on an ellipsis.

12. **Severity:** Nit
    **Affected Scope:** Task 5, Step 3, `src/scratch.ts` snippet (around line 606)
    **Discrepancy Description & Rationale:** `turn: turnIndex` — `turnIndex` appears nowhere else in the blueprint (no producer, no definition in any task; the `Checkpoint` type has a `turn` field but nothing assigns it). Task 2's id format (`${sessionId}-t${turn}-…`) shows the turn number matters for the ref id, so this is not a throwaway. Define where `turnIndex` comes from (e.g. a counter incremented per `turn_end` per session, reset in `session_start`) or name the Pi surface that provides it.

13. **Severity:** Nit
    **Affected Scope:** Task 3, Step 3, `restoreCheckpoint` snippet (around line 489)
    **Discrepancy Description & Rationale:** The implementation references `ZEROS` (`if (cp.headSha !== ZEROS)`), but no task declares it. Task 2's prose mentions "ZEROS if unborn" and the threat matrix cites "R3 `ZEROS`", so the symbol is real in pi-rewind — but in this plan it is an orphan identifier in the only place it is used. Declare it in Task 2's constants/capture step (e.g. `const ZEROS = "0".repeat(40)`) so Task 3's code references a defined symbol.

14. **Severity:** Nit
    **Affected Scope:** Task 2, Step 3, `createCheckpoint` snippet (around line 363)
    **Discrepancy Description & Rationale:** Comment step 3: "if `lastTree.get(sessionId) === worktreeTree` → …" — the parameter is named `o` and the field is `o.sessionId` (step 4 in the same comment uses `o.sessionId`); bare `sessionId` is not in scope in the shown shape. Cosmetic, but the snippet is presented as the shape the executor extends, and mixing `o.sessionId` and `sessionId` in two adjacent lines reads as two different sources of the id.

15. **Severity:** Nit
    **Affected Scope:** Task 1, Step 1 test snippet (around line 218)
    **Discrepancy Description & Rationale:** `test("constants match the contract values", () => { const c = await import("../src/core/constants.ts"); … })` — a `()`-form callback containing `await`. This is a pre-existing defect in the snippet (not introduced by a revision), but it is in a file the plan designates as the drift guard's "in-repo precursor" and it will not run as written (`await` outside an async function). Make it `async () => { … }`.

---

## Cross-reference and terminology sweep (no further findings at these severities)

Verified and consistent (for the record): all R1–R13 citations point at the right rows (the R1–R5 vs R1–R13 numbering is the two different documents' own numbering and is used correctly in context); `findings.md` R3's 525-stale-refs figure is cited accurately; the `MAX_UNTRACKED_FILE_SIZE` values agree across Global Constraints (10·1024·1024), Task 1 (10·1024·1024), Task 1's test (10·1024·1024), and the Task 11 fence (10485760); the Task 11 fence matches Task 1's `constants.ts` export verbatim; the `happy path` SATISFIES names match the briefing except for the one renamed R7 scenario (Finding 1); the branch regex is identical in all five places it appears; `observeWorktreeRoot` is used consistently after the `verifyStandardRoot` rename (the single residual `verifyStandardRoot` at line 964 is inside the retirement record itself, which is correct); "branch-keyed check/restore" is the consistent current term for the restore check in all non-retirement contexts except the two Threat Matrix cells (Finding 4) and the mermaid node (Finding 6).

No Blocker/Major/Nit found in: Terms section (all five terms match their use), File Structure vs Artifacts list, Alternatives vs briefing (faithful), Build Order edges vs task dependencies, or the OD-1/OD-3 checkpoint note in Task 1.
