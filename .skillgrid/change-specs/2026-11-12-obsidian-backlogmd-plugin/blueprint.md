# Obsidian Backlog.md Board — Implementation Blueprint (steps, files, verification)

> **For agentic workers:** REQUIRED SUB-SKILL: Use subagent-execution (recommended) or simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** PROPOSED

**Tier:** T2

**Build shape:** Tracer thread

**Goal:** Ship an Obsidian plugin under `plugins/obsidian/backlogmd/` that renders backlog.md task files as a kanban board + backlog list and persists create/edit/move operations to frontmatter, honoring a configurable backlog directory.

**Architecture:** BoardMD-style layered TypeScript: `src/domain` (pure rules, no Obsidian imports), `src/application` (commands/queries over ports), `src/infrastructure` (vault + frontmatter + watcher + GitHub/GitLab client), `src/settings` (declarative settings tab), `src/ui` (views/modals/drag-drop), wired in `src/compose.ts`. Mutations rewrite only YAML frontmatter; body is preserved. Plugin resolves the backlog directory from each project's backlog.md config (default `.backlog/`), proposing `.skillgrid/projectwiki/backlog` as the canonical skillgrid location.

**Tech Stack:** TypeScript, Obsidian API ≥1.13, esbuild, vitest, `gray-matter` (frontmatter parse/serialize), `js-yaml`.

**Spec:** `./briefing.md`

## Terms

- backlog directory — the folder holding `config.yml` + `tasks/` etc.; resolved from backlog.md config (`backlog_directory`), default `.backlog/`. See `.skillgrid/projectwiki/glossary/02-technical-terms.md`.
- task — a Markdown file with YAML frontmatter + body description. See briefing Terms.
- status — one of a project's configured statuses; columns are statuses.

## Hypothesis

