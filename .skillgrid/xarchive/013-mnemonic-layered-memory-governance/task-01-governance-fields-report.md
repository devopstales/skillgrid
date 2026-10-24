# Step 01 Report — `01-governance-fields` (change 013)

**Status: PASS** — governance fields (owner, append-version, status, retrieval usage, private-by-default visibility) on every memory; surfaced by new `mem_share` / `mem_governance` tools; existing `mem_*` names + required params unchanged (additive only).

## Sub-task coverage

| Sub-task | Status | Evidence |
|----------|--------|----------|
| 01.1 `[RED]` Mnemonic tool surface | GREEN | `TestMemGovernanceTools` + `TestMemGovernanceRoundTrip` (mcp). RED→GREEN captured. |
| 01.2 `[RED]` Data leak / visibility | GREEN | `TestVisibility` + `TestBadGovernanceArgs` (memory). RED→GREEN captured. |
| 01.3 `[AFK]` Additive 017 schema | GREEN | `TestStoreOpenAppliesGovernanceSchema` + `TestGovernanceSchemaUpgradeFromLegacy` (store). |
| 01.4 `[AFK]` `mem_save` private-by-default + owner | GREEN | `TestPrivateDefault` (memory + service). |
| 01.5 `[AFK]` `mem_share` explicit widen; unknown rejected | GREEN | `TestMemShare` (memory + service) + `TestBadGovernanceArgs`. |
| 01.6 `[AFK]` `mem_update` appends recoverable version | GREEN | `TestUpdateVersion` (memory + service). |
| 01.7 `[AFK]` `superseded`/`archived` set explicitly | GREEN | `TestStatusExplicit` (memory + service). |
| 01.8 `[AFK]` retrieval-usage increments on search (≠ duplicate_count) | GREEN | `TestUsageCount` (memory + service). |
| 01.9 `[AFK]` `mem_governance` surfaces asset fields | GREEN | `TestGovernanceQuery` (memory + service) + `TestMemGovernanceRoundTrip` (mcp). |
| 01.10 `[AFK]` existing `mem_*` names + required params unchanged; bad args rejected | GREEN | `expectedMemToolSurface` baseline lock + `TestAllToolsRegistered` (77 tools) + `TestMemGovernanceTools` bad-args. |

## Migration number

The brief said `015_layered_memory_governance.sql` but 015 was already taken by 008's knowledge graph. Per the task instructions, the migration is **`017_layered_memory_governance.sql`** (the next-free number after `016_pdg_taint.sql`). Additive: `ALTER TABLE observations ADD COLUMN` for `owner`/`status`/`visibility`/`retrieval_usage` + `CREATE TABLE IF NOT EXISTS observation_versions` + `acl_grants`. No rewrite of existing tables; the legacy-upgrade test proves a pre-017 store upgrades additively with rows intact.

## How private-by-default + the owner model works

- **Owner:** every `Save` sets `owner` = the creating identity (`SaveInput.Owner`, falling back to the session ID so an asset is never unowned). A new observation always has an owner.
- **Private-by-default:** the `visibility` column defaults to `'private'` (explicit in the INSERT). The owner/agent always sees their own; a *second owner* does not, until an explicit `mem_share`.
- **Cross-owner isolation** is modeled at the read path via the `owner` column (single-operator mapping for a local store). `SearchOwner(readerOwner, …)` applies a visibility filter:
  - owner branch: `o.owner = ? OR COALESCE(visibility,'private') != 'private'`
  - non-owner branch: `team`-visible assets + `restricted`/`agent` assets explicitly granted to the reader (`EXISTS` over `acl_grants`).
- **`AdminCrossOwnerList(adminOwner)`:** returns the admin's own rows + non-private assets; a *private* observation of another owner is absent (private is invisible to other owners even to an admin's read list).

## How `mem_update` appends a version (prior content recoverable)

`Update` (timeline_update.go) now, before rewriting content: reads the current `revision_count`, calls `AppendVersion(id, rev+1)` which inserts the *pre-update* content into `observation_versions (observation_id, revision, content, created_at)`, then the `UPDATE` advances `revision_count = revision_count + 1` and rewrites content. The latest version (the live `observations.content`) is the read path; prior content is recoverable via `mem_governance` (which returns `versions` newest-first). The topic-key upsert path in `Save` also appends a version (revision 1) before rewriting, so an upsert is never a silent overwrite either.

## How restricted ACL is enforced

