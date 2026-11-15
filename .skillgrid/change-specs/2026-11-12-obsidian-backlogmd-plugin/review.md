# Code Review — Obsidian Backlog.md Plugin

Change: `2026-11-12-obsidian-backlogmd-plugin` (Obsidian plugin for Backlog.md — read/write the skillgrid backlog layout: discovery, kanban, backlog list, milestones, GitHub pull, watcher).

## Setup

| Field | Value |
|---|---|
| Diff range | `ccb63495..ae71a40b` (13 commits; code under `plugins/obsidian/backlogmd/` — TypeScript, vitest, esbuild) |
| Pre-generated diffs | `review-code.diff` (3381 lines — code, excludes `main.js` + `package-lock.json` build artifacts), `review-spec.diff` (931 lines) |
| Frozen candidate hash | sha256 of `review-code.diff`, computed at package time (see commit) |
| Topology | core 5 (Standards, Spec, Security, EdgeCases, VerificationGaps) + Accessibility + Performance + Red team (last, consolidated) — 8 lenses, all `task` agents, fresh context, read-only |
| Blindness | per-lens template contract: no implementer narrative, no prior verdicts, graph-first where applicable, findings in template format |
| Lens status | 7/8 conforming output. **EdgeCases returned non-conforming output** (numbered prose, truncated at item 48 of ~48; no JSON array) — recorded per the code-review resilience rule; its 16 extractable candidates were triaged by the coordinator and verified at their locations. |
| GREEN side | `npx vitest run` → 46/46 pass, 10 files (re-run by Standards, Spec, coordinator). Bundle freshness verified: esbuild rebuild matches committed `main.js`. |

## Lens summaries

