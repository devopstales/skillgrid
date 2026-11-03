---
type: Blueprint
title: "2026-10-20-parallel-cleave-port — blueprint"
status: draft

---

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
- Canonical terms live in `.skillgrid/decisions/` (`01-business-terms.md` /
  `02-technical-terms.md`); nothing here redefines them.

## Must-Haves (goal-backward verification)

**Truths** (observable behaviors that must hold):

- Running `scripts/compute-waves` on a 3-ticket `tasks.md` whose `Blocked by:`
  bullets + mermaid block agree (T3 blocked by T2; T1, T2 free) prints exactly
  two waves, with `T3` in the later wave and `T1`/`T2` in wave 0; exit code 0.
  (R1)
- Running `scripts/compute-waves` on a 2-ticket cycle (T1 blocked by T2, T2
  blocked by T1) prints both tickets (broken into the same wave) and a `CYCLE:`
  line naming them; exit code 0 (a broken cycle is a recoverable input, not a
  crash). (R1)
- Running `scripts/compute-waves` on a ticket with a self-`Blocked by:` or a
  `Blocked by:` reference to an unknown id drops that edge, reports it on a
  `DROPPED: <id> blocked-by <self|unknown>` line, and still orders the rest;
  exit code 0. (R1)
- Running `scripts/compute-waves` on a `tasks.md` where a ticket's `Blocked by:`
  bullet set disagrees with its mermaid incoming edges prints a `MISMATCH:` line
  naming the ticket, proceeds on the bullets, and exits 0. (R1, decision A+B)
- Running `scripts/compute-waves` on a duplicate ticket id prints a `DUP:` line,
  keeps the first occurrence's edges, and exits 0. (R1)
- Running `scripts/compute-waves` on an empty/edge-less `tasks.md` prints
  `WAVES: 0` without error; exit code 0. (R1)
- Running `scripts/compute-waves` on unparseable input (a `## Dependency graph`
  block with no ticket ids or edges) prints a `PARSE:` line and exits 2. (R1)
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
  script, stdlib only, parses `tasks.md` `Blocked by:` bullets (primary) +
  `## Dependency graph` mermaid (cross-check), prints ordered waves +
  `CYCLE:`/`DROPPED:`/`MISMATCH:`/`DUP:` lines.
- `.agents/skills/execution/parallel-execution/tests/compute-waves.test.mjs` —
  fixture-based test (subprocess) covering linear, diamond, cycle, self/unknown
  edge, bullets-vs-mermaid mismatch, duplicate id, empty, unparseable.
- `.agents/skills/execution/isolated-workspace/scripts/classify-dirty` — Node
  script, stdlib only, reads `git status --porcelain` + `Owns:` globs, prints
  buckets + `CHECKPOINT:` + `AUTO_RESOLVABLE:`.
- `.agents/skills/execution/isolated-workspace/tests/classify-dirty.test.mjs` —
  fixture-repo test (creates a temp git repo) covering mixed, volatile-only,
  all-related, no-git.
- `.agents/skills/execution/parallel-execution/scripts/churn-check` — Node
  script, stdlib only, computes the reappear fraction over two rounds' issue
  lists, prints `REAPPEAR n/m (frac)` + `FIRES: true|false`.
- `.agents/skills/execution/parallel-execution/tests/churn-check.test.mjs` —
  fixture test (fires at ≥0.5, not below, empty-prev).
- `concurrent-leaves.md` — wave step naming `compute-waves` (gated ≥4 leaves) +
  a **Churn breaker** section naming `churn-check`.
- `isolated-workspace/SKILL.md` — preflight step naming `classify-dirty`.
- `subagent-execution/` — a reference line from its task loop to the churn
  breaker.
- `adr.md` (change manifest) — pointers to ADR-0020/0016/0003 as respected,
  and an explicit "no new ADR" note with the promote-to-ADR condition.

**Key links** (critical connections that must work together):

- `compute-waves` must parse the *same* `tasks.md` graph `slicing` writes:
  per-ticket `- **Blocked by:** <ids>` bullets (primary) **and** the
  `## Dependency graph` mermaid block (cross-check). The two must be read
  together (decision A+B): bullets are authoritative, mermaid is the
  consistency check, disagreement is a `MISMATCH:` line, not a silent pick.
  Parsing only one of the two would silently mis-order on a drifted graph.