- `Share(id, {Visibility: restricted|agent, Grants: […]} validates + replaces `acl_grants` for the observation. An empty grant list is allowed (owner-only) and surfaced via `Governance.RestrictedNoGrants = true` (a condition, not an error).
- **Read enforcement** is via `canRead` (single-read `ReadAs`) and the `EXISTS` grant clause in `SearchOwner`/`AdminCrossOwnerList`: a `restricted`/`agent` observation is readable by the owner + granted agents only. A non-granted reader gets `ErrNotFoundForReader` (absent, not an error). A `restricted` observation with no grants is owner-only.
- **Bad `mem_share` args rejected clearly:** unknown target (not `team|restricted|agent`), an empty grantee for a `restricted`/`agent` share, and an invalid `SetStatus` all return a validation error with the observation's visibility/status left unchanged (no governance data invented).

## How the existing `mem_*` schema is proven stable + baseline updated (75→77)

- `expectedMemToolSurface` (mcp/tools_memory_governance_test.go) pins all 24 existing `mem_*` tools' names + required-param lists; `TestMemGovernanceTools` asserts each is unchanged on the live server.
- `TestAllToolsRegistered` updated to expect the two new tools and `77` total; `tools_code_affected_test.go` + `tools_code_knowledge_test.go` count assertions bumped `75 → 77`.
- `observationDTO` (tools_memory.go) gains additive `owner`/`visibility`/`status`/`retrieval_usage` on `mem_search` / `mem_get_observation` responses — existing fields untouched.
- `mem_share` required params = `id, target`; `mem_governance` required params = `id`.

## RED→GREEN evidence (01.1, 01.2)

**01.1 RED** — `go test ./internal/mnemonic/mcp/... -run GovernanceTools -count=1` (governance tools absent):
```
internal/mnemonic/mcp/server.go:91:2: undefined: registerMemoryGovernanceTools
internal/mnemonic/mcp/tools_memory_governance_test.go:67:14: undefined: handleMemShare
internal/mnemonic/mcp/tools_memory_governance_test.go:77:13: undefined: handleMemShare
internal/mnemonic/mcp/tools_memory_governance_test.go:88:14: undefined: handleMemGovernance
FAIL	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/mcp [build failed]
```
**01.1 GREEN** — `ok …/internal/mnemonic/mcp` (full mcp suite).

**01.2 RED** — `go test ./internal/mnemonic/memory/... -run Visibility -count=1` (governance API absent):
```
governance_test.go:113:21: fx.svc.Governance undefined (type *Service has no field or method Governance)
governance_test.go:134:19: fx.svc.SetStatus undefined
governance_test.go:168:24: fx.svc.SearchOwner undefined
governance_test.go:176:19: fx.svc.SearchOwner undefined
FAIL	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory [build failed]
```
**01.2 GREEN** — `ok …/internal/mnemonic/memory` (full memory suite).

## Full relevant suite + output

`go test ./internal/mnemonic/memory/... ./internal/mnemonic/store/... ./internal/mnemonic/mcp/... ./internal/mnemonic/service/... -count=1`
```
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory  14.862s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store   6.712s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/mcp    62.371s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service 44.914s
```

## Baselines still pass (005/008/010/011)

`go test ./internal/mnemonic/route/... ./internal/mnemonic/affected/... ./internal/mnemonic/community/... ./internal/mnemonic/pdg/... ./internal/mnemonic/codeindex/... -count=1`
```
ok  …/route        1.602s
ok  …/affected    19.669s
ok  …/community     3.627s
ok  …/pdg           4.867s
ok  …/codeindex    29.012s
```
The full `./internal/mnemonic/...` (23 packages, incl. http/integration/graph/search/tiered/webcache/embedder/eval/extract/files/knowledge/process/project) also passes — no cross-package regression from the `obsSelectCols`/`scanObservations` changes.

## Commits

- `28467ec` feat(store): 017 governance schema (owner/version/status/usage/visibility)
- `7caa2b7` feat(memory): governance core (private-by-default, append-version, status, usage, visibility/ACL)
- `dfbea4e` feat(mcp): register mem_share/mem_governance + stable mem_* schema (75→77 tools)

## Review finding (round 2) — per-owner read enforcement wired end-to-end

The review flagged that `canRead`/`ReadAs`/`SearchOwner`/`AdminCrossOwnerList`
were tested at the service seam but **never wired into a live `mem_*` tool
handler** — `mem_search`/`mem_get_observation` still called the unscoped
`Search`/`Get` with no visibility clause, so a live reader was not actually
gated per-owner. Fixed by wiring the seam into both handlers (commits
`c93b87d` + `8563453`).

### How the read path is wired

- **Reader identity (no new auth system):** the read handlers accept optional
  `reader_owner` / `reader_agent` params. Blank `reader_owner` falls back to
  the request `session_id`. This mirrors the **owner the save path already
  uses**: `mem_save` now threads an optional `owner` param into
  `SaveInput.Owner`, and the `Save` method falls back to `session_id` (and, as
  a belt-and-suspenders invariant, to the `legacy` sentinel) so an observation
  is never unowned. Therefore the **SAME session that saved can read it back,
  and a DIFFERENT owner is gated** — consistent with the existing
  single-operator owner model. `reader_agent` carries the ACL grantee for
  `restricted`/`agent` visibility.
- **`mem_search` → scoped search:** the handler now calls the new
  `memory.Service.SearchOwnerScoped(readerOwner, readerAgent, query, matchMode,
  scope, limit)` instead of the unscoped `SearchWithScope`. `SearchOwnerScoped`
  is `SearchWithScope` + the `visibilityFilter(owner, readerOwner, readerAgent)`
  clause, with the same retrieval-usage bump on each hit. A private
  observation is invisible to a different owner until shared; the owner always
  sees their own; team/restricted/agent follow the existing filter semantics.
  (When no reader identity is supplied — the single-operator default — the
  reader is treated as the owner, so existing callers keep seeing everything.)
- **`mem_get_observation` → scoped read:** the handler now calls
  `memory.Service.ReadAs(readerOwner, id, readerAgent)` (which enforces
  `canRead`) instead of the unscoped `Get`. On `ErrNotFoundForReader` it
  returns a not-found shape (`{id, not_found, not_found_for_reader}`) rather
  than leaking a visibility error — a reader who can't see the observation sees
  it as **absent**, consistent with `ErrNotFoundForReader`.

### Empty-owner consistency fix (the minor finding)

`canRead` previously returned `ErrVisibilityNotSet` when the owner column was
empty, while `visibilityFilter` uses `COALESCE(visibility,'private')` — so a
legacy (pre-017) row could read via search but error via `ReadAs`. Now `canRead`
maps an empty owner to the `legacy` sentinel (private by default), matching the
save-path fallback, so **both read paths agree**. `TestLegacyEmptyOwnerConsistency`
inserts a legacy row (empty owner) and asserts both `SearchOwner` and `ReadAs`
return the same visibility result (absent for every live reader, and
`ReadAs` does not surface `ErrVisibilityNotSet`).

### End-to-end tool-handler test (the missing assertion)

`TestMemSearchGetOwnerEnforcedWiring` (mcp) drives the **actual tool
handlers** — `handleMemSearch` and `handleMemGetObservation` via
`openService` → project handle → scoped read — not `SearchOwner` directly. It
saves as owner A (A's session), then:
- **before share:** owner B (`reader_owner=owner-b-reader`) gets 0 hits from
  `mem_search` and an absent/not-found result from `mem_get_observation`
  (no visibility error leak);
- **after `mem_share` to team:** owner B sees the observation in both
  `mem_search` and `mem_get_observation`.

This is the end-to-end assertion the reviewer said was missing.

### Additive-surface constraint (still holds)

`reader_owner`/`reader_agent` are **optional** params on `mem_search` and
`mem_get_observation`; `owner` is optional on `mem_save`. The 24-tool
`expectedMemToolSurface` lock (names + **required** params) stays green —
`TestMemGovernanceTools` + `TestAllToolsRegistered` (77 tools) confirm.

### Full relevant suite + output (round 2)

`go test ./internal/mnemonic/memory/... ./internal/mnemonic/store/... ./internal/mnemonic/mcp/... ./internal/mnemonic/service/... -count=1`
```
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory  15.346s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store   7.559s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/mcp    63.812s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service 46.019s
```

### Baselines still pass (005/008/010/011)

`go test ./internal/mnemonic/route/... ./internal/mnemonic/affected/... ./internal/mnemonic/community/... ./internal/mnemonic/pdg/... ./internal/mnemonic/codeindex/... -count=1`
```
ok  …/route        1.729s
ok  …/affected    17.196s
ok  …/community     3.863s
ok  …/pdg           4.678s
ok  …/codeindex    24.699s
```
The full `./internal/mnemonic/...` (all packages, incl. http/integration) also passes — the wiring is additive and no other package regressed.

### Commits (round 2)

- `c93b87d` feat(memory): wire per-owner read enforcement + empty-owner consistency
- `8563453` feat(mcp): wire per-owner read enforcement into mem_search/mem_get_observation

## Concerns / notes

- **FTS reserved-token gotcha:** the FTS5 query is built by `buildFTSQuery` (quotes each token, joins with `AND`/`OR`). A multi-token or reserved-looking term (e.g. `"alpha"` tokenizes to `"al" AND "pha"`, leaking a bare `AND`) breaks the query; owner ids with hyphens (`owner-b`) parse as `owner MINUS b`. The test fixtures use single clean tokens (`widget`, `ownerA`) — this is a pre-existing `buildFTSQuery` property, not a regression, but worth noting for step 03 (budgeted retrieval).
- **Owner model is single-operator:** cross-owner isolation is modeled by the `owner` column within one store bucket (no multi-tenant teams/roles — out of scope per the brief). Step 02/03 consume these fields as-is.
- `mem_share` target `private` is rejected (private is the default, not an explicit widen in step 01); a `private` observation is simply "not shared".
