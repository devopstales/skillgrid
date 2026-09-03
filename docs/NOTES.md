
# Notes

## Validate

* Excalidraw
* v0
* deep research
* deepseak harness
* Backlog.md
* beads
* chloe


## Selected

* [X] skills
* [X] @cucumber/cucumber
* mcp-installer
* [?] engram
* [ ] agent-browser
* [ ] playwright
* contextmode
* girafe

## Agent Configs

* [ ] kilo
  * `~/.kiro/settings/mcp.json`
  * `~/.config/kilo/kilo.jsonc`
* [ ] opencode
  * `~/.config/opencode/opencode.json`
* cursor
* claud
* codex
* gemini cli
* antigravity

## Skills

* ponitail
* [unlaz](https://github.com/Leonxlnx/unlazy/tree/main)

## Tools

### Agent Orchectration

* [ ] [opencode-4hol](https://dev.to/uenyioha/porting-claude-codes-agent-teams-to-opencode-4hol)
* [ ] [Cleave](https://cleave.dev/)
* TEAMS:
  * https://docs.cline.bot/sdk/guides/multi-agent-teams
  * https://dev.to/uenyioha/porting-claude-codes-agent-teams-to-opencode-4hol
  * https://github.com/hueyexe/opencode-ensemble

### Memory

* [Openclow SQlight Memory](https://www.pingcap.com/blog/local-first-rag-using-sqlite-ai-agent-memory-openclaw/)
* [X] [codegraph](https://github.com/colbymchenry/codegraph)
* [?] [srclight](https://github.com/srclight/srclight) - 42 mcp tools
  * [Docs](https://dev.to/tofutim/how-we-built-a-hybrid-fts5-embedding-search-for-code-and-why-you-need-both-4ec2)
* [?] [codebase-memory](https://github.com/DeusData/codebase-memory-mcp)
* [?] [mcp-injector](https://github.com/foldwork-dev/mcp-injector)


* [?] [CocoIndex-Code](https://github.com/cocoindex-io/cocoindex-code)
* [?] [GitNexus](https://github.com/abhigyanpatwari/GitNexus)
* [?] [graphify](https://github.com/Graphify-Labs/graphify)
* [?] [codebase-index](https://github.com/denfry/codebase-index)
* [?] [foldwork](https://github.com/foldwork-dev/mcp-injector) - [Docs](https://www.foldwork.dev/)

* [Local-First Documentation](https://neuledge.com/blog/2026-02-19/local-first-documentation-for-ai)

* [?] [deepseek-harness](https://github.com/deepseek-ai/deepseek-harness)
* [X] [Engram](https://github.com/Gentleman-Programming/engram)
* [X] **Skillgrid Mnemonic** — v1 merged [PR #2](https://github.com/devopstales/skillgrid/pull/2) to `release/2` ([spec](superpowers/specs/2026-08-26-skillgrid-mnemonic-design.md)); replaces Engram when `indexing.profile: mnemonic`

### Spec driven development

* PRD
* [-] [openspec](https://openspec.dev/)
* [X] [Superpowers](https://github.com/obra/superpowers)

### Ticketingt

* Atlassian MCP
* GitHub MCP
* Gitlab MCP
* Vercel

### checkoint

* Superpowers - uses commits

### Design Tools

* design.md
* [ ] [taste-skill](https://github.com/Leonxlnx/taste-skill)
* [ ] [npxskillui](https://github.com/amaancoderx/npxskillui)
* [ ] [impeccable](https://github.com/pbakaus/impeccable)
* [ ] [open-design](https://github.com/nexu-io/open-design)
* [ ] [kombai](https://kombai.com/)
* [ ] [penpot](https://penpot.app/self-host#options)
* [ ] [penpot-desktop](https://github.com/author-more/penpot-desktop/wiki/Installation)

### Usage Data

* [ ] [context-mode](https://github.com/mksglu/context-mode)
* [ ] [gryph](https://github.com/safedep/gryph)


### Testing

* [X] playwright
* [X] agent-browser
* [ ] [cucumber](https://cucumber.io/docs/cucumber/)
* [ ] [gherkin](https://cucumber.io/docs/gherkin/reference)

### Security

* [ ] Trivy
* [ ] [secure-rules](https://github.com/TikiTribe/claude-secure-coding-rules/tree/main)

## Plugins

```
npm install superpowers@git+https://github.com/obra/superpowers.git --prefix "$HOME/.config/kilo"

npm install superpowers@git+https://github.com/obra/superpowers.git --prefix "$HOME/.config/opencode"

{
  "plugin": ["~/.config/opencode/node_modules/superpowers"]
}
```

* `https://github.com/kdcokenny/opencode-background-agents`

## Skills

* [X] skills

```
skills add obra/superpowers --agent amp -g -s '*' -y
skills add gentleman-programming/engram --agent amp -g -s 'engram-memory' -y
skills add gentleman-programming/engram --agent amp -g -s 'engram-memory-protocol' -y
skills add gentleman-programming/engram --agent amp -g -s 'engram-testing-coverag' -y
```

* SDD - Spec-Driven Development
* TDD - Test-Driven Development
* DDD - [Behaviour-Driven Development](https://cucumber.io/docs/bdd) - [Behaviour-Driven-Template](https://github.com/intent-driven-dev/behavior-driven-template)
* IDD - [intent-driven-template](https://github.com/intent-driven-dev/intent-driven-template/tree/main)

## Rules

Copy `~/.skillgrid/config.d/AGENTS.md` to `~/.agents/AGENTS.md` and add it to the configs:

* `~/.config/kilo/kilo.jsonc`
* `~/.config/opencode/opencode.json`
* `https://github.com/obra/superpowers/blob/main/CLAUDE.md`
* `https://github.com/multica-ai/andrej-karpathy-skills/blob/main/CLAUDE.md`

## Installers

* bash script
  * use prebuilt binary
* brew
* nix flake

# Usage

## Init

* init subcommand
  * force project level code index
* project init by skill and command
  * generate project leve AGENTS.md

### What to Include into AGENTS.md

Keep this file short, but make sure it covers the full working agreement:

* Project overview: one or two lines on what the project is and what kind of work the agent is doing.
* Environment and tooling: language version, package manager, run commands, virtual environment names, test commands, and lint or format commands.
* Engineering standards: expectations for tests, error handling, code quality, and reviewable diffs.
* Security and escalation boundaries: where the agent may work, what it must not access, and which actions require approval.
* Dependency policies: which dependency tools are allowed, whether the standard library is preferred, and when new packages need approval.
* Architectural constraints: design choices that are frozen unless a human approves a change.
* Definition of done: the checks that must pass before the work is complete.

Here is a concise example:

```md
# Project Overview
CLI tool for anomaly detection on time-series CSVs.

# Environment & Tooling
- Python 3.12+
- Dependency management: uv
- Dependencies defined in pyproject.toml
- Commit uv.lock
- Run: uv run python -m app
- Tests: uv run pytest
- Lint/format: ruff check . && ruff format .

# Engineering Standards
- Add or update tests for behavior changes
- Handle invalid input explicitly
- Keep diffs small and reviewable

# Security & Escalation Boundaries
- Work only in this repository
- No network access except the model connection
- Do not read secrets or credential stores
- Ask before installing dependencies, deleting files, or changing auth logic

# Dependency Policies
- Prefer the standard library
- New packages require approval
- Do not use pip or conda

# Architecture Constraints
- Single-process CLI application
- No external database
- No network calls

# Definition of Done
- Tests pass
- Lint and formatting pass
- Edge cases are covered
- No new warnings are introduced
- Update the spec if behavior changes
```
