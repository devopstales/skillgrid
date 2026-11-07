Feature: Development Workflow execution guide in the projectwiki vault

  # A docs-only change: the deliverable is an attested vault page + a cross-link
  # + a glossary row. The checks are structural (file existence, frontmatter,
  # headings, cross-link, glossary term) plus the no-new-drift projectwiki gate
  # run on a freshly-built binary (the installed 0.2.0 is stale, pre-rename).

  @R1
  Scenario: happy path workflow-page-exists-with-all-steps
    Given the projectwiki vault at `.skillgrid/projectwiki/`
    When the Development Workflow page is authored at `workflows/development-workflow.md`
    Then the file exists
    And its frontmatter carries `type: BusinessWorkflow` and `title: "Development Workflow"` and an `anchors:` list with both resolvable anchors
    And it contains all 11 step headings `## Step 0` through `## Step 10`
    And each step names its Skill, Subagents, MCP, CLI, Files, and Gates

  @R2
  Scenario: happy path step-fidelity-spot-checks
    Given the Development Workflow page exists
    When each step's substance is spot-checked against the approved plan
    Then Step 1 names `brainstorming` and the HARD GATE
    And Step 2 names `writing-blueprints` and the critic subagent (mandatory T2/T3) and the plan-review subagent
    And Step 3 names `acceptance-test-authoring` and the cucumber dry-run and the gherkin lint
    And Step 4 names `slicing` and `ticketing` and the 4 Delivery Strategy guard lines and the 4.3 user gate
    And Step 5 names `subagent-execution` and the per-ticket loop and the four stops
    And Step 6 names `qa` and 0 subagents and the 9.5 gate chain and the four-state verdict
    And Step 7 names `requesting-code-review` and exactly 3 parallel reviewers and Step 7.5 names `review-ledger` with the `review: record round <N>` commit
    And sub-steps 1a/1b/1c land under `## Step 1` and 7.5 under `## Step 7`

  @R3
  Scenario: happy path pipeline-page-cross-links
    Given the Development Workflow page exists
    When `workflows/sdd-pipeline.md` is read
    Then it references `development-workflow.md` as the step-by-step execution guide

  @R4
  Scenario: happy path glossary-term-present
    Given the business glossary at `glossary/01-business-terms.md`
    When the `Development Workflow` term is added
    Then the glossary contains a `Development Workflow` row with Definition, Use When, and Avoid

  @R5
  Scenario: happy path kb-indexed-and-valid
    Given a freshly-built `mnemonic` binary from `mnemonic-v2/`
    And the page + cross-link + glossary edits are complete
    When `projectwiki index` runs in the same edit
    And `projectwiki validate --strict` runs
    And `projectwiki lint` runs
    Then `projectwiki index` exits 0 and indexes the new page
    And `projectwiki validate --strict` reports the new page `ok` with the 2 pre-existing drifts only (no new drift)
    And `projectwiki lint` exits 0 with no issues

  @R6
  Scenario: failure kb-validate-detects-anchor-drift
    Given the new page is `ok` (2 pre-existing drifts baseline)
    When one anchor in the new page is broken and re-indexed
    Then the `projectwiki validate --strict` per-page drift count rises to 3
    When the anchor is restored and re-indexed
    Then the `projectwiki validate --strict` per-page drift count returns to 2 (new page `ok`)
