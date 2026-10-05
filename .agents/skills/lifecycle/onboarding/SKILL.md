---
name: onboarding
description: Use when setting up Skillgrid for a new project. Detects stack, testing, and tracker; writes .skillgrid/config.yaml and an AGENTS.md block. Run once per project.
license: MIT
disable-model-invocation: true
metadata:
  author: devopstales
  version: "1.1"
  part-of: skillgrid
  based_on: skillgrid-v2:sdd-onboard + mattpocock-skills:setup-matt-pocock-skills + BMAD:bmad-project-context
---

# Onboarding

Detect project facts, confirm with the user, write `.skillgrid/config.yaml` and an AGENTS.md block. Run once per project — subsequent runs merge, not overwrite.

**The real job is inherited code.** The typical onboarding target is not a shiny new project — it is a codebase somebody else wrote, years old, with conventions to reverse-engineer and no documentation (or outdated documentation). Onboard it by reading what's actually there, not what a project "like this" would look like. Plan the next slice **on top of reality**, against the constraints the existing system already imposes — the data it has, the patterns it follows. **Reuse instead of regenerate.**

**Announce at start:** "I'm using the skillgrid:onboarding skill to configure Skillgrid for this project."

## When to Use

- First time running any Skillgrid skill in a new project (no `.skillgrid/config.yaml` exists)
- User says "set up Skillgrid", "init Skillgrid", "onboard this project"
- Project stack changed significantly (new language, new test runner)

**When NOT to use:** the project already has a valid `.skillgrid/config.yaml` and nothing about the stack changed — onboarding re-detects and confirms, it's not a per-task step. (Re-running to pick up a stack change IS a use, via merge mode.)

## Keep the AGENTS.md block lean

The `## Skillgrid` block is injected into the agent's context on **every** request, so bloat in it poisons *all* prompts. Treat it as a **navigation spine** — it points at where things live — not a knowledge dump. Source: the "configure your AI so it doesn't suck" discipline (WebDev Simplified), folded in here.

Apply these when rendering the block and whenever the user asks to extend the AGENTS.md:

1. **Keep it small.** Every line here ships with every prompt. If a section isn't needed for *most* requests, it does not belong inline.
2. **Split section-specific conventions out** into the files the block already references (`.skillgrid/artifacts/`, the skills, `docs/user-guide/`) and *link* to them, instead of inlining CSS/API/TS/delivery rules. A one-line "reference →" beats a ten-line dump.
3. **No "getting started / setup" section.** Onboarding, install, and first-run steps are not what the agent file is for — the agent already has the repo.
4. **Grow only on an observed failure (observe → fix).** Add a line to the block only when the agent *actually* did something wrong that this file can prevent. Don't preemptively document mistakes it hasn't made — you don't know which it will make. Notice a problem → record it in the right place → it stops repeating.
5. **Reference, don't write.** A standards line names ONE canonical file (or skill) in the pattern `**<Topic>** — reference `<path>` for detailed <topic>.` — never restate the standard's content in the block. The canonical `## Skillgrid` block carries the four reference lines (coding, testing, commits/verification, memory/code-index/web-cache) by default; a project with no shared-standards file for a topic skips that line rather than inventing content.
6. **High-level layout only.** A one-line "what's here" map is fine; never point at individual files or churning paths — they go stale and mislead the agent.
7. **Start minimal.** On a greenfield project, a couple of sentences (what the project is, the package manager if non-standard) + the references is the right size. Never "read the repo and generate a giant AGENTS.md" — that yields verbose, hard-to-use files.
8. **One source of truth across platforms.** `AGENTS.md` is the source of truth. `CLAUDE.md` / `GEMINI.md` get a **one-line pointer** (Skillgrid's choice over a symlink, which breaks under git/CI on some systems). Never two full blocks — they drift.
9. **Nested AGENTS.md for monorepos/areas.** Area-specific rules (a payments module, a UI kit, a `apps/api` vs `apps/web` split) live in a nested `<area>/AGENTS.md`; the agent loads root + the nested file for the folder it's working in. The root block stays global and lean.

**The preamble is separate from the sentinel block.** The rules above govern the `## Skillgrid` **sentinel block** — the lean navigation spine that ships with every prompt. The **preamble** (the `# Standards` region: `Environment & Tooling`, `Key Directories`, `Security & Escalation Boundaries`, `Dependency Policies`, `Architecture Constraints`, `Definition of Done`, `Engineering Standards`) is the one place AGENTS.md states *repo-specific facts* inline, because they're exactly what every request needs and there's no better canonical file for them. The "reference, don't write" rule applies to shared standards (testing, memory, code-indexing, SDD) — those stay reference pointers to the installed rule files; it does not apply to the repo-specific preamble sections. Render the preamble by filling `.skillgrid/config.yaml`'s `agents:` block and running `skillgrid init` (which writes `templates/agents-preamble.md`'s structure) — never by hand-editing the region.

These principles govern *what goes in the block*. The block's exact shape and the idempotent upsert rule stay authoritative in `_shared/agent-config/block.md`. The preamble's exact shape stays authoritative in `templates/agents-preamble.md`.

## The Process

### Step 1: Detect

Read the project and detect facts. Never guess — detect, then confirm.

