# Review Ledger — 2026-11-12-obsidian-backlogmd-plugin

## Round 1 — 2026-11-13

**Frozen candidate:** base ccb63495 → head ae71a40b, 13 commits, review-code.diff sha256 a500b5d072680a44 (excludes `main.js` + `package-lock.json` build artifacts)
**Reviewer:** code-review: 8-lens fan-out (core 5 + Accessibility + Performance + red team), all fresh-context subagents, coordinator triage
**Independence grade:** B
**Budget:** round 1/2-3 (tier T2)

### Findings

| ID | Severity | File:Line | Finding | Disposition | Fix Commit | Lineage |
|---|---|---|---|---|---|---|
| R1-F001 | CRITICAL | src/infrastructure/vault.ts:41-48 | C1 round-trip lossy: unknown frontmatter fields dropped (16-field whitelist), field order canonicalized, body only containment-tested | fixed | fc102e00 | R1-F001 |
| R1-F002 | CRITICAL | src/application/pull-issues.ts:35,46 | C2 GitHub `assignee` stored as nested user object → `[object Object]`; `null` never clears local assignee | fixed | fc102e00 | R1-F002 |
| R1-F003 | CRITICAL | src/ui/task-modal.ts:110-133 | C3 stale clobber: edit modal saves open-time snapshot; create-modal TOCTOU on stale list | fixed | fc102e00 | R1-F003 |
| R1-F004 | CRITICAL | src/infrastructure/task-repository.ts:13-27 | C4 one malformed task file blanks the entire board; no per-file isolation, no error surface | fixed | fc102e00 | R1-F004 |
| R1-F005 | CRITICAL | src/ui/task-modal.ts:110-133 | C5 modal-created tasks omit required `references:` + DoD/plan body sections | fixed | fc102e00 | R1-F005 |
| R1-F006 | WARNING | src/ui/task-modal.ts:107-117 | H1 `linkTasks` had no production caller; dependency edit wrote one side only | fixed | fc102e00 | R1-F006 |
| R1-F007 | WARNING | src/main.ts:30-44 | H2 "Switch project" + "Edit task" palette commands missing (6 of 7 registered) | fixed | fc102e00 | R1-F007 |
| R1-F008 | WARNING | src/ui/task-modal.ts:100-140 | H3 no autosave, no saved-report; modal closes silently after save | fixed | fc102e00 | R1-F008 |
| R1-F009 | WARNING | src/infrastructure/config.ts:156-170 | H4 discovery single-level: nested projects and manual-root subfolders never found | fixed | fc102e00 | R1-F009 |
| R1-F010 | WARNING | src/ui/milestone-modal.ts | H5 duplicate milestone ids: TOCTOU mints same id from stale snapshot (vault already has m-10 ×2) | fixed | fc102e00 | R1-F010 |
| R1-F011 | WARNING | src/infrastructure/watcher.ts | H6 watcher: listeners leaked across reloads, no `rename` event, over-triggers on any vault `.md` | fixed | fc102e00 | R1-F011 |
| R1-F012 | WARNING | src/ui/task-modal.ts:102-133 | H7 no try/catch on save (silent unhandled rejection); Create button never disabled | fixed | fc102e00 | R1-F012 |
| R1-F013 | WARNING | src/infrastructure/github.ts:10-17 | H8 GitHub fetch single page (per_page=100), no timeout | fixed | fc102e00 | R1-F013 |
| R1-F014 | WARNING | src/ui/pull-issues-modal.ts:44-53 | H9 token persisted to secret storage before the pull succeeds | fixed | fc102e00 | R1-F014 |
| R1-F015 | WARNING | src/application/pull-issues.ts:33-36 | H10 reopened GitHub issue stays Done (one-way transition) | fixed | fc102e00 | R1-F015 |
| R1-F016 | WARNING | src/application/milestone.ts:22-62 | H11 carry-over writes tasks serially then milestone; mid-loop throw → torn state; button re-clickable | fixed | fc102e00 | R1-F016 |
| R1-F017 | WARNING | src/application/move-task.ts:17-25 | H12 `reorderColumn` no in-flight guard — double reorders interleave writes | fixed | fc102e00 | R1-F017 |
| R1-F018 | WARNING | plugins/obsidian/backlogmd/package.json | H13 dependency set landed with no ADR (AGENTS.md breach) | fixed | c0993731 | R1-F018 |
| R1-F019 | SUGGESTION | src/application/create-task.ts; src/application/pull-issues.ts | M1 two divergent `nextTaskId` implementations | fixed | fc102e00 | R1-F019 |
| R1-F020 | SUGGESTION | src/ui/backlog-view.ts | M2 filter re-reads every task+milestone file per keystroke, no debounce | fixed | fc102e00 | R1-F020 |
| R1-F021 | SUGGESTION | src/infrastructure/task-repository.ts | M3 `list()` reads N files serially | fixed | fc102e00 | R1-F021 |
| R1-F022 | SUGGESTION | src/ui/kanban-view.ts | M4 modal close re-renders captured project, not the current one | fixed | fc102e00 | R1-F022 |
| R1-F023 | SUGGESTION | src/compose.ts; src/settings/settings-tab.ts | M5 `manualRoots` string → settings `display()` throws; discovery iterates character-by-character | fixed | fc102e00 | R1-F023 |
| R1-F024 | SUGGESTION | src/infrastructure/config.ts | M6 malformed config → `onload` throws; ancestor-discovery errors silently swallowed | fixed | fc102e00 | R1-F024 |
| R1-F025 | SUGGESTION | src/infrastructure/config.ts:126 | M7 `backlog_directory` equal to vault root rejected | fixed | fc102e00 | R1-F025 |
| R1-F026 | SUGGESTION | src/application/pull-issues.ts:48 | M8 pulled bodies wrapped in single ``` fence; bodies with own fences mis-render | fixed | fc102e00 | R1-F026 |
| R1-F027 | SUGGESTION | src/ui/task-modal.ts | M9 create-modal TOCTOU id collision — resolved via C3 re-list | fixed | fc102e00 | R1-F027 |
| R1-F028 | SUGGESTION | src/application/create-task.ts | M10 task id without numeric suffix → `task-000` filename collision | parked | — | R1-F028 |
| R1-F029 | SUGGESTION | src/application/link-tasks.ts | M11 self-link unguarded; `updated_date` not stamped; non-atomic double write — updated_date + guard via H1; atomicity recorded | fixed | fc102e00 | R1-F029 |
| R1-F030 | SUGGESTION | tsconfig.json | M12 tsconfig lacks `strict: true` | fixed | fc102e00 | R1-F030 |
| R1-F031 | SUGGESTION | package.json | M13 `"obsidian": "latest"` floating devDependency | fixed | fc102e00 | R1-F031 |
| R1-F032 | SUGGESTION | src/domain/status.ts; styles.css; src/infrastructure/vault.ts | M14 dead code: unused fns, unused CSS classes, identical `writeText` branches, hardcoded `new Date()` where `today()` exists | fixed | fc102e00 | R1-F032 |
| R1-F033 | SUGGESTION | src/ui/*.ts (15 findings) | D1 accessibility: keyboard reachability, ARIA labels, focus-visible, contrast ~2.9:1, target size, aria-live, focus restore | parked | — | R1-F033 |
| R1-F034 | SUGGESTION | src/ui/kanban-view.ts; src/application/move-task.ts | D2 performance remainder: 60Hz dragover layout reads, full re-reads on card/note-header click, exists-then-write, serial writes, self-write watcher re-render | parked | — | R1-F034 |
| R1-F035 | SUGGESTION | .skillgrid/projectwiki/backlog/milestones/ | D3 vault data repair: duplicate `m-10` milestone file (rename one; 28 tasks reference the id) | parked | — | R1-F035 |
| R1-F036 | SUGGESTION | briefing.md | D4 briefing feature gaps requiring re-scope: R11 settings, R9 GitLab + opt-in render, R4 filter field coverage, R8 scroll preservation | parked | — | R1-F036 |
| R1-F037 | SUGGESTION | src/ui/kanban-view.ts; src/ui/backlog-view.ts | D5 drag-and-drop machinery duplicated across views; no RED evidence on any gate (W012, WARNING); `sprint` vestigial field | parked | — | R1-F037 |

### Dispositions

- **R1-F001** — fixed in `fc102e00`. `serializeTask` merges known fields over the original frontmatter (insertion order preserved, delete-on-undefined, unknown fields pass through); byte-identity round-trip test in `vault.test.ts`.
- **R1-F002** — fixed in `fc102e00`. `assignee?.login` mapping; `null` from GitHub clears the local assignee.
- **R1-F003** — fixed in `fc102e00`. Edit-mode save re-reads the task from the repo; create path re-lists at save (M9/M27 resolved with this).
- **R1-F004** — fixed in `fc102e00`. Per-file try/catch in `list()` (skip + `lastErrors`); kanban/backlog views render an error card. Hardened in Round 2 (R2-F002) against gray-matter tokenizer statefulness.
- **R1-F005** — fixed in `fc102e00`. Create path writes `references: []` + DoD/plan scaffolding.
- **R1-F006** — fixed in `fc102e00`. `linkTasks` wired into the modal's dependency edit (both sides, `updated_date` stamped, self-link guarded — M11/F029 with this).
- **R1-F007** — fixed in `fc102e00`. `switch-project` (cycles the open view's projectIndex, opens kanban when none open) + `edit-task` (active MarkdownView, reuses `openTaskModalForFile`) commands registered.
- **R1-F008** — fixed in `fc102e00`. Debounced autosave in edit mode + inline saved indicator.
- **R1-F009** — fixed in `fc102e00`. Depth-bounded DFS below each scan root and manual root.
- **R1-F010** — fixed in `fc102e00`. Milestone modal re-lists at save before computing the next id; callers now pass 3 args (the stale 4-arg call was a latent type error caught by the strict pass, R2-F006).
- **R1-F011** — fixed in `fc102e00`. Handlers registered through the plugin `registerEvent` (released on unload); `rename` added to the event set; triggers scoped to the live project dirs' `tasks/` + `milestones/`.
- **R1-F012** — fixed in `fc102e00`. Save wrapped in try/catch + Notice; buttons disabled from summary/in-flight state.
- **R1-F013** — fixed in `fc102e00`. Page loop capped at 10 pages/1000 issues; 30s AbortController timeout per page.
- **R1-F014** — fixed in `fc102e00`. Token persisted only after a successful pull; empty field → `setSecret(key, "")` (storeSecret is a no-op for empty, so the clear is explicit) then placeholder swap.
- **R1-F015** — fixed in `fc102e00`. Reopen maps to the first open status (leaves Done).
- **R1-F016** — fixed in `fc102e00`. Milestone file written last; on error → Notice + re-render; in-flight guard on the buttons (plain-HTML `disabled`, not `ButtonComponent.setDisabled` — caught by strict pass, R2-F006).
- **R1-F017** — fixed in `fc102e00`. `dropInFlight` guard in kanban-view.
- **R1-F018** — fixed in `c0993731`. ADR-0066 (`.skillgrid/projectwiki/decisions/04-adr-0066-obsidian-backlogmd-plugin-deps.md`) + § LOCKED row.
- **R1-F019** — fixed in `fc102e00`. Single `nextTaskId` in `src/domain/task.ts` (prefix-anchored, uppercased prefix); both call sites use it.
- **R1-F020** — fixed in `fc102e00`. Filter debounced; re-filters the loaded list, no disk re-read per keystroke.
- **R1-F021** — fixed in `fc102e00`. `Promise.all` parallel reads in `list()`.
- **R1-F022** — fixed in `fc102e00`. Modal close re-resolves the current project before re-rendering.
- **R1-F023** — fixed in `fc102e00`. `coerceManualRoots` type guard at compose + settings sites; pinned by `compose.test.ts`.
- **R1-F024** — fixed in `fc102e00`. Per-config try/catch with `console.error` including the path; malformed project skipped; ancestor-discovery errors surfaced consistently.
- **R1-F025** — fixed in `fc102e00`. `backlog_directory == vault root` now accepted (`vaultRel ""` = project root).
- **R1-F026** — fixed in `fc102e00`. Fence escalation to 4 backticks when the body contains its own ```.
- **R1-F027** — fixed in `fc102e00`. Via R1-F003 (create re-list at save).
- **R1-F028** — parked. Ruling: manual-id edge (a task id with no numeric suffix) is out of the review round's scope; recorded for the next correctness pass. paladm.
- **R1-F029** — fixed in `fc102e00`. Via R1-F006 (updated_date stamp + self-link guard); non-atomic double write recorded (two CLI calls, acceptable at plugin scale).
- **R1-F030** — fixed in `fc102e00`. `"strict": true` in tsconfig (subsumes `noImplicitAny` + `strictNullChecks`); 9 surfaced errors fixed (see R2-F006).
- **R1-F031** — fixed in `fc102e00`. `obsidian` devDependency pinned `1.14.4`; lockfile re-synced (`npm install --package-lock-only`).
- **R1-F032** — fixed in `fc102e00`. `status.ts` deleted; dead CSS classes removed; `writeText` duplicate branches merged; `today()` reused.
- **R1-F033** — parked. Ruling: separate accessibility pass after correctness is established (15 findings: keyboard, ARIA, focus-visible, contrast, targets, aria-live, focus restore). paladm.
- **R1-F034** — parked. Ruling: performance pass after correctness (60Hz dragover reads, click re-reads, exists-then-write, serial writes, self-write re-render). paladm.
- **R1-F035** — parked. Ruling: data migration — rename the duplicate `m-10` milestone file (28 tasks reference the id); separate commit. paladm.
- **R1-F036** — parked. Ruling: briefing re-scope (R11 settings set, R9 GitLab + opt-in render, R4 filter field coverage, R8 scroll/selection preservation), then a new change. paladm.
- **R1-F037** — parked. Ruling: drag-and-drop dedup refactor + RED evidence records (W012 `MISSING_RED`) + `sprint` vestigial-field decision — separate pass. paladm.

