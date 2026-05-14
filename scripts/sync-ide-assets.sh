#!/usr/bin/env bash
# Sync hub IDE mirrors from .agents sources of truth.
# Usage:
#   ./scripts/sync-ide-assets.sh           # write commands/prompts plus agent/rule mirrors
#   ./scripts/sync-ide-assets.sh --check   # exit 1 if any mirror differs (CI)
# Removes command/prompt/agent/rule files under mirrors when the matching source file is deleted.
set -uo pipefail
shopt -s nullglob

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CHECK=0
if [[ "${1:-}" == "--check" ]]; then
  CHECK=1
fi

EXIT=0

sync_pair() {
  local src="$1" dst="$2"
  if [[ $CHECK -eq 1 ]]; then
    if [[ ! -f "$dst" ]] || ! cmp -s "$src" "$dst"; then
      echo "DRIFT: $dst (expected content from $src)" >&2
      EXIT=1
    fi
  else
    mkdir -p "$(dirname "$dst")"
    cp "$src" "$dst"
  fi
}

# Cursor/Kilo/OpenCode workflow commands are rendered from .agents/workflows.
if ! python3 - "$ROOT" "$CHECK" <<'PY'
import re
import sys
from pathlib import Path

root = Path(sys.argv[1])
check = sys.argv[2] == "1"
src_dir = root / ".agents/workflows"
dest_dirs = [root / ".cursor/commands", root / ".kilo/commands", root / ".opencode/commands"]
drift = False


def frontmatter(text: str, src: Path):
    match = re.match(r"^---\n(.*?)\n---\n?", text, re.DOTALL)
    if not match:
        print(f"sync-ide-assets: missing frontmatter: {src}", file=sys.stderr)
        sys.exit(2)
    fields = []
    for line in match.group(1).splitlines():
        if not line.strip() or ":" not in line:
            continue
        key, value = line.split(":", 1)
        fields.append((key.strip(), value.strip()))
    return fields, text[match.end() :]


def field_map(fields):
    return {key: value for key, value in fields}


def render_command(src: Path) -> str:
    text = src.read_text(encoding="utf-8")
    fields, body = frontmatter(text, src)
    values = field_map(fields)
    stem = src.stem
    description = values.get("description")
    if not description:
        print(f"sync-ide-assets: missing description: {src}", file=sys.stderr)
        sys.exit(2)

    rendered = [
        "---",
f"name: {values.get('id') or stem}",
         f"id: {values.get('id') or stem}",
        f"category: {values.get('category') or 'Workflow'}",
        f"description: {description}",
    ]
    for key, value in fields:
        if key in {"name", "id", "category", "description"}:
            continue
        rendered.append(f"{key}: {value}")
    rendered.append("---")
    return "\n".join(rendered) + "\n" + body


files = sorted(src_dir.glob("*.md"))
expected = {src.name for src in files}
for dest_dir in dest_dirs:
    for src in files:
        out = render_command(src)
        dest = dest_dir / src.name
        if check:
            if not dest.exists() or dest.read_text(encoding="utf-8") != out:
                print(f"DRIFT: {dest} (expected content from {src})", file=sys.stderr)
                drift = True
        else:
            dest.parent.mkdir(parents=True, exist_ok=True)
            dest.write_text(out, encoding="utf-8")

    # Drop mirror commands removed from .agents/workflows.
    for dest in sorted(dest_dir.glob("*.md")):
        if dest.name not in expected:
            if check:
                print(f"ORPHAN: {dest} (no longer in .agents/workflows)", file=sys.stderr)
                drift = True
            else:
                dest.unlink()

sys.exit(1 if drift else 0)
PY
then
  EXIT=1
fi

