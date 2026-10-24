# Archive report — 013-mnemonic-layered-memory-governance

**Archived**: docs/skillgrid/archive/013-mnemonic-layered-memory-governance/
**Commit**: (archive commit, staged below)
**Move**: git mv changes/ -> archive/; source gone; diff -r EMPTY ✅
**Ticket**: none (change.md has no Ticket:)

## Gates
- verify: PASS (change-level); 01/02/03 all PASS; 34/34 scenarios COMPLIANT at runtime; -race clean.
- tasks: no open `- [ ]` under any `### Tasks` (01.x 20 + 02.x 20 + 03.x 19 all [x]; remaining unchecked = DoD/archive template lines).
- human QA: WAIVED (user, this turn).
- DoD / archive checklist: satisfied.
- Workspace: allowed edit roots (skillgrid-cli/, docs/skillgrid/).

## Final state
- Step 01 (governance-fields): owner, append-version (observation_versions, latest=read path, revision_count), status (explicit never inferred), retrieval_usage (distinct from duplicate_count), private-by-default visibility + restricted ACL; mem_share (only widen) + mem_governance tools; per-owner read enforcement WIREd into mem_search/mem_get_observation. Commits 28467ec, 7caa2b7, dfbea4e, c93b87d, 8563453.
- Step 02 (layered-distill): session-close L0→L1 atoms→L2 scenario→L3 persona, provenance-linked (no orphan / no-op), LLM-cached by content-hash + no-LLM floor (reuses 005 CapturePassive), L1 atoms correctable via version-append; opt-in + async (detached goroutine in SessionEnd) + best-effort; mem_layers tool. Commits 547c5d0, 66f9b6c, 5ede761, 2829fce, 7dc3441, a4c6202, c86414c.
- Step 03 (retrieval-budgets): layered retrieval (L2/L3-first, L1/L0 RRF fallback via existing BlendedSearch, real owner-gated production caller via CLI mem search) + uniform budget (item+char+enforced timeout via Bound+ApplyRead, never hangs) on mem_search/mem_context/mem_timeline; mem_get_observation ONLY full-content path; budget tunable via config; CLI parity. Commits 6fce689, 91fd4a4, 9142064, 7508faa, 7e29c3d, c25e122, 0cfa827, c19789a.
- Additive: migrations 017_layered_memory_governance.sql + 018_layered_distill.sql; tool surface 75→78 (mem_share, mem_governance, mem_layers); 005 mem_* names+required params unchanged (name+required-param lock); pure Go (no new CGo).

## Non-blocking follow-ups (not gating archive)
- In-list "N chars omitted" marker is asserted but the exact omitted count is not pinned in every path.
- mem_context/mem_timeline item-cap uses their own `limit` param (char+timeout uniform) — documented divergence, not a constraint violation.

## 013 SDD cycle COMPLETE.
