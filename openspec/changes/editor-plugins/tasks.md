# Tasks: Editor Plugins

## Epic 1: OpenCode Plugin

- [ ] 1-1 Create `plugins/opencode/` directory with `plugin.json` manifest and `skillgrid.js` entry point
- [ ] 1-2 Implement `config` hook to register skills directory in OpenCode's skill discovery path
- [ ] 1-3 Implement `experimental.chat.messages.transform` hook to prepend bootstrap meta-skill to first user message
- [ ] 1-4 Implement MCP config merge from `config.d/*.yaml` into runtime config
- [ ] 1-5 Implement slash command installation (status, sync, update)
- [ ] 1-6 Write unit tests for MCP merge logic and bootstrap generation
- [ ] 1-7 Verify: install plugin in test project, confirm skills discoverable and bootstrap injected

## Epic 2: Kilo Plugin

- [ ] 2-1 Create `plugins/kilo/` directory with `package.json` (npm package definition) and `plugin.json` manifest
- [ ] 2-2 Implement `src/index.ts` plugin entry exporting the Plugin function
- [ ] 2-3 Implement `config` hook to inspect and modify Kilo config at startup
- [ ] 2-4 Implement `experimental.chat.system.transform` hook to append bootstrap to system prompt
- [ ] 2-5 Implement `event` hook to listen for `session.created` events
- [ ] 2-6 Implement MCP config merge from `config.d/*.yaml`
- [ ] 2-7 Implement slash command installation in `.kilo/command/`
- [ ] 2-8 Write unit tests for MCP merge logic and config parsing
- [ ] 2-9 Publish npm package `skillgrid-kilo-plugin`
- [ ] 2-10 Verify: `kilo plugin skillgrid-kilo-plugin` installs and works end-to-end

## Epic 3: Cursor Plugin

- [ ] 3-1 Create `plugins/cursor/` directory with `plugin.json` manifest
- [ ] 3-2 Implement `hooks/hooks-cursor.json` for session-start bootstrap injection
- [ ] 3-3 Implement MCP config merge from `config.d/*.yaml`
- [ ] 3-4 Implement slash command installation
- [ ] 3-5 Write unit tests for MCP merge logic
- [ ] 3-6 Verify: `/add-plugin skillgrid` installs and works end-to-end

## Epic 4: CLI Integration

- [ ] 4-1 Add `install` subcommand to `skillgrid-cli` with `--target` flag (opencode, kilo, cursor)
- [ ] 4-2 Implement editor auto-detection (check `$PATH` for opencode, kilo, cursor binaries)
- [ ] 4-3 Implement per-target install actions (write opencode.json, run `kilo plugin`, print cursor instructions)
- [ ] 4-4 Add `sync` subcommand to push latest skills and MCP config
- [ ] 4-5 Write unit tests for install detection and config generation
- [ ] 4-6 Verify: `skillgrid install --target=kilo` works end-to-end

## Epic 5: Bootstrap Meta-Skill

- [ ] 5-1 Define canonical bootstrap content (1% rule, workflow enforcement, OpenSpec pipeline reference)
- [ ] 5-2 Ensure all three plugins inject identical bootstrap content
- [ ] 5-3 Verify: "Let's make a react todo list" triggers brainstorming skill before code

## Epic 6: Validation

- [ ] 6-1 Run `openspec validate editor-plugins --type change --strict` before archive
- [ ] 6-2 Archive the change via `openspec archive editor-plugins`
