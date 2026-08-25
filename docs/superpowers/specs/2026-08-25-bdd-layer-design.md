# 2026-08-25 aiskillgrid BDD Layer Design

> **STATUS: DRAFT (2026-08-25)** — awaiting decision on spec-management approach (A vs B). Shared core is final; the open decision is boxed below.

## Goal

Extend aiskillgrid so that `install` distributes a **Behavior-Driven Development layer** to other projects, based on the pattern in [intent-driven-dev/behavior-driven-template](https://github.com/intent-driven-dev/behavior-driven-template) and delivered as **superpowers-style skills**.

The layer gives any project the template's three properties, enforced mechanically rather than by prompt discipline:

1. **Intent articulated in the spec** — business use cases written as Gherkin inside Markdown specs; those scenarios run as the acceptance suite that every change must keep green.
2. **Spec as source** — code is never modified without a driving spec change; red scenarios define the work.
3. **Agent containment** — harness rules (acceptance suite always green; specs and code never modified together) keep agents from reward-hacking "done" or silently editing the spec to pass tests.

Scope decisions (already made):

- Applies to **other projects** (distribute via aiskillgrid); aiskillgrid itself stays a plain Go CLI.
- Runner stack is **Node / Cucumber.js** (+ `gherkin-lint`) — reuses the template's proven machinery and our npm-based installer.
- Spec management is **open**: Option A (OpenSpec CLI) or Option B (superpowers skill) — this doc proposes both.

## Source Material (what the template actually is)

Verified from the repo:

| Piece | Template implementation |
|-------|------------------------|
| Specs | `openspec/specs/<capability>/spec.md` + change deltas under `openspec/changes/*/specs/` — Markdown with Gherkin in column-0 fenced blocks; line-preserving extraction so lint/test line numbers map 1:1 to `spec.md` |
| Workflow | OpenSpec behavior-driven schema: proposal → specs → design → tasks, with **acceptance tests first** in the generated task list |
| Runner | independent `acceptance-tests/` Node project: `extract-gherkin.cjs` (Markdown → `.feature`), `cucumber.cjs` (effective-spec composition, source-of-truth vs active deltas, archive exclusion, HTML report), `gherkin-lint` gate, Page Object Model step defs |
| Rule 1 | acceptance tests must always pass (definition of done) |
| Rule 2 | specs and code never modified together — enforced by a **zone-guard hook**, not prompt discipline |
| Tooling notes | `gherkin-lint` needs explicit `--config` (no default rules); extraction output dir is gitignored and written via Bash (zone guard intercepts file edits only) |

## Shared Core (final — both options include this)

Distributed to each project by `aiskillgrid install`:

| Piece | Lands at | Source of truth in aiskillgrid |
|-------|----------|--------------------------------|
| Gherkin-in-Markdown spec layout | `specs/<capability>/spec.md` (+ `specs/archive/`) | `config.d/bdd/spec-template.md` |
| Gherkin extraction script | `acceptance-tests/extract-gherkin.cjs` | `config.d/bdd/extract-gherkin.cjs` (copied from template, adapted) |
| `gherkin-lint` config + gate command | `acceptance-tests/gherkin-lintrc.json` | `config.d/bdd/gherkin-lintrc.json` |
| Cucumber.js runner scaffold | `acceptance-tests/` (own `package.json`, `cucumber.cjs`, `steps/`, `support/`) | `config.d/bdd/acceptance-tests/` scaffold |
| Skills (see options) | via existing `skills.yaml` pipeline (step 6) | `config.d/skills.yaml` additions + `config.d/skills/` |
| BDD rules | merged into the project rules file + `AGENTS.md` registration | `config.d/bdd/rules.md` (Rule 1 + Rule 2 + lint gate + acceptance-first) |
| Zone-guard hook | hook script wired into the agent's hook system (kilo: `--hooks`, opencode: plugin) | `config.d/bdd/zone-guard.sh` |

Install behavior constraints (from plan/spec):

- All BDD steps **warn and continue** on failure — the rest of the install never aborts because of BDD.
- Idempotent: re-running never duplicates rules/skills; scaffolds are created only if missing (user projects own their `acceptance-tests/` after first scaffold).
- Backups before any edit of agent configs, as with today's MCP/rules steps.

## The Open Decision: Spec Management Approach

### Option A — OpenSpec CLI (`opsx`) distributed by aiskillgrid

aiskillgrid adds `@fission-ai/openspec` to `config.d/tools.yaml` (so the existing `npm install --prefix ~/.aiskillgrid` step covers it), and the BDD skills tell agents to drive changes with `opsx propose` / `opsx apply` / `opsx archive`. Each project gets `openspec/config.yaml` pointing at the behavior-driven schema; `specs/` becomes `openspec/specs/`.

What the skill set must teach:

1. `opsx propose <change>` → proposal/spec/delta specs with fenced Gherkin → lint the extracted Gherkin → commit spec **before** any code.
2. `opsx apply <change>` → work the generated task list (acceptance setup is task 1) → implement until the acceptance suite is green → `opsx archive`.
3. Zone-guard hook blocks commits where `openspec/**` and implementation files change together.

- **Pros:** exactly the template's workflow; proposal/spec/design/tasks generation for free; delta + archive semantics built in; the schema mechanically enforces acceptance-first in generated tasks.
- **Cons:** one more external CLI in the install surface; `opsx` runs in the project workspace; the agent learns two vocabularies (superpowers skills + opsx commands); aiskillgrid inherits OpenSpec's release cadence.
- **Extra install work:** `tools.yaml` entry, `openspec/config.yaml` template, skills that reference `opsx` commands.

### Option B — Superpowers skill `bdd-workflow` (no OpenSpec)

A single superpowers-style skill (authored alongside the existing `config.d/skills/` pattern, installable through the existing `skills.yaml` pipeline) encodes the *entire* workflow as agent behavior over plain files:

1. **Spec-delta phase** — create/modify `specs/<capability>/spec.md` with fenced Gherkin; extract + `gherkin-lint` (zero errors required); commit spec alone (zone-guard enforces).
2. **Acceptance phase** — ensure `acceptance-tests/` scaffolds exist; add/adjust step definitions so the suite runs the new scenarios; run suite to confirm the *intended* red (unimplemented behavior, not plumbing).
3. **Implementation phase** — TDD skills (already distributed) drive inside-out work; suite must end green.
4. **Archive phase** — promote delta content into the source-of-truth spec, move delta to `specs/archive/`.

Change lifecycle (propose/apply/archive) is simulated with a `specs/changes/<name>/{spec-delta.md,tasks.md}` convention + `specs/archive/` — the same semantics as OpenSpec deltas, implemented as files + skill instructions.

- **Pros:** zero dependency beyond Cucumber.js; everything fits the superpowers model (skill → agent behavior); works in any agent because it's instructions + files; aiskillgrid stays the sole installer; the template's own docs concede "OpenSpec is only one example."
- **Cons:** we own the workflow definition (propose/apply semantics, task generation) — more to design and maintain; no tool-generated task list (skill must make the agent write acceptance-first `tasks.md`); if the skill's instructions are weaker than the schema's, agents drift and only the zone-guard catches it.
- **Extra install work:** skill repo(s) in `skills.yaml`, `config.d/bdd/` templates, zone-guard hook, rules text.

### Recommendation

**Option B** — self-contained, matches the "based on superpowers" requirement, and the shared core is identical either way, so switching A → B (or B → A) later only touches the spec-management layer, not specs, runners, or rules. Choose A only if you want tool-generated task lists and are willing to carry the opsx dependency.

## Component Breakdown (both options)

```
aiskillgrid install
├── step 6 (skills)     + bdd skills (spec-authoring, acceptance-test-authoring, bdd-workflow)
├── NEW step 6b (bdd)   + scaffold specs/, acceptance-tests/ (if missing)
│                        + copy extract-gherkin.cjs, gherkin-lintrc.json, zone-guard.sh
│                        + merge BDD rules into project rules file + AGENTS.md registration
└── (agent configs)      + hook wiring: kilo --hooks / opencode plugin for zone-guard
```

- **Unit: `config.d/bdd/`** — the distributable. Testable by running install against a temp project HOME (extend the existing smoke test pattern) and asserting file presence + valid JSONC + lintable Gherkin.
- **Unit: skills** — validated by the existing skills pipeline; behavior verified by the template's acceptance test ("Let's make a react todo list"-style trigger check).
- **Unit: zone-guard hook** — pure shell script; unit-testable by staging a git commit touching both zones and asserting exit 1.

## Error Handling

- Scaffold copy fails → warn, continue (project can add scaffolds manually).
- `gherkin-lint` not resolvable (no node) → warn; rule 1/2 still installed.
- Hook wiring fails for one agent → warn; other agents proceed.
- Existing `specs/` or `acceptance-tests/` in target project → **never overwrite**; create-only-if-missing, then report what's already there.

## Testing

1. Unit: scaffold copy + rules merge (temp HOME smoke test, same pattern as existing `internal/smoke`).
2. Unit: zone-guard hook allow/deny matrix (both-zone commit → deny; single-zone → allow; `tasks.md` exempt).
3. Integration: install into a scratch Node project, add a 2-scenario spec, confirm extract → lint → cucumber run produces intended-red then green after stub implementation.
4. Dry-run: `install --dry-run` lists planned BDD artifacts without writing them.

## Open Questions

1. **A or B** — the decision this doc exists to support.
2. Hook mechanism per agent: kilo hook config vs opencode plugin file — needs a spike per agent before implementation (bounded work once A/B is chosen).
3. Does the zone-guard scope to the target project's git repo, or to the session? (Template: per-commit; default per-commit.)

## Non-Goals

- No Windows hooks support in v1 (consistent with CLI plan).
- No CI integration of the acceptance suite (project's own CI owns that).
- No BDD dogfooding of the aiskillgrid Go CLI itself (separate effort).
- No Go/Python runner adapters in v1 (Node/Cucumber.js only; design keeps specs stack-agnostic so adapters are additive).
