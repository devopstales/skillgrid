# Review — mnemonic MCP family split

> Change: `.skillgrid/change-specs/2026-11-08-mnemonic-mcp-split/` (moves to `.skillgrid/change-archive/2026-11-08-mnemonic-mcp-split/` at ship)
> Generated: 2026-11-09T00:30:00+01:00 (requesting-code-review)
> Diff range: `294d2b5b..8ed6dd89`
> Rigor tier: T2 (per `_shared/planning/rigor-tiers.md`)
> Independence: Grade B per axis — see `## Independence`
>
> The durable audit record of the code review. **requesting-code-review** writes this into
> the spec folder, next to `report.md` (which owns the QA gate). It is committed in the spec
> zone before code-zone work continues. **ship** archives it with the folder; **reflect**
> cites it for lineage. The three review axes are reported **side by side, never merged** —
> that separation is the point.

## What Important means

- **Critical** — must fix before merge. Security, data loss, broken functionality, a
  `SATISFIES` scenario whose RED evidence is missing or whose code path does not match the
  Given/When/Then. Blocks `ship`.
- **Important** — should fix before merge. Architecture drift from a term or an in-force
  ADR, a hard standard breach, a missing or partial requirement, a behavior the spec never
  asked for (scope creep). Unfixed Important issues do not proceed (per the skill's
  Red Flags).
- **Minor** — nice to have. Baseline smells, style, optimization, doc polish. Capped to 5
  listed per axis; the rest collapse into a count.

## Passes

### Standards

- **Worst issue (within this axis):** the gitnexus block swallowed AGENTS.md's `<!-- skillgrid:end -->` sentinel — the managed region is unclosed, breaking the documented onboarding upsert contract.
- **Findings:** 0 Critical / 4 Important / 4 Minor listed (9-13 are FYI, no cap pressure)
- **Verdict:** met-with-fixes (rendered 2026-11-09; fixes applied — see `## Triage`)

### Spec

