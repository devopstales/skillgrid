# Parallel Decomposition (Cleave Port) — Implementation Blueprint (steps, files, verification)

> **Role:** this is the *how* — the implementation plan: concrete steps,
> files to touch, code, tests, and verification. The companion `briefing.md` is
> the *what* (requirements & intent). Do not rename either file.
> Assume the engineer has zero context for the codebase. DRY. YAGNI. TDD.
> Frequent commits. No placeholders.

> **For agentic workers:** REQUIRED SUB-SKILL: Use subagent-execution
> (recommended) or simple-execution to implement this plan task-by-task. Steps
> use checkbox (`- [ ]`) syntax for tracking.

**Status:** PROPOSED

**Tier:** T2

**Classification:** standard (T2) — verification floor L2 (full suite + build).
New scripts have their own focused tests (L1) inside the L2 gate.

**Build shape:** Smallest usable whole. The first task (`compute-waves` +
fixture test) is the thinnest increment that makes a parallel run deterministic
and is independently usable and testable; the next two land as separate usable
increments. No cross-task integration is needed — the three guardrails are
independent, so the three script/prose tasks form one parallel wave and the
ADR/manifest task follows.

**Goal:** Port Cleave's three deterministic parallel-run guardrails (wave
ordering, dirty-tree classification, review churn breaker) into skillgrid as
dependency-free JS scripts + skill prose, with RED→GREEN fixture tests — no
new Go code, no new npm dependency, no runtime, no SDD-ledger change.

**Architecture:** Three leaf guardrails, each a dependency-free Node script
under the owning skill's `scripts/` dir plus one line of skill prose that
invokes it and says what to do with the output. `compute-waves` is a pure
function over a `Needs:` graph (Kahn topological sort + cycle break) invoked by
`concurrent-leaves.md`. `classify-dirty` reads `git status --porcelain` + the
change's `Owns:` globs and buckets each path (related/volatile/unrelated) with a
suggested checkpoint message, invoked by `isolated-workspace`. The churn
breaker is a named rule (stable issue key + 0.5 reappear threshold +
stop-park-escalate) in `concurrent-leaves.md`, referenced from
`subagent-execution`'s fix loop. The SDD ledger remains the execution record
(ADR-0020); the scripts emit stable, machine-parseable lines the skill prose
and the tests both read.

**Tech Stack:** Node.js (stdlib only — `child_process`, `fs`, `path`, `process`;
no `yaml`, no third-party import, so the scripts run without a `node_modules`).
git (invoked as a CLI by `classify-dirty`). Skill content: Markdown. No Go, no
`package.json` change, no `go.mod` change.

**Spec:** `./briefing.md` (the what). Decisions respected (in-force, from
`.skillgrid/ASSUMPTIONS.md` § LOCKED): ADR-0020 (ledger is the execution
record — scripts are inputs to it), ADR-0016 (skills content zone), ADR-0003
(single Go binary — scripts run at plan/execution time, not in the binary).
No new ADR for this change (per the briefing's Open Questions; promote to a
candidate ADR row only if the review deems the scripts cross-change infra).

## Terms

- **wave** — a set of leaves whose `Needs:` are all satisfied by earlier waves;
  the unit of parallel dispatch. (Relies on the existing `Needs:` term.)
- **churn** — the reappear-fraction of review issues across two fix rounds; the
  convergence signal the breaker trips on. (Sharpened this plan.)
- **auto-resolvable tree** — a dirty tree whose only uncommitted paths are
  volatile; `classify-dirty` flags it so the operator need not prompt.
- **stable issue key** — `file + symbol + issue-type`, the identity used to
  detect that an issue *reappeared* (the same issue) vs. a new one. (Sharpened
  this plan.)
- Canonical terms live in `.skillgrid/artifacts/` (`01-business-terms.md` /
  `02-technical-terms.md`); nothing here redefines them.

## Must-Haves (goal-backward verification)

**Truths** (observable behaviors that must hold):

- Running `scripts/compute-waves` on a 4-task graph with a single `c needs b`
  edge prints exactly two waves, with `c` in the later wave and `a`/`b` in
  wave 0; exit code 0. (R1)
- Running `scripts/compute-waves` on a 2-task cycle prints both tasks (broken
  into the same wave) and a `CYCLE:` line naming them; exit code 0 (a broken
  cycle is a recoverable input, not a crash). (R1)
