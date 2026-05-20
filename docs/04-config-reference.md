# Configuration reference (`.skillgrid/config.json`)

The project harness reads **`.skillgrid/config.json`** to align SDD phases, PRD board columns, quality gates, and optional tooling. The file is created or merged during **`/sdd-init`**; you can edit it by hand afterward.

**Consumers:** phase skills and workflows (agents), `skillgrid serve` (dashboard lanes and workflow order), and optional shell helpers. Not every key is enforced by `sdd-gate.sh` yet — see [Enforcement](#enforcement) below.

## Quick map

| Section | Purpose |
| --- | --- |
| `artifactStore` | Where durable artifacts prefer to live (`disk`, `memory`, `hybrid`) |
| `vdd` | Verification-Driven Development refinement loop (Beads path, iteration caps) |
| `prdWorkflow` | PRD status columns and phase → status mapping for the board |
| `workflow` | Dashboard / workflow view phase order |
| `tdd` | Test-driven development policy and evidence retention |
| `apply` | How `/sdd-apply` runs (sequential agent, TDD, resume) |
| `tasks` | Granularity and context-budget rules for task breakdown |
| `executor` | Subagent executor defaults (retries, parallel slices) |
| `verify` | Spec compliance gate and security classification |
| `review` | Stage 2 review (Trivy, TrueCourse, iterations) |
| `pre_merge` | Pre-archive / merge readiness checks |
| `archive` | Post-verify archive disposition |
| `parallel` | Optional parallel slice limits |
| `skills` | Auto-invoked and mandatory skills around apply/review |
| `diagnose` | `/sdd-diagnose` protocol |

## `artifactStore`

```json
{ "artifactStore": { "mode": "hybrid" } }
```

| Key | Values | Effect |
| --- | --- | --- |
| `mode` | `disk` \| `memory` \| `hybrid` | **hybrid** (default): repo files are canonical; concise summaries may go to Engram when enabled. **disk**: files only. **memory**: prefer memory backend where skills support it. |

Referenced by `/sdd-review` and artifact-handoff skills.

## `vdd`

```json
{
  "vdd": {
    "chainlink_enabled": true,
    "beads_path": ".beads",
    "max_refinement_iterations": 10,
    "hallucination_threshold": 0.7
  }
}
```

| Key | Default | Effect |
| --- | --- | --- |
| `chainlink_enabled` | `true` | Enable VDD chain-link refinement between verify and review |
| `beads_path` | `.beads` | Local Beads store when VDD/Beads sync is used |
| `max_refinement_iterations` | `10` | Cap on adversarial refinement loops |
| `hallucination_threshold` | `0.7` | Convergence threshold for `vdd-converge` (lower = stricter) |

## `prdWorkflow`

Drives PRD **status lanes** on the web UI and logical phase labels. `skillgrid serve` reads `statuses`, `fallbackStatus`, and `phaseStatusMap`.

```json
{
  "prdWorkflow": {
    "source": "preset",
    "preset": "skillgrid-default",
    "fallbackStatus": "draft",
    "statuses": [
      { "id": "draft", "label": "Draft" },
      { "id": "todo", "label": "Todo" }
    ],
    "phaseStatusMap": {
      "plan": "draft",
      "breakdown": "todo",
      "apply": "inprogress",
      "validate": "devdone",
      "finish": "archived"
    },
    "providerMapping": {}
  }
}
```

| Key | Effect |
| --- | --- |
| `source` | `preset`, `provider`, `import`, or `custom` — how statuses were chosen at init |
| `preset` | Named preset (e.g. `skillgrid-default`) |
| `fallbackStatus` | Status id when phase mapping is missing |
| `statuses` | Ordered board columns (`id` + `label`) |
| `phaseStatusMap` | Maps harness phases (`plan`, `breakdown`, `apply`, `validate`, `finish`) to a status `id` |
| `providerMapping` | Optional remote provider column ids (GitHub/GitLab/Jira) |

**Note:** Some workflows still mention top-level `beads_enable`; the hub default uses **`vdd.chainlink_enabled`** and Beads under `.beads`. Prefer `vdd` for new projects.

## `workflow`

```json
{
  "workflow": {
    "phaseOrder": [
      "brainstorm", "design", "plan", "breakdown", "apply",
      "test", "security", "validate", "finish"
    ]
  }
}
```

Controls **Workflow** view ordering in the dashboard ([Web UI](18-webui.md)). Events remain authoritative; this only affects lane display and empty-state order.

## `tdd`

```json
{
  "tdd": {
    "enforcement": "strict",
    "allow_hitl_tests": true,
    "evidence_retention_days": 30,
    "auto_cleanup_on_violation": true
  }
}
```

| Key | Effect |
| --- | --- |
| `enforcement` | `strict` \| `advisory` — whether `enforced-tdd-protocol` is mandatory |
| `allow_hitl_tests` | Allow human-in-the-loop test tasks without AFK apply |
| `evidence_retention_days` | How long TDD violation evidence is kept |
| `auto_cleanup_on_violation` | Remove stale evidence after violations |

Works with `skills.mandatory` / `skills.auto_invoke` (see below).

## `apply`

```json
{
  "apply": {
    "execution_mode": "sequential_agent",
    "tdd_enforcement": true,
    "resume_enabled": true
  }
}
```

| Key | Effect |
| --- | --- |
| `execution_mode` | `sequential_agent` — one fresh subagent per task via `sequential-agent-executor` |
| `tdd_enforcement` | Apply phase must respect TDD when true |
| `resume_enabled` | Allow resuming partial apply from handoff/checkpoints |

## `tasks`

Granular planning constraints for `sdd-tasks` / `granular-planning`:

| Key | Default | Effect |
| --- | --- | --- |
| `granularity_min_minutes` | `2` | Minimum estimated task size |
| `granularity_max_minutes` | `5` | Maximum estimated task size |
| `require_complete_code` | `true` | Tasks must include full code snippets, not placeholders |
| `max_files_per_task` | `3` | Cap files touched per atomic task |
| `context_budget_threshold` | `5` | Soft limit for context-heavy tasks (also overridable as `contextBudgetThreshold` in some skills) |

## `executor`

| Key | Default | Effect |
| --- | --- | --- |
| `default_model_tier` | `fast` | Default subagent model tier |
| `max_retries_per_task` | `2` | Retries per failed task |
| `stop_on_first_failure` | `false` | Halt entire apply run on first failure |
| `enable_final_integration_review` | `true` | Run integration review after all tasks |
| `parallel_slices` | `false` | When true, allows parallel slice execution (see `parallel` section) |

## `verify` (Stage 1)

```json
{
  "verify": {
    "require_full_coverage": true,
    "allow_partial": false,
    "auto_evidence_collection": true,
    "security": {
      "classify_sensitive": true,
      "require_heimdall_when_sensitive": true,
      "advisory_scan": false
    }
  }
}
```

| Key | Effect |
| --- | --- |
| `require_full_coverage` | Every spec requirement must have evidence |
| `allow_partial` | Allow PASS with explicit partial scope |
| `auto_evidence_collection` | Collect test/lint evidence automatically |
| `security.classify_sensitive` | Run `classify-security-sensitive.sh` |
| `security.require_heimdall_when_sensitive` | Require Heimdall persona report when classified sensitive |
| `security.advisory_scan` | Security scan is advisory only when true |

Artifact paths: [Review and verify artifacts](16-review-artifacts.md).

## `review` (Stage 2)

```json
{
  "review": {
    "required": true,
    "allow_auto_approve": false,
    "max_iterations": 3,
    "require_two_reviewers": false,
    "auto_assign_persona": null,
    "security": {
      "trivy_scan": true,
      "trivy_severity": ["CRITICAL", "HIGH", "MEDIUM"],
      "fail_on_cve": true,
      "fail_on_secret": true,
      "fallback_scan": true
    },
    "architecture": {
      "truecourse_enabled": false,
      "truecourse_mode": "diff",
      "truecourse_llm": false,
      "fail_on_new_violations": true,
      "min_severity_to_fail": "high"
    }
  }
}
```

| Key | Effect |
| --- | --- |
| `required` | `/sdd-review` must run before archive |
| `allow_auto_approve` | Coordinator may auto-approve (default false) |
| `max_iterations` | Review fix loops before escalation |
| `require_two_reviewers` | Dual review when true |
| `auto_assign_persona` | Optional persona id for auto-dispatch |
| `security.trivy_scan` | Run Trivy MCP / `trivy-security` skill |
| `security.trivy_severity` | Severities reported |
| `security.fail_on_cve` / `fail_on_secret` | Hard-fail gates |
| `security.fallback_scan` | Use `vulnerability-scanner` when Trivy unavailable |
| `architecture.truecourse_enabled` | Run TrueCourse on diff |
| `architecture.truecourse_mode` | `diff` or full scan mode |
| `architecture.fail_on_new_violations` | Fail on new architecture violations |
| `architecture.min_severity_to_fail` | Minimum violation severity to fail |

## `pre_merge`

Final gate before merge/archive orchestration:

| Key | Default | Effect |
| --- | --- | --- |
| `require_spec_compliance` | `true` | Spec traceability required |
| `require_code_review` | `true` | Code quality review required |
| `require_tests_green` | `true` | Tests must pass |
| `require_lint_clean` | `false` | Lint must be clean when true |
| `allow_uncommitted_changes` | `false` | Block finish with dirty tree |
| `auto_merge_on_pass` | `false` | Never auto-merge by default |
| `security_scan` | `true` | Security scan in pre-merge bundle |

Used by `pre-merge-verification` skill.

## `archive`

| Key | Effect |
| --- | --- |
| `default_disposition` | `prompt` \| `merge` \| `keep` — what to do after verify+review |
| `auto_merge_after_verify` | Auto-merge when verify passes (usually false; review still required) |
| `keep_verification_reports` | Retain verify/review markdown under the change |
| `retention_policy_days` | Days to keep archived verification artifacts |

## `parallel`

| Key | Effect |
| --- | --- |
| `slice_independence_check` | Verify slices do not share mutable state |
| `max_parallel_slices` | Max concurrent slices when `executor.parallel_slices` is true |

Skillgrid’s **default** execution model remains linear single-clone — see [Skillgrid logic](03-skillgrid-logic.md).

## `skills`

```json
{
  "skills": {
    "auto_invoke": ["enforced-tdd-protocol"],
    "mandatory": ["enforced-tdd-protocol"],
    "pre_review_checklist": true,
    "post_review_integration": true
  }
}
```

| Key | Effect |
| --- | --- |
| `auto_invoke` | Skills loaded automatically in apply/review |
| `mandatory` | Skills that must complete successfully |
| `pre_review_checklist` | Run checklist before Stage 2 review |
| `post_review_integration` | Integration pass after review fixes |

## `diagnose`

| Key | Effect |
| --- | --- |
| `protocol` | `four_phase` — reproduce → minimise → fix → regression |
| `auto_escalate_after_fixes` | Escalate to human after N fix attempts |
| `require_evidence_first` | No fix without reproduction evidence |

## Enforcement

| Layer | Reads `config.json`? |
| --- | --- |
| Phase skills (`sdd-verify`, `sdd-review`, `sdd-apply`, …) | **Yes** — agents instructed to read relevant sections |
| `skillgrid serve` | **Yes** — `prdWorkflow`, `workflow.phaseOrder` |
| `sdd-gate.sh` | **Partial** — verify/review/archive gates; not all `review.security` keys yet ([TODO](TODO.md)) |
| Git hooks | Indirect — via gate script |

When changing gates, update config **and** confirm the skill or script that enforces it.

## Example: stricter security review

```json
{
  "review": {
    "required": true,
    "security": {
      "trivy_scan": true,
      "trivy_severity": ["CRITICAL", "HIGH"],
      "fail_on_cve": true,
      "fail_on_secret": true,
      "fallback_scan": true
    },
    "architecture": {
      "truecourse_enabled": true,
      "truecourse_mode": "diff",
      "fail_on_new_violations": true,
      "min_severity_to_fail": "medium"
    }
  },
  "verify": {
    "security": {
      "classify_sensitive": true,
      "require_heimdall_when_sensitive": true
    }
  }
}
```

## Related docs

- [Workflow usage](02-workflow-usage.md) — init choices and day-to-day commands
- [Skillgrid logic](03-skillgrid-logic.md) — PRD/INDEX/OpenSpec hierarchy
- [Commands reference](05-commands-reference.md) — slash commands per phase
- [Review artifacts](16-review-artifacts.md) — verify/review output paths
- [Web UI](18-webui.md) — dashboard use of `workflow` and `prdWorkflow`
- [MCP servers](13-mcp-servers.md) — Trivy and optional scanners
