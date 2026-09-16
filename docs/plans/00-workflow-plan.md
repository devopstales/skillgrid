# Plan

## Workflow

Idea -> Briefing -> Blueprint

* Think
  * `brainstorming`
  * `interview`
* Plan
  * `write-blueprint`
  * `slice`
* Build
  * `simple-execution`
  * `subagent-execution`
  * `parallel-execution`
  * `structured-debugging`
  * `isolated-workspace`
  * `test-driven-development`
* Review
  * `request-code-review`
  * `receiving-code-review`
  * `test-driven-verification`
* Test
* Ship
* Reflect

## Files

briefing.md  - AI Generated Plan
blueprint.md - Technical Implementation document
tasks.md     - Execution Phases, Vertical slice
research.md  - research resource Optional

### File Structures

```bash
.skillgrid/
  config.yaml
  agents/
    grosery.md
  archive/
    YYYY-MM-DD-function/
  spec/
    YYYY-MM-DD-function/
      briefing.md
      blueprint.md
      research.md
      acceptance.feature
      tasks.md

docs/
  prd.md
  architecture.md
```

### File Formats

```yaml
# .skillgrid/config.yaml
---
schema: skillgrid-sdd/v1
git:
  project: <project-name>
  repo_url: <repo-url>
  branching_strategx: [<list>,<of>,<branching>,<order>]
issue_tracker:
  enabled: <true|false>
  name: <backlog.md|github|gitlab|jira>
  workflow: ["todo","inprogress","done"]
technology:
  language: <programing-language>
  framework: <coding-framework>
testing:
  runner: <how-to-run-test>
  layers: [<list>,<of>,<test>,<types>]
  coverage: <min-60-optional-80-100>
  tdd: <true|false>
  bdd: <true|false>
```

## Ticketing

| Concept | Jira-style | GitHub-style | Gitlan-style | Where it lives |
|---------|------------|--------------|--------------|----------------|
| | Epic | Milestone | Milestone | |
| | Task | Issue | Issue | |
| | | Sub-Task | Sub-issues | Sub-issues | |
