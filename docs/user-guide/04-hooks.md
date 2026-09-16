# Hooks

AISkillGrid enforces discipline with two kinds of hooks: **git hooks** (commit-time) and **agent Stop hooks** (block the agent from ending its turn while an invariant is red). "A rule asks; a hook guarantees."

## Quick path

| Hook | Type | Blocks when |
|------|------|-------------|
| `pre-commit` | git | cwd drift, commit on a protected branch/detached HEAD, spec-zone mixed with code-zone, generated/scratch staged |
| `commit-msg` | git | non-conventional subject, or a `Co-Authored-By` / `Generated-By` AI trailer |
| `stop` | agent Stop | `testing.runner` is red |
| `gate-stop` | agent Stop | any `G<n>` unmet, or a happy-path requirement missing its gate |

`checkpoint-state.sh` is the dispatcher. Git hooks exec it; agent hooks exec the dedicated script.

## Layout

```text
.agents/
├── hooks/           # logic
│   ├── checkpoint-state.sh        # dispatcher + snapshot/restore/guard-msg
│   ├── precommit-guard.sh         # cwd-drift + protected-ref/detached
│   ├── precommit-zone-guard.sh    # spec-zone XOR code-zone
│   ├── precommit-ignore-guard.sh  # generated/scratch paths (auto-fix + FATAL)
│   ├── gate-lint.sh               # advisory linter for #### Gates
│   ├── gate-state.sh              # gate state engine (--status / --reverify)
│   ├── gate-stop.sh               # agent Stop: gate-level
│   └── stop-tests.sh              # agent Stop: suite-level
└── git-hooks/       # thin shims
    ├── pre-commit   ──> checkpoint-state.sh guard
    ├── commit-msg   ──> checkpoint-state.sh guard-msg "$1"
    ├── stop         ──> stop-tests.sh        (drains stdin)
    └── gate-stop    ──> gate-stop.sh         (reads harness JSON for session_id)
```

`install-hooks.sh` copies per-repo shims into the repo's active hooks dir, honoring an existing `core.hooksPath`. Agent Stop hooks install with `install-hooks.sh --with-stop`.

## The pre-commit chain

`checkpoint-state.sh guard` runs, in order:

1. **`precommit-guard.sh guard`** — FATAL on:
   - *cwd drift* (worktrees only): a prior `cd` left the worktree toplevel.
   - *protected ref / detached HEAD*: committing on `main`/`master`/`develop`/`trunk`/`release/` (override `SKILLGRID_PROTECTED_BRANCHES`) or a detached HEAD. First commit of a repo (unborn HEAD) is allowed. In a worktree, the branch must match the per-agent regex `^(agent-|worktree-agent-|worktree-wf_)[A-Za-z0-9._/-]+$` (override `SKILLGRID_AGENT_BRANCH_REGEX`).
2. **`precommit-zone-guard.sh`** — FATAL if a commit stages both spec-zone (`.skillgrid/specs/**` or `acceptance-tests/features/`) and code in one shot. Spec is committed *before* the code that satisfies it.
3. **`precommit-ignore-guard.sh`** — FATAL (and auto-fix) on generated/scratch paths: `acceptance-tests/.extracted/*` and the worktree dir (from `worktree_dir:`, default `.worktrees/`, plus `worktrees/`). If a path is already gitignored it is unstaged + WARNING; if not, the dir is **auto-added to `.gitignore`** ("auto-git"), unstaged, NOTE.
4. **`gate-lint.sh`** — **advisory** only (always exit 0). Flags a gate id not matching `G<n>`, a runnable gate missing CHECK/EXPECT, blank CHECK/EXPECT, an EXPECT that is a bare number also appearing in CHECK (tautology), an `ABANDON` with a blank reason, and a happy-path requirement with no gate (traceability gap).

Every FATAL prints a `RECOVERY:` line and exits 1, blocking the commit.

## Commit message guard

`checkpoint-state.sh guard-msg "$1"` (`$1` = message file):

- Subject must match the conventional-commits regex `^(build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test)(\([a-z0-9._-]+\))?!?: .+` → FATAL + RECOVERY (exit 1) if not.
- Rejects `Co-Authored-By` / `Generated-By` AI-attribution trailers → FATAL (exit 1).
- Exit 2 if the message file is missing.

## Agent Stop hooks