**Source precedence** (for project name, stack, testing, tracker): **AGENTS.md/CLAUDE.md → `.skillgrid/config.yaml` → Mnemonic → git remote → project files**. First source that answers wins; later sources fill gaps. Do not re-scan the world for a fact that an earlier source already answers.

**Git worktree guard:** before any artifact write, run `git rev-parse --is-inside-work-tree`. If it fails, run `git init` (respects monorepos — never nest a repo inside one). All artifacts then land in a tracked repo.

**Project name:**
- Existing `AGENTS.md` / `CLAUDE.md` / `.skillgrid/config.yaml` → read it
- `git remote get-url origin` → extract repo name
- Fallback: directory name
- If neither: ask the user

**Stack & context:**
- Read manifests: `package.json`, `go.mod`, `pyproject.toml`, `Cargo.toml`, `Makefile`, `Taskfile.yml`
- Read CI config: `.github/workflows/`, `.gitlab-ci.yml`, `Jenkinsfile`
- Compose a one-line summary: stack + purpose + constraints
- Keep it ≤ 5 lines

**Testing:**
- Detect test runner from manifests:
  - `package.json` → `scripts.test` or `"test": "jest"` / `"test": "vitest"`
  - `go.mod` → `go test ./...`
  - `pyproject.toml` → `pytest`
  - `Cargo.toml` → `cargo test`
- Detect setup command:
  - `package.json` → `npm install` (or `pnpm install` / `yarn` if lockfile present)
  - `go.mod` → `go mod download`
  - `pyproject.toml` → `pip install -e .` or `poetry install`
  - `Cargo.toml` → (no setup needed)
- Detect test layers: check for `test/`, `tests/`, `*_test.go`, `*.test.ts`, `*.spec.ts`
- Detect coverage: check for coverage tool in CI or manifests
- Detect mutation testing: check for `stryker.config.mjs` / `stryker.config.js` / `stryker.config.cjs` in project root. If present, `mutation: "npx stryker run"`. If absent, `mutation: ""` (disabled).

**Security scanner (Trivy):**
- Check if the `trivy` CLI is available: `command -v trivy`
- If available, check version: `trivy --version`
- If available: `security.trivy.command: "trivy fs"` (filesystem scan of the current directory)
- If not available: `security.trivy.command: ""` (disabled — QA falls back to manual checks)
- Defaults: `severities: "CRITICAL"`, `scan_types: "vuln,secret,misconfig"`, `fail_on: "CRITICAL"`

**Commands:**
- Build: `package.json` → `scripts.build`; `go.mod` → `go build ./...`; `Makefile` → `make build`
- Lint: `package.json` → `scripts.lint`; `go.mod` → `golangci-lint run`; `pyproject.toml` → `ruff check`
- Format: `package.json` → `scripts.format` or `prettier --write .`; `go.mod` → `go fmt ./...`
- Typecheck: `package.json` → `scripts.typecheck` or `tsc --noEmit`; `go.mod` → `go vet ./...`

**AGENTS.md preamble facts** (the repo-specific sections rendered by `skillgrid init` from `templates/agents-preamble.md` into `.skillgrid/config.yaml`'s `agents:` block — brownfield: read the repo + `ASSUMPTIONS.md`, don't guess; greenfield: leave the lists empty):
- `agents.directories` — walk the top-level tree; one `{path, purpose}` row per significant directory (app factory, core libs, blueprints/routes, tests, docs, the task/build manifest). This becomes the `Key Directories` table. Empty on greenfield (the section is dropped).
- `agents.security_boundaries` — bullets from README/CI/security docs (what the agent may not touch, what needs approval, which protections are enforced).
- `agents.dependency_policies` — bullets (prefer stdlib, require approval for new deps, pin versions, language version pins).
- `agents.architecture_constraints` — bullets from `.skillgrid/ASSUMPTIONS.md` § Locked constraints + layering docs (layering rules, replication constraints, storage modes).
- `agents.definition_of_done` — bullets from CI/Taskfile gates (tests pass, lint/format pass, security scan clean, spec updated).
- `agents.engineering_standards` — bullets reversed from the code (test behavior changes, small reviewable diffs, follow existing patterns, reuse before creating).

**Ticketing:**
- `git remote get-url origin` → contains `github.com` → `gh`
- `git remote get-url origin` → contains `gitlab.com` → `glab`
- No remote or user preference → `backlogmd` (default, local)
- User names Jira → `jira` (ask for instance URL + project key)
- Ask the user: "Do you want to publish tickets to a tracker? Or work local-only from tasks.md?"
  - If yes → `ticketing.enabled: true`, use detected tracker type
  - If no → `ticketing.enabled: false`, skip tracker type

**TDD mode:**
- Always `true` — TDD is non-negotiable in Skillgrid

**Commit discipline:**
- Always on — `skillgrid:work-unit-commits` is fully active during execution
- Ask the user: "Commit mode — (1) commit-only, (2) commit + session events (recommended)?"
  - (2) → install the `pre-commit` + `commit-msg` hook shims; each work-unit commit feeds a `commit` event
  - (1) → conventional commits + guards only, no session event detail

**ADR style:**
- Detect an existing `adr_style` in `.skillgrid/config.yaml` (if the file exists) → use it
- No existing value → default `madr-minimal`, but ask the user which they prefer (see Step 2)

