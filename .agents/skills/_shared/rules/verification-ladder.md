# Verification Ladder (shared across all Skillgrid skills)

Four levels of verification, matched to change class. Higher class = higher verification floor.

## Levels

| Level | What | When |
|---|---|---|
| **L1** | Focused test command + exit code | Every change, every work unit |
| **L2** | Full test suite + build | Standard changes |
| **L3** | L2 + coverage report + lint/typecheck | Risky changes |
| **L4** | L3 + mutation testing + security scan (Trivy) | High-risk changes (auth, data migration, public API, money, concurrency) |

## Change Classification

| Class | Verification Floor | Criteria |
|---|---|---|
| `trivial` | L1 | Single concern, ≤3 files, no behavior change |
| `small` | L1 | Single file/concern, ≤10 files, no new capability |
| `standard` | L2 | Normal feature or bug fix |
| `risky` | L3 | New trust boundary, migration, multi-domain impact |
| `high-risk` | L4 | Auth, data migration, public API, money, concurrency |

## Work Unit Evidence (required at L1+)

Every work unit, regardless of class, MUST produce:

| Evidence | Required value |
|---|---|
| Focused test command + exact result | Smallest command proving this unit (command, exit/result, relevant counts) |
| Runtime harness command/scenario + exact result | Real integration/runtime path; explicit `N/A` + reason only if no runtime boundary exists |
| Rollback boundary | Exact files/behavior that can be reverted without removing unrelated work |

## Rules

- L1 is the floor — even a trivial change needs a focused test that passes.
- The `qa` skill enforces the appropriate level based on `quality:` thresholds in config.
- A change that cannot reach its verification floor is not done — it is blocked with a named gap.
