# Broken Windows

> Cross-change defect register. Advisory findings (WARNING/SUGGESTION) not
> fixed before archive. Auto-appended by `skillgrid:qa` at gate-render time.
> Waived items are tracked here until fixed or explicitly closed.
>
> **Lifecycle:** `qa` appends open findings → `ship` archives the change
> (windows stay) → future changes' `qa` surfaces open windows as context.

| Window | Change | Severity | Finding | Waived By | Date | Status |
|--------|--------|----------|---------|-----------|------|--------|
| W001 | 2026-09-24-mnemonic-memory-improvements | WARNING | handleMemSearch drops an EmbedQuery error with no test that enters that branch | qa-auto | 2026-10-02 | closed |
| W002 | 2026-09-24-mnemonic-memory-improvements | WARNING | six tickets have no separable RED commit (strict-TDD off) | qa-auto | 2026-10-02 | open |
| W003 | 2026-09-24-mnemonic-memory-improvements | WARNING | qa-gate.mjs state-drift and size-budget scripts not found | qa-auto | 2026-10-02 | closed |
| W004 | 2026-09-24-mnemonic-memory-improvements | SUGGESTION | Service decayCfg comment says zero config enables decay; blend reranks only when Enabled is true | qa-auto | 2026-10-02 | open |
| W005 | 2026-09-24-mnemonic-memory-improvements | SUGGESTION | go vet context leak in budget.go:114 is pre-existing | qa-auto | 2026-10-02 | open |
| W006 | 2026-09-24-mnemonic-memory-improvements | SUGGESTION | private-row hiding has a single test | qa-auto | 2026-10-02 | open |
| W007 | 2026-09-24-mnemonic-memory-improvements | SUGGESTION | qa-gate.mjs looks in .agents/skills/qa/scripts; helpers live under verification/qa/scripts | qa-auto | 2026-10-02 | open |
| W008 | 2026-10-02-mnemonic-webui-rewrite | SUGGESTION | GraphPage has no mounted test; G4 is a source-text grep | qa-auto | 2026-10-02 | closed |
| W009 | 2026-10-02-mnemonic-webui-rewrite | SUGGESTION | lib/api.ts and lib/apiBase.ts have no direct unit test | qa-auto | 2026-10-02 | closed |
| W010 | 2026-10-02-mnemonic-webui-rewrite | SUGGESTION | dompurify 3.4.15 → 3.4.16 (GHSA-p98j-92pf-mc4p, LOW, not reachable) | qa-auto | 2026-10-02 | open |
| W011 | 2026-10-02-mnemonic-webui-rewrite | SUGGESTION | L3 browser walk of /mnemonic/graph against skillgrid serve was not run | qa-auto | 2026-10-02 | open |
