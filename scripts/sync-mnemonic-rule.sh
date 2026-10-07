#!/usr/bin/env bash
# Rebuild rules/mnemonic.mdc from plugins/opencode/memory-protocol.md.
# Edit the shared protocol, then run this script.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SHARED="$ROOT/plugins/opencode/memory-protocol.md"
OUT="$ROOT/rules/mnemonic.mdc"

if [[ ! -f "$SHARED" ]]; then
  echo "missing $SHARED" >&2
  exit 1
fi

mkdir -p "$(dirname "$OUT")"

# Shared file starts with "# Mnemonic Memory Protocol"; keep a single H1 in the rule.
proto_body=$(sed '1{/^# Mnemonic Memory Protocol$/d;}' "$SHARED" | sed '1{/^$/d;}')

python3 - "$OUT" "$proto_body" <<'PY'
import pathlib, sys
out_path, proto = sys.argv[1], sys.argv[2]
front = """---
description: Always apply the Mnemonic memory protocol for skillgrid projects.
globs: "**/*"
alwaysApply: true
---

# Mnemonic Memory Protocol

"""
pathlib.Path(out_path).write_text(front + proto.rstrip() + "\n")
print(f"wrote {out_path}")
PY
