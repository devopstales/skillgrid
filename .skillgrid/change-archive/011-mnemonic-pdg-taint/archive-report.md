# Archive report — 011-mnemonic-pdg-taint

**Archived**: docs/skillgrid/archive/011-mnemonic-pdg-taint/
**Commit**: (archive commit, staged below)
**Move**: git mv changes/ -> archive/; source gone; diff -r EMPTY ✅
**Ticket**: none (change.md has no Ticket:)

## Gates
- verify: PASS (change-level); 01-cfg-pdg PASS, 02-taint-solver PASS; 21/21 scenarios COMPLIANT at runtime.
- tasks: no open `- [ ]` under any `### Tasks` (01.x 55 + 02.x 46 all [x]; remaining unchecked = DoD/archive template lines).
- human QA: WAIVED (user, this turn).
- DoD / archive checklist: satisfied.
- Workspace: allowed edit roots (skillgrid-cli/, docs/skillgrid/).

## Final state
- Step 01 (cfg-pdg): opt-in --pdg per-function CFG (gotreesitter AST, no new grammar) + control/data PDG (confidence-labeled, truncateCFG cap, //go:nocfg malformed-skip); independent opt-in --lsp edge tier (external-process JSON-RPC, LSP_RESOLVED 4th confidence, hermetic lspResolver seam, enforced round-trip timeout); code_pdg_query tool. Commits 61f36e3..3081a5d + fix 4682339.
- Step 02 (taint-solver): intraprocedural source->sink solver over PDG data-deps; deterministic configurable source/sink sets; worst-hop path-level confidence (EXTRACTED only if all hops resolved); LSP RESOLVED feeder-not-dependency (with/without --lsp two-way); no fabricated findings; code_taint tool + search taint CLI. Commits 13695c2..c38f769 + fix fd94b0d.
- Additive: migration 016_pdg_taint.sql (cfg_blocks/cfg_edges/pdg_edges/taint_findings; brief's 014 was taken by 008 step 02); tool surface 73 -> 74 -> 75 (code_pdg_query, code_taint); CGo-free; go build clean.
- Opt-in byte-for-byte baseline: content-hash 164fd991779f234 stable (non-flag index unchanged); 005/008/010 suite (route/affected/community) ok.

## Non-blocking follow-ups (not gating archive)
- Real external-process JSON-RPC path pdg/lsp.go:resolveLSPServer exercised only via the hermetic lspResolver seam (edge-write + absent/failing/timeout paths tested; live handshake not).
- taint_findings persists path-LEVEL confidence + stops-at note, not per-hop rows (deliberate schema choice); per-hop reconstruction on query is a future enhancement.
- Package-level scanRoot mutable global in pdg_pass.go (single-threaded Run in practice).

## 011 SDD cycle COMPLETE.
