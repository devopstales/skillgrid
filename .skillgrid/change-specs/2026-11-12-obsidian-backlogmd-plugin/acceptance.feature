Feature: Obsidian Backlog.md Board plugin

  Background:
    Given a vault containing a project folder with a valid backlog config
    And the plugin is enabled

  Scenario: project-discovery-from-scan
    Given a folder "Projects/Demo" contains ".backlog/config.yml"
    When the plugin loads
    Then "Demo" appears in the project switcher

  Scenario: project-discovery-manual-root
    Given settings add a manual root "Extra"
    And "Extra/Demo2" contains a backlog config
    When the plugin rescans
    Then "Demo2" appears in the project switcher

  Scenario: project-discovery-dedupe
    Given a project is both scanned and manually added
    When the project list is computed
    Then it appears only once

  Scenario: frontmatter-roundtrip
    Given a task file "OB-1.md" with frontmatter and a body
    When the plugin reads and rewrites only the status field
    Then the file body is byte-identical
    And only the status field changed

  Scenario: frontmatter-malformed-surfaces-error
    Given a task file with invalid YAML frontmatter
    When the plugin renders the board
    Then an error card is shown and the plugin does not crash

  Scenario: kanban-columns-from-statuses
    Given a project with statuses "To Do, In Progress, Done"
    When the board view opens
    Then three columns render in configured order

  Scenario: kanban-card-in-status-column
    Given a task "OB-1" with status "In Progress"
    When the board renders
    Then "OB-1" appears in the "In Progress" column

  Scenario: drag-move-updates-status
    Given a task "OB-1" in "To Do"
    When the user drags it to "In Progress"
    Then the task file's status becomes "In Progress"
    And the card appears in "In Progress" without full reload

  Scenario: backlog-list-filter
    Given the backlog list shows 5 tasks
    When the user filters by type "bug"
    Then only bug tasks remain
    And clearing the filter restores all tasks

  Scenario: create-task-requires-summary
    Given the create task modal is open
    When the summary is empty
    Then the Create button is disabled
    And when a summary is entered the button is enabled

  Scenario: create-task-writes-file
    Given the create task modal with summary "New work"
    When the user clicks Create
    Then a new file is created under the backlog tasks folder
    And its id uses the project key prefix

  Scenario: edit-task-autosave
    Given an open edit modal for a task
    When the user edits the priority and waits
    Then the file is updated and the modal reports saved

  Scenario: bidirectional-link-add
    Given tasks "OB-1" and "OB-7"
    When a dependency link is added from "OB-1" to "OB-7"
    Then "OB-1" lists "OB-7" in dependencies
    And "OB-7" lists "OB-1" in blocked_by

  Scenario: bidirectional-link-remove
    Given tasks "OB-1" and "OB-7" are linked
    When the link is removed
    Then neither task references the other

  Scenario: blocked-icon
    Given task "OB-4" depends on unfinished "OB-7"
    When the board renders
    Then "OB-4" shows a blocked warning icon

  Scenario: milestone-filter
    Given tasks assigned to milestones "M1" and "M2"
    When the user filters by milestone "M1"
    Then only "M1" tasks remain

  Scenario: milestone-list-loads
    Given a project with milestone notes under milestones/
    When the milestones view opens
    Then the milestones are listed

  Scenario: watcher-refresh-on-external-edit
    Given the board is open
    When a task file is modified externally
    Then the board updates within 1 second

  Scenario: github-pull-creates-tasks
    Given a project linked to a GitHub repository
    When "Pull issues" runs
    Then new task files are created with a source property

  Scenario: github-pull-syncs-without-overwrite
    Given previously pulled tasks
    When "Pull issues" runs again
    Then title, assignee, and open/closed state update
    And sprint, rank, points, priority, parent, links, and description are not overwritten

  Scenario: github-pull-descriptions-code-blocked
    Given descriptions are pulled by default
    When a task note is inspected
    Then the description is stored as plain text in a code block

  Scenario: commands-registered
    When the command palette opens
    Then "Open backlog", "Open sprint board", "Create task", "Create milestone", "Switch project", "Pull issues", and "Edit task" are available

  Scenario: cmd-click-opens-note
    Given a card on the board
    When the user Cmd-clicks it
    Then the task note opens in a new tab

  Scenario: settings-persist
    Given the user changes a setting in the settings tab
    When Obsidian reloads
    Then the setting persists in data.json

  Scenario: token-in-secret-storage
    Given a GitHub token is entered
    When settings are saved
    Then the token is stored in Obsidian secret storage, not data.json

  Scenario: honors-backlog-directory
    Given a project config sets backlog_directory
    When the plugin resolves the backlog folder
    Then it uses the configured path instead of .backlog
