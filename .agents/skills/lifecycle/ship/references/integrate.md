### Step 1: Verify Tests on the Integrated Tree

Run the project's full test suite (`testing.runner` from `config.yaml`).

**If tests fail**, report the failures and stop — the menu comes after a green suite:

```
Tests failing (<N> failures). Must fix before shipping:

[show failures]
```

**If tests pass:** continue to Step 2.

### Step 2: Detect Environment

```bash
GIT_DIR=$(cd "$(git rev-parse --git-dir)" 2>/dev/null && pwd -P)
GIT_COMMON=$(cd "$(git rev-parse --git-common-dir)" 2>/dev/null && pwd -P)
# Capture now, while still inside the workspace — Step 5 changes directory
# before cleanup (Step 7) needs this value.
WORKTREE_PATH=$(git rev-parse --show-toplevel)
```

This determines which menu to show and how cleanup works:

| State | Menu | Cleanup |
|-------|------|---------|
| `GIT_DIR == GIT_COMMON` (normal repo) | Standard 3 options | No worktree to clean up |
| `GIT_DIR != GIT_COMMON`, named branch | Standard 3 options | Provenance-based (Step 7) |
| `GIT_DIR != GIT_COMMON`, detached HEAD | Reduced 2 options (no merge) | Externally managed — leave in place |

**Submodule guard:** before concluding "already in a worktree," verify you are not in a submodule:

```bash
git rev-parse --show-superproject-working-tree 2>/dev/null
```

If it prints a path, you are in a submodule — treat as a normal repo.

### Step 3: Determine Base Branch

The base branch is whatever this work forked from. Resolve it from `tasks.md` `## Delivery Strategy` → **`Chain strategy:`**:

| `Chain strategy:` | Base branch |
|---|---|
| `stacked-to-main` | `main` (each PR merges to main in order) |
| `feature-branch-chain` | The work-unit table's **base boundary** for the final work unit (PR #1 base = main/tracker; PR #2 base = PR #1 branch; …). Ship integrates the **tracker** (last) unit to its base boundary. |
| `size-exception` | The single PR's target (usually `main`) — record the accepted exception. |
| `pending` | Not decided — **ask** the user which chain strategy / base to use before proceeding. |

If it is still not known, ask: "This branch split from `<your best guess>` — is that correct?"

**Confirm before merging: merging into the wrong base is expensive to undo.**

### Step 4: Present Options

**Normal repo and named-branch worktree — present exactly these 3 options:**

```
Implementation complete. What would you like to do?

1. Merge back to <base-branch> locally
2. Push and create a Pull Request
3. Keep the branch as-is (I'll handle it later)

Which option?
```

**Detached HEAD — present exactly these 2 options:**

```
Implementation complete. You're on a detached HEAD (externally managed workspace).

1. Push as new branch and create a Pull Request
2. Keep as-is (I'll handle it later)

Which option?
```

Present the menu exactly as written. Discarding the work happens only in response to an explicit request to throw the work away (below). **The integration decision is the user's** — wait for the answer.

> **Fast-track variant (`trivial` / `small`):** the menu still appears, but the expected answer is merge (or PR); there is no PR-body generation step and the reflect `report.md` is the light form (one verdict line + the move readback). No release mechanics, no docs check — in any variant.

### Step 5: Execute Choice

**Option 1 — Merge Locally:**

```bash
MAIN_ROOT=$(git -C "$(git rev-parse --git-common-dir)/.." rev-parse --show-toplevel)
cd "$MAIN_ROOT"

# Merge first — verify success before removing anything.
git checkout <base-branch>
git pull
git merge <feature-branch>

# Verify tests on the merged result.
<test command>
```

If tests fail on the merged result: stop, leave the worktree and branch in place, and investigate — nothing has been pushed, so the merge is local and recoverable.

Once the merged result is green: clean up the worktree (Step 7), then delete the branch:

```bash
git branch -d <feature-branch>
```

**Option 2 — Push and Create PR:**

```bash
git push -u origin <feature-branch>
# From a detached HEAD, name the new branch on the remote:
# git push origin HEAD:refs/heads/<new-branch>
```

