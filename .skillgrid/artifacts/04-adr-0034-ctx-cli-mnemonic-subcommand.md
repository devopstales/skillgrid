# ADR-0034: `ctx` is a `mnemonic` subcommand group, not a standalone binary

---
status: "accepted"
supersedes: none
date: 2026-10-08
---

## Context

The `mnemonic` binary is a hand-rolled `flag`-based dispatcher
(`mnemonic/cmd/mnemonic/main.go`, `runServe` in `mcp.go`); it already ships
`index`, `prime`, `compact`, `pages`, `serve`, and a **`stats`** command
(per-agent activity rollup). The CTX plan adds a large `ctx …` CLI surface
(`proxy`, `wrap`, `stats`, `search`, `purge`, `metrics`, `traffic`, `learn`).
The open question was where that surface lives: a standalone `ctx` binary, or a
subcommand group on the `mnemonic` binary.

## Decision

The CTX CLI surface is the **`mnemonic ctx …` subcommand group** on the
`mnemonic` binary. One new `case "ctx"` in the dispatcher routes to a
`runCtx` package that binds the verbs to the `ctx/` module library. Verbs:
`mnemonic ctx proxy [status|mode <m>]`, `mnemonic ctx wrap <agent>`,
`mnemonic ctx stats [--history]`, `mnemonic ctx search <query>`,
`mnemonic ctx index <path>`, `mnemonic ctx purge`, `mnemonic ctx metrics`,
`mnemonic ctx traffic memories|report|rules reload`, `mnemonic ctx learn
[report|apply <id>|revoke <id>]`, `mnemonic ctx judge`, `mnemonic ctx
checkpoint list|read <id>`.

**The existing `mnemonic stats` (per-agent activity rollup) is unchanged and
distinct.** The unified "ctx stats" (Q4) unifies *within the CTX surface* —
proxy/session metrics plus the Context Language Model row counts
(`tool_outputs`, `indexed_files`) as a sub-slice — under `mnemonic ctx
stats`. The two `stats` live under different parents and are documented as
distinct; there is no collision and zero churn to the shipped command.

## Consequences

- **Positive:** One binary the operator already runs; no second artifact to
  install or version. The `ctx` group is a natural extension of the
  `mnemonic` surface that already owns the DB, the session events, and the UI
  server the observability endpoints fold into.
- **Negative:** The `mnemonic` dispatcher grows another verb tree; an operator
  must know `ctx` lives under `mnemonic`, not as its own command.
- **Migration:** none — additive `case "ctx"` in the dispatcher.
