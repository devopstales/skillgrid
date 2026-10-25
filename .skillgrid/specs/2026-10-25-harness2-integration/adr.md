---
type: Spec
title: "2026-10-25-harness2-integration — adr"
status: draft

---

# ADR Review Manifest

- **Change:** `2026-10-25-harness2-integration`
- **Status:** in review
- **Review date:** 2026-10-25

## In-Force ADRs Reviewed

- `.skillgrid/decisions/04-adr-0005-trust-boundary-mcp-spawn-model.md` — the MCP-spawned process is the real trust boundary of the Distribution Surface. This change adds a *second* spawned-process boundary (Go → external `harness2` Python CLI). It is **additive and narrower**: the child is read-only against skillgrid state (it writes only under `HARNESS2_RUNS_DIR`), fixed-argv, env allow-listed, and timeout-bounded. It does not change the MCP model.
- `.skillgrid/decisions/04-adr-0008-state-drift-guard-js-yaml.md` — precedent for an external, `uv`/npm-managed CLI that is shell-ed out to rather than imported. Harness² is the same shape: a `uv`-managed package in the install flow, non-fatal, `--dry-run`-aware.
- `.skillgrid/decisions/04-adr-0031-go-version-floor.md` — Go 1.22+ floor holds; no Go language feature change. The change is stdlib-only.
- `.skillgrid/decisions/04-adr-0034-ctx-cli-mnemonic-subcommand.md` — `ctx` is a `mnemonic …` subcommand group, not a standalone binary. `harness` follows the same pattern: a `mnemonic harness …` subcommand group under `cmd/mnemonic/`.
- `.skillgrid/decisions/04-adr-0004-four-component-decomposition.md` — component map is a 4-way decomposition. This change touches the **Distribution Surface** (CLI group + REST + SPA) and the **Installer** (tool list). The Mnemonic Engine is untouched — no new data ownership, no schema change. Harness² is an external process the Surface hosts; skillgrid owns no Harness² state except the `runs/` dir it points at.
- Locked constraint "No new dependencies without an ADR" — satisfied by the new ADR below: **no new Go module dependency** (stdlib `os/exec` + `encoding/json` only). Harness² is a *machine-level* Python runtime dependency installed by the installer, not a `go.mod` entry.

## New Durable ADRs Created

- `.skillgrid/decisions/04-adr-0044-harness2-host-integration.md` — Harness² (Python, `uv`-managed) is hosted as an external subprocess; no Go module dependency; `runs/` under the skillgrid data dir via `HARNESS2_RUNS_DIR`; `mnemonic harness …` subcommand group; read-heavy REST routes + SPA view.

## Supersessions

- None. ADR-0005 (trust boundary) is extended, not changed; ADR-0034 (CLI subcommand pattern) is reused. ADR-0044 adds the Harness² host layer.
