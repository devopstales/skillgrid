# Obsidian Backlog.md Board plugin — acceptance contract
#
# Backfilled 2026-11-13 at the QA gate (C-1 fix): the original flat Gherkin
# scenarios carried no executable oracle. Layout follows
# 2026-10-06-context-harness-clm and 2026-10-31-mnemonic-v2-knowledge-base:
# ## Requirements / ### Requirement / #### Scenario (gherkin fences) with a
# #### Gates block per requirement. Each gate mirrors exactly one scenario.
#
# Gate convention: `# G<n>: desc (mirrors <scenario>)` + `#   CHECK:` (a
# runnable command, or `manual` for Obsidian-runtime seams) + `#   EXPECT:`
# (the success marker) + `#   EVIDENCE:` (run result or live-probe citation).
# CHECK commands run from `plugins/obsidian/backlogmd/`.
#
# Shared precondition (original Gherkin Background): a vault containing a
# project folder with a valid backlog config, with the plugin enabled.
#
# Source: briefing.md ## Context; blueprint tasks SATISFIES lines.

## Requirements

### Requirement: Project discovery

The system SHALL discover projects by scanning the vault for a backlog
config, merge manually added roots, and dedupe; a project with no config
SHALL NOT appear.

#### Scenario: project-discovery-from-scan

```gherkin
      Given a folder "Projects/Demo" contains ".backlog/config.yml"
      When the plugin loads
      Then "Demo" appears in the project switcher
```

#### Scenario: project-discovery-manual-root

```gherkin
      Given settings add a manual root "Extra"
      And "Extra/Demo2" contains a backlog config
      When the plugin rescans
      Then "Demo2" appears in the project switcher
```

#### Scenario: project-discovery-dedupe

```gherkin
      Given a project is both scanned and manually added
      When the project list is computed
      Then it appears only once
```

#### Gates
# G1: a scanned backlog config makes the project appear (mirrors project-discovery-from-scan)
#   CHECK: npx vitest run tests/infrastructure/config.test.ts -t "discovers a child folder with a legacy .backlog/config.yml from a scan"
#   EXPECT: 1 passed
#   EVIDENCE: live probe 2026-11-13 — "Demo" project discovered in the switcher
# G2: manual roots merge into the scan (mirrors project-discovery-manual-root)
#   CHECK: npx vitest run tests/infrastructure/config.test.ts -t "merges manual roots with the scan and dedupes"
#   EXPECT: 1 passed
#   EVIDENCE: live probe 2026-11-13 — manually added root listed
# G3: a project reached by both paths appears once (mirrors project-discovery-dedupe)
#   CHECK: npx vitest run tests/infrastructure/config.test.ts -t "merges manual roots with the scan and dedupes"
#   EXPECT: 1 passed
#   EVIDENCE: live probe 2026-11-13 — no duplicate switcher entry

### Requirement: Vault round-trip and error handling

A task file read and rewritten with only frontmatter patched SHALL keep
the body byte-identical; malformed YAML frontmatter SHALL surface an error
and SHALL NOT crash the plugin.

#### Scenario: frontmatter-roundtrip

```gherkin
      Given a task file "OB-1.md" with frontmatter and a body
      When the plugin reads and rewrites only the status field
      Then the file body is byte-identical
      And only the status field changed
```

#### Scenario: frontmatter-malformed-surfaces-error

```gherkin
      Given a task file with invalid YAML frontmatter
      When the plugin renders the board
      Then an error card is shown and the plugin does not crash
```

#### Gates
# G4: parse + serialize is lossless, only patched fields change (mirrors frontmatter-roundtrip)
#   CHECK: npx vitest run tests/infrastructure/vault.test.ts
#   EXPECT: 6 passed
#   EVIDENCE: 46/46 suite green 2026-11-13 (vitest run)
# G5: malformed YAML throws at parse, not silent corruption (mirrors frontmatter-malformed-surfaces-error)
#   CHECK: npx vitest run tests/infrastructure/vault.test.ts -t "parse throws on malformed YAML frontmatter"
#   EXPECT: 1 passed
#   EVIDENCE: live probe 2026-11-13 — board surfaces the error without crashing (kanban-view render rethrow handled by Obsidian item-view)

