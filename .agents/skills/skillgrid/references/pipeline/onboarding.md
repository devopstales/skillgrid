# Onboarding Process

## Detect

**Git worktree guard:** before any artifact write, run `git rev-parse --is-inside-work-tree`. If it fails, run `git init` (respects monorepos — never nest a repo inside one). All artifacts then land in a tracked repo.

**Project name:**
- Existing `AGENTS.md` / `CLAUDE.md` / `.skillgrid/config.yaml` → read it
- `git remote get-url origin` → extract repo name
- Fallback: directory name
- If neither: ask the programmer
- If inconsistent: ask the programmer

**Stack & context:**
- Read manifests: `package.json`, `go.mod`, `pyproject.toml`, `Cargo.toml`, `Makefile`, `Taskfile.yml`
- Read CI config: `.github/workflows/`, `.gitlab-ci.yml`, `Jenkinsfile`
- Compose a one-line summary: stack + purpose + constraints
- Keep it ≤ 5 lines

**Commands:**
- Build: `package.json` → `scripts.build`; `go.mod` → `go build ./...`; `Makefile` → `make build`
- Lint: `package.json` → `scripts.lint`; `go.mod` → `golangci-lint run`; `pyproject.toml` → `ruff check`
- Format: `package.json` → `scripts.format` or `prettier --write .`; `go.mod` → `go fmt ./...`
- Typecheck: `package.json` → `scripts.typecheck` or `tsc --noEmit`; `go.mod` → `go vet ./...`

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
- If not available: warn the programmer by printing "WARNING: trivy is not installed."
- Defaults: `severities: "CRITICAL"`, `scan_types: "vuln,secret,misconfig"`, `fail_on: "CRITICAL"`

**Research sources (tooling budget):**
- Optional research CLIs (transcript extractors, login-walled-platform readers,
  structured-data fetchers) are budgeted per
  `_shared/rules/adoption-decisions.md` § Tooling budget: their allow-list is
  the `research.sources` list in `.skillgrid/config.yaml`, and onboarding
  detects each as **non-fatal** — skip if the binary is on PATH, warn if its
  package manager (e.g. `uv`) is missing, and record the detected tools in
  `research.sources`. An absent tool degrades the research stack; it does not
  fail onboarding. Detect, don't force.

**Ticketing:**
- `git remote get-url origin` → contains `github` → `gh`
- `git remote get-url origin` → contains `gitlab` → `glab`
- No remote or programmer preference → `backlogmd` (default, local)
- Ask the programmer: "The following ticketing was detected: <tracker-type>. Do you want to override it? Answer no to continue, or name the tracker to override."
    - If no keep detected
    - Otherwise use programmer answer. Validate before use:
        - Programmer names Jira → `jira` (ask for instance URL + project key)
        - Programmer names Gitlab → `glab`
        - Programmer names Github → `gh`
        - Programmer names backlog → `backlogmd`
        - Programmer names backlog.md → `backlogmd`
        - Programmer names local → `backlogmd`
- Ask the programmer: "Do you want to publish tickets to a tracker? Or work local-only from tasks.md?"
  - If yes → `ticketing.enabled: true`, use detected tracker type
  - If no → `ticketing.enabled: false`, use `backlogmd` tracker type

**Commit discipline:**
- Always on — `skillgrid:work-unit-commits` is fully active during execution
- Ask the programmer: "Commit mode — (1) commit-only, (2) commit + session events (recommended)?"
  - (2) → install the `pre-commit` + `commit-msg` hook shims; each work-unit commit feeds a `commit` event
  - (1) → conventional commits + guards only, no session event detail

**ADR style:**
- Detect an existing `adr_style` in `.skillgrid/config.yaml` (if the file exists) → use it
- No existing value → default `madr-minimal`, but ask the programmer which they prefer (see Step 2)

**BDD / Acceptance testing:**
- Always `enabled: true` — BDD is non-negotiable in Skillgrid (like TDD)
- Detect an existing `bdd.stack` in `.skillgrid/config.yaml` (if the file exists) → use it
- No existing value → default `stack: javascript` (cucumber-js)

**Mnemonic:**
- Check if the `skillgrid` CLI is available: `command -v skillgrid`
- If available, verify the MCP server can start: `skillgrid mcp --help`
- If not available: note and warn the programmer "Mnemonic not detected — memory, code index, and web cache will be inactive"
- If available: `mnemonic: enabled: true` in config

**Embedder (mnemonic vector search):**
- Only relevant when Mnemonic is enabled
- Detect existing config: check `.skillgrid/config.yaml` for an `mnemonic.embedder:` block. If present, use it as the starting point and confirm with the programmer.
- Ask the programmer: "How should Mnemonic embed code for vector search? Options:
  1. **Ollama (local LLM server)** — run embeddings on a local Ollama instance. Requires Ollama installed and running.
  2. **ONNX (built-in)** — pure-Go ONNX runtime, no external server. Model auto-downloaded to `~/.skillgrid/cache/models/`.
  3. **External API** — any OpenAI-compatible `/embeddings` endpoint (OpenAI, OpenRouter, LiteLLM, etc.)
  4. **Off** — no vector leg; hybrid search falls back to FTS + signals only.
  Default: ONNX (no setup required). Which do you want?"
