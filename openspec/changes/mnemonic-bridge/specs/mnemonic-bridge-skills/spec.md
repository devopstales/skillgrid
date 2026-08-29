## ADDED Requirements

### Requirement: GitNexus skills carry Mnemonic Integration sections

The system SHALL provide six GitNexus skills under `.agents/skills/` whose
SKILL.md files include a "Mnemonic Integration" section mapping each GitNexus
workflow step to the corresponding Mnemonic `mem_*` / `code_*` / `web_*` call.

#### Scenario: gitnexus-cli Mnemonic Integration

- **GIVEN** the agent reads `.agents/skills/gitnexus-cli/SKILL.md`
- **WHEN** the agent reaches the "Mnemonic Integration" section
- **THEN** it contains `mem_context` before analyze, `mem_save(discovery)` after
  a successful index, `mem_save(bugfix)` for stale-still cases, and
  `web_cache_lookup` for tool documentation

#### Scenario: gitnexus-debugging Mnemonic Integration

- **GIVEN** the agent is debugging with GitNexus
- **WHEN** it reads `.agents/skills/gitnexus-debugging/SKILL.md`
- **THEN** the workflow calls `mem_search` on the error text before `query`,
  and `mem_save(bugfix)` with `topic_key` after the fix

#### Scenario: gitnexus-exploring Mnemonic Integration

- **GIVEN** the agent is exploring an unfamiliar codebase
- **WHEN** it reads `.agents/skills/gitnexus-exploring/SKILL.md`
- **THEN** step 3 of the workflow is `mem_context` (recall prior notes), and
  the close-out calls `mem_save(architecture|discovery)` with a stable
  `topic_key`

#### Scenario: gitnexus-guide Working alongside Mnemonic

- **GIVEN** the agent reads `.agents/skills/gitnexus-guide/SKILL.md`
- **WHEN** it reaches the "Working alongside Mnemonic" section
- **THEN** the section contains a 1:1 tool-to-tool mapping table
  (GitNexus tool ↔ Mnemonic complement), a Mnemonic `type` table, and a
  session-close checklist ending with `mem_session_summary` / `mem_session_end`

#### Scenario: gitnexus-impact-analysis Mnemonic Integration

- **GIVEN** the agent is running impact analysis
- **WHEN** it reads `.agents/skills/gitnexus-impact-analysis/SKILL.md`
- **THEN** the workflow calls `mem_search` for prior assessments before
  `impact`, and `mem_save(decision)` after HIGH/CRITICAL reports or
  `mem_save(discovery)` for UNKNOWN / partial / truncated results

#### Scenario: gitnexus-refactoring Mnemonic Integration

- **GIVEN** the agent is refactoring with GitNexus
- **WHEN** it reads `.agents/skills/gitnexus-refactoring/SKILL.md`
- **THEN** the workflow calls `mem_context` before touching the graph,
  and `mem_save(convention|decision)` after rename/extract/split with a
  stable `topic_key`

#### Scenario: AGENTS.md references the bridge skills

- **GIVEN** the agent reads `.agents/AGENTS.md`
- **WHEN** it reaches the GitNexus "Read this skill file" table
- **THEN** all six skill paths resolve to `.agents/skills/gitnexus-*/SKILL.md`
  and two rows exist for `mnemonic-memory` and `mnemonic-memory-protocol`