**BDD / Acceptance testing:**
- Always `enabled: true` — BDD is non-negotiable in Skillgrid (like TDD)
- Detect an existing `bdd.stack` in `.skillgrid/config.yaml` (if the file exists) → use it
- No existing value → default `stack: javascript` (cucumber-js)
- Detect `acceptance-tests/` at repo root → if present, confirm `stack: javascript` (or infer from contents)

**Mnemonic:**
- Check if the `skillgrid` CLI is available: `command -v skillgrid`
- If available, verify the MCP server can start: `skillgrid mcp --help`
- If not available: note "Mnemonic not detected — memory, code index, and web cache will be inactive"
- If available: `mnemonic: enabled: true` in config

**Embedder (mnemonic vector search):**
- Only relevant when Mnemonic is enabled
- Detect existing config: check `.skillgrid/config.d/indexing.yaml` for an `embedder:` block. If present, use it as the starting point and confirm with the user.
- Ask the user: "How should Mnemonic embed code for vector search? Options:
  1. **Ollama (local LLM server)** — run embeddings on a local Ollama instance. Requires Ollama installed and running.
  2. **ONNX (built-in)** — pure-Go ONNX runtime, no external server. Model auto-downloaded to `~/.skillgrid/models/`.
  3. **External API** — any OpenAI-compatible `/embeddings` endpoint (OpenAI, OpenRouter, LiteLLM, etc.)
  4. **Off** — no vector leg; hybrid search falls back to FTS + signals only.
  Default: ONNX (no setup required). Which do you want?"
- If **Ollama**:
  - Verify Ollama is running: `curl -sf http://localhost:11434/api/tags | head -5`
  - If not reachable: "Ollama not running on localhost:11434. Is it on a different host/port? If not installed, you can start with `brew install ollama` (macOS) or `curl -fsSL https://ollama.com/install.sh | sh` (Linux). Install the embedding model with `ollama pull nomic-embed-code`."
  - Ask for base URL (default `http://localhost:11434`) and model name (default `nomic-embed-code`)
- If **External API**:
  - Ask for base URL (e.g. `https://api.openai.com/v1`), model name (e.g. `text-embedding-3-small`), and API key
  - API key: ask the user to provide it, or note that it can be set later in `~/.skillgrid/config.d/indexing.yaml` (machine-local, not committed)
- If **ONNX** or **Off**: no further config needed

### Step 1.5: Enroll (brownfield)

If Step 1 shows an existing codebase (real commit history + feature code, not just a scaffold), enroll what's already there instead of planning it from zero:

