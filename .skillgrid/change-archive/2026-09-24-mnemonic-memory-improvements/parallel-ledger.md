# Parallel ledger — memory-improvements wave

Wave A (Needs satisfied, Owns disjoint). TICKET-05 waits for TICKET-04.

| agent | task | status | Owns | changed files | test result |
|---|---|---|---|---|---|
| decay-config | TASK-030.05 TICKET-03 | dispatched | config/load.go, config/load_test.go, service/service.go, memory/search_blend_test.go | | |
| query-cache | TASK-030.02 TICKET-04 | dispatched | store/migrations/045_search_aids.sql, store/migrations_045_test.go, memory/query_cache.go, memory/query_cache_test.go, mcp/tools_memory.go | | |
| compact-hook | TASK-030.03 TICKET-06 | dispatched | memory/skills.go, memory/compact_hook_test.go, cmd/skillgrid/mem.go | | |

Ruling: agents do not commit. Parent commits each Owns set after re-verify so three writers do not race the git index. Serial-development stays one branch.