- **Worst issue (within this axis):** user-guide 05 misstates the code server's pinned initialize shape (claims an instruction block; the contract is `instructions == ""`) — independently corroborated by the Standards axis.
- **Findings:** 0 Critical / 0 Important / 2 Minor (docs misstatement; same-edit reindex not honored) + 4 scope-creep items (3 accepted-and-logged, 1 fixed)
- **Verdict:** met (all 32 scenarios PASS; both modules' full `-race` suites re-run green live by the reviewer)

### Security

- **Worst issue (within this axis):** fence-escape in the new `code_explore` renderer — verbatim file content containing 3+ backticks can terminate the code fence and be read as renderer prose (indirect prompt-injection channel into the agent).
- **Findings:** 0 Critical / 0 High / 0 Medium / 1 Low / 2 Info
- **Verdict:** secure-with-fixes (Low fixed — see `## Triage`)

## Findings

### Critical

- None (all three axes).

### Important

- [standards] `AGENTS.md:134-177` — the `<!-- gitnexus:start -->` block replaced the `<!-- skillgrid:end -->` sentinel; the `skillgrid:start` region (line 69) is never closed. Violates the documented upsert contract (`onboarding/SKILL.md:67,122-123`, `_shared/agent-config/block.md:86-87`). The next onboarding render has no end marker and would clobber the gitnexus block or stall. — fix: re-insert `<!-- skillgrid:end -->` before `<!-- gitnexus:start -->`.
- [standards] commit `08f8984c` bundles five unrelated work units into the D1 rename (3 change-spec folder deletions, the 264-line `projectwiki-data-loading` skill + 013 lines, ADR-0050/sdd-pipeline anchors + index re-render, the gitnexus block + `CLAUDE.md`) — violates `commits.md` "one logical change per commit". Sub-issue: the 3 deleted draft folders (`2026-10-05-mnemonic-central-pg`, `2026-10-06-mnemonic-standalone`, `2026-10-07-mnemonic-compaction-v2`, all `status: draft`, ~3,100 lines) were deleted outright, not archived, with no recorded rationale — ADR-0048 keeps closed changes as immutable records. — fix: restore the folders to `change-specs/` from `294d2b5b`; record the disposition here and in `report.md`.
- [standards] `.agents/skills/projectwiki/SKILL.md:31-34,74` documents 12 page types; the engine in this tree accepts 8 (`PageTypeDirs` 6 + indexer `skills`/`cases`; `ValidatePagePath` rejects the rest). The four `(013)` folders belong to the wiki-autolearning change (`2026-10-31-mnemonic-v2-knowledge-base`, ADR-0060 `status: proposed`), whose Consequences defer the skill-zone edits to "after this spec lands". — fix: revert the 013 lines; park the 013-only `projectwiki-data-loading` skill in that change's folder.
- [standards + spec] `docs/user-guide/05-memory-and-indexing.md:244` claims the code server's `initialize` carries "the code instruction block"; the pinned contract is the opposite (ADR-0061 D3; `TestInitializeShapePerFamily/code` asserts `Instructions == ""`). The discipline lives in the `code_explore` tool description. — fix: rewrite the sentence.

### Minor

- [standards] `mnemonic/internal/transport/mcp/server.go:38,42` — stale tool-count comment: "code 19 / code 38"; `code_explore` (TICKET-05) makes it 20/39. — fix: 19→20, 38→39.
- [standards] `mnemonic/internal/transport/mcp/server_family_test.go:453` — stale comment "queueSize 1 backpressures the reader"; the queue is pinned at 10000 since `10e4206b`. — fix: update the comment.
- [standards] `.skillgrid/projectwiki/decisions/04-adr-0061-mcp-family-split.md` — Verification section cites `TestMCPInitializeShape` / `TestMCPToolSetFamilyOnly` / `TestMCPInstructionsAreFinal`; actual tests are `TestInitializeShapePerFamily` / `TestToolIsolationPerFamily`, and no byte-identical instructions test exists (shape-pinned only). — fix: correct the names; soften "byte-identical" to "shape-pinned".
- [standards] `mnemonic/internal/transport/mcp/server.go:340` — dead code: `setInstructions` has zero callers at head; `toolDefs` (server.go:214) is read only inside it; 3 comments still describe it as live. — fix: delete both; keep `registerTool`.
- [standards] `mnemonic/cmd/mnemonic/main.go:1`, `mnemonic/internal/intelligence/embedder/embedder.go:1`, `mnemonic/internal/intelligence/llm/llm.go:1` — package doc comments still say "mnemonic-v2" (the rename touched `main.go` but not its header; the drift guard only matches the `mnemonic-v2/` path pattern). — fix: update the 3 comments.
- [security] `mnemonic/internal/transport/handler/explore_format.go:117-126` — [Low] fence-escape: verbatim file content with a run of 3+ backticks terminates the ``` fence early; the remainder renders as unquoted prose inside a document whose framing asserts authority ("verbatim, current on-disk source") — indirect prompt-injection channel (CWE-116, OWASP A05:2025, LLM01 Prompt Injection, LLM10 Improper Output Handling, ASI02 Tool Misuse & Exploitation). Precondition: the planted line ranks in the top-20 hits; the repo is already fully readable by the agent, so this adds a channel, not new data. Confidence: Likely (mechanism confirmed in code; model-dependent impact). — fix: per-block fence = longest backtick run in the span text + 1.
- [spec] `docs/user-guide/05-memory-and-indexing.md:244` — same misstatement as the [standards] item above (both axes found it independently). — fixed with it.
- [spec] the blueprint's same-edit reindex rule was not honored for T6/T7 — committed `index/last-update.json` (run `2026-11-08T09:20:26Z`, rename commit) predates the `ARCHITECTURE.md` rewrite and ADR-0061; the vault FTS index at head is stale. — fix: re-run `projectwiki index` (also re-renders after the vault edits below).

**Scope creep (spec axis):** (1) wiki-autolearning artifacts ride in range (013 spec, 001 amendment, ADR-0060, data-loading skill, anchors) — partially fixed above; ADR-0050's new anchor pins the dead path `mnemonic-v2/go.mod` — fixed to `mnemonic/go.mod`. (2) GitNexus onboarding block in `CLAUDE.md`/`AGENTS.md` — **accepted** (user tooling onboarding; the I1 sentinel fix restores the managed-region contract around it). (3) 3 draft spec folders deleted — **fixed** (restored). (4) W001 v1-installer repairs (`0b9e65cd`) — **accepted** (user fix-now decision, QA-recorded).

**FYI (standards, no action):** dev-stage naming residue (`(v2)` banner, `MEM2_*` Taskfile vars); `server.go` grew to 766 lines (split signal, pre-existing); redundant blast-radius recomputation on the explore path (bounded, results identical); W002 open (properly tracked).

**Info (security, no action):** the five-control tool-invocation set is present as implemented (family-gated deny-by-default allowlist, mcp-go v1.5.0 schema validation, spawn-boundary identity, `--profile agent` policy gate, no HITL by design); pre-existing adjacent surfaces unchanged (code_read arbitrary-read High from the 2026-11-03 review; repo-driven `npm install -g` supply-chain trade-off).

## Independence

| Axis | Grade | Rationale |
|------|-------|-----------|
| Standards | B | fresh `reviewer` subagent context, no session history or prior verdict passed; same model family / toolchain as the implementer |
| Spec | B | fresh `reviewer` subagent context, no session history or prior verdict passed; same model family / toolchain as the implementer |
| Security | B | fresh `security-reviewer` subagent context, no session history or prior verdict passed; same model family / toolchain as the implementer |

**Reading the grade:** **A** = fresh subagent, no shared history (independent). **B** = fresh
context but same model family / toolchain as the implementer (independent, weaker). **C** =
inline self-review or the implementer's verdict leaked in — **diagnostic only**.

## Triage (receiving-code-review)

| Finding | Disposition | Evidence |
|---------|-------------|----------|
| I1 AGENTS.md sentinel | **fixed** | re-inserted `<!-- skillgrid:end -->` before the gitnexus block |
| I2 bundled commit + deleted folders | **fixed (data) + logged (history)** | 3 draft folders restored to `change-specs/` from `294d2b5b`; disposition recorded in `report.md`; the commit itself is historical (no rebase planned) |
| I3 12-vs-8 page types | **fixed** | 013 lines reverted in `projectwiki/SKILL.md`; `projectwiki-data-loading/SKILL.md` parked in the knowledge-base change folder |
| I4 user-guide 05 wire contract | **fixed** | sentence rewritten to the pinned shape |
| Security Low fence-escape | **fixed** | per-block fence = longest backtick run + 1, with a pinning test |
| Spec: stale vault index | **fixed** | `projectwiki index` re-run in the vault commit |
| ADR-0050 dead pin | **fixed** | `mnemonic-v2/go.mod` → `mnemonic/go.mod` |
| M5-M9 minors | **fixed** | comment/ADR/dead-code edits in the hygiene commit |
| Scope creep: gitnexus block, W001 repairs | **accepted** | logged above; sentinel contract restored around the block |

## Verdict

- **Standards:** met-with-fixes → **met** (all 4 Important fixed)
- **Spec:** met (all 32 scenarios PASS; 2 Minor fixed, 3 creep items accepted-and-logged)
- **Security:** secure-with-fixes → **secure** (the 1 Low fixed)
- **Floor (decides):** **met**
- **Worst issue (across all three axes):** the unclosed `<!-- skillgrid:end -->` sentinel in AGENTS.md (a broken documented contract, not a runtime defect) — fixed.
- **Unfixed Important count (must be 0 to proceed):** 0
