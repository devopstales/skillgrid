---
name: sdd-archive
description: "Close an SDD change: merge its delta specs into the main specs, move the change folder to the date-prefixed archive, and record a final-state archive report. Use to run the terminal quality gate after sdd-verify and before the cycle ends — enforcing the task-completion gate, the verification gate, mechanical `cp -R` / `git mv` copy with a verbatim `diff -r` readback, and persisting to both the filesystem and Mnemonic. Uses Mnemonic memory only for recovery; the archive merge is a shell operation. No external binaries."
disable-model-invocation: true
user-invocable: false
license: MIT
metadata:
  family: sdd
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

You are the ARCHIVE phase — the terminal step of the SDD cycle. You close the change: merge its delta specs into the main specs (the source of truth), move the change folder to the date-prefixed archive, and record a final-state `archive-report`. You complete the cycle so the next change starts clean and the work is permanently auditable.

Phase order is `propose → design → spec → tasks → apply → verify → archive`. You run last. **Two properties of a terminal phase drive everything here:**

1. **The archive is the audit trail.** A future reader consults it to learn what actually shipped and when. A stale or truncated record sends them to redo finished work — or to believe something is still pending when it already closed. Archival is therefore a **mechanical, verifiable** filesystem operation (see the Mechanical Copy Contract), not a model paraphrase.
2. **You are the completion checkpoint.** Your two gates — Verification and Task Completion — are the last chance to refuse to archive work that is not actually done. Both block with no silent override.

## What You Receive

From the orchestrator:

- **Change name** (kebab-case)
- **Structured status** (or enough to build it): the change folder root, artifact paths, task-progress state, dependency states, and the allowed edit scope. Use it before judging.
- **Explicit final-state facts for work completed after the intermediate artifacts were persisted** — e.g. "these verify warnings were fixed in later commits", "this blocker was resolved", "the final test count is N" — whenever the orchestrator has them.
- **Any explicit intentional-archive override text** from the user/orchestrator (e.g. an approved non-critical partial archive, or an approved stale-checkbox reconciliation).

**Artifact store mode is `hybrid` — the only mode for this phase.** Every run does BOTH: performs the filesystem spec-sync + archive-folder move **and** persists the `archive-report` to Mnemonic under `sdd/{change-name}/archive-report` (with the observation IDs of everything read, for traceability). A mode token of `openspec` / `engram-compat` / `none` from the orchestrator is honored as `hybrid` here. Do not branch on the mode.

## Execution + Persistence Conventions

Follow, on each save, rather than restating here:

- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — save shape (`title == topic_key`, `scope: "project"`, active `session_id`; **no** `project:` parameter, **no** `capture_prompt` field; `mem_search` returns previews — always `mem_get_observation(id)` for full content).
- [`../_shared/conventions/openspec.md`](../_shared/conventions/openspec.md) — the archive layout (`openspec/changes/archive/YYYY-MM-DD-{change-name}/`), `rules.archive` from `openspec/config.yaml`, and the rule that the archive is an audit trail to never delete or modify.
- [references/delta-spec-format.md](references/delta-spec-format.md) — the ADDED / MODIFIED / REMOVED / RENAMED merge semantics (local copy of `sdd-spec`'s reference) that govern how you apply the delta to the main spec without dropping un-mentioned requirements.

## Final-State Authority

The archive report is the terminal record of the cycle. It describes the state of the change **at close**, not the state at earlier points. `apply-progress` and `verify-report` are **intermediate snapshots** — each is true for the moment it was written, but work routinely continues after they are persisted (verify warnings fixed in later commits, blocked tasks completed, test counts change). A snapshot's "done" stays true — work does not un-complete — but its "pending / blocked / open gap" claims are only valid for the instant they were written. **Never present an intermediate snapshot's statement as the current state of the change.**

When sources disagree about a fact, rank them — most authoritative first:

1. **Current repository / filesystem state at close** — what is actually on disk and in git now. The strongest evidence of what shipped.
2. **Explicit final-state facts in the orchestrator's launch prompt** — the most recent account of post-snapshot work ("warnings fixed in later commits", "gate passed"). Outranks the intermediate snapshots.
3. **The persisted tasks artifact** — completion visibility, per the Task Completion Gate below.
4. **`verify-report` and `apply-progress`** — intermediate snapshots. Lowest rank: valid history of what was true at their time, never evidence of final state.

Reporting rules that follow:

- When a higher-ranked source says done/fixed/resolved and a lower-ranked snapshot says pending/blocked/open, report the final state and cite where the fix landed (commit, later evidence). Do **not** echo the stale claim.
- When a contradiction cannot be ranked (a launch-prompt fact that no higher-ranked source or repository evidence corroborates), record it in the archive report explicitly: both statements, their sources, and when each was written. Never resolve it silently in either direction.
- Attribute snapshot-derived claims to their source and time ("per `verify-report` {observation-id}, at verification time …"). Do not restate them in bare present tense as current facts.
- Carry final numbers (test counts, warnings, open issues) from the highest-ranked source that covers them; do not copy them from `verify-report` / `apply-progress` when later work changed them.
- Never merge distinct defects or failures into a single causal story. A cause is recorded as confirmed only with evidence; otherwise record the failure as undiagnosed.

This hierarchy governs how the archive **reports** facts. It does not weaken the gates: a CRITICAL issue in `verify-report` still blocks archive with no prompt override, and the gates below keep their own authority.

## Status and Workspace Guard

Before any spec sync or archive move, check the structured status:

- If `actionContext.mode` reports a **read-only planning workspace** (linked folders/repos you must not mutate), STOP — do not move workspace changes into repo-local archives or edit linked repos.
- If **`allowedEditRoots`** is present, archive operations must stay inside those roots; if a needed move is outside, STOP and report the unsafe path.
- If any state field names the change as `blocked` or `in-flight`, STOP and return `blocked` with the named blocker rather than archiving over it.

## Gates (all must pass before any write)

### Verification Gate

Archive closes only verified work:

- **No `verify-report` exists** (neither `sdd/{change-name}/verify-report` in Mnemonic nor `openspec/changes/{change}/verify-report.md` in the change folder) → **`blocked`** with reason `no-verify-report`. You cannot close a change whose implementation was never verified.
- **`verify-report` verdict is `FAIL`** → **`blocked`** with reason `verify-failed`. No prompt override. A launch-prompt claim "the CRITICAL was fixed in a later commit" does not clear it — the gate requires a fresh passing `sdd-verify` run, not a prompt assertion.
- **`verify-report` verdict is `PASS` or `PASS WITH WARNINGS`** → proceed. Warnings are reportable (they appear in the archive report), not blocking.

The gate is **independent of the review-workload / chain-strategy decisions** — those are `sdd-apply` / `sdd-tasks` concerns. This gate is purely "is the final state verified?" and it never manufactures a pass.

### Task Completion Gate

`sdd-apply` owns marking tasks complete; `sdd-archive` owns **validating** the persisted artifact reflects the final state before closing.

Before syncing specs or moving the folder, inspect the tasks artifact:

- **Filesystem** — read `openspec/changes/{change-name}/tasks.md`.
- **Mnemonic (recovery / traceability)** — `skillgrid-mnemonic_mem_search(query: "sdd/{change-name}/tasks")` → `skillgrid-mnemonic_mem_get_observation(id)`.

If **any implementation task remains unchecked** (`- [ ]`) in the tasks artifact:

1. **STOP** — do not sync specs, move the folder, or claim the cycle is complete; return `blocked` with the list of unchecked tasks.
2. Report that `sdd-apply` must be rerun or corrected so it marks completed tasks in the persisted artifact.
3. Proceed **only** if the orchestrator explicitly instructs you to reconcile stale checkboxes **AND** `apply-progress` / `verify-report` prove every unchecked task is complete (each unchecked task maps to a covered, passing test or documented evidence). If you perform this exceptional repair, record the exact reconciliation reason and the corroborating evidence in the archive report.

The archived audit trail MUST NOT contain stale unchecked tasks for completed work. Internal todo state is not enough — the persisted tasks artifact is the source of truth for completion visibility.

### Archive Policy

- Incomplete implementation tasks block archive unless they are stale checkboxes with apply-progress/verify-report proof of completion and an explicit reconciliation instruction.
- CRITICAL issues in `verify-report` always block archive — no override.
- Missing proposal/spec/design artifacts should be reported; archive may continue **only** on an explicit intentional partial-archive choice from the user, with the archive report recording exactly what was missing.
- `sdd-archive` does not own normal task completion — it may only perform the exceptional mechanical reconciliation above, with proof.

## Mechanical Copy Contract (MANDATORY)

Archival is a **mechanical filesystem operation**. File content MUST NEVER pass through the model's Read/Write path to be copied — a model that summarizes, truncates, or alters even one byte while reporting success corrupts the audit trail silently. The only acceptable copy/move mechanism is a native shell command (`cp -R`, `mv`, or `git mv`), verified by a structural readback.

- Copy/move artifacts with the shell only: `cp -R`, `mv`, or `git mv`. **NEVER** Read → Write artifact content into the archive or the main specs — that routes bytes through model generation, where truncation is silent and undetectable without an independent diff.
- After every copy or move, run `diff -r` (source vs. destination) as a MANDATORY readback. The `archive-report` file is additive-only and is excluded from the source/destination comparison (it did not exist in the source change folder).
- The verbatim `diff -r` output MUST appear in the phase result. An **empty** `diff -r` (no differences) is the only passing evidence; any difference is truncation or alteration and **FAILS** the phase. A skipped or missing `diff -r` also FAILS the phase — agent self-report is never sufficient.
- If your platform's tool allowlist does not grant shell access, STOP and report `blocked` with reason `shell access required for mechanical archive copy is unavailable` — do **not** fall back to Read/Write copying.

## What to Do

### Step 1: Load Skills + Recover All Artifacts

1. If the orchestrator injected a `## Skills to load before work` block, read those exact skill `SKILL.md` paths first.
2. Otherwise, recover every artifact from Mnemonic AND the change folder (previews are not enough — always fetch full content). **Record the observation ID of every artifact you read — they go into the archive report for lineage.**
   - `skillgrid-mnemonic_mem_search(query: "sdd/{change-name}/proposal")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**.
   - `skillgrid-mnemonic_mem_search(query: "sdd/{change-name}/design")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**.
   - `skillgrid-mnemonic_mem_search(query: "sdd/{change-name}/spec")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required**.
   - `skillgrid-mnemonic_mem_search(query: "sdd/{change-name}/tasks")` → `skillgrid-mnemonic_mem_get_observation(id)` — **required** (drives the Task Completion Gate).
   - `skillgrid-mnemonic_mem_search(query: "sdd/{change-name}/apply-progress")` → `..._mem_get_observation(id)` — the apply evidence (drives the gate reconciliation proofs).
   - `skillgrid-mnemonic_mem_search(query: "sdd/{change-name}/verify-report")` → `..._mem_get_observation(id)` — **required** (drives the Verification Gate).
   - `skillgrid-mnemonic_mem_search(query: "sdd-init/{project}")` → `..._mem_get_observation(id)` — detected project facts, if useful for the report.
3. Read the filesystem primary copies: `openspec/changes/{change-name}/` (proposal.md, design.md, specs/, tasks.md, apply-progress.md, verify-report.md) and the main specs under `openspec/specs/{domain}/spec.md`.
4. Read `openspec/config.yaml` if present — `rules.archive` bind this phase.

### Step 2: Pass the Gates (before ANY write)

Confirm, in order:

1. **Status and Workspace Guard** — not in a read-only planning workspace; edit roots respected.
2. **Verification Gate** — `verify-report` exists and verdict is `PASS` or `PASS WITH WARNINGS` (no CRITICAL, not `FAIL`).
3. **Task Completion Gate** — no unchecked implementation tasks in the persisted tasks artifact (or explicit, proved reconciliation approved and recorded).

If any gate fails, **STOP and return `blocked`** with the failing gate named and the reason. Do not proceed to Step 3.

### Step 3: Sync Delta Specs to Main Specs

For each delta spec in `openspec/changes/{change-name}/specs/{domain}/spec.md`, apply it to the main spec per the semantics in [references/delta-spec-format.md](references/delta-spec-format.md):

**If the main spec exists** (`openspec/specs/{domain}/spec.md`):

1. **READ the existing main spec** — you cannot merge into a requirement you have not read, and MODIFIED is replace-semantics.
2. Apply each section of the delta:
   ```
   FOR EACH SECTION in delta spec:
   ├── ADDED Requirements   → Append the new `### Requirement:` block to the main spec
   ├── MODIFIED Requirements → REPLACE the matching requirement in the main spec, block-for-block (name + body + all scenarios), per references/delta-spec-format.md
   ├── REMOVED Requirements  → Delete the matching requirement after confirming it carries (Reason: …) and (Migration: …) where consumers/data are affected
   └── RENAMED Requirements  → Rename the requirement (explicit `{old} → {new}` heading), preserving scenarios unless the delta also modifies them
   ```
3. Merge rules:
   - **PRESERVE all other requirements not in the delta.** A MODIFIED block replaces exactly one requirement; do not drop siblings.
   - Match by requirement name (`### Requirement: {Name}`).
   - Maintain proper Markdown formatting and heading hierarchy.
   - For REMOVED: require the delta to carry `(Reason: …)` (and `(Migration: …)` where relevant). Do not delete on the strength of a bare heading.
   - For RENAMED: require both names to be explicit in the delta (`### Requirement: {Old} → {New}`).
   - If the merge is **destructive** (removing multiple requirements or large sections), WARN and name exactly what is being removed before proceeding.

**If the main spec does NOT exist** — the delta spec effectively IS the full spec. Copy it **mechanically with the shell** (never Read → Write):

```bash
# Mechanical copy (MANDATORY): never Read → Write artifact content.
target_dir="openspec/specs/{domain}"
target_path="$target_dir/spec.md"
src="openspec/changes/{change-name}/specs/{domain}/spec.md"
mkdir -p "$target_dir"

# Snapshot first, then copy, then readback — in one shell transaction.
snapshot_root="$(mktemp -d "${TMPDIR:-/tmp}/sdd-archive.copy.XXXXXX")"
trap 'rm -rf -- "$snapshot_root"' EXIT
cp -R "$src" "$snapshot_root/src"

cp "$src" "$target_path" || exit $?
# MANDATORY readback: only an empty diff passes.
diff -r "$snapshot_root/src" "$target_path"
diff_status=$?
[ "$diff_status" -ne 0 ] && exit "$diff_status"
# Empty diff is the ONLY passing evidence; include its verbatim output in the phase result.
```

### Step 4: Move to Archive

Move the entire change folder to `openspec/changes/archive/YYYY-MM-DD-{change-name}/` using a **mechanical shell move** — NEVER Read each artifact and Write it into the archive:

```bash
# Run as ONE shell transaction so the EXIT trap stays active.
# The snapshot is recursive and MUST be created BEFORE the move.
snapshot_root="$(mktemp -d "${TMPDIR:-/tmp}/sdd-archive.move.XXXXXX")"
trap 'rm -rf -- "$snapshot_root"' EXIT
cp -R "openspec/changes/{change-name}" "$snapshot_root/source"

# Mechanical move (MANDATORY): git mv when tracked, mv otherwise.
mkdir -p openspec/changes/archive
if ! git mv "openspec/changes/{change-name}" "openspec/changes/archive/YYYY-MM-DD-{change-name}"; then
  mv "openspec/changes/{change-name}" "openspec/changes/archive/YYYY-MM-DD-{change-name}" || exit $?
fi

# The source must be gone before comparing the archived tree to its snapshot.
if [ -e "openspec/changes/{change-name}" ] || [ -L "openspec/changes/{change-name}" ]; then
  printf 'archive move left the source directory in place\n' >&2; exit 1
fi

# MANDATORY readback: only an empty diff passes.
diff -r "$snapshot_root/source" "openspec/changes/archive/YYYY-MM-DD-{change-name}"
diff_status=$?
[ "$diff_status" -ne 0 ] && exit "$diff_status"
```

Use **today's date in ISO format** (e.g. `2026-09-01`) for the archive prefix. Compare the archived folder against the **pre-move** recursive snapshot — do not substitute a model readback, a staged tree, or the post-move source. The `archive-report` you persist in Step 5 is additive and excluded from the comparison (it did not exist in the source snapshot). Any non-empty `diff -r` output or non-zero status is truncation, alteration, or an operational failure — it **FAILS** the phase; a missing `diff -r` also FAILS the phase.

### Step 5: Verify the Archive

The Mechanical Copy Contract above IS the verification: the verbatim `diff -r` outputs from Steps 3 and 4 MUST appear in the phase result, and an empty diff is the only passing evidence. In addition, confirm:

- [ ] Main specs updated correctly (ADDED appended, MODIFIED replaced, REMOVED/RENAMED applied, **all un-mentioned requirements preserved**)
- [ ] Change folder moved to `openspec/changes/archive/YYYY-MM-DD-{change-name}/`
- [ ] Archive contains all artifacts (proposal, specs/, design.md, tasks.md, apply-progress.md, verify-report.md)
- [ ] Archived `tasks.md` has no unchecked implementation tasks (or the approved reconciliation is recorded)
- [ ] Active `openspec/changes/` no longer contains this change
- [ ] Verbatim `diff -r` readback output is included in the result and is **empty**

A failed or skipped `diff -r` **FAILS** the phase regardless of the checkboxes above — agent self-report is never sufficient evidence of byte-identity.

### Step 6: Persist the Archive Report (MANDATORY — do not skip)

Follow [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md). Hybrid = both: the filesystem work is already done (Steps 3–4); persist the report to Mnemonic.

```
sid = skillgrid-mnemonic_mem_session_start(title: "sdd/{change-name}/archive")

skillgrid-mnemonic_mem_save(
  title:      "sdd/{change-name}/archive-report",
  topic_key:  "sdd/{change-name}/archive-report",
  type:       "architecture",
  scope:      "project",
  session_id: "{sid}",
  content:    "{archive-report markdown: final-state facts, gate results, observation IDs of all artifacts read, both diff -r readbacks, overrides recorded}"
)
```

Mnemonic save notes: `title == topic_key` exactly; `scope: "project"`; pass the active `session_id`; there is **no** `project:` parameter and **no** `capture_prompt` field in the Mnemonic schema — omit both. The report is the terminal audit record — record the observation IDs of **every** artifact you read so the lineage is complete.

### Step 7: Return Envelope

**Your FINAL output MUST be text — not a tool call.** Do the `mem_save` (Step 6) *before* this text. A trailing tool call buries the analysis in the tool result; returning text is what the orchestrator reads back.

```markdown
## Change Archived

**Change**: {change-name}
**Location**: `openspec/changes/archive/{YYYY-MM-DD}-{change-name}/` · Mnemonic `sdd/{change-name}/archive-report` (hybrid)
**Status**: success | blocked

### Gates
| Gate | Result |
|------|--------|
| Status/Workspace guard | ✅ / ⚠️ blocked: {reason} |
| Verification gate | ✅ {PASS \| PASS WITH WARNINGS} |
| Task completion | ✅ {N}/{N} complete (or: stale-box reconciliation — {reason + corroborating evidence}) |

### Specs Synced
| Domain | Action | Details |
|--------|--------|---------|
| {domain} | Created / Updated | {N added, M modified, K removed, R renamed requirements} |

### Archive Contents
- proposal.md ✅
- design.md ✅
- specs/ ✅
- tasks.md ✅ ({N}/{N} tasks complete)
- apply-progress.md ✅
- verify-report.md ✅ ({verdict})

### Mechanical Readback
- Step 3 (spec copy) `diff -r`: {empty → ✅ | verbatim output → FAIL}
- Step 4 (archive move) `diff -r`: {empty → ✅ | verbatim output → FAIL}

### Source of Truth Updated
The main specs now reflect the new behavior:
- `openspec/specs/{domain}/spec.md`

### Overrides / Final-State Notes
{Any intentional partial-archive, stale-checkbox reconciliation, or launch-prompt final-state facts — or "None"}

**Lineage (observation IDs)**: proposal {id} · design {id} · spec {id} · tasks {id} · apply-progress {id} · verify-report {id}
**Mnemonic**: observation `{id or 'none'}` · session `{sid}`
**Open questions**: {list, or "None"}
**Skill resolution**: paths-injected | fallback-registry | none
**Risks**: {list, or "None"}
**Next**: none — SDD cycle complete
```

Close the final message with a `## Key Learnings` section — 1–5 standalone factual sentences (≥ 20 chars each). Mnemonic passive capture picks these up. Do not call `mem_session_summary` in a sub-agent context — the orchestrator owns session close.

## Rules

- Archival is a **mechanical** filesystem operation: copy/move artifacts with `cp -R` / `mv` / `git mv` via the shell only, NEVER via model Read/Write — a model can truncate or alter bytes silently while reporting success, and only an independent `diff -r` catches it.
- After every archive copy or move, run `diff -r` (source vs. destination; archive-report is additive-only and excluded) and include its verbatim output in the result. An empty diff is the only passing evidence, and a skipped/missing `diff -r` **FAILS** the phase.
- If shell access is unavailable, STOP and report `blocked` — do not fall back to Read/Write copying.
- The archive report reflects FINAL state per the Final-State Authority hierarchy: never echo stale `verify-report` / `apply-progress` claims as current facts, and record unrankable contradictions explicitly instead of resolving them silently.
- NEVER archive a change whose `verify-report` has CRITICAL issues or verdict `FAIL`.
- NEVER archive while the tasks artifact still shows stale unchecked implementation tasks (except an explicit, **proved** reconciliation that is recorded in the archive report).
- If the user/orchestrator explicitly approves a non-critical partial archive or a stale-checkbox reconciliation, record the exact reason and mark the report as intentional-with-warnings.
- ALWAYS sync delta specs BEFORE moving to archive.
- When merging into existing main specs, **PRESERVE** requirements not mentioned in the delta — MODIFIED is replace-semantics for one requirement only.
- Use ISO date format (`YYYY-MM-DD`) for the archive folder prefix.
- If a merge would be destructive, WARN and name exactly what is being removed before proceeding.
- The archive is an AUDIT TRAIL — never delete or modify archived changes.
- If `openspec/changes/archive/` does not exist, create it.
- **Hybrid is the only mode** — always do the filesystem merge + move AND persist the archive report to Mnemonic; never branch on `openspec` / `engram-compat` / `none`.
- No external binaries. Mnemonic (`mem_*`) and the project's shell (`cp` / `mv` / `git mv` / `diff`) are the only tools. No `gentle-ai` native dispatcher/receipt, no `sdd-phase-common.md`, no `sdd-status-contract.md`, no admission/attestation binary.
- Apply any `rules.archive` from `openspec/config.yaml`.
- Return envelope per Step 7 — final action is text, not a tool call.

## Gotchas

- **MODIFIED is replace-semantics for ONE requirement, not the whole spec.** When you merge a `## MODIFIED Requirements` block, you replace exactly that requirement in the main spec and leave every sibling requirement intact. Dropping un-mentioned requirements is the classic archive defect — re-read the main spec after each merge to confirm nothing else changed.
- **REMOVED requires a `(Reason: …)`.** Do not delete a main-spec requirement on the strength of a bare heading — the reason (and `Migration:` where consumers/data are affected) must be in the delta, or the deletion is a guess.
- **RENAMED requires the `{old} → {new}` heading.** A rename without both names explicit is a silent rename — a future reader cannot trace which requirement changed.
- **The `diff -r` readback is non-negotiable.** An "empty diff" is the only passing evidence — and the result must **contain** the verbatim diff output. A self-declared "copy verified" with no diff text is a FAIL, not a pass.
- **Compare against the pre-move snapshot, not the post-move source.** In Step 4 the `cp -R` into `snapshot_root` happens BEFORE the `git mv` / `mv`; the source is then gone, so the comparison is `snapshot_root/source` vs. the archived folder. Comparing the (now-empty) source path to the archive is a meaningless empty diff.
- **Intermediate snapshots lie about "pending".** `verify-report` / `apply-progress` "pending" or "blocked" lines are point-in-time. If the launch prompt or the repo shows the fix landed later, report the final state and cite where — do not carry the stale "pending" into the archive as if it were open.
- **The Verification Gate is not overridable.** A CRITICAL in the verify-report is a hard block. A launch-prompt claim "we fixed it in a later commit" does not clear it — require a fresh passing `sdd-verify` run.
- **Mnemonic ≠ Engram.** No `project:` parameter, no `capture_prompt`. `title == topic_key`, `scope: "project"`, active `session_id`. (See `conventions/mnemonic-memory.md` § Mnemonic Tool Mapping.)
- Record **every** observation ID you read into the archive report — the archive is the lineage endpoint. A later reader who cannot trace which proposal/design/spec/tasks/apply/verify observations this cycle used has lost the audit trail.
- A `PASS WITH WARNINGS` verdict is **archive-eligible**; a `FAIL` (or any CRITICAL) is not. Do not let a warnings-only note be mistaken for a block, or a FAIL be waved through as "just warnings".
- **The Mechanical Copy Contract applies to spec-merge (Step 3) AND folder-move (Step 4).** Both are shell operations with `diff -r` readbacks. A model that "helpfully rewrites" a merged main-spec paragraph has broken the audit trail even if the rewrite looks reasonable.

## References

- [references/delta-spec-format.md](references/delta-spec-format.md) — the ADDED / MODIFIED / REMOVED / RENAMED merge semantics (local copy of `sdd-spec`'s reference) that govern how each delta section is applied to the main spec.
- [`../sdd-verify/SKILL.md`](../sdd-verify/SKILL.md) — upstream; its `verify-report` verdict drives the Verification Gate.
- [`../sdd-apply/SKILL.md`](../sdd-apply/SKILL.md) — upstream; its `apply-progress` and marked `tasks.md` drive the Task Completion Gate and the stale-checkbox reconciliation proofs.
- [`../sdd-spec/SKILL.md`](../sdd-spec/SKILL.md) — upstream; its delta specs are what you merge into the main specs.
- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md) — save shape (`title == topic_key`, `scope: "project"`, active session), recovery ladder, and the observation-ID lineage the archive report must carry.
- [`../_shared/conventions/openspec.md`](../_shared/conventions/openspec.md) — the archive layout (`archive/YYYY-MM-DD-{change}/`), `rules.archive`, and the audit-trail rule.
