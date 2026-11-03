# ARCHITECTURE.md

**Project:** {project}
**Version:** {version}
**Last Updated:** {last_updated}

> This file describes the **live structure** of the codebase and is the source
> of truth for "how the system is built". It does **not** re-argue decisions —
> architectural decisions are recorded as `### ADR-NNNN` entries in
> `.skillgrid/ASSUMPTIONS.md` (LOCKED) and this file points at them.
>
> **Keep it factual and verifiable.** Every diagram, table row, and code snippet
> must correspond to something real in the repo (a package, a file:line, a
> config key). When you are unsure, write `<detect>` and fill it from the code.

## Table of Contents

1. Architecture Overview
2. Top-Level Layout
3. Application Layering
4. Entry Points & Command Surface
5. Domain / Core Logic
6. Data & Persistence
7. Storage & State
8. Cross-Cutting Transports (API / MCP / HTTP)
9. Frontend / UI (if any)
10. Plugin & Extension Architecture
11. Configuration & Feature Flags
12. Workflow / Pipeline (if any)
13. Enforcement & Quality Gates (tests, hooks, lint)
14. Error Handling & Degradation
15. Security
16. Testing Strategy
17. Known Gaps & TODOs

## 1. Architecture Overview

<!-- Fill: a 2–6 sentence plain-English summary of what the system does and how
     the pieces relate. Then ONE ASCII diagram of the major components and the
     arrows between them. Keep it to what a newcomer needs before reading deeper. -->

<detect: plain-English system summary>

```
<detect: ASCII component diagram of the major subsystems and their relationships>
```

## 2. Top-Level Layout

<!-- Fill: a table of the top-level directories/files and what each is for.
     Only include paths that actually exist. One row per entry. -->

| Path | Purpose |
|---|---|
| <detect: path> | <detect: purpose> |

## 3. Application Layering

<!-- Fill: the layers of the application (e.g. CLI → commands → services →
     domain → infra) as an ASCII diagram, plus the rule for how dependencies
     may flow (which layer may import which). Call out any layering violations. -->

```
<detect: ASCII layering diagram with dependency-direction arrows>
```

**Dependency rules:** <detect: which layer may depend on which; what is forbidden>

## 4. Entry Points & Command Surface

<!-- Fill: how the program starts (binary/entry file), and a table of the
     top-level commands/operations with a one-line description each. -->

- **Entry point:** <detect: file:line of main / entry>
- **Versioning:** <detect: how the version is injected (e.g. -ldflags) and read>

| Command / Operation | What it does |
|---|---|
| <detect: name> | <detect: one-line purpose> |

## 5. Domain / Core Logic

<!-- Fill: the core business logic. A package/module map table (module →
     responsibility) and short notes on the key invariants. Reference the ADRs
     that pin down domain rules instead of restating them. -->

| Module / Package | Responsibility |
|---|---|
| <detect: module> | <detect: responsibility> |

## 6. Data & Persistence

<!-- Fill: the primary data models / schemas, how they flow, and any pipelines
     (ingest → index → search, etc.) as an ASCII diagram. Reference the actual
     schema/migration files. -->

```
<detect: ASCII data-flow diagram>
```

## 7. Storage & State

<!-- Fill: where durable state lives (files, databases, on-disk layouts), the
     storage engine (e.g. SQLite WAL, Postgres), migration strategy, and any
     per-project vs per-user state zones. -->

- **Primary store:** <detect: engine + location pattern>
- **Migrations:** <detect: how schema changes are applied/versioned>
- **State zones:** <detect: per-project vs per-user vs global>

## 8. Cross-Cutting Transports (API / MCP / HTTP)

<!-- Fill: any server-facing surfaces — HTTP API, MCP server, WebSocket, etc.
     List each transport, its port/protocol, and how they share code (a common
     facade/handler set, or separate). Omit this section if there is none. -->

| Transport | Port / Protocol | Shares handlers with |
|---|---|---|
| <detect: e.g. MCP stdio> | <detect> | <detect> |

## 9. Frontend / UI (if any)

<!-- Fill: the UI stack, build tooling, and the dev vs prod split. Omit this
     section if the project has no frontend. -->

- **Stack:** <detect: framework + build tool>
- **Dev vs prod:** <detect: how the two differ>

## 10. Plugin & Extension Architecture

<!-- Fill: how the system extends itself — plugin registry, hook points,
     multi-agent/adapter stages, skill/command registries. Omit if none. -->

<detect: extension model, registry location, and lifecycle>

## 11. Configuration & Feature Flags

<!-- Fill: the config sources and precedence order (CLI flag > env > file),
     the schema of the main config file, and where feature flags live. -->

- **Precedence:** <detect: e.g. CLI > env > config file > default>
- **Main config:** <detect: path + key sections>
- **Feature flags:** <detect: location + naming>

## 12. Workflow / Pipeline (if any)

<!-- Fill: any staged workflow (e.g. SDD phases, CI pipeline) as a diagram with
     per-stage artifacts and the gate that must pass to advance. Omit if none. -->

```
<detect: ASCII pipeline diagram with per-stage artifact + gate>
```

## 13. Enforcement & Quality Gates (tests, hooks, lint)

<!-- Fill: what is machine-enforced — git hooks, CI checks, lint/typecheck
     gates, and the test suite layout. Reference the actual hook/CI files. -->

| Gate | Enforced by | Fails on |
|---|---|---|
| <detect: e.g. typecheck> | <detect: hook/CI file> | <detect> |

## 14. Error Handling & Degradation

<!-- Fill: how errors propagate and, critically, the fail-open/fail-closed
     behavior of each subsystem (what happens when a dependency is missing or
     down). This is the section that prevents "it just works until it doesn't". -->

- **Error convention:** <detect: how errors are wrapped/returned>
- **Degradation floors:** <detect: which subsystems fail open vs closed and why>

## 15. Security

<!-- Fill: an ASCII diagram of the security layers and a table of the controls
     (auth, secrets handling, scan coverage, trust boundaries). Reference the
     actual scan/secret config. -->

```
<detect: ASCII security-layer diagram>
```

| Control | Where | Notes |
|---|---|---|
| <detect: e.g. secret scan> | <detect: config/file> | <detect> |

## 16. Testing Strategy

<!-- Fill: the test pyramid/strategy, where each layer's tests live, the runner,
     and how coverage is measured. Give real file counts / runner commands. -->

- **Runner:** <detect: test command>
- **Layers:** <detect: unit / integration / e2e and where each lives>
- **Coverage:** <detect: how measured, any thresholds>

## 17. Known Gaps & TODOs

<!-- Fill: honest list of things that are stubbed, deferred, or inconsistent.
     This section earns the file its trust — do not leave it empty out of
     politeness. -->

- <detect: gap 1>
- <detect: gap 2>
