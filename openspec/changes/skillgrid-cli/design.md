## Context

`skillgrid-cli` is a Go CLI binary that installs Skillgrid's AI-assisted development hub onto a machine. It clones (or pulls) the Skillgrid repo, installs selected agents via npm, installs shared global tools, and copies `.agents/` into `~/.agents/`.

The CLI is built with `task` (Taskfile.yml) and supports cross-compilation to linux (amd64+386) and darwin (amd64+arm64). Version is derived from `git describe --tags --always --dirty`.

## Goals / Non-Goals

**Goals:**
- Document all CLI commands, flags, and their semantics
- Document the install pipeline step ordering
- Document agent selection (interactive and preset)
- Document the config system (Config struct, agents, tools)
- Document error handling and dry-run behavior
- Serve as canonical reference for operators and contributors

**Non-Goals:**
- Changing CLI behavior
- Adding new commands or flags
- Documenting MemIndex or editor plugin CLI commands (those are separate changes)

## Decisions

### 1. Documentation format

**Decision:** Use OpenSpec change format (proposal, design, specs, adr, tasks) even though this is documentation-only.

**Alternatives considered:**
- A standalone markdown doc in `docs/` — rejected: doesn't follow the OpenSpec convention already established in this repo
- Code comments only — rejected: not discoverable as a reference

**Rationale:** Consistency with the repo's documentation approach. The OpenSpec format allows the CLI to be referenced alongside the changes it enables.

### 2. Scope

**Decision:** Document only the existing `skillgrid-cli` (install + sync-repo). Do not document planned commands (`mcp`, `serve`, `index`, `setup`, `sync`, `install --target`).

**Alternatives considered:**
- Include planned commands — rejected: those are separate OpenSpec changes (editor-plugins, memindex)
- Document only commands, not internals — rejected: operators need to understand the install pipeline

**Rationale:** This document describes what exists. Future commands are documented in their own changes.

## Risks / Trade-offs

- **Doc drift** -> Document is tied to source code file/line references; update when code changes
- **Over-documentation** -> Focus on behavior and contracts, not implementation details

## Migration Plan

None — documentation-only change.

## Open Questions

None — this documents existing, implemented behavior.
