# pi-worktree

Pi plugin for [Worktrunk](https://worktrunk.dev/) worktree workflows that
implements the skillgrid worktree standard: turn-level scratch checkpoints
(capture, dedup, safe restore) + a `wt` runner with linked-session movement,
no-replay recovery, and the skillgrid invariants (branch allow-list,
`--no-squash --no-commit` merge defaults).

## Install

```sh
# pi extension install
pi install npm:@devopstales/pi-worktree

# or bun / npm (host Pi resolves the pi packages)
bun add @devopstales/pi-worktree
npm install @devopstales/pi-worktree
```

Or run a file directly during development: `pi -e ./src/scratch.ts
-e ./src/wt.ts`.

## Requirements

- **Pi ≥ 0.99.1** (linked-session API; matches pi-worktrunk) — declared in
  `package.json` `engines`.
- **`wt` (Worktrunk) ≥ 0.50.0** — verified against `wt 0.50.0` (2026-11-22).
  The `WORKTRUNK_DIRECTIVE_CD_FILE` directive protocol is
  worktrunk-internal (not documented public API); the runner probes protocol
  support at startup (`supportsDirectiveProtocol`).
- git on PATH.

### Two extension files, one package

`package.json` `pi.extensions` wires **two independent extension files**:

| extension file | provides |
|---|---|
| `src/scratch.ts` | `turn_end` capture, `scratch-restore` tool, `/checkpoint` command, footer `◆ N checkpoints` |
| `src/wt.ts` | `worktrunk` tool, `/wt` command, linked-session moves, approval dialog, no-replay recovery |

Comment out one manifest line to use only that half (a git-only machine wants
scratch without `wt`). `src/core/` is pure git with zero Pi imports — enforced
by `tests/import-scan.test.ts`.

### git-fallback behavior (when `wt` is absent)

The wt extension probes the `WORKTRUNK_DIRECTIVE_CD_FILE` protocol at first
use (`supportsDirectiveProtocol` — wt must exist, report a version, and
actually write a directive file). When `wt` is absent or the protocol is not
supported, the probe returns false and the extension **skips directive
movement**: no linked-session moves are attempted (the run result is
passthrough — the wt output without a `moved`/`stopped` outcome). No
worktrunk-specific feature degrades silently; the skip is the result of a
named probe, not a guess. The scratch extension is unaffected — checkpoints
work with plain git. Skill-wise this is the tooling tiers' git fallback:
worktrees are created with `git worktree add` (execute Step 0b, tier 3) and
the standard `.worktree/<branch>` layout is used manually; the plugin never
creates or removes worktrees on the fallback path.

## Markers — sole-writer warning

`src/markers.ts` is the **only writer** of Worktrunk marker state in this
plugin (`session_start`→💬, `agent_start`→🤖, `agent_end`→💬,
`session_shutdown`→clear), and the two extension files import it rather than
writing markers directly.

Worktrunk ships its **own** pi plugin that writes the same markers. If both
are installed, both write marker state — no runtime detection is performed
(Goldfish Stage 5 R1-F003); keep only one. Uninstall Worktrunk's pi plugin
when using this one. A marker left by a session killed before shutdown is
reported and cleared manually with `wt config state marker clear`.

## Features

- **Scratch checkpoints** (`refs/skillgrid/`): one ref per mutating turn
  (temp-index capture — HEAD + index tree + worktree tree, real index
  untouched, no commit in history); read-only turns create no ref (tree-SHA
  dedup); 50 refs/session cap + background cross-session prune (1h
  liveness).
- **Safe, decision-gated restore**: `scratch-restore` tool / `/checkpoint`
  command, diff preview + `files-only`/`cancel` dialog, automatic
  before-restore checkpoint, branch-keyed check (cross-branch restores are
  refused with both branch names), safeClean never deletes ignored dirs,
  pre-existing untracked files, or skipped-large items. No auto-restore,
  ever.
- **`worktrunk` tool + `/wt`**: sequential `wt` runs over the
  `WORKTRUNK_DIRECTIVE_CD_FILE` protocol; valid directory directives move the
  pi session into a linked session (source stays `/resume`-able); print/json
  mode reports `stopped` + restart directory.
- **No-replay recovery**: cwd removed out from under a session → recover to a
  surviving worktree of the same repository (identity re-verified, default
  branch preferred); a command requested after the directory died is NOT run.
- **Skillgrid invariants**: branch allow-list at creation
  (`agent-*` / `worktree-agent-*` / `worktree-wf_*`; primary checkout
  unconstrained), `--no-squash --no-commit` merge defaults (the one argv
  rewrite — OD-3), first-use worktree-root observation
  ("observed root differs — noted" outside `<projectroot>/.worktree/`).
- **Approval passthrough**: unapproved project commands → dialog; approving
  saves exactly those commands (`wt config approvals add`) and retries once;
  the model never passes `-y`/`--yes`.
- **Status markers**: 💬/🤖 in `wt list` per the session lifecycle.

### Reference docs

- [`references/wt-commands.md`](references/wt-commands.md) — the `wt`
  commands this plugin uses and their default rewrites (one page; the live
  catalog is `wt --help`).
- [`references/scratch-restore.md`](references/scratch-restore.md) — the
  restore protocol: branch-keyed check, refusal reasons, before-restore
  checkpoint, safeClean semantics.

## Design decisions on record

- **OD-1 — ref namespace `refs/skillgrid/`**: approved; once in use it is not
  renamable without orphaning refs.
- **OD-3 — merge-default argv rewrite** (`--no-squash --no-commit` prepended):
  approved; the one place the plugin mutates the user's command.
- ~~OD-2 — worktree-root half of the restore check~~: retired (Goldfish Stage
  5 R1-F001) — a branch name exists in exactly one live worktree, so the
  branch check implies root identity.

## Development

```sh
bun install
bun test            # full suite (cores run without a Pi install)
bun run bench       # capture latency table (≤150ms at 5k files)
```

The contract-drift guard (`tests/contract-drift.test.ts`) compares
`src/core/constants.ts` against the `~~~skillgrid-constants` fence in the
skillgrid-skills contract
(`.agents/skills/_shared/work-unit-commits/README.md`); it requires
`SKILLGRID_CONTRACT_PATH` to point at that file (CI pins a copy).

Evaluations: `evals/evals.json` (pinned `{name, description, input, expect}`
format) — the ported `execute` scratch scenarios + marker and approval
scenarios.
