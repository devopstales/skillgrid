# Review — mnemonic-project-init

> Change: `.skillgrid/specs/2026-10-02-mnemonic-project-init/`
> Generated: 2026-10-03T07:50:00Z (parallel-code-review)
> Diff range: `f739d371..17652346`
> Rigor tier: T2 (escalated to fan-out: 1115 lines ≥ 50)
> Independence: Grade A all axes (fresh subagents)
>
> Specialists: Standards, Spec, Edge, Verification-gap, Security, Performance, Red-team (last).
> Accessibility skipped (no UI).

## What Important means

- **Critical** — must fix before merge (security, data loss, broken SATISFIES).
- **Important** — should fix before merge (ADR/SSOT hard breach, missing requirement, scope creep).
- **Minor** — nits; capped at 5 listed per axis.

## Independence

| Axis | Grade |
|------|-------|
| Standards | A |
| Spec | A |
| Edge | A |
| Verification-gap | A |
| Security | A |
| Performance | A |
| Red-team | A |

## Per-specialist counts

| Specialist | Count | Worst |
|------------|-------|-------|
| Standards | 5 Important + 3 Minor | agent-config SSOT / mutable `initRunIndex` / swallowed walkErr |
| Spec | scenarios PASS; notes on help-test proxy | help test doesn't call production |
| Edge | symlink jail, Rel `.`, `..foo` false reject, symlink dirs | symlink jail bypass |
| Verification-gap | help/production disconnect; onboarding source-text | broken-verification on help |
| Security | 5 (3 High symlink) | symlink → host read/write |
| Performance | 4 | unbounded WalkDir / N+1 Save / double open / warm Scan |
| Red-team | 7 | cwd Abs, flag order, Project `.`, boot-before-resolve |

## Deduped entries (coordinator verdicts)

| ID | Verdict | Source | Location | Finding | Route |
|----|---------|--------|----------|---------|-------|
| D1 | high | Security+Edge | `init_ingest.go` jail / Stat | Symlink escapes lexical `--docs`/default jail into host files | **fix now** |
| D2 | high | Security | `init_boot.go` WriteFile | Symlink boot target overwrites host file | **fix now** |
| D3 | medium | Security+Perf | `saveIngestFile` ReadFile | No size bound / binary docs blob | **fix now** (size+skip binary) |
| D4 | medium | Verif+Standards | `TestInitHelpListsFlags` | Help test doesn't call `newInitFlagSet` | **fix now** |
| D5 | medium | Standards | `ingestExtra` WalkDir | `walkErr` swallowed | **fix now** |
| D6 | medium | Red | `ingestExtra` Abs | `--docs` Abs uses cwd not target dir | **fix now** |
| D7 | medium | Edge+Red | Rel `"."` | `--docs` project root walks entire tree | **fix now** |
| D8 | medium | Standards | `initRunIndex` var | Package mutable func for test stub | **defer** (test seam; inject later) |
| D9 | medium | Standards | `renderSentinel` / template | Hardcoded tracker/memory; duplicated `block.md` | **defer** (needs config fill; follow-up) |
| D10 | medium | Perf | double Open + N+1 Save | Store open twice; per-file Save | **defer** |
| D11 | medium | Red | warm `indexed:0` | Success signal misleading | **defer** |
| D12 | medium | Red | boot before resolve | Boot written then open fails | **defer** (matches briefing: boot fatal only) |
| D13 | low | Red | flags after dir dropped | `flag.Parse` argv order | **defer** (document / follow-up) |
| D14 | low | Red | onboarding commit then init | Dirty tree after Step 8 | **defer** (skill order) |
| D15 | maybe-false | Standards | CLAUDE-only → write CLAUDE | Contradicts Gotchas but matches blueprint Task 2 | **noise** (blueprint wins) |
| D16 | low | Edge | `..foo` false jail | HasPrefix `../` vs `..` | **fix now** (cheap) |

## Floor

**not met** until D1–D7 + D16 fixed (symlink Highs block ship). Re-render after fix wave.

## Verdict

**BACK-TO-APPLY** — fix now: symlink Lstat/EvalSymlinks jail + boot refuse-symlink; size/type guard; production help test; walkErr; Abs relative to dir; reject Rel `.`; fix `..` prefix check.