### Requirement: Task operations

Creating, editing, and linking tasks SHALL persist through the vault: a
created task gets the next prefixed ID and a 1000-step ordinal, an edit
stamps updated_date and writes the patched task, and a dependency link is
bidirectional (removed symmetrically).

#### Scenario: create-task-requires-summary

```gherkin
      Given the create task modal is open
      When the summary is empty
      Then the Create button is disabled
      And when a summary is entered the button is enabled
```

#### Scenario: create-task-writes-file

```gherkin
      Given the create task modal with summary "New work"
      When the user clicks Create
      Then a new file is created under the backlog tasks folder
      And its id uses the project key prefix
```

#### Scenario: edit-task-autosave

```gherkin
      Given an open edit modal for a task
      When the user edits the priority and waits
      Then the file is updated and the modal reports saved
```

#### Scenario: bidirectional-link-add

```gherkin
      Given tasks "OB-1" and "OB-7"
      When a dependency link is added from "OB-1" to "OB-7"
      Then "OB-1" lists "OB-7" in dependencies
      And "OB-7" lists "OB-1" in blocked_by
```

#### Scenario: bidirectional-link-remove

```gherkin
      Given tasks "OB-1" and "OB-7" are linked
      When the link is removed
      Then neither task references the other
```

#### Gates
# G6: empty summary disables Create (mirrors create-task-requires-summary)
#   CHECK: manual — open the create-task modal in the live vault
#   EXPECT: Create button disabled with empty summary, enabled on input
#   EVIDENCE: live probe 2026-11-13 — modal validation observed
# G7: createTask writes the task with the next prefixed ID (mirrors create-task-writes-file)
#   CHECK: npx vitest run tests/application/create-task.test.ts -t "writes a new task with the next prefixed id, default status, and body"
#   EXPECT: 1 passed
#   EVIDENCE: live probe 2026-11-13 — new file with prefixed ID under tasks/
# G8: editTask persists the patch and stamps updated_date (mirrors edit-task-autosave)
#   CHECK: npx vitest run tests/application/edit-task.test.ts
#   EXPECT: 2 passed
#   EVIDENCE: live probe 2026-11-13 — modal reports saved, file updated
# G9: adding a link writes both sides (mirrors bidirectional-link-add)
#   CHECK: npx vitest run tests/application/link-tasks.test.ts -t "add: both sides reference each other"
#   EXPECT: 1 passed
#   EVIDENCE: live probe 2026-11-13 — both task files reference each other
# G10: removing a link clears both sides (mirrors bidirectional-link-remove)
#   CHECK: npx vitest run tests/application/link-tasks.test.ts -t "remove: neither task references the other"
#   EXPECT: 1 passed
#   EVIDENCE: live probe 2026-11-13 — no mutual references remain

### Requirement: Kanban board

The board SHALL render one column per configured status in order, place
each card in its status column, update the file on drag, show a blocked
warning for unfinished dependencies, and open the note on Cmd-click.

#### Scenario: kanban-columns-from-statuses

```gherkin
      Given a project with statuses "To Do, In Progress, Done"
      When the board view opens
      Then three columns render in configured order
```

#### Scenario: kanban-card-in-status-column

```gherkin
      Given a task "OB-1" with status "In Progress"
      When the board renders
      Then "OB-1" appears in the "In Progress" column
```

#### Scenario: drag-move-updates-status

```gherkin
      Given a task "OB-1" in "To Do"
      When the user drags it to "In Progress"
      Then the task file's status becomes "In Progress"
      And the card appears in "In Progress" without full reload
```

#### Scenario: blocked-icon

```gherkin
      Given task "OB-4" depends on unfinished "OB-7"
      When the board renders
      Then "OB-4" shows a blocked warning icon
```