- Running `scripts/compute-waves` on a task with a self-`Needs:` or a `Needs:`
  reference to an unknown id drops that edge, reports it on a `DROPPED:` line,
  and still orders the rest; exit code 0. (R1)
- Running `scripts/compute-waves` on an empty task list prints `WAVES: 0` (or
  an empty wave list) without error; exit code 0. (R1)
- Running `scripts/compute-waves` on unparseable input prints a `PARSE:` error
  line and exits non-zero (2). (R1)
- Running `scripts/classify-dirty` in a fixture repo with 1 related, 1 volatile,
  1 unrelated dirty file prints the three paths under `RELATED:`, `VOLATILE:`,
  `UNRELATED/UNKNOWN:` respectively, a non-empty `CHECKPOINT:` message covering
  the related set, and `AUTO_RESOLVABLE: false`. (R2)
- Running `scripts/classify-dirty` in a fixture repo whose only dirty path is a
  volatile file prints `AUTO_RESOLVABLE: true` and an empty `RELATED:` set. (R2)
- The `concurrent-leaves.md` **Churn breaker** section names the stable issue
  key (`file + symbol + issue-type`), the 0.5 default threshold, and the
  stop-park-escalate action (park still-open issues in the ledger with a
  `Ruling:` and do not start the next round); the breaker fires iff the
  reappear fraction of round R over R−1 is ≥ 0.5. (R3)
- `subagent-execution`'s task-loop references the churn breaker so the per-task
  fix loop inherits it. (R3)
- `grep` for `compute-waves` in `concurrent-leaves.md`, `classify-dirty` in
  `isolated-workspace/SKILL.md`, and `churn` in both `concurrent-leaves.md` and
  `subagent-execution/` each returns ≥1 hit carrying a run-when + use-output
  instruction (not just a bare name). (R4)

**Artifacts** (files that must exist with real implementation, not stubs):

- `.agents/skills/execution/parallel-execution/scripts/compute-waves` — Node
  script, stdlib only, parses `tasks.md` `Needs:` edges, prints ordered waves +
  `CYCLE:`/`DROPPED:` lines.
- `.agents/skills/execution/parallel-execution/tests/compute-waves.test.mjs` —
  fixture-based test (subprocess) covering linear, diamond, cycle, self/unknown
  edge, empty, unparseable.
- `.agents/skills/execution/isolated-workspace/scripts/classify-dirty` — Node
  script, stdlib only, reads `git status --porcelain` + `Owns:` globs, prints
  buckets + `CHECKPOINT:` + `AUTO_RESOLVABLE:`.
- `.agents/skills/execution/isolated-workspace/tests/classify-dirty.test.mjs` —
  fixture-repo test (creates a temp git repo) covering mixed, volatile-only,
  all-related, no-git.
- `concurrent-leaves.md` — wave step naming `compute-waves` + a **Churn
  breaker** section.
- `isolated-workspace/SKILL.md` — preflight step naming `classify-dirty`.
- `subagent-execution/` — a reference line from its task loop to the churn
  breaker.
- `adr.md` (change manifest) — pointers to ADR-0020/0016/0003 as respected,
  and an explicit "no new ADR" note with the promote-to-ADR condition.

**Key links** (critical connections that must work together):

- `compute-waves` must read the *same* `Task <N>: Needs: <ids>` line format that
  `concurrent-leaves.md` and `subagent-execution` already tell the agent to
  write — a format mismatch means the script silently drops every edge.
- `classify-dirty`'s `Owns:` glob input must match the `Task <N>: Owns: <globs>`
  line format already written by `concurrent-leaves.md`.
- The skill prose in `concurrent-leaves.md` must invoke `compute-waves` *before*
  the first dispatch and instruct the agent to dispatch only the current wave —
  the script's output is the dispatch order, not advice.
- The churn-breaker rule must compose with `subagent-execution`'s existing
  round cap (5): the breaker is a *convergence* stop that can fire *before* the
  cap; the cap remains the *hard* stop. The two are different signals and the
  prose must not conflate them.
- Every script's printed line format is the contract the test parses and the
  skill prose relies on — changing a line's prefix (`WAVES:`, `CYCLE:`,
  `RELATED:`, `AUTO_RESOLVABLE:`) without updating the test and prose is a
  contract break.

**One-way-door decisions** (hard to reverse — flag for explicit user approval
before implementing):

- **None.** No migration, no Go command, no published MCP/HTTP contract change,
  no data model, no new dependency. The scripts are additive content; deleting
  them returns the skills to today's prose-only behavior. (The stable output
  line format is a *soft* contract within this change — it is versioned in-repo
  with the scripts and is not a cross-repo published API.)

