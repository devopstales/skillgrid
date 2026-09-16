# QA plan — 011-mnemonic-pdg-taint

Change-level verify verdict: **PASS** (21/21 scenarios COMPLIANT at runtime; 005/008/010 baseline intact; CGo-free). Per-step verdicts: 01 PASS, 02 PASS. No Ticket.

This is a Go library/CLI feature (opt-in `--pdg`/`--lsp` index passes + `code_pdg_query`/`code_taint` MCP tools + `search pdg`/`search taint` CLI). QA is manual smoke on a real repo + the MCP/CLI surface. The automated suite already covers the invariants (byte-for-byte opt-in isolation, LSP feeder-not-dependency, no-fabricated-findings, determinism, CGo-free) — this plan is for a human to confirm the *experience* on a real codebase.

## What to exercise

### Happy path
1. `skillgrid index --pdg <some-go-repo>` then `skillgrid search pdg <symbol> <statement>` → returns control/data dependents for a statement in a real function.
2. `skillgrid index --pdg <repo>` then `skillgrid search taint` → for a repo with an untrusted-input→SQL/shell flow, a finding with source kind, sink kind, and path; `--json` for CI.
3. `skillgrid index --lsp <repo>` (with gopls on PATH) → member calls the static pass couldn't type get `LSP_RESOLVED` edges; `skillgrid index --lsp --pdg` → taint path continues through the resolved boundary.

### Edge
4. `skillgrid index <repo>` (NO `--pdg`) → `search pdg` and `search taint` both return a clear "run index --pdg" hint (not an error), and the common code-graph output is unchanged.
5. `skillgrid index --lsp <repo>` with no language server on PATH → warning, index unchanged (static), no hard error.
6. A large/deep function → its CFG is truncated with a note; the index still completes.

### Failure
7. A repo with a function that has no source→sink flow → `search taint` reports no finding for it (never fabricated).

## Environment / data
- macOS; a small Go repo you own (or the skillgrid repo itself) with at least one function containing a branch/loop and, ideally, an input→exec flow. gopls installed for the `--lsp` arms (or accept the no-server warning for #5).

## Pass / fail criteria
- Pass: #1–#7 behave as described; no panic; non-`--pdg` output identical to a pre-change run; CGo-free (`CGO_ENABLED=0` build works).
- Fail: a non-`--pdg` index changes any 005/008/010 output; a taint finding with an empty/fabricated hop list; an `--lsp` failure aborts the index; a PDG/taint query on a non-`--pdg` index errors instead of hinting.

## Waive
The automated suite (per-step Verdicts + change-level runtime proof) already proves every @p0/@p1 invariant at runtime, and this step is opt-in additive with a byte-for-byte baseline gate. Human QA may be **waived** if the above smoke is covered by the test evidence and the user accepts the residual (the live JSON-RPC LSP handshake is tested only via the hermetic seam; `taint_findings` persists the path-level label, not per-hop rows).
