# Tasks: 003-mnemonic-self-evolving-context-database — Step 02-tiered-storage

> Goal: L0/L1/L2 FS, tiered module, `migrate --tier`, background hooks.
> Depends on: 01-schema-extensions

> Change-level Review Workload Forecast: see `../01-schema-extensions/tasks.md`
> Decision needed before apply: Yes · Chained PRs recommended: Yes · Chain strategy: pending · 400-line budget risk: High

## Execution

- [ ] 02.1 Create `skillgrid-cli/internal/mnemonic/tiered/summarizer.go` with `Summarizer` interface + stub/heuristic adapters (`Abstract`, `Overview`).
- [ ] 02.2 Create `skillgrid-cli/internal/mnemonic/tiered/tiered.go` to generate/read L0 (`.abstract`) / L1 (`.overview`) / L2 and register paths in `tiered_contents`.
- [ ] 02.3 Create `skillgrid-cli/internal/mnemonic/tiered/hook.go` implementing non-blocking `ContentWriteHook.AfterContentWrite` (does not await summarization).
- [ ] 02.4 Create `skillgrid-cli/cmd/skillgrid/migrate.go` with `runMigrate` / `--tier` backfill of L0/L1 from existing L2.
- [ ] 02.5 Cover WHAT: content-write seam yields L0/L1 sidecars + path columns without blocking the L2 write.
- [ ] 02.6 Cover WHAT: `skillgrid migrate --tier` backfills L0/L1; L2 bytes unchanged.
- [ ] 02.7 Cover WHAT edge: summarizer failure leaves L2 intact; error logged.
