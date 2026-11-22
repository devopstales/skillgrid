# Ship Closeout: Change Closed, Rules, Fast-Track, Gotchas

## Change Closed

**Change**: YYYY-MM-DD-<topic>
**Acceptance verdict**: accepted | accepted-with-open-items | rejected
**Base branch**: {base-branch} · **Chain strategy**: {strategy}
**Integration**: {merged to {base} @ {commit} | PR {url} | kept branch {name}}
**Archived**: `.skillgrid/change-archive/YYYY-MM-DD-<topic>/`
**Status**: success | blocked

### Gates
| Gate | Result |
|------|--------|
| QA gate | ✅ {PASS \| WAIVED \| CONCERNS — human override recorded} |
| Review gate (advisory) | ✅ {met \| met-with-fixes \| not met notes} |
| Acceptance verdict (advisory) | {accepted \| …} — recorded, not enforced |

### Integration Test Evidence
- {test command} on integrated tree: {N passed, M failed} → {green}

### Sourced Learnings
- **Decision**: {choice + tradeoff} — {source}
- **Lesson**: {do differently + root cause} — {source}
- **Pattern**: {reusable approach} — {source}
- **Surprise**: {gotcha/edge case} — {source}

### Environment Retro
{the table rows, or "Clean — no findings"}

### Records
- PR body: {url | written to {path}}
- Changelog: {CHANGELOG.md | skipped — not user-facing / fast-track}
- Release note: {path | skipped}
- Projectwiki: {N pages written/updated + index run | nothing durable to file}

### Mechanical Move
- Pre-move snapshot → `git mv`/`mv` → `change-archive/`
- `diff -r` readback: {empty → ✅ | verbatim output → FAIL}

### Overrides / Waivers
{QA waiver, human CONCERNS override, fast-track class — or "None"}

**Rollback plan**: {trigger → procedure}
**Mnemonic**: observation `{id or 'none'}` · session `{sid}`
**Open items (→ next change)**: {list, or "None"}
**Next**: none — cycle complete
```

Close the final message with a `## Key Learnings` section — 1–5 standalone factual sentences (≥ 20 chars each). Mnemonic passive capture picks these up.

## Rules

- You close **verified** work. A `FAIL` in `report.md` → `## Gate Decision` or an unresolved CRITICAL blocks ship — no prompt override; require a fresh passing test run.
- **The iron law** (above) — no merge, push, or folder move before a fresh green run on the integrated tree.
- **Every learning is sourced.** A finding with no file:line / commit / ticket / scenario is a guess — cite it or mark it undiagnosed.
- **The acceptance verdict is advisory.** `rejected` is recorded and surfaced, never a hard block. The human decides follow-up.
- **The environment retro proposes, never implements.** Real builds become tickets + `mem_save`, not edits made in this phase.
- **Documentation writes from the record.** `git log` + `git diff` + the change folder are the sources; the AI's memory of its own work is not.
- **The move is mechanical** — one call to `ship-archive.mjs`, never model Read/Write and never hand-rolled `git mv` + `diff -r`, with an empty `diff -r` readback (empty = only passing evidence, verbatim in the JSON) after it.
- **The move is the LAST step** — everything that must be archived (retro half, environment retro, docs, wiki pages) is committed before the folder leaves `change-specs/`.
- **The archive is an audit trail** — never deleted or modified after the move. A post-archive defect is a new change.
- **The integration decision (merge / PR / keep / discard) is the user's** — present the menu and wait. Discard happens only on the typed word `discard`.
- **No release mechanics** (no VERSION bump) — the human-facing record is the changelog / release note from Step 4, and only where the change warrants it.
- **Decision debt ships, it doesn't clear.** An assumed blueprint flag is carried into the Return Envelope and the archive — never removed by shipping.
- If shell access is unavailable, STOP and return `blocked` — do not fall back to Read/Write moving.
- Return envelope per Step 8 — final action is text, not a tool call.

## Fast-Track Variant

For a `trivial` / `small` change (waiver recorded in `briefing.md`): the full steps still run, lightly —

- **Reflect:** one acceptance verdict line + a single high-value learning if there is one; no full retro half.
- **Environment retro:** the sweep runs but is expected to be quiet; no forced findings.
- **Document:** the PR body (if the PR option is chosen) is the only record — no changelog / release note offers.
- **Commit / archive:** unchanged — the mechanical move and the `diff -r` readback are **never** waived.

Note the class in the Return Envelope: `Fast-track: {trivial|small} (waiver in briefing.md)`.

## Gotchas

- **"Tests passed earlier this session."** Run the suite on the tree you are about to integrate. A green run only proves the tree it ran on.
- **"The base branch is obviously main."** Resolve it from `chain.strategy` in `config.yaml` + the work-unit table, or confirm/ask. Merging into the wrong base is expensive to undo. For `feature-branch-chain`, ship integrates the *tracker* (last) unit to *its* base boundary — not necessarily `main`.
- **"I remember what I changed, I'll write the docs from memory."** Memory is not a source. The AI's account of its own work is exactly the thing that drifts. `git log` + `git diff` + the change folder are the record.
- **Moving before the source is gone-check** gives a meaningless empty diff. The `diff -r` compares the *pre-move snapshot* to the archived folder, after confirming the source path no longer exists.
- **`--force` on a refused worktree removal** destroys files that exist only in that worktree. Show the user `git -C "$WORKTREE_PATH" status --porcelain -uall` and ask; never `--force` on your own initiative.
- **`CONCERNS` is not a block.** It is advisory — surface open items, the human decides. But `FAIL` / unresolved CRITICAL is a hard block.
- **A learning with no source is a guess.** "It was confusing" is not a learning. "The `feature-branch-chain` base boundary for PR #3 pointed at PR #1, not PR #2 — `tasks.md:47`" is.
- **A page without its index entry is drift the moment it lands.** The projectwiki same-edit rule: page write + `projectwiki index` ship together.

