---
name: shell
description: >
  Google's official Shell style guide. Covers Bash scripting, naming conventions, error handling, portability, and shell best practices. Use when writing, reviewing, or
  refactoring Shell code in this repo, or when the user asks for the Shell style
  convention. Part of the skillgrid code-style group.
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  based_on: Google style guide, vendored from testdino-hq/google-styleguides-skills (MIT, TestDino 2026)
---
# Google Shell Style Guide

> Official Google Shell scripting standards for consistent, maintainable Bash scripts.

## Quick Reference

### Golden Rules

1. **Use `#!/bin/bash`** — specify interpreter
2. **Quote variables** — `"$var"` not `$var`
3. **Check exit codes** — `set -e` or explicit checks
4. **Use functions** — organize code logically
5. **Avoid parsing `ls`** — use globs or find
6. **Use `[[` over `[`** — more features, fewer gotchas
7. **ShellCheck for linting** — catch common mistakes

### Naming Conventions (Preview)

| Element | Convention | Example |
|---|---|---|
| Functions | lowercase_with_underscores | `get_user_count` |
| Variables | lowercase_with_underscores | `user_count` |
| Constants | UPPER_SNAKE_CASE | `MAX_RETRIES` |
| Environment vars | UPPER_SNAKE_CASE | `PATH` |

### Basic Patterns (Preview)

```bash
#!/bin/bash
# ✓ CORRECT

set -euo pipefail  # Exit on error, undefined vars, pipe failures

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

function main() {
  local user_count="$1"
  
  if [[ -z "${user_count}" ]]; then
    echo "Error: user_count required" >&2
    return 1
  fi
  
  echo "Processing ${user_count} users"
}

main "$@"
```

## When to Use This Guide

- Writing new shell scripts
- Refactoring existing scripts
- Code reviews
- Setting up ShellCheck
- Onboarding new team members

## Install

```bash
npx skills add testdino-hq/google-styleguides-skills/shell
```

## Full Guide

See [shell.md](shell.md) for complete details, examples, and edge cases.
