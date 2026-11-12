# Obsidian Backlog.md Board plugin (requirements & intent)

> **STATUS:** `draft` (2026-11-12)

> **Role:** the *what* — requirements, intent, success criteria. The companion
> `blueprint.md` is the *how*.

## Problem / Intent

Backlog.md (MrLesk/Backlog.md) stores project tasks as plain Markdown files with
YAML frontmatter under a `.backlog/` folder (`tasks/`, `completed/`, `drafts/`,
`milestones/`, `config.yml`). It ships a CLI and a web board, but no native
Obsidian integration. Obsidian users who keep their repo in a vault therefore
have no in-editor board view and must leave Obsidian (or drop to the CLI) to
triage, re-rank, or move tasks between statuses.

BoardMD (ericfisherdev/boardmd) proves the shape: a Jira-style backlog + sprint
board where every issue/sprint is a plain note, with drag-and-drop, configurable
workflows, multi-project switching, bidirectional issue links, and GitHub/GitLab
pull. The gap is that BoardMD owns its own note schema; it cannot read or edit
backlog.md files.

This change builds an Obsidian plugin that **renders and edits backlog.md task
files in place** — no schema change, no generated sidecar files — so the vault,
the CLI, and the plugin all agree on the same Markdown source of truth.

## Purpose & Success Criteria

- **Purpose:** give Obsidian users a kanban/backlog view + create/edit/move
  affordances over the backlog.md files already in their vault, without
  duplicating or re-owning the data.
- **Success criteria:**
  - A vault folder containing `.backlog/config.yml` is auto-discovered and
    appears in the project switcher within one scan.
  - Every task file under `<project>/.backlog/tasks/` (and `completed/`,
    `drafts/`) renders as a card in the column matching its `status`, using the
    project's configured statuses.
  - Dragging a card to another column rewrites only that task's `status`
    frontmatter field and writes the file; the change is visible in Obsidian's
    editor and survives a reload.
  - Create/edit forms persist via frontmatter round-trip; body (description) is
    preserved byte-for-byte except for the edited field.
  - Dependency/parent links are written on both sides (bidirectional), and a
    task blocked by an unfinished dependency shows a warning icon on its card.
  - External edits (CLI or manual) refresh the views within ~1s via the file
    watcher.
  - Desktop only; Obsidian ≥ 1.13.

## Requirements

1. **R1 — Project discovery (scan + manual root).**
   - **Current:** no Obsidian plugin exists.
   - **Target:** on load, scan the vault for folders containing a backlog
     config (`config.yml` inside the configured backlog directory, default
     `.backlog/`) and register each as a project; settings tab also accepts
     manual root folders and a per-project `backlog_directory` override,
     merged and deduped with scanned ones. Backlog.md supports relocating the
     backlog folder via `backlog config set backlog_directory <path>`; the
     plugin honors that setting when present.
   - **Acceptance:** a folder with `config.yml` appears in the switcher after
     scan; a manually added root's subfolders appear too; duplicates collapse.

2. **R2 — Task model + frontmatter round-trip.**
   - **Current:** none.
   - **Target:** parse each task's YAML frontmatter into a `Task` (id, title,
     status, type, priority, labels, assignee, milestone, created/updated
     dates, dependencies, references, documentation, parent_task_id) with body
     as description; serialize back preserving body and field order where
     possible.
   - **Acceptance:** round-trip a task file through parse→serialize and diff is
     empty except for an edited field; malformed frontmatter surfaces an error
     card, not a crash.

3. **R3 — Kanban board view.**
   - **Current:** none.
   - **Target:** a view with one column per configured status; cards show key,
     title, type, priority, status, assignee, points (if present), blocked
     icon; drag-and-drop between columns triggers `MoveTask`.
   - **Acceptance:** all statuses render as columns in configured order; a drag
     to a new column calls `MoveTask(task, newStatus)` and the card lands in
     the new column without a full reload.

4. **R4 — Backlog list view.**
   - **Current:** none.
   - **Target:** a list view grouped by status (or milestone), with a filter box
     covering status/type/priority/label/milestone and a create-task button.
   - **Acceptance:** filtering narrows the list live; clearing restores all.

