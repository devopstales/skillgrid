# skill-pi permission layer: Pi extension with 3-tier config, Tree-sitter bash matching, tool hiding, diff preview, hybrid approvals

---
status: "accepted"
date: 2026-10-09
supersedes: none
---

## Context and Problem Statement

skill-pi (a customized Pi coding agent) runs with the launching user's full permissions and has no built-in permission system (findings.md line 375). Two reference implementations exist:

- **JayGhiya/pi-opencode-permissions-and-preview**: Opencode-style `allow`/`deny`/`ask` rules, 3-tier config (global `~/.permission.settings.json`, project, local), Tree-sitter AST for bash matching, `@pierre/diffs` preview UI, rule precedence `deny → persisted → session → ask → allow → defaultMode`.
- **gotgenes/pi-permission-system**: Centralized deterministic gates, hides disallowed tools pre-start, wildcard bash matching, sensitive path rules, symlink-resolved, fails closed, forwards subagent prompts, broadcasts `permissions:ui_prompt` / `permissions:decision` events.

skill-pi needs a permission layer that is:
1. Configurable per-user, per-project, and per-machine (local overrides)
2. Precise enough to match complex bash commands (pipes, subshells, command substitution)
3. Proactive (saves LLM turns by hiding tools the agent would always be denied)
4. Visible (the user sees what the agent is about to change before approving)
5. Flexible (pre-configured rules for the 80% case, in-prompt approvals for the long tail)

This is a hard-to-reverse decision: the config file schema, the extension's event hooks, and the tool-hiding mechanism all shape how skill-pi's config tree, the extension's API surface, and the user's muscle memory work. Switching matching strategies later would require migrating all persisted rules. The "no new dependencies without an ADR" rule in AGENTS.md makes the Tree-sitter and `@pierre/diffs` choices explicit records.

## Considered Options

- **Config layering**: (a) 3 tiers (global + project + local), (b) 2 tiers (global + project), (c) project tier only
- **Bash matching**: (a) Tree-sitter AST extraction, (b) wildcard/regex patterns, (c) wildcards first, Tree-sitter opt-in
- **Tool visibility**: (a) hide disallowed tools pre-start, (b) block at call time only, (c) both layers
- **Edit preview**: (a) diff preview UI (`@pierre/diffs`), (b) plain confirm/reject, (c) diff for edits, plain for new files
- **Approval model**: (a) in-prompt approvals only, (b) pre-config rules only, (c) hybrid (rules + in-prompt approvals)

## Decision Outcome

Chosen option: **(a) 3-tier config, (a) Tree-sitter AST matching, (a) hide tools pre-start, (a) diff preview UI, (c) hybrid approvals**, because:

- **3-tier config** matches skill-pi's existing config layering (global `~/.skill-pi/` + project `<project>/.skill-pi/` per findings.md) and the local tier is essential for per-user overrides without touching committed files. This is the proven model from the JayGhiya extension.
- **Tree-sitter AST matching** is the only strategy that handles pipes, subshells, and command substitution correctly. Wildcard matching (`git *`) is a strict subset of AST matching — any rule expressible as a wildcard is also expressible as an AST rule, but not vice versa. The "no new dependencies" rule is satisfied by recording this ADR. Tree-sitter is already used in the mnemonic code index (Extractor term in 02-technical-terms.md), so the dependency is not new to the project's dependency surface — it is a new dependency for the skill-pi package specifically, which is what this ADR records.
- **Hide tools pre-start** saves LLM turns. The `tool_call` event fires after the agent has already decided to call the tool, so blocking is reactive. Hiding is proactive. The gotgenes extension proves this works with Pi's `exposure: 'hidden'` tool property. For a coding agent that makes many tool calls per session, saving even 1-2 turns per session compounds.
- **Diff preview UI** is the highest-value UX feature for a coding agent that makes many file edits. The `@pierre/diffs` dependency is proven in the JayGhiya extension. A user who cannot see what the agent is about to change before approving is forced to trust the agent blindly — the diff preview closes that gap.
- **Hybrid approvals** (pre-configured rules + in-prompt one-off/session/persisted approvals) covers both the 80% case (read tools always allowed, `rm -rf` always denied) and the long tail (new tools, edge cases). The "Reject with feedback" flow (JayGhiya) sends the denial reason back to the LLM so it can adjust its approach — this is a core part of what makes opencode's permission model feel good.

### Consequences

- Good, because the permission layer is a Pi extension (`pi.on("tool_call")` / `pi.on("tool_result")`), not a fork — skill-pi remains an opinionated version of `pi-coding-agent`, not a separate codebase.
- Good, because the 3-tier config file schema is forward-compatible: adding new rule types (e.g., `sensitivePath`, `networkAccess`) does not break existing rules.
- Good, because hiding tools pre-start is a strict improvement over blocking at call time — it cannot be worse, only better.
- Bad, because Tree-sitter adds a significant dependency surface to the skill-pi package (the `tree-sitter` core + language grammars for bash). The AST extraction logic must handle parse errors gracefully (fail open to wildcard matching on parse failure).
- Bad, because the diff preview UI (`@pierre/diffs`) is a second new dependency. If it proves too heavy or has compatibility issues with Pi's TUI, the preview degrades to a plain confirm/reject prompt (the rule still works, just with less visibility).
- Bad, because the hybrid approval model is the most complex of the three options — it requires both a rule evaluation engine AND an in-prompt approval UI. The rule engine must be deterministic and fast (no LLM calls in the hot path); the approval UI must not block the agent's turn.

## Dependencies

- **Tree-sitter**: `tree-sitter` npm package + `tree-sitter-bash` grammar. Already used in the mnemonic code index (the `Extractor` term in 02-technical-terms.md). New dependency for the skill-pi package; recorded here per the "no new dependencies without an ADR" rule.
- **@pierre/diffs**: Diff rendering for the edit preview UI. Used in the JayGhiya reference extension. New dependency for the skill-pi package; recorded here.

## Supersedes

None. This is a new decision. The existing constraint in findings.md ("Pi runs with the launching user's permissions, no built-in permission system") is a description of the current state, not a locked decision — this ADR adds the permission layer as a new capability.
