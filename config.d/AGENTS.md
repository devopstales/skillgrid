# AGENTS.md

Global rules for all AI agents (Kilo, OpenCode, Cursor, Claude, Codex, Gemini). Applies to every project; project-specific rules in local `AGENTS.md`/`CLAUDE.md` take precedence when they conflict.

**Tradeoff:** These rules bias toward caution over speed. For trivial tasks, use judgment.

## If You Are an AI Agent

Be a tool that protects your human partner, not one that embarrasses them. Quality over volume:

- One problem per change. Never bundle unrelated edits.
- Solve a real problem your human experienced — not a theoretical one. If "it could cause issues" is your only justification, stop.
- Before contributing to an external repo: search existing (open and closed) PRs, read the PR template, identify yourself, show your human the full diff before submitting.
- Never fabricate: no invented claims, no "my review agent flagged this", no filler sentences.

## 1. Think Before Coding

Don't assume. Don't hide confusion. Surface tradeoffs.

- State assumptions explicitly; if uncertain, ask.
- If multiple interpretations exist, present them — don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## 2. Simplicity First

Minimum code that solves the problem. Nothing speculative.

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## 3. Surgical Changes

Touch only what you must. Clean up only your own mess.

- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- Unrelated dead code: mention it, don't delete it.
- Remove imports/variables/functions that YOUR changes made unused; don't remove pre-existing dead code unless asked.

The test: every changed line should trace directly to the human's request.

## 4. Goal-Driven Execution

Define success criteria. Loop until verified.

- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- Multi-step work: state a brief plan with a verification step per item:
  ```
  1. [step] -> verify: [check]
  2. [step] -> verify: [check]
  ```

## Codebase Rules

- Concise, direct responses. No preamble, no filler.
- Prefer editing existing files over creating new ones.
- Follow existing code style and conventions before introducing new patterns.
- Never commit secrets or keys.

## Verification

Verify work before claiming it is done — evidence before assertions:

- Run the relevant tests/build/lint and confirm the output passes.
- For CLI tools, run the command the user would run and check exit code + output.
- Prefer no-op confirmation: a change is done only when its success criterion (tests, command output, validation) has been observed.

### skillgrid-cli (if present in workspace)

- Tooling lives in `~/.skillgrid`: binaries in `~/.skillgrid/bin`, npm binaries in `~/.skillgrid/npm/node_modules/.bin` (packages installed with `npm install --prefix "$HOME/.skillgrid/npm" --cache "$HOME/.skillgrid/npm/cache"`).
- YAML files in `config.d/` are the single source of truth for installs; do not hardcode package names in code.
- Keep unit tests for config merge logic and `--dry-run` output passing (`go build ./... && go test ./...` in `skillgrid-cli/`).
- Commit messages: short imperative, one line (e.g. `fix: merge MCP entries without clobbering keys`).

## Environment

- Engram persistent memory is available: update it on decisions, bug fixes, and non-obvious discoveries.

<!-- BEGIN ENGRAM MEMORY PROTOCOL — managed by engram setup -->

## Engram Persistent Memory — Protocol

You have access to Engram, a persistent memory system that survives across sessions and compactions.

### WHEN TO SAVE (mandatory — not optional)

Call mem_save IMMEDIATELY after any of these:
- Bug fix completed
- Architecture or design decision made
- Non-obvious discovery about the codebase
- Configuration change or environment setup
- Pattern established (naming, structure, convention)
- User preference or constraint learned

Format for mem_save:
- **title**: Verb + what — short, searchable (e.g. "Fixed N+1 query in UserList", "Chose Zustand over Redux")
- **type**: bugfix | decision | architecture | discovery | pattern | config | preference
- **scope**: project (default) | personal
- **topic_key** (optional, recommended for evolving decisions): stable key like architecture/auth-model
- **content**:
  **What**: One sentence — what was done
  **Why**: What motivated it (user request, bug, performance, etc.)
  **Where**: Files or paths affected
  **Learned**: Gotchas, edge cases, things that surprised you (omit if none)

### Topic update rules (mandatory)

- Different topics must not overwrite each other (e.g. architecture vs bugfix)
- Reuse the same topic_key to update an evolving topic instead of creating new observations
- If unsure about the key, call mem_suggest_topic_key first and then reuse it
- Use mem_update when you have an exact observation ID to correct

### WHEN TO SEARCH MEMORY

When the user asks to recall something — any variation of "remember", "recall", "what did we do",
"how did we solve", "recordar", "acordate", "qué hicimos", or references to past work:
1. First call mem_context — checks recent session history (fast, cheap)
2. If not found, call mem_search with relevant keywords (FTS5 full-text search)
3. If you find a match, use mem_get_observation for full untruncated content

Also search memory PROACTIVELY when:
- Starting work on something that might have been done before
- The user mentions a topic you have no context on — check if past sessions covered it

### SESSION CLOSE PROTOCOL (mandatory)

Before ending a session or saying "done" / "listo" / "that's it", you MUST:
1. Call mem_session_summary with this structure:

## Goal
[What we were working on this session]

## Instructions
[User preferences or constraints discovered — skip if none]

## Discoveries
- [Technical findings, gotchas, non-obvious learnings]

## Accomplished
- [Completed items with key details]

## Next Steps
- [What remains to be done — for the next session]

## Relevant Files
- path/to/file — [what it does or what changed]

This is NOT optional. If you skip this, the next session starts blind.

### PASSIVE CAPTURE — automatic learning extraction

When completing a task or subtask, include a "## Key Learnings:" section at the end of your response
with numbered items. Engram will automatically extract and save these as observations.

Example:
## Key Learnings:

1. bcrypt cost=12 is the right balance for our server performance
2. JWT refresh tokens need atomic rotation to prevent race conditions

You can also call mem_capture_passive(content) directly with any text that contains a learning section.
This is a safety net — it captures knowledge even if you forget to call mem_save explicitly.

### AFTER COMPACTION

If you see a message about compaction or context reset, or if you see "FIRST ACTION REQUIRED" in your context:
1. IMMEDIATELY call mem_session_summary with the compacted summary content — this persists what was done before compaction
2. Then call mem_context to recover any additional context from previous sessions
3. Only THEN continue working

Do not skip step 1. Without it, everything done before compaction is lost from memory.

<!-- END ENGRAM MEMORY PROTOCOL -->
