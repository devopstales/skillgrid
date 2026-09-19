#!/usr/bin/env python3
"""Generate site/layouts/partials/guide-content.html from docs/user-guide/0X-*.md.

Reads the docs IN PLACE (no copy) and emits a single Hugo partial whose
<h2> anchors match the sidebar links. Run before `hugo build` (see Taskfile
site:build / gh-pages) so the guide is always current with docs/.

The markdown is inlined into a `{{ `...` | markdownify }}` action. To keep the
Hugo template parser happy we:
  - keep backticks AS-IS (fence + inline-code delimiters for goldmark)
  - escape < to &lt; in PROSE only (code fences keep < literal for Mermaid <br/>)
  - replace }} and {# with safe placeholders (Gates/CHECK: syntax)
"""
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent  # repo root
DOCS = ROOT / "docs" / "user-guide"
OUT = ROOT / "site" / "layouts" / "partials" / "guide-content.html"

ORDER = [
    "01-layout.md",
    "02-skills.md",
    "03-workflow-usage.md",
    "04-hooks.md",
    "05-memory-and-indexing.md",
    "06-multi-agent-work.md",
    "07-ticketing.md",
    "08-concepts.md",
]

# Placeholders for }} and {# (Gates/CHECK: syntax). Must not contain
# backticks (raw-string terminators) or }} / {# (template delimiters).
_PH_CLOSE = "SQCLOSE"      # was }}
_PH_COMMENT = "SQCOMMENT"  # was {#
_PH_FENCE = "SF"           # was ``` (2-char, no backticks)


def strip_frontmatter(text: str) -> str:
    if text.startswith("---"):
        end = text.find("\n---", 3)
        if end != -1:
            return text[end + 4 :].lstrip("\n")
    return text


def slug(title: str) -> str:
    t = title.strip().lower().replace(" ", "-").replace("&", "and")
    t = re.sub(r"[^a-z0-9\-]", "", t)
    return t


def html_escape(text: str) -> str:
    # For literal HTML text nodes (the <h2> title).
    return text.replace("&", "&amp;").replace("<", "&lt;")


def _fix_odd_backticks(part: str) -> str:
    # Safety net: if a prose segment has an odd backtick count (unclosed
    # inline-code span in the source), strip backticks so the Hugo raw string
    # stays balanced. Pairs (even count) are kept for goldmark inline code.
    if part.count("`") % 2 != 0:
        return part.replace("`", "")
    return part


def extract_mermaid(text: str) -> tuple:
    """Pull out ```mermaid blocks as <pre class="mermaid"> and replace them
    with a placeholder in the markdown. Returns (markdown_without_mermaid,
    [pre_blocks])."""
    pres = []

    def _repl(m):
        src = m.group(1).strip()
        pres.append(f'<pre class="mermaid">{src}</pre>')
        return "\n\n"

    md = re.sub(r"```mermaid\n(.*?)```", _repl, text, flags=re.S)
    return md, pres


def tpl_escape(text: str) -> str:
    # Hugo raw strings can't contain backticks. So:
    # 1. Extract ```mermaid blocks -> <pre class="mermaid"> (no backticks).
    # 2. Replace remaining ``` fences with SF (2-char placeholder).
    # 3. Strip ALL remaining backticks (inline code -> plain text).
    # 4. Escape < in prose (keep literal in code for <br/> etc.).
    # 5. Replace }} and {# with safe placeholders.
    text, pres = extract_mermaid(text)
    text = text.replace("```", _PH_FENCE)
    parts = text.split(_PH_FENCE)
    out = []
    for i, part in enumerate(parts):
        part = part.replace("`", "")
        part = part.replace("}}", _PH_CLOSE).replace("{#", _PH_COMMENT)
        if i % 2 == 0:
            part = part.replace("<", "&lt;")
        out.append(part)
    result = _PH_FENCE.join(out)
    # Re-insert the <pre> mermaid blocks: they were at fence positions,
    # which are now SF delimiters. We track them by re-splitting the
    # original text on the mermaid regex to find positions.
    # Simpler: emit all <pre> blocks after the markdownify div.
    if pres:
        result += "\n" + "\n".join(pres)
    return result


def render_file(path: Path) -> str:
    raw = path.read_text(encoding="utf-8")
    body = strip_frontmatter(raw)
    m = re.search(r"^#\s+(.+)$", body, re.M)
    if m:
        title = m.group(1).strip()
        body = body.replace(m.group(0), "", 1)
    else:
        title = path.stem
    body = re.sub(
        r"(?m)^##\s+(Guide map|Quick path|Start here|Next step)\s*$.*?(?=^##\s|\Z)",
        "",
        body,
        flags=re.S,
    )
    body = body.strip()
    return title, body


def main() -> int:
    parts = []
    missing = []
    for fname in ORDER:
        p = DOCS / fname
        if not p.exists():
            missing.append(fname)
            continue
        title, body = render_file(p)
        parts.append(f'<h2 id="{slug(title)}">{html_escape(title)}</h2>')
        parts.append('<div class="doc-body">{{ `' + tpl_escape(body) + '` | markdownify }}</div>')
        # (raw string delimited by single backtick; tpl_escape replaces ```
        #  with a 2-char placeholder so no consecutive backticks remain)
    if missing:
        print(f"WARN: missing docs: {missing}", file=sys.stderr)
    OUT.write_text("\n\n".join(parts) + "\n", encoding="utf-8")
    print(f"wrote {OUT} ({len(parts)} blocks)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
