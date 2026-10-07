# Onboarding Process (Steps 1–4)

The step-by-step procedure for `skillgrid:onboarding`. Load this only when actually running onboarding. The spine (What It Does, Merge Mode, Rationalizations, Red Flags, Verification) stays in SKILL.md.

## Contents
- Step 1: Detect
- Step 1.5: Enroll (brownfield)
- Step 1.7: Generate ARCHITECTURE.md
- Step 2: Present & Confirm
- Step 3: Write
- Step 4: Verify

## Step 1: Detect

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
- `agents.architecture_constraints` — bullets from the AGENTS.md `### Rules` section (operational rules) + layering docs (layering rules, replication constraints, storage modes). Decisions are not constraints; they are ADRs in `.skillgrid/ASSUMPTIONS.md` § LOCKED.
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

## Step 1.5: Enroll (brownfield)

If Step 1 shows an existing codebase (real commit history + feature code, not just a scaffold), enroll what's already there instead of planning it from zero:

1. **Detect existing capabilities.** From the source tree, routes, manifests, and git history, list the features/capabilities that already exist (a few lines each — what it does, where it lives).
2. **Mark them `existing`.** They are not planned-from-zero and not `planned` — they predate this workflow. Record them in the `## Skillgrid` block's context note (or the briefing of the first change) so `skillgrid:writing-blueprints` designs the next slice against them: the data they hold, the patterns they follow. The first blueprint's Build shape and File Structure start from "what's already there", not from an empty tree.
3. **Reverse-engineer conventions.** Capture the project's actual conventions (error handling, naming, layering, test style) into the `## Skillgrid` block and the terms files (`artifacts/01-business-terms.md` / `02-technical-terms.md`), from the code — not from assumptions about the stack. If a convention is surprising or undocumented, flag it for the user rather than guessing.
4. **Nested AGENTS.md note.** Area-specific rules (a payments module, a UI kit) live in `<area>/AGENTS.md` — the root block stays global and lean. Note the areas that would warrant their own file; create them lazily (ship's reconcile creates one for a net-new area, never for a pre-existing undocumented one — it flags instead).

A greenfield project (scaffold, no feature history) skips Step 1.5 — there is nothing to enroll.

## Step 1.7: Generate ARCHITECTURE.md

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

## Step 2: Present & Confirm

Show the detected facts to the user. Confirm one at a time — don't dump everything at once:

1. **Project name** — "Detected project: X. Correct?"
2. **Stack & context** — "I see a Go 1.22 project with tests. Here's the context I'll write: [show]. Does that capture it?"
3. **Testing** — "Test runner: `go test ./...`. Setup: `go mod download`. TDD: always on. Correct?"
4. **Commands** — "Build: `go build ./...`. Lint: `golangci-lint run`. Format: `go fmt ./...`. Typecheck: `go vet ./...`. Correct?"
5. **AGENTS.md preamble** — "I'll fill the repo-specific AGENTS.md preamble (Key Directories, Security Boundaries, Dependency Policies, Architecture Constraints, Definition of Done, Engineering Standards) from what I detected in the repo. Here's the draft: [show the `agents.*` bullets/rows]. Correct?" (greenfield: "This is a greenfield project — I'll leave the repo-specific preamble sections empty for now; they fill in as the code takes shape. OK?")
6. **Ticketing** — "Do you want to publish tickets to a tracker? I detected GitHub — I'd use `gh`. Or Backlog.md (local), GitLab, Jira, or none (tasks.md only)?"
7. **ADR style** — "How should I format ADRs? Options: `madr-full` (detailed tradeoff record), `madr-minimal` (context/options/decision/consequences — recommended default), `nygard` (classic status/context/decision/consequences), `y-statement` (one sentence), or `custom` (your house format). I'll store the choice and never re-ask."
8. **Artifact paths** — "Durable knowledge lives at the repo root: `.skillgrid/ASSUMPTIONS.md` (project understanding + all architectural decisions/ADRs + locked constraints) and `.skillgrid/ARCHITECTURE.md` (live repo/program structure). Durable artifacts (terms glossary + research findings): `.skillgrid/artifacts/`. Prototypes (feasibility probes, permanently retained): `.skillgrid/prototypes/`. State (dynamic: phase, current change, progress): `.skillgrid/state.yaml`. Specs: `.skillgrid/specs/`. Scratch: `.skillgrid/sdd/`. Worktrees: `.worktrees/`. Any overrides?"

8. **Domain model** — "The architectural-decision-records skill maintains the project vocabulary at `.skillgrid/artifacts/01-business-terms.md` + `02-technical-terms.md` and the ADR trail as `.skillgrid/artifacts/04-adr-NNNN-slug.md`, with a pointer row in the `.skillgrid/ASSUMPTIONS.md` § `LOCKED` ADR index. ADRs use the `{adr_style}` format you just picked. Operational rules (the hard limits a change must respect) are owned by the AGENTS.md `### Rules` section, not `ASSUMPTIONS.md`. Nothing to set up — they appear on first use. OK?"

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

## Step 3: Write

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
- If `.skillgrid/ASSUMPTIONS.md` is absent, create it from the shape in `_shared/rules/sdd-structure.md` — the four tiers `## VERIFIED`, `## INFERRED (HYPOTHESIS)`, `## LOCKED` (the ADR index: one pointer row per in-force decision, path to `artifacts/04-adr-NNNN-slug.md`; `status`/`supersedes`/`date` live in the file's frontmatter, not the row), and `## Open Questions`. Operational rules are NOT a tier here — they live in AGENTS.md's `### Rules` section. On a fresh project the tiers start empty.
- If `.skillgrid/ARCHITECTURE.md` is absent, `skillgrid init` scaffolds it from `templates/architecture.md` (the deterministic structure: header + 17-section skeleton with `<detect>` markers). Onboarding then **fills** it — see *Step 1.7* above. Do not hand-write the 17-section skeleton; let init scaffold it and replace the `<detect>`/`<detect: …>` markers with detected facts. If `skillgrid` is not on PATH, fall back to copying `templates/architecture.md` yourself (filling `{project}`/`{version}`/`{last_updated}` by hand) so the zone is complete.
- If `artifacts/README.md` is absent, create it from the topic-index shape in `_shared/rules/sdd-structure.md`.
- If `artifacts/06-research-findings.md` is absent, create it with an empty `## Findings` list.
- Brownfield migration of the old paths: if `artifacts/00-architecture.md` exists and `ARCHITECTURE.md` does not, fold the architecture into `ARCHITECTURE.md` and `git mv` the old file to `.skillgrid/archive/`. `artifacts/00-prd.md` stays where it is (it is the requirements reference); if `ASSUMPTIONS.md` is missing, create it from `templates/ASSUMPTIONS.md` with the product statement and a link to `00-prd.md`. If ADR bodies are inlined in `ASSUMPTIONS.md` and the matching `artifacts/04-adr-NNNN-slug.md` is missing, write each body out to that file and leave only the path in the `## LOCKED` ADR index. Do not fold ADR files or the PRD back into `ASSUMPTIONS.md`. If a legacy `### Locked constraints` section (or `artifacts/05-locked-constraints.md`) still exists, move those bullets into AGENTS.md's `### Rules` section (the operational rules now live there), then archive/delete the old section and `git mv` `05-locked-constraints.md` to `.skillgrid/archive/`. A legacy `.skillgrid/glossary/` → `01/02-*-terms.md`.
- The terms files, the ADR files, `artifacts/00-prd.md`, and the ARCHITECTURE record are created lazily by their owning skills — onboarding only guarantees the zone + index + research stubs and the `ASSUMPTIONS.md` skeleton exist.

**4. Write AGENTS.md block (idempotent upsert):**
- Render the canonical `## Skillgrid` block from `../../_shared/agent-config/block.md` (fill `{project}`, `{tracker_line}`, `{memory_line}`, `{rules_block}`)
- `{tracker_line}` is copied exactly from the `{tracker_line}` table in that file for `ticketing.type`. It points at that tracker's convention file. Do not paraphrase storage paths, ID rules, or CLI flags into the block.
- `{rules_block}` is owned by AGENTS.md's own `### Rules` section (one `-` bullet per rule) — it is the real source of the operational rules, so edit it there. If AGENTS.md has no `### Rules` section yet, render `No operational rules yet — add them to AGENTS.md's `### Rules` section.` (decisions are ADRs in `ASSUMPTIONS.md` § LOCKED, not rules).
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

## Step 4: Verify

Show the user:
- The written config (summarized, not full file)
- The `state.yaml` + `artifacts/` zone created
- The AGENTS.md block (with the `### Rules` section as the real source of the operational rules; decisions live in `ASSUMPTIONS.md` § LOCKED)
- The rendered AGENTS.md preamble (`# Standards` region) — confirm it contains the rich sections with no leftover `<detect>` placeholder when the `agents:` config was filled
- The filled `.skillgrid/ARCHITECTURE.md` — confirm the 17 sections are present and the `<detect>` markers are replaced with verified facts for everything that exists (greenfield: only the sections with real content are filled, the rest are honestly marked)
- "Skills now read from `.skillgrid/config.yaml` + `.skillgrid/state.yaml` + `.skillgrid/ASSUMPTIONS.md` + `.skillgrid/ARCHITECTURE.md` + `.skillgrid/artifacts/`. Run `skillgrid:brainstorming` to start your first project."
