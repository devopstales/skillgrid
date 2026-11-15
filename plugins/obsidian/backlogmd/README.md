# backlogmd — Obsidian plugin

Renders Backlog.md task files as an Obsidian board: kanban sprint view, backlog view (milestone sections with start/complete actions, plus a draft box), milestone view, and task create/edit modals. Writes go through the `backlog` CLI (`--no-git --zero-padded-ids 3`); the plugin never edits task files directly.

## Prerequisites

- Node.js 20+ (build), Obsidian 1.7+ (runtime)
- `backlog` CLI on PATH (the plugin shells out to it for writes; reads are direct)
- Backlog layout: a root `backlogmd.config.yml` with `backlog_directory` (default `.backlog`; this repo uses `.skillgrid/projectwiki/backlog`), and a `tasks/` directory under it. Milestones live in `milestones/` alongside.

## Multi-project

`manual_roots` in `data.json` (or the "Manual vault roots" setting, one vault path per line) registers additional vault roots. Each root's `backlogmd.config.yml` describes a project. The project switcher in the backlog/kanban views selects which project is shown; the "Switch project" command cycles the open view.

## Development

```sh
npm install
npm run dev        # watch build → main.js
npm test           # vitest (unit: domain + infrastructure)
npm run build      # tsc -noEmit (strict) + production bundle
```

Install: copy `main.js`, `manifest.json`, `styles.css` into a vault's `.obsidian/plugins/backlogmd/`, or use the plugin's source folder via a symlink in development.

## Behavior notes

- **Reads** are direct file reads (no CLI round-trip); **writes** always go through the `backlog` CLI (`task edit` for status/milestone moves, `task create`, `milestone create/start/complete`).
- A task file with malformed frontmatter is isolated: it appears in an error card at the bottom of the backlog/kanban view instead of a board column, and it is never re-rendered as a ghost card.
- Board updates are debounced (300 ms) and fire only for files under a project's `tasks/` or `milestones/` directory, including CLI renames.
- GitHub issue pull is available via the "Pull issues" command (token stored in Obsidian's secret storage) and from the backlog view toolbar.
