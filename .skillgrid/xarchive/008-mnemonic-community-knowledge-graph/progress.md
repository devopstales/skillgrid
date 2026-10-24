# SDD ledger — plan: docs/skillgrid/changes/008-mnemonic-community-knowledge-graph/tasks.md

## ⚠️ Plan changed mid-apply (commit 0100db4)
A parallel session rewrote tasks.md (008–013). Step 01 grew 9→21 sub-tasks (added retrieval-eval + ranker layer). All now done.

## ALL 3 STEPS APPLIED (per-step Verdict PASS) — State: verify

- Task 01 (community-detection): COMPLETE. 01.5-01.8 + 01.13-01.16 (community) + 01.1-01.4, 01.9-01.12, 01.17-01.21 (retrieval-eval + ranker). Commits 899c3a2..edd8697. Reviews: content-hash cache fix (c042f74), shippedRank wired (06ce4ee+edd8697). Verdict PASS.
- Task 02 (process-flows): COMPLETE. 02.1-02.9. Commits ed140a3, 25e56de, ffbfa77. Review approved-with-fixes: query-time pass tracked as step-03 deferral (task 03.8). Verdict PASS.
- Task 03 (knowledge-graph-nodes): COMPLETE. 03.1-03.9. Commits cb117c3, f5f10f2, cfdf6f8, 4bc0391, aa98d1c, 0d01b76, b8a0d02. Review approved-with-fixes: 03.8 "same tx" comment inaccurate (post-commit/advisory), SaveConfig lowest-id on collision, isSQL non-.sql scan, dead HasPrefix — all fixed (b8a0d02). Verdict PASS.

## Key cross-step facts
- 010 step 01 (framework routes) was applied as the prerequisite that unblocked 008 step 02 (entry points). 010 steps 02 (code_affected) + 03 (autosync-watcher) still unapplied.
- 03.8 IndexerHook: 005+route in one tx; community/process/knowledge passes run post-commit on a reopened DB (modernc single-conn deadlock), advisory.
- Tool surface: 63 (005 baseline) → 65 (+code_route/code_navigates) → 67 (+code_processes/code_process) → 71 (+4 knowledge tools). Additive throughout.
- Migrations: 012 (community), 013 (framework routes), 014 (process flows), 015 (knowledge graph) — all additive.
- go.mod: go 1.26 (loom-forced), loom + onnxer in direct block.

## Next
- sdd-verify (independent quality gate over all 3 steps) → sdd-archive.
- Separately: 010 steps 02 + 03.
