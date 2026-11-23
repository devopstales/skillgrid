# Review Ledger — 2026-11-22-pi-worktree-plugin

## Round 1 — 2026-11-22

**Frozen candidate:** base n/a (untracked change folder) → head n/a, 1 file (`blueprint.md`), hash `9c1c33f52b391905824e73186b1172473e145928` (sha256 of blueprint.md at review start; after this round's inline fixes the file is `997a7f91d449a6f543deadb1c2620a9f4c889cec5fd880dcdfcac6e43e39027e` — Stage 6's frozen-candidate re-check dispatches against the post-fix hash)
**Reviewer:** Goldfish Stage 5 — zoom-out (Skeptical Architect), fresh `reviewer` subagent, 4-part isolation contract re-asserted in the dispatch prompt (fresh context + no network harness-enforced; read-only self-enforced; no project instructions via prompt-scoped context)
**Independence grade:** A
**Budget:** round 1/5 (tier T2)

### Findings

| ID | Severity | File:Line | Finding | Disposition | Fix Commit | Lineage |
|---|---|---|---|---|---|---|
| R1-F001 | WARNING | blueprint.md: Task 3 (restore) | OD-2 (worktree-root half of the atomic pair) is a redundant guard — branch identity is the full safety check; drop the `worktreeRoot` param, refusal reason, test, and the OD-2 one-way-door | fixed (this change's working tree; committed with the change folder) | — (pre-commit working tree) | R1-F001 |
| R1-F002a | WARNING | blueprint.md: Task 6 `verifyStandardRoot` | verifying an owner-set worktrunk config value is outside the plugin's trust boundary — demote to a one-line first-use observation (`observeWorktreeRoot`) | fixed | — (pre-commit working tree) | R1-F002a |
| R1-F002b | WARNING | blueprint.md: Task 7 recovery "prefer the default branch's worktree" | re-imports the default-branch assumption the skill system exists to avoid | waived | — (pre-commit working tree) | R1-F002b |
| R1-F003 | WARNING | blueprint.md: Task 8 `detectForeignMarkerPlugin` | filesystem probe of another package's install layout for a cosmetic conflict the README already names — drop the detector, keep the README note + install warning | fixed | — (pre-commit working tree) | R1-F003 |
| R1-F004 | WARNING | blueprint.md: File Structure / Task 9-10 catalog | generated `wt-commands.md` + `make-catalog.ts` + e2e catalog step are weeds — R1's on-demand `--help` pattern gets the live catalog from the binary; keep a hand-written one-page pointer | fixed | — (pre-commit working tree) | R1-F004 |
| R1-F005 | SUGGESTION | blueprint.md: Task 1 (unused tsconfig/typescript/statSync), Task 9 (per-constant prose regexes) | (a) dead scaffold in Task 1's own code; (b) drift guard parsing prose with N regexes breaks on rewording — machine-readable `skillgrid-constants` fence in the contract (Task 11 adds it) | fixed | — (pre-commit working tree) | R1-F005 |

### Dispositions

- **R1-F001** — fixed. OD-2 retired: `restoreCheckpoint(root, id)` is branch-only; refusal reasons now `branch` | `missing`; the cross-clone case is unreachable (the ref can't be read from a different repo). Accepted by the user at the Stage 5 resolve gate (auto accept).
- **R1-F002a** — fixed. `verifyStandardRoot` → `observeWorktreeRoot` (git worktree list, one-line "observed root differs" note when the worktree is not under `<projectroot>/.worktree/`); MH-9/R7 softened to "observed, never silently followed". Accepted at the Stage 5 resolve gate.
- **R1-F002b** — waived. Risk accepted: "main preferred" is the contract's recovery wording (`findings.md` R1, pi-worktrunk's own recovery order); the alternative (arbitrary survivor selection) is worse for the fan-out story. Named human: the user, at the Stage 5 resolve gate (auto accept = keep 2b).
- **R1-F003** — fixed. Detector + hook + test removed; the conflict is a README note + install warning (the briefing's own 🟡 disposition). Accepted at the Stage 5 resolve gate.
- **R1-F004** — fixed. Generator, committed generated artifact, and e2e catalog step removed; `references/wt-commands.md` is now a hand-written one-page pointer to `wt --help`. Accepted at the Stage 5 resolve gate.
- **R1-F005** — fixed. (a) `tsconfig.json`, `typescript` devDep, and the dead `statSync` import removed from Task 1; (b) the drift guard now parses the `skillgrid-constants` fence (added by Task 11) instead of N prose regexes. Accepted at the Stage 5 resolve gate.

### Round Verdict

findings-dispositioned (6 findings: 5 fixed, 1 waived; no CRITICAL; convergence requires 2 consecutive clean rounds — the frozen-candidate re-check runs before Stage 6)

## Round 2 — 2026-11-22

**Frozen candidate:** base n/a → head n/a, 1 file (`blueprint.md`), hash `997a7f91d449a6f543deadb1c2620a9f4c889cec5fd880dcdfcac6e43e39027e` (sha256 of blueprint.md after Round 1's inline fixes; the reviewer dispatched against this hash)
**Reviewer:** Goldfish Stage 6 — comprehension (Naive Reader), fresh `reviewer` subagent, 4-part isolation contract re-asserted in the dispatch prompt (fresh context + no network harness-enforced; read-only self-enforced; no project instructions via prompt-scoped context)
**Independence grade:** A
**Budget:** round 2/5 (tier T2)

### Findings

| ID | Severity | File:Line | Finding | Disposition | Fix Commit | Lineage |
|---|---|---|---|---|---|---|
| R2-F001 | WARNING | blueprint.md: Task 6 Step 1 | test calls `checkBranchAllowListSync` — the declared API is `checkBranchAllowList`; no `Sync` variant exists | fixed (the function is a pure sync regex check — declared sync, test name kept) | — (pre-commit working tree) | R2-F001 |
| R2-F002 | WARNING | blueprint.md: Task 4 Step 1 | prune test fossilizes a mistyped `refs/skillguard/` via `.replace("skillguard","skillgrid")` — contradicts OD-1 | fixed (plain `"refs/skillgrid/"` literal) | — (pre-commit working tree) | R2-F002 |
| R2-F003 | WARNING | blueprint.md: Task 5 Step 3 | shape code calls `restoreCheckpoint(root, toplevelOf(cwd), id)` — 3 args + unknown helper vs the 2-arg signature declared in Task 3 (vestige of retired OD-2) | fixed (2-arg call) | — (pre-commit working tree) | R2-F003 |
| R2-F004 | WARNING | blueprint.md: Task 9 Step 1 | drift-guard default path is an absolute workstation path while the plugin repo has no in-repo contract — ENOENT on any other machine | fixed (`SKILLGRID_CONTRACT_PATH` is required; unset → named failure message) | — (pre-commit working tree) | R2-F004 |
| R2-F005 | WARNING | blueprint.md: Task 11 Step 2 | the `skillgrid-constants` fence was shown nested inside a ` ```markdown ` fence, so as rendered the inner fence terminates the outer one; the column-0 parse requirement was unstated | fixed (tilde fence `~~~skillgrid-constants`, shown as a blockquote, column-0 + line-anchored parse named in three places) | — (pre-commit working tree) | R2-F005 |
| R2-F006 | SUGGESTION | blueprint.md: Global Constraints | "zero runtime deps beyond the Pi packages" reads as if they were package.json deps | fixed (clause added: provided by the host Pi install, peer-only at most) | — (pre-commit working tree) | R2-F006 |
| R2-F007 | SUGGESTION | blueprint.md: Task 3 Step 4 | test-count arithmetic off by one (4+5=9, not 10; confirmed by Task 4's 11) | fixed (9 tests) | — (pre-commit working tree) | R2-F007 |
| R2-F008 | SUGGESTION | blueprint.md: Task 7 Step 1 | recovery test uses undefined fixture identifiers (`mainWorktree`/`agentDemoWorktree`) with no fixture shown | fixed (fixture named in a comment block: temp main checkout + `wt switch --create agent-demo`) | — (pre-commit working tree) | R2-F008 |
| R2-F009 | SUGGESTION | blueprint.md: Task 10 Step 1 | e2e bullet omits the setup for its marker/allow-list assertions (allow-list rejection only fires in a worktree context — ordering unstated) | fixed (setup + explicit step ordering added) | — (pre-commit working tree) | R2-F009 |

### Dispositions

- **R2-F001** — fixed. `checkBranchAllowList` declared synchronous (pure regex, no I/O); the test already called it sync-style, so the declaration now matches.
- **R2-F002** — fixed. The fossilized `.replace` was an authoring artifact; the literal is now plain.
- **R2-F003** — fixed. The 3-arg call was a vestige of the OD-2 retirement; the call now matches Task 3's 2-arg signature.
- **R2-F004** — fixed. The env var is required (the plugin repo has no in-repo contract); unset → a named failure message. The "in-repo path" default claim in Interfaces was removed.
- **R2-F005** — fixed. The fence is now a tilde fence (`~~~skillgrid-constants`) so it can nest inside the blueprint's own backtick fences without terminating them; the drift guard matches the line-anchored opener at column 0 — stated in Task 9's test, Task 11's step, and the File Structure entry.
- **R2-F006** — fixed. Prose now says the Pi packages are provided by the host install and are not listed in package.json.
- **R2-F007** — fixed. 9 tests (Task 2: 4 + Task 3: 5); Task 4's "11" (9+2) was already correct.
- **R2-F008** — fixed. The fixture is named: a temp main checkout with wt initialized + one `agent-demo` worktree via `wt switch --create`.
- **R2-F009** — fixed. e2e now states the setup (wt initialized with the standard template, marker state enabled) and the step ordering (allow-list rejection AFTER the agent-e2e move, because it only fires in a worktree context).

### Round Verdict

findings-dispositioned (9 findings: 9 fixed, no CRITICAL; both paraphrases were coherent — no Blockers; convergence requires 2 consecutive clean rounds — the next clean round is Stage 7)

## Round 3 — 2026-11-22

**Frozen candidate:** base n/a → head n/a, 1 file (`blueprint.md`), hash `997a7f91d449a6f543deadb1c2620a9f4c889cec5fd880dcdfcac6e43e39027e` (post-Round-2 hash; all three reviewers dispatched against this hash — the style reviewer verified the inline copy matches the on-disk file byte-for-byte)
**Reviewer:** Goldfish Stage 7 — critic review, THREE fresh `reviewer` subagents in parallel (design-critic `SeriousHeron`, structural `MajorBadger`, style/consistency `SlipperySole`), findings merged per the stage protocol; 4-part isolation contract re-asserted in each dispatch prompt
**Independence grade:** A
**Budget:** round 3/5 (tier T2)

### Findings

| ID | Severity | File:Line | Finding | Disposition | Fix Commit | Lineage |
|---|---|---|---|---|---|---|
| R3-F001 | CRITICAL | Self-Review item 6 | "Contradictions: none" while the R7 verify→observe revision (user-approved in Stage 5) was never propagated to the reconciliation — briefing says "verifies" + `standard-root-verified`, blueprint says "observes" + renamed scenario | fixed (the revision is now explicitly recorded in Self-Review item 6 as a user-approved requirement-level revision; the acceptance spec uses `standard-root-observed`) | — (pre-commit working tree) | R3-F001 |
| R3-F002 | CRITICAL | Threat Matrix "Untrusted input" test cell | names `loadCheckpointFromRef` — a function that exists nowhere (actual API: `loadCheckpoint`) | fixed (renamed the reference) | — (pre-commit working tree) | R3-F002 |
| R3-F003 | WARNING | Task 4 `pruneOtherSessions` | cross-session prune can delete a concurrently-running sibling session's refs (5-worktree fan-out shares `refs/skillgrid/`; no liveness check) | fixed (liveness-aware: prune only refs older than `PRUNE_MIN_AGE_MS` = 1h; `now` injectable; two-session test added; MH-5 + Validation R4 updated) — user chose option (a) age-threshold at the Stage 7 gate | — (pre-commit working tree) | R3-F003 |
| R3-F004 | WARNING | Task 3 code vs Threat Matrix "Commit state" | detached-HEAD refusal string drifts (`checkpoint on 'X', current 'DETACHED'` vs `"branch mismatch: (detached)"`) | fixed (threat matrix references the Task 3 string; one source) | — (pre-commit working tree) | R3-F004 |
| R3-F005 | WARNING | Threat Matrix "Git repository selection" row | dead OD-2 terminology — a toplevel "pair check" no task builds or tests | fixed (row rewritten to the surviving design: branch-keyed refusal; `repoRoot` resolves cwd→toplevel; test = subdirectory resolution + cross-worktree branch refusal) — confirmed by both the design critic (N-2) and the style critic (Major 4) | — (pre-commit working tree) | R3-F005 |
| R3-F006 | WARNING | Global Constraints vs Task 1 package.json | "Pi floor in `package.json` `engines`" but the snippet has no `engines` field | fixed (`"engines": {"pi": ">=0.99.1"}` added to Task 1's snippet + Artifacts + File Structure lines) | — (pre-commit working tree) | R3-F006 |
| R3-F007 | SUGGESTION | Task 2/5 `lastTree` dedup | not seeded on resume — a resumed session's first unchanged turn emits a duplicate ref | fixed (`rebuildCache` seeds `lastTree` from the session's newest ref — stated in Task 5's session_start) | — (pre-commit working tree) | R3-F007 |
| R3-F008 | SUGGESTION | mermaid T3 node | still says "pair check" after OD-2 retirement | fixed (renamed to "branch-keyed check") | — (pre-commit working tree) | R3-F008 |
| R3-F009 | SUGGESTION | Task 3/4 test groups | `resetSessionState()` missing from tests that depend on the module-level `lastTree` (it persists across tests in the same file) | fixed (reset added to all 7 affected tests) | — (pre-commit working tree) | R3-F009 |
| R3-F010 | SUGGESTION | Validation R13 row | "path fallback error" names an undefined behavior | fixed ("unreadable path → hard failure, not silent pass") | — (pre-commit working tree) | R3-F010 |
| R3-F011 | SUGGESTION | Task 8 Step 3 | `wt <marker-clear>` placeholder in a user-visible string (findings.md R1 names the exact command) | fixed (`wt config state marker clear`) | — (pre-commit working tree) | R3-F011 |
| R3-F012 | SUGGESTION | Global Constraints constants bullet | "13 upstream dir names" enumerates 12 | fixed (12 upstream + `.worktree` = 13 total, made explicit) | — (pre-commit working tree) | R3-F012 |
| R3-F013 | SUGGESTION | Task 7 Step 3 | approval sentence ended mid-ellipsis | fixed (closed: retry once, re-show on decline, non-interactive fails with list; cites MH-12 + findings R1) | — (pre-commit working tree) | R3-F013 |
| R3-F014 | SUGGESTION | Task 5 Step 3 | `turnIndex` used but never declared | fixed (declared + reset in session_start, incremented per turn_end) | — (pre-commit working tree) | R3-F014 |
| R3-F015 | SUGGESTION | Task 3 Step 3 | `ZEROS` used but never declared | fixed (declared in Task 2: `const ZEROS = "0".repeat(40)`) | — (pre-commit working tree) | R3-F015 |
| R3-F016 | SUGGESTION | Task 2 Step 3 comment | `o.sessionId` vs bare `sessionId` in adjacent lines | fixed (consistent `o.sessionId`) | — (pre-commit working tree) | R3-F016 |
| R3-F017 | SUGGESTION | Task 1 Step 1 test | `await` in a non-async `test()` callback | fixed (`async () => { … }`) | — (pre-commit working tree) | R3-F017 |
| R3-F018 | SUGGESTION | Self-Review item 6 mapping `R4→T4` | internally consistent but ambiguous against the header's findings.md R4 pointer (two different R-numbering schemes) | fixed (one-word disambiguation — the mapping is the briefing's R-numbering; stated) | — (pre-commit working tree) | R3-F018 |

### Dispositions

- **R3-F001** — fixed (CRITICAL). The "no contradictions" claim was true only because the Stage 5 R7 revision was treated as a fix, not a requirement-level change. Self-Review item 6 now records it explicitly: briefing R7's "verifies" + `standard-root-verified` is superseded by "observes" + `standard-root-observed` (user-approved at the Stage 5 gate); the acceptance spec uses the renamed scenario.
- **R3-F002** — fixed (CRITICAL). Phantom function renamed to the real `loadCheckpoint` API.
- **R3-F003** — fixed (WARNING, user-chosen option (a) age-threshold). `pruneOtherSessions` now deletes only refs older than 1h (`PRUNE_MIN_AGE_MS`); `now` is injectable; a two-session test proves a fresh sibling ref survives while a dead session's ref is pruned. MH-5 and Validation R4 updated.
- **R3-F004** — fixed. The threat matrix now references the Task 3 string; one source of truth.
- **R3-F005** — fixed. The "Git repository selection" row no longer describes the retired pair check; it describes the branch-keyed refusal + `repoRoot` resolution that the plan actually builds.
- **R3-F006** — fixed. `engines` added to the only package.json the plan specifies.
- **R3-F007** — fixed. `rebuildCache` seeds `lastTree` from the session's newest ref at session_start (the resume case).
- **R3-F008..R3-F017** — fixed. All mechanical (mermaid label, test resets, R13 wording, marker command, dir count, approval sentence, turnIndex, ZEROS, sessionId consistency, async callback).
- **R3-F018** — fixed. The mapping disambiguated: it is the briefing's R1–R13 numbering (not findings.md's R1–R5).

### Round Verdict

findings-dispositioned (18 findings: 18 fixed, 2 CRITICAL + 4 WARNING + 12 SUGGESTION; convergence requires 2 consecutive clean rounds — the next clean round is Stage 8)

## Round 4 — 2026-11-22

**Frozen candidate:** base n/a → head n/a, 1 file (`blueprint.md`), hash `f8c7886c3966fe3450d98f807eaf479767ffaaf31df078fbca2572d20b75a566` (sha256 after Round 3's inline fixes; the reviewer dispatched against this hash)
**Reviewer:** Goldfish Stage 8 — intermediate zoom-out (Skeptical Architect, FRESH — not the Stage 5 reviewer), specific mandate: catch altitude drift + scope creep INTRODUCED by the Stage 5/7 fixes
**Independence grade:** A
**Budget:** round 4/5 (tier T2)

### Findings

| ID | Severity | File:Line | Finding | Disposition | Fix Commit | Lineage |
|---|---|---|---|---|---|---|
| R4-F001 | WARNING | Task 4 `PRUNE_MIN_AGE_MS` + liveness prune | the liveness-aware prune (Stage 7 R3-F003) rests on siblings sharing `refs/skillgrid/`, but the contract says "per-worktree by construction, each worktree has its own ref namespace" — under the stated architecture the hazard is unreachable; the 1h threshold's own rationale is also weak (a session restarted after >1h is still live in Pi) | fixed (user chose option (a): the 1h constant is now documented as a DEFENSIVE second line of defense — the contract's per-worktree sentence is the primary boundary; git refs are repo-shared in practice, so the threshold guards the fan-out case) | — (pre-commit working tree) | R4-F001 |
| R4-F002 | SUGGESTION | Task 4 Step 4 | stale test count: "PASS (12 tests)" after the liveness test was added (second stale count in two rounds — R2-F007 class) | fixed (13 tests total: capture 4 + restore 5 + prune 4) | — (pre-commit working tree) | R4-F002 |
| R4-F003 | SUGGESTION | threat matrix + Task 5 | `repoRoot(ctx.cwd)` is now a load-bearing named component but declared by no task's Produces — the phantom-symbol class R3-F002 just caught, one layer down | fixed (declared in Task 2's Produces: `repoRoot(cwd: string): Promise<string>` — the contract's existing `--show-toplevel` snippet) | — (pre-commit working tree) | R4-F003 |
| R4-F004 | SUGGESTION | Task 1 code vs Task 1 test vs Task 11 fence | `IGNORED_DIRS` is a `Set` in the code but compared with ordered `toEqual` in the test and recorded as an ordered array in the fence — set-vs-order unstated | fixed (Task 9's drift-guard comment: the fence lists an array, the plugin exports a Set, compare as sorted arrays) | — (pre-commit working tree) | R4-F004 |

### Dispositions

- **R4-F001** — fixed (user chose option (a) at the Stage 8 gate: keep the 1h threshold). The constant's comment now states the contract's primary boundary ("per-worktree by construction") and names the threshold as a defensive second line for the fan-out case (git refs are repo-shared in practice). The contract's "per-worktree by construction" sentence stays as the primary boundary — the plugin implements against it.
- **R4-F002** — fixed. 13 tests (the second stale count in two rounds — the executor's `bun test` would catch it, but the plan's "expected" line should be right).
- **R4-F003** — fixed. `repoRoot` declared in Task 2's Produces with its contract provenance.
- **R4-F004** — fixed. The set-vs-order comparison strategy is now stated where it belongs (Task 9's drift-guard test).

### Round Verdict

findings-dispositioned (4 findings: 4 fixed, 1 WARNING + 3 SUGGESTION; **no scope creep** — the reviewer's explicit finding: "every device added by the fixes traces to the briefing's R1–R13 and changes no architecture/interface/dependency set; the Stage 5 fixes net out as complexity *removed*"; the Stage 8 review is clean at WARNING+ for altitude/scope)

## Round 5 — 2026-11-22

**Frozen candidate:** base n/a → head n/a, 1 file (`blueprint.md`), hash `f8c7886c3966fe3450d98f807eaf479767ffaaf31df078fbca2572d20b75a566` (post-Round-4 hash; all three reviewers dispatched against this hash)
**Reviewer:** Goldfish Stage 9 — implementation readiness, THREE fresh `reviewer` subagents in parallel (implementation-readiness `ProgressiveEarthworm`, codebase-details `MarineCatfish`, style/consistency `TerribleKingfisher` — second pass), findings merged per the stage protocol; **pass bar: the Zero-Decision Rule**
**Independence grade:** A
**Budget:** round 5/5 (tier T2) — the cap; the Stage 9 FAIL is resolved by pinning, and the re-verification is the User Review Gate's job (not a 6th round)

### Findings

| ID | Severity | File:Line | Finding | Disposition | Fix Commit | Lineage |
|---|---|---|---|---|---|---|
| R5-F001 | CRITICAL | Task 2 Produces / threat matrix | `repoRoot` mis-attributed to "the contract's existing `--show-toplevel` snippet" — no such snippet; a plain `--show-toplevel` from the primary checkout resolves the primary root, not the worktree root | fixed (pinned the exact command `git -C <cwd> rev-parse --show-toplevel` — the `-C` makes cwd authoritative; corrected in Task 2, File Structure, threat matrix) | — (pre-commit working tree) | R5-F001 |
| R5-F002 | WARNING | Task 4 prune test | second prune test expects `n===1` for a FRESH (<1h) other-session ref that the liveness-aware code keeps | fixed (the test now uses the injectable `now` — advances the clock past the threshold, deterministic) | — (pre-commit working tree) | R5-F002 |
| R5-F003 | WARNING | MH-1 / Task 2 test / contract | metadata test asserts `untracked [` but the contract says `preexisting-untracked` (three variants) | fixed (one field name — `preexisting-untracked` — in MH-1, the test, and the metadata message format) | — (pre-commit working tree) | R5-F003 |
| R5-F004 | WARNING | Task 2 capture comment | the .gitignore'd-paths half of the snapshot filter is missing from the capture comment and all test groups | fixed (the filter is `--porcelain` (excludes ignored) + `IGNORED_DIRS` + size/count caps; pinned in Task 2 Step 3 + a test case: an ignored-but-present file is absent from the captured tree) | — (pre-commit working tree) | R5-F004 |
| R5-F005..F016 | WARNING | Tasks 5/7/8/9/10 | 12 Zero-Decision Rule failures: `ui.ts` shape; `wt.ts` tool-result/error strategy; approval detection; "default branch" rule; `recordWorktrees` storage; `safeClean` file-listing command; linked-session creation + stub-`wt` test; fence grammar + `evals.json` format + scenario content; marker SET argv; capture add-filter mechanics; `lastTree` cross-test lifecycle; `e2e.sh` driver + `npm test` vs `bun test` | fixed (each closed with a pinned sentence, signature block, or snippet in the named task — the open choices are now pre-made) | — (pre-commit working tree) | R5-F005..F016 |
| R5-F017 | SUGGESTION | Global Constraints | "git fallback applies (skill Step 0c)" cites the wrong sub-step (it's Tier 3 of Step 0b) | fixed (citation corrected) | — (pre-commit working tree) | R5-F017 |
| R5-F018 | SUGGESTION | Task 5 shape code | async `repoRoot()` called without `await` at all four call sites | fixed (`await` added) | — (pre-commit working tree) | R5-F018 |
| R5-F019 | SUGGESTION | Task 4 comment | `PRUNE_MIN_AGE_MS` comment attributes "Stage 8 M-1"; the decision was Stage 7's (R3-F003) | fixed ("Stage 7 R3-F003 decision; Stage 8 R4-F001 documentation") | — (pre-commit working tree) | R5-F019 |
| R5-F020 | SUGGESTION | Self-Review | "mutex for codemode queued calls" — the queued-call scenario is Task 7's tool wiring | fixed ("the mutex in `runDirectedWt` (Task 6, defense-in-depth for the Task 7 tool's codemode queued calls)") | — (pre-commit working tree) | R5-F020 |
| R5-F021 | SUGGESTION | Task 11 | fence closing line unpinned | fixed (exact shape: opener `~~~skillgrid-constants` at column 0, one `KEY: value` per line, closer `~~~` at column 0, one blank line after; the drift guard matches `^~~~skillgrid-constants$` … `^~~~$`) | — (pre-commit working tree) | R5-F021 |
| R5-F022 | SUGGESTION | Task 1 package.json | missing `"files"` field | fixed (`"files": ["src/", "references/"]`) | — (pre-commit working tree) | R5-F022 |

### Dispositions

- **R5-F001** — fixed (CRITICAL). The `repoRoot` command is now pinned: `git -C <cwd> rev-parse --show-toplevel`. The `-C` makes cwd authoritative, so from a worktree's subdirectory it returns the WORKTREE root. Corrected in three places (Task 2 Produces, File Structure, threat matrix).
- **R5-F002** — fixed. The prune test uses the injectable `now` (deterministic; the liveness test already does this).
- **R5-F003** — fixed. One field name: `preexisting-untracked` (the contract's).
- **R5-F004** — fixed. The filter mechanics are pinned (porcelain excludes ignored; `IGNORED_DIRS` is belt-and-suspenders; a test case proves an ignored file is absent).
- **R5-F005..F016** — fixed. All 12 Zero-Decision Rule failures closed with pinned shapes: `ui.ts` (F1), `wt.ts` tool-result shapes (F2), approval detection via exit code 3 + `[approval]` marker (F3), default-branch rule via `git symbolic-ref` (F4, 4 places), `recordWorktrees` storage (F5), `safeClean` via `git diff --name-only --diff-filter=AD` (F6), linked-session mechanism + stub-`wt` PATH shim (F7), fence grammar + `evals.json` format + scenario content (F8), marker SET argv (F9), capture add-filter mechanics (F10), `resetSessionState` moved to Task 2's Produces (F11), `e2e.sh` driver + `npm test` = `bun test` (F12).
- **R5-F017..F022** — fixed. All mechanical (citation, awaits, attribution, fence shape, files field).

### Round Verdict

findings-dispositioned (22 findings: 22 fixed, 1 CRITICAL + 12 WARNING + 9 SUGGESTION; **Stage 9 verdict: FAIL against the Zero-Decision Rule** — 12 Major open choices named per the rule, all now pinned. The scratch core (Tasks 1–4) was "close to first-attempt executable"; the wt side + wiring layer are now pinned. **Re-verification is the User Review Gate's job** (the 5-round cap is reached; a fresh Stage 9 pass would be a 6th round — the gate's explicit OD-1/OD-3 approval + a spot-check of the pinned shapes substitutes). The style reviewer (second pass) confirmed: "clean at Blocker/Major — no new inconsistencies introduced by the Stage 7/8 fixes; all 15 first-pass fixes verified landed.")

## Round 6 — 2026-11-23

**Frozen candidate:** plugin repo `.worktree/pi-worktree-wf_t5`, base `f548ef8` (the 11-ticket build) → review fixes committed `9376e19` (this round's fix pass)
**Reviewer:** the Phase 4 adversarial audit — FIVE fresh `reviewer` subagents in parallel (Standards `StdLens`, Spec `SpecLens`, Edge `EdgeLens`, Verification `VerifyLens`, Security `SecLens`); **pass bar: end-user consequence to the BUILT plugin, not the blueprint**
**Independence grade:** A (each lens read the tree from disk; no shared memory; the coordinator re-grade is a separate actor)
**Budget:** post-goldfish code review (round 6) — rounds 1–5 (the goldfish) reviewed the PLAN; this round reviews the BUILT plugin. The red-team lens was folded into the coordinator's re-grade (not separately dispatched — stated for honesty).

### Findings

| ID | Severity | File:Line | Finding | Disposition | Fix Commit | Lineage |
|---|---|---|---|---|---|---|
| R6-A | WARNING | src/wt.ts:222,292,326 | `applyMergeDefaults`/`checkBranchAllowList`/`observeWorktreeRoot` exported + unit-tested but **never called** on the extension path (R7/R8/R9 not enforced at runtime) | ESCALATE (wire into `runWorktrunk` vs amend-spec) — programmer decision at the qa-verdict gate | — | R6-A |
| R6-E | WARNING | src/wt.ts:154,186,83-85 | approval passthrough broken in the piped runtime; both R10 gates vacuous (0 tests named "approval") | ESCALATE (wt-CLI/approval fix vs document as a limitation) — programmer decision at the qa-verdict gate | — | R6-E |
| R6-B1 | WARNING | src/core/scratch-core.ts:176,341 | C-quoted (non-ASCII) path → `JSON.parse` throws → the turn fails | fixed (`core.quotePath=false` + `decodePath`) | 9376e19 | R6-B1 |
| R6-B2 | WARNING | src/core/scratch-core.ts:338 | rename `R old -> new` → pathspec `old` no longer exists → fatal | fixed (`decodePath` keeps `new`) | 9376e19 | R6-B2 |
| R6-B4 | WARNING | src/core/scratch-core.ts:156-158 | unborn HEAD → branch `"HEAD"` → restore can't match | fixed (`git symbolic-ref` real branch) | 9376e19 | R6-B4 |
| R6-D | WARNING | src/scratch.ts:58 | fresh/resumed session re-captures an unchanged first turn (baseline starts `""`) | fixed (`seedDedup` at `session_start`) + new resume test | 9376e19 | R6-D |
| R6-H1 | SUGGESTION | src/core/scratch-core.ts:444-445 | before-restore outcome swallowed (empty catch) + message says "taken" when skipped | fixed (`beforeRestore` field) | 9376e19 | R6-H1 |
| R6-G1 | SUGGESTION | tests/scratch.test.ts | `-t "resume"` gate matched 0 tests | fixed (new resume-dedup test, 0→1 match) | 9376e19 | R6-G1 |
| R6-G10 | SUGGESTION | tests/wt-core.test.ts:284+ | recovery tests flake at 5s under load (real-wt fixtures) | fixed (`{timeout:30000}` on the 3 recovery tests) | 9376e19 | R6-G10 |
| R6-K3 | SUGGESTION | src/core/scratch-core.ts:236-244 | `idOk` dead re-check (re-runs `refOk` on a validated value) | fixed (removed) | 9376e19 | R6-K3 |
| R6-B3 | WARNING | src/core/scratch-core.ts:209 | index file bytes differ when staged (cache-tree extension persisted by `write-tree`); R1 "byte-identical" literal violated (benign) | DEFER (DW) | — | R6-B3 |
| R6-C | WARNING | src/core/scratch-core.ts:337-361 | safeClean staged-new file → `AD` state after restore | DEFER (DW) | — | R6-C |
| R6-F | WARNING | src/core/wt-core.ts:457-491 | recovery can land in a foreign recreated repo (survivor's live common-dir not re-checked) | DEFER (DW) | — | R6-F |
| R6-G2..G7 | SUGGESTION | tests/ | verification gaps (validateDirective not asserted; no-replay on a constant; prune identity not asserted; e2e move false-positive window) | DEFER (DW) | — | R6-G |
| R6-J1..J4 | SUGGESTION | src/ | robustness (swallowed wt spawn err; `git()` inherits `GIT_*`; `timeoutMs` never set; `loadCheckpoint` no 40-hex check) | DEFER (DW) | — | R6-J |
| R6-K1..K8 | SUGGESTION | src/ | cosmetic (ui.ts comment; `findCheckpoint` hardcode; camelCase keys; etc.) | DEFER / NOISE (DW) | — | R6-K |

### Dispositions

- **R6-A / R6-E — ESCALATED** to the programmer at the qa-verdict gate (autonomy: check-in). **A:** the three core invariants are correct + unit-tested but not called on the runtime path, so `wt merge main` runs with wt's default, `wt switch --create feature-x` in a worktree is accepted, and foreign roots are never noted. Decision: wire them into `runWorktrunk` (makes R7/R8/R9 hold) vs amend `acceptance.feature` to record the deferral (core provides, wt governs). **E:** wt is spawned piped/non-interactive, so the dialog-approval store path fails and the `[approval]` detection misses the piped exit-0 case; both R10 gates are vacuous. Decision: fix the approval flow vs document it as a wt-runtime limitation.
- **11 findings fixed this pass** (B1, B2, B4, D, H1, G1, G10, K3, K9) with covering tests, verified: `bun test` 44/44 (was 43, +1 resume test), `e2e` 6/6, drift guard green. All core fixes proven empirically (non-ascii→created, rename→created, resume→skipped, unborn→`main`, beforeRestore→`skipped`).
- **~20 findings deferred** to `.skillgrid/deferred-work.md` (DW-*) — edge cases (B3, C, F), test-quality gaps (G2–G7), robustness (J1–J4), cosmetic (K1–K8).

### Round Verdict

findings-dispositioned (26 findings: 11 fixed, 2 ESCALATED, 13 deferred; **no CRITICAL after re-grade by end-user consequence** — no data loss, git-state corruption, or security break; the two most severe, A and E, are spec-completeness gaps, not correctness breaks). **Review signal at the gate: BACK-TO-APPLY — Critical 0 · Warning 5 (A, E open; B3, C, F deferred) · Suggestion ~20 (deferred).**

**Resolution (Phase 5, `f1b58b8`):** at the qa-verdict gate the programmer chose **A=wire** + **E=fix** + **bench=accept**. Both escalated items are now fixed — R7/R8/R9 wired into `runWorktrunk` (`applyMergeDefaults` + `checkBranchAllowList` + `observeWorktreeRoot` via `isWorktreeContext`), R10 approval fixed (`probeUnapprovedHooks` detection + `storeApprovalsPty` PTY store + core `handleWtApproval`, +2 approval gates). B3/C/F re-graded SUGGESTION (benign edge cases). **Final: PASS — Critical 0 · Warning 0 open · Suggestion ~20 deferred.** `bun test` 48/48 (was 44; +4), `e2e` 6/6, drift guard green.