1. **Detect existing capabilities.** From the source tree, routes, manifests, and git history, list the features/capabilities that already exist (a few lines each — what it does, where it lives).
2. **Mark them `existing`.** They are not planned-from-zero and not `planned` — they predate this workflow. Record them in the `## Skillgrid` block's context note (or the briefing of the first change) so `skillgrid:writing-blueprints` designs the next slice against them: the data they hold, the patterns they follow. The first blueprint's Build shape and File Structure start from "what's already there", not from an empty tree.
3. **Reverse-engineer conventions.** Capture the project's actual conventions (error handling, naming, layering, test style) into the `## Skillgrid` block and the terms files (`artifacts/01-business-terms.md` / `02-technical-terms.md`), from the code — not from assumptions about the stack. If a convention is surprising or undocumented, flag it for the user rather than guessing.
4. **Nested AGENTS.md note.** Area-specific rules (a payments module, a UI kit) live in `<area>/AGENTS.md` — the root block stays global and lean. Note the areas that would warrant their own file; create them lazily (ship's reconcile creates one for a net-new area, never for a pre-existing undocumented one — it flags instead).

A greenfield project (scaffold, no feature history) skips Step 1.5 — there is nothing to enroll.

### Step 1.7: Generate ARCHITECTURE.md

`skillgrid init` (Step 3 item 10) scaffolds `.skillgrid/ARCHITECTURE.md` from `templates/architecture.md` — a 17-section skeleton with `<detect>`/`<detect: …>` markers and a ToC. Your job is to **fill the markers with detected facts** so the document is a faithful, verifiable map of the codebase — not a generic template. This is the onboarding deliverable that makes the repo *legible*: a newcomer (or the agent on a fresh session) reads this before reading code.

**Principles**
1. **Facts, not opinions.** Every diagram, table row, and code snippet must correspond to something real — a package, a `file:line`, a config key, a port. If you cannot verify it from the code, leave it `<detect>` rather than invent it.
2. **Point at ADRs, don't re-argue them.** Architecture *decisions* live as `### ADR-NNNN` in `.skillgrid/ASSUMPTIONS.md` (LOCKED). This file describes *how it is built* and links to the ADR that pins a rule; it does not restate the tradeoff. (See `skillgrid:architectural-decision-records`.)
3. **Verbatim snippets are load-bearing.** Short quoted snippets (a migration header, a port constant, an entry-point line) make the doc *checkable*. Quote the real line, cite `file:line`. Keep each snippet ≤ ~15 lines.
4. **Drop what doesn't exist.** A project with no frontend omits §9; no pipeline omits §12; no plugin system omits §10. Do not leave an empty section with placeholder prose — remove the section and renumber the ToC, or mark it `_(none)_`.
5. **Greenfield stays a skeleton.** On a scaffold with no feature history, fill only what exists (stack, top-level layout, entry point, test strategy) and leave the rest `<detect>` — the doc grows as the code does. Never pad a greenfield doc.

**How to fill each section** — for every `<detect: …>` marker, read the named source of truth:

| Section | Detect from |
|---|---|
| 1. Overview | README + entry point; one ASCII diagram of major subsystems |
| 2. Top-Level Layout | `eza`/`ls` of the root; one row per significant path |
| 3. Application Layering | the import graph / package boundaries (which layer imports which) |
| 4. Entry Points & Command Surface | `main`/entry file `file:line` + the command dispatch table |
| 5. Domain / Core Logic | the core packages + their responsibilities |
| 6. Data & Persistence | schema/migration files + the ingest→index→search (or equivalent) flow |
| 7. Storage & State | the store (engine + on-disk layout) + migration strategy + state zones |
| 8. Transports | any HTTP/MCP/WebSocket server: protocol, port, shared handler set |
| 9. Frontend/UI | the UI stack + build tool + dev-vs-prod split (omit if none) |
| 10. Plugin & Extension | the registry, hook points, multi-agent/adapter stages (omit if none) |
| 11. Configuration & Flags | config sources + precedence order + feature-flag location |
| 12. Workflow / Pipeline | the staged workflow (e.g. SDD phases) with per-stage artifact + gate (omit if none) |
| 13. Enforcement & Quality Gates | git hooks + CI + lint/typecheck/test gates that block |
| 14. Error Handling & Degradation | error convention + which subsystems fail open vs closed |
| 15. Security | the security layers + the controls table (auth, secrets, scans) |
| 16. Testing Strategy | the test layers + where each lives + the runner + coverage |
| 17. Known Gaps & TODOs | honest list of stubbed/deferred/inconsistent things (never leave empty out of politeness) |

**Method (TDD-flavored for documentation):** before writing a section, *read* its source of truth; write the section; then *verify* the load-bearing claims (re-open the `file:line` you cited and confirm the snippet matches). A section you cannot verify stays `<detect>`.

**Brownfield:** use the reverse-engineered conventions from Step 1.5 — the doc is where "how this *actually* works" (vs how it *should* work) is recorded. Flag any place the code contradicts the docs.

**When the CLI is not on PATH:** `skillgrid init` is the deterministic scaffolder, but onboarding must not block on it. Copy `templates/architecture.md` to `.skillgrid/ARCHITECTURE.md`, replace `{project}`/`{version}`/`{last_updated}` by hand, and fill the markers as above. The result is identical.

### Step 2: Present & Confirm

Show the detected facts to the user. Confirm one at a time — don't dump everything at once:

1. **Project name** — "Detected project: X. Correct?"
2. **Stack & context** — "I see a Go 1.22 project with tests. Here's the context I'll write: [show]. Does that capture it?"
3. **Testing** — "Test runner: `go test ./...`. Setup: `go mod download`. TDD: always on. Correct?"
4. **Commands** — "Build: `go build ./...`. Lint: `golangci-lint run`. Format: `go fmt ./...`. Typecheck: `go vet ./...`. Correct?"
5. **AGENTS.md preamble** — "I'll fill the repo-specific AGENTS.md preamble (Key Directories, Security Boundaries, Dependency Policies, Architecture Constraints, Definition of Done, Engineering Standards) from what I detected in the repo. Here's the draft: [show the `agents.*` bullets/rows]. Correct?" (greenfield: "This is a greenfield project — I'll leave the repo-specific preamble sections empty for now; they fill in as the code takes shape. OK?")
6. **Ticketing** — "Do you want to publish tickets to a tracker? I detected GitHub — I'd use `gh`. Or Backlog.md (local), GitLab, Jira, or none (tasks.md only)?"
7. **ADR style** — "How should I format ADRs? Options: `madr-full` (detailed tradeoff record), `madr-minimal` (context/options/decision/consequences — recommended default), `nygard` (classic status/context/decision/consequences), `y-statement` (one sentence), or `custom` (your house format). I'll store the choice and never re-ask."
8. **Artifact paths** — "Durable knowledge lives at the repo root: `.skillgrid/ASSUMPTIONS.md` (project understanding + all architectural decisions/ADRs + locked constraints) and `.skillgrid/ARCHITECTURE.md` (live repo/program structure). Durable artifacts (terms glossary + research findings): `.skillgrid/artifacts/`. Prototypes (feasibility probes, permanently retained): `.skillgrid/prototypes/`. State (dynamic: phase, current change, progress): `.skillgrid/state.yaml`. Specs: `.skillgrid/specs/`. Scratch: `.skillgrid/sdd/`. Worktrees: `.worktrees/`. Any overrides?"

8. **Domain model** — "The architectural-decision-records skill maintains the project vocabulary at `.skillgrid/artifacts/01-business-terms.md` + `02-technical-terms.md` and the ADR trail as `.skillgrid/artifacts/04-adr-NNNN-slug.md`, with a path row in `.skillgrid/ASSUMPTIONS.md` (§ `### In-force set`). ADRs use the `{adr_style}` format you just picked. User-locked project-wide boundaries live in `.skillgrid/ASSUMPTIONS.md` (§ Locked constraints) and render into the AGENTS.md `### Rules` section. Nothing to set up — they appear on first use. OK?"

9. **BDD / Acceptance testing** — "Executable acceptance tests: always on (like TDD). Stack: `javascript` (cucumber-js). Specs live at `.skillgrid/specs/<id>/acceptance.feature`; the runner at `acceptance-tests/` is scaffolded on first use. The acceptance scenarios ARE the executable spec. Correct?"

10. **QA gates** — "The `skillgrid:qa` skill enforces hard quality gates before merge. Here are the defaults — adjust any of them:
    - **Coverage minimum:** 80% (set 0 to disable)
    - **Mutation minimum:** 80% (set 0 to disable — requires a mutation tool like Stryker)
    - **P0 pass rate:** 100% (all P0 tests must pass)
    - **P1 pass rate:** 95% (allow 1 in 20 P1 tests to fail)
    - **Mutation command:** <detected or "not configured">
    - **Trivy security scan:** <detected + version or "not installed">
      - Severities: CRITICAL (default) / CRITICAL,HIGH / CRITICAL,HIGH,MEDIUM
      - Scan types: vuln,secret,misconfig (default) / +license
      - Fail on: CRITICAL (hard gate) / "" (report only, never blocks)
    Any changes? (If you say "defaults", I'll use them as-is.)"

11. **Rigor tier default** — "How much verification does a typical change in this project want? Tiers: T0 Prototype (self-check only), T1 Alpha (light blueprint + verify), T2 Beta (full pipeline + two-axis review), T3 GA (mutation + security + parallel fresh-model review). I recommend **{recommended tier}** for {one-line reason from the stated product's risk/size — e.g. 'a user-facing product with payments'}. Any change can override per-change in its briefing (`Tier: T<n>`)." Store in `rules.tiers.default`. Never re-ask once set.

12. **Mnemonic** — (only when the `skillgrid` CLI was detected) "Persistent memory detected (skillgrid CLI). Memory, code index, and web cache will be active. Artifacts saved to Mnemonic survive /clear and branch switches. OK?" — (when not detected) "Mnemonic not detected (skillgrid CLI not found). Memory, code index, and web cache will be inactive until the CLI is installed."

13. **Embedder** — (only when Mnemonic is enabled) "Vector search embedder: **{provider}** ({one-line description}). Write the `embedder:` block to `.skillgrid/config.d/indexing.yaml`? — (ollama) "provider: ollama, base_url: {url}, model: {model}" — (external) "provider: external, base_url: {url}, model: {model}, api_key: {key}" — (onnx) "provider: onnx, model: nomic-embed-code (auto-downloaded)" — (off) "no vector leg; FTS + signals only". Correct?"

If the user corrects anything, use their value. If they say "looks good" or "yes", move on.

### Step 3: Write

**1. Write `.skillgrid/config.yaml`:**
- Copy `templates/config.yaml` from this skill's directory
- Fill in all detected values, including the `agents:` block (directories, security_boundaries, dependency_policies, architecture_constraints, definition_of_done, engineering_standards) from the preamble facts detected in Step 1 — `skillgrid init` renders these into the AGENTS.md preamble (see `templates/agents-preamble.md`). Do not hand-write the preamble sections; fill config and let init render.
- If the file already exists, merge: keep existing values the user confirmed, update changed ones
- Do NOT overwrite `rules:` sections the user has customized

**2. Write `.skillgrid/state.yaml` (create if absent; merge if present):**
- Schema per `_shared/rules/sdd-structure.md` (§ `state.yaml` and `artifacts/`).
- Initialize: `pipeline.current_phase: brainstorming`, `pipeline.current_change: ""`, `pipeline.status: pending`, `progress.completed_changes: <count of `.skillgrid/archive/*/` dirs>`, `progress.blocked_changes: 0`, `constraints_ref: .skillgrid/ASSUMPTIONS.md`, `notes: ""`.
- If the file already exists (re-onboarding), keep the existing `pipeline` + `progress` values; only fill missing keys. Never reset `completed_changes`.

**3. Create the durable-knowledge zone (create if absent; merge if present):**
- If `.skillgrid/ASSUMPTIONS.md` is absent, create it from the shape in `_shared/rules/sdd-structure.md` — the four tiers `## VERIFIED`, `## INFERRED (HYPOTHESIS)`, `## LOCKED` (with a `### In-force set` table whose rows are paths to `artifacts/04-adr-NNNN-slug.md`, plus `### Locked constraints` and `### Locked assumptions`), and `## Open Questions`. On a fresh project the tiers start empty.
- If `.skillgrid/ARCHITECTURE.md` is absent, `skillgrid init` scaffolds it from `templates/architecture.md` (the deterministic structure: header + 17-section skeleton with `<detect>` markers). Onboarding then **fills** it — see *Generating ARCHITECTURE.md* below. Do not hand-write the 17-section skeleton; let init scaffold it and replace the `<detect>`/`<detect: …>` markers with detected facts. If `skillgrid` is not on PATH, fall back to copying `templates/architecture.md` yourself (filling `{project}`/`{version}`/`{last_updated}` by hand) so the zone is complete.
- If `artifacts/README.md` is absent, create it from the topic-index shape in `_shared/rules/sdd-structure.md`.
- If `artifacts/06-research-findings.md` is absent, create it with an empty `## Findings` list.
- Brownfield migration of the old paths: if `artifacts/00-architecture.md` exists and `ARCHITECTURE.md` does not, fold the architecture into `ARCHITECTURE.md` and `git mv` the old file to `.skillgrid/archive/`. `artifacts/00-prd.md` stays where it is (it is the requirements reference); if `ASSUMPTIONS.md` is missing, create it from `templates/ASSUMPTIONS.md` with the product statement and a link to `00-prd.md`. If ADR bodies are inlined in `ASSUMPTIONS.md` and the matching `artifacts/04-adr-NNNN-slug.md` is missing, write each body out to that file and leave only the path in the `### In-force set` table. Do not fold ADR files or the PRD back into `ASSUMPTIONS.md`. Same for `artifacts/05-locked-constraints.md` (fold its `## Locked` list into `ASSUMPTIONS.md` § `### Locked constraints`, then archive) and a legacy `.skillgrid/glossary/` → `01/02-*-terms.md`.
- The terms files, the ADR files, `artifacts/00-prd.md`, and the ARCHITECTURE record are created lazily by their owning skills — onboarding only guarantees the zone + index + research stubs and the `ASSUMPTIONS.md` skeleton exist.

**4. Write AGENTS.md block (idempotent upsert):**
- Render the canonical `## Skillgrid` block from `../../_shared/agent-config/block.md` (fill `{project}`, `{tracker_line}`, `{memory_line}`, `{rules_block}`)
- `{tracker_line}` is copied exactly from the `{tracker_line}` table in that file for `ticketing.type`. It points at that tracker's convention file. Do not paraphrase storage paths, ID rules, or CLI flags into the block.
- `{rules_block}` is rendered from `.skillgrid/ASSUMPTIONS.md` § `### Locked constraints`: one `-` bullet per line. If that section is empty, render `No locked constraints yet — see `.skillgrid/ASSUMPTIONS.md`.`
- Target decision: if `AGENTS.md` exists → update it. Else if `CLAUDE.md` exists → update it. Else → create `AGENTS.md`.
- **Idempotent upsert** (per `../../_shared/agent-config/block.md`):
  1. Search the target file for `<!-- skillgrid:start -->`.
  2. **Found** → replace everything from that marker to `<!-- skillgrid:end -->` (inclusive) with the freshly rendered block. Never append a second copy.
  3. **Not found** → append the block at the end of the file.
- **Multi-platform rule:** if both `AGENTS.md` and `CLAUDE.md` exist, write the full block to `AGENTS.md` (source of truth) and put only a one-line pointer in `CLAUDE.md`. Two full blocks = two sources that drift.
  - **Pointer, not symlink.** Skillgrid's choice is a one-line pointer (`See AGENTS.md — the Skillgrid block there is the source of truth.`), not a symlink (`ln -s AGENTS.md CLAUDE.md`), because symlinks can break under git/CI and some editors. A symlink is a fine user-side alternative if they insist — but onboarding writes a pointer by default and never creates a second full block.
- The block is idempotent — running onboarding twice never duplicates it.
- **Nested AGENTS.md:** for monorepos or distinct areas (a `payments` module, `apps/api` vs `apps/web`), create a nested `<area>/AGENTS.md` for that area's specific rules — the agent loads root + nested for the folder it's in. The root block stays global and lean; do not push area-specific conventions up into it.

**5. Update `.gitignore`:**
- Ensure `.skillgrid/sdd/` is gitignored (scratch, ephemeral)
- Ensure `.skillgrid/brainstorm/` is gitignored (visual companion state)
- Do NOT gitignore `.skillgrid/config.yaml`, `.skillgrid/state.yaml`, `.skillgrid/specs/`, or `.skillgrid/artifacts/` (committed artifacts)

**6. Tracker bootstrap (if `ticketing.enabled: true`):**
- If `ticketing.type: backlogmd`: run `backlog init "<project>" --integration-mode cli --backlog-dir .backlog --config-location folder --zero-padded-ids 3`, verify with `backlog status`. Copy each existing `.skillgrid/artifacts/04-adr-NNNN-slug.md` into `.backlog/decisions/` (shape in `skillgrid:architectural-decision-records` → Backlog.md copy).
- If `ticketing.type: gh`: verify `gh` CLI is available (`gh --version`)
- If `ticketing.type: glab`: verify `glab` CLI is available (`glab --version`)
- If `ticketing.type: jira`: verify `jira` CLI is available (`jira config`)
- If `ticketing.enabled: false`: skip all tracker bootstrap

**7. Mnemonic saves (if `mnemonic.enabled: true`):** follow the save shape + session protocol in `_shared/rules/mnemonic-memory.md`. For onboarding specifically:
- `mem_session_start(title: "skillgrid-init/{project}")`
- `mem_save` topic_key: `skillgrid-init/{project}`, type: `architecture` — detected project context (name, stack, repo, deploy)
- `mem_save` topic_key: `skillgrid/{project}/issue_tracker`, type: `config` — tracker id + CLI + storage path
- `mem_save` topic_key: `skillgrid/{project}/testing-capabilities`, type: `config` — testing capabilities table
- `mem_session_summary` + `mem_session_end`
- If the session fails (MCP not wired yet), note it and continue — the saves will be made on first `skillgrid:mnemonic` invocation

**8. Write `.skillgrid/config.d/indexing.yaml` (embedder config, only when Mnemonic is enabled and the user chose a non-default provider):**
- Create the directory if absent: `mkdir -p .skillgrid/config.d`
- Write the `embedder:` block into `indexing.yaml`. If the file already exists, merge — keep any existing keys (include, exclude, chunk_lines, web_cache, etc.) and add/update only the `embedder:` section.
- The YAML schema (under the `mnemonic:` key, matching `config.EmbedderConfig`):

```yaml
mnemonic:
  embedder:
    provider: ollama            # ollama | onnx | local | external | off
    base_url: http://localhost:11434  # ollama/external only
    model: nomic-embed-code     # model name
    dimension: 768              # output vector dimension
    api_key: ""                 # external only
    # indexing_params / query_params: asymmetric input types (optional)
    # indexing_params:
    #   input_type: passage
    # query_params:
    #   input_type: query
```

- **Ollama** example:
```yaml
mnemonic:
  embedder:
    provider: ollama
    base_url: http://localhost:11434
    model: nomic-embed-code
    dimension: 768
```
- **External API** example:
```yaml
mnemonic:
  embedder:
    provider: external
    base_url: https://api.openai.com/v1
    model: text-embedding-3-small
    dimension: 1536
    api_key: sk-...
```
- **ONNX** (default) — no block needed; the built-in default applies. Write the block only if the user explicitly chose ONNX and wants it visible:
```yaml
mnemonic:
  embedder:
    provider: onnx
    model: nomic-embed-code
    dimension: 768
```
- **Off** — write the block to disable vector search explicitly:
```yaml
mnemonic:
  embedder:
    provider: off
```
- **Precedence note:** this repo-local file takes precedence over `~/.skillgrid/config.d/indexing.yaml` (home-local). The home-local file is a per-key fallback — it supplies any key the repo-local file leaves unset. Machine-specific endpoints (e.g. a non-default Ollama host) can live in the home-local file to avoid committing them. Mention this to the user if they express concern about committing a base_url.
- If the user chose the default (ONNX) and no `indexing.yaml` exists in the repo, skip the write — the built-in default applies without a file.

**9. Commit:**
- `git add .skillgrid/config.yaml` + `.skillgrid/state.yaml` + the `.skillgrid/artifacts/` files + the AGENTS.md file
- If `indexing.yaml` was written: `git add .skillgrid/config.d/indexing.yaml`
- `git commit -m "chore: add Skillgrid config + state + artifacts"`

**10. Project init (deterministic finish):**
- Run `skillgrid init` from the project root (add `--force` only if the user asked to rebuild the preamble). `skillgrid init` renders the rich AGENTS.md preamble (`templates/agents-preamble.md`) from the `agents:` block you just filled **and scaffolds `.skillgrid/ARCHITECTURE.md`** from `templates/architecture.md` — the deterministic finish that produces the `# Standards` region and the 17-section architecture skeleton.
- `skillgrid init` only *scaffolds* ARCHITECTURE.md (it writes the skeleton once and keeps an existing file). The **filling** of its `<detect>` markers is onboarding's job (Step 1.7) — do it before you consider onboarding complete, or flag the remaining `<detect>` markers to the user.
- If the user named extra documentation paths, pass each as `--docs <path>`.
- Do not walk files or call `mem_save` for docs here — the CLI owns ingest, the preamble render, the architecture scaffold, and the code index.
- If `skillgrid` is not on PATH, tell the user to install the CLI and re-run `skillgrid init`; fall back to copying `templates/architecture.md` by hand so the zone is complete.

### Step 4: Verify

Show the user:
- The written config (summarized, not full file)
- The `state.yaml` + `artifacts/` zone created
- The AGENTS.md block (with the `### Rules` section rendered from `ASSUMPTIONS.md` § Locked constraints)
- The rendered AGENTS.md preamble (`# Standards` region) — confirm it contains the rich sections with no leftover `<detect>` placeholder when the `agents:` config was filled
- The filled `.skillgrid/ARCHITECTURE.md` — confirm the 17 sections are present and the `<detect>` markers are replaced with verified facts for everything that exists (greenfield: only the sections with real content are filled, the rest are honestly marked)
- "Skills now read from `.skillgrid/config.yaml` + `.skillgrid/state.yaml` + `.skillgrid/ASSUMPTIONS.md` + `.skillgrid/ARCHITECTURE.md` + `.skillgrid/artifacts/`. Run `skillgrid:brainstorming` to start your first project."

## Merge Mode (Re-running)

If `.skillgrid/config.yaml` already exists:
1. Read the existing config
2. Re-detect facts (stack may have changed)
3. Show what changed: "Detected changes: test runner changed from X to Y. Update?"
4. Merge: update changed values, keep user-customized `rules:` sections
5. Re-write AGENTS.md block (idempotent upsert) — re-render `{rules_block}` from the current `ASSUMPTIONS.md` § Locked constraints
6. Reconcile `state.yaml` + the durable-knowledge zone: fill missing keys in `state.yaml` (never reset `pipeline`/`progress`); ensure `ASSUMPTIONS.md` / `ARCHITECTURE.md` exist; if `ARCHITECTURE.md` was scaffolded but still has unfilled `<detect>` markers (a stack change or first-pass gap), re-run Step 1.7 to fill them — never clobber the filled sections, only replace `<detect>` markers with newly detected facts; create any missing `artifacts/` stubs; run the brownfield migration (architecture into the root file; ADR bodies out to `artifacts/04-adr-*.md` with a path row left in `ASSUMPTIONS.md`; `00-prd.md` stays in `artifacts/`; locked constraints and glossary as in the migration bullet) if the project was onboarded before this layout

## What This Does NOT Do

- Does not install the Skillgrid CLI or skills — that's a machine-level concern
- Does not create `.skillgrid/specs/` or `.skillgrid/sdd/` directories — they're created on first use
- Does not write `.skillgrid/config.user.yaml` — there is one config file, committed
- Does not reset `state.yaml` `pipeline`/`progress` on re-run — those are the live project state, not onboarding outputs

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "The config already looks right — skip re-detection" | Stack and test runner drift between runs. Merge mode exists exactly to re-detect and show what changed. |
| "The manifests are obvious — no need to confirm with the user" | Step 2 is explicit: present each fact, confirm one at a time. A wrong `tdd` or tracker value silently breaks every later phase. |
| "I'll bootstrap the tracker after onboarding is done" | If `ticketing.enabled: true`, the CLI must be verified during onboarding (`gh --version`, `backlog status`, etc.). A tracker that isn't verified can't be trusted later. |
| "Both AGENTS.md and CLAUDE.md exist — write the block to both" | Two full blocks are two sources that drift. Write full to `AGENTS.md`, one-line pointer in `CLAUDE.md`. |
| "I'll add a bunch of conventions to the block up front so the agent doesn't get anything wrong" | The block ships with every prompt. Grow it only on an observed failure (observe → fix); split the rest into the referenced files. See *Keep the AGENTS.md block lean*. |
| "Let me read the whole repo and write a comprehensive AGENTS.md" | "Read the repo and generate a giant AGENTS.md" yields a verbose, hard-to-use file. Start minimal (a couple sentences + references) and let it grow on evidence. |
| "I'll hand-write the ARCHITECTURE.md skeleton" | `skillgrid init` scaffolds the 17-section structure deterministically from `templates/architecture.md`. Onboarding's job is to *fill* the `<detect>` markers with verified facts — not to re-invent the skeleton. Hand-writing it drifts from the template every project. |
| "I'll pad every ARCHITECTURE.md section to make it look complete" | A section with no real content stays `<detect>` or is dropped — padding a brownfield/greenfield doc with generic prose makes it *less* trustworthy. Facts over completeness. |
| "The user said 'looks good' — skip re-reading their corrections" | "Looks good" is only valid once every detected fact has been presented and confirmed individually. |

## Red Flags

**Never:**
- Guess a test runner — detect from manifests, then confirm
- Overwrite a user-customized `rules:` section
- Skip the confirmation step — always present detected facts
- Create both AGENTS.md and CLAUDE.md — pick the first that exists, or create AGENTS.md
- Hard-code paths that the config already defines
- Inline convention/how-to content into the `## Skillgrid` block — it is a navigation spine; point at the referenced files instead. The Issue Tracker line names the one convention file for `ticketing.type` (see `block.md`); do not restate that file
- Add a "getting started / setup" section to the block, or point it at individual churning source paths

## Verification

- [ ] `.skillgrid/config.yaml` exists and parses (`yaml` load succeeds); every detected field (stack, testing, commands, ticketing, TDD, commit discipline, BDD) is filled
- [ ] `.skillgrid/state.yaml` exists, parses, and has `pipeline` + `progress` + `constraints_ref` pointing at `.skillgrid/ASSUMPTIONS.md` (existing values preserved on re-run)
- [ ] `.skillgrid/ASSUMPTIONS.md` exists with the four tiers; `.skillgrid/ARCHITECTURE.md` exists, was scaffolded by `skillgrid init` (or copied by hand), and its `<detect>` markers are filled with verified facts for everything that exists (brownfield: all 17 sections addressed; greenfield: only sections with real content filled, the rest honestly left `<detect>` or dropped); every cited `file:line` was re-verified against the code
- [ ] AGENTS.md (or CLAUDE.md) contains exactly one `<!-- skillgrid:start -->` … `<!-- skillgrid:end -->` block with the correct `{project}`, tracker/memory lines, and a `### Rules` section rendered from `ASSUMPTIONS.md` § Locked constraints
- [ ] The block is lean: no inlined conventions/how-to or "getting started" section. Issue Tracker is the exact `{tracker_line}` for `ticketing.type` (a pointer to that convention file). No pointers at churning source paths. If both exist, `CLAUDE.md` holds only a one-line pointer to `AGENTS.md`
- [ ] Detected test runner command runs and exits 0 (e.g. `go test ./...`, `npm test`, `pytest`)
- [ ] Tracker CLI verified if `ticketing.enabled: true` (`gh --version` / `glab --version` / `jira config` / `backlog status`); skipped if `false`. If `ticketing.type: backlogmd`, every `.skillgrid/artifacts/04-adr-*.md` has a `.backlog/decisions/decision-NNN` copy
- [ ] `git rev-parse --is-inside-work-tree` succeeds; onboarding artifacts are committed (`git log -1` shows `chore: add Skillgrid config + state + artifacts`)
- [ ] If Mnemonic is enabled and a non-default embedder was chosen, `.skillgrid/config.d/indexing.yaml` exists with a valid `mnemonic.embedder` block (provider matches the user's choice; base_url/model/dimension are filled for ollama/external)
- [ ] If the project is brownfield (existing feature code + history), Step 1.5 ran: existing capabilities are recorded as `existing`, conventions were reverse-engineered into the block + glossary, and the first slice is planned against the existing system (reuse, not regenerate)
