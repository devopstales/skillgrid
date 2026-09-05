# Applicability-Driven Threat Matrix (skillgrid)

Load this only when the plan touches at least one of: routing, shell commands, subprocesses, version-control automation, PR automation, executable-file classification, process integration, **Mnemonic tool contracts** (`mem_*` / `code_*` / `web_cache_*`), or any `_shared/conventions/*` file.

Mark every row `Applicable` or explicit `N/A: reason`. Do not invent tests for `N/A` rows. Do not mark a row `N/A` on a guess — name the boundary you checked and why it is out of this change's scope.

**Applicable rows are plan requirements.** They MUST propagate into `sdd-spec` as RED-test tasks in `tasks.md` and as scenarios in the owning step's `acceptance.feature`. An explicit `N/A` row requires no test and no scenario but MUST carry a reason a reviewer can challenge later.


## Core boundaries

| Boundary | Minimum adversarial cases | Applicability | Design response | Planned RED tests |
|---|---|---|---|---|
| Documentation-like paths | `requirements.txt`, `CMakeLists.txt`, executable Markdown/MDX, `README.sh` | Applicable / N/A: reason | Classification and execution boundary | One test per applicable class |
| Git repository selection | `git -C`, relative paths, absolute paths, worktree vs main checkout | Applicable / N/A: reason | Repository/cwd authority | One test per applicable selector |
| Commit state | staged, `commit -a`, empty index, `Co-authored-by` trailers | Applicable / N/A: reason | Index/worktree semantics | One test per applicable state |
| Push state | tracking branch, first push, explicit refspec, non-fast-forward | Applicable / N/A: reason | Destination/ref resolution | One test per applicable state |
| PR commands | explicit `--head`, environment prefix, composed commands | Applicable / N/A: reason | Argument composition and ownership | One test per applicable form |

## Skillgrid-specific boundaries

These two rows are specific to this project's memory + conventions architecture. They are the most dangerous kind of "N/A" — a convention a design author *forgot* to update is a cross-skill contract break that no code review will catch, because the code is locally consistent.

| Boundary | Minimum adversarial cases | Applicability | Design response | Planned RED tests |
|---|---|---|---|---|
| **Mnemonic tool surface** (`mem_*` / `code_*` / `web_cache_*`) | new tool, new required param, new return shape, new error code, new `scope` value, chunk-boundary shift in `code_read` | Applicable / N/A: reason | Explicit contract delta — which observation or chunk shape changes, and what existing callers must adapt. State in the plan's `## Mnemonic Integration` section. | One test per changed contract: a fixture `mem_save` that would 500 after the change, or a `code_read` where the chunk boundary moved |
| **Shared-convention drift** | edit to any `_shared/conventions/*.md`, any `_shared/issue-tracker/*.md`, any `_shared/agent-config/*.md` | Applicable / N/A: reason | One-line impact statement: "this file is now the source of truth for X — all sdd-* skills that reference it are now bound to the new rule." Name every skill that breaks if it is missed. | A grep test that no sdd-* skill references the old path or the old rule string; a link-resolution check that all inbound links still resolve |

## How to use this in a plan

- Include the applicable rows (and only the applicable rows) in the plan's `## Threat Matrix` section, with the `Applicable / N/A: reason` column filled.
- Every applicable row must have a `Planned RED test` entry concrete enough to become a task in `tasks.md` and a scenario in `acceptance.feature` without guessing. "A test for the edge case" is not a test. "Given `git -C` to a worktree, expect the command to fail with `not a git repository` before touching the index" is a test.
- **The handoff chain is change.md → sdd-spec (`tasks.md` + `acceptance.feature`).** The change names each applicable row's RED test and the owning step; **`sdd-spec` turns each into a RED-test task ordered before its production task and a scenario in the owning step's Feature**. A change that loses an applicable row at any boundary was not a plan — it was a sketch.