#### Scenario: cmd-click-opens-note

```gherkin
      Given a card on the board
      When the user Cmd-clicks it
      Then the task note opens in a new tab
```

#### Gates
# G11: one column per status, in configured order (mirrors kanban-columns-from-statuses)
#   CHECK: manual — open the board view in the live vault
#   EXPECT: columns match the project's status list order
#   EVIDENCE: live probe 2026-11-13 — three columns in config order
# G12: a card renders in the column of its status (mirrors kanban-card-in-status-column)
#   CHECK: manual — open the board view in the live vault
#   EXPECT: "OB-1" in the "In Progress" column
#   EVIDENCE: live probe 2026-11-13 — card placement observed
# G13: a drag updates the file's status and re-renders (mirrors drag-move-updates-status)
#   CHECK: npx vitest run tests/application/move-task.test.ts -t "writes status, ordinal and updated_date, keeping body"
#   EXPECT: 1 passed
#   EVIDENCE: live probe 2026-11-13 — drag rewrote status/ordinal/updated_date only; no full reload
# G14: an unfinished dependency flags the task blocked (mirrors blocked-icon)
#   CHECK: npx vitest run tests/domain/task.test.ts -t "isBlocked true when dependency not done"
#   EXPECT: 1 passed
#   EVIDENCE: live probe 2026-11-13 — warning icon rendered on the blocked card
# G15: Cmd-click opens the task note (mirrors cmd-click-opens-note)
#   CHECK: manual — Cmd-click a card in the live vault
#   EXPECT: the task note opens in a new tab
#   EVIDENCE: live probe 2026-11-13 — note opened

### Requirement: Backlog list

The backlog list SHALL filter tasks by type and restore the full list
when the filter is cleared.

#### Scenario: backlog-list-filter

```gherkin
      Given the backlog list shows 5 tasks
      When the user filters by type "bug"
      Then only bug tasks remain
      And clearing the filter restores all tasks
```

#### Gates
# G16: type filter narrows and clears (mirrors backlog-list-filter)
#   CHECK: manual — filter the backlog list in the live vault
#   EXPECT: only "bug" tasks while filtered; all tasks restored on clear
#   EVIDENCE: live probe 2026-11-13 — filter + clear observed

### Requirement: Milestones

The plugin SHALL list milestone notes from `milestones/` and let the
board filter by milestone.

#### Scenario: milestone-filter

```gherkin
      Given tasks assigned to milestones "M1" and "M2"
      When the user filters by milestone "M1"
      Then only "M1" tasks remain
```

#### Scenario: milestone-list-loads

```gherkin
      Given a project with milestone notes under milestones/
      When the milestones view opens
      Then the milestones are listed
```

#### Gates
# G17: milestone notes parse and list (mirrors milestone-list-loads)
#   CHECK: npx vitest run tests/domain/milestone.test.ts -t "parses the CLI's minimal format"
#   EXPECT: 1 passed
#   EVIDENCE: live probe 2026-11-13 — milestones view listed the notes
# G18: board filters by milestone (mirrors milestone-filter)
#   CHECK: manual — apply a milestone filter in the live vault
#   EXPECT: only "M1" tasks while filtered
#   EVIDENCE: live probe 2026-11-13 — milestone filter observed

### Requirement: Watcher

An external modification of a task file SHALL refresh the open board
within 1 second.

#### Scenario: watcher-refresh-on-external-edit

```gherkin
      Given the board is open
      When a task file is modified externally
      Then the board updates within 1 second
```

#### Gates
# G19: external edit refreshes the board (mirrors watcher-refresh-on-external-edit)
#   CHECK: manual — edit a task file with the board open in the live vault
#   EXPECT: the board reflects the change within 1 second
#   EVIDENCE: live probe 2026-11-13 — refresh observed under 1s

### Requirement: GitHub pull

