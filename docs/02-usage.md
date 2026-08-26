# Usage

Once built, the CLI is intentionally small: two commands, a handful of flags. The value is in what it does for you — reproducing the environment, not re-remembering it.

## Commands

| Command | Alias | Purpose |
|---------|-------|---------|
| `install` | `in` | Run the full install flow |
| `sync-repo` | — | Copy a local checkout into `~/.skillgrid/repos/skillgrid` (plus `config.d`) without running the rest |
| `index` | — | Incremental code index for the cwd git root (respects `config.d/indexing.yaml`) |
| `index --status` | — | Print index stats (same as MCP `code_status`) |
| `mcp` | — | Stdio MCP server (`mem_*`, `code_*`, `web_*` tools) |
| `serve` | — | HTTP API for OpenCode/Kilo mnemonic plugins |
| `help` | — | Print usage |

Flags parse the same before or after the command name: `install --dry-run` and `--dry-run install` both work.

## Flags

| Flag | Applies to | Effect |
|------|-----------|--------|
| `--skip-clone` | install | Skip the clone step; use existing `~/.skillgrid` state |
| `--sync-repo <path>` | install, sync-repo | Sync a local repo into `~/.skillgrid/repos/skillgrid` |
| `--dry-run` | install | Print planned changes; no npm installs, no MCP/rules writes, no backups |
| `--verbose` | install | Print detailed MCP entries instead of one-line summaries |
| `--yes` | install | Skip the interactive agent selector (defaults to all agents) |

Environment: `SKILLGRID_REPO_URL` overrides the clone source.

## Typical Day-Loops

Reconcile the environment on this machine:

```bash
./bin/skillgrid install
```

Change to `config.d/` and preview the diff before committing:

```bash
./bin/skillgrid install --dry-run
```

Develop against a local checkout (fastest iteration loop — no network clone needed):

```bash
./bin/skillgrid install --sync-repo $(pwd)
```

See exactly what would land in each agent config:

```bash
./bin/skillgrid install --sync-repo $(pwd) --verbose
```

## PATH Setup

After a successful install, the CLI prints the two lines to add to your shell rc:

```bash
export PATH="$HOME/.skillgrid/bin:$PATH"
export PATH="$HOME/.skillgrid/npm/node_modules/.bin:$PATH"
```

- `~/.skillgrid/bin` — the `engram` binary
- `~/.skillgrid/npm/node_modules/.bin` — all agent CLIs and tools (kilo, opencode, skills, playwright, …) once `npm --prefix` installed them

Once in your rc file, you can invoke `kilo`, `opencode`, `engram`, `skills`, etc. directly.

## Logs

Every run appends to `~/.skillgrid/logs/install.log` at INFO/WARN/ERROR. If the terminal is unclear about what failed, read the log:

```bash
tail -50 ~/.skillgrid/logs/install.log
```

## Safety Model

- Every edit to `~/.config/kilo/kilo.jsonc` or `~/.config/opencode/opencode.jsonc` is preceded by a timestamped backup under `~/.skillgrid/backups/`.
- Backups keep the last 10 per file; older ones are pruned.
- The merge is JSON-aware and idempotent — running install twice does not duplicate keys.
- The `--dry-run` flag guarantees zero writes (no npm, no config edits, no backups).

## Mnemonic (memory + code index + web cache)

When `config.d/indexing.yaml` sets `profile: mnemonic`, the same `skillgrid` binary exposes MCP tools for session memory, repository text search, and cached web research. Agent rule:

> `mem_*` = decisions/history; `code_*` = repo text search; **`web_*` = cache Context7/Exa/DeepWiki/fetch before re-querying**; GitNexus opt-in for impact graphs.

### Index lifecycle

1. After clone (or when opening a repo for the first time), run `skillgrid index` from the repo root.
2. Re-run `skillgrid index` when the codebase changes materially or `code_status` reports stale stats.
3. During chat, agents call `code_status` before grepping large unknown areas; use `code_search` → `code_read` after the index is warm.

```bash
# first-time or refresh
skillgrid index

# check stats without re-indexing
skillgrid index --status
```

Data lives under `~/.skillgrid/mnemonic/` (override with `SKILLGRID_MNEMONIC_DATA_DIR`). Index settings merge from the nearest `config.d/indexing.yaml` walking up from cwd.