5. **R5 — Create / edit task modal.**
   - **Current:** none.
   - **Target:** create form with summary required (button disabled until
     non-empty) plus optional type/priority/status/points/assignee/milestone/
     parent/dependencies; edit form with dropdowns and type-ahead pickers;
     autosave shortly after typing stops; "Done" flushes pending and closes.
   - **Acceptance:** creating a task writes a new `.backlog/tasks/<id>.md` with
     a generated id from the project config; editing persists and the modal
     reports "saved".

6. **R6 — Bidirectional links + blocked flag.**
   - **Current:** none.
   - **Target:** dependency/parent edits write both sides (e.g. task A's
     `dependencies` ↔ task B's `blocked_by` equivalent as defined by
     backlog.md conventions); a task with an unfinished dependency renders a
     red warning icon.
   - **Acceptance:** adding a link updates both notes; removing unlinks both;
     blocked icon appears iff at least one dependency is not in a "done"
     status.

7. **R7 — Milestones view (sprints parity).**
   - **Current:** backlog.md has `milestones/` and tasks reference milestones.
   - **Target:** group/filter tasks by milestone; a simple milestone board
     showing committed vs completed; completing a milestone records outcome
     (optional, best-effort).
   - **Acceptance:** tasks filter by milestone; milestone list loads from
     `.backlog/milestones/`.

8. **R8 — File watcher refresh.**
   - **Current:** none.
   - **Target:** watch task/config files; refresh views on create/modify/delete
     with debounce (~300ms) and preserve scroll/selection where feasible.
   - **Acceptance:** an external edit to a task file updates the board without
     manual reload.

9. **R9 — GitHub/GitLab pull.**
   - **Current:** none.
   - **Target:** per-project optional repo link (service, URL, token in secret
     storage); "Pull issues" command creates task notes with a `source`
     property; later pulls keep title/assignee/open-closed state in sync without
     overwriting sprint/milestone/rank/points/priority/parent/links/description.
     Descriptions stored as plain text in a code block by default; opt-in
     Markdown render for trusted repos.
   - **Acceptance:** first pull creates N tasks with `source`; second pull
     updates only title/assignee/state; descriptions are code-blocked by
     default.

10. **R10 — Commands & navigation.**
    - **Current:** none.
    - **Target:** command palette entries: Open backlog, Open board, Create
      task, Create milestone, Switch project, Pull issues, Edit task (for open
      note). Cmd/Ctrl-click on a card opens the note in a new tab; cards also
      expose an "Edit task" header button.
    - **Acceptance:** all commands registered and functional.

11. **R11 — Settings tab.**
    - **Current:** none.
    - **Target:** declarative settings (Obsidian ≥1.13): projects folder /
      manual roots, active project, per-project key prefix & default sprint
      length (read from config.yml), workflow view/edit, repo link + token,
      description render toggle.
    - **Acceptance:** settings persist in `data.json`; tokens in secret storage
      only.

## Context

- Backlog.md task format: YAML frontmatter (`id`, `title`, `status`, `type`,
  `priority`, `labels`, `assignee`, `milestone`/`sprint`, `created_date`,
  `updated_date`, `dependencies`, `references`, `documentation`,
  `parent_task_id`) + markdown body as description. Project config in
  `.backlog/config.yml` (statuses, types, priorities, labels, defaults).
- Backlog.md supports relocating the backlog folder via
  `backlog config set backlog_directory <path>` (see ADVANCED-CONFIG.md).
  **Proposal:** skillgrid adopts `.skillgrid/projectwiki/backlog` as the
  canonical backlog location (`backlog config set backlog_directory
  .skillgrid/projectwiki/backlog`) so task notes live inside the project wiki
  vault — committed, indexable by the projectwiki engine, and agent-readable
  alongside decisions/ADRs. The plugin must honor a configured
  `backlog_directory` rather than hard-coding `.backlog/`.