- If **Ollama**:
  - Verify Ollama is running: `curl -sf http://localhost:11434/api/tags | head -5`
  - If not reachable: "Ollama not running on localhost:11434. Is it on a different host/port? If not installed, you can start with `brew install ollama` (macOS) or `curl -fsSL https://ollama.com/install.sh | sh` (Linux). Install the embedding model with `ollama pull nomic-embed-code`."
  - Ask for base URL (default `http://localhost:11434`) and model name (default `nomic-embed-code`)
- If **External API**:
  - Ask for base URL (e.g. `https://api.openai.com/v1`), model name (e.g. `text-embedding-3-small`), and API key
  - API key: ask the programmer to provide it, or note that it can be set later in `~/.skillgrid/config.yaml` (machine-local, not committed)
- If **ONNX** or **Off**: no further config needed

**Chain strategy**
- Detect an existing `chain.strategy` in `.skillgrid/config.yaml` (if the file exists) → use it
- If it exists, validate it with the programmer
- If it does not exist, ask the programmer: "What chain strategy should we use?" — the three options (`stacked-to-main`, `feature-branch-chain`, `size:exception`) are defined in `_shared/rules/chain-strategy.md`; present those, store the pick in `config.yaml` `chain.strategy`.

## Step 1.5: Enroll (brownfield)

If Step 1 shows an existing codebase (real commit history + feature code, not just a scaffold), enroll what's already there instead of planning it from zero:

