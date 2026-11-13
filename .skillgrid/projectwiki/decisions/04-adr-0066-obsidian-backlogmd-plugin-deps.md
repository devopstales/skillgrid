---
type: TechnicalDecision
title: "Obsidian Backlog.md plugin: runtime deps gray-matter + js-yaml, build/test toolchain esbuild + typescript + vitest"
status: "accepted"
supersedes: none
date: 2026-11-13
---

# Obsidian Backlog.md plugin: dependency set

## Context and Problem Statement

The change `2026-11-12-obsidian-backlogmd-plugin` adds an Obsidian desktop
plugin (`plugins/obsidian/backlogmd/`) that reads and writes the Backlog.md
layout (`backlog.config.yml` → `.skillgrid/projectwiki/backlog/`). It is the
repo's first TypeScript module and therefore introduces the repo's first
npm dependency set:

- **Runtime** (bundled into `main.js`): `gray-matter ^4.0.3` (frontmatter
  parse/serialize for task + milestone files), `js-yaml ^4.1.0` (the
  advanced root config `backlog.config.yml` and legacy `config.yml`).
- **Dev/build** (never shipped): `esbuild`, `typescript`, `vitest`,
  `obsidian` (type declarations, external at bundle time), `@types/node`,
  `builtin-modules`.

AGENTS.md ("No new dependencies without an ADR") requires this decision to be
recorded before it lands (precedent: ADR-0008, the `yaml` package).

## Considered Options

- **A. `gray-matter` + explicit `js-yaml`** — `gray-matter` is the standard
  Obsidian-plugin frontmatter library (it ships in the official sample
  plugin); it parses with `yaml.safeLoad` and stringifies with
  `yaml.safeDump` (verified in the installed tree — no `!!js/*` schema
  reachable). `js-yaml` is already a transitive dependency of `gray-matter`
  (locked 4.3.2); the plugin also uses it directly in
  `src/infrastructure/config.ts`, so it is declared explicitly per the
  direct-dependency rule — a consistent, not redundant, choice.
- **B. `yaml` (eemeli) for both paths** — one parser for frontmatter and
  config. Rejected: frontmatter round-tripping (quote style, field order,
  key order preservation) is exactly what `gray-matter`/`matter.stringify`
  is built for, and Obsidian plugin conventions standardize on it; mixing
  two frontmatter libraries across the plugin family would be a worse
  consistency trade.
- **C. Hand-rolled frontmatter parser** — no runtime dependency. Rejected:
  the task frontmatter is real YAML (lists, nested values on GitHub-pulled
  fields, arbitrary user fields the plugin must pass through); a regex
  parser would silently mis-handle the cases ADR-0008 rejected regex for.

## Decision Outcome

Chosen option: **A** — `gray-matter ^4.0.3` + `js-yaml ^4.1.0` as the two
runtime dependencies; `esbuild` (bundle, `obsidian` marked external),
`typescript` (type-check via `tsc -noEmit`), `vitest` (unit tests),
`obsidian`/`@types/node`/`builtin-modules` as dev-only. All are
MIT/Apache-2.0, actively maintained, the standard Obsidian-plugin stack, and
the `package-lock.json` is committed (lockfileVersion 3, registry-pinned
with integrity hashes; security review 2026-11-13 found no known
vulnerabilities at the locked versions). The `obsidian` devDependency is
pinned to the lockfile-resolved version (review finding M13).

### Implementation

- `plugins/obsidian/backlogmd/package.json` — the dependency set (runtime:
  `gray-matter`, `js-yaml`; dev: the toolchain).
- `plugins/obsidian/backlogmd/esbuild.config.mjs` — bundle config
  (`obsidian` external; no sourcemap in prod).
- `plugins/obsidian/backlogmd/tsconfig.json`, `vitest` under `tests/`.

### Consequences

- Good, because the dependency set is the minimal standard Obsidian-plugin
  stack — nothing exotic, nothing moving at runtime, all permissive
  licenses, all lockfile-pinned.
- Good, because safe-schema YAML parsing on both paths was verified in the
  installed tree (no code-execution path via `!!js/function`), so the two
  parsers do not weaken the security posture of task-file round-trips.
- Bad, because the repo now carries a second toolchain (npm alongside Go) —
  `npm install` is a prerequisite for building/testing the plugin.
  Mitigation: the plugin is a self-contained subproject with its own
  `package.json`/lockfile; nothing outside `plugins/obsidian/backlogmd/`
  imports its dependencies.
- Bad, because `gray-matter` is in maintenance mode (stable, but slow to
  move). Mitigation: it wraps js-yaml, whose pinned version is under our
  control; a migration is a one-library swap behind `parseTask`/
  `serializeTask`.

### Revisit criteria

Revisit this ADR (write a superseding ADR) when **any** of:
1. A second Obsidian plugin (or another frontmatter consumer) lands — at
   which point the frontmatter library choice should be a shared decision
   across the plugin family, not a per-plugin one.
2. `gray-matter` is archived or its major version moves with breaking
   frontmatter semantics — re-evaluate option B (`yaml` for both paths).
3. The plugin gains a server/daemon mode (it is desktop-only today,
   `isDesktopOnly: true`) — the Node-`fs` assumptions and the dependency
   boundary (bundle vs runtime) need re-examination.
