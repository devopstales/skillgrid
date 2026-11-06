# Final Review Fix — Handoff WIP Tracking (merge-blocking)

**Branch:** release/2
**Finding:** `skillgrid-ui/src/app.tsx:95` and `skillgrid-ui/src/components/layout/AppLayout.tsx:17` (committed in 76ec335, part of this plan's size:exception PR) import `./features/handoff/HandoffHubPage`, but the handoff feature files were UNTRACKED. A clean checkout of HEAD failed `npm run build:check` (tsc could not resolve the module).
**Fix accepted:** commit the handoff WIP files so the PR is self-contained.
**Commit:** `73e6f63 feat(handoff): handoff hub feature (WIP companion to decision bridge)` — conventional, no AI attribution. Commit-msg hook / zone guard: no complaints (clean exit 0).

## Files committed (30)

### New untracked handoff feature (17)
- skillgrid-ui/src/features/handoff/HandoffHubPage.tsx
- skillgrid-ui/src/features/handoff/api.ts
- skillgrid-cli/cmd/skillgrid/handoff.go
- skillgrid-cli/cmd/skillgrid/handoff_test.go
- skillgrid-cli/internal/mnemonic/handoff/checkpoint.go
- skillgrid-cli/internal/mnemonic/handoff/context.go
- skillgrid-cli/internal/mnemonic/handoff/handoff.go
- skillgrid-cli/internal/mnemonic/handoff/handoff_test.go
- skillgrid-cli/internal/mnemonic/handoff/refs.go
- skillgrid-cli/internal/mnemonic/handoff/rollup.go
- skillgrid-cli/internal/mnemonic/handoff/snapshot.go
- skillgrid-cli/internal/mnemonic/handoff/testhelpers_test.go
- skillgrid-cli/internal/mnemonic/http/handoff.go
- skillgrid-cli/internal/mnemonic/http/handoff_test.go
- skillgrid-cli/internal/mnemonic/mcp/tools_handoff.go
- skillgrid-cli/internal/mnemonic/store/handoff_hub_schema_test.go
- skillgrid-cli/internal/mnemonic/store/migrations/039_handoff_hub.sql

### Modified, handoff-related per `git diff` (13)
- .agents/hooks/checkpoint-state.sh — adds best-effort `skillgrid handoff record` mirror in the snapshot/commit path (change 015-handoff-hub)
- docs/user-guide/09-serve-dashboard.md — new "Handoff" section documenting the `/handoff` Handoff Hub (change 015) + adds "Handoff" to the unaffected-views list. Explicit exception: diff is handoff-related → staged.
- skillgrid-cli/cmd/skillgrid/main.go — wires the `handoff` command (`runHandoff`)
- skillgrid-cli/internal/mnemonic/http/activity.go — SSE stream extended to emit `event: snapshot` from `change_snapshots` (Handoff Hub)
- skillgrid-cli/internal/mnemonic/http/activity_test.go — snapshot-event assertion in the SSE test (change 015)
- skillgrid-cli/internal/mnemonic/relay/cleave.go — Bundle gains FromCommit/ToCommit/SpecDir (handoff commit-range join)
- skillgrid-cli/internal/mnemonic/relay/relay.go — writes `handoff_refs` join; adds HandoffRef / RecordTeamRef / GetHandoffRef (change 015)
- skillgrid-cli/internal/mnemonic/relay/relay_test.go — TestHandoffRecordsRefJoin (change 015)
- skillgrid-cli/internal/mnemonic/mcp/server.go — registers `registerHandoffTools`
- skillgrid-cli/internal/mnemonic/mcp/server_test.go — expects handoff_snapshot/status/checkpoint/verify/rollup tools (015)
- skillgrid-cli/internal/mnemonic/mcp/tools_session_handoff.go — from_commit/to_commit/spec_dir params + resume surfaces the handoff_refs join
- skillgrid-cli/internal/mnemonic/mcp/tools_session_handoff_test.go — tool count raised to >= 87 (+5 handoff)
- skillgrid-cli/internal/mnemonic/mcp/tools_teams.go — mark_done records a `team` handoff_refs row via relay.RecordTeamRef (change 015)

## Files inspected-and-left-unstaged (10)

- skillgrid-cli/internal/mnemonic/mcp/tools_code_affected_test.go — tool-count bump `!= 83` → `< 87`; no handoff/handoff_refs/change_snapshots identifiers, not in the explicit list
- skillgrid-cli/internal/mnemonic/mcp/tools_code_knowledge_test.go — same: count-bump only, no handoff identifiers
- skillgrid-cli/internal/mnemonic/mcp/tools_memory_governance_test.go — same: count-bump only, no handoff identifiers
- skillgrid-cli/internal/mnemonic/mcp/tools_memory_layer_args_test.go — same: count-bump only, no handoff identifiers
- skillgrid-cli/internal/mnemonic/mcp/tools_memory_layer_test.go — same: count-bump only, no handoff identifiers
- skillgrid-cli/internal/mnemonic/mcp/tools_memory_retrieval_test.go — same: count-bump only, no handoff identifiers
- skills-lock.json — adds the `create-design-md` skill entry (source ibelick/ui-skills); not a handoff lock change
- docs/user-guide/04-hooks.md — documents checkpoint-state.sh; the single "handoff record" mention is a docs-only cross-reference, not handoff code (explicitly excluded)
- .agents/skills/resume/SKILL.md — skill doc change; excluded (not handoff code)
- .agents/skills/ship/SKILL.md — skill doc change; excluded (not handoff code)
- .agents/skills/work-unit-commits/SKILL.md — skill doc change; excluded (not handoff code)

Untracked-but-not-handoff, also left alone (not in the explicit file list): `.opencode/`, `docs/assets/devopstales.png`, `site/.hugo_build.lock`, `site/DESIGN.md`, `skillgrid-cli/internal/mnemonic/store/fk_probe_test.go` (generic FK probe — logs `PRAGMA foreign_keys` state, no handoff identifiers, not in the list).

## build:check result

`cd skillgrid-ui && npm run build:check` → PASS. tsc now resolves `./features/handoff/HandoffHubPage`; Rollup built in 31.11s; bundle-size budget OK (index 295.6 kB / budget 400 kB).

## Note on the count-bump test files

The six `*_test.go` files left unstaged all change only the shared MCP tool-surface assertion from `!= 83` to `< 87` — a mechanical consequence of the +5 handoff tools registered by `server.go`/`tools_handoff.go`. They contain no handoff-specific identifiers and were not enumerated in the explicit "stage ONLY these" list, so per the RULE they were left unstaged. **Caveat for the verifier:** these six assertions are what make `go test ./internal/mnemonic/mcp/...` go green now that the handoff tools exist; the task notes Go tests were green with these file contents already present in the tree, so the commit (which changes no file contents, only tracking) keeps them green locally. If this commit is checked out alone and `go test` is run, these six unstaged files are the only ones that would need to follow.