Pulling issues SHALL create new task files with a source property, sync
title/assignee/state without overwriting local planning fields, and store
descriptions as plain text in a code block.

#### Scenario: github-pull-creates-tasks

```gherkin
      Given a project linked to a GitHub repository
      When "Pull issues" runs
      Then new task files are created with a source property
```

#### Scenario: github-pull-syncs-without-overwrite

```gherkin
      Given previously pulled tasks
      When "Pull issues" runs again
      Then title, assignee, and open/closed state update
      And sprint, rank, points, priority, parent, links, and description are not overwritten
```

#### Scenario: github-pull-descriptions-code-blocked

```gherkin
      Given descriptions are pulled by default
      When a task note is inspected
      Then the description is stored as plain text in a code block
```

#### Gates
# G20: first pull creates tasks with a source ref (mirrors github-pull-creates-tasks)
#   CHECK: npx vitest run tests/application/pull-issues.test.ts -t "first pull creates tasks with source ref and code-blocked body"
#   EXPECT: 1 passed
#   EVIDENCE: live probe 2026-11-13 — pulled tasks carry source
# G21: second pull syncs only title/assignee/state (mirrors github-pull-syncs-without-overwrite)
#   CHECK: npx vitest run tests/application/pull-issues.test.ts -t "second pull syncs title/assignee/state only, keeps local planning and body"
#   EXPECT: 1 passed
#   EVIDENCE: live probe 2026-11-13 — local planning fields untouched on re-pull
# G22: descriptions are code-blocked plain text (mirrors github-pull-descriptions-code-blocked)
#   CHECK: npx vitest run tests/application/pull-issues.test.ts
#   EXPECT: 6 passed
#   EVIDENCE: live probe 2026-11-13 — note shows description inside a code block

### Requirement: Backlog directory configuration

A project config's `backlog_directory` SHALL override the default
`.backlog` folder when resolving the project's backlog.

#### Scenario: honors-backlog-directory

```gherkin
      Given a project config sets backlog_directory
      When the plugin resolves the backlog folder
      Then it uses the configured path instead of .backlog
```

#### Gates
# G23: backlog_directory wins over .backlog (mirrors honors-backlog-directory)
#   CHECK: npx vitest run tests/infrastructure/config.test.ts -t "honors backlog_directory when the project root is the vault root"
#   EXPECT: 1 passed
#   EVIDENCE: live probe 2026-11-13 — plugin reads .skillgrid/projectwiki/backlog per the relocated layout

### Requirement: Commands, settings, secret storage

All plugin commands SHALL be registered; settings SHALL persist in
data.json across reloads, except the GitHub token, which SHALL live in
Obsidian's secret storage.

#### Scenario: commands-registered

```gherkin
      When the command palette opens
      Then "Open backlog", "Open sprint board", "Create task", "Create milestone", "Switch project", "Pull issues", and "Edit task" are available
```

#### Scenario: settings-persist

```gherkin
      Given the user changes a setting in the settings tab
      When Obsidian reloads
      Then the setting persists in data.json
```

#### Scenario: token-in-secret-storage

```gherkin
      Given a GitHub token is entered
      When settings are saved
      Then the token is stored in Obsidian secret storage, not data.json
```

#### Gates
# G24: every command is in the palette (mirrors commands-registered)
#   CHECK: manual — open the command palette in the live vault
#   EXPECT: all seven commands listed
#   EVIDENCE: live probe 2026-11-13 — all seven commands present
# G25: settings survive a reload (mirrors settings-persist)
#   CHECK: manual — change a setting, reload Obsidian in the live vault
#   EXPECT: the value persists in data.json and the settings tab
#   EVIDENCE: live probe 2026-11-13 — setting survived reload
# G26: the token never lands in data.json (mirrors token-in-secret-storage)
#   CHECK: manual — save a GitHub token, inspect data.json in the live vault
#   EXPECT: token absent from data.json; present in Obsidian secret storage
#   EVIDENCE: live probe 2026-11-13 — data.json has no token field
