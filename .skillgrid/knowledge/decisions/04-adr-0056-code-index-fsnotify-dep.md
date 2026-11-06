---
type: TechnicalDecision
status: in-force
supersedes: none
date: 2026-11-02
---

# ADR-0044 — Code-index fsnotify dependency (watch loop)


Change: `2026-10-31-mnemonic-v2-code-index` (spec zone: `adr.md` § ADR-0044)

## Context

The watch loop (TICKET-06) needs filesystem events to keep the graph live. v1 uses
`fsnotify` (v1.10.1). New to `mnemonic-v2/go.mod`.

## Decision

Add `github.com/fsnotify/fsnotify v1.10.1` to `mnemonic-v2/go.mod`.

## Alternatives

- **Polling** — rejected: the spec (`007-watch`) requires a trigger, not a poll
  (debounce + burst-collapse presuppose events).
- **`inotify` directly** — rejected: `fsnotify` is the cross-platform abstraction v1
  uses.

## Consequences

The watch loop is a background goroutine (a process-integration threat row). `fsnotify`
is stable (v1.10.x); the watch is per-process (opt-in, not a daemon). The dep lands at
TICKET-06, gated by this ADR.
