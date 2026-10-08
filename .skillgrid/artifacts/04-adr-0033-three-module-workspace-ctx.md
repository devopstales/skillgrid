# ADR-0033: three-module Go workspace — `ctx/` as a library module

- **Status:** accepted
- **Date:** 2026-10-08
- **Supersedes:** none (supersedes the two-module workspace *framing* established by the `2026-10-06-mnemonic-standalone` change, which was a design decision, not a numbered ADR)
- **Superseded by:** none

## Context

The `2026-10-06-mnemonic-standalone` change split the repo into a two-module
`go.work`: `mnemonic/` (module `github.com/devopstales/skillgrid/mnemonic`,
binary at `mnemonic/cmd/mnemonic/`) and `skillgrid-cli/` (installer). The Context
Orchestrator (CTX) is a thin orchestration layer that sits *above* the Context
Harness (ADR-0025) and the CLM layer (ADR-0027): it composes multi-source
queries into recipes, applies reversible compression, shapes model I/O, hosts a
transparent proxy, and exposes observability.

The plan (`.skillgrid/artifacts/context-orchestrator-plan.md` v1.11) describes a
3rd module `ctx/`. The open question was whether CTX should be a standalone Go
module with its own binary, a package inside `mnemonic/`, or a 3rd module that
imports `mnemonic` and whose CLI surface is a `mnemonic` subcommand group.

## Decision

CTX is a **3rd Go module** at `ctx/` (module `github.com/devopstales/skillgrid/ctx`),
added to `go.work`. It **imports `github.com/devopstales/skillgrid/mnemonic`** for
read-only DB access (it never opens its own handle to the shared store and never
DDLs non-`ctx_` tables). Its **CLI surface is the `mnemonic ctx …` subcommand
group** in the `mnemonic` binary (the binary at `mnemonic/cmd/mnemonic/`), not a
standalone `ctx` binary. The `ctx/` module holds the library code (proxy,
observability, traffic learner, judgment, recipe engine, store/migrations);
`mnemonic/cmd/mnemonic/` wires the `ctx` subcommand to that library.

`ctx/internal/store/migrations/` holds the `ctx_*` migrations, applied by the
`mnemonic` binary to the shared DB (CTX's migrations are additive `ctx_`-prefixed
tables only).

## Consequences

- **Positive:** CTX is a clean, independently-testable library with a single
  dependency direction (ctx → mnemonic), so it cannot reach into mnemonic's
  internals. No second process: the proxy runs as a `mnemonic ctx proxy`
  subcommand (loopback :8787) and the observability endpoints fold into the
  existing `mnemonic serve` UI server. The CLI is one binary the operator
  already knows.
- **Negative:** `go.work` grows to 3 modules; adding CTX's migrations to the
  mnemonic binary means the binary owns applying another module's migrations
  (mitigated: CTX migrations are additive `ctx_`-only, applied under the
  `mnemonic ctx` subcommand path, not the base `mnemonic index` path).
- **Migration:** none — additive. The two-module layout is unchanged; `ctx/`
  is a new sibling. Existing imports are untouched.
