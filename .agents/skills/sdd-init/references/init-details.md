# SDD Init Details

## Detection Checklist

### project_name

1. Existing `AGENTS.md`/`CLAUDE.md`/`GEMINI.md` — explicit `project_name` or repo title.
2. `openspec/config.yaml` `context:` or `project:` field.
3. Mnemonic `sdd-init/{project}` observation.
4. `git remote -v` owner/repo — derive short repo name.
5. Working directory basename (last resort, confirm with user).

### tech_stack

Inspect in parallel:

- JS/TS: `package.json` (name, scripts, deps, devDeps, workspaces), `tsconfig.json`, `pnpm-workspace.yaml`.
- Go: `go.mod` (module path, go version, require block).
- Python: `pyproject.toml`, `setup.py`, `requirements*.txt`, `Pipfile`.
- Rust: `Cargo.toml`. Java: `pom.xml`, `build.gradle`.
- CI: `.github/workflows/*.yml`, `.gitlab-ci.yml`, `Jenkinsfile`, `.circleci/config.yml`.
- Runtime: `Dockerfile`, `docker-compose.yml` — deployment target, node/python/go versions.

Summarize as: language + major framework versions + package manager + runtime/deploy hints.

### testing_capabilities

- Test runner: `package.json` `scripts.test`, devDeps; `pyproject.toml` `[tool.pytest]`; `pytest.ini`; `go.mod` (`go test`); `Cargo.toml` (`cargo test`); `Makefile` test targets.
- Test layers:
  - Unit: `vitest`, `jest`, `pytest`, `go test`, `tokio::test`.
  - Integration: `testing-library`, `httpx`, `httptest`, `supertest`, `WebApplicationFactory`.
  - E2E: `playwright`, `cypress`, `selenium`, `chromedp`, `puppeteer`.
- Coverage: `vitest --coverage`, `jest --coverage`, `c8`/`nyc`, `pytest-cov`, `go test -cover`, `cargo-llvm-cov`.
- Quality: linter + version and command, type checker command, formatter command (`.biome.json`, `.eslintrc*`, `ruff.toml`, `golangci.yml`, `prettier`, `gofmt`, `black`, `rustfmt`).

Format per the Testing Capabilities template below.

### issue_tracker