- `classify-dirty`'s `Owns:` glob input must match the `Task <N>: Owns: <globs>`
  line format already written by `concurrent-leaves.md` (the dispatch brief /
  ledger, not `tasks.md`).
- The skill prose in `concurrent-leaves.md` must invoke `compute-waves` *before*
  the first dispatch and instruct the agent to dispatch only the current wave —
  the script's output is the dispatch order, not advice.
- **Wave gate (critic):** `compute-waves` is invoked only when the plan has ≥4
  leaves in flight (or ≥2 `Blocked by:` edges) — matching
  `concurrent-leaves.md`'s existing note that wave machinery "only pays off at
  4+ leaves." Below that, the agent orders by hand (the prose contract is
  enough).
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
  — pure graph→waves CLI (parse `Blocked by:` bullets + mermaid cross-check,
  Kahn order, cycle break). One responsibility: given a `tasks.md` graph, print
  dispatch order.
- `.agents/skills/execution/parallel-execution/tests/compute-waves.test.mjs`
  — subprocess test over fixture task lists.
- `.agents/skills/execution/isolated-workspace/scripts/classify-dirty`
  — `git status` → related/volatile/unrelated buckets + checkpoint message. One
  responsibility: classify a dirty tree against `Owns:` globs.
- `.agents/skills/execution/isolated-workspace/tests/classify-dirty.test.mjs`
  — temp-git-repo test over fixture dirty trees.
- `.agents/skills/execution/parallel-execution/scripts/churn-check`
  — reappear-fraction CLI (`REAPPEAR n/m (frac)` + `FIRES:`). One
  responsibility: given two rounds' issue lists, report whether the breaker
  fires.
- `.agents/skills/execution/parallel-execution/tests/churn-check.test.mjs`
  — fixture test (fires ≥0.5, not below, empty-prev).
- `.agents/skills/execution/parallel-execution/references/concurrent-leaves.md`
  — modify: add the wave-computation step (name `compute-waves`, gated ≥4
  leaves) + a **Churn breaker** section naming `churn-check`.
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

> **Checkpoint / door check (corrected per critic + user decision A+B).** The
> dependency graph in a real `tasks.md` does **not** use `Needs:` lines — it
> uses per-ticket `- **Blocked by:** <ids>` bullets (stable, line-based, written
> by `slicing` today) **and** a `## Dependency graph` mermaid `graph TD` block
> (the declared graph). The script parses **both**: the bullets are the primary
> edge source; the mermaid block is a consistency cross-check. **Stop if** the
> real `tasks.md` bullets cannot be parsed reliably — that is the premise this
> door check actually tests (not a `Needs:` line, which does not exist in
> `tasks.md`).

**Files:**
- Create: `.agents/skills/execution/parallel-execution/scripts/compute-waves`
- Test: `.agents/skills/execution/parallel-execution/tests/compute-waves.test.mjs`
- Read (format source, do not edit): the real sample
  `.skillgrid/specs/2026-10-06-second-brain-learnings/tasks.md` — it carries
  the `## Dependency graph` mermaid block and per-ticket `- **Blocked by:**
  <ids>` bullets. Lock the exact grammar the parser accepts against it.

**Interfaces:**
- Consumes: a `tasks.md` (or stdin) carrying per-ticket `- **Blocked by:**
  <ids>` bullets and a `## Dependency graph` mermaid `graph TD` block.
- Produces: a CLI `compute-waves <tasks-file>` (or stdin) whose stdout is a
  stable line format:
  - `WAVES: <count>`
  - `WAVE <i>: <id> <id> ...` (one line per wave, in order; ids sorted)
  - `DROPPED: <id> blocked-by <self|unknown>` (per pruned edge; `<reason>` is a
    fixed enum, no free text)
  - `CYCLE: <id> <id> ...` (ids whose edges were cleared to break a cycle)
  - `MISMATCH: <id> bullets=<set> mermaid=<set>` (a ticket whose `Blocked by:`
    bullet set differs from its mermaid incoming edges; the script proceeds on
    the bullets and names the conflict)
  - `DUP: <id>` (a ticket id that appears more than once; first occurrence's
    edges win; a warning, not an error)
  - `PARSE: <reason>` + exit 2 on genuinely unparseable input (a `## Dependency
    graph` block present but no ticket ids or edges can be extracted)
  - `WAVES: 0` for an empty/edge-less graph; exit 0 (a valid empty graph is not
    an error)