## Development Workflows (SDD, IDD, BDD, TDD)

skillgrid augments [superpowers](https://github.com/obra/superpowers) with **Spec-Driven Development (SDD)**, **Intent-Driven Development (IDD)**, optional **Behavior-Driven Development (BDD)**, and **Test-Driven Development (TDD)** during implementation. All agent-facing artifacts live under `docs/`; application code stays outside it.

After `skillgrid install`, agents load superpowers at session start and discover IDD/BDD skills from `config.d/skills/` via `skills.paths`. See [05-skills](05-skills.md) and [07-plugins](07-plugins.md) for install mechanics.

**Design reference:** [2026-08-26-idd-bdd-design.md](superpowers/specs/2026-08-26-idd-bdd-design.md)

**Methodology references:**

- [How TDD and BDD Actually Fit Into Spec-Driven Development](https://intent-driven.dev/blog/2026/08/23/tdd-bdd-spec-driven-development/) — macro BDD + micro TDD layering, strict TDD anti–reward-hacking
- [SDD with Multi-Model Spec Review and Glossary](https://intent-driven.dev/blog/2026/06/27/sdd-adversarial-authoring-glossary/) — glossary + adversarial spec review before human approval

### Layer model: SDD → BDD → TDD

From [intent-driven.dev](https://intent-driven.dev/blog/2026/08/23/tdd-bdd-spec-driven-development/), the three workflows operate at different layers — they complement each other; none replaces another:

| Layer | Workflow | Question it answers | Feedback loop |
|-------|----------|---------------------|---------------|
| **Spec** | SDD / IDD | What must be true? | Design + plan before code |
| **Macro** | BDD | Does the system behave correctly from the outside? | Gherkin → Cucumber acceptance (red → green) |
| **Micro** | TDD (Mockist) | Are internal units decoupled with clear contracts? | One failing test → minimal code → refactor |

```
SDD/IDD     specification is source of truth (docs/superpowers/)
    ↓
BDD         user-facing acceptance scenarios guard observable behavior
    ↓
TDD         object-level tests drive narrow contracts and collaboration boundaries
```

**Why you need all three when quality matters:** SDD + BDD alone can satisfy every acceptance scenario while the implementation stays a tangle of coupled units — macro scenarios do not force clear object boundaries. TDD supplies the micro loop that keeps units decoupled as behavior grows. See the [layer diagram in the post](https://intent-driven.dev/blog/2026/08/23/tdd-bdd-spec-driven-development/).

**Mnemonic:** SDD defines what must be true → BDD verifies observable behavior → TDD shapes how the code is structured.

### How they relate

| Workflow | Focus | When to use |
|----------|-------|-------------|
| **SDD** | Spec and plan before code | Default superpowers flow — any non-trivial change |
| **IDD** | Intent, decisions, and durable ADRs | Features that need explicit *why*, cross-cutting decisions, or audit trail |
| **BDD** | Executable acceptance scenarios | User-visible behavior you want Gherkin + Cucumber to guard |
| **TDD** | Unit/integration red → green → refactor | Every apply step — new features, bug fixes, behavior changes |

They stack — **TDD runs inside apply**; **BDD wraps apply** when enabled:

```
SDD     brainstorm → design → glossary → plan → [apply + strict TDD] → verify → commit
IDD     proposal → [adversarial] → design → glossary → adr → plan → [apply + strict TDD] → promote
BDD     (after design) gherkin → extract → red (macro) → [apply + strict TDD (micro)] → green → promote
TDD     (each behavior) red test → run fail → green code → run pass → refactor → commit
```

BDD does not replace TDD. Both use red → green at different layers ([macro vs micro](https://intent-driven.dev/blog/2026/08/23/tdd-bdd-spec-driven-development/)).

### Spec quality before apply

Before human review, tighten specs with these skills ([source](https://intent-driven.dev/blog/2026/06/27/sdd-adversarial-authoring-glossary/)):

| Skill | Phase | Purpose | Output |
|-------|-------|---------|--------|
| `grilling` | proposal / plan | Interrogate the plan with your human partner — one question at a time | refined proposal or plan |
| `glossary` | design | Define domain + technical terms once; agents reuse them in every later spec | `docs/glossary/{business,technical}.md` |
| `adversarial-authoring` | proposal (optional) | Author sub-agent writes; reviewer sub-agent challenges; council notes capture resolutions | proposal + review trail |

**Glossary example:** an agent might coin _"translation boundary service"_ for an import layer; your team calls it **anti-corruption layer** (ACL). Define ACL once in `docs/glossary/technical.md` — every subsequent spec uses the same term ([DDD context mapping](https://intent-driven.dev/blog/2026/06/27/sdd-adversarial-authoring-glossary/)).

**Adversarial authoring:** use when you have access to two models (e.g. one writes proposal, another reviews). Keep to two or three sub-agents — more becomes noise. Produces **council notes**: what was written, what was challenged, and whether each challenge was accepted or rejected. Not vendored in skillgrid v1; use `grilling` + human review as the default.

**Combined with `grill-me` / `grilling`:** adversarial authoring reduces model bias; glossary enforces terminological consistency; grilling stress-tests the plan before build. The spec that reaches human review should need less cleanup.

### Command order by workflow type

Use these tables as the canonical sequence. **Stop at human approval gates** before crossing from `docs/` to application code.

#### SDD — command order

| Step | Invoke | Zone | Action / command | Output |
|------|--------|------|------------------|--------|
| 1 | `brainstorming` | `docs/` | Explore intent; agree on scope | (dialogue) |
| 2 | `brainstorming` or direct write | `docs/` | Write design spec | `docs/superpowers/specs/YYYY-MM-DD-<topic>-design.md` |
| 3 | `glossary` | `docs/` | Define or update terms used in the design | `docs/glossary/{business,technical}.md` |
| 4 | — | — | **Human approves design** | — |
| 5 | `grilling` (optional) | `docs/` | Stress-test the plan before build | refined plan |
| 6 | `writing-plans` | `docs/` | Break design into checkbox tasks | `docs/superpowers/plans/YYYY-MM-DD-<topic>.md` |
| 7 | — | — | **Commit docs only** (`docs/` zone) | — |
| 8 | `executing-plans` or `subagent-driven-development` | code | Implement plan tasks (strict TDD per task — see below) | application code + tests |
| 9 | `verification-before-completion` | code | Run project test/build/lint + coverage/mutation if configured | evidence in terminal |
| 10 | — | code | **Commit code only** (one vertical slice per commit when using strict TDD) | — |

**Example prompts (in order):**

1. "Brainstorm CSV export — what should it do?"
2. "Write the design spec for export."
3. "Update the glossary with any new domain terms."
4. "Grill the export plan before we implement."
5. "Create the implementation plan."
6. "Implement `docs/superpowers/plans/2026-08-26-export.md` with strict TDD — one behavior per commit."

SDD does not require `proposal/` or `adr/`.

---

#### IDD — command order

| Step | Invoke | Zone | Action / command | Output |
|------|--------|------|------------------|--------|
| 1 | `brainstorming` | `docs/` | Explore intent | (dialogue) |
| 2 | `idd-workflow` | `docs/` | Write proposal | `docs/superpowers/proposal/YYYY-MM-DD-<topic>.md` |
| 3 | `adversarial-authoring` (optional) | `docs/` | Cross-model author + reviewer; capture council notes | proposal + review trail |
| 4 | — | — | **Human approves proposal** | — |
| 5 | `idd-workflow` | `docs/` | Write design (requirements + approach) | `docs/superpowers/specs/YYYY-MM-DD-<topic>-design.md` |
| 6 | `glossary` | `docs/` | Define terms used in proposal and design | `docs/glossary/{business,technical}.md` |
| 7 | — | — | **Human approves design** | — |
| 8 | `idd-workflow` + `architectural-decision-records` | `docs/` | Record durable decisions | `docs/superpowers/adr/YYYY-MM-DD-<topic>.md` |
| 9 | `grilling` (optional) | `docs/` | Stress-test plan branches with human | refined understanding |
| 10 | `writing-plans` | `docs/` | Checkbox task plan | `docs/superpowers/plans/YYYY-MM-DD-<topic>.md` |
| 11 | — | — | **Commit docs only** | — |
| 12 | `executing-plans` + `test-driven-development` | code | Apply plan (strict TDD per task) | application code + tests |
| 13 | `verification-before-completion` | code | Run test/build/lint + coverage/mutation if configured | evidence |
| 14 | `idd-workflow` | `docs/` | Promote: plan → `STATUS: ARCHIVED`; design → `STATUS: DECIDED` | updated STATUS headers |
| 15 | — | code + `docs/` | **Commit promote separately from apply** | — |

**Example prompts (in order):**

1. "Start IDD for user auth — proposal first."
2. "Run adversarial review on the proposal." *(optional, multi-model)*
3. "Design is approved; update glossary and write the ADR."
4. "Grill the auth plan before implementation."
5. "Implement the plan with strict TDD — one vertical slice per commit."
6. "Promote user-auth — archive plan, mark design DECIDED."

---

#### BDD — command order (opt-in; stacks on SDD or IDD)

Run SDD steps 1–7 or IDD steps 1–11 first so `-design.md` and glossary exist. BDD is the **macro** loop; it inserts **after design approval**, **before** full apply ([source](https://intent-driven.dev/blog/2026/08/23/tdd-bdd-spec-driven-development/)).

| Step | Invoke | Zone | Action / command | Output |
|------|--------|------|------------------|--------|
| 1 | `gherkin-authoring` | `docs/` | Add Gherkin fenced blocks to design | updated `*-design.md` |
| 2 | `acceptance-test-authoring` | `docs/` | Scaffold runner if missing | `docs/acceptance-tests/{cucumber.cjs,steps/,…}` |
| 3 | — | `docs/` | Extract scenarios | `node extract-gherkin.cjs ../superpowers/specs/YYYY-MM-DD-<topic>-design.md <topic>.feature` |
| 4 | — | `docs/` | Lint extracted Gherkin | `npx gherkin-lint <topic>.feature` |
| 5 | — | `docs/` | Run acceptance — expect **RED** | `node cucumber.cjs` |
| 6 | — | — | **Commit docs only** (design + runner; not `.feature` if gitignored) | — |
| 7 | `bdd-workflow` + `test-driven-development` | code | Implement features + step defs (strict TDD **micro** loop per task) | application code + `docs/acceptance-tests/steps/` |
| 8 | — | `docs/` | Re-extract and run acceptance — expect **GREEN** | `node extract-gherkin.cjs … && node cucumber.cjs` |
| 9 | `verification-before-completion` | code + `docs/` | Full suite green + project tests/build/lint | evidence |
| 10 | `bdd-git-discipline` | `docs/` | Promote (same as IDD step 12) | STATUS: ARCHIVED / DECIDED |
| 11 | — | — | **Separate commits: docs promote vs code** | — |

**Shell sequence (acceptance loop):**

```bash
cd docs/acceptance-tests
node extract-gherkin.cjs ../superpowers/specs/2026-08-26-user-auth-design.md user-auth.feature
npx gherkin-lint user-auth.feature
node cucumber.cjs                    # RED before apply
# … apply with TDD …
node extract-gherkin.cjs ../superpowers/specs/2026-08-26-user-auth-design.md user-auth.feature
node cucumber.cjs                    # GREEN after apply
```

**Example prompts (in order):**

1. "Add Gherkin scenarios to the user-auth design doc."
2. "Extract and run acceptance tests — they should fail first."
3. "Implement until Cucumber is green, then promote."

---

#### TDD — command order (runs inside every apply step)

Repeat this cycle **for each plan checkbox** that changes behavior. This is the **micro** (Mockist) loop — one vertical slice at a time. Do not batch multiple independent behaviors in one test or one commit ([anti-patterns](https://intent-driven.dev/blog/2026/08/23/tdd-bdd-spec-driven-development/)).

| Step | Invoke | Zone | Action / command | Output |
|------|--------|------|------------------|--------|
| 1 | `test-driven-development` | code | Write **one** failing test for **one** behavior | test file |
| 2 | — | code | Run test — must **fail** for the right reason (not error, not pass) | `<project test command>` |
| 3 | `test-driven-development` | code | Write minimal production code to pass **only** that test | source file |
| 4 | — | code | Run test — must **pass**; suite stays green | `<project test command>` |
| 5 | `test-driven-development` | code | Refactor production + test code (no new behavior) | cleaned source/tests |
| 6 | — | code | Re-run test suite | all green |
| 7 | — | code | **Commit** this vertical slice (git history = proof of order) | one commit |
| 8 | — | — | Mark plan checkbox `[x]`; repeat from step 1 for next behavior | — |

**Typical test commands (use what the project defines):**

```bash
go test ./...                          # Go
npm test path/to/test.test.ts          # Node (targeted)
npm test                               # Node (full suite)
uv run pytest tests/test_foo.py        # Python (targeted)
```

**Iron law:** if production code was written before step 1, delete it and restart the cycle.

#### Strict TDD — definition of done (anti–reward-hacking)

Prompting "use strict TDD" alone does not stop agents from reward-hacking: broad batched tests, code added without a failing test, weak assertions that stay green when behavior changes. Add these constraints during apply ([full analysis](https://intent-driven.dev/blog/2026/08/23/tdd-bdd-spec-driven-development/)):

| Constraint | What it prevents | How to verify |
|------------|------------------|---------------|
| **Vertical slicing** | One test covering standard + express + free shipping + currency in a single step | One behavior per red-green cycle; one commit per slice |
| **Fail for the right reason** | Test passes or errors instead of failing on missing feature | Read failure message before writing code |
| **Precise assertions** | `toThrow()` without type/message; loose numeric checks | Assert exact outputs, error types, collaborator args |
| **Boundary coverage** | Missing equals-case at thresholds (e.g. 5kg vs 4.9kg) | Test both sides of every meaningful branch |
| **Branch coverage** | Code paths never exercised | Project coverage tool (branch, not just line) |
| **Mutation testing** | Tests green but wrong implementation survives | Stryker or equivalent — surviving mutants → tighter assertions |
| **Test refactor** | Duplicated setup obscures behavior | jscpd on test files; shared fixtures only for repeated structure |

**Agent failure modes to watch for:**

| Anti-pattern | Symptom | Fix |
|--------------|---------|-----|
| Horizontal slicing | One test asserts many independent behaviors | Split into one test per behavior |
| Fake TDD | Many small tests but code written ahead of failures | Delete code; restart from red |
| Weak assertions | Green suite, wrong error type or collaborator args | Pin exact outputs, types, call signatures |
| Test duplication | Same fixture copy-pasted everywhere | Extract shared setup; keep boundary values visible in table-driven cases |

**Example prompts (in order):**

1. "Task 2.1 with strict TDD — one failing test, one behavior, commit when green."
2. "Run the test and confirm it fails because the feature is missing."
3. "Minimal code to green; refactor tests; run mutation testing on this module."

---

#### Combined: IDD + BDD + TDD (full feature)

| Phase | Steps | Zone | Layer |
|-------|-------|------|-------|
| Intent + spec quality | IDD 1–11 (+ glossary, optional grilling/adversarial) | `docs/` → commit | SDD / IDD |
| Acceptance setup | BDD 1–6 | `docs/` → commit | BDD macro |
| Implementation | Strict TDD cycle per plan task + BDD step 7 | code → commit per slice | TDD micro |
| Close | BDD 8–10 + IDD promote | re-run Cucumber → STATUS | verify + archive |

**One-line mnemonic** ([source](https://intent-driven.dev/blog/2026/08/23/tdd-bdd-spec-driven-development/)):

```
IDD docs (+ glossary) → BDD red (macro) → TDD green per slice (micro) → BDD green → promote docs
```

### Project layout

Everything the agent reads before coding lives under `docs/`:

```
project/
├── docs/
│   ├── superpowers/
│   │   ├── proposal/   YYYY-MM-DD-<topic>.md
│   │   ├── specs/      YYYY-MM-DD-<topic>-design.md
│   │   ├── adr/        YYYY-MM-DD-<topic>.md
│   │   └── plans/      YYYY-MM-DD-<topic>.md
│   ├── glossary/       business.md, technical.md
│   └── acceptance-tests/
│       ├── <topic>.feature          # extracted from -design.md
│       ├── cucumber.cjs, steps/, …  # runner (created on first BDD apply)
│       └── .extracted/, reports/    # gitignored scratch
└── AGENTS.md                        # project agent rules (repo root)
```

**Correlation:** the same `YYYY-MM-DD-<topic>` slug links proposal, design, plan, ADR, and acceptance feature:

```
2026-08-26-user-auth
  → docs/superpowers/proposal/2026-08-26-user-auth.md
  → docs/superpowers/specs/2026-08-26-user-auth-design.md
  → docs/superpowers/plans/2026-08-26-user-auth.md
  → docs/acceptance-tests/user-auth.feature   (extracted at test time)
```

Human-facing docs (`docs/02-usage.md`, etc.) are separate from the IDD/BDD artifact chain — agents treat `docs/superpowers/`, `docs/glossary/`, and `docs/acceptance-tests/` as the specs zone.

### SDD — Spec-Driven Development

**Idea:** agree on behavior and implementation steps in writing before touching application code.

**Command order:** see [SDD — command order](#sdd--command-order) above.

**Artifacts:**

| File | Role |
|------|------|
| `docs/superpowers/specs/YYYY-MM-DD-<topic>-design.md` | Requirements, design, acceptance criteria |
| `docs/superpowers/plans/YYYY-MM-DD-<topic>.md` | Checkbox task breakdown |

**Skills used (in order):** `brainstorming` → design spec → `glossary` → `grilling` (optional) → `writing-plans` → `executing-plans` or `subagent-driven-development` → strict `test-driven-development` (per task) → `verification-before-completion`

### IDD — Intent-Driven Development

**Idea:** extend SDD with explicit intent (proposal), durable decisions (ADR), and a promote/archive lifecycle.

**Command order:** see [IDD — command order](#idd--command-order) above.

**Additional artifacts:**

| File | Role |
|------|------|
| `docs/superpowers/proposal/YYYY-MM-DD-<topic>.md` | Why, scope, capabilities |
| `docs/superpowers/adr/YYYY-MM-DD-<topic>.md` | Cross-cutting decisions (immutable once accepted) |
| `docs/glossary/{business,technical}.md` | Domain terms agents must use consistently |

**Skills used (in order):** `brainstorming` → `idd-workflow` → `adversarial-authoring` (optional) → design → `glossary` → `architectural-decision-records` → `grilling` (optional) → `writing-plans` → strict `test-driven-development` → `verification-before-completion` → promote

**STATUS progression:**

| Artifact | Values |
|----------|--------|
| `proposal/*.md` | draft → active → superseded |
| `specs/*-design.md` | draft → active → **decided** → superseded |
| `adr/*.md` | proposed → accepted → superseded |
| `plans/*.md` | **active** → **archived** |

Change-scoped choices stay in `-design.md`; decisions that outlive the feature go to `adr/`.

### BDD — Behavior-Driven Development

**Idea:** express acceptance behavior as Gherkin in the design doc, extract to Cucumber, run red → green before promote. BDD is the **macro** feedback loop — it verifies observable behavior from the outside ([source](https://intent-driven.dev/blog/2026/08/23/tdd-bdd-spec-driven-development/)).

**Command order:** see [BDD — command order](#bdd--command-order-opt-in-stacks-on-sdd-or-idd) above. Requires an approved `-design.md` and glossary first (from SDD or IDD).

**Source of truth:** Gherkin in `docs/superpowers/specs/*-design.md`. **Derived:** `docs/acceptance-tests/<topic>.feature` — extract on each run; do not hand-edit.

**Skills used (in order):** `gherkin-authoring` → `acceptance-test-authoring` → extract/lint/run (shell) → `bdd-workflow` + strict `test-driven-development` (micro, per task) → extract/run again → `bdd-git-discipline` → promote

Line numbers from `gherkin-lint` map back to the design doc, not the extracted file.

### TDD — Test-Driven Development

**Idea:** write a failing test first, watch it fail for the right reason, write minimal code to pass, refactor. Runs **inside apply** as the **micro** (Mockist) loop for SDD, IDD, and BDD ([source](https://intent-driven.dev/blog/2026/08/23/tdd-bdd-spec-driven-development/)).

**Command order:** see [TDD — command order](#tdd--command-order-runs-inside-every-apply-step) and [Strict TDD — definition of done](#strict-tdd--definition-of-done-antireward-hacking).

**Skill:** superpowers `test-driven-development` (not vendored by skillgrid).

**Iron law:** no production code without a failing test first.

**When to use:** new features, bug fixes, refactoring, behavior changes. Exceptions (prototypes, generated code, config-only) need human approval first.

**Where it sits:**

```
plan → apply → [strict TDD cycle × N behaviors, 1 commit each] → verification-before-completion → promote
              ↑ BDD macro (Cucumber) may stay red until micro slices complete
```

### The two rules (all workflows)

1. **Zone separation** — never co-edit `docs/` (superpowers, glossary, acceptance-tests) and application code in one uncommitted unit. Zone-guard (`config.d/hooks/zone-guard.sh`) enforces this after install.
2. **Verify before promote** — acceptance suite green (when BDD is enabled) and `verification-before-completion` passes before marking a change done.

### STATUS and archive

Active work = plans without `STATUS: ARCHIVED` and designs without `STATUS: DECIDED`. Do not move files to archive folders; update STATUS headers in place.

The acceptance runner uses **DECIDED** designs and **non-archived** plans only.

### Choosing a workflow

| Situation | Command order to follow |
|-----------|-------------------------|
| Small fix, clear scope | [SDD](#sdd--command-order) (skip early steps if design exists) → [Strict TDD](#strict-tdd--definition-of-done-antireward-hacking) only |
| New feature, needs alignment on *why* | [IDD](#idd--command-order) full sequence (+ [glossary](#spec-quality-before-apply)) |
| Terminology drift across specs | Run `glossary` before next design; reuse terms from `docs/glossary/` |
| User-visible behavior, regression-sensitive | [IDD](#idd--command-order) → [BDD macro](#bdd--command-order-opt-in-stacks-on-sdd-or-idd) → [TDD micro](#strict-tdd--definition-of-done-antireward-hacking) — or [Combined](#combined-idd--bdd--tdd-full-feature) |
| Multi-model spec review | [IDD](#idd--command-order) step 3 `adversarial-authoring` (optional) |
| Plan needs stress-test before build | `grilling` on proposal or plan ([source](https://intent-driven.dev/blog/2026/06/27/sdd-adversarial-authoring-glossary/)) |
| Bug fix only | [Strict TDD](#strict-tdd--definition-of-done-antireward-hacking) cycle only |
| Spike or feasibility | `brainstorming` spike path — no numbered sequence |

### What skillgrid ships vs optional

| Skill | Status | Role |
|-------|--------|------|
| `glossary` | vendored (`config.d/skills/`) | Terminology consistency during design |
| `grilling` | vendored (`config.d/skills/`) | Human-in-the-loop plan stress-test |
| `gherkin-authoring`, `bdd-workflow`, … | vendored | BDD macro layer |
| `test-driven-development` | superpowers plugin | TDD micro layer (not duplicated) |
| `adversarial-authoring` | not vendored v1 | Cross-model proposal review — optional when you have two models |

### What skillgrid does not ship

- **OpenSpec CLI** or `openspec/` directory trees
- Nested `changes/` / `capabilities/` folder layouts
- A replacement for superpowers `test-driven-development` (use the plugin skill as-is)

Optional **SDD sub-agent skills** (`sdd-propose`, `sdd-spec`, `sdd-apply`, …) from the broader ecosystem can persist to Engram instead of files. skillgrid's file-based layout above is the superpowers + IDD/BDD path — no OpenSpec dependency.

### Further reading

- [How TDD and BDD Actually Fit Into Spec-Driven Development](https://intent-driven.dev/blog/2026/08/23/tdd-bdd-spec-driven-development/) — macro BDD, micro Mockist TDD, strict TDD harness, mutation testing
- [SDD with Multi-Model Spec Review and Glossary](https://intent-driven.dev/blog/2026/06/27/sdd-adversarial-authoring-glossary/) — adversarial authoring, glossary as first-class artifact
- [Intent-Driven Template](https://github.com/intent-driven-dev/intent-driven-template) — upstream skills skillgrid adapts (without OpenSpec CLI)
- **Testing enforcement:** [2026-08-26-testing-enforcement-design.md](superpowers/specs/2026-08-26-testing-enforcement-design.md) — test layers L0–L5, tools per stack, CI gates ([plan](superpowers/plans/2026-08-26-testing-enforcement.md))
