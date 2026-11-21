# Proposal: Hermes Kanban as a first-class ticketing option

**Status:** implemented (2026-11-21) — see gaps 1–5, all closed in `feat/v2-rewrite`
**Branch:** `feat/v2-rewrite`
**Scope:** `.agents/skills/slicing/`, `.agents/skills/skillgrid/`, `.agents/skills/execute/`, `.agents/skills/ship/`, `README.md`

## Summary

Hermes Kanban is already wired into the slicing skill's skill-side contract —
`ticketing.type: hermes-kanban` is a legal value in `slicing/SKILL.md`'s config
table, tracker-adapter table, slicing strategy, ticket lifecycle, red flags, and
verification, and a full CLI reference
(`slicing/references/ticketing/hermes-kanban/hermes-kanban.md`) plus body
template exist. The remaining work is closing the integration gaps so the option
is discoverable at onboarding, follows the tracker-adapter pattern other
trackers set, and is consistent end-to-end through execution and ship.

## What exists today

| Artifact | State |
|---|---|
| `slicing/SKILL.md` — config keys `ticketing.type`, `ticketing.kanban.board`, `ticketing.kanban.assignee` | ✅ present (L51–54) |
| `slicing/SKILL.md` — tracker adapter table row, discovery line, slicing strategy row, lifecycle specifics, red flags, verification | ✅ present (L331, L341, L396, L410, L458–460, L480) |
| `slicing/references/ticketing/hermes-kanban/hermes-kanban.md` — CLI surface, discovery, publication mapping, post-create verification, Backlog.md mirror, lifecycle table, epic, rationalizations | ✅ present (119 lines) |
| `slicing/references/ticketing/hermes-kanban/templates/ticket.md` — task body template | ✅ present |
| `skillgrid/references/patterns/config.yaml` — `kanban:` block documented ("written by `skillgrid install --agents hermes`") | ✅ present (L15–18) |
| `README.md` — `hermes` in Tool Dependency | ✅ present (L49) |

## Gaps (what this proposal adds)

### 1. Onboarding: detection + validation branch (required)

`skillgrid/references/pipeline/onboarding.md` (L59–74) lists detection rules for
`gh` / `glab` / `backlogmd` and a validation list for programmer-named trackers,
but has no hermes path. A programmer who answers "hermes" or "kanban" to the
override question has no defined behavior.

**Change:**

- Add a detection rule: `command -v hermes` on PATH → offer `hermes-kanban`
  (non-fatal, like the research CLI budget — detect, don't force). When the
  remote already resolves to `gh`/`glab`, hermes is still offered as an
  override, not auto-detected.
- Add validation branches:
  - "hermes" → `hermes-kanban`
  - "kanban" → `hermes-kanban`
- Add a follow-up question when `hermes-kanban` is chosen: board slug
  (default: the `project` name from `config.yaml`) and assignee profile
  (default: `default`), recorded under `ticketing.kanban.*`.

### 2. Formatting peer to match the tracker-adapter pattern (consistency)

Every tracker has two reference files: the CLI standard (`<tracker>.md`) and a
formatting standard (`<tracker>-formatting.md`). Hermes has only the CLI file —
the adapter table links `templates/ticket.md` directly instead.

**Change:** create
`slicing/references/ticketing/hermes-kanban-formatting.md` containing:

- Title convention: `TICKET-<NN>: <title>` (ASCII only — same rule the other
  trackers' formatting references carry for slugified names)
- Epic card convention: `Epic: <epic summary>`, `--workspace scratch`, parent of
  every ticket
- Per-ticket body structure: Scope / Acceptance / Fails-when / Blocked by
  (moved from `templates/ticket.md` notes)
- Workspace rule: coding → `--workspace worktree`, research → `--workspace
  scratch`
- Component split rule: one task per component, `Blocked by:` line explicit

Then update the `slicing/SKILL.md` adapter table row to the standard shape:
`hermes-kanban.md` → `hermes-kanban-formatting.md` → `templates/` when building
bodies. `templates/ticket.md` stays as the body scaffold it already is.

### 3. Install path documentation (discoverability)

`config.yaml` says `ticketing.kanban.*` is "written by `skillgrid install
--agents hermes`", but nothing in the skills describes that install.

**Change:** in `onboarding.md` (or a short section of `hermes-kanban.md`),
state what `skillgrid install --agents hermes` does — installs/wires the
`hermes` binary (or records its expected location), writes
`ticketing.kanban.board` / `assignee`, and is the only path that should set
those keys outside of the onboarding override question. This makes the config
comment true rather than dangling.

### 4. Downstream consistency checks (execute + ship)

`execute/SKILL.md` and `ship/SKILL.md` currently have zero mentions of hermes.

**Check / change:**

- `execute/SKILL.md` — the ticket-lifecycle table in slicing says `claim` is
  "atomic; prints the resolved workspace path — use it as the worktree for the
  implementer." Confirm `execute`'s worktree-setup step reads that path (or the
  `conventions.worktree_dir` default) for `hermes-kanban` tickets. If it only
  knows `.worktree/`, add one line: when the tracker is `hermes-kanban`, the
  implementer's worktree is the path `claim` printed, not a fresh
  `.worktree/<name>` directory.
- `ship/SKILL.md` — confirm Step 6 (Integrate) is unaffected: `Closes
  <task-id>` in the closing commit and the board `complete` at done (per the
  lifecycle table) already close the loop. If `ship`'s PR-body step should name
  the Kanban task IDs in the PR body for `hermes-kanban`, add it to Step 4's
  "where the change warrants it" list.

### 5. Evals (verification)

`slicing/evals/` should gain at least one hermes-kanban case mirroring the
existing tracker evals: a `tasks.md` + `config.yaml` fixture with
`ticketing.type: hermes-kanban` asserting the adapter reads
`hermes-kanban.md`, publishes parents-before-children, and records task IDs back
into `tasks.md`.

## Out of scope

- The `hermes` CLI itself (external tool; assumed available on PATH, same
  contract as `gh`/`glab`/`jira`/`backlog`)
- `skillgrid` CLI changes — `install --agents hermes` lives in the CLI repo,
  not the skills repo; this proposal only makes the skills side consistent with
  it
- Backlog.md-as-mirror semantics — already specified in `hermes-kanban.md`
  (board is source of truth, mirror status flows from the board)

## Acceptance criteria

- [ ] Onboarding names "hermes" / "kanban" resolve to `hermes-kanban` and ask
      for board slug + assignee
- [ ] `hermes-kanban-formatting.md` exists and the adapter table follows the
      two-file pattern like the other four trackers
- [ ] `skillgrid install --agents hermes` is described somewhere in the skills
      (onboarding or the CLI reference)
- [ ] `execute`'s worktree step is consistent with `claim`'s printed workspace
      path
- [ ] A slicing eval fixture with `ticketing.type: hermes-kanban` passes

## Open questions

1. Should onboarding **auto-detect** hermes (binary on PATH) and offer it, or
   only accept it as a programmer-named override? Leaning: detect + offer,
   non-fatal, matching the research CLI budget rule.
2. Should `hermes-kanban` support a Backlog.md mirror by default when
   `ticketing.type: hermes-kanban` (today: "may hold no entry at all")?
   Leaning: keep optional; the current text already permits both.