- Seam: the script's `parseGraph(text) -> {ids, edges, source}` function — the
  place the `tasks.md` grammar (bullets + mermaid) is interpreted. Behavior can
  change there without touching the wave algorithm.
- Deletion test: if deleted, the agent re-derives wave order by hand every run
  (the gap this task closes); callers are the `concurrent-leaves.md` prose
  (Task 4) and its test.
- Adapters: 2 (production `tasks.md` parsing + the test's synthetic graphs) —
  the seam is justified.

**SATISFIES:** `compute-waves-orders-independent-and-dependent`

- [ ] **Step 1: Write the failing test.** Fixture cases (write each as a temp
  `tasks.md` the test feeds the script via its arg or stdin). Each fixture uses
  the **real** grammar: `- **Blocked by:**` bullets + a matching `## Dependency
  graph` mermaid block.
  - linear + diamond (bullets + mermaid agree) → expect exact `WAVE` lines
  - 2-cycle → expect both ids on one `CYCLE:` line + both emitted
  - self-`Blocked by:` + `Blocked by:` to an unknown id → expect `DROPPED:`
    lines, rest ordered
  - bullets vs mermaid disagree on one ticket → expect a `MISMATCH:` line, exit
    still 0 (bullets win, conflict named)
  - a duplicate ticket id → expect a `DUP:` line, exit 0, first-occurrence wins
  - empty (no tickets / no edges) → expect `WAVES: 0`
  - a `## Dependency graph` block present but unparseable (no ids, no edges) →
    expect `PARSE:` + exit 2

  ```js
  import { test } from "node:test";
  import assert from "node:assert/strict";
  import { run } from "./helpers.mjs"; // helper: spawns node <script> <file>, returns {code, out}

  const tasks = (tickets, mermaid) =>
    `## Dependency graph\n\`\`\`mermaid\ngraph TD\n${mermaid}\n\`\`\`\n\n` +
    tickets.map(t => `### ${t.id}\n- **Blocked by:** ${t.blockedBy}`).join("\n\n");

  test("linear: dependent task lands in a later wave (bullets+mermaid agree)", () => {
    const out = run(tasks(
      [{ id: "T1", blockedBy: "none" }, { id: "T2", blockedBy: "none" }, { id: "T3", blockedBy: "T2" }],
      "T1 --> T3\nT2 --> T3"));
    assert.equal(out.code, 0);
    assert.match(out.out, /WAVE 0: T1 T2/);   // T1, T2 have no blockers
    assert.match(out.out, /WAVE 1: T3/);       // T3 blocked by T2
  });

  test("cycle: broken and named, still emitted, exit 0", () => {
    const out = run(tasks(
      [{ id: "T1", blockedBy: "T2" }, { id: "T2", blockedBy: "T1" }], "T1 --> T2\nT2 --> T1"));
    assert.equal(out.code, 0);
    assert.match(out.out, /CYCLE: T1 T2/);
  });

  test("bullets vs mermaid mismatch: MISMATCH line, exit 0, bullets win", () => {
    const out = run(tasks(
      [{ id: "T1", blockedBy: "none" }, { id: "T2", blockedBy: "T1" }], // T2 blocked by T1 (bullet)
      "T1 --> T1"));                                                    // mermaid says T1 self-edge (disagrees)
    assert.equal(out.code, 0);
    assert.match(out.out, /MISMATCH:/);
  });

  test("empty: WAVES: 0, exit 0", () => {
    const out = run(`## Dependency graph\n\`\`\`mermaid\ngraph TD\n\`\`\`\n`);
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
  2. Parse ticket ids from `### <id>` headings. Parse edges from
     `- **Blocked by:** <a, b>` bullets (split on commas; `none`/blank = no
     edge). Parse the `## Dependency graph` mermaid `graph TD` block for a
     second edge set (`A --> B` = B is blocked by A). Be lenient: ignore other
     headings, blank lines, checkbox marks, and the `## Waves` table (not
     parsed — it is a human summary). A `## Dependency graph` block with neither
     ids nor edges → `PARSE:` (exit 2).
  3. Cross-check: for each ticket compare its bullet-blocker set to its mermaid
     incoming-edge set; on disagreement emit `MISMATCH:` (bullets are
     authoritative). A duplicate ticket id emits `DUP:` (first occurrence's
     edges win).
  4. Prune: drop self-`Blocked by:` and `Blocked by:` to an unknown id; emit
     one `DROPPED: <id> blocked-by <self|unknown>` per pruned edge.
  5. Kahn cycle break (Cleave's `detectAndBreakCycles`): compute in-degree, peel
     in-degree-0 nodes; the un-visited set is the cycle → clear their edges,
     emit `CYCLE:` with those ids.
  6. `computeDispatchWaves` (Cleave's): repeatedly emit all remaining tasks
     whose blockers are all satisfied; if a pass yields none but tasks remain
     (shouldn't after cycle break — defensive), emit the residual as one final
     wave with a `CYCLE:` note.
  7. Print `WAVES: <n>` then one `WAVE <i>: <ids>` line each, ids sorted.

  ```js
  #!/usr/bin/env node
  import { readFileSync } from "node:fs";
  // parseGraph: bullets (primary) + mermaid (cross-check) -> {ids, edges}
  // cross-check (MISMATCH/DUP), prune (DROPPED), detectAndBreakCycles (CYCLE),
  // computeDispatchWaves, print WAVES/WAVE/DROPPED/CYCLE/MISMATCH/DUP/PARSE
  // exit 0 on a valid (possibly cycle-broken/mismatched) graph; exit 2 with PARSE: on unparseable input
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
  stdout is (each bucket header on its own line, then **one path per line**
  indented beneath it — never a space-joined path list, so the test and the
  skill prose parse it identically):
  - `RELATED:` then one `  <path>` per related path (or `RELATED: (none)`)
  - `VOLATILE:` then one `  <path>` per volatile path (or `VOLATILE: (none)`)
  - `UNRELATED/UNKNOWN:` then one `  <path>` per path (or
    `UNRELATED/UNKNOWN: (none)`)
  - `CHECKPOINT: <suggested commit message covering the related set>` (or
    `CHECKPOINT: (nothing to checkpoint)` — the empty-set marker — when related
    is empty)
  - `AUTO_RESOLVABLE: true|false` (true iff related is empty AND
    unrelated/unknown is empty AND volatile is non-empty; a **clean** tree —
    no dirty paths at all — is `AUTO_RESOLVABLE: false` with all three buckets
    `(none)`, so the operator knows the script ran)
  - Exit 0 always (classification is advisory; the operator picks the action).
- Paths are always resolved to **repo-root-relative** before glob matching. A
  dirty **submodule** path is bucketed by its own path (it is not expanded).
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
- Create: `.agents/skills/execution/parallel-execution/scripts/churn-check` —
  the deterministic half of the breaker (critic: the reappear-fraction is
  repeated logic that belongs in a script per `skill-write`). The prose states
  the *decision* (stop-park-escalate); the script states the *calculation*.
- Test: `.agents/skills/execution/parallel-execution/tests/churn-check.test.mjs`

**Interfaces:**
- Produces: a named, checkable rule **plus** a deterministic checker. The rule
  text MUST state all of:
  - **Stable issue key:** `file + symbol + issue-type`.
  - **Threshold:** reappear fraction ≥ **0.5** (Cleave-parity default;
    overridable per change by a one-line `Churn threshold: <n>` in the ledger).
  - **Action:** stop the fix loop, park each still-open issue in the ledger
    with a `Ruling:`, and escalate; do **not** start the next round.
  - **Relation to the round cap:** the breaker is a *convergence* stop that can
    fire *before* the existing round-5 cap; the cap remains the *hard* stop.
  - **The calculation is `churn-check`**, not the agent's arithmetic: the prose
    tells the agent to run `churn-check` on the two rounds' issue lists and act
    on its verdict.
- `churn-check` CLI: `churn-check <prev-rounds-file> <current-rounds-file>
  [threshold]` where each file is one stable issue key per line (`file|symbol|
  issue-type`). Output: `REAPPEAR <n>/<m> (<fraction>)` + `FIRES: true|false`
  (true iff fraction ≥ threshold, default 0.5). Exit 0 always (the verdict is
  advisory; the agent decides the action).
- Seam: `reappearFraction(prevKeys, curKeys) -> number` — the pure
  calculation; the file read + printing wrap it.
- Deletion test: if deleted, a non-converging review spins to the round cap
  without a convergence flag (the gap this task closes); and the fraction is
  re-derived by the agent each round (the `skill-write` gap).
- Adapters: 2 (the prose decision + the `churn-check` calculation).

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

- [ ] **Step 3: Write the `churn-check` test (RED).** `churn-check.test.mjs`:
  two rounds where 2 of 3 prev issues reappear (fraction 0.67 ≥ 0.5) → `FIRES:
  true`; two rounds where 1 of 4 reappears (0.25 < 0.5) → `FIRES: false`;
  empty prev round → `FIRES: false` (no baseline to churn against).

  ```js
  test("reappear fraction >= 0.5 fires", () => {
    const prev = ["a|f|bug", "b|g|bug", "c|h|perf"];
    const cur  = ["a|f|bug", "b|g|bug", "c|h|security"]; // a,b reappear (2/3)
    const out = runChurn(prev, cur);
    assert.match(out.out, /FIRES: true/);
    assert.match(out.out, /REAPPEAR 2\/3/);
  });
  ```

- [ ] **Step 4: Write `churn-check` (GREEN).** Read the two files (one key per
  line), compute `reappearFraction = |prev ∩ cur| / |prev|` (0 when `prev` is
  empty), print `REAPPEAR <n>/<m> (<frac>)` + `FIRES: true|false` against the
  threshold (default 0.5, overridable by the 3rd arg). Exit 0.

- [ ] **Step 5: Run `churn-check` test to verify it passes.** Expected: PASS.

- [ ] **Step 6: Write the rule.** Add the **Churn breaker** section to
  `concurrent-leaves.md` and the one-line reference in the `subagent-execution`
  fix loop, per the interface above. The prose states the decision and tells
  the agent to run `scripts/churn-check` on the two rounds' issue lists and act
  on its `FIRES:` verdict. Cite Cleave `review.ts` as the source of the 0.5
  default in a parenthetical.

- [ ] **Step 7: Re-run the presence/shape test (Step 1) to verify it passes.**
  Expected: PASS.
- [ ] **Step 8: Commit.**

  ```bash
  git add .agents/skills/execution/parallel-execution/references/concurrent-leaves.md \
          .agents/skills/execution/parallel-execution/scripts/churn-check \
          .agents/skills/execution/parallel-execution/tests/churn-check.test.mjs \
          .agents/skills/execution/parallel-execution/tests/churn-breaker-prose.test.mjs \
          .agents/skills/execution/subagent-execution/
  git commit -m "feat(skills): add review churn breaker (churn-check + concurrent-leaves rule)"
  ```

### Task 4: Wire the skill prose to the scripts

**Files:**
- Modify: `.agents/skills/execution/parallel-execution/references/concurrent-leaves.md`
  — in the **Dispatch** section, add: when the plan has **≥4 leaves in flight
  (or ≥2 `Blocked by:` edges)**, run `scripts/compute-waves <tasks-file>` before
  the first dispatch; dispatch only the current wave; re-run after each wave
  completes to get the next. Below that threshold, order by hand (the prose
  contract is enough — matches `concurrent-leaves.md`'s existing "only pays off
  at 4+ leaves" note).
- Modify: `.agents/skills/execution/isolated-workspace/SKILL.md`
  — in the preflight/clean-tree step, add: if the tree is dirty, run
  `scripts/classify-dirty <repo> <Owns-globs...>`; present the three buckets to
  the operator with the five actions (checkpoint / stash-unrelated /
  stash-volatile / proceed-without-cleave / cancel); auto-resolve when
  `AUTO_RESOLVABLE: true`; cap manual resolution at 3 attempts (Cleave-parity)
  then stop and name the remaining paths.

**Interfaces:**
- Consumes: the `compute-waves`, `classify-dirty`, and `churn-check` CLIs
  (Tasks 1–3).
- Produces: the run-when + use-output instructions R4 requires, including the
  ≥4-leaf wave gate.
- Seam / Deletion test / Adapters: none (prose wiring).

**SATISFIES:** `skills-reference-deterministic-guardrails`

- [ ] **Step 1: Write the presence check.** Extend
  `tests/churn-breaker-prose.test.mjs` (or a sibling
  `tests/wiring-prose.test.mjs`) to assert:
  - `concurrent-leaves.md` references `compute-waves` with the ≥4-leaf (or ≥2
    `Blocked by:`) gate + a dispatch-only-current-wave instruction;
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
| Path traversal via `Owns:` globs / `Blocked by:` ticket ids | Applicable | `Owns:` globs are anchored to the repo root and matched by the script's own minimal matcher (no `..` segment crosses the root); `Blocked by:` ticket ids are matched against the parsed `### <id>` set, so a `Blocked by:` referencing an unknown id (e.g. `../../etc`) is `DROPPED:`, never a file read. | `compute-waves.test.mjs` unknown-id case; `classify-dirty` case with a glob containing `..` that does not escape the root. |
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

**Wrong condition:** `compute-waves` cannot parse the real pipeline `tasks.md`
`Blocked by:` bullets + mermaid block reliably (door check fails), OR wiring
requires a Go command / new npm dep / a second execution record to work — any
of which means the "scripts + prose" approach is wrong and the runtime-port or
Go-command approach must be reconsidered.

**Thinnest MVP:** Task 1 (`compute-waves` + its fixture test) — the first
increment that makes dispatch order deterministic and is independently usable.

**Door check:** Task 1. If the `Blocked by:` bullet + mermaid grammar can't be
parsed reliably from a real `tasks.md` (the user-resolved A+B input), stop and
revisit the input source. (The earlier `Needs:`-in-`tasks.md` premise was
refuted by the critic and is no longer the door-check subject.)

## Self-Review (run before handoff)

1. **Spec coverage:** R1 → Task 1; R2 → Task 2; R3 → Task 3; R4 → Task 4;
   ADR manifest + VERIFIED note → Task 5. No requirement without a task.
2. **Must-haves coverage:** every Truth maps to a task's test/prose; every
   Artifact is a Create/Modify in a task; Key links are named in the Key links
   section and enforced by the prose-wiring test (Task 4). One-way-door: None
   (stated).
3. **Owed-Decision Gate (input coverage):** every value a script emits has a
   named source — wave order from the `Blocked by:` bullets + mermaid graph
   (decision A+B; parse grammar named in Task 1 against the real `tasks.md`
   sample), bucket from the `Owns:` glob + volatile `const` (named in Task 2),
   churn fraction from `churn-check` over the two rounds' issue lists (named in
   Task 3), threshold 0.5 (named, Cleave-sourced, in Task 3). The one genuinely
   open choice — which `tasks.md` source carries the graph — was **resolved by
   the user (A+B: bullets primary, mermaid cross-check)** and is no longer owed.
   No `ASSUMED` block required.
4. **Placeholder scan:** no TBD / "implement later" / "similar to Task N" —
   each task names exact files, the exact output line format, and the exact
   test cases.
5. **Type/name consistency:** the output line prefixes (`WAVES:`, `WAVE `,
   `DROPPED:`, `CYCLE:`, `MISMATCH:`, `DUP:`, `PARSE:`, `RELATED:`,
   `VOLATILE:`, `UNRELATED/UNKNOWN:`, `CHECKPOINT:`, `AUTO_RESOLVABLE:`,
   `REAPPEAR`, `FIRES:`) are used identically in the blueprint's Interfaces,
   the test asserts, and the skill-prose references. `DROPPED:` uses the fixed
   enum `<self|unknown>` (no free text). The `Blocked by:` bullet / mermaid
   grammar is referenced from the same named source (the real `tasks.md`
   sample) in Task 1 and Task 4; the `Owns:` line format is referenced from the
   dispatch brief / ledger (not `tasks.md`) in Task 2 and Task 4.
