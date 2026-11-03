# AGENTS.md Template

> **Template.** Single source of truth for the AGENTS.md shape.
> `skillgrid init` renders the ` ````markdown ` fenced region into `AGENTS.md`,
> filling `{placeholders}` from `.skillgrid/config.yaml`. Authoring hints in
> `<!-- ... -->` are stripped from the output. Unfilled placeholders remain as
> `<...>` for the agent or onboarding to fill.
>
> **Design principles:**
> - One-sentence project description — anchors every decision the agent makes
> - Only non-inferable details — if the agent can find it in 10 seconds, leave it out
> - Rules are constraints (never/always), Conventions are structure
> - Gates are trigger tables (when to run what), tool links are for the full recipe
> - Tool conventions are 1-2 lines per tool (flags, gotchas, suppression)
> - Boundaries are permission tiers (✅/⚠️/🚫)
> - Memory is a 4-line trigger list, protocol lives in linked files
> - Target: under 200 lines. Every token costs on every invocation.

````markdown
# {project}

{description}

```
{project_tree}
```

## Commands

```bash
# Build
{build}

# Test
{test}

# Lint
{lint}

# Format
{format}

# Security scan
{security_scan}
```

## Quality gate

{quality_gate}

{quality_tools}

## Security gate

{security_gate}

{security_tools}

## Rules

{rules}

## Conventions

{conventions}

### Quality tool conventions

{quality_tool_conventions}

### Security tool conventions

{security_tool_conventions}

## Boundaries

### ✅ Do without asking
{boundaries_do}

### ⚠️ Ask first
{boundaries_ask}

### 🚫 Never
{boundaries_never}

### Dependency policies
{dependency_policies}

### Definition of done
{definition_of_done}

### Engineering standards
{engineering_standards}

## Memory

Save decisions, gotchas, and non-obvious findings with `{memory_save_command}` so they survive the session. Full protocol: [memory rules]({memory_protocol_rules}) · [artifact naming]({memory_protocol_artifacts}).

- **Save when:** architecture decision, bug fix (root cause), gotcha, convention, discovery
- **Recover when:** starting familiar work, user references past work, after compaction — `{memory_recover_ladder}`
- **Session close:** `{memory_summary_command}` (sections: {memory_summary_sections}) → `{memory_end_command}` before saying "done"
- **After compaction:** FIRST `{memory_summary_command}` with compacted content, THEN `mem_context`, only then continue

## Read before changing

{read_before_changing}
````