**Claim:** An Obsidian plugin can render backlog.md task files as a kanban board and persist status changes by rewriting frontmatter, with no schema change and no generated sidecar files, while staying in sync with external (CLI) edits.
**Right condition:** Dragging a card to a new column rewrites only the `status` field; the file round-trips byte-for-byte otherwise; external edits refresh the view within ~1s.
**Wrong condition:** Frontmatter serialization alters body or field order in a way that breaks the backlog CLI or diff review; or the watcher fails to refresh.
**Thinnest MVP:** Task 1–6 (scaffold + domain Task model + frontmatter adapter + a single Kanban view rendering one project's tasks, read-only).
**Door check:** Task 3 (frontmatter round-trip test) — if round-trip is not byte-stable, stop and pick a different serializer before building UI.

## Must-Haves (goal-backward verification)

**Truths:**
- A vault folder containing a backlog config is auto-discovered and appears in the project switcher.
- Every task file renders as a card in the column matching its `status`.
- Dragging a card to another column rewrites only that task's `status` frontmatter and writes the file.
- Create/edit forms persist via frontmatter round-trip; body is preserved.
- Dependency/parent links are written on both sides; a task blocked by an unfinished dependency shows a warning icon.
- External edits refresh the views within ~1s.
- Desktop only; Obsidian ≥1.13.

**Artifacts:**
- `plugins/obsidian/backlogmd/package.json`, `tsconfig.json`, `esbuild.config.mjs`, `manifest.json`, `versions.json`
- `plugins/obsidian/backlogmd/src/domain/task.ts` and pure rules
- `plugins/obsidian/backlogmd/src/infrastructure/vault.ts` (frontmatter adapter)
- `plugins/obsidian/backlogmd/src/ui/kanban-view.ts`, `backlog-view.ts`
- `plugins/obsidian/backlogmd/src/main.ts`
- Passing vitest suite for domain + application

**Key links:**
- `MoveTask` must call the vault adapter's `updateFrontmatter`, and the updated task must be re-readable with the new status.
- Create/edit modal must route through `application` commands, not touch the vault directly.
- Link edits must write both task files.

**One-way-door decisions:** None (all choices are reversible local implementation details; no migration, no public API, no data model change beyond frontmatter fields already defined by backlog.md).

**Deferred (not blocking the build):** relocating this repo's own backlog to `.skillgrid/projectwiki/backlog` is an operational step outside this plugin's scope; the plugin supports any configured `backlog_directory`. Final decision recorded separately.

## Global Constraints

- Obsidian API ≥1.13; desktop only (no mobile handling).
- TypeScript strict mode; no `any` in domain/application.
- No generated `index.json`/`AGENTS.md` files (backlog CLI owns them).
- Mutations rewrite only YAML frontmatter; body preserved.
- Backlog directory resolved from config (`backlog_directory`), default `.backlog/`.
- No new Go dependencies (plugin is TS-only; Go build unaffected).

## File Structure

```
plugins/obsidian/backlogmd/
  package.json
  tsconfig.json
  esbuild.config.mjs
  manifest.json
  versions.json
  styles.css
  src/
    main.ts
    compose.ts
    domain/
      task.ts
      status.ts
      ranking.ts
      links.ts
    application/
      ports.ts
      list-tasks.ts
      move-task.ts
      create-task.ts
      edit-task.ts
      link-tasks.ts
    infrastructure/
      vault.ts
      config.ts
      watcher.ts
      github.ts
    settings/
      settings-tab.ts
    ui/
      kanban-view.ts
      backlog-view.ts
      task-modal.ts
      project-switcher.ts
  tests/
    domain/
      task.test.ts
      links.test.ts
    application/
      move-task.test.ts
    infrastructure/
      vault.test.ts
```

Rejected: putting the plugin at repo root or under `skillgrid-ui/` — skillgrid-ui was removed (ADR-0059); `plugins/obsidian/` matches the existing per-agent plugin asset convention. Rejected: shelling out to the `backlog` CLI for mutations — adds a runtime dependency on the CLI being installed; direct frontmatter edits keep the plugin self-contained.

No alternatives recorded in briefing beyond the above.

---

### Task 1: Scaffold the plugin project

**Files:**
- Create: `plugins/obsidian/backlogmd/package.json`
- Create: `plugins/obsidian/backlogmd/tsconfig.json`
- Create: `plugins/obsidian/backlogmd/esbuild.config.mjs`
- Create: `plugins/obsidian/backlogmd/manifest.json`
- Create: `plugins/obsidian/backlogmd/versions.json`
- Create: `plugins/obsidian/backlogmd/styles.css`
- Create: `plugins/obsidian/backlogmd/src/main.ts`

**Interfaces:**
- Consumes: none
- Produces: plugin entry point `main.ts` exporting default `Plugin` subclass
- Seam: none (in-process)
- Deletion test: pass-through, delete it
- Adapters: 1 (Obsidian runtime)

**SATISFIES:** (scaffolding — no scenario)

- [ ] **Step 1: Create package.json**

```json
{
  "name": "backlogmd-obsidian",
  "version": "0.1.0",
  "scripts": {
    "dev": "node esbuild.config.mjs",
    "build": "tsc -noEmit -skipLibCheck && node esbuild.config.mjs production",
    "test": "vitest run",
    "lint": "eslint src"
  },
  "devDependencies": {
    "@types/node": "^20.0.0",
    "esbuild": "^0.21.0",
    "typescript": "^5.4.0",
    "vitest": "^1.6.0",
    "obsidian": "latest"
  },
  "dependencies": {
    "gray-matter": "^4.0.3",
    "js-yaml": "^4.1.0"
  }
}
```

- [ ] **Step 2: Create tsconfig.json**

```json
{
  "compilerOptions": {
    "baseUrl": ".",
    "inlineSourceMap": true,
    "inlineSources": true,
    "module": "ESNext",
    "target": "ES2022",
    "allowJs": true,
    "noImplicitAny": true,
    "moduleResolution": "bundler",
    "importHelpers": true,
    "isolatedModules": true,
    "strictNullChecks": true,
    "lib": ["DOM", "ES2022"]
  },
  "include": ["src/**/*.ts"]
}
```

- [ ] **Step 3: Create esbuild.config.mjs**

```js
const prod = process.argv[2] === "production";
require("esbuild").build({
  entryPoints: ["src/main.ts"],
  bundle: true,
  external: ["obsidian"],
  format: "cjs",
  target: "es2022",
  logLevel: "info",
  sourcemap: prod ? false : "inline",
  treeShaking: true,
  outfile: "main.js",
  minify: prod,
}).catch(() => process.exit(1));
```

- [ ] **Step 4: Create manifest.json and versions.json** (id `backlogmd`, name `Backlog.md Board`, minAppVersion `1.13.0`, desktop only via `isDesktopOnly: true`).

- [ ] **Step 5: Create minimal main.ts**

```ts
import { Plugin } from "obsidian";
export default class BacklogMdPlugin extends Plugin {
  async onload() {
    // wiring added in later tasks
  }
  onunload() {}
}
```

- [ ] **Step 6: Install deps and verify build**

Run: `cd plugins/obsidian/backlogmd && npm install && npm run build`
Expected: `main.js` produced, no type errors.

- [ ] **Step 7: Commit**

```bash
git add plugins/obsidian/backlogmd
git commit -m "feat(obsidian): scaffold backlogmd plugin"
```

---

### Task 2: Domain — Task model and pure rules

**Files:**
- Create: `plugins/obsidian/backlogmd/src/domain/task.ts`
- Create: `plugins/obsidian/backlogmd/src/domain/status.ts`
- Create: `plugins/obsidian/backlogmd/tests/domain/task.test.ts`

**Interfaces:**
- Consumes: none
- Produces: `Task` interface, `isDone(status, doneStatuses)`, `isBlocked(task, allTasks, doneStatuses)`
- Seam: none (in-process pure functions)
- Deletion test: if deleted, task model reappears inline in application; keep it
- Adapters: 1 (unit tests)

**SATISFIES:** `task-model` (unit-level)

- [ ] **Step 1: Write failing test** — assert `isBlocked` returns true when a dependency's status is not done; false otherwise; `isDone` matches configured done statuses.

- [ ] **Step 2: Run test to verify it fails**

- [ ] **Step 3: Implement**

```ts
export interface Task {
  id: string;
  title: string;
  status: string;
  type?: string;
  priority?: string;
  labels?: string[];
  assignee?: string;
  milestone?: string;
  createdDate?: string;
  updatedDate?: string;
  dependencies?: string[];
  references?: string[];
  documentation?: string[];
  parentTaskId?: string;
  body: string;
  filePath: string;
}

export function isDone(status: string, doneStatuses: string[]): boolean {
  return doneStatuses.includes(status);
}

export function isBlocked(task: Task, all: Task[], doneStatuses: string[]): boolean {
  if (!task.dependencies?.length) return false;
  return task.dependencies.some(depId => {
    const dep = all.find(t => t.id === depId);
    return !dep || !isDone(dep.status, doneStatuses);
  });
}
```

Done statuses derived as configured statuses whose name matches /done|completed|closed/i (configurable override in settings).

- [ ] **Step 4: Run tests — PASS**

- [ ] **Step 5: Commit**

---

### Task 3: Infrastructure — frontmatter vault adapter (door check)

**Files:**
- Create: `plugins/obsidian/backlogmd/src/infrastructure/vault.ts`
- Create: `plugins/obsidian/backlogmd/tests/infrastructure/vault.test.ts`

**Interfaces:**
- Consumes: Obsidian `Vault` (injected)
- Produces: `readTask(file): Task`, `updateFrontmatter(file, patch): void`, `createTask(dir, task): string`
- Seam: `VaultAdapter` interface (port) in `application/ports.ts`
- Deletion test: if deleted, frontmatter parsing reappears in every command; keep it
- Adapters: 2 (real Obsidian vault, in-memory test vault)

**SATISFIES:** `frontmatter-roundtrip`

- [ ] **Step 1: Write failing round-trip test** — parse a fixture task file, serialize with a changed `status`, assert body byte-identical and only `status` changed.

- [ ] **Step 2: Run test to verify it fails**

- [ ] **Step 3: Implement using gray-matter**

```ts
import matter from "gray-matter";
export function parseTask(raw: string, filePath: string): Task { ... }
export function serializeTask(task: Task): string { return matter.stringify(task.body, frontmatterObj); }
```

Preserve field order by building the frontmatter object in a stable key order. Test against fixtures including multiline body, arrays, quoted wikilinks.

- [ ] **Step 4: Run tests — PASS** (door check: if round-trip not byte-stable, stop and switch serializer)

- [ ] **Step 5: Commit**

---

### Task 4: Infrastructure — config loader + project discovery

**Files:**
- Create: `plugins/obsidian/backlogmd/src/infrastructure/config.ts`
- Create: `plugins/obsidian/backlogmd/src/application/ports.ts`

**Interfaces:**
- Consumes: Obsidian `Vault`
- Produces: `ProjectConfig` (statuses, types, priorities, labels, defaultStatus, backlogDir), `discoverProjects(vault, manualRoots): Project[]`
- Seam: `ConfigPort` in `ports.ts`
- Deletion test: project discovery reappears in main.ts; keep it
- Adapters: 2 (real vault, test vault)

**SATISFIES:** `project-discovery`

- [ ] **Step 1: Write failing tests** — discover a folder with `config.yml`; honor `backlog_directory` override; merge manual roots; dedupe.

- [ ] **Step 2: Run tests to verify they fail**

- [ ] **Step 3: Implement** — read `config.yml` via `js-yaml`; resolve `backlogDir` (default `.backlog`); walk vault for folders whose `<backlogDir>/config.yml` exists.

- [ ] **Step 4: Run tests — PASS**

- [ ] **Step 5: Commit**

---

### Task 5: Application — commands and queries

**Files:**
- Create: `plugins/obsidian/backlogmd/src/application/list-tasks.ts`
- Create: `plugins/obsidian/backlogmd/src/application/move-task.ts`
- Create: `plugins/obsidian/backlogmd/src/application/create-task.ts`
- Create: `plugins/obsidian/backlogmd/src/application/edit-task.ts`
- Create: `plugins/obsidian/backlogmd/src/application/link-tasks.ts`
- Create: `plugins/obsidian/backlogmd/tests/application/move-task.test.ts`

**Interfaces:**
- Consumes: `VaultPort`, `ConfigPort`
- Produces: `listTasks(project)`, `moveTask(task, newStatus)`, `createTask(project, input)`, `editTask(task, patch)`, `linkTasks(a, b, kind)`
- Seam: ports (injected)
- Deletion test: if deleted, command logic reappears in UI handlers; keep it
- Adapters: 1 (mocked ports in tests)

**SATISFIES:** `move-task`, `create-task`, `edit-task`, `link-tasks`

- [ ] **Step 1: Write failing tests** for `moveTask` (updates status, writes file, returns updated task) and `linkTasks` (writes both sides).

- [ ] **Step 2: Run tests to verify they fail**

- [ ] **Step 3: Implement** commands delegating to ports; `createTask` generates id from project config prefix + next number.

- [ ] **Step 4: Run tests — PASS**

- [ ] **Step 5: Commit**

---

### Task 6: Composition root

**Files:**
- Create: `plugins/obsidian/backlogmd/src/compose.ts`
- Modify: `plugins/obsidian/backlogmd/src/main.ts`

**Interfaces:**
- Consumes: all layers
- Produces: wired `AppContext` (ports + commands) attached to plugin
- Seam: none
- Deletion test: pass-through
- Adapters: 1

**SATISFIES:** (wiring — no scenario)

- [ ] **Step 1: Implement compose.ts** instantiating vault/config adapters and commands; attach to plugin instance.

- [ ] **Step 2: Update main.ts** to call compose in `onload`, register views/commands/settings.

- [ ] **Step 3: Build — PASS**

- [ ] **Step 4: Commit**

---

### Task 7: UI — Kanban board view

**Files:**
- Create: `plugins/obsidian/backlogmd/src/ui/kanban-view.ts`
- Modify: `plugins/obsidian/backlogmd/src/main.ts` (register view type)

**Interfaces:**
- Consumes: `AppContext`, Obsidian `ItemView`
- Produces: `KanbanView` rendering columns per status, cards with blocked icon, drag-and-drop calling `moveTask`
- Seam: Obsidian view registration
- Deletion test: if deleted, board UI reappears inline; keep it
- Adapters: 1 (Obsidian runtime)

**SATISFIES:** `kanban-board`, `drag-move`

- [ ] **Step 1: Implement view** — render columns from project statuses; render cards; HTML5 drag-and-drop between columns invokes `moveTask` then re-renders.

- [ ] **Step 2: Register view type and command "Open board"** in main.ts.

- [ ] **Step 3: Build — PASS**

- [ ] **Step 4: Commit**

---

### Task 8: UI — Backlog list view

**Files:**
- Create: `plugins/obsidian/backlogmd/src/ui/backlog-view.ts`

**Interfaces:**
- Consumes: `AppContext`
- Produces: `BacklogView` with filter box (status/type/priority/label/milestone) and create button

**SATISFIES:** `backlog-list`, `filter`

- [ ] **Step 1: Implement** list grouped by status, live filter, create button opens modal.

- [ ] **Step 2: Register view + command "Open backlog"**

- [ ] **Step 3: Build — PASS**

- [ ] **Step 4: Commit**

---

### Task 9: UI — Task create/edit modal

**Files:**
- Create: `plugins/obsidian/backlogmd/src/ui/task-modal.ts`

**Interfaces:**
- Consumes: `AppContext`, Obsidian `Modal`
- Produces: `TaskModal` with summary-required validation, dropdowns, type-ahead pickers, autosave

**SATISFIES:** `create-task`, `edit-task`, `autosave`

- [ ] **Step 1: Implement modal** — form fields; Create button disabled until summary non-empty; autosave debounce ~500ms; "Done" flushes.

- [ ] **Step 2: Wire** "Create task" command and card click → edit modal; Cmd/Ctrl-click opens note in new tab.

- [ ] **Step 3: Build — PASS**

- [ ] **Step 4: Commit**

---

### Task 10: File watcher refresh

**Files:**
- Create: `plugins/obsidian/backlogmd/src/infrastructure/watcher.ts`
- Modify: `plugins/obsidian/backlogmd/src/compose.ts`

**Interfaces:**
- Consumes: Obsidian `Vault` events
- Produces: debounced (~300ms) refresh callback for active views

**SATISFIES:** `watcher-refresh`

- [ ] **Step 1: Implement watcher** — listen to `vault.on('modify'|'create'|'delete')` for task/config files; debounce; trigger view re-render.

- [ ] **Step 2: Wire into compose + views**

- [ ] **Step 3: Build — PASS**

- [ ] **Step 4: Commit**

---

### Task 11: Settings tab

**Files:**
- Create: `plugins/obsidian/backlogmd/src/settings/settings-tab.ts`

**Interfaces:**
- Consumes: `AppContext`, Obsidian `PluginSettingTab`
- Produces: declarative settings (Obsidian ≥1.13): manual roots, active project, workflow view/edit, repo link + token (secret storage), description render toggle

**SATISFIES:** `settings`

- [ ] **Step 1: Implement settings tab** using `Setting` with `addText`/`addDropdown` etc.

- [ ] **Step 2: Register in main.ts**

- [ ] **Step 3: Build — PASS**

- [ ] **Step 4: Commit**

---

### Task 12: Milestones view

**Files:**
- Create: `plugins/obsidian/backlogmd/src/ui/milestone-view.ts` (or extend backlog view)

**Interfaces:**
- Consumes: `AppContext`
- Produces: milestone grouping/filtering; load from `.backlog/milestones/`

**SATISFIES:** `milestones`

- [ ] **Step 1: Implement** milestone filter + list.

- [ ] **Step 2: Build — PASS**

- [ ] **Step 3: Commit**

---

### Task 13: GitHub/GitLab pull

**Files:**
- Create: `plugins/obsidian/backlogmd/src/infrastructure/github.ts`
- Modify: `plugins/obsidian/backlogmd/src/application/` (add `pull-issues.ts`)

**Interfaces:**
- Consumes: `VaultPort`, `ConfigPort`, fetch API
- Produces: `pullIssues(project)` creating/updating task notes with `source` property; descriptions code-blocked by default

**SATISFIES:** `github-pull`

- [ ] **Step 1: Implement client** + pull command with token from secret storage.

- [ ] **Step 2: Wire command "Pull issues"**

- [ ] **Step 3: Build — PASS**

- [ ] **Step 4: Commit**

---

### Task 14: Project switcher + note header button

**Files:**
- Create: `plugins/obsidian/backlogmd/src/ui/project-switcher.ts`
- Modify: `plugins/obsidian/backlogmd/src/main.ts`

**Interfaces:**
- Consumes: `AppContext`
- Produces: dropdown switching active project; "Edit task" button in note header

**SATISFIES:** `switch-project`, `edit-from-note`

- [ ] **Step 1: Implement switcher** in board/backlog views.

- [ ] **Step 2: Add note header button via `registerMarkdownPostProcessor` or view header.**

- [ ] **Step 3: Build — PASS**

- [ ] **Step 4: Commit**

---

## Review Summary

**Verdict:** (to be filled by goldfish pipeline)

**Rounds run:** TBD

**Findings by severity:**
- CRITICAL: none yet
- WARNING: none yet
- SUGGESTION: none yet

**Parked findings:** none

**Reviewed:** TBD
