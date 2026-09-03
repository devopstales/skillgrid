---
name: sdd-archive
description: "Close an SDD change: gate on every step's passing verification.md and the persisted [x] state of each step's tasks.md, then move the entire change folder from docs/skillgrid/changes/<NNN-slug>/ to docs/skillgrid/archive/<NNN-slug>/ with a mechanical copy and verbatim diff -r readback. Use to run the terminal quality gate after sdd-verify and before the cycle ends. The archive is the audit trail. Uses Mnemonic memory only for recovery; the archive move is a shell operation. No external binaries."
disable-model-invocation: true
user-invocable: false
license: MIT
metadata:
  author: devopstales
  version: "2.0"
  family: sdd
  part-of: skillgrid
  phase-order: "tasks → apply → verify → archive"
  prev: [sdd-verify]
  next: []
  artifact: archive-report
  delegate_only: true
---

# sdd-archive

## Execution Role

Confirm your role before acting. You are the dedicated `sdd-archive` sub-agent **unless** you loaded this skill directly through the `skill()` tool.

- **Sub-agent (primary)**: you were delegated here by the SDD orchestrator. Continue with the phase work below. Do not re-delegate. Do not call the `skill()` tool again.
- **Orchestrator (skill() loaded this directly)**: STOP. Delegate to the dedicated `sdd-archive` sub-agent using your platform's delegation primitive (e.g. `task(...)`) instead of doing the work inline.

## Purpose

You are the ARCHIVE phase — the terminal step of the SDD cycle. You close the change: verify that **every step** has a PASS verdict in its `steps/<NN-name>/verification.md` and a fully-checked `steps/<NN-name>/tasks.md`, then move the entire change folder from `docs/skillgrid/changes/<NNN-slug>/` to `docs/skillgrid/archive/<NNN-slug>/`, and record a final-state `archive-report`. You complete the cycle so the next change starts clean and the work is permanently auditable.

Phase order is `init → explore → propose → design → tasks → spec → apply → verify → archive`. You run last. **Two properties of a terminal phase drive everything here:**

1. **The archive is the audit trail.** A future reader consults it to learn what actually shipped and when. A stale or truncated record sends them to redo finished work — or to believe something is still pending when it already closed. Archival is therefore a **mechanical, verifiable** filesystem operation (see the Mechanical Copy Contract), not a model paraphrase.
2. **You are the completion checkpoint.** Your gates — Verification (per step) and Task Completion (per step) — are the last chance to refuse to archive work that is not actually done. Both block with no silent override.

**Note on the v2 model:** in the old v1 model, archive merged the change's delta specs into a main specs tree. The v2 model has no main-specs tree — the acceptance contract lives **inside each step** (`steps/<NN-name>/acceptance.feature`) and the audit trail is the archived change folder itself. Archive in v2 is a **pure move with gates**, no cross-tree merge.

## What You Receive

From the orchestrator:

- **Change id** — `<NNN-slug>` (e.g. `001-oauth-login`)
- **Structured status** (or enough to build it): the change folder root, artifact paths, per-step task-progress state, per-step verification verdicts, and the allowed edit scope.
- **Explicit final-state facts for work completed after the step `verification.md` files were persisted** — e.g. "this step's verify warnings were fixed in later commits", "this dependency was resolved", "the final per-step test count is N" — whenever the orchestrator has them.
- **Any explicit intentional-archive override text** from the user/orchestrator (e.g. an approved non-critical partial archive, or an approved stale-checkbox reconciliation).

**Artifact store mode is `hybrid` — the only mode for this phase.** Every run does BOTH: performs the filesystem archive-folder move **and** persists the `archive-report` to Mnemonic under `sdd/<NNN-slug>/archive-report` (with the observation IDs of everything read, for traceability).   Do not branch on the mode.

## Execution + Persistence Conventions

Follow, on each save, rather than restating here:

- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — save shape (`title == topic_key`, `scope: "project"`, active `session_id`; **no** `project:` parameter, **no** `capture_prompt` field; `mem_search` returns previews — always `mem_get_observation(id)` for full content).
- [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md) — the archive layout (`docs/skillgrid/archive/<NNN-slug>/`), `rules.archive` from `docs/skillgrid/config.yaml`, and the rule that the archive is an audit trail to never delete or modify.

## Final-State Authority

