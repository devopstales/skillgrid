# QA plan — 008-mnemonic-community-knowledge-graph

Change scope: community detection + retrieval-eval + ranker layer (step 01), process flows (step 02), doc/config/SQL knowledge nodes + indexer hook + `code_path` span (step 03). Additive on 005; CGo-free; advisory (never load-bearing).

## How to exercise (all via `skillgrid` CLI from a fixture repo with code + routes + docs + configs + SQL)

### Happy path
1. `skillgrid index` on the fixture → confirm it completes (exit 0) and the community/process/knowledge passes run (advisory; a warn-and-continue line is acceptable).
2. `skillgrid code communities` (MCP: `code_communities`) → labeled subsystems; `code_god_nodes --exclude-hubs` → hubs suppressed; `code_explain_community <id>` → members + entry points.
3. `skillgrid code processes` (MCP: `code_processes`) → precomputed flows with steps + cross-community flag + labels. NOTE: flows are populated at INDEX time (03.8 hook); if empty, the index must have run the process pass with the LLM stub (labels may be the deterministic stub text).
4. `skillgrid code route --handler <sym>` / `code navigates` → route→handler + navigates edges; an ambiguous ref is dropped + warned (`route_drops`).
5. `code_path --from <code-sym> --to <table-or-doc>` → a path spanning code→doc→config→table (traverses ≥2 knowledge edge kinds).
6. `skillgrid doctor --strict` → reports redaction + freshness; exits non-zero on a violation.
7. `skillgrid eval --corpus self` → baseline vs shipped (factor-based) recall with CI + p (shipped should show a positive, non-vacuous delta).

### Edge / failure
- **Empty / tiny repo** (< 2 symbols): `code_communities` yields one trivial community, no crash.
- **Framework with no routes**: `code_route` reports none, no fabricated nodes.
- **Malformed doc/SQL**: skip the bad part, index the rest (index still succeeds).
- **Unresolvable config ref**: stored `AMBIGUOUS` (not dropped). **Ambiguous route ref**: dropped + warned (not stored).
- **LLM down** (process labels): flow cached UNLABELED, never fabricated.
- **005 regression check**: `code_search`/`code_read`/`code_orient`/`code_path` (code-to-code) still work with names/required params unchanged; `code_search` response only GAINS confidence/action/rerank_reasons/redacted fields.

### Pass / fail criteria
- **Pass**: all happy-path commands succeed and return coherent, non-fabricated data; all edge cases behave as specified; `go build ./...` is clean EXCEPT the known parallel-session `setup` breakage; the 005 tool-surface baseline holds.
- **Fail**: any fabricated node/edge (drop-not-guess violated), a 005 tool name/param changed, a CGo dependency introduced, a non-additive migration, or the index crashes on a malformed file.

### Waiver
Agent gate is PASS WITH WARNINGS: the only non-green runtime is the parallel-session `internal/mnemonic/setup` refactor (helpers moved to new untracked `internal/install/`; `opencode.go` mid-edit) — not 008's, not a 008 global-constraint violation. All 008 scenario tests pass at runtime and all 008 packages build. A human may **waive** the `setup`-cascaded `mcp`/`cmd/skillgrid` build failures for 008's purposes, or re-run those two e2e tests once the parallel session lands `setup`/`install`.

## Environments
Local, current branch `feat/v2-structure`, Go 1.26 toolchain (loom requires it). Fixture repo with at least: a Go/Python web framework (routes), a `.md` doc with a `[[wikilink]]`, a `.yaml` config referencing a symbol, and a `.sql` DDL file.