1. `AGENTS.md`/`CLAUDE.md`/`GEMINI.md` `## Agent skills → Issue tracker` line (or `<!-- skillgrid-sdd:start/end -->` block).
2. `docs/agents/issue-tracker.md` existing content.
3. Mnemonic `sdd/{project}/issue_tracker`.
4. Git remote host: GitHub → `gh`, GitLab → `glab`, other → default **Backlog.md**.
5. Jira → `jira` CLI ([jira-cli](https://github.com/ankitpokhrel/jira-cli)): detected when `jira config` resolves an instance + project key (from `jira init`), or the user names an instance + key. Project key is mandatory — capture it.

Tracker identifiers: github=gh, gitlab=glab, jira=jira, backlogmd=backlog.

## Issue Tracker Options

| Tracker | CLI | Storage |
|---|---|---|
| Backlog.md | `backlog` | `.backlog/tasks/<ID>.md` |
| GitHub | `gh` | repo Issues |
| GitLab | `glab` | repo Issues |
| Jira | `jira` | instance project key (`jira init`) |

Seed templates live in `../_shared/issue-tracker/` (`backlogmd.md`, `github.md`, `gitlab.md`, `jira.md`).

Conventions doc: `docs/agents/issue-tracker.md`. Triage labels: `_shared/triage-labels.md`. `issue-creation` skill maps tasks → tracker issues with duplicate-search first.

## Skill Registry Scan Rules

Registry at `docs/agents/skill-registry.md` is an **index** (index → exact path), not a summary. Sub-agents read the full `SKILL.md` from the path.

Scan user skills in: `~/.agents/skills/`, `~/.config/kilo/skills/`, `~/.claude/skills/`, `~/.gemini/skills/`, `~/.cursor/skills/`, `~/.config/agents/skills/`.

Scan project skills in: `{root}/.agents/skills/`, `{root}/.claude/skills/`, `{root}/.github/skills/`, `{root}/skills/`.

Rules:
- Skip `sdd-*`, `_shared`, `skill-registry` entries from the registry listing (they are workflow machinery, not project skills).
- Deduplicate by skill name, preferring project-level over user-level.
- Per skill extract: `name`, trigger text (frontmatter `description`), full `SKILL.md` path, scope (project|user).
- Also index project convention files: `AGENTS.md`, `CLAUDE.md`, `.cursorrules`, `GEMINI.md`, `copilot-instructions.md`, `CONTRIBUTING.md`.

Registry entry format:

```markdown
| name | trigger | path | scope |
|---|---|---|---|
| foo | "Use when …" | ~/.agents/skills/foo/SKILL.md | user |
```

## OpenSpec Skeleton

Create if `openspec/` is absent:

```
openspec/
├── config.yaml
├── specs/
└── changes/
    └── archive/
```

`config.yaml` — keep `context:` ≤ 10 lines:

```yaml
project: <project_name>
context: >
  <one-line stack summary. One-line purpose. One-line constraints.>
issue_tracker: backlogmd | gh | glab | jira
issue_tracker_project_key: <KEY>  # required when issue_tracker: jira, else omit
testing:
  runner: <command>
  layers: [unit, integration]
  coverage: <command or null>
  strict_tdd: true | false
conventions:
  artifacts_dir: openspec
  scratch_dir: .skillgrid/sdd
  registry: docs/agents/skill-registry.md
```

## Agent Config Block

The payload and all target rules live in the shared template family — do **not** keep a copy here:

- [`../../_shared/agent-config/block.md`](../../_shared/agent-config/block.md) — the canonical `## Agent skills` block, placeholder table, and sentinel upsert. **Render this verbatim.**
- [`../../_shared/agent-config/README.md`](../../_shared/agent-config/README.md) — which target file gets the full block (decision matrix), multi-platform pointer rule, shared rules.
- Per-target placement: [`../../_shared/agent-config/agents.md`](../../_shared/agent-config/agents.md) · [`../../_shared/agent-config/claude.md`](../../_shared/agent-config/claude.md) · [`../../_shared/agent-config/gemini.md`](../../_shared/agent-config/gemini.md).

Targets in order (first existing wins as primary): `AGENTS.md` → `CLAUDE.md` → `GEMINI.md`. If none exists, ask the user which to create (suggest `AGENTS.md`).

The full block (for reference — the shared file is authoritative):

```markdown
<!-- skillgrid-sdd:start -->
## Agent skills

Skillgrid SDD is active in this repo. The workflow, registry, and tracker below are the source of truth for agent work here.

### Workflow
`init → explore → propose → design → spec → tasks → issue-creation → apply → verify → archive`

- Skill registry (index of installed skills + triggers): `docs/agents/skill-registry.md`
- Project facts (stack, testing, tracker, conventions): `openspec/config.yaml` and Mnemonic (`sdd/{project}/…`)
- Triage labels: `docs/agents/issue-tracker.md` + the tracker's label map

### Issue tracker
{one-line tracker summary — see block.md placeholder table}. See `docs/agents/issue-tracker.md`.
<!-- skillgrid-sdd:end -->
```

If the `<!-- skillgrid-sdd:start/end -->` markers already exist in the target, replace that range in place; never create a second root config file or a duplicate block. Secondary targets (other platform files present) get only the one-line pointer from the shared README.

## Testing Capabilities Format

```markdown
## Testing Capabilities

**Strict TDD Mode**: {enabled/disabled}
**Detected**: {date}

### Test Runner

- Command: `{command}`
- Framework: {name}

### Test Layers

| Layer       | Available | Tool        |
|-------------|-----------|-------------|
| Unit        | ✅ / ❌   | {tool or —} |
| Integration | ✅ / ❌   | {tool or —} |
| E2E         | ✅ / ❌   | {tool or —} |

### Coverage

- Available: ✅ / ❌
- Command: `{command or —}`

### Quality Tools

| Tool         | Available | Command        |
|--------------|-----------|----------------|
| Linter       | ✅ / ❌   | {command or —} |
| Type checker | ✅ / ❌   | {command or —} |
| Formatter    | ✅ / ❌   | {command or —} |
```

Strict TDD resolution: explicit marker/config wins → test runner exists but no marker ⇒ `strict_tdd: true` → no runner ⇒ `strict_tdd: false`, explain unavailable.

## Mnemonic Saves

Per [`../../_shared/conventions/mnemonic-memory.md`](../../_shared/conventions/mnemonic-memory.md): `scope: project`, active `session_id`, `title == topic_key`. Reuse `topic_key` on re-runs (upsert, do not duplicate).

```text
mem_save topic_key: sdd-init/{project}
  type: architecture
  content: detected project context (name, stack, repo, deploy)

mem_save topic_key: sdd/{project}/issue_tracker
  type: config
  content: tracker id + CLI + storage path + convention doc path

mem_save topic_key: sdd/{project}/testing-capabilities
  type: config
  content: testing capabilities markdown table

mem_save topic_key: skill-registry
  type: config
  content: registry index markdown
```

## Backlog.md Bootstrap

Only when tracker = `backlogmd`:

```
backlog.config.yml            # backlog_directory: .backlog
.backlog/
├── readme.md
└── tasks/                    # one <ID>.md per issue
```

`backlog.config.yml`:

```yaml
backlog_directory: .backlog
labels:
  triage: needs-triage
  info: needs-info
  ready_agent: ready-for-agent
  ready_human: ready-for-human
  wontfix: wontfix
```

## Output Envelope

```text
status: ok
project: <name>
tech_stack: <summary>
issue_tracker: <id>
issue_tracker_project_key: <KEY>   # Jira only; omit for backlogmd/gh/glab
testing_capabilities: <table>
artifacts:
  created: [paths…]
  updated: [paths…]
  mnemonic: [topic_key → observation id…]
validations:
  confirmed_by_user: true
next: "/sdd-explore <change-idea>"
risks: []
```

If `openspec/` pre-existed: `updated` lists touched files, `risks` notes what was preserved.