1. **Detect existing capabilities.** From the source tree, routes, manifests, and git history, list the features/capabilities that already exist (a few lines each — what it does, where it lives).
2. **Reverse-engineer conventions.** Capture the project's actual conventions (error handling, naming, layering, test style) into the terms files (`glossary/01-business-terms.md` / `02-technical-terms.md`) and `projectwiki/concepts/` pages, from the code — not from assumptions about the stack. If a convention is surprising or undocumented, flag it for the programmer rather than guessing.
3. **Nested AGENTS.md note.** Area-specific rules (a payments module, a UI kit) live in `<area>/AGENTS.md` — the root block stays global and lean. Note the areas that would warrant their own file; create them lazily (ship's reconcile creates one for a net-new area, never for a pre-existing undocumented one — it flags instead).

A greenfield project (scaffold, no feature history) skips Step 1.5 — there is nothing to enroll.

## Step 1.5b: Bootstrap (brownfield)

After Step 1.5 reverse-engineers the conventions, **bootstrap them into the project wiki as checkable drafts** so a future session inherits them.

1. **Create draft candidates.** For each enrolled convention, create a candidate concept per the `skillgrid:projectwiki` contract with `status: draft` and `verification.anchors` — stable literals from the code the convention is observed in (`{ path, all_of: [...] }`). A draft without `verification.anchors` cannot be confirmed against the code and must not be written.
2. **Confirm one at a time.** Present each draft individually — the convention, anchor file(s), quoted literal: "This convention: X, anchored at `path:line`. Confirm?" Only an explicit yes promotes it (`status: stable` + `verified` attestation).
3. **Drop what isn't confirmed.** An unconfirmed draft is **dropped** — removed, never kept as a `draft` zombie; the default outcome is drop.

Greenfield projects skip Step 1.5b — nothing enrolled to bootstrap.

## Step 1.7: Generate ARCHITECTURE.md

Scaffolds `.skillgrid/projectwiki/ARCHITECTURE.md` from `references/patterns/architecture.md` — a 17-section skeleton with `<detect>`/`<detect: …>` markers and a ToC. Your job is to **fill the markers with detected facts** so the document is a faithful, verifiable map of the codebase — not a generic template. Also creates the wiki directories (`raw/`, `sources/`, `entities/`, `concepts/`, `syntheses/`), seeds `index.md` with the detected facts (stack, entry points, key directories), and creates the first `log.md` entry. This is the onboarding deliverable that makes the repo *legible*: a newcomer (or the agent on a fresh session) reads `ASSUMPTIONS.md` and `ARCHITECTURE.md` first, then `index.md` to drill into the relevant pages.

**Principles**
1. **Facts, not opinions.** Every diagram, table row, and code snippet must correspond to something real — a package, a `file:line`, a config key, a port. If you cannot verify it from the code, leave it `<detect>` rather than invent it.
2. **Point at ADRs, don't re-argue them.** Architecture *decisions* live one per file in `.skillgrid/projectwiki/decisions/NNNN-slug.md`, indexed in `ASSUMPTIONS.md` § LOCKED and `index.md`. The `concepts/` pages describe *how it is built* and link to the ADR that pins a rule; they do not restate the tradeoff. (See `_shared/adr-records/README.md`.)
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

## Step 2: Present & Confirm

Show the detected facts to the programmer. Confirm one at a time — don't dump everything at once:

1. **Project name** — "Detected project: X. Correct?"
2. **Stack & context** — "I see a Go 1.22 project with tests. Here's the context I'll write: [show]. Does that capture it?"
3. **Testing** — "Test runner: `go test ./...`. Setup: `go mod download`. TDD: always on. Correct?"
4. **Commands** — "Build: `go build ./...`. Lint: `golangci-lint run`. Format: `go fmt ./...`. Typecheck: `go vet ./...`. Correct?"
5. **AGENTS.md preamble** — "I'll fill the repo-specific AGENTS.md preamble (Key Directories, Security Boundaries, Dependency Policies, Architecture Constraints, Definition of Done, Engineering Standards) from what I detected in the repo. Here's the draft: [show the `agents.*` bullets/rows]. Correct?" (greenfield: "This is a greenfield project — I'll leave the repo-specific preamble sections empty for now; they fill in as the code takes shape. OK?")
6. **Ticketing** — "Do you want to publish tickets to a tracker? I detected GitHub — I'd use `gh`. Or Backlog.md (local), GitLab, Jira, or none (tasks.md only)?"
7. **ADR style** — "How should I format ADRs? Options: `madr-full` (detailed tradeoff record), `madr-minimal` (context/options/decision/consequences — recommended default), `nygard` (classic status/context/decision/consequences), `y-statement` (one sentence), or `custom` (your house format). I'll store the choice and never re-ask."
8. **Artifact paths** — "Durable knowledge lives in the `.skillgrid/projectwiki/` LLM-wiki: `ASSUMPTIONS.md` (project understanding + ADR index), `ARCHITECTURE.md` (live repo structure), `glossary/` (terms), `invariants/` (locked constraints), `raw/` (immutable source docs), `sources/` (summaries), `entities/` (things the program touches), `concepts/` (invariants, patterns, error model), `decisions/` (one ADR per file), `syntheses/` (cross-entity analyses). `index.md` is the catalog; `log.md` is the timeline. Prototypes (feasibility probes, permanently retained): `.skillgrid/prototype/`. State (dynamic: phase, current change, progress): `.skillgrid/state.yaml`. Active specs: `.skillgrid/change-specs/`. Closed changes (single sink, immutable): `.skillgrid/change-archive/`. Live execution scratch (in-flight ledgers only, gitignored): `.skillgrid/work/`. Worktrees: in project root `.worktree/`. Any overrides?"

9. **Domain model** — "The ADR templates and contract live in `_shared/adr-records/` (selected by `adr_style` in config — the table is in `_shared/adr-records/README.md`). The architectural-decision-records skill maintains the project vocabulary in `projectwiki/glossary/01-business-terms.md` + `02-technical-terms.md` (from the shared business/technical templates) and the ADR trail as `projectwiki/decisions/04-adr-NNNN-slug.md`, indexed in `ASSUMPTIONS.md` § LOCKED and `projectwiki/index.md`. ADRs use the `{adr_style}` format you just picked. Onboarding seeds the wiki from the detected facts and `references/patterns/glossary.md`; the ADR skill maintains it from then on. Operational rules (the hard limits a change must respect) are owned by the AGENTS.md `### Rules` section. OK?"

10. **BDD / Acceptance testing** — "Executable acceptance tests: always on (like TDD). Stack: `javascript` (cucumber-js). Specs live at `.skillgrid/change-specs/<id>/acceptance.feature`; the runner at `acceptance-tests/` is scaffolded on first use. The acceptance scenarios ARE the executable spec. Correct?"

11. **QA gates** — "The `skillgrid:qa` skill enforces hard quality gates before merge. Here are the defaults — adjust any of them:
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

12. **Rigor tier default** — "How much verification does a typical change in this project want? Tiers: T0 Prototype (self-check only), T1 Alpha (light blueprint + verify), T2 Beta (full pipeline + two-axis review), T3 GA (mutation + security + parallel fresh-model review). I recommend **{recommended tier}** for {one-line reason from the stated product's risk/size — e.g. 'a user-facing product with payments'}. Any change can override per-change in its briefing (`Tier: T<n>`)." Store in `rules.tiers.default`. Never re-ask once set.

13. **Mnemonic** — (only when the `skillgrid` CLI was detected) "Persistent memory detected (skillgrid CLI). Memory, code index, and web cache will be active. Artifacts saved to Mnemonic survive /clear and branch switches. OK?" — (when not detected) "Mnemonic not detected (skillgrid CLI not found). Memory, code index, and web cache will be inactive until the CLI is installed."

14. **Embedder** — (only when Mnemonic is enabled) "Vector search embedder: **{provider}** ({one-line description}). Write the `mnemonic.embedder:` block to `.skillgrid/config.yaml`? — (ollama) "provider: ollama, base_url: {url}, model: {model}" — (external) "provider: external, base_url: {url}, model: {model}, api_key: {key}" — (onnx) "provider: onnx, model: nomic-embed-code (auto-downloaded)" — (off) "no vector leg; FTS + signals only". Correct?"

If the programmer corrects anything, use their value. If they say "looks good" or "yes", move on.


## Next

Continue with `references/pipeline/onboarding-write.md` for Step 3 (write) and Step 4 (verify).