## Global Constraints

- **Dependency-free scripts.** Node stdlib only. No `yaml`, no third-party
  import. AGENTS.md § Rules: "No new dependencies without an ADR" (ADR-0008
  precedent) — satisfied by adding none.
- **No Go, no `package.json`/`go.mod` change.** Scripts run at plan/execution
  time as the repo's established JS pipeline-helper pattern
  (`scripts/*.mjs`); ADR-0003's single-binary invariant is untouched.
- **No SDD-ledger change.** ADR-0020: the scripts are inputs to
  `.skillgrid/sdd/<plan>/progress.md` / `parallel-ledger.md`, never a second
  execution record. A script must not persist a per-run state file.
- **Spec zone before code zone.** The `pre-commit` zone guard blocks a commit
  that mixes spec/`tasks.md` changes with code. Commit `briefing.md` +
  `blueprint.md` (spec zone) first; the scripts + prose (code zone) in a later
  commit.
- **Conventional commits, no AI attribution.** (AGENTS.md § Rules.)
- **Serial development** — one change at a time.
- **Tests must be real RED→GREEN:** a failing test run before the
  implementation, a passing run after, with the command + exit in the task.

## File Structure

- `.agents/skills/execution/parallel-execution/scripts/compute-waves`
  — pure graph→waves CLI (parse `Needs:`, Kahn order, cycle break). One
  responsibility: given a task graph, print dispatch order.
- `.agents/skills/execution/parallel-execution/tests/compute-waves.test.mjs`
  — subprocess test over fixture task lists.
- `.agents/skills/execution/isolated-workspace/scripts/classify-dirty`
  — `git status` → related/volatile/unrelated buckets + checkpoint message. One
  responsibility: classify a dirty tree against `Owns:` globs.
- `.agents/skills/execution/isolated-workspace/tests/classify-dirty.test.mjs`
  — temp-git-repo test over fixture dirty trees.
- `.agents/skills/execution/parallel-execution/references/concurrent-leaves.md`
  — modify: add the wave-computation step (name `compute-waves`) + a **Churn
  breaker** section.
- `.agents/skills/execution/isolated-workspace/SKILL.md`
  — modify: add the preflight-classify step (name `classify-dirty`).
- `.agents/skills/execution/subagent-execution/SKILL.md` (or its
  `references/task-loop.md` if the fix loop is described there — the implementer
  must locate the actual fix-loop text and reference the breaker at that spot)
  — modify: one reference line to the churn breaker.
- `.skillgrid/specs/2026-10-20-parallel-cleave-port/adr.md`
  — create: the change's ADR manifest.

---

### Task 1: `compute-waves` script + test (the door check)

> **Checkpoint:** this is the door check. If `tasks.md` `Needs:` lines cannot
> be parsed reliably from the real `tasks.md` the pipeline produces, STOP — the
> premise (parse `tasks.md` directly) is wrong and the briefing's Open
> Question must be resolved (switch to a JSON input the agent extracts) before
> Tasks 2–5 proceed.

**Files:**
- Create: `.agents/skills/execution/parallel-execution/scripts/compute-waves`
- Test: `.agents/skills/execution/parallel-execution/tests/compute-waves.test.mjs`
- Read (format source, do not edit): an existing `tasks.md` that contains
  `Task <N>: Needs: <ids>` lines (e.g.
  `.skillgrid/specs/2026-10-06-second-brain-learnings/tasks.md`) to lock the
  exact line grammar the parser must accept.

**Interfaces:**
- Consumes: none (self-contained).
- Produces: a CLI `compute-waves <tasks-file>` (or stdin) whose stdout is a
  stable line format:
  - `WAVES: <count>`
  - `WAVE 0: <id> <id> ...` (one line per wave, in order)
  - `DROPPED: <id> needs <missing|self> <reason>` (per pruned edge)
  - `CYCLE: <id> <id> ...` (ids whose edges were cleared to break a cycle)
  - `PARSE: <reason>` + exit 2 on unparseable input
- Seam: the script's `parseNeeds(lines) -> {tasks, edges}` function — the
  place the `tasks.md` grammar is interpreted. Behavior can change there without
  touching the wave algorithm.
- Deletion test: if deleted, the agent re-derives wave order by hand every run
  (the gap this task closes); callers are the `concurrent-leaves.md` prose
  (Task 4) and its test.
