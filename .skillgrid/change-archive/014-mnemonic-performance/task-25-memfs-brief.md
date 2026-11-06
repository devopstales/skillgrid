## 25-memfs

### Goal

`memfs` virtual filesystem: `mem ls`, `mem tree`, `mem find` alongside existing `mem search`.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-25` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 07, 18, 19

**Files:**
- `internal/mnemonic/memfs/memfs.go` — CREATE: URI resolution + scope management + filesystem operations
- `internal/mnemonic/memfs/ls.go` — CREATE: `mem ls <scope>` directory listing
- `internal/mnemonic/memfs/tree.go` — CREATE: `mem tree <scope>` hierarchical tree view
- `internal/mnemonic/memfs/find.go` — CREATE: `mem find <pattern>` filesystem-style search
- `internal/mnemonic/memfs/scope.go` — CREATE: scope management (`mem://user/{id}/`, `mem://project/{id}/`)
- `internal/mnemonic/memfs/memfs_test.go` — CREATE: memfs tests
- `internal/mnemonic/cmd/skillgrid/mem.go` — MODIFY: add `fs` subcommand group

**Interfaces:**
- Consumes: triple-store linkage from step 07; memory types from step 18; directory retrieval from step 19
- Produces: `mem fs` CLI group with `ls`, `tree`, `find`; `mem://` URI space; scope-based filtering on the same store

### Tasks

- [ ] 25.1 `[RED]` mem ls lists observations in a scope (e.g., project/{id}/preferences)
  - [ ] 25.1.a Write failing test (`TestMemLSScopeListing`): create observations in scopes `project/A/preferences`, `project/A/entities`, `user/B/profile`; invoke `mem ls project/A/preferences`; verify it lists only the preferences observations; invoke `mem ls project/A/`; verify it lists all observations under project A; invoke `mem ls user/B/`; verify it lists only user B's observations
  - [ ] 25.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memfs/ -run 'TestMemLSScopeListing'` — Expected: FAIL
  - [ ] 25.1.c Minimal implementation — create `memfs.go` with `ResolveURI(uri string) (ScopeFilter, error)` that parses `mem://` URIs into scope filters; create `ls.go` with `List(ctx, scope string) ([]Observation, error)` that queries the store with `WHERE memory_type = ? AND scope = ?`; the scope is an additive metadata column on observations
  - [ ] 25.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memfs/ -run 'TestMemLSScopeListing'` — Expected: PASS
  - [ ] 25.1.e Commit — `feat(mnemonic): mem ls scope-based directory listing`
- [ ] 25.2 `[RED]` mem tree shows hierarchical tree view of memory scopes
  - [ ] 25.2.a Write failing test (`TestMemTreeHierarchical`): create observations in nested scopes `project/A/preferences/sub1`, `project/A/preferences/sub2`, `project/A/entities`; invoke `mem tree project/A/`; verify the output shows a hierarchical tree with `preferences/` containing `sub1` and `sub2`, and `entities/` as a sibling; verify the tree is indented correctly
  - [ ] 25.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memfs/ -run 'TestMemTreeHierarchical'` — Expected: FAIL
  - [ ] 25.2.c Minimal implementation — create `tree.go` with `Tree(ctx, scope string) (string, error)` that queries all scopes under the given prefix; builds a hierarchical tree structure; renders it as indented text with `├──` and `└──` characters; group by scope path segments
  - [ ] 25.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memfs/ -run 'TestMemTreeHierarchical'` — Expected: PASS
  - [ ] 25.2.e Commit — `feat(mnemonic): mem tree hierarchical scope view`
- [ ] 25.3 `[RED]` mem find does filesystem-style pattern matching across observations
  - [ ] 25.3.a Write failing test (`TestMemFindPatternMatching`): create observations with titles "auth.go", "auth_test.go", "payment.go", "user.go"; invoke `mem find "auth*"`; verify it returns "auth.go" and "auth_test.go"; invoke `mem find "*.go"`; verify it returns all 4; invoke `mem find "auth*test*"`; verify it returns only "auth_test.go"
  - [ ] 25.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memfs/ -run 'TestMemFindPatternMatching'` — Expected: FAIL
  - [ ] 25.3.c Minimal implementation — create `find.go` with `Find(ctx, pattern string, scope string) ([]Observation, error)` that translates glob patterns to SQL `LIKE` queries; support `*` (any chars) and `?` (single char); search across observation titles and content; scope the search to the given scope
  - [ ] 25.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memfs/ -run 'TestMemFindPatternMatching'` — Expected: PASS
  - [ ] 25.3.e Commit — `feat(mnemonic): mem find with glob pattern matching`
- [ ] 25.4 `[RED]` mem:// URI resolution maps to SQL queries on the same store
  - [ ] 25.4.a Write failing test (`TestMemURIResolution`): resolve `mem://project/A/preferences` and verify it produces a SQL filter for `project_id = 'A' AND memory_type = 'preferences'`; resolve `mem://user/B/` and verify it produces a filter for `user_id = 'B'`; resolve an invalid URI `mem://` and verify it returns a parse error; verify the URI resolution is fast (sub-millisecond)
  - [ ] 25.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memfs/ -run 'TestMemURIResolution'` — Expected: FAIL
  - [ ] 25.4.c Minimal implementation — create `scope.go` with `ParseURI(uri string) (ProjectID, UserType, Scope, error)` that parses `mem://project/{id}/` and `mem://user/{id}/` patterns; validate the URI format; return structured scope components; ensure resolution is a pure string operation (no DB call)
  - [ ] 25.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memfs/ -run 'TestMemURIResolution'` — Expected: PASS
  - [ ] 25.4.e Commit — `feat(mnemonic): mem:// URI resolution engine`
- [ ] 25.5 `[AFK]` mem ls coexists with mem list without conflict
  - [ ] 25.5.a Write failing test (`TestMemLSCoexistsWithMemList`): create observations; invoke `mem list` and verify it returns flat output (all observations); invoke `mem ls project/A/` and verify it returns directory-like output (scoped); verify both commands work independently; verify `mem ls` does not affect `mem list` output and vice versa
  - [ ] 25.5.b Run to confirm fail — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemLSCoexistsWithMemList'` — Expected: FAIL
  - [ ] 25.5.c Minimal implementation — ensure `mem fs ls` (and its siblings) are registered as a separate subcommand group under `mem fs`; do not modify the existing `mem list` command; both query the same store but with different filters (scoped vs flat)
  - [ ] 25.5.d Run to confirm pass — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemLSCoexistsWithMemList'` — Expected: PASS
  - [ ] 25.5.e Commit — `feat(mnemonic): mem fs coexists with mem list without conflict`

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memfs/ -run 'TestMemLSScopeListing\|TestMemTreeHierarchical\|TestMemFindPatternMatching'` | PASS | | |
| Acceptance `@step-25` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memfs/` | PASS | | |
| Rollback boundary | verify memfs queries same store, does not replace mem search | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): memfs virtual filesystem with ls, tree, find alongside mem search`

---

