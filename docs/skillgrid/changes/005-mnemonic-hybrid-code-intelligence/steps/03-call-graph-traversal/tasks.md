# Tasks: 005-mnemonic-hybrid-code-intelligence — Step 03-call-graph-traversal

> Goal: Edge resolve + Confidence Labels + Tier-2 graph tools (callers/callees/dependents/…).
> Depends on: 02-identifier-fts-orientation
> Ticket: TASK-005

> Change-level forecast: see `../01-schema-extractors/tasks.md`
> Decision needed before apply: No · Chained PRs recommended: Yes · Chain strategy: stacked-to-main · 400-line budget risk: High

## Execution

- [ ] 03.1 Create `skillgrid-cli/internal/mnemonic/graph/resolve.go` — resolve Symbol → walk Edges with Confidence Label `EXTRACTED|INFERRED|AMBIGUOUS` on every edge.
- [ ] 03.2 Create `skillgrid-cli/internal/mnemonic/mcp/tools_code_graph.go` — `code_get_callers`, `code_get_callees`, `code_get_dependents`, `code_get_implementors`, `code_get_tests_for`, `code_get_type_hierarchy` (JSON-only); register beside orient tools.
- [ ] 03.3 Extend `skillgrid-cli/internal/mnemonic/service/service.go` — graph facade for callers/callees/dependents/implementors/hierarchy/tests-for.
- [ ] 03.4 Extend `skillgrid-cli/cmd/skillgrid/code_intel.go` — CLI parity for Tier-2 graph commands.
- [ ] 03.5 Cover WHAT: for a known symbol, callers, callees, transitive dependents, implementors, hierarchy, and tests-for return edges each carrying a Confidence Label.
- [ ] 03.6 Cover WHAT edge: ambiguous resolution → `AMBIGUOUS` label, not silent drop.
