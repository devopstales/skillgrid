---
id: TASK-104
title: TICKET-01 Store extension + 9-table migration + Index driver + door check
status: done
assignee: []
created_date: '2026-11-02 15:51'
updated_date: '2026-11-03 18:24'
labels: []
milestone: 2026-10-31-mnemonic-v2-code-index
dependencies: []
priority: high
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Extend the Store interface (ADR-0046) with code-index write/read + vector methods + trail seam + ExecDDL + Begin/Txn; types_code.go; migrations/100_code_index.sql (7 content + 2 FTS tables); SQLite render; minimal codeindex.Index driver; the door check. See .skillgrid/change-specs/2026-10-31-mnemonic-v2-code-index/tasks.md TICKET-01.
<!-- SECTION:DESCRIPTION:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
TICKET-01 door check — Store extension (9-table migration + vector + trail seam + ExecDDL/Txn), types_code.go, migrations/100_code_index.sql, SQLite render, codeindex.Index driver. Green.
<!-- SECTION:FINAL_SUMMARY:END -->
