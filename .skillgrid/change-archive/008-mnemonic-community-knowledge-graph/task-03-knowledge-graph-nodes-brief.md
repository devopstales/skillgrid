## 03-knowledge-graph-nodes

### Goal

Doc/config/SQL extractors + non-code nodes + `references`/`configures`/`reads`/`writes` edges + indexer hook + knowledge MCP/CLI tools so the graph spans code and its knowledge, and `code_path` can trace code→doc→config→table in one query.

### Out of scope / Non-Goals

- Community detection (step 01); process flows (step 02)
- Video/audio/image; LLM semantic doc↔code links

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-03` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 02-process-flows

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/knowledge/doc.go`
- Create: `skillgrid-cli/internal/mnemonic/knowledge/config.go`
- Create: `skillgrid-cli/internal/mnemonic/knowledge/sql.go`
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_code_knowledge.go`
- Modify: `skillgrid-cli/internal/mnemonic/codeindex/indexer.go` (hook community + process + knowledge passes after 005 extraction, same tx)
- Modify: `skillgrid-cli/internal/mnemonic/mcp/server.go` (register new tool sets)
- Modify: `skillgrid-cli/cmd/skillgrid/main.go` (CLI dispatch)
- Test: `skillgrid-cli/internal/mnemonic/knowledge/...`, `skillgrid-cli/internal/mnemonic/codeindex/...`, `skillgrid-cli/internal/mnemonic/mcp/...`, `skillgrid-cli/cmd/skillgrid/...`

**Interfaces:**
- Consumes: 005 symbols/edges + `code_path`, 01 communities, 02 processes, indexer hook seam
- Produces: `doc_nodes` / `config_nodes` / `sql_schema_nodes` tables; `references` (doc→doc), `configures` (config→code), `reads`/`writes` (code→table) edges; indexer hook running community + process + knowledge passes in the same incremental tx; knowledge query tools; `code_path` spanning code→doc→config→table

### Tasks

- [ ] 03.1 `[RED]` Mnemonic tool surface — knowledge tools register + 005 tools stable + bad args rejected (Scenario: Knowledge tools register and reject bad args) — threat: Mnemonic tool surface
  - [ ] 03.1.a Write failing test — knowledge query tools registered with distinct `code_*` names; 005 tools still name/param-stable; bad/missing args rejected clearly
  - [ ] 03.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run KnowledgeTools -count=1` — Expected: FAIL
  - [ ] 03.1.c Minimal implementation — `tools_code_knowledge.go` + `server.go` registration + `main.go` dispatch
  - [ ] 03.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run KnowledgeTools -count=1` — Expected: PASS
  - [ ] 03.1.e Commit — `feat(mnemonic): register knowledge-graph query tools`
- [ ] 03.2 `[RED]` Markdown links/wikilinks become references edges (Scenario: Markdown links and wikilinks become references edges)
  - [ ] 03.2.a Write failing test — `.md` files with `[text](./other.md)` and `[[wikilinks]]` produce `doc_nodes` + `references` edges between doc nodes, each confidence-labeled
  - [ ] 03.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/knowledge/... -run DocLinks -count=1` — Expected: FAIL
  - [ ] 03.2.c Minimal implementation — `knowledge/doc.go` (markdown link/wikilink parser → `references`)
  - [ ] 03.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/knowledge/... -run DocLinks -count=1` — Expected: PASS
  - [ ] 03.2.e Commit — `feat(mnemonic): markdown doc-link extractor`
- [ ] 03.3 `[RED]` Config refs become configures edges (Scenario: Config references become configures edges)
  - [ ] 03.3.a Write failing test — `.yaml`/`.toml`/`.json` files produce `config_nodes` + `configures` edges to the code they configure; explicit syntax is `EXTRACTED`, resolved-but-inferred refs are `INFERRED`
  - [ ] 03.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/knowledge/... -run ConfigRefs -count=1` — Expected: FAIL
  - [ ] 03.3.c Minimal implementation — `knowledge/config.go` (key→symbol resolver → `configures`)
  - [ ] 03.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/knowledge/... -run ConfigRefs -count=1` — Expected: PASS
  - [ ] 03.3.e Commit — `feat(mnemonic): config-ref extractor`
- [ ] 03.4 `[RED]` SQL DDL becomes table/column nodes with reads/writes (Scenario: SQL DDL becomes table and column nodes with reads and writes)
  - [ ] 03.4.a Write failing test — `.sql` DDL produces `sql_schema_nodes` (tables + columns) and code that references them gets `reads`/`writes` edges, confidence-labeled
  - [ ] 03.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/knowledge/... -run SqlSchema -count=1` — Expected: FAIL
  - [ ] 03.4.c Minimal implementation — `knowledge/sql.go` (DDL parser + identifier-ref scan → `reads`/`writes`)
  - [ ] 03.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/knowledge/... -run SqlSchema -count=1` — Expected: PASS
  - [ ] 03.4.e Commit — `feat(mnemonic): sql-schema extractor`
- [ ] 03.5 `[AFK]` Unresolvable config ref is AMBIGUOUS, not dropped (Scenario: Unresolvable config ref is ambiguous not dropped) — `Run: go test ./skillgrid-cli/internal/mnemonic/knowledge/... -run AmbiguousConfigRef -count=1` — Expected: PASS
- [ ] 03.6 `[AFK]` Malformed doc file: skip bad links, index the rest (Scenario: Malformed doc file falls back and indexes the rest) — `Run: go test ./skillgrid-cli/internal/mnemonic/knowledge/... -run MalformedDoc -count=1` — Expected: PASS
- [ ] 03.7 `[AFK]` Malformed SQL statement: skip it, index the rest (Scenario: Malformed SQL statement is skipped and the rest is indexed) — `Run: go test ./skillgrid-cli/internal/mnemonic/knowledge/... -run MalformedSQL -count=1` — Expected: PASS
- [ ] 03.8 `[AFK]` Indexer hook runs all three passes in the same transaction (Scenario: Indexer hook runs community, process, and knowledge in one transaction) — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/... -run IndexerHook -count=1` — Expected: PASS
- [ ] 03.9 `[AFK]` code_path traces code to doc to config to table (Scenario: Path tool traces code to doc to config to table) — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/internal/mnemonic/knowledge/... -run CodePathSpan -count=1` — Expected: PASS

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/knowledge/... ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/mcp/... -count=1` | PASS | | |
| Acceptance `@step-03` / `@p0` | BDD / mapped unit scenarios (incl. `DocLinks`, `ConfigRefs`, `SqlSchema`) | PASS | | |
| Runtime harness | `skillgrid index` on fixture with docs/configs/SQL; `code_path` code→doc→config→table | PASS | | |
| Rollback boundary | Drop `knowledge/` + knowledge tools + indexer hook | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): doc, config, and sql knowledge-graph nodes`

---