## How to measure it

Per `_shared/craft/measurement.md`.

| | Indicator | Data source | Direction |
|---|-----------|-------------|-----------|
| Leading | % of shipped changes whose `state.yaml` update, archive move, and `mem_save` all landed in the same commit (no manual follow-up) | Git history (commit scope) + `state.yaml` diff | should rise |
| Lagging | Rollbacks and "re-open" commits on an already-archived change (the integration was wrong and had to be undone) | Git history (commits touching a `.skillgrid/change-archive/` path or reverting the integration) | should fall |

## Anti-Patterns

| Anti-pattern | Symptom | Fix |
|---|---|---|
| Shipping on "the gates are probably fine" | QA gate shows `FAIL` or an unresolved CRITICAL but the phase proceeds instead of returning `blocked` | A `FAIL`/CRITICAL in `report.md` → `## Gate Decision` is a hard block — "it's fixed in a later commit" does not clear it; a fresh passing test run is required. |
| Skipping the workspace guard | A commit, merge, or move happens across a read-only or out-of-roots boundary | STOP and report — do not commit, merge, or move across an unsafe boundary. |
| Trusting an earlier green run | "Tests passed earlier this session" is cited as the pre-integration check | A green run only proves the tree it ran on — run the suite again on the tree you are about to integrate (the iron law). |
| Unsourced learnings | A report entry with no file:line, commit, ticket, or scenario | Unsourced is a guess, not a learning — cite it or mark it undiagnosed. |
| Generic changelog | "bug fixes and improvements" as an entry | Every bullet names what changed and where. |
| Prose for a mechanical rule | A new line in the standards file for a checkable violation | A linter rule or hook catches it every time; the standards file is for judgement calls only. |
| No guardrail, small repo | A repo with no lint/check at all | The absence is itself a finding — the first lint rule is the cheapest insurance. |
| Treating `rejected` as a hard block | The cycle stops on an advisory acceptance verdict | The verdict is recorded and surfaced, never a hard block — the human decides follow-up; don't conflate it with the (hard) QA Gate. |
| Deferred projectwiki reindex | A page written without `projectwiki index` (`status --json` shows `stale > 0`) | The same-edit rule: page + reindex ship together — a stale index silently drops the page from recall. |
| Hand-rolled archive move | Model Read/Write or ad-hoc `git mv` + manual `diff -r` instead of `ship-archive.mjs`; or the script's `readback` field non-empty/missing | One call to `ship-archive.mjs`; an empty `diff -r` readback is the only passing evidence. |
| Docs written from memory | A claim in the docs that names no commit, file, or change-folder source | `git log` + `git diff` + the change folder are the record — the AI's account of its own work is exactly the thing that drifts. |

## Red Flags

- A merge, push, or folder move happens before a fresh green test run on the integrated tree.
- A reusable Lesson/Surprise recorded only as a one-off, never routed to the governing skill or `_shared` rule — the next change repeats it.
- The PR body omitting the `report.md` verdict — the gate result is part of the record.
- The Return Envelope missing ship context (base branch, integration evidence, gate results).

## Verification

- [ ] All gates passed before any mutation: QA gate (no `FAIL`, no unresolved CRITICAL) + review gate (advisory) — or `blocked` returned with no mutation.
- [ ] The test suite is green on the tree being integrated (fresh run, not the QA run); for a local merge, green on the merged result.
- [ ] `report.md` completed in place with every learning sourced and an acceptance verdict recorded; the `## Environment Retro` section present (or an explicit clean note).
- [ ] Every Lesson/Surprise triaged process-bug vs one-off: process bugs named the governing skill/`_shared` rule and were recorded as follow-up items.
- [ ] Human-facing records written from the real diff (spot-check at least three claims against their source); the `report.md` verdict is named in the PR body / release note.
- [ ] Projectwiki pages (if any) have valid frontmatter + 2+ outgoing wikilinks; `projectwiki index` ran in the same edit; `validate` + `lint` show no new findings.
- [ ] Close-out committed per the work-unit commit contract (conventional subject, no AI-attribution trailer, `[skillgrid-context]` block; spec zone before code zone).
- [ ] The archive move ran via `ship-archive.mjs` with exit 0 and JSON `moved: true`, `readback: ""`, `source_gone: true`; the change folder is at `.skillgrid/change-archive/YYYY-MM-DD-<topic>/`.
- [ ] The script's JSON reports the `state.yaml` close-out (entry removed, `completed_changes` incremented); session lock released (JSON `lock` field); `mem_save` done (or explicitly degraded).
- [ ] The Return Envelope is rendered with the acceptance verdict, gate results, integration evidence, move evidence, and rollback plan.