- Adapters: 2 (production `tasks.md` parsing + the test's synthetic graphs) —
  the seam is justified.

**SATISFIES:** `compute-waves-orders-independent-and-dependent`

- [ ] **Step 1: Write the failing test.** Fixture cases (write each as a temp
  `tasks.md` the test feeds the script via its arg or stdin):
  - linear + diamond → expect exact wave lines
  - 2-cycle → expect both ids on one `CYCLE:` line + both emitted
  - self-`Needs:` + unknown-id `Needs:` → expect `DROPPED:` lines, rest ordered
  - empty → expect `WAVES: 0`
  - unparseable (a line that matches neither a task nor the grammar) → expect
    `PARSE:` + exit 2

  ```js
  import { test } from "node:test";
  import assert from "node:assert/strict";
  import { run } from "./helpers.mjs"; // helper: spawns node <script> <file>, returns {code, out}

  test("linear: dependent task lands in a later wave", () => {
    const out = run(`Task 1: Owns: a\nTask 2: Owns: b\nTask 3: Needs: 2\n`);
    assert.equal(out.code, 0);
    assert.match(out.out, /WAVE 0: 1 2/);      // 1 and 2 have no needs
    assert.match(out.out, /WAVE 1: 3/);         // 3 depends on 2
  });

  test("cycle: broken and named, still emitted, exit 0", () => {
    const out = run(`Task 1: Needs: 2\nTask 2: Needs: 1\n`);
    assert.equal(out.code, 0);
    assert.match(out.out, /CYCLE: 1 2/);
  });

  test("empty: WAVES: 0, exit 0", () => {
    const out = run(``);
    assert.equal(out.code, 0);
    assert.match(out.out, /WAVES: 0/);
  });
  ```

- [ ] **Step 2: Run test to verify it fails.**

  Run: `node --test .agents/skills/execution/parallel-execution/tests/compute-waves.test.mjs`
  Expected: FAIL (script not found / `WAVES:` lines absent).
  Also confirm the `[compute-waves-orders-independent-and-dependent]` scenario
  is RED in `acceptance.feature` once authored (dry-run).

- [ ] **Step 3: Write minimal implementation.** `compute-waves`:
  1. Read the file (arg) or stdin.
  2. Parse `Task <N>: Owns:` lines → task ids; `Task <N>: Needs: <a, b>` lines
     → edges. Validate the grammar against the real `tasks.md` sample; any line
     that is neither a recognized `Task`/`Owns`/`Needs` form nor a blank/heading
     is a `PARSE:` error (exit 2). (Keep it lenient: ignore headings, blank
     lines, checkbox marks.)
  3. Prune: for each edge, drop self-`Needs:` and `Needs:` to an unknown id;
     emit one `DROPPED:` line per pruned edge.
  4. Kahn cycle break (Cleave's `detectAndBreakCycles`): compute in-degree, peel
     in-degree-0 nodes; the un-visited set is the cycle → clear their `needs`,
     emit `CYCLE:` with those ids.
  5. `computeDispatchWaves` (Cleave's): repeatedly emit all remaining tasks
     whose `needs` are all satisfied; if a pass yields none but tasks remain
     (shouldn't after cycle break — defensive), emit the residual as one final
     wave with a `CYCLE:` note.
  6. Print `WAVES: <n>` then one `WAVE <i>: <ids>` line each, ids sorted.

  ```js
  #!/usr/bin/env node
  import { readFileSync } from "node:fs";
  // ... parse, prune, detectAndBreakCycles, computeDispatchWaves, print WAVES/WAVE/DROPPED/CYCLE
  // exit 0 on a valid (possibly cycle-broken) graph; exit 2 with PARSE: on unparseable input
  ```

- [ ] **Step 4: Run test to verify it passes.**

  Run: `node --test .agents/skills/execution/parallel-execution/tests/compute-waves.test.mjs`
  Expected: PASS (all cases).

- [ ] **Step 5: Commit.**

  ```bash
  git add .agents/skills/execution/parallel-execution/scripts/compute-waves \
          .agents/skills/execution/parallel-execution/tests/compute-waves.test.mjs
  git commit -m "feat(skills): add compute-waves deterministic dispatch-order script"
  ```

### Task 2: `classify-dirty` script + test

**Files:**
- Create: `.agents/skills/execution/isolated-workspace/scripts/classify-dirty`
- Test: `.agents/skills/execution/isolated-workspace/tests/classify-dirty.test.mjs`

**Interfaces:**
- Consumes: none (self-contained); reads `git status --porcelain` from the repo.
- Produces: a CLI `classify-dirty <repo-path> <own-glob> [<own-glob> ...]` whose
  stdout is:
  - `RELATED: <path> ...` (one per line, or `RELATED: (none)`)
  - `VOLATILE: <path> ...`
  - `UNRELATED/UNKNOWN: <path> ...`
  - `CHECKPOINT: <suggested commit message covering the related set>` (or
    `CHECKPOINT: (nothing to checkpoint)` when related is empty)
  - `AUTO_RESOLVABLE: true|false` (true iff related is empty AND
    unrelated/unknown is empty AND volatile is non-empty)
  - Exit 0 always (classification is advisory; the operator picks the action).
- Seam: `classify(path, ownGlobs, volatileAllowlist) -> bucket` — the pure
  decision function; the `git status` read and the printing wrap it.
- Deletion test: if deleted, `isolated-workspace` reverts to a bare
  clean-tree requirement with no classification (the gap this task closes);
  callers are the `isolated-workspace/SKILL.md` prose (Task 4) and its test.
- Adapters: 2 (a real temp git repo + the no-git degraded path) — justified.

**SATISFIES:** `classify-dirty-buckets-and-autoresolve`

- [ ] **Step 1: Write the failing test.** Create a temp git repo (`git init`,
  a tracked file, then dirty variants):
  - mixed: 1 path matching an `Owns:` glob, 1 volatile (`dist/x.js`), 1
    unrelated (`notes.txt`) → assert three buckets + non-empty `CHECKPOINT:` +
    `AUTO_RESOLVABLE: false`
  - volatile-only: only `dist/y.js` dirty → assert `AUTO_RESOLVABLE: true`,
    empty `RELATED:`
  - all-related: two paths under an `Owns:` glob → both under `RELATED:`,
    `AUTO_RESOLVABLE: false`
  - no-git: a non-repo dir → paths fall to `UNRELATED/UNKNOWN:` with a
    degraded note, no stack trace, exit 0

  ```js
  import { test } from "node:test";
  import assert from "node:assert/strict";
  import { mkdtempRepo } from "./helpers.mjs"; // git init + commit a baseline file

  test("mixed tree: three buckets + checkpoint + not auto-resolvable", () => {
    const repo = mkdtempRepo();
    // dirty: src/auth/login.ts (matches 'src/auth/**'), dist/x.js, notes.txt
    const out = runClassify(repo, "src/auth/**");
    assert.equal(out.code, 0);
    assert.match(out.out, /RELATED: src\/auth\/login\.ts/);
    assert.match(out.out, /VOLATILE: dist\/x\.js/);
    assert.match(out.out, /UNRELATED\/UNKNOWN: notes\.txt/);
    assert.match(out.out, /AUTO_RESOLVABLE: false/);
    assert.doesNotMatch(out.out, /CHECKPOINT: \(nothing to checkpoint\)/);
  });

  test("volatile-only: auto-resolvable, empty related", () => {
    const repo = mkdtempRepo();
    // dirty: dist/y.js only
    const out = runClassify(repo, "src/auth/**");
    assert.match(out.out, /AUTO_RESOLVABLE: true/);
    assert.match(out.out, /RELATED: \(none\)/);
  });
  ```

- [ ] **Step 2: Run test to verify it fails.**

  Run: `node --test .agents/skills/execution/isolated-workspace/tests/classify-dirty.test.mjs`
  Expected: FAIL (script not found).

- [ ] **Step 3: Write minimal implementation.** `classify-dirty`:
  1. `git status --porcelain` in `<repo-path>` (via `child_process.execSync`,
     captured; on non-zero exit → degrade: every path is `UNRELATED/UNKNOWN:`
     with a `NOTE: git status failed; treating all paths as unknown` line,
     exit 0).
  2. Parse porcelain lines (XY + space + path; handle `??` untracked and the
     `old -> new` rename form by taking the new path).
  3. For each path, `classify()`: match against the `Owns:` globs (a small
     glob matcher handling `**`, `*`, literal segments — implement minimally
     since stdlib has no glob; anchor to repo root, case-sensitive) → `related`;
     else match the volatile allowlist (a fixed `const` of suffixes/patterns:
     `dist/`, `build/`, `node_modules/`, `*.log`, `*.tmp`, `.env.local`,
     `coverage/`, `*.min.js`, `.DS_Store`) → `volatile`; else
     `unrelated/unknown`.
  4. `AUTO_RESOLVABLE = (related.length === 0 && unknown.length === 0 &&
     volatile.length > 0)`.
  5. `CHECKPOINT:` message = `chore(cleave): checkpoint related changes` plus
     the related paths (only when related is non-empty).
  6. Print the stable line format.

- [ ] **Step 4: Run test to verify it passes.**

  Run: `node --test .agents/skills/execution/isolated-workspace/tests/classify-dirty.test.mjs`
  Expected: PASS.

- [ ] **Step 5: Commit.**

  ```bash
  git add .agents/skills/execution/isolated-workspace/scripts/classify-dirty \
          .agents/skills/execution/isolated-workspace/tests/classify-dirty.test.mjs
  git commit -m "feat(skills): add classify-dirty preflight bucket script"
  ```

### Task 3: Churn breaker rule in `concurrent-leaves.md` + `subagent-execution` reference

> Prose-only task; the "test" is a presence + shape check (grep + the
> `acceptance.feature` scenario), not a subprocess test.

**Files:**
- Modify: `.agents/skills/execution/parallel-execution/references/concurrent-leaves.md`
  — add a **Churn breaker** section (after the Parent re-verify section).
- Modify: the `subagent-execution` fix-loop text (locate where rounds 1–5 /
  the breaker adjudication are described — `SKILL.md` and/or
  `references/task-loop.md`; reference the churn breaker at that exact spot).

**Interfaces:**
- Produces: a named, checkable rule. The rule text MUST state all of:
  - **Stable issue key:** `file + symbol + issue-type`.
  - **Threshold:** reappear fraction ≥ **0.5** (default; overridable per change
    by a one-line `Churn threshold: <n>` in the ledger).
  - **Action:** stop the fix loop, park each still-open issue in the ledger
    with a `Ruling:`, and escalate; do **not** start the next round.
  - **Relation to the round cap:** the breaker is a *convergence* stop that can
    fire *before* the existing round-5 cap; the cap remains the *hard* stop.
- Seam: none (in-prose rule).
- Deletion test: if deleted, a non-converging review spins to the round cap
  without a convergence flag (the gap this task closes).
- Adapters: n/a (prose).

**SATISFIES:** `churn-breaker-stops-reappearing-fix-loop`

- [ ] **Step 1: Write the presence/shape check.** A small script or grep-assert
  (co-located as `tests/churn-breaker-prose.test.mjs` under
  `parallel-execution/tests/`) that:
  - reads `concurrent-leaves.md`, asserts it contains `Churn breaker`, the
    stable key `file + symbol + issue-type` (or equivalent), `0.5`, and the
    stop-park-escalate action words;
  - reads the `subagent-execution` fix-loop file, asserts it references
    `churn`.

  ```js
  import { test } from "node:test";
  import assert from "node:assert/strict";
  import { readFileSync } from "node:fs";

  test("churn breaker rule present and complete", () => {
    const cl = readFileSync(new URL("../references/concurrent-leaves.md", import.meta.url), "utf8");
    assert.match(cl, /Churn breaker/i);
    assert.match(cl, /0\.5/);
    assert.match(cl, /issue[- ]type|symbol/i);
    assert.match(cl, /Ruling:/);
  });
  ```

- [ ] **Step 2: Run test to verify it fails.** Expected: FAIL (section absent).
- [ ] **Step 3: Write the rule.** Add the **Churn breaker** section to
  `concurrent-leaves.md` and the one-line reference in the `subagent-execution`
  fix loop, per the interface above. Cite Cleave `review.ts` as the source of
  the 0.5 default in a parenthetical.
- [ ] **Step 4: Run test to verify it passes.** Expected: PASS.
- [ ] **Step 5: Commit.**

  ```bash
  git add .agents/skills/execution/parallel-execution/references/concurrent-leaves.md \
          .agents/skills/execution/parallel-execution/tests/churn-breaker-prose.test.mjs \
          .agents/skills/execution/subagent-execution/
  git commit -m "docs(skills): add review churn breaker rule to concurrent-leaves"
  ```

### Task 4: Wire the skill prose to the scripts

**Files:**
- Modify: `.agents/skills/execution/parallel-execution/references/concurrent-leaves.md`
  — in the **Dispatch** section, add: run `scripts/compute-waves <tasks-file>`
  before the first dispatch; dispatch only the current wave; re-run after each
  wave completes to get the next.
- Modify: `.agents/skills/execution/isolated-workspace/SKILL.md`
  — in the preflight/clean-tree step, add: if the tree is dirty, run
  `scripts/classify-dirty <repo> <Owns-globs...>`; present the three buckets to
  the operator with the five actions (checkpoint / stash-unrelated /
  stash-volatile / proceed-without-cleave / cancel); auto-resolve when
  `AUTO_RESOLVABLE: true`; cap manual resolution at 3 attempts then stop and
  name the remaining paths.

**Interfaces:**
- Consumes: the `compute-waves` and `classify-dirty` CLIs (Tasks 1–2).
- Produces: the run-when + use-output instructions R4 requires.
- Seam / Deletion test / Adapters: none (prose wiring).

**SATISFIES:** `skills-reference-deterministic-guardrails`

- [ ] **Step 1: Write the presence check.** Extend
  `tests/churn-breaker-prose.test.mjs` (or a sibling
  `tests/wiring-prose.test.mjs`) to assert:
  - `concurrent-leaves.md` references `compute-waves` with a run-when +
    dispatch-only-current-wave instruction;
  - `isolated-workspace/SKILL.md` references `classify-dirty` with the five
    actions + the 3-attempt cap + auto-resolve.

- [ ] **Step 2: Run test to verify it fails.** Expected: FAIL.
- [ ] **Step 3: Write the prose.** Add the two wiring blocks above.
- [ ] **Step 4: Run test to verify it passes.** Expected: PASS.
- [ ] **Step 5: Commit.**

  ```bash
  git add .agents/skills/execution/parallel-execution/references/concurrent-leaves.md \
          .agents/skills/execution/isolated-workspace/SKILL.md \
          .agents/skills/execution/parallel-execution/tests/
  git commit -m "docs(skills): wire compute-waves and classify-dirty into dispatch + preflight"
  ```

### Task 5: Change ADR manifest + VERIFIED note

**Files:**
- Create: `.skillgrid/specs/2026-10-20-parallel-cleave-port/adr.md` — the change
  manifest: list ADR-0020 / ADR-0016 / ADR-0003 as **respected** (pointer +
  one-line how this change obeys each), and an explicit **"No new ADR"** note
  with the promote condition (promote to a candidate ADR row if these scripts
  are invoked by ≥2 unrelated changes).
- Modify: `.skillgrid/ASSUMPTIONS.md` — one line under **VERIFIED** noting the
  three guardrail scripts exist and their paths (so a future session can find
  them). Do **not** add a LOCKED row (no new ADR this change).

**Interfaces:** none.
**SATISFIES:** `skills-reference-deterministic-guardrails` (final coherence
check that the manifest matches what landed).

- [ ] **Step 1: Write `adr.md`** with the respected-ADR pointers + no-new-ADR
  note.
- [ ] **Step 2: Add the VERIFIED line** to `ASSUMPTIONS.md`.
- [ ] **Step 3: Verify** the manifest's respected-ADR list matches the
  blueprint header and that `ASSUMPTIONS.md` gained exactly one VERIFIED line
  and zero LOCKED rows (`git diff --stat` + grep).
- [ ] **Step 4: Commit** (spec zone — this and the VERIFIED note are spec-zone
  artifacts; per the zone rule they commit in the spec-zone batch, before the
  code-zone script commits if not already committed there).

  ```bash
  git add .skillgrid/specs/2026-10-20-parallel-cleave-port/adr.md .skillgrid/ASSUMPTIONS.md
  git commit -m "docs(skillgrid): manifest ADRs for parallel cleave port; note guardrail scripts"
  ```

---

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| Shell command injection (the scripts build a `git` argv / spawn `node`) | Applicable | `classify-dirty` uses `child_process.execFileSync` (argv array, not a shell string) so a path with spaces/`$` is one arg, not interpreted; no user input is ever concatenated into a shell command string. | `classify-dirty.test.mjs` case with a dirty path containing a space + `$` classifies correctly and does not crash. |
| Path traversal via `Owns:` globs / task `Needs:` ids | Applicable | `Owns:` globs are anchored to the repo root and matched by the script's own minimal matcher (no `..` segment crosses the root); `Needs:` ids are matched against the parsed task-id set, so a `Needs:` referencing `../../etc` is an unknown id → `DROPPED:`, never a file read. | `compute-waves.test.mjs` unknown-id case; `classify-dirty` case with a glob containing `..` that does not escape the root. |
| Executable-file classification (a dirty path that is a build artifact vs source) | Applicable | The volatile allowlist is a fixed, named `const` in `classify-dirty` (not heuristic); a path matching neither `Owns:` nor the allowlist is `unrelated/unknown`, never silently treated as volatile. | `classify-dirty.test.mjs` mixed case asserts a source file is NOT bucketed volatile. |
| Version-control automation (checkpoint commit the script *suggests*) | N/A: the script only *suggests* a message; it never runs `git add`/`git commit`. The operator (or the existing `isolated-workspace` flow) performs the commit. | `classify-dirty` emits `CHECKPOINT:` as data, not a side effect; its test runs in a temp repo and asserts no new commit exists after the run. |
| Subprocess / process integration (scripts spawned by the agent) | Applicable | Both scripts are stdlib-only, exit non-zero only on a defined error (`PARSE:`), and print a stable line format; the skill prose tells the agent to fall back to hand-ordering / hand-classification on a non-zero exit (fail-open, matching ADR-0021's fail-open posture). | `compute-waves.test.mjs` unparseable → exit 2; the prose references the fallback. |
| `_shared/{planning,craft,verification,knowledge}/` file touch | N/A: this change edits `execution/` skill files and `scripts/` under them, not a `_shared/` rules file. The churn breaker lives in `concurrent-leaves.md` (an `execution/` reference), not a `_shared/` rule. | No `_shared/` file is modified; `sdd-structure.md` is read-only here. | grep over the change's `git diff --name-only` shows no `_shared/` path. |

## Hypothesis

**Claim:** Porting Cleave's three deterministic guardrails as dependency-free
skill scripts + prose makes a skillgrid parallel run deterministic (wave
order), safe (dirty-tree handled explicitly), and convergent (review churn
bounded) — without touching the Go binary, the dependency surface, or the SDD
ledger.

**Right condition:** The three scripts each pass a RED→GREEN fixture test; the
skill prose names each script with a run-when + use-output instruction; and a
review of the final diff confirms zero `go.mod`/`package.json` changes, zero
`_shared/` rule edits, and zero new SDD-ledger state file.

**Wrong condition:** A script cannot parse the real pipeline `tasks.md`
`Needs:` grammar (door check fails), OR wiring requires a Go command / new npm
dep / a second execution record to work — any of which means the "scripts +
prose" approach is wrong and the runtime-port or Go-command approach must be
reconsidered.

**Thinnest MVP:** Task 1 (`compute-waves` + its fixture test) — the first
increment that makes dispatch order deterministic and is independently usable.

**Door check:** Task 1. If the `Needs:` line grammar can't be parsed reliably
from a real `tasks.md`, stop and resolve the briefing's Open Question (parse
`tasks.md` vs. JSON input) before continuing.

## Self-Review (run before handoff)

1. **Spec coverage:** R1 → Task 1; R2 → Task 2; R3 → Task 3; R4 → Task 4;
   ADR manifest + VERIFIED note → Task 5. No requirement without a task.
2. **Must-haves coverage:** every Truth maps to a task's test/prose; every
   Artifact is a Create/Modify in a task; Key links are named in the Key links
   section and enforced by the prose-wiring test (Task 4). One-way-door: None
   (stated).
3. **Owed-Decision Gate (input coverage):** every value a script emits has a
   named source — wave order from the `Needs:` graph (parse grammar named in
   Task 1), bucket from the `Owns:` glob + volatile `const` (named in Task 2),
   threshold 0.5 (named, Cleave-sourced, in Task 3). The one genuinely open
   choice — `tasks.md` parse vs. JSON input — is captured as the door check
   (Task 1) and the briefing's Open Question, not invented mid-build. No
   `ASSUMED` block required.
4. **Placeholder scan:** no TBD / "implement later" / "similar to Task N" —
   each task names exact files, the exact output line format, and the exact
   test cases.
5. **Type/name consistency:** the output line prefixes (`WAVES:`, `WAVE `,
   `DROPPED:`, `CYCLE:`, `PARSE:`, `RELATED:`, `VOLATILE:`,
   `UNRELATED/UNKNOWN:`, `CHECKPOINT:`, `AUTO_RESOLVABLE:`) are used
   identically in the blueprint's Interfaces, the test asserts, and the
   skill-prose references. The `Needs:`/`Owns:` line grammar is referenced
   from the same named source (the real `tasks.md` sample) in Task 1 and Task 4.