- **Standards** (`code-reviewer.md`): 0 Critical / 4 Important / 11 Minor. Verdict "Standards met? With fixes". Important: (1) new dependency set landed with no ADR (AGENTS.md rule breach — fixed by ADR-0066, this commit); (2) two divergent `nextTaskId` implementations; (3) one malformed task file blanks the entire board; (4) `TaskModal` save/create has no error handling.
- **Spec** (`spec-reviewer.md`): 26 scenarios → **16 PASS / 4 PARTIAL / 6 FAIL**. Verdict "Spec met? Partial". FAIL: `project-discovery-from-scan` (nested projects never reached — single-level scan), `project-discovery-manual-root` (manual-root subfolders not scanned), `frontmatter-malformed-surfaces-error` (no error card exists), `edit-task-autosave` (no autosave, no saved-report), `commands-registered` (6 of 7 named palette commands), `bidirectional-link-add/remove` (UI writes one side only; `linkTasks` has no caller). Briefing gaps (Missing): R11 settings set (active project, per-project repo link, description render toggle), R9 GitLab + opt-in render, R4 filter field coverage, R8 scroll preservation. Scope creep noted: in-column drag re-ranking, "Open milestones" command, milestone carry-over mutation, kanban auto-follow, PR exclusion. TDD evidence: no gate carries a RED commit (W012 `MISSING_RED`, WARNING per 2026-09-21 Standard-mode precedent).
- **Security** (`security-reviewer.md`): **1 Info** (floating `"obsidian": "latest"` devDependency — CWE-1108), 7 candidates triaged out with verified mitigations (textContent-only DOM, safe YAML schemas in the installed tree, keychain token, literal `api.github.com` host, slug path sanitization, no sourcemap in prod bundle). "Nothing exploitable."
- **EdgeCases** (`edge-case-hunter.md`): non-conforming/truncated output — 16 candidates extracted (config-load throw, per-file isolation, vault-root `backlog_directory`, `data.json` typing, GitHub pagination, assignee null, one-way reopen, create-modal TOCTOU, ID without numeric suffix, self-link, `updated_date` not stamped on link, status outside `project.statuses`, stale snapshots, ordinal ties).
- **VerificationGaps** (`verification-gap.md`): 5 broken verifications — G4/G21 round-trip is lossy at the file level (whitelist drop + canonical reordering; body only containment-tested); G9/G10 `linkTasks` unreachable; G6 create button never disabled (evidence overclaims); G8 no saved-report/autosave (evidence overclaims); G5 no error-card code path (evidence half-backed); G24 6 of 7 palette commands (evidence overcounts). +3 other findings (dead `writeText` branch, divergent `nextTaskId`, `backlog-view.render()` missing try/catch).
- **Accessibility** (`accessibility.md`): 15 findings — cards/rows mouse-only (not keyboard-reachable), no `aria-label` on inputs/dropdowns/buttons, no `:focus-visible`, silent modal validation, small touch targets, chip contrast ~2.9:1, no `aria-live` on milestone Start/Complete, focus not restored on modal close.
- **Performance** (`performance.md`): 14 findings — serial FS reads in `list()` (every render/keystroke/click on a 250-task vault), filter re-reads disk per keystroke with no debounce, `FileWatcher` re-renders on ANY `.md` change (incl. external notes), self-writes trigger watcher re-renders, dragover reads `offsetHeight` at 60Hz, `reorderColumn` serial N writes, exists-then-write, full re-read on card click and note-header click, no GitHub fetch timeout, single-page fetch, serial pull writes, serial carry-over writes.
- **Red team** (last lens, consolidated input): **13 findings all missed by the other 7** — (1) GitHub `/issues` returns `assignee` as a user object; plugin stores the whole object (typed `string`), displays `[object Object]`; test stub returns a string matching the wrong interface; (2) edit modal saves the open-time snapshot — external changes between open and save are silently clobbered; (3) modal close re-renders the *captured* project, not the current one (switcher/list can disagree after a project switch); (4) the real vault holds **two milestone files with id `m-10`** and 28 tasks reference it — second unselectable, `completeMilestone` carries over all 28; (5) `nextMilestoneId` TOCTOU mints duplicate ids from a stale snapshot (explains #4); (6) carry-over writes tasks serially then the milestone — mid-loop throw leaves torn state, no error, no re-render, button re-clickable; (7) ordering persists via `ordinal:` + vestigial `sprint:` — no real CLI task file has either; a CLI round-trip may silently drop board ordering; `updated_date` format diverges (date-only vs full timestamp); (8) watcher registers modify/create/delete but not `rename` — the CLI renames task files when titles change; board goes stale; (9) wrong-typed `manualRoots` string is iterated character-by-character as folder roots in discovery; (10) pulled bodies wrapped in a single ``` fence — bodies containing their own fences mis-render; (11) modal-created tasks omit the project-required `references:` frontmatter and DoD/plan body sections — every plugin-created task is an incomplete ticket by the project's own ticketing rules; (12) typed token persisted to secret storage **before** the pull succeeds — a failing pull stores the typo'd token; (13) `reorderColumn` has no in-flight guard — two quick reorders interleave writes, settled order matches neither drag.

## Consolidated findings (coordinator severity)

### Critical — data loss / wrong data on primary paths

| ID | Finding | Location | Sources | Disposition |
|---|---|---|---|---|
| C1 | Round-trip is lossy: unknown frontmatter fields (`points`, `rank`, custom) silently **dropped** on every write (16-field whitelist); field order canonicalized; body only containment-tested. Board ordering rides on `ordinal:`, which the CLI does not manage; `updated_date` written date-only (CLI writes full timestamps) | `src/infrastructure/vault.ts:41-48` | VG-G4/G21, Standards-M10, Spec PARTIAL, Red#7 | **Fix this round**: pass-through of original frontmatter (merge known fields over original data, original key order, delete-when-undefined), byte-identity round-trip test |
| C2 | GitHub `assignee` stored as nested user object (typed `string`) → `[object Object]` in every view; `null` never clears a local assignee | `src/application/pull-issues.ts:35,46`; `src/infrastructure/github.ts:5` | Red#1, Security cand-6, Edge-85 | **Fix this round**: map `assignee?.login`, clear on `null` |
| C3 | Stale clobber: edit modal saves the open-time task snapshot; external changes (CLI/git/sync) between open and save are silently overwritten. Also covers create-modal TOCTOU (stale `existing` list at save) | `src/ui/task-modal.ts:110-133` | Red#2, Edge-93 | **Fix this round**: re-read the task (re-list for create) from the repo at save time, merge patch onto fresh state |
| C4 | One malformed task file → `list()` rejects → entire board blank; no per-file isolation; no user-visible error (G5 "error card" has no code path) | `src/infrastructure/task-repository.ts:13-27`; `src/ui/kanban-view.ts:57-61,98-99` | Standards-I3, VG-G5, Edge-37, Spec WRONG-3 | **Fix this round**: per-file try/catch in `list()` (skip + record), views render an error card for skipped files |
| C5 | Modal-created tasks omit required `references:` frontmatter and DoD/plan body sections — incomplete ticket by the project's own ticketing gate (`backlogmd-formatting.md`) | `src/ui/task-modal.ts:110-133` | Red#11 | **Fix this round**: create path writes `references: []` + DoD/plan scaffolding |

### High — spec FAIL / wrong behavior

| ID | Finding | Location | Sources | Disposition |
|---|---|---|---|---|
| H1 | `linkTasks` has no production caller — bidirectional-link scenarios FAIL; the only dependency-edit path (modal) writes one side | `src/application/link-tasks.ts`; `src/ui/task-modal.ts:107-117` | Standards-M1, VG-G9/G10, Spec WRONG-6, Edge-33 | **Fix this round**: wire `linkTasks` into the modal's dependencies edit (both sides written); stamp `updated_date`; guard self-link |
| H2 | 6 of 7 named palette commands registered; "Switch project" and "Edit task" absent from the palette (G24 evidence overcounts) | `src/main.ts:30-44` | VG-G24, Spec WRONG-1, R10 | **Fix this round**: register both commands |
| H3 | No autosave, no saved-report (R5/G8 evidence unsupported); modal closes silently after save | `src/ui/task-modal.ts:100-140` | VG-G8, Spec WRONG-2 | **Fix this round**: debounced autosave (edit mode) + inline saved indicator |
| H4 | Discovery single-level: nested projects (`Projects/Demo/.backlog`) and manual-root subfolders never discovered (G1/G2 fixtures shallower than scenario Givens) | `src/infrastructure/config.ts:156-170` | Spec WRONG-4/5, R1 | **Fix this round**: depth-bounded recursive scan; scan manual-root subfolders |
| H5 | Duplicate milestone ids: TOCTOU mints the same id from a stale snapshot; real vault already has `m-10` ×2 with 28 referencing tasks (second unselectable; `completeMilestone` carries over all 28) | `src/ui/milestone-modal.ts`; `src/domain/milestone.ts` | Red#4/5 | **Fix this round**: re-list milestones at save before computing the next id; vault data repair → deferred (D3) |
| H6 | `FileWatcher`: listeners never unregistered on unload (leak across reloads); no `rename` event (CLI title edits go stale); over-triggers on any vault `.md` change | `src/infrastructure/watcher.ts`; `src/main.ts:89` | Standards-M5, Red#8, Perf | **Fix this round**: `registerEvent`, add `rename`, scope to project task/milestone dirs |
| H7 | `TaskModal`: no try/catch on save (silent unhandled rejection), Create button never disabled (silent no-op — G6 evidence overclaims) | `src/ui/task-modal.ts:102-133` | Standards-I4, VG-G6 | **Fix this round**: try/catch + Notice; `setDisabled` from summary state |
| H8 | GitHub fetch single page (`per_page=100`, no pagination) and no timeout | `src/infrastructure/github.ts:10-17` | Perf, Edge-79, Standards-M7 | **Fix this round**: page loop (cap), fetch timeout |
| H9 | Token persisted to secret storage before the pull succeeds — a failing pull stores the typo'd token | `src/ui/pull-issues-modal.ts:44-53` | Red#12 | **Fix this round**: save after success |
| H10 | Reopened GitHub issue stays Done (one-way state transition) | `src/application/pull-issues.ts:33-36` | Edge-86 | **Fix this round**: reopen → first open status |
| H11 | Carry-over writes tasks serially then the milestone; mid-loop throw → torn state, no error surface, button re-clickable | `src/application/milestone.ts:22-62`; `src/ui/backlog-view.ts` | Red#6 | **Fix this round**: milestone file last; on error → Notice + re-render; in-flight guard |
| H12 | `reorderColumn` no in-flight guard — quick double reorders interleave writes; settled order matches neither drag | `src/application/move-task.ts:17-25` | Red#13 | **Fix this round**: per-view in-flight guard |
| H13 | New dependency set (`gray-matter`, `js-yaml` + build/dev toolchain) landed with no ADR — AGENTS.md "No new dependencies without an ADR" breach | `plugins/obsidian/backlogmd/package.json` | Standards-I1 | **Fixed this commit**: ADR-0066 + § LOCKED row |

### Medium

| ID | Finding | Sources | Disposition |
|---|---|---|---|
| M1 | Two divergent `nextTaskId` implementations (any-trailing-digit + uppercased vs prefix-anchored + case-preserving) — same concept, two behaviors, each pinned by its own test | Standards-I2, VG-other2 | **Fix this round**: one domain function, prefix-anchored, uppercased prefix (CLI `TASK-NNN` frontmatter convention); filename derivation unchanged |
| M2 | Filter re-reads every task+milestone file per keystroke, no debounce | Perf-2 | **Fix this round**: debounce + re-filter the loaded list |
| M3 | `list()` reads N files serially (every render/keystroke/click) | Perf-1 | **Fix this round**: parallel reads |
| M4 | Modal close re-renders the captured project, not the current one (post-switch desync) | Red#3 | **Fix this round**: re-resolve current project at close/render |
| M5 | `data.json` `manualRoots` string → settings tab `display()` throws; discovery iterates it character-by-character | Edge-71/76, Red#9, Standards-M6 | **Fix this round**: type guard/coercion at both sites |
| M6 | Malformed `config.yml`/`backlog.config.yml` → `onload` throws (no try/catch around `yaml.load`); `discoverProjects` silently swallows ancestor-config errors (inconsistent) | Edge-45, Standards-M6 | **Fix this round**: per-config try/catch, `console.error` with path, skip project |
| M7 | `backlog_directory` equal to the vault root is rejected (strict `startsWith(vaultRel + "/")`) | Edge-57 | **Fix this round**: allow equality |
| M8 | Pulled bodies wrapped in a single ``` fence — bodies containing their own fences mis-render | Red#10 | **Fix this round**: use a longer fence when the body contains ``` |
| M13 | `"obsidian": "latest"` floating devDependency (build-environment drift) | Security-Info, Standards-M2 | **Fix this round**: pin to lockfile version |
| M14 | Dead code: `status.ts` unused fns, `VaultPort`, `updateFrontmatter`, `isDone`, unused CSS classes, `writeText` identical branches; hardcoded `new Date()` where `today()` exists | Standards-M1/M3/M9 | **Fix this round**: delete; reuse `today()` |
| M12 | `tsconfig` lacks `strict: true` | Standards-M4 | **Fix this round** (polish pass, sequential) |
| M9 | Create-modal TOCTOU ID collision (two modals → same id) | Edge-93 | **Fixed via C3** (re-list at save) |
| M10 | Task id without numeric suffix → filename `task-000` collision | Edge-101 | Deferred (manual-id edge; recorded) |
| M11 | Self-link unguarded; `linkTasks` doesn't stamp `updated_date`; non-atomic double write | Edge-33 | updated_date + guard **via H1**; atomicity recorded |

### Deferred (recorded, separate passes)

| ID | Finding | Sources | Note |
|---|---|---|---|
| D1 | Accessibility: 15 findings (keyboard reachability, ARIA labels, focus-visible, contrast, target size, aria-live, focus restore) | A11y lens | Separate a11y pass |
| D2 | Performance remainder: 60Hz dragover layout reads, card/note-header click full re-reads, exists-then-write, serial writes (reorder/pull/carry-over), self-write watcher re-render | Perf lens | Perf pass after correctness |
| D3 | Vault data repair: duplicate `m-10` milestone file (rename one; 28 tasks reference the id) | Red#4 | Data migration, separate commit |
| D4 | Briefing feature gaps requiring re-scope: R11 settings (active project, per-project repo link, description render toggle), R9 GitLab provider + opt-in render, R4 filter field coverage (status/priority/milestone), R8 scroll/selection preservation | Spec "Missing" | Re-scope the briefing, then new change |
| D5 | Drag-and-drop machinery duplicated across views; no RED evidence on any gate (W012 `MISSING_RED`, WARNING per precedent); `sprint` vestigial field | Standards-M8, Spec, Red#7 | Refactor/records |

## Verdicts

- **Standards met?** With fixes (H13 fixed this commit; C1-C5, H1-H12, M-list fixes in the code round).
- **Spec met?** Partial — 6 FAIL scenarios have in-round fixes (H1, H2, H3, H4, C4, H7 covers the G6 state); D4 items require briefing re-scope, recorded, not fixed in a review round.
- **Security:** clean (1 Info, fixed via M13).
- **Round budget:** 2-3 rounds per the 2026-10-31 `mnemonic-v2-settle-index` review precedent (T2 two-axis). Round 1 = this fix loop.

## Round record

- **Round 1 (2026-11-13):** spec-zone commit (this file + ADR-0066 + § LOCKED row) → parallel fix slices (data layer / UI layer) → sequential polish pass (pin, strict, dead code, README) → verification (build + full suite + targeted smokes: round-trip on a real CLI task file, malformed-file error card, assignee mapping) → code-zone commit `fc102e00` (2026-11-15) → ledger recorded (`review-ledger.md`: R1 31 fixed / 6 parked, R2 verification pass 6 fixed).