Both read `testing.runner` / gates from `.skillgrid/config.yaml`. Neither should drain the harness stdin (the `stop` shim drains it so the harness doesn't see a broken pipe; `gate-stop` deliberately does not, because the script reads the harness JSON for `session_id`).

### `stop-tests.sh` (suite-level)

- No `testing.runner` configured → allow the stop (no-op), exit 0.
- Runs the command; on failure prints the last 40 lines + `STOP BLOCKED: tests are failing...` and **exit 2** (blocks the stop). Success → exit 0.

### `gate-stop.sh` (gate-level)

Runs a fresh `gate-state.sh --reverify`:

- No spec discoverable → ALLOW (degrade, don't trap), exit 0.
- `unmet=0` and `missing=0` → ALLOW, exit 0; if only `ABANDON` gates remain, allows with a **HANDOFF REQUIRED** note (abandoned = handoff, never a pass).
- Otherwise → **BLOCK, exit 2**, listing outstanding items (capped display at 5).
- **Loop guard:** per `(session_id, repo)` state in `.skillgrid/sdd/gate-stop-state.json`. The counter accumulates only while the *resolved gate state* (hash of per-gate lines, not raw spec bytes — a comment edit must not re-arm) is unchanged; any progress resets it. After `SKILLGRID_GATE_MAX_BLOCKS` (default 6) blocks without progress → ALLOW with a release message.

## Gate state engine (`gate-state.sh`)

Parses a change's `#### Gates` blocks and reports each `G<n>`'s state. A gate is met only when its CHECK **freshly** exits 0 AND its output matches EXPECT (substring, or `/.../` POSIX ERE with optional `i` flag).

- `--status` — non-executing ledger view (runnable gates shown as `unmet (not run)`); exit 0 always.
- `--reverify` — executing: runs every runnable CHECK (bounded by `SKILLGRID_GATE_TIMEOUT`, default 120s, via `timeout`/`gtimeout` when available), compares to EXPECT.
- Spec discovery: explicit `--spec`, else path in `checkpoint.json`, else `acceptance.feature` in `git diff HEAD`, else newest under the specs root (from `bdd.specs_dir`).
- Output: one line per gate/requirement (`G1=met`, `G2=unmet`, `G3=abandoned`, `G4=manual`, `REQ=<name>=missing`) plus `SUMMARY met=.. unmet=.. abandoned=.. manual=.. missing=..`. `NO_SPEC` + zeroed summary if nothing found. **Exit 0 in both modes** — the caller interprets.

## Checkpoint (manual, skills call it)

| Subcommand | What it does |
|------------|--------------|
| `checkpoint-state.sh snapshot` | Derives + writes `.skillgrid/sdd/checkpoint.json` (branch, last commit, and the Task/Decisions/Remaining/Tried fields pulled from the last commit's `[skillgrid-context]` block). Prints `CHECKPOINT_WRITTEN <path>`. |
| `checkpoint-state.sh restore` | Prints the JSON plus live git state (branch, HEAD, `git status --short`) for a fresh session to resume from. `NO_CHECKPOINT` + exit 0 if absent. |
| `checkpoint-state.sh post-check` | WARNING (non-blocking) if `HEAD~1..HEAD` deleted tracked files. |

## Environment overrides

| Variable | Role |
|----------|------|
| `SKILLGRID_PROTECTED_BRANCHES` | Branches the pre-commit guard refuses |
| `SKILLGRID_AGENT_BRANCH_REGEX` | Per-agent branch allow-list (worktrees) |
| `SKILLGRID_TEST_CMD` | Override the `testing.runner` command |
| `SKILLGRID_GATE_TIMEOUT` | Per-CHECK timeout for `--reverify` (default 120s) |
| `SKILLGRID_GATE_MAX_BLOCKS` | Stop-hook no-progress release threshold (default 6) |

## Agent tool hooks (beyond git + Stop)

The two kinds above run at **commit** time and at **turn end**. Harnesses also expose **PreToolUse / PostToolUse** hooks that wrap individual tool calls. They are the right home for *intercepting and reshaping tool I/O* — not just gating — and they follow the same "a rule asks; a hook guarantees" contract.

Three idioms worth knowing (all graceful: a missing dependency or malformed input degrades to `exit 0`, letting the tool call through):

- **Read-only cache via `exit 2` + stderr.** A PreToolUse hook that wants to *replace* a tool's result (e.g. serve a cached `WebFetch` body instead of re-fetching) writes the body to **stderr** and **exits 2** — the harness then delivers that text in place of the tool output. This is how a `sha256(url)`-keyed cache, revalidated with `ETag`/`Last-Modified` (`304` only, no TTL), can cut repeated research fetches to zero. A PostToolUse pair stores the body + validators after each real fetch.
- **Graceful degradation is the default.** Check dependencies up front (`command -v jq || exit 0`); on any parse failure, log a warning and let the call through. A hook that cannot do its job must never block the session it is supposed to protect.
- **Hide, don't lose, model-facing noise.** A hook pair can swap large or vendor blocks for a `BLOCK_<hash>` placeholder on Read, expand the model's edits back onto the real content after Edit/Write, and restore the true file on Stop — so the model never re-reads or re-edits the hidden block, yet nothing is lost.

Skillgrid's current hooks are git + Stop; these are the patterns to reach for when you add a PreToolUse/PostToolUse hook (e.g. a tool-agnostic research cache to complement the in-skill `mnemonic` web cache).

## Next step

[Memory and indexing](05-memory-and-indexing.md)