Then create the pull/merge request against `<base-branch>` with the forge's tooling (its CLI if available, or the creation URL most forges print on push), following the repo's PR template if present, and report the URL. **The PR body is written by `skillgrid:document` (type `pr`)** — it writes from the real diff (`git log`/`git diff`) + the change folder (`briefing.md` goal, `blueprint.md` approach, `tasks.md` what shipped, `report.md` gate + evidence), never from memory. Hand the body to the forge CLI or present it per `document`'s Step 4. **Keep the worktree** — the user iterates on PR feedback there.

**Human-facing record offers (after the integration lands, before Step 6):** offer `skillgrid:document` for the record types the change warrants — `changelog` (always, if the change is user-facing or a `CHANGELOG.md` exists), `release-note` (if a tag/version is involved), `postmortem` (if the change folder contains a debug state file with a design-flaw note, or the user names an incident). The user picks; `document` writes and commits. The fast-track light variant skips the offers (the PR body is the only record).

**Option 3 — Keep As-Is:**

Report: "Keeping branch `<name>`. Worktree preserved at `<path>`."

**If the user asks to discard the work** (explicit request only). Confirm first:

```
This will permanently delete:
- Branch <name>
- All commits: <commit-list>
- Worktree at <path>

Type 'discard' to confirm.
```

Wait for that exact confirmation. Then clean up the worktree (Step 7) and force-delete the branch:

```bash
git branch -D <feature-branch>
```

### Step 6: Capture Ship Context (for reflect)

Record the following in your working context (it goes into reflect's `report.md`):

- Base branch + chain strategy + work-unit table reference.
- Integration test evidence (the green run on the integrated tree).
- PR/merge outcome (URL, or merge commit, or "kept").
- Worktree state (cleaned up / preserved / host-managed).
- Gate results (QA verdict + any waiver/override; review status).
- **Open decision debt:** if the change's `blueprint.md` carried `**Status: ASSUMED**`, list it — "assumed decision (blueprint <path>) owes ratification". The flag survives the merge into the archive and the final `report.md`; shipping never clears it (only Ratify does).

These are passed to `skillgrid:reflect` via the Return Envelope.

### Step 6.5: Confirm the event stream holds the position

Before the folder move, confirm the last work-unit commit carries its
`[skillgrid-context]` block (`git log -1` shows the block): the session's
`commit` events are the resume record, so the archived change stays resumable
from `skillgrid session <session-id>` after the move. No separate close-out
step — the events already hold every committed unit.

### Step 7: Mechanical Move to Archive (LAST step)

The change folder moves from `specs/` (active) to a top-level `archive/` (closed historical record). This is a **mechanical filesystem operation** — file content MUST NEVER pass through the model's Read/Write path. The only acceptable move is a native shell command (`git mv` / `mv`), verified by a structural `diff -r` readback.

```bash
# Run as ONE shell transaction so the EXIT trap stays active.
# The snapshot is recursive and MUST be created BEFORE the move.
snapshot_root="$(mktemp -d "${TMPDIR:-/tmp}/ship.move.XXXXXX")"
trap 'rm -rf -- "$snapshot_root"' EXIT
cp -R ".skillgrid/specs/YYYY-MM-DD-<topic>" "$snapshot_root/source"

# Mechanical move (MANDATORY): git mv when tracked, mv otherwise.
mkdir -p .skillgrid/archive
if ! git mv ".skillgrid/specs/YYYY-MM-DD-<topic>" ".skillgrid/archive/YYYY-MM-DD-<topic>"; then
  mv ".skillgrid/specs/YYYY-MM-DD-<topic>" ".skillgrid/archive/YYYY-MM-DD-<topic>" || exit $?
fi

# The source must be gone before comparing the archived tree to its snapshot.
if [ -e ".skillgrid/specs/YYYY-MM-DD-<topic>" ] || [ -L ".skillgrid/specs/YYYY-MM-DD-<topic>" ]; then
  printf 'archive move left the source directory in place\n' >&2; exit 1
fi

# MANDATORY readback: only an empty diff passes.
diff -r "$snapshot_root/source" ".skillgrid/archive/YYYY-MM-DD-<topic>"
diff_status=$?
[ "$diff_status" -ne 0 ] && exit "$diff_status"
```

Use **today's date in ISO format** (`YYYY-MM-DD`) — it is already the folder's name prefix. Compare the archived folder against the **pre-move** recursive snapshot — do not substitute a model readback, a staged tree, or the post-move source. The move must come out **EXACTLY equal**: `report.md`'s QA half moves with the folder, so nothing is additive at move time. Any non-empty `diff -r` output or non-zero status is truncation, alteration, or an operational failure — it **FAILS** the phase. (Reflect's retro append to `report.md` happens AFTER this readback, in the archive — it is the documented post-move completion, not part of the move.)