for dest in "$ROOT/.github/agents"; do
  for f in "$ROOT/.cursor/agents"/*.md; do
    sync_pair "$f" "$dest/$(basename "$f")"
  done
  # Drop mirror agent files removed from .cursor/agents (e.g. disabled personas)
  for f in "$dest"/*.md; do
    [[ -f "$f" ]] || continue
    base="$(basename "$f")"
    if [[ ! -f "$ROOT/.cursor/agents/$base" ]]; then
      if [[ $CHECK -eq 1 ]]; then
        echo "ORPHAN: $f (no longer in .cursor/agents)" >&2
        EXIT=1
      else
        rm -f "$f"
      fi
    fi
  done
done

for dest in "$ROOT/.kilo/rules" "$ROOT/.opencode/rules"; do
  for f in "$ROOT/.agents/rules"/skillgrid-*.mdc "$ROOT/.cursor/rules"/skillgrid-*.mdc; do
    sync_pair "$f" "$dest/$(basename "$f")"
  done
  # Drop mirror rule files removed from hub/cursor rule sources.
  for f in "$dest"/skillgrid-*.mdc; do
    [[ -f "$f" ]] || continue
    base="$(basename "$f")"
    if [[ ! -f "$ROOT/.agents/rules/$base" ]] && [[ ! -f "$ROOT/.cursor/rules/$base" ]]; then
      if [[ $CHECK -eq 1 ]]; then
        echo "ORPHAN: $f (no longer in .agents/rules or .cursor/rules)" >&2
        EXIT=1
      else
        rm -f "$f"
      fi
    fi
  done
done

# Kilo/OpenCode agents use OpenCode's Markdown agent schema:
# - the filename is the agent id (e.g. tyr.md — same stem as Norse persona key)
# - access is configured through permission, not comma-separated tools
if ! python3 - "$ROOT" "$CHECK" <<'PY'
import re
import sys
from pathlib import Path

root = Path(sys.argv[1])
check = sys.argv[2] == "1"
cur = root / ".cursor/agents"
dest_dirs = [root / ".kilo/agents", root / ".opencode/agents"]
drift = False


def frontmatter(text: str):
    match = re.match(r"^---\n(.*?)\n---\n?", text, re.DOTALL)
    if not match:
        return None, text
    fields = {}
    for line in match.group(1).splitlines():
        if ":" not in line:
            continue
        key, value = line.split(":", 1)
        fields[key.strip()] = value.strip()
    return fields, text[match.end() :]


PERSONA_AGENT_STEMS = frozenset(
    {"board", "odin", "thor", "tyr", "heimdall", "frigg", "loki", "mimir", "bragi", "vidar"}
)
# Markdown agents with the same frontmatter + body section contract as Norse personas
CONTRACT_AGENT_STEMS = PERSONA_AGENT_STEMS | frozenset({"orchestrator"})


def verify_agent_contract(src: Path, fields: dict, body: str):
    if src.stem not in CONTRACT_AGENT_STEMS:
        return
    required_fields = ["name", "description", "tools", "color"]
    for field in required_fields:
        if not fields.get(field):
            print(f"sync-ide-assets: {src} missing frontmatter field: {field}", file=sys.stderr)
            sys.exit(2)
    required_sections = [
        "## Identity and discipline",
        "## Mandatory Context",
        "## Rules",
        "## Composition",
    ]
    for section in required_sections:
        if section not in body:
            print(f"sync-ide-assets: {src} missing required section: {section}", file=sys.stderr)
            sys.exit(2)


def render_agent(src: Path) -> str:
    text = src.read_text(encoding="utf-8")
    fields, body = frontmatter(text)
    if fields is None:
        return text
    verify_agent_contract(src, fields, body)

    description = fields.get("description")
    if not description:
        print(f"sync-ide-assets: missing description: {src}", file=sys.stderr)
        sys.exit(2)

    tools = {tool.strip() for tool in fields.get("tools", "").split(",") if tool.strip()}
    mode_raw = fields.get("mode", "subagent").strip().strip('"').strip("'").lower()
    mode = "primary" if mode_raw == "primary" else "subagent"

    lines = [
        "---",
        f"description: {description}",
        f"mode: {mode}",
        "permission:",
        "  read: allow",
        "  glob: allow",
        "  grep: allow",
    ]
    if mode == "primary":
        edit_ok = "Edit" in tools or "Write" in tools
        lines.append(f"  edit: {'allow' if edit_ok else 'deny'}")
        lines.append(f"  task: {'allow' if 'Task' in tools else 'deny'}")
    else:
        lines.append("  edit: deny")
        lines.append("  task: deny")
    lines.append(f"  bash: {'allow' if 'Bash' in tools else 'deny'}")
    if "WebSearch" in tools:
        lines.append("  websearch: allow")
    if "WebFetch" in tools:
        lines.append("  webfetch: allow")
    if color := fields.get("color"):
        c = color.strip().strip('"').strip("'")
        lines.append(f'color: "{c}"')
    lines.append("---")
    return "\n".join(lines) + "\n" + body


files = sorted(cur.glob("*.md"))
for dest_dir in dest_dirs:
    for src in files:
        out = render_agent(src)
        dest = dest_dir / src.name
        if check:
            if not dest.exists() or dest.read_text(encoding="utf-8") != out:
                print(f"DRIFT: {dest}", file=sys.stderr)
                drift = True
        else:
            dest.parent.mkdir(parents=True, exist_ok=True)
            dest.write_text(out, encoding="utf-8")

    # Drop mirror files removed from .cursor/agents (e.g. disabled personas)
    for dest in sorted(dest_dir.glob("*.md")):
        src = cur / dest.name
        if not src.exists():
            if check:
                print(f"ORPHAN: {dest} (no longer in .cursor/agents)", file=sys.stderr)
                drift = True
            else:
                dest.unlink()

sys.exit(1 if drift else 0)
PY
then
  EXIT=1
fi

# GitHub Copilot prompts: workflow body with Copilot-compatible frontmatter.
if ! python3 - "$ROOT" "$CHECK" <<'PY'
import re
import sys
from pathlib import Path

root = Path(sys.argv[1])
check = sys.argv[2] == "1"
src_dir = root / ".agents/workflows"
gh = root / ".github/prompts"
files = sorted(src_dir.glob("*.md"))
drift = False
for f in files:
    text = f.read_text(encoding="utf-8")
    m = re.match(r"^---\n(.*?)\n---\n", text, re.DOTALL)
    if not m:
        print(f"sync-ide-assets: missing frontmatter: {f}", file=sys.stderr)
        sys.exit(2)
    fm = m.group(1)
    dm = re.search(r"^description:\s*(.+)$", fm, re.MULTILINE)
    if not dm:
        print(f"sync-ide-assets: missing description: {f}", file=sys.stderr)
        sys.exit(2)
    desc = dm.group(1).strip()
    extra_lines = []
    for key in ("allowed-tools", "argument-hint"):
        km = re.search(rf"^{re.escape(key)}:\s*(.+)$", fm, re.MULTILINE)
        if km:
            extra_lines.append(f"{key}: {km.group(1).strip()}")
    body = text[m.end() :]
    extra = ("\n".join(extra_lines) + "\n") if extra_lines else ""
    out = f"---\ndescription: {desc}\n{extra}---\n{body}"
    dest = gh / f"{f.stem}.prompt.md"
    if check:
        if not dest.exists() or dest.read_text(encoding="utf-8") != out:
            print(f"DRIFT: {dest} (expected content from {f})", file=sys.stderr)
            drift = True
    else:
        dest.parent.mkdir(parents=True, exist_ok=True)
        dest.write_text(out, encoding="utf-8")
# Remove GitHub prompt files for workflows deleted from .agents/workflows.
expected = {f"{src.stem}.prompt.md" for src in files}
for dest in sorted(gh.glob("*.prompt.md")):
    if dest.name not in expected:
        if check:
            print(f"ORPHAN: {dest} (no longer in .agents/workflows)", file=sys.stderr)
            drift = True
        else:
            dest.unlink()
sys.exit(1 if drift else 0)
PY
then
  EXIT=1
fi

exit "$EXIT"
