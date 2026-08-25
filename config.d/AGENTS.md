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

### aiskillgrid-cli (if present in workspace)

- Tooling lives in `~/.aiskillgrid`: binaries in `~/.aiskillgrid/bin`, npm binaries in `~/.aiskillgrid/node_modules/.bin` (packages installed with `--prefix "$HOME/.aiskillgrid"`).
- YAML files in `config.d/` are the single source of truth for installs; do not hardcode package names in code.
- Keep unit tests for config merge logic and `--dry-run` output passing (`go build ./... && go test ./...` in `aiskillgrid-cli/`).
- Commit messages: short imperative, one line (e.g. `fix: merge MCP entries without clobbering keys`).

## Environment

- Engram persistent memory is available: update it on decisions, bug fixes, and non-obvious discoveries.
