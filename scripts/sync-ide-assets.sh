#!/usr/bin/env bash
# sync-ide-assets.sh — keep IDE mirrors in sync with the canonical .github assets.
#
# The canonical prompts live in .github/prompts/ and the canonical agents in
# .github/agents/. Some harnesses (Cursor, Copilot, ...) consume per-IDE copies
# under .<harness>/; those copies are mirrors, not sources. This script:
#
#   --check   (CI) verify the canonical set is intact and that any IDE mirror
#             present on disk is in sync with it. Exits non-zero on drift.
#             No IDE mirror exists yet -> check the canonical set only (green).
#
#   --sync    (default) regenerate any present IDE mirrors from the canonical
#             assets, and print what would change if a mirror were added.
#
# Usage:
#   bash scripts/sync-ide-assets.sh --check     # CI
#   bash scripts/sync-ide-assets.sh             # local sync
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROMPTS_DIR="$ROOT/.github/prompts"
AGENTS_DIR="$ROOT/.github/agents"

MODE="${1:---check}"

# IDE mirror dirs: <dir> maps prompts to <dir>/prompts, agents to <dir>/agents.
# Extend this list when a new IDE copy lands in the repo.
IDE_MIRRORS=( ".cursor" ".copilot" )

fail() { echo "FAIL: $*" >&2; exit 1; }
DRIFT=0

check_canonical() {
  [ -d "$PROMPTS_DIR" ] || fail "canonical prompts dir missing: .github/prompts"
  [ -d "$AGENTS_DIR" ]  || fail "canonical agents dir missing: .github/agents"

  local n_prompts n_agents
  n_prompts="$(find "$PROMPTS_DIR" -name '*.md' -type f | wc -l | tr -d ' ')"
  n_agents="$(find "$AGENTS_DIR" -name '*.md' -type f ! -name 'README.md' | wc -l | tr -d ' ')"

  [ "$n_prompts" -ge 1 ] || fail "no prompt files in $PROMPTS_DIR"
  [ "$n_agents" -ge 1 ] || fail "no agent files in $AGENTS_DIR"

  # Every prompt must be non-empty and carry a title: either a frontmatter
  # `description:` (the canonical .github/prompts format) or a `#` heading.
  local f
  for f in "$PROMPTS_DIR"/*.md; do
    [ -s "$f" ] || fail "empty prompt: $f"
    head -5 "$f" | grep -Eq '^(description:|#)' || fail "prompt missing title: $f"
  done

  echo "canonical OK: $n_prompts prompts, $n_agents agents"
}

# compare one IDE mirror (if present) against canonical; report drift
check_mirror() {
  local ide="$1"
  local dir="$ROOT/$ide"
  [ -d "$dir" ] || { echo "mirror $ide: not present (nothing to check)"; return 0; }

  local sub label src dst
  for sub in prompts agents; do
    src="$ROOT/.github/$sub"
    dst="$dir/$sub"
    [ -d "$src" ] || continue
    if [ -d "$dst" ]; then
      if diff -rq "$src" "$dst" >/dev/null 2>&1; then
        echo "mirror $ide/$sub: in sync"
      else
        echo "mirror $ide/$sub: DRIFT (run --sync)" >&2
        DRIFT=1
      fi
    else
      echo "mirror $ide: $sub/ missing (run --sync to create)" >&2
      DRIFT=1
    fi
  done
}

sync_one() {
  local ide="$1" dir="$ROOT/$ide"
  mkdir -p "$dir/prompts" "$dir/agents"
  # Mirror the canonical .md assets (skip the agents README — it's hub docs).
  cp -f "$PROMPTS_DIR"/*.md "$dir/prompts/" 2>/dev/null || true
  ( cd "$AGENTS_DIR" && for f in *.md; do [ "$f" = "README.md" ] || cp -f "$f" "$dir/agents/"; done )
  echo "synced $ide"
}

case "$MODE" in
  --check)
    check_canonical
    for ide in "${IDE_MIRRORS[@]}"; do check_mirror "$ide"; done
    [ "$DRIFT" -eq 0 ] || fail "IDE mirror drift detected"
    echo "sync-check OK"
    ;;
  --sync)
    check_canonical
    for ide in "${IDE_MIRRORS[@]}"; do
      [ -d "$ROOT/$ide" ] || { echo "$ide: not present, skipping (add the dir to mirror)"; continue; }
      sync_one "$ide"
    done
    echo "sync done"
    ;;
  *)
    echo "usage: sync-ide-assets.sh [--check|--sync]" >&2; exit 2 ;;
esac
