#!/usr/bin/env bash
# session-start.sh — inject the using-skillgrid router into every new session.
#
# The router (using-skillgrid) currently relies on its frontmatter description
# matching to be loaded. That is a soft guarantee: a harness that does not
# auto-load skills, or one whose router description changes, can start a
# session with the router out of context. This SessionStart hook closes that
# gap — it emits the router's routing rules as a priority-IMPORTANT message so
# the agent has them before its first response, in any session, in any repo.
#
# Harness: Claude Code (and any harness that reads a SessionStart hook's stdout
# as a JSON hook payload). Other harnesses can reuse the script; each wires it
# in its own settings file (see .claude/settings.json for the Claude Code form).
#
# Output contract: a single JSON object on stdout. On any error the hook degrades
# to an INFO note and exits 0 — it never blocks session start.
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ROUTER="$ROOT/.agents/skills/using-skillgrid/SKILL.md"

# JSON-escape a single line: backslashes, double quotes, tabs, CR.
# (Newlines never reach here — the caller passes one line at a time.)
jesc() {
  printf '%s' "$1" \
    | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g' -e 's/\t/\\t/g' -e 's/\r//g'
}

INTRO="skillgrid loaded. Route through the using-skillgrid flow before ANY response (including clarifying questions)."

if [ -f "$ROUTER" ]; then
  # Assemble the message: intro, a blank line, then the router body. Each line
  # is escaped on its own; lines are joined with a literal \n so the payload is
  # one valid JSON line.
  msg=""
  while IFS= read -r line || [ -n "$line" ]; do
    [ -n "$msg" ] && msg="${msg}\\n"
    msg="${msg}$(jesc "$line")"
  done < "$ROUTER"

  body="$(jesc "$INTRO")\\n\\n${msg}"
  printf '{"hookSpecificName":"SessionStart","priority":"IMPORTANT","message":"%s"}\n' "$body"
else
  printf '{"hookSpecificName":"SessionStart","priority":"INFO","message":"skillgrid: using-skillgrid router not found at %s - skills may still be available individually."}\n' "$ROUTER"
fi
exit 0
