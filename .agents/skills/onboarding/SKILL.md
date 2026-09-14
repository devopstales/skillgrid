---
name: onboarding
description: Use when setting up Skillgrid for a new project. Detects stack, testing, and tracker; writes .skillgrid/config.yaml and an AGENTS.md block. Run once per project.
# based on skillgrid-v2:sdd-onboard + mattpocock-skills:setup-matt-pocock-skills + BMAD:bmad-project-context
---

# Onboarding

Detect project facts, confirm with the user, write `.skillgrid/config.yaml` and an AGENTS.md block. Run once per project — subsequent runs merge, not overwrite.

**Announce at start:** "I'm using the skillgrid:onboarding skill to configure Skillgrid for this project."

## When to Use

- First time running any Skillgrid skill in a new project (no `.skillgrid/config.yaml` exists)
- User says "set up Skillgrid", "init Skillgrid", "onboard this project"
- Project stack changed significantly (new language, new test runner)

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

**Checkpoint (commit discipline + resume handle):**
- Always `enabled: true` — `skillgrid:work-unit-commits` is fully active during execution
- `state_file` defaults to `<scratch_dir>/checkpoint.json` (`.skillgrid/sdd/checkpoint.json`)
- Ask the user: "Checkpoint mode — (1) commit-only, (2) commit + resume handle (recommended)?"
  - (2) → `checkpoint.enabled: true`, install the `pre-commit` + `commit-msg` hook shims
  - (1) → `checkpoint.enabled: false`, conventional commits + guards only, no `checkpoint.json`

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

### Step 2: Present & Confirm

Show the detected facts to the user. Confirm one at a time — don't dump everything at once:

1. **Project name** — "Detected project: X. Correct?"
2. **Stack & context** — "I see a Go 1.22 project with tests. Here's the context I'll write: [show]. Does that capture it?"
3. **Testing** — "Test runner: `go test ./...`. Setup: `go mod download`. TDD: always on. Correct?"
4. **Commands** — "Build: `go build ./...`. Lint: `golangci-lint run`. Format: `go fmt ./...`. Typecheck: `go vet ./...`. Correct?"
5. **Ticketing** — "Do you want to publish tickets to a tracker? I detected GitHub — I'd use `gh`. Or Backlog.md (local), GitLab, Jira, or none (tasks.md only)?"
6. **ADR style** — "How should I format ADRs? Options: `madr-full` (detailed tradeoff record), `madr-minimal` (context/options/decision/consequences — recommended default), `nygard` (classic status/context/decision/consequences), `y-statement` (one sentence), or `custom` (your house format). I'll store the choice and never re-ask."
7. **Artifact paths** — "Specs: `.skillgrid/specs/`. Scratch: `.skillgrid/sdd/`. Worktrees: `.worktrees/`. PRD: `docs/PRD.md`. ARCHITECTURE: `docs/ARCHITECTURE.md`. Glossary: `.skillgrid/glossary/` (business.md + technical.md). ADRs: `.skillgrid/adr/`. Any overrides?"

8. **Domain model** — "The architectural-decision-records skill maintains a project glossary at `.skillgrid/glossary/` (business.md + technical.md) and an ADR trail at `.skillgrid/adr/`, created lazily as terms and decisions crystallize during brainstorming/interviewing. ADRs use the `{adr_style}` format you just picked. Nothing to set up — they appear on first use. OK?"

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

11. **Mnemonic** — (only when the `skillgrid` CLI was detected) "Persistent memory detected (skillgrid CLI). Memory, code index, and web cache will be active. Artifacts saved to Mnemonic survive /clear and branch switches. OK?" — (when not detected) "Mnemonic not detected (skillgrid CLI not found). Memory, code index, and web cache will be inactive until the CLI is installed."

If the user corrects anything, use their value. If they say "looks good" or "yes", move on.

### Step 3: Write

**1. Write `.skillgrid/config.yaml`:**
- Copy `templates/config.yaml` from this skill's directory
- Fill in all detected values
- If the file already exists, merge: keep existing values the user confirmed, update changed ones
- Do NOT overwrite `rules:` sections the user has customized

