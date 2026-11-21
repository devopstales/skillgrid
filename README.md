
* `skillgrid` - trigger

* `brainstorm`
  * question
  * write breafing
  * `blind-validate`
* `blueprint`
  * `research`/`code-research`
  * write intent driven tests
  * `prototype`
* `sliece`
  * `ticketing`
  * weticle sliceing
  * task size max 20% of context window
* `execute`
  * tdd
  * style guide
  * inline execute
  * sub agent execute
  * paralel execute
* `test`
  * intent testing
  * functional testing
  * code qualtity: jscpd
  * security: trivy, wapity, nuklei, akca, owasp
* `ship`
  * reflect into projectwiki
  * environment-retro into projectwiki
  * documenting into projectwiki
  * commit
  * pr
  * archive

## Tool Dependency

Tools the skills invoke that you need installed. "optional" = only when the corresponding tracker/feature is configured in `.skillgrid/config.yaml` or a live target is available.

- **Install (binary/CLI):**
  - `trivy` — static security scanning: deps, secrets, IaC, containers, SBOM (test; always for security gate)
  - `nuclei` — broad DAST at a running target, CVE templates (test; optional — needs a live target)
  - `wapiti` — lightweight DAST fuzzer (test; optional — needs a live target)
  - `akca` — deep evidence-backed DAST, browser-backed (test; optional — needs Go 1.25+ and Chrome/Chromium)
  - `playwright-cli` — browser automation for functional testing (test; install with `npm install -g @playwright/cli@latest`)
  - `gh` — GitHub CLI for issues / PRs / labels (slicing, ship; optional — when `ticketing.type: gh`)
  - `glab` — GitLab CLI for issues / MRs / labels (slicing, ship; optional — when `ticketing.type: glab`)
  - `jira` — Jira CLI for epics / tasks (slicing; optional — when `ticketing.type: jira`)
  - `backlog` — Backlog.md CLI for file-based tickets (slicing, ship; optional — when `ticketing.type: backlogmd`)
  - `hermes` — Hermes Kanban CLI for ticket tracking (slicing; optional — when `ticketing.type: hermes-kanban`)
- **MCP servers — wired by a skill (install + wire in the agent's MCP config):**
  - `mnemonic` — run as `mnemonic mcp`; also a CLI (`mnemonic init`, `mnemonic projectwiki ...`); exposes `mem_*` + `projectwiki_*` tools (optional — only when `mnemonic.enabled: true` in `.skillgrid/config.yaml`; needs an embedder: ollama / external / onnx) — used by the SDD memory convention (`_shared/rules/mnemonic-memory.md`) and projectwiki
  - `agent-browser` — run as `agent-browser mcp --tools all`; browser automation with stable element refs — preferred driver for the functional browser pass (`test/references/functional-testing.md`)
  - `trivy` — also run as `trivy mcp`; `findings_list`/`findings_get` to read scan results as structured data (the scan itself still runs via the CLI) — `test/references/security.md`
  - `backlog` — also run as `backlog mcp start`; structured create/read/update that avoids shell-quoting footguns — `slicing/references/ticketing/backlogmd/`
  - `codescene` — run as `npx @codescene/codehealth-mcp`; Code Health score (different metric than jscpd) for before/after refactoring deltas — `codebase-refactoring/SKILL.md`
- **MCP servers — remote, no install (just enabled in the agent's MCP config):**
  - `context7` — current library/framework API docs (`resolve-library-id` → fetch doc page) — blueprint research + prototype
  - `exa` — `web_search_exa` (find the page) + `web_fetch_exa` (read it) — blueprint research
- **Run via npx (no install needed, fetched per run):**
  - `npx jscpd` — duplication / dead-code / complexity (test, codebase-refactoring, execute)
  - `npx cucumber-js` — BDD acceptance test runner (blueprint, execute; `@cucumber/cucumber` as project devDependency)
  - `npx gherkin-lint` — Gherkin spec linting (blueprint)
  - `npx stryker run` — mutation testing (test; optional — only if `stryker.config.js` is present)
  - `npx playwright cli` — fallback install for `playwright-cli` (test)
- **Run via uvx/uv (no install needed, fetched per run):**
  - `uvx pygount` — LOC / codebase sizing (execute brownfield baseline)
  - `uv run scripts/get_transcript.py` — YouTube transcript extraction (youtube-transcript; uses `youtube-transcript-api`)
- **Test runners (per project, installed by the project itself):**
  - `pytest` — Python test runner (blueprint, test)