### Round Verdict

findings-dispositioned (31 fixed, 6 parked)

## Round 2 — 2026-11-15

**Frozen candidate:** base c0993731 → head fc102e00, 46 files, file-list hash 1427f93159373c9e
**Reviewer:** in-session verification pass (build + full suite + targeted determinism runs + dead-class/usage grep) — not a blind review
**Independence grade:** C (same session as the fixes; verification, not independence)
**Budget:** verification pass (round budget exhausted by Round 1 fix loop)

### Findings

| ID | Severity | File:Line | Finding | Disposition | Fix Commit | Lineage |
|---|---|---|---|---|---|---|
| R2-F001 | WARNING | tests/compose.test.ts; vitest.config.ts (new) | Suite-load failure: `Failed to resolve entry for package "obsidian"` (its `package.json` `main` is `""`; `vi.mock` cannot intercept a failed resolution) — first suite run since plugin creation, so the M5 test never ran | fixed | fc102e00 | R2-F001 |
| R2-F002 | CRITICAL | src/infrastructure/vault.ts:28-31 | gray-matter 4.0.3 tokenizer keeps global regex state: a malformed frontmatter file THROWS on first parse but returns `{}` on later parses in the same process → ghost card + data-corruption path (`serializeTask` strips the frontmatter of a `{}`-parsed file). Made `task-repository.test.ts` flaky | fixed | fc102e00 | R2-F002 |
| R2-F003 | SUGGESTION | tests/infrastructure/tmp-debug.test.ts; tmp-repro.test.ts | Two throwaway debug test files in the tree | fixed | fc102e00 | R2-F003 |
| R2-F004 | SUGGESTION | styles.css:58-60 | `.backlogmd-error*` classes referenced by three view files had no CSS (error cards rendered unstyled); three dead classes present (`.backlogmd-list`, `-list-row`, `-status` — zero usages) | fixed | fc102e00 | R2-F004 |
| R2-F005 | SUGGESTION | README.md (new) | No README — review checklist expected docs matching actual behavior | fixed | fc102e00 | R2-F005 |
| R2-F006 | WARNING | src/main.ts:43; src/ui/backlog-view.ts:201-220; src/infrastructure/watcher.ts:26,44,49 | Strict pass (R1-F030) surfaced 9 type errors; 2 were latent bugs: `new MilestoneModal(..., all)` 4-arg call vs 3-arg constructor (H10's fix never type-checked), `setDisabled` on plain `HTMLButtonElement` ×4 | fixed | fc102e00 | R2-F006 |

### Dispositions

- **R2-F001** — fixed in `fc102e00`. `vitest.config.ts` `resolve.alias` → `tests/mocks/obsidian.ts` value stub (Plugin, TFile, Notice, Modal, Setting); also fixed the test's wrong relative import (`../../src` → `../src`).
- **R2-F002** — fixed in `fc102e00`. `parseTask` pre-validates the frontmatter block with stateless top-level `js-yaml` before `matter()` — malformed files now always throw and are always isolated in `lastErrors` (no ghost cards, no rewrite corruption); fixture replaced with a deterministic unterminated double-quoted scalar. 3 consecutive green runs of the formerly flaky suite confirmed.
- **R2-F003** — fixed in `fc102e00`. Deleted.
- **R2-F004** — fixed in `fc102e00`. Error-card styles added (border-left accent, `--text-error` title); dead classes deleted after usage grep.
- **R2-F005** — fixed in `fc102e00`. README created: prerequisites (backlog CLI + root `backlogmd.config.yml` + layout), multi-project, dev commands, behavior notes.
- **R2-F006** — fixed in `fc102e00`. All 9 errors fixed: vault-event union overload cast (widened to `TAbstractFile`), `Timeout | null` → `| undefined` with unguarded `clearTimeout` (style rule), `EventRef` typing on the watcher register callback, MilestoneModal arg count, `disabled` property on plain buttons.

### Round Verdict

findings-dispositioned (6 fixed, 0 parked)
