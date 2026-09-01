#!/usr/bin/env bash
# validate_skill.sh — validate an Agent Skill directory against the core spec rules.
# Usage: validate_skill.sh <skill-dir>
# Exits 0 on success, 1 on any failure. Prints every problem found.
set -u

if [ $# -ne 1 ]; then
  echo "Usage: $0 <skill-dir>" >&2
  exit 1
fi

dir="${1%/}"
[ -d "$dir" ] || { echo "FAIL: '$dir' is not a directory"; exit 1; }

errors=0
fail() { echo "FAIL: $1"; errors=$((errors+1)); }
ok()   { echo "OK:   $1"; }

skill_md="$dir/SKILL.md"
[ -f "$skill_md" ] || { echo "FAIL: $skill_md not found"; exit 1; }
ok "SKILL.md exists"

# --- frontmatter presence ---
first_line=$(head -n1 "$skill_md")
if [ "$first_line" != "---" ]; then
  fail "SKILL.md must start with '---' (YAML frontmatter)"
  exit 1
fi

# Extract frontmatter block (lines between first --- and next ---)
fm=$(awk 'NR>1 && /^---[[:space:]]*$/{exit} NR>1{print}' "$skill_md")

# --- name ---
name=$(printf '%s\n' "$fm" | sed -n 's/^name:[[:space:]]*//p')
if [ -z "$name" ]; then
  fail "frontmatter missing required field: name"
else
  base=$(basename "$dir")
  if [ "$name" != "$base" ]; then
    fail "name '$name' does not match directory name '$base'"
  fi
  if [ "${#name}" -gt 64 ]; then
    fail "name exceeds 64 characters (${#name})"
  fi
  if ! printf '%s' "$name" | grep -qE '^[a-z0-9]+(-[a-z0-9]+)*$'; then
    fail "name is invalid: must be lowercase letters, digits, single hyphens (no leading/trailing/consecutive hyphens), got '$name'"
  fi
fi

# --- description ---
desc=$(printf '%s\n' "$fm" | sed -n 's/^description:[[:space:]]*//p' | head -n1)
if [ -z "$desc" ]; then
  fail "frontmatter missing required field: description"
else
  if [ "${#desc}" -gt 1024 ]; then
    fail "description exceeds 1024 characters (${#desc})"
  fi
  if ! printf '%s' "$desc" | grep -qEi 'use (when|to|for)'; then
    echo "WARN: description does not contain 'use when/to/for' — agents may fail to trigger this skill on the right prompts"
  fi
  ok "description: ${#desc} chars"
fi

# --- optional fields present but sized correctly ---
compat=$(printf '%s\n' "$fm" | sed -n 's/^compatibility:[[:space:]]*//p' | head -n1)
if [ -n "$compat" ] && [ "${#compat}" -gt 500 ]; then
  fail "compatibility exceeds 500 characters (${#compat})"
fi

# --- body budget (progressive disclosure guidance, ~500 lines) ---
total_lines=$(wc -l < "$skill_md")
if [ "$total_lines" -gt 500 ]; then
  echo "WARN: SKILL.md is $total_lines lines (recommended < 500) — move detail to references/"
fi

# --- reference dir hygiene: referenced files actually exist (code blocks and quoted prose examples excluded) ---
ref_grep=$(awk '/^```/{in_block=!in_block; next} !in_block' "$skill_md" | grep -vE '^[[:space:]]*#' | grep -oE '(references|assets)/[A-Za-z0-9_.-]+' | sort -u)
if [ -n "$ref_grep" ]; then
  echo "Referenced files:"
fi
for ref in $ref_grep; do
  if [ -f "$dir/$ref" ]; then
    echo "OK:   referenced $ref exists"
  else
    echo "FAIL: SKILL.md references '$ref' but it does not exist"
    errors=$((errors+1))
  fi
done

# --- verdict ---
if [ "$errors" -eq 0 ]; then
  echo ""
  echo "PASS: $dir is a valid agent skill"
  exit 0
else
  echo ""
  echo "FAIL: $errors problem(s) found in $dir"
  exit 1
fi
