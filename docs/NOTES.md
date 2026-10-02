
# Notes


https://youtu.be/v4F1gFy-hqg?si=CddF2O7TE1hPO4tp
https://youtu.be/Vok_nReMFaU?si=r4-LZDy5aVJ7B6ET
https://youtu.be/6kM27uGP4n4?si=0TcMe6Xvu-boL5Hi
https://youtu.be/MbiMwgbGdxw?si=MFckvBJwiaw01woJ

You us grep read eza ls rg. Why you didn't use the skillgrid mcp?

* rename spike to prototype
* all the existing changes
* admin dashboard
* hooks
* agent personas
* prd in artifacts vs ASSUMPTIONS.md
* coding standards, testing, security tests
  * https://github.com/vigolium/vigolium
* hooks
 https://github.com/raultov/opencode-hooks-plugin 
 https://github.com/KristjanPikhof/OpenCode-Hooks 
 https://github.com/shanebishop1/opencode-command-hooks 
* TEAMS
  * .skillgrid/sdd
  * https://docs.cline.bot/sdk/guides/multi-agent-teams
  * https://dev.to/uenyioha/porting-claude-codes-agent-teams-to-opencode-4hol
  * https://github.com/hueyexe/opencode-ensemble
* memory
  * https://github.com/Gentleman-Programming/engram
  * https://github.com/mnemon-dev/mnemon/tree/master
  * https://github.com/Aamirofficiall/mnemonic
* vector db
  * https://github.com/DeusData/codebase-memory-mcp
  * 
* context
  * https://cleave.dev/
  * https://github.com/context-hub/generator
  * https://github.com/foldwork-dev/mcp-injector
  * https://github.com/headroomlabs-ai/headroom
  * https://github.com/mksglu/context-mode
  * https://github.com/thedotmack/claude-mem
* second brain
  * https://github.com/brobertsaz/claude-os/tree/main

## Selected

* contextmode
* girafe


## Tools



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
