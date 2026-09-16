# Trust boundary: MCP-spawned process with project read access

---
status: "accepted"
supersedes: none
date: 2026-09-16
---

## Context and Problem Statement

The PRD's non-functional requirements section needs a trust-boundary statement.
The local-first posture (data stays in `~/.skillgrid/mnemonic/`, HTTP binds
127.0.0.1, no telemetry, no network except install-time npm/git and optional
external embedder) is the core, but there is one real boundary that a future
reader will wonder about: when an agent registers `skillgrid mcp` as a stdio
server, the agent process *spawns the binary*, and the binary then has full
read access to the project it indexes. That is a trust decision, not an
implementation detail.

## Considered Options

- **Local-first only** — data stays local, HTTP is 127.0.0.1, no telemetry; stop there
- **Local-first + MCP-spawn model named** — add the one real boundary: the agent process spawns the binary, which has project read access
- **Local-first + MCP-spawn + `doctor --strict` redaction guarantee** — also spell out the redaction guarantee in the trust section

## Decision Outcome

Chosen option: "Local-first + MCP-spawn model named," because the MCP-spawn
model is the one boundary that is (a) surprising without context (a local CLI
that "just indexes code" is actually a process the agent spawns with project
read access) and (b) hard to reverse (changing who spawns whom is an interface
change across the Surface). The `doctor --strict` redaction guarantee is a
`doctor` detail that belongs in the command reference, not the trust section —
promoting it here would blur the line between "trust boundary" and "command
behavior."

### Consequences

- Good, because the PRD names the one real boundary in one sentence, and the
  threat-matrix rows for the Surface (who can call the HTTP API, what the
  stdio server can read) are grounded in it.
- Good, because the local-first posture is stated as a positive claim (what
  the product does NOT do: no telemetry, no network except named exceptions)
  rather than just a list of what it does.
- Bad, because the redaction guarantee is now documented in the command
  reference, not the trust section; a reader looking for "does `doctor` leak
  secrets?" in the NFR section will have to go to the appendix.
