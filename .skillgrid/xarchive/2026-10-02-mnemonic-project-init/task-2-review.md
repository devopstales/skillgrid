# Task 2 review — TICKET-02 Preamble and sentinel upsert

**Base:** b5239413 · **Head:** 9dd840f5
**Diff:** `.skillgrid/sdd/2026-10-02-mnemonic-project-init/task-2.diff`
**Constraints checked:** ADR-0012 (no second store / no KB); exactly one `<!-- skillgrid:start -->` / `<!-- skillgrid:end -->` pair; `--force` replaces preamble only; no index/ingest in this ticket.

**Spec:** PASS
**Quality:** approved
**Overall:** APPROVED

---

### Spec Compliance

- ✅ Spec compliant
- ⚠️ Cannot verify from diff: focused `TestInit*` PASS (parent re-verified; not re-run here). Cucumber dry-run absence is environmental, not a code defect for this ticket.

**Checked against brief / TICKET-02:**

| Requirement | Evidence |
|---|---|
| `writeBootFile(dir, force)` | `init_boot.go:71` |
| `projectInit` calls `writeBootFile` | `init_cmd.go:103-111` |
| Happy path: preamble markers + `# Definition of Done` + exactly one sentinel pair | `TestInitUpsertsPreambleAndSentinel`; `upsertPreamble` + `upsertSentinel` |
| Second init: no sentinel duplicate; user text outside regions kept | `TestInitForceRewritesPreamble` merge phase (`USER KEEP` / `USER BELOW`); `upsertSentinel` replace-in-place `init_boot.go:261-268` |
| `--force` rewrites preamble only | `upsertPreamble` forced branch `init_boot.go:243-248`; test asserts `USER KEEP` gone, `USER BELOW` retained |
| Boot target AGENTS vs CLAUDE + CLAUDE pointer | `resolveBootPath` `init_boot.go:102-137`; `ensureClaudePointer` `init_boot.go:125-146` |
| Sentinel from `block.md` placeholders (local-only tracker, enabled memory, rules) | `sentinelTemplate` + `renderSentinel` `init_boot.go:24-69,148-155` |
| No `RunCodeIndex` / ingest | `projectInit` still stubs `svc` / `extraDocs` (`init_cmd.go:91-93`); Indexed/Ingested remain zero |
| ADR-0012 | No new store, KB, or memory write path in this diff |

**Missing / Extra / Misunderstood:** none for T02 scope.

---

### Strengths

- Clean split: boot upsert isolated in `init_boot.go`; `projectInit` is a thin wire-up.
- Preamble states (`written` / `kept` / `forced`) match the blueprint contract and flow into `initResult.Preamble`.
- Sentinel upsert is replace-in-place (first start…end), so a second run cannot append a second pair on well-formed files.
- Acceptance tests match the brief verbatim and exercise merge + force in the mandated scenarios.
- Stdlib-only; no new dependencies.

---

### Issues

#### Critical (Must Fix)

None.

#### Important (Should Fix)

None.

#### Minor (Nice to Have)

1. **`init_boot.go:261-268` / `239-258` — orphan start marker appends a second block.** If `<!-- skillgrid:start -->` (or preamble start) exists without a following end marker, the code falls through to append a full block, producing duplicate start markers. Out of T02 acceptance scope (assumes well-formed regions); harden later if corrupt boot files are in threat model.
2. **`init_cmd_test.go:361-395` — force path does not re-assert end-marker count.** Merge asserts one `skillgrid:start`; after `--force`, only preamble content is checked. Behavior is correct via replace-in-place; a one-line `strings.Count` on both markers after force would lock the acceptance scenario more tightly.
3. **Sentinel body duplicated as a Go string** (`init_boot.go:24-69`) vs `block.md`. Explicitly allowed by the blueprint threat matrix; drift risk remains as noted in the implementer report — not blocking.

---

### Assessment

**Task quality:** Approved

**Reasoning:** Diff implements TICKET-02 end-to-end: preamble insert/keep/force, single-pair sentinel upsert, AGENTS/CLAUDE resolution, and no index/ingest leakage. Acceptance tests cover the three SATISFIES scenarios; no Critical or Important defects.

**Tests:** Trusted parent re-verification of focused `TestInit*` PASS; no re-run (no specific behavioral doubt from reading the diff).
