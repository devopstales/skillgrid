## ADDED Requirements

### Requirement: Change list view

The system SHALL provide a terminal UI view listing all OpenSpec changes.

#### Scenario: List all changes
- **GIVEN** the TUI is launched
- **WHEN** the change list view loads
- **THEN** it displays all changes with name, status, progress, and last-modified columns

#### Scenario: Filter by status
- **GIVEN** the change list is displayed
- **WHEN** the user presses a filter key (e.g., `f` for in-progress, `c` for complete)
- **THEN** the list filters to show only matching changes

#### Scenario: Sort changes
- **GIVEN** the change list is displayed
- **WHEN** the user presses a sort key (e.g., `s` for last-modified)
- **THEN** the list re-sorts by the selected column

### Requirement: Task list view

The system SHALL provide a terminal UI view listing tasks for a selected change.

#### Scenario: Open change tasks
- **GIVEN** a change is selected in the list view
- **WHEN** the user presses Enter
- **THEN** the task list view opens showing tasks grouped by epic

#### Scenario: Toggle task checkbox
- **GIVEN** the task list view is displayed
- **WHEN** the user presses Space on a task
- **THEN** the task checkbox toggles between checked/unchecked and persists to tasks.md

#### Scenario: Task progress indicator
- **GIVEN** the task list view is displayed
- **WHEN** tasks are partially complete
- **THEN** a progress bar shows completed/total tasks

### Requirement: Keyboard navigation

The system SHALL support vim-style keyboard navigation.

#### Scenario: Vim movement
- **GIVEN** the TUI is focused
- **WHEN** the user presses `j`/`k` or arrow keys
- **THEN** the selection moves down/up

#### Scenario: Quick quit
- **GIVEN** the TUI is focused
- **WHEN** the user presses `q` or `Ctrl+C`
- **THEN** the TUI exits cleanly

#### Scenario: Search
- **GIVEN** the change list is displayed
- **WHEN** the user presses `/` and types a query
- **THEN** the list filters to matching items in real-time

### Requirement: Data integration

The TUI SHALL read from and write to OpenSpec's existing data sources.

#### Scenario: Read from openspec CLI
- **GIVEN** the TUI needs change/task data
- **WHEN** it fetches data
- **THEN** it calls `openspec list --json` and `openspec status --change <name> --json`

#### Scenario: Write task state
- **GIVEN** the user toggles a task checkbox
- **WHEN** the toggle persists
- **THEN** it updates the `- [ ]` / `- [x]` state in `tasks.md`

#### Scenario: Refresh data
- **GIVEN** the TUI is idle
- **WHEN** the user presses `r`
- **THEN** it re-fetches data from the openspec CLI
