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

findings-dispositioned (4 findings: 4 fixed, 1 WARNING + 3 SUGGESTION; **no scope creep** — the reviewer's explicit finding: "every device added by the fixes traces to the briefing's R1–R13 and changes no architecture/interface/dependency set; the Stage 5 fixes net out as complexity *removed*"; the Stage 8 review is a clean round at WARNING+ for altitude/scope — convergence (2 consecutive clean rounds) is tracked at the stage level: Stage 7 was findings-dispositioned, Stage 8 is clean → Stage 9 must also be clean to converge)