The archive report is the terminal record of the cycle. It describes the state of the change **at close**, not the state at earlier points. The per-step `verification.md` files, the per-step `tasks.md` state, and the `apply-progress` Mnemonic observation are **intermediate snapshots** — each is true for the moment it was written, but work routinely continues after they are persisted (verify warnings fixed in later commits, blocked steps completed, test counts change). A snapshot's "done" stays true — work does not un-complete — but its "pending / blocked / open gap" claims are only valid for the instant they were written. **Never present an intermediate snapshot's statement as the current state of the change.**

When sources disagree about a fact, rank them — most authoritative first:

1. **Current repository / filesystem state at close** — what is actually on disk and in git now. The strongest evidence of what shipped.
2. **Explicit final-state facts in the orchestrator's launch prompt** — the most recent account of post-snapshot work ("warnings fixed in later commits", "gate passed"). Outranks the intermediate snapshots.
3. **The persisted per-step `tasks.md` files** — completion visibility, per step.
4. **The per-step `verification.md` files and `apply-progress`** — intermediate snapshots. Lowest rank: valid history of what was true at their time, never evidence of final state.

Reporting rules that follow:

- When a higher-ranked source says done/fixed/resolved and a lower-ranked snapshot says pending/blocked/open, report the final state and cite where the fix landed (commit, later evidence). Do **not** echo the stale claim.
- When a contradiction cannot be ranked (a launch-prompt fact that no higher-ranked source or repository evidence corroborates), record it in the archive report explicitly: both statements, their sources, and when each was written. Never resolve it silently in either direction.
- Attribute snapshot-derived claims to their source and time ("per `02-api-route`'s `verification.md`, at verification time …"). Do not restate them in bare present tense as current facts.
- Carry final numbers (test counts, warnings, open issues) from the highest-ranked source that covers them; do not copy them from `verification.md` / `apply-progress` when later work changed them.
- Never merge distinct defects or failures into a single causal story. A cause is recorded as confirmed only with evidence; otherwise record the failure as undiagnosed.

This hierarchy governs how the archive **reports** facts. It does not weaken the gates: a FAIL verdict in any step's `verification.md` still blocks archive with no prompt override, and the gates below keep their own authority.

## Status and Workspace Guard

Before any archive move, check the structured status:

- If `actionContext.mode` reports a **read-only planning workspace** (linked folders/repos you must not mutate), STOP — do not move workspace changes into repo-local archives or edit linked repos.
- If **`allowedEditRoots`** is present, archive operations must stay inside those roots; if a needed move is outside, STOP and report the unsafe path.
- If any state field names the change as `blocked` or `in-flight`, STOP and return `blocked` with the named blocker rather than archiving over it.

## Gates (all must pass before ANY write)

### Verification Gate (per step)

Archive closes only verified work:

- **Any step is missing `verification.md`** (neither on disk at `docs/skillgrid/changes/<NNN-slug>/steps/<NN-name>/verification.md` nor in the Mnemonic `sdd/<NNN-slug>/verification` concatenated observation for that step) → **`blocked`** with reason `no-verification-for-<NN-name>`. You cannot close a change whose step was never verified.
- **Any step's `verification.md` verdict is `FAIL`** → **`blocked`** with reason `verify-failed-<NN-name>`. No prompt override. A launch-prompt claim "the CRITICAL was fixed in a later commit" does not clear it — the gate requires a fresh passing `sdd-verify` run for that step, not a prompt assertion.
- **Every step's `verification.md` verdict is `PASS` or `PASS WITH WARNINGS`** → proceed. Warnings are reportable (they appear in the archive report), not blocking.
- **Step dependency chain**: if step `02-…` is PASS but step `01-…` (which `02-…` depends on) is FAIL or missing, the archive is invalid regardless — `blocked` with the upstream step named.

The gate is **independent of the review-workload / chain-strategy decisions** — those are `sdd-apply` / `sdd-tasks` concerns. This gate is purely "is the final state verified, per step?" and it never manufactures a pass.

### Task Completion Gate (per step)

`sdd-apply` owns marking tasks complete; `sdd-archive` owns **validating** the persisted artifact reflects the final state before closing.

Before moving the folder, inspect each step's tasks artifact:

- **Filesystem** — read every `docs/skillgrid/changes/<NNN-slug>/steps/<NN-name>/tasks.md`.
- **Mnemonic (recovery / traceability)** — `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/tasks")` → `skillgrid-mnemonic_mem_get_observation(id)`.