**2. Write AGENTS.md block (idempotent upsert):**
- Render the canonical `## Skillgrid` block from `../_shared/agent-config/block.md` (fill `{project}`, `{tracker_line}`, `{memory_line}`)
- Target decision: if `AGENTS.md` exists → update it. Else if `CLAUDE.md` exists → update it. Else → create `AGENTS.md`.
- **Idempotent upsert** (per `../_shared/agent-config/block.md`):
  1. Search the target file for `<!-- skillgrid:start -->`.
  2. **Found** → replace everything from that marker to `<!-- skillgrid:end -->` (inclusive) with the freshly rendered block. Never append a second copy.
  3. **Not found** → append the block at the end of the file.
- **Multi-platform rule:** if both `AGENTS.md` and `CLAUDE.md` exist, write the full block to `AGENTS.md` (source of truth) and put only a one-line pointer in `CLAUDE.md`. Two full blocks = two sources that drift.
- The block is idempotent — running onboarding twice never duplicates it.

**3. Update `.gitignore`:**
- Ensure `.skillgrid/sdd/` is gitignored (scratch, ephemeral)
- Ensure `.skillgrid/brainstorm/` is gitignored (visual companion state)
- Do NOT gitignore `.skillgrid/config.yaml` or `.skillgrid/specs/` (committed artifacts)

**4. Tracker bootstrap (if `ticketing.enabled: true`):**
- If `ticketing.type: backlogmd`: run `backlog init "<project>" --integration-mode cli --backlog-dir .backlog --config-location folder --zero-padded-ids 3`, verify with `backlog status`
- If `ticketing.type: gh`: verify `gh` CLI is available (`gh --version`)
- If `ticketing.type: glab`: verify `glab` CLI is available (`glab --version`)
- If `ticketing.type: jira`: verify `jira` CLI is available (`jira config`)
- If `ticketing.enabled: false`: skip all tracker bootstrap

**5. Mnemonic saves (if `mnemonic.enabled: true`):**
- `mem_session_start(title: "skillgrid-init/{project}")`
- `mem_save` topic_key: `skillgrid-init/{project}`, type: `architecture` — detected project context (name, stack, repo, deploy)
- `mem_save` topic_key: `skillgrid/{project}/issue_tracker`, type: `config` — tracker id + CLI + storage path
- `mem_save` topic_key: `skillgrid/{project}/testing-capabilities`, type: `config` — testing capabilities table
- `mem_session_summary` + `mem_session_end`
- If the session fails (MCP not wired yet), note it and continue — the saves will be made on first `skillgrid:mnemonic` invocation

**6. Commit:**
- `git add .skillgrid/config.yaml` + the AGENTS.md file
- `git commit -m "chore: add Skillgrid config"`

### Step 4: Verify

Show the user:
- The written config (summarized, not full file)
- The AGENTS.md block
- "Skills now read from `.skillgrid/config.yaml`. Run `skillgrid:brainstorming` to start your first project."

## Merge Mode (Re-running)

If `.skillgrid/config.yaml` already exists:
1. Read the existing config
2. Re-detect facts (stack may have changed)
3. Show what changed: "Detected changes: test runner changed from X to Y. Update?"
4. Merge: update changed values, keep user-customized `rules:` sections
5. Re-write AGENTS.md block (idempotent upsert)

## What This Does NOT Do

- Does not install the Skillgrid CLI or skills — that's a machine-level concern
- Does not create `.skillgrid/specs/` or `.skillgrid/sdd/` directories — they're created on first use
- Does not write `.skillgrid/config.user.yaml` — there is one config file, committed

## Red Flags

**Never:**
- Guess a test runner — detect from manifests, then confirm
- Overwrite a user-customized `rules:` section
- Skip the confirmation step — always present detected facts
- Create both AGENTS.md and CLAUDE.md — pick the first that exists, or create AGENTS.md
- Hard-code paths that the config already defines
