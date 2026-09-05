# Commit conventions (shared)

Shared commit contract every SDD skill honors whenever it commits work — `sdd-apply`, `sdd-verify`, and any skill that ends a task with a commit. This file is the contract; keep it tight. It does not restate git mechanics.

**Hard rules:**
1. **Conventional Commits v1.0.0 is mandatory.** Every commit subject follows the [Conventional Commits 1.0.0 specification](https://www.conventionalcommits.org/en/v1.0.0/). No free-form subjects.
2. **Commits are checkpoints.** Each commit is a clean, restorable state at a task boundary. After compaction you recover from `git log`, not memory — a commit you can't describe from its message is a checkpoint that fails.
3. **If a ticket/issue id exists for this work, it appears in the commit message.** No id, no line (don't invent one). A real id with no footer is a broken checkpoint (see § Ticket id).

If a repository pins a stricter policy (`.commitlintrc*`, `commitlint` in CI, `husky`/`simple-git-hooks`, a `.git/COMMIT_EDITMSG` template, or a project AGENTS.md/CLAUDE.md/GEMINI.md commit rule), that policy wins — align with it instead of overriding.

## Commits are checkpoints (the Superpowers model)

Borrowed from [Superpowers](https://github.com/obra/superpowers) `subagent-driven-development`: conversation memory does not survive compaction. Controllers that lost their place have re-dispatched entire completed task sequences — the single most expensive failure observed. **The commit chain is the recovery map.** What you do, the order, and the review state of each step must be reconstructable from `git log` alone, because after a compaction or a `git clean -fdx`, the ledger lives in git history and nothing else.

Consequences:

- **Commit at task boundaries**, where the work is reviewed and verified — not mid-edit, not at the end of the session. A commit is the record that "task N was done and clean."
- **Each commit is independently reviewable and restorable.** Reverting or checking out that commit should leave the build green and the tests green — a red commit is a checkpoint you can't trust.
- **One commit per logical change** (config, implementation, tests, and docs for that change). A reviewer must understand *what* changed and *why* from the message and diff alone — never "WIP," "fix stuff," `asdf`, or "changes."
- **Frequent commits over few large ones.** A 40-commit history of 30-second checkpoints beats one 4000-line "big-bang" commit — the fine-grained chain is cheaper to `git bisect`, to revert, and to review.
- **Recover from `git log` before asking.** If you are unsure what was done, `git log --oneline -20` and the SDD ledger — not a fresh memory of "I think we did X."

This makes the commit message part of the system, not decoration. It is the only durable name for each checkpoint.

## Message format (Conventional Commits v1.0.0)

Per the [v1.0.0 spec](https://www.conventionalcommits.org/en/v1.0.0/), the full message is:

```
<type>[optional scope][!]: <description>
[BLANK LINE]
[optional body]
[BLANK LINE]
[optional footer(s)]
```

Rules (from the spec):

- **`type`** — required. Must reflect the *user-visible effect* of the change. The spec recognizes `feat` and `fix` as the two mandatory types; `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert` are the common extended set. The type is **singular** per commit — do not write `feat+fix: …` or `fix/docs: …`.
- **`scope`** — optional. Project-specific; use it where the repo standardizes a scope (`auth`, `ui`, `api`). If there is no established scope, omit it — do not invent one.
- **`!`** — optional. Present only when the change breaks the public contract. Pair with a `BREAKING CHANGE:` footer explaining what broke. `<type>!: subject` and `<type> scope!: subject` are both valid.
- **`description`** — required. **Imperative, present tense, one verb-led clause**, no trailing period, no "I" / "we" / "the agent." Example: `fix: merge MCP entries without clobbering keys`, not `Fixed an issue where MCP entries were being clobbered.`
- **Body** (optional) — only when the *why* is non-obvious. Wrap at 72. Blank line between subject and body.
- **Footers** (optional) — `Name: value` tokens separated by blank lines. See § Ticket id and § No AI trailers.

### Type selection guide

| Type | Use when | Example |
|---|---|---|
| `feat` | New user-facing behavior | `feat: add rate-limit middleware` |
| `fix` | Bug repair with a regression test | `fix: merge MCP entries without clobbering keys` |
| `docs` | Docs, comments, README — no code path change | `docs: record the code-indexing ladder in the shared convention` |
| `style` | Formatting / whitespace; no behavior change | `style: align gofmt in the indexer` |
| `refactor` | Restructuring without behavior change | `refactor: split the code-index indexer into store + search` |
| `perf` | Measurable performance improvement | `perf: batch the FTS queries by project` |
| `test` | Adding or fixing tests only | `test: cover FTS5 phrase queries with CJK input` |
| `build` | Build tooling, dependencies, install scripts | `build: pin trivy to v0.58` |
| `ci` | CI config, pipeline steps | `ci: run go test on PRs to dev only` |
| `chore` | Repo maintenance (no user-visible change) | `chore: rotate the CI badge URL in README` |
| `revert` | Undoing a prior commit — reference the commit hash reverted in the body | `revert: "feat: add rate-limit middleware" (#142)` |

**Priority rule:** when two types apply, pick the one that most affects users. A bugfix beats a formatting pass; a feature beats a chore. A `chore:` that is really a `fix:` hides a bugfix from change-log generation (`conventional-changelog` and friends) — pick the honest type.

## Ticket id (required when one exists)

When the work is tied to a ticket, **the ticket id appears in the footer of the commit message** in the tracker's close-token form. This is how a commit is traceable to the issue it resolves — drop the id and the checkpoint is untraced.

Format (per the Conventional Commits footer spec and each tracker):

```
<footer: keyword>: <id>
```

| Tracker | Footer form | Example full message |
|---|---|---|
| **Backlog.md** ([conventions](../issue-tracker/backlogmd.md)) | `Refs: <ID>` | `feat: add rate limiting\n\nRefs: BG-42` |
| **GitHub** ([conventions](../issue-tracker/github.md)) | `Closes #<n>` / `Fixes #<n>` / `Refs #<n>` (use `Closes`/`Fixes` to auto-close) | `fix: null-safety in MCP store\n\nCloses #187` |
| **GitLab** ([conventions](../issue-tracker/gitlab.md)) | `Closes #<iid>` / `Fixes #<iid>` / `Refs #<iid>` (GitLab iid, not global id) | `feat: add gitlab auth flow\n\nCloses !203` |
| **Jira** | bare `<PROJECT>-<ID>` or `Closes <PROJECT>-<ID>` | `fix: auth token expiry\n\nCloses SDD-142` |

Rules:

- **If a ticket id exists for this work, the footer is mandatory.** "Exists" means: an active backlog ticket, an open issue, a linked Jira story, or a PR that is itself a ticket. If none exist for this change, leave the footer blank — do not invent a placeholder.
- The footer is on its own line (separated from the body by a blank line). Per the spec, footers are `keyword: value` pairs; `Refs:`, `Closes:`, `Fixes:` are the accepted issue-reference keywords.
- Multiple ids: `Closes #1 #2` or `Refs: BG-41 BG-42` (space-separated) — one keyword, many values.
- **`Ref:` (no "s") is nonstandard** in v1.0.0 footers; use `Refs:` (plural). (GitHub accepts both, but the spec form is `Refs`.)
- The close keyword only appears when the commit actually closes/fixes; for partial reference, use `Refs:` — do not `Closes` a ticket you are not resolving.

If the ticket is on a tracker not in the table, check its convention and use the closest v1.0.0-compatible footer form.

## What goes in a commit

- **One logical change per commit.** A feature + its tests + the config that enables it = one commit. A feature + the docs for it = another. A refactor + the new behavior = two commits (see § Checkpoint boundaries).
- **Only intended files.** `git status` first, then stage explicitly. Never commit `node_modules`, `dist`, `.env*`, `*.sqlite`, `coverage/`, `.idea/`, `.vscode/`, or secrets. `git add .` / `-A` in a directory with any of those is a fat commit.
- **Verify before you commit.** Tests you claim to pass actually pass — the project's test command, not a guess. "Should pass" commits publish a red checkpoint.
- **Do not commit before a code review / verify gate** if the repo requires one — an `amend` after a hook rejection is cleaner than a red commit + fixup commit pair.

## No AI trailers (mandatory)

- **Never** add `Co-authored-by:`, `Co-Authored-By:`, `Generated-with:`, `AI-generated:`, `Signed-off-by: agent@…`, or any other trailer implying an AI/LLM author.
- Human authors only. The `commit-msg` hook (installed by `skillgrid install`) strips `Co-authored-by` *after* the commit — a safety net, not the policy. Do not rely on it.
- If a human reviewer explicitly wants a trailer on a hand-written commit, it goes in the human's own commit line.

## Checkpoint boundaries (multi-commit batches)

When one logical change is too big for a single commit, split by the natural seams in the diff — not by file count. Each resulting commit is its own checkpoint: it must build on its own (or on the previous commit on the branch, which is acceptable for a PR stack), and each must be independently reviewable.

Common, correct seams:

1. Config / scaffolding change.
2. Core implementation.
3. Tests for the implementation.
4. Docs / examples.

**Bad seams** that produce fragile checkpoints: splitting the same function's refactor across two commits, a `chore:` for a "just this" tweak that is really a `fix:`, or a commit that only deletes lines it is about to re-add in the next.

In Superpowers, the checkpoint discipline is reinforced by the SDD ledger: each task's completion is recorded as `Task <N>: complete (commits <base7>..<head7>, review clean)` — the two SHAs *are* the checkpoint boundary the controller recorded before dispatching the implementer (`BASE = git rev-parse HEAD` before the task). Recovery after compaction = read the ledger, then `git log <base>..<head>` to reconstruct what shipped.

## Message template

```text
<type>[optional scope][!]: <imperative subject, ≤ 72 chars>

<optional body — why, wrapping at 72>

<optional footer(s)>
```

## Examples

```text
feat: add rate limiting to the public API

Refs: BG-42

fix: merge MCP entries without clobbering existing keys

Closes #187

docs: record the code-indexing ladder in the shared convention

security!: pin axios to the patched release

BREAKING CHANGE: the auth middleware now rejects unversioned tokens

revert: "perf: inline the hot loop" (#134)

This reverts commit 2a91c4. The inlined path regressed on ARM64.
```

## Gotchas

- `type` is **singular** per commit — no `feat+fix: …`. One type.
- Subject is **imperative present tense** ("add", "fix", "drop", "record"), past tense ("added", "fixed") is invalid per v1.0.0.
- `Co-authored-by:` for an AI is the single most common agent commit mistake — the hook strips it *after* the fact, and the hook may not be installed. Write it right the first time.
- A subject longer than 72 chars breaks `git log --oneline` and most log/PR renderers.
- The conventional-commits type drives `conventional-changelog`. A `chore:` that is really a `fix:` hides a bugfix from release notes.
- Footer keyword must be `Refs` (plural) or `Closes`/`Fixes` — per the v1.0.0 spec. `Ref:` (singular) is GitHub-dialect and not spec; use the plural form.
- Do not `git commit --amend` a pushed commit without coordination — the push will reject, and any rebase is a conflict for reviewers.
- If the repo has a `commitlint` config with a `scope-enum`, pick a scope from that list. If no scope is expected, `<type>: subject` is correct — do not invent a scope.
- A "wip / fixup / asdf" commit on a shared branch is a checkpoint that hides the real work — `git rebase -i` to squash *before* review, not during.

## SDD workflow integration

- **The commit *is* the checkpoint.** After each SDD **step** (`## NN-<name>` in change-level `tasks.md`) completes cleanly, commit before moving on. The step's `### Commit` line + the commit SHA together form the recovery record — do not start the next step until the prior step's commit exists and its tasks are marked `[x]`.
- **`single-pr` is a delivery mode, not a commit mode.** Shipping multiple steps in one PR does **not** authorize one mega-commit. Commit each step separately **always** — including under `Delivery strategy: single-pr`. PR count ≠ commit count.
- **Before committing**: confirm the change is in the expected state per `sdd-verify` — tests pass; the step's `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS` (or the step DoD otherwise allows the planned commit).
- **Do not commit** work whose step Verification says FAIL (or anything but pass / pass-with-warnings).
- **The commit message names *what* changed and *which ticket* it resolves.** The *why* and *how* live in the plan (the step's `acceptance.feature` for the behavior contract) and the PR description — not the commit.
- Archive the change with `sdd-archive` *after* the commits land, not before.

## CLI / tool interactions

- **`skillgrid install`** installs a `commit-msg` hook that strips `Co-authored-by` and other AI trailers. It is a safety net, not a policy.
- **`git commit`** reads `COMMIT_EDITMSG` from the repo if a template is configured. Templates are per-repo — check before overriding.
- **`git rebase -i`** may squash *within* a step (fixup/WIP noise into that step's checkpoint) before pushing. Do **not** squash distinct SDD steps into one commit to "simplify" a `single-pr` delivery — those step boundaries are the recovery map. Squash *before* PR review, not during.
- **Superpowers subagent-driven-development** records the pre-dispatch `BASE` per task and the completion SHAs in its ledger. Reconstruct checkpoints with `git log <base>..<head>`. The `scripts/review-package PLAN_FILE BASE HEAD` script emits the commit list + stat + full diff for a reviewer — use BASE, never `HEAD~1`, which silently truncates multi-commit tasks.