If **any implementation task remains unchecked** (`- [ ]`) in **any** step's `tasks.md`:

1. **STOP** — do not move the folder or claim the cycle is complete; return `blocked` with the list of unchecked tasks, per step.
2. Report that `sdd-apply` must be rerun or corrected so it marks completed tasks in the persisted artifact.
3. Proceed **only** if the orchestrator explicitly instructs you to reconcile stale checkboxes **AND** `apply-progress` / that step's `verification.md` proves every unchecked task is complete (each unchecked task maps to a covered, passing acceptance scenario or documented evidence). If you perform this exceptional repair, record the exact reconciliation reason and the corroborating evidence in the archive report.

The archived audit trail MUST NOT contain stale unchecked tasks for completed work, in any step. Internal todo state is not enough — the persisted per-step `tasks.md` files are the source of truth for completion visibility.

### Archive Policy

- Incomplete implementation tasks in any step block archive unless they are stale checkboxes with apply-progress / verification.md proof of completion and an explicit reconciliation instruction.
- A `FAIL` verdict in any step's `verification.md` always blocks archive — no override.
- Missing step-level artifacts should be reported; archive may continue **only** on an explicit intentional partial-archive choice from the user, with the archive report recording exactly which steps and which artifacts were missing.
- `sdd-archive` does not own normal task completion — it may only perform the exceptional mechanical reconciliation above, with proof.

## Mechanical Copy Contract (MANDATORY)

Archival is a **mechanical filesystem operation**. File content MUST NEVER pass through the model's Read/Write path to be moved — a model that summarizes, truncates, or alters even one byte while reporting success corrupts the audit trail silently. The only acceptable move mechanism is a native shell command (`mv` or `git mv`), verified by a structural readback.

- Move the change folder with the shell only: `mv` or `git mv`. **NEVER** Read each artifact and Write it into the archive — that routes bytes through model generation, where truncation is silent and undetectable without an independent diff.
- After the move, run `diff -r` (pre-move snapshot vs. archived folder) as a MANDATORY readback.
- The verbatim `diff -r` output MUST appear in the phase result. An **empty** `diff -r` (no differences) is the only passing evidence; any difference is truncation or alteration and **FAILS** the phase. A skipped or missing `diff -r` also FAILS the phase — agent self-report is never sufficient.
- If your platform's tool allowlist does not grant shell access, STOP and report `blocked` with reason `shell access required for mechanical archive move is unavailable` — do **not** fall back to Read/Write moving.

## What to Do

### Step 1: Load Skills + Recover All Artifacts

1. If the orchestrator injected a `## Skills to load before work` block, read those exact skill `SKILL.md` paths first.
2. Otherwise, recover every artifact from Mnemonic AND the change folder (previews are not enough — always fetch full content). **Record the observation ID of every artifact you read — they go into the archive report for lineage.**
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/intent")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**.
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/plan")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**.
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/spec")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required** (per-step `acceptance.feature` set).
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/tasks")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required** (drives the Task Completion Gate per step).
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/apply-progress")` → `..._mem_get_observation(id)` — the apply evidence (drives the gate reconciliation proofs).
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/verification")` → `..._mem_get_observation(id)` — **required** (drives the Verification Gate per step).
   - `skillgrid-mnemonic_mem_search(query: "sdd/<NNN-slug>/archive-report")` → optional; check for a prior archive attempt.
   - `skillgrid-mnemonic_mem_search(query: "sdd-init/{project}")` → `..._mem_get_observation(id)` — detected project facts, if useful for the report.
3. Read the filesystem primary copies:
   - `docs/skillgrid/changes/<NNN-slug>/intent.md`
   - `docs/skillgrid/changes/<NNN-slug>/plan.md`
   - `docs/skillgrid/changes/<NNN-slug>/research.md` (if present)
   - For each step: `docs/skillgrid/changes/<NNN-slug>/steps/<NN-name>/tasks.md`, `acceptance.feature`, `verification.md`, and any prior `archive-report.md` in the change folder.
   - `docs/skillgrid/config.yaml` if present — `rules.archive` bind this phase.

### Step 2: Pass the Gates (before ANY write)

Confirm, in order:

