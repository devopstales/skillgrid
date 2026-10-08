# ADR-0036: CTX config lives at `config.d/ctx.yaml` (machine-local + repo-local override)

- **Status:** accepted
- **Date:** 2026-10-08
- **Supersedes:** none
- **Superseded by:** none

## Context

CTX needs a config surface (proxy bind/port/mode, wrap agents, observability
toggles, traffic-learning rules, judgment model/endpoint). The repo already has
a `config.d` loader (`mnemonic/internal/config/pricing.go:96`, `findConfigD`)
used by `indexing.yaml`, `mcp.yaml`, and `tools.yaml`: repo-local
`.skillgrid/config.d/<name>.yaml` overrides machine-local
`~/.skillgrid/config.d/<name>.yaml`, first-found-wins per key. The open question
was whether CTX config belongs in the repo's `.skillgrid/config.yaml` or in the
machine-local `config.d/` pattern.

## Decision

CTX config is **`ctx.yaml` in the `config.d` pattern**: machine-local
`~/.skillgrid/config.d/ctx.yaml` (the default, per-machine — e.g. the local
Ollama endpoint, proxy port, wrap agents detected on the box) with an optional
repo-local `.skillgrid/config.d/ctx.yaml` override (project-specific, e.g.
which traffic rules or proxy mode a given project uses). Precedence matches
`indexing.yaml`: repo-local wins per key over machine-local. The existing
`config.d` loader is reused — no new config mechanism. The `clm:` block
(ADR-0027, already in `.skillgrid/config.yaml`) is **not** moved; CLM config
stays where ADR-0027 put it. Only the *new* CTX v1.11 blocks (`proxy`, `wrap`,
`observability`, `traffic_learning`, `judgment`) live in `ctx.yaml`.

## Consequences

- **Positive:** Per-machine defaults (Ollama endpoint, proxy port, wrap agents)
  live in one place the operator edits once; per-project overrides are
  optional and committed. Reuses the proven `config.d` loader — no new
  mechanism to test.
- **Negative:** CTX config is split across two files (machine + repo), so a
  reader must know the `config.d` precedence rule to find a value.
- **Migration:** none — additive `ctx.yaml` in both locations; absent file =
  all defaults.
