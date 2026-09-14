# Applicability-Driven Threat Matrix (Skillgrid) — CANONICAL

> Canonical source. All skills link here. Do not fork content.

Load this only when the plan touches at least one of: routing, shell commands, subprocesses, version-control automation, PR automation, executable-file classification, process integration, **Mnemonic tool contracts** (`mem_*` / `code_*` / `web_cache_*`), or any `_shared/conventions/*` file.

Mark every row `Applicable` or explicit `N/A: reason`. Do not invent tests for `N/A` rows. Do not mark a row `N/A` on a guess — name the boundary you checked and why it is out of this change's scope.

**Applicable rows are plan requirements.** They MUST propagate into `acceptance.feature` as covering scenarios and into `tasks.md` as RED-test tasks ordered before their production tasks. An explicit `N/A` row requires no test and no scenario but MUST carry a reason a reviewer can challenge later.

## Core boundaries

| Boundary | Minimum adversarial cases | Applicability | Design response | Planned RED tests |
|---|---|---|---|---|
| Documentation-like paths | `requirements.txt`, `CMakeLists.txt`, executable Markdown/MDX, `README.sh` | Applicable / N/A: reason | Classification and execution boundary | One test per applicable class |
| Git repository selection | `git -C`, relative paths, absolute paths, worktree vs main checkout | Applicable / N/A: reason | Repository/cwd authority | One test per applicable selector |
| Commit state | staged, `commit -a`, empty index, `Co-authored-by` trailers | Applicable / N/A: reason | Index/worktree semantics | One test per applicable state |
| Push state | tracking branch, first push, explicit refspec, non-fast-forward | Applicable / N/A: reason | Destination/ref resolution | One test per applicable state |
| PR commands | explicit `--head`, environment prefix, composed commands | Applicable / N/A: reason | Argument composition and ownership | One test per applicable form |

## Skillgrid-specific boundaries

These rows are specific to Skillgrid's memory + conventions architecture. A convention a design author *forgot* to update is a cross-skill contract break that no code review will catch, because the code is locally consistent.

| Boundary | Minimum adversarial cases | Applicability | Design response | Planned RED tests |
|---|---|---|---|---|
| **Mnemonic tool surface** (`mem_*` / `code_*` / `web_cache_*`) | new tool, new required param, new return shape, new error code, new `scope` value | Applicable / N/A: reason | Explicit contract delta — which observation or chunk shape changes, and what existing callers must adapt. | One test per changed contract |
| **Shared-convention drift** | edit to any `_shared/conventions/*.md`, any `_shared/references/*.md`, any `agent-config/*.md` | Applicable / N/A: reason | One-line impact statement: "this file is now the source of truth for X — all skills that reference it are now bound to the new rule." Name every skill that breaks if it is missed. | A grep check that no skill references the old path or old rule; a link-resolution check |

## How to use this in a blueprint

- Include the applicable rows (and only the applicable rows) in the blueprint's `## Threat Matrix` section.
- Every applicable row must have a `Planned RED test` entry concrete enough to become a task in `tasks.md` and a scenario in `acceptance.feature` without guessing.
- **The handoff chain is blueprint (marks Applicable) → acceptance.feature (covering scenario) → tasks.md (RED-test task ordered before its production task).**