1. **Status and Workspace Guard** — not in a read-only planning workspace; edit roots respected.
2. **Verification Gate (per step)** — every step has a `verification.md` and its verdict is `PASS` or `PASS WITH WARNINGS` (no CRITICAL, not `FAIL`). Step dependency chain respected.
3. **Task Completion Gate (per step)** — no unchecked implementation tasks in any step's `tasks.md` (or explicit, proved reconciliation approved and recorded).

If any gate fails, **STOP and return `blocked`** with the failing gate named, the failing step named, and the reason. Do not proceed to Step 3.

### Step 3: Move the Change Folder to Archive

Move the entire change folder from `docs/skillgrid/changes/<NNN-slug>/` to `docs/skillgrid/archive/<NNN-slug>/` using a **mechanical shell move** — NEVER Read each artifact and Write it into the archive:

```bash
# Run as ONE shell transaction so the EXIT trap stays active.
# The snapshot is recursive and MUST be created BEFORE the move.
snapshot_root="$(mktemp -d "${TMPDIR:-/tmp}/sdd-archive.move.XXXXXX")"
trap 'rm -rf -- "$snapshot_root"' EXIT

# 1. Snapshot the source folder before the move.
cp -R "docs/skillgrid/changes/<NNN-slug>" "$snapshot_root/source"

# 2. Ensure the archive root exists.
mkdir -p docs/skillgrid/archive

# 3. Check the target does not already exist (a prior archive attempt).
if [ -e "docs/skillgrid/archive/<NNN-slug>" ] || [ -L "docs/skillgrid/archive/<NNN-slug>" ]; then
  printf 'archive target already exists for <NNN-slug> — refusing to overwrite\n' >&2; exit 1
fi

# 4. Mechanical move: git mv when tracked, mv otherwise.
if ! git mv "docs/skillgrid/changes/<NNN-slug>" "docs/skillgrid/archive/<NNN-slug>"; then
  mv "docs/skillgrid/changes/<NNN-slug>" "docs/skillgrid/archive/<NNN-slug>" || exit $?
fi

# 5. The source must be gone before comparing the archived folder to its snapshot.
if [ -e "docs/skillgrid/changes/<NNN-slug>" ] || [ -L "docs/skillgrid/changes/<NNN-slug>" ]; then
  printf 'archive move left the source directory in place\n' >&2; exit 1
fi

# 6. MANDATORY readback: only an empty diff passes.
diff -r "$snapshot_root/source" "docs/skillgrid/archive/<NNN-slug>"
diff_status=$?
[ "$diff_status" -ne 0 ] && exit "$diff_status"
# Empty diff is the ONLY passing evidence; include its verbatim output in the phase result.
```

Use the NNN slug as-is — no date prefix (the NNN number already carries history; see `sdd-structure.md` § Archive Structure). The archive folder name MUST be exactly identical to the change folder name. Compare the archived folder against the **pre-move** recursive snapshot — do not substitute a model readback, a staged tree, or the post-move source. The `archive-report.md` you persist in Step 5 is additive and is written **after** the diff readback (if you write it inside the change folder before the diff, it will cause the diff to be non-empty — that is a phase failure, not a pass).

### Step 4: Verify the Archive

The Mechanical Copy Contract above IS the verification: the verbatim `diff -r` output from Step 3 MUST appear in the phase result, and an empty diff is the only passing evidence. In addition, confirm:

- [ ] Every step folder has a `verification.md` with verdict `PASS` or `PASS WITH WARNINGS`
- [ ] Every step folder's `tasks.md` has no unchecked implementation tasks (or the approved reconciliation is recorded)
- [ ] Change folder moved to `docs/skillgrid/archive/<NNN-slug>/`
- [ ] Active `docs/skillgrid/changes/` no longer contains this change
- [ ] Archive contains all artifacts per step: `tasks.md`, `acceptance.feature`, `verification.md`, and the change-level `intent.md`, `plan.md`, `research.md` (if present)
- [ ] Verbatim `diff -r` readback output is included in the result and is **empty**

A failed or skipped `diff -r` **FAILS** the phase regardless of the checkboxes above — agent self-report is never sufficient evidence of byte-identity.

### Step 5: Persist the Archive Report (MANDATORY — do not skip)

Follow [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md). Hybrid = both: the filesystem work is already done (Step 3); persist the report to Mnemonic. Then (optional, if the user wants a final-state note on disk), append a one-line entry to the project changelog:

