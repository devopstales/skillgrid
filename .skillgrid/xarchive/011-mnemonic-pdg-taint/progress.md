# 011-mnemonic-pdg-taint — progress

## Step 01 — cfg-pdg (DONE, Verdict PASS)

- 15 sub-tasks 01.1–01.15 all `[x]`.
- Implementer: 7 commits (61f36e3, 9627a8c, f91d05d, b243bce, 3c7f1f9, 1bfa34e, 3081a5d).
- Review: task reviewer `approved with fixes` → fix commit 4682339 (enforced LSP round-trip timeout — was a no-op via `ctxWithTimeout`; added dedicated 01.8/01.9/01.10 scenario tests). Scoped re-review of 4682339: clean.
- Residual (non-blocking, deferred to 011 sdd-verify): real external-process JSON-RPC `resolveLSPServer` exercised only via the hermetic `lspResolver` seam; package-level `scanRoot` mutable global (single-threaded `Run` in practice).
- Opt-in byte-for-byte baseline: content-hash `164fd991779f234` stable (non-`--pdg` index unchanged).
- Migration 016_pdg_taint.sql (brief's 014 was taken by 008 step 02). Tool surface 73→74 (`code_pdg_query`).
- Deliverables: per-function CFG (cfg.go), control/data PDG (pdg.go), LSP edge tier w/ LSP_RESOLVED (lsp.go), code_pdg_query tool, `index --pdg/--lsp` + `search pdg` CLI.

## Step 02 — taint-solver (PENDING)

- Blocked by 01 (now done).

## Step 02 — taint-solver (DONE, Verdict PASS)

- 11 sub-tasks 02.1–02.11 all `[x]`.
- Implementer: 9 commits (13695c2, 54a116f, c504233, 8db2b9d, f22e408, 8171708, fbb0351, 5bc8fd2, c38f769).
- Review: task reviewer `approved` (clean). Fix commit fd94b0d (PersistTaint doc-comment — per-hop path is in-memory, not persisted; row carries worst-hop label + stops-at note; + stray tab in pdg.go).
- Note for sdd-verify: `taint_findings` persists path-LEVEL confidence + note, NOT per-hop rows (deliberate schema choice, 016). Per-hop reconstruction on query = possible future enhancement.
- Deliverables: intraprocedural source→sink solver (pdg/taint.go), worst-hop confidence labels (4th value LSP_RESOLVED), LSP feeder-not-dependency (with/without --lsp two-way), no-fabricated-findings, code_taint tool (surface 74→75), search taint CLI.
- Opt-in byte-for-byte baseline `164fd991779f234` stable; 005/008/010 suite ok; CGo-free.

## 011 BOTH STEPS DONE → State.phase = verify. Next: sdd-verify (change-level gate) then sdd-archive.
