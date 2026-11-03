# 013-mnemonic-layered-memory-governance — progress

## Step 01 — governance-fields (DONE, Verdict PASS)
- 10 sub-tasks [x]. Commits 28467ec (017 schema), 7caa2b7 (governance core), dfbea4e (mem_share/mem_governance, 75→77), c93b87d + 8563453 (wire per-owner read enforcement into mem_search/mem_get_observation + empty-owner fix).
- Review `approved with fixes` → fix wired the unattached read-enforcement seam (same pattern as 011/008).

## Step 02 — layered-distill (DONE, Verdict PASS)
- 10 sub-tasks [x]. Commits 547c5d0 (018 schema), 66f9b6c (mem_layers + 77→78), 5ede761, 2829fce, 7dc3441, a4c6202, c86414c (wire session-close distill hook into real close path via detached goroutine).
- Review `approved with fixes` → fix wired the unattached session-close distill hook.

## Step 03 — layered-retrieval-budgets (DONE, Verdict PASS)
- 9 sub-tasks [x]. Commits 6fce689 (budget), 91fd4a4 (layered retrieve), 9142064 (route reads), 7508faa (in-list + only-full-content), 7e29c3d (CLI parity), c25e122 (enforce timeout via Bound+ApplyRead), 0cfa827 (owner-gated layered + production entry point), c19789a (uniform CLI budget).
- Review `approved with fixes` → fix: enforced the no-op timeout, gave BudgetedRetrieval a real production caller (CLI mem search, owner-gated), wired CLI context/timeline budget flags, uniform budget.

## 013 ALL 3 STEPS DONE → State.phase = verify. Next: sdd-verify (change-level gate) then sdd-archive.
- Migrations: 017 (governance) + 018 (layer distill). Tool surface 75→78 (mem_share, mem_governance, mem_layers).
- Recurring pattern: each step's reviewer caught a "tested seam not wired to the handler" (01 read-enforcement, 02 session-close hook, 03 timeout + production caller) — all fixed in-place.
