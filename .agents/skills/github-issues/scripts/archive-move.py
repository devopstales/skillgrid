#!/usr/bin/env python3
"""Move an archived change from Active Changes to Archived Changes in a tracking (Epic) issue body.

Usage:
    python3 archive-move.py <body-file> <change-issue-number> <change-name>

Reads the issue body from <body-file>, rewrites it in place. Idempotent: re-running with the
same arguments does not duplicate entries.

Why not sed: `sed 's/## Active Changes/## Archived Changes\\n- ...\\n\\n## Active Changes/'`
inserts a second "## Archived Changes" heading and leaves the entry under Active Changes.
"""
import sys


def main() -> int:
    if len(sys.argv) != 4:
        print(__doc__.strip(), file=sys.stderr)
        return 2

    path, num, name = sys.argv[1], sys.argv[2], sys.argv[3]
    lines = open(path).read().splitlines()

    try:
        a = lines.index("## Active Changes")
        z = lines.index("## Archived Changes")
    except ValueError:
        print("error: body needs both '## Active Changes' and '## Archived Changes'", file=sys.stderr)
        return 1

    # The archived block runs until the footer rule or end of body.
    end = z + 1
    while end < len(lines) and not lines[end].startswith("---"):
        end += 1

    active, archived = lines[a + 1 : z], lines[z + 1 : end]

    moved = [l for l in active if l.startswith("- ") and f"#{num} " in l]
    active = [l for l in active if l not in moved]

    already = [l for l in archived if l.startswith("- ") and f"#{num} " in l]
    if not moved and not already:
        moved = [f"- Closes #{num} — {name}"]

    archived = [l for l in archived if l.strip() != "- None yet"]

    def tidy(block, empty="- None"):
        block = [l for l in block if l.strip()]
        return (block if block else [empty]) + [""]

    out = (
        lines[: a + 1]
        + tidy(active)
        + ["## Archived Changes"]
        + tidy(archived + moved)
        + lines[end:]
    )
    open(path, "w").write("\n".join(out).rstrip() + "\n")
    return 0


if __name__ == "__main__":
    sys.exit(main())
