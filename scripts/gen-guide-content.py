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


def extract_code_blocks(text: str) -> tuple:
    """Pull out ALL fenced code blocks. Mermaid blocks become
    <pre class="mermaid">; others become <pre><code>. The markdown keeps
    only prose (no backticks, no bare <). Returns (md_without_fences,
    [pre_blocks])."""
    pres = []

    def _repl(m):
        lang = (m.group(1) or "").strip()
        src = m.group(2)
        if src.endswith("\n"):
            src = src[:-1]
        if lang.lower() == "mermaid":
            pres.append(f'<pre class="mermaid">{src}</pre>')
        else:
            escaped = (
                src.replace("&", "&amp;")
                .replace("<", "&lt;")
                .replace(">", "&gt;")
            )
            cls = f' class="language-{lang}"' if lang else ""
            pres.append(f"<pre><code{cls}>{escaped}</code></pre>")
        return "\n\n"

    md = re.sub(r"```(\w*)\n(.*?)```", _repl, text, flags=re.S)
    return md, pres


def tpl_escape(text: str) -> str:
    # Hugo raw strings can't contain backticks or bare <. So:
    # 1. Extract ALL fenced code blocks -> <pre> raw HTML (no backticks,
    #    no bare <). Mermaid blocks keep <br/> literal; others escape <.
    # 2. Strip ALL remaining backticks (inline code -> plain text).
    # 3. Escape < in the remaining prose.
    # 4. Replace }} and {# with safe placeholders.
    # 5. Append the extracted <pre> blocks (raw HTML, outside markdownify).
    text, pres = extract_code_blocks(text)
    text = text.replace("`", "")
    text = text.replace("}}", _PH_CLOSE).replace("{#", _PH_COMMENT)
    text = text.replace("<", "&lt;")
    result = text
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