> **If shell access is unavailable**, STOP and report `blocked` with reason `shell access required for mechanical archive move is unavailable` — do **not** fall back to Read/Write moving.

### Step 7.5: Reconcile (knowledge stays current after the change)

The change just merged — the durable knowledge must match the repo now, or it
rots. Reconcile is **surgical**: it adds lines and rewrites only the single
lines it owns. It never rewrites curated prose.

**Boundaries (the exact contract — reconcile touches nothing outside it):**

| Target | Reconcile does | Owner of the rest |
|---|---|---|
| `## Skillgrid` block (AGENTS.md / CLAUDE.md) | surgical update of drifted facts (stack, commands) via the sentinel upsert | onboarding (structure), human (curated prose) |
| `.skillgrid/artifacts/` terms (01-business-terms.md + 02-technical-terms.md) | add one line per new term the change introduced that is absent | `architectural-decision-records` (definitions) |
 | `.skillgrid/artifacts/04-adr-*.md` (index: `ASSUMPTIONS.md` § In-force set) | **flag only** — in-force ADRs the diff contradicts or supersedes, listed for the human | `architectural-decision-records` (an accepted ADR file is never deleted — supersede with a new file, don't edit) |
| Archived change folder | read (for the debt list); never modified after the move | the archive is an audit trail |
| `.skillgrid/config.yaml` | **flag only** — a detected stack/runner change the human should re-onboard for | onboarding (merge mode) |

**Process:**
1. Re-verify the `## Skillgrid` block facts against the repo (manifests, test runner, commands). If a line drifted (the change introduced a new dependency or changed the runner), update that line via the idempotent sentinel upsert (same sentinels as onboarding). If the drift is structural (a new area with its own conventions, a renamed runner), flag it — do not restructure.
2. Terms: scan the change's new/modified files for terms absent from the terms files (`artifacts/01-business-terms.md` / `02-technical-terms.md`). Add one line each (term + one-line definition) to the right file. If a term needs a real definition and ADR, flag for `architectural-decision-records` instead of guessing.
3. ADRs: list in-force ADRs the diff contradicts or supersedes. **Flag, never edit** — ADR immutability holds. The human decides whether a superseding ADR is owed.
4. Config: if the change changed the stack or test runner in a way `config.yaml` doesn't reflect, flag "run `skillgrid:onboarding` (merge mode)" — do not edit the config.
5. **Open decision debt:** list every `ASSUMED` blueprint in the archive (from Step 6) into the Return Envelope's `Open decision debt` line.

Record what reconcile changed (or "nothing to reconcile") in the ship context
for reflect's `report.md`.

### Step 8: Persist to Mnemonic (hybrid)

The filesystem move is already done (Step 7); persist the ship context as an index/backup so reflect can resume if interrupted.

```
mem_save(
  title:      "ship — YYYY-MM-DD-<topic>",
  topic_key:  "skillgrid/YYYY-MM-DD-<topic>/ship",
  type:       "architecture",
  scope:      "project",
  session_id: "{sid}",
  content:    "{ship context: base branch, chain strategy, integration test evidence, PR/merge outcome, worktree state, gate results, diff -r readback}"
)
```

> If `mnemonic.enabled` is `false`, the in-repo artifacts + git history are the sole record — skip this step (degrade explicitly, never fail silently).

