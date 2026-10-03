# Review — mnemonic-project-init

> Change: `.skillgrid/specs/2026-10-02-mnemonic-project-init/`
> Generated: 2026-10-03T07:50:00Z (parallel-code-review)
> Re-rendered: 2026-10-03T07:58:00Z after fix `cb077edf`
> Diff range (original): `f739d371..17652346`
> Fix commit: `cb077edf`
> Rigor tier: T2 (escalated to fan-out: 1115 lines ≥ 50)
>
> Specialists: Standards, Spec, Edge, Verification-gap, Security, Performance, Red-team (last).
> Accessibility skipped (no UI).

## What Important means

- **Critical** — must fix before merge (security, data loss, broken SATISFIES).
- **Important** — should fix before merge (ADR/SSOT hard breach, missing requirement, scope creep).
- **Minor** — nits; capped at 5 listed per axis.

## Independence

| Axis | Grade | Rationale |
|------|-------|-----------|
| Standards | B | fresh subagent, same-family retry after model-limit |
| Spec | B | fresh subagent, same-family retry after model-limit |
| Edge | B | fresh subagent (`composer-2.5-fast`) |
| Verification-gap | B | fresh subagent, same-family retry after model-limit |
| Security | B | fresh subagent, same-family retry after model-limit |
| Performance | B | fresh subagent (`composer-2.5-fast`) |
| Red-team | B | fresh subagent after merged findings |

First dispatch of Standards/Spec/Verification/Security failed (`Other Models` limit). Retried on the session model. Grade B, not C — no implementer verdict leaked.

## Per-specialist counts

| Specialist | Count | Worst |
|------------|-------|-------|
| Standards | 8 Important + 5 Minor (capped) | mutable `initRunIndex`; swallowed extra walk; inlined sentinel |
| Spec | 18 SATISFIES product paths PASS; process gaps | spec-zone files untracked; two MISSING_RED tests |
| Edge | 11 | extra `--docs` project-root walk; broken marker upsert |
| Verification-gap | 8 broken-verification | help and `--docs` tests unbound from production flags |
| Security | 2 | boot symlink overwrite; lexical `--docs` jail |
| Performance | 7 | unbounded WalkDir; N+1 Save; no size cap |
| Red-team | 5 | 7-day TTL on ingest; session-private owner; `MNEMONIC_PROJECT` split |

## Deduped entries (coordinator verdicts)

| ID | Verdict | Source | Location | Finding | Route |
|----|---------|--------|----------|---------|-------|
| D1 | high | Security+Edge | `init_ingest.go` jail | Symlink escapes lexical `--docs` jail | **fixed** in `cb077edf` (EvalSymlinks + Lstat + tests) |
| D2 | high | Security | `init_boot.go` WriteFile | Symlink boot target overwrites host file | **fixed** in `cb077edf` (Lstat refuse + test) |
| D3 | medium | Security+Perf | `saveIngestFile` | No size bound / binary blob | **fixed** in `cb077edf` (512KiB + NUL probe) |
| D4 | medium | Verif+Standards | `TestInitHelpListsFlags` | Help test didn't call `newInitFlagSet` | **fixed** in `cb077edf` |
| D5 | medium | Standards+Edge | `ingestExtra` WalkDir | `walkErr` / last saveErr swallowed | **fixed** in `cb077edf` |
| D6 | medium | Red | `ingestExtra` Abs | `--docs` Abs used cwd not target dir | **fixed** in `cb077edf` |
| D7 | medium | Edge+Red | Rel `"."` | `--docs` project root walks entire tree | **fixed** in `cb077edf` |
| D16 | low | Edge | `..foo` false jail | `HasPrefix("..")` vs `"../"` | **fixed** via `outsideJail` in `cb077edf` |
| D8 | medium | Standards | `initRunIndex` var | Package mutable func for test stub | **defer** |
| D9 | medium | Standards | `sentinelTemplate` | Inlined copy of `agent-config/block.md` | **defer** |
| D10 | medium | Perf | double Open + N+1 Save | Store open twice; per-file Save | **defer** |
| D11 | medium | Red | warm `indexed:0` | Success signal misleading | **defer** |
| D12 | medium | Red | boot before resolve | Boot written then open fails | **noise** (briefing: boot is the only fatal) |
| D13 | low | Red | flags after dir dropped | `flag.Parse` argv order | **defer** |
| D14 | low | Red | onboarding commit then init | Dirty tree after Step 8 | **defer** |
| D15 | maybe-false | Standards | CLAUDE-only writes CLAUDE.md | Matches blueprint Task 2 | **noise** |
| D17 | high | Red-team | `saveIngestFile` SaveInput | No `ExpiresAt` → default 7-day TTL | **defer** (store default; follow-up durable ingest) |
| D18 | medium | Red-team | `saveIngestFile` Owner | Owner = session UUID, visibility private | **defer** (governance default; search from later sessions) |
| D19 | medium | Red-team | hash-floor before topic_key | Identical content noops second topic_key | **defer** (AUDN floor; rare for distinct docs) |
| D20 | medium | Red-team | `OpenForDirectory` | `MNEMONIC_PROJECT` can split boot vs store | **defer** (existing resolve contract) |
| D21 | medium | Spec | spec dir | `acceptance.feature` / briefing / blueprint / tasks / adr still untracked | **fix now** (commit spec zone) |
| D22 | medium | Verif | several Then clauses | Weak assertions (counts, README skip, jail not-ingested) | **defer** (help + symlink + root covered in `cb077edf`) |

## Floor

- **Standards:** met-with-fixes (D8/D9 deferred)
- **Spec:** met-with-fixes (product scenarios hold; spec-zone commit pending as D21)
- **Security:** secure-with-fixes (D1/D2 closed in `cb077edf`)
- **Floor (decides):** met-with-fixes
- **Worst remaining:** D17 ingest TTL (deferred — not introduced as a new store default here)
- **Unfixed Important in-scope:** D21 only (commit the change's spec files)

## Verdict

**met-with-fixes** after `cb077edf`. Symlink Highs, size/binary bound, help production path, walk errors, cwd-relative `--docs`, and project-root walk are closed (`TestInit*` 29.8s ok). D21 closed (spec files tracked). D3/D6 regression tests in `9fa312fe`. Next: `skillgrid:ship`.