- BoardMD (reference implementation) provides the UX/layering template:
  `src/domain` (pure rules), `src/application` (commands/queries over ports),
  `src/infrastructure` (vault, frontmatter, agent files, GitHub/GitLab),
  `src/settings`, `src/ui`, wired in `src/compose.ts`. Adapt, don't copy.
- This repo is Go-based; the plugin is a separate TS project under
  `plugins/obsidian/backlogmd/` with its own `package.json`, esbuild, vitest.
  No Go code or Go dependency is touched.

## Implementation Decisions

- **No generated files.** The plugin does not write `index.json` or `AGENTS.md`;
  the backlog CLI owns those. The plugin is a renderer + editor only, avoiding
  conflicts with the CLI.
- **Backlog directory location.** Proposed (not yet decided): relocate this
  repo's own backlog to `.skillgrid/projectwiki/backlog` via
  `backlog config set backlog_directory .skillgrid/projectwiki/backlog`, so
  tasks become part of the committed project wiki vault. The plugin must
  resolve the backlog dir from config rather than assuming `.backlog/`. Final
  decision deferred to the Owed-Decision Gate / blueprint.
- **Frontmatter is the source of truth.** All mutations rewrite the task note's
  frontmatter; body preserved.
- **Bidirectional links are maintained by the plugin** on edit (both sides
  written), matching BoardMD; link fields come in pairs.
- **Desktop only.** No mobile-specific handling in v1.
- **Dependencies injected via constructors** (BoardMD pattern); domain layer
  pure and testable without Obsidian APIs.

## Testing Decisions

- Domain + application layers tested with vitest using mocked ports (no
  Obsidian runtime).
- Frontmatter round-trip tested against fixture task files (incl. multiline
  body, arrays, quoted wikilinks).
- A manual smoke checklist for drag-and-drop and file-watcher refresh (Obsidian
  runtime not unit-testable).
- No Go tests affected; `go build ./...` / `go test ./...` remain green.

## Impact on Global Docs

- `.skillgrid/projectwiki/ARCHITECTURE.md`: add `plugins/obsidian/` subtree
  note under §2/§12 (plugin assets) — Obsidian plugins are a new artifact
  kind distinct from agent-harness plugins.
- `.skillgrid/projectwiki/ASSUMPTIONS.md`: no new ADR required (no Go
  dependency added; TS deps are plugin-local). If the maintainer wants the
  plugin shipped via the skillgrid installer, that is a separate change.
- PRD: none (no PRD file in live projectwiki).

## Clarity Report

| Dimension           | Score | Min  | Status | Notes                              |
|---------------------|-------|------|--------|------------------------------------|
| Goal Clarity        | 0.90  | 0.75 | ok     | render+edit backlog.md in Obsidian |
| Boundary Clarity    | 0.85  | 0.70 | ok     | no generated files; CLI owns those |
| Constraint Clarity  | 0.85  | 0.65 | ok     | desktop only, Obsidian ≥1.13       |
| Acceptance Criteria | 0.80  | 0.70 | ok     | R1–R11 numeric-ish                 |
| **Clarity**         | 0.15  | ≤0.20| ok     |                                    |

## Open Questions & Assumptions

- **Assumption:** backlog.md's `dependencies` field stores task ids (not
  wikilinks); the plugin resolves them to files by id. If a project uses
  wikilink-style dependencies, the adapter normalizes.
- **Assumption:** "done" status is determined by the project config's status
  category (backlog.md has no explicit category; treat status name containing
  "done"/"completed"/"closed" as done, configurable in settings).
- **Open:** whether to support multiple vaults sharing one backlog root — v1
  assumes one vault.
- **Open:** milestone completion recording is best-effort; full parity with
  backlog sprints may require a follow-up.

## Terms

- **backlog.md task:** a Markdown file under `.backlog/tasks/` with YAML
  frontmatter + body description.
- **project:** a folder containing `.backlog/config.yml`.
- **status:** one of the project's configured statuses; columns are statuses.
- **milestone:** a note under `.backlog/milestones/`; tasks reference it.
- **blocked:** a task with at least one dependency not in a done status.