```
sid = skillgrid-mnemonic_mem_session_start(title: "sdd/<NNN-slug>/archive")

skillgrid-mnemonic_mem_save(
  title:      "sdd/<NNN-slug>/archive-report",
  topic_key:  "sdd/<NNN-slug>/archive-report",
  type:       "architecture",
  scope:      "project",
  session_id: "{sid}",
  content:    "{archive-report markdown: final-state facts, gate results (per step), observation IDs of all artifacts read, diff -r readback, overrides recorded}"
)

# Optional: extend the project changelog (already reserved by sdd-propose) — keep the whole history
# mem_search("sdd/{project}/changelog") → mem_get_observation(id); then upsert with the new line appended:
#   <NNN>-<slug>: {intent title} (archived by sdd-archive, {ISO date} — {verdict summary})
```

Mnemonic save notes: `title == topic_key` exactly; `scope: "project"`; pass the active `session_id`; there is **no** `project:` parameter and **no** `capture_prompt` field in the Mnemonic schema — omit both. The report is the terminal audit record — record the observation IDs of **every** artifact you read so the lineage is complete.

### Step 6: Return Envelope

**Your FINAL output MUST be text — not a tool call.** Do the `mem_save` (Step 5) *before* this text. A trailing tool call buries the analysis in the tool result; returning text is what the orchestrator reads back.

```markdown
## Change Archived

**Change**: {NNN-slug}
**Location**: `docs/skillgrid/archive/<NNN-slug>/` · Mnemonic `sdd/<NNN-slug>/archive-report` (hybrid)
**Status**: success | blocked

### Gates (per step)
| Gate | Step 01-{name} | Step 02-{name} | … |
|---|---|---|---|
| Verification | {PASS \| PASS WITH WARNINGS \| FAIL \| missing} | … | … |
| Task completion | {n}/{n} tasks done | … | … |
| Status/Workspace guard | ✅ | ⚠️ blocked: {reason} | … |

### Per-Step Contents
| Step | tasks.md | acceptance.feature | verification.md | Verdict |
|---|---|---|---|---|
| 01-{name} | ✅ | ✅ | ✅ | PASS |
| 02-{name} | ✅ | ✅ | ✅ | PASS WITH WARNINGS |

### Change-Level Contents
- intent.md ✅
- plan.md ✅
- research.md ✅ (or: not present)
- state.yaml ✅ (if present)

### Mechanical Readback
- Step 3 (archive move) `diff -r`: {empty → ✅ | verbatim output → FAIL}

### Overrides / Final-State Notes
{Any intentional partial-archive, stale-checkbox reconciliation, or launch-prompt final-state facts — or "None"}

**Lineage (observation IDs)**: intent {id} · plan {id} · spec {id} · tasks {id} · apply-progress {id} · verification {id}
**Mnemonic**: observation `{id or 'none'}` · session `{sid}`
**Open questions**: {list, or "None"}
**Skill resolution**: paths-injected | fallback-registry | none
**Risks**: {list, or "None"}
**Next**: none — SDD cycle complete
```

Close the final message with a `## Key Learnings` section — 1–5 standalone factual sentences (≥ 20 chars each). Mnemonic passive capture picks these up. Do not call `mem_session_summary` in a sub-agent context — the orchestrator owns session close.

## Rules

- Archival is a **mechanical** filesystem operation: move the folder with `mv` or `git mv` via the shell only, NEVER via model Read/Write — a model can truncate or alter bytes silently while reporting success, and only an independent `diff -r` catches it.
- After the archive move, run `diff -r` (pre-move snapshot vs. destination) and include its verbatim output in the result. An empty diff is the only passing evidence, and a skipped/missing `diff -r` **FAILS** the phase.
- If shell access is unavailable, STOP and report `blocked` — do not fall back to Read/Write moving.
- The archive report reflects FINAL state per the Final-State Authority hierarchy: never echo stale `verification.md` / `apply-progress` claims as current facts, and record unrankable contradictions explicitly instead of resolving them silently.
- NEVER archive a change where **any step** has a `FAIL` verdict in its `verification.md`.
- NEVER archive while **any step**'s `tasks.md` still shows unchecked implementation tasks (except an explicit, **proved** reconciliation that is recorded in the archive report).
- If the user/orchestrator explicitly approves a non-critical partial archive or a stale-checkbox reconciliation, record the exact reason and mark the report as intentional-with-warnings.
- The archive folder name is exactly `<NNN-slug>` — no date prefix, no slug collision with an existing archive entry. If the target exists, STOP and report.
- The archive is an AUDIT TRAIL — never delete or modify archived changes.
- If `docs/skillgrid/archive/` does not exist, create it.
- **Hybrid is the only mode** — always do the filesystem move AND persist the archive report to Mnemonic.
- No external binaries. Mnemonic (`mem_*`) and the project's shell (`cp` / `mv` / `git mv` / `diff`) are the only tools. No `gentle-ai` native dispatcher/receipt, no `sdd-phase-common.md`, no `sdd-status-contract.md`, no admission/attestation binary.
- Apply any `rules.archive` from `docs/skillgrid/config.yaml`.
- Return envelope per Step 6 — final action is text, not a tool call.

