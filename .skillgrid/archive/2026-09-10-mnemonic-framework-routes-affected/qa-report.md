# QA plan — 010-mnemonic-framework-routes-affected

Change scope: framework-route extraction + drop-not-guess edge policy (step 01), `code_affected` (--base) + `code_rename` (step 02), autosync watcher + fingerprint gate + warm embedder (step 03). Additive on 005/008; CGo-free.

## How to exercise (via `skillgrid` CLI from a fixture web app with code + routes + docs + configs + SQL)

### Happy path
1. `skillgrid index` → completes (exit 0); community/process/route/knowledge passes run (advisory).
2. `skillgrid code route --handler <sym>` / `code navigates` → route→handler + navigates edges; ambiguous ref dropped + warned (`route_drops`).
3. `git diff --name-only | skillgrid search affected --stdin` → affected test files (transitive import + tests-for).
4. On a fixture branch: `skillgrid search affected --base origin/main` → changed set from merge-base diff, grouped into **areas** + **git-blame owners** ("who to tag"); `--json` = ready PR comment.
5. `skillgrid search rename <old> <new>` (dry-run default) → graph edits (high-conf) vs text-search edits (low-conf, "review carefully"); `--apply` edits only listed files, no commit/push.
6. `skillgrid serve` + edit a file mid-session → auto-reopen within ~5s (no restart); a pending referenced file gets `⚠️ <file> is pending sync — Read it directly`.
7. `SKILLGRID_NO_WATCH=1` + edit + a `code_*` query → the fingerprint gate re-indexes structurally (embedder untouched) so the query reflects the edit.
8. `skillgrid index status` → `### Pending sync:` section.

### Edge / failure
- **Framework with no routes**: `code_route` reports none, no fabricated nodes.
- **Unresolvable route ref**: dropped + warned (not stored). **Unresolvable config ref**: AMBIGUOUS (not dropped).
- **Two `serve` processes on one project**: the second exits with a clear writer-lock error pointing at the live writer.
- **Re-index failure during watch**: old index stays live (copy-and-swap keeps it; in-place mid-write stops with the may-have-mutated guard).
- **Watcher disabled**: `SKILLGRID_NO_WATCH=1` → manual index only (fingerprint gate still the backstop).
- **Absent/failed embedder model**: degrades to FTS + signals (no crash); warm cache holds no RAM for `off`/external.
- **005/008 regression check**: `code_search`/`code_read`/`code_path` (code-to-code)/`code_communities`/`code_processes`/`code_affected` names + required params unchanged; tool surface 73 (additive only).

### Pass / fail criteria
- **Pass**: all happy-path commands succeed with coherent, non-fabricated data; all edge cases behave as specified; `go build ./...` clean; blast-radius traverses only resolved edges (drop policy); 73-tool baseline holds.
- **Fail**: any fabricated route/edge (drop-not-guess violated), a 005/008 tool name/param changed, a CGo dependency, a torn index observed by a reader, a false positive inflating `code_affected`, or a crash on an absent model / second writer.

### Waiver
Agent gate is `PASS` (all 3 steps PASS, 29 scenarios COMPLIANT at runtime, `go build ./...` clean, 005/008/010 baselines preserved, CGo-free, additive). No open tasks, no Ticket. A human may accept the QA plan (happy/edge/failure above) or waive — the per-step Verdicts + sub-agent reviews + runtime proof are the agent-gate evidence.

## Environments
Local, branch `feat/v2-structure`, Go 1.26 toolchain (loom requires it). Fixture: a Go/Python web app (routes), a `.md` with a `[[wikilink]]`, a `.yaml` config referencing a symbol, a `.sql` DDL file, and a git repo with a base ref for `--base`.