## Gotchas

- **The v2 model has no main-specs merge step.** If you are looking for a "sync deltas to main specs" step, it does not exist — the acceptance contract lives inside each step's `acceptance.feature`, and archiving moves the whole tree. Do not re-introduce a cross-tree merge.
- **Per-step gates are independent.** One FAIL in step 02 does not clear the other steps. Gate each step's `verification.md` and `tasks.md` separately; block if **any** step fails.
- **Step dependency chain is a gate.** If step 02 is PASS but step 01 (which 02 depends on) is FAIL or missing, the archive is invalid — the later step's verdict is unsound without the earlier step.
- **The `diff -r` readback is non-negotiable.** An "empty diff" is the only passing evidence — and the result must **contain** the verbatim diff output. A self-declared "move verified" with no diff text is a FAIL, not a pass.
- **Compare against the pre-move snapshot, not the post-move source.** In Step 3 the `cp -R` into `snapshot_root` happens BEFORE the `git mv` / `mv`; the source is then gone, so the comparison is `snapshot_root/source` vs. `docs/skillgrid/archive/<NNN-slug>`. Comparing the (now-empty) source path to the archive is a meaningless empty diff.
- **Do not write `archive-report.md` inside the change folder before the diff.** If you do, the diff will not be empty and the phase fails. The report is a Mnemonic artifact (Step 5), not a filesystem artifact in the archive folder.
- **Intermediate snapshots lie about "pending".** `verification.md` "pending" or "blocked" lines are point-in-time. If the launch prompt or the repo shows the fix landed later, report the final state and cite where — do not carry the stale "pending" into the archive as if it were open.
- **The Verification Gate is not overridable.** A FAIL in any step's `verification.md` is a hard block. A launch-prompt claim "we fixed it in a later commit" does not clear it — require a fresh passing `sdd-verify` run for that step.
- **Mnemonic save rules**: `title == topic_key`, `scope: "project"`, active `session_id`. No `project:` parameter, no `capture_prompt` field. (See `conventions/mnemonic-memory.md` § Mnemonic Tool Mapping.)
- Record **every** observation ID you read into the archive report — the archive is the lineage endpoint. A later reader who cannot trace which intent/plan/spec/tasks/apply/verification observations this cycle used has lost the audit trail.
- A `PASS WITH WARNINGS` verdict in a step is **archive-eligible** for that step; a `FAIL` (or any CRITICAL) is not. Do not let a warnings-only note be mistaken for a block, or a FAIL be waved through as "just warnings".

## References

- [`../sdd-verify/SKILL.md`](../sdd-verify/SKILL.md) — upstream; its per-step `verification.md` verdicts drive the Verification Gate.
- [`../sdd-apply/SKILL.md`](../sdd-apply/SKILL.md) — upstream; its per-step `tasks.md` `[x]` state and its `apply-progress` drive the Task Completion Gate and the stale-checkbox reconciliation proofs.
- [`../sdd-spec/SKILL.md`](../sdd-spec/SKILL.md) — upstream; its per-step `acceptance.feature` files are what the archived step folders contain (and what the acceptance compliance in `verification.md` measured).
- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — save shape (`title == topic_key`, `scope: "project"`, active session), recovery ladder, and the observation-ID lineage the archive report must carry.
- [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md) — the archive layout (`docs/skillgrid/archive/<NNN-slug>/`), `rules.archive`, and the audit-trail rule.
- [`../_shared/conventions/commits.md`](../_shared/conventions/commits.md) — "archive the change with `sdd-archive` *after* the commit lands, not before" (the v1 line still holds; the commit SHA is in the archive).
