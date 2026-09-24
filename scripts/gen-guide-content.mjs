#!/usr/bin/env node
// Generate site/layouts/partials/guide-content.html from docs/user-guide/0X-*.md.
//
// Reads the docs IN PLACE (no copy) and emits a single Hugo partial whose
// <h2> anchors match the sidebar links. Run before `hugo build` (see Taskfile
// site:build / gh-pages) so the guide is always current with docs/.
//
// The markdown is inlined into a `{{ '...' | markdownify }}` action. To keep the
// Hugo template parser happy we:
//   - keep backticks AS-IS (fence + inline-code delimiters for goldmark)
//   - escape < to &lt; in PROSE only (code fences keep < literal for Mermaid <br/>)
//   - replace }} and {# with safe placeholders (Gates/CHECK: syntax)
import { readFileSync, writeFileSync } from "node:fs";
import { basename, dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = join(dirname(fileURLToPath(import.meta.url)), "..");
const DOCS = join(ROOT, "docs", "user-guide");
const OUT = join(ROOT, "site", "layouts", "partials", "guide-content.html");

const ORDER = [
  "01-layout.md",
  "02-skills.md",
  "03-workflow-usage.md",
  "04-hooks.md",
  "05-memory-and-indexing.md",
  "06-multi-agent-work.md",
  "07-ticketing.md",
  "08-concepts.md",
];

// Placeholders for }} and {# (Gates/CHECK: syntax). Must not contain
// backticks (raw-string terminators) or }} / {# (template delimiters).
const PH_CLOSE = "SQCLOSE"; // was }}
const PH_COMMENT = "SQCOMMENT"; // was {#

function stripFrontmatter(text) {
  if (text.startsWith("---")) {
    const end = text.indexOf("\n---", 3);
    if (end !== -1) {
      return text.slice(end + 4).replace(/^\n/, "");
    }
  }
  return text;
}

function slug(title) {
  let t = title
    .trim()
    .toLowerCase()
    .replace(/ /g, "-")
    .replace(/&/g, "and");
  t = t.replace(/[^a-z0-9-]/g, "");
  return t;
}

function htmlEscape(text) {
  // For literal HTML text nodes (the <h2> title).
  return text.replace(/&/g, "&amp;").replace(/</g, "&lt;");
}

function extractCodeBlocks(text) {
  // Pull out ALL fenced code blocks. Mermaid blocks become
  // <pre class="mermaid">; others become <pre><code>. The markdown keeps
  // only prose (no backticks, no bare <). Returns [md_without_fences, pre_blocks].
  const pres = [];
  const md = text.replace(/```(\w*)\n([\s\S]*?)```/g, (_, langRaw, src0) => {
    const lang = (langRaw || "").trim();
    let src = src0;
    if (src.endsWith("\n")) {
      src = src.slice(0, -1);
    }
    if (lang.toLowerCase() === "mermaid") {
      pres.push(`<pre class="mermaid">${src}</pre>`);
    } else {
      const escaped = src
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;");
      const cls = lang ? ` class="language-${lang}"` : "";
      pres.push(`<pre><code${cls}>${escaped}</code></pre>`);
    }
    return "\n\n";
  });
  return [md, pres];
}

function tplEscape(text) {
  // Hugo raw strings can't contain backticks or bare <. So:
  // 1. Extract ALL fenced code blocks -> <pre> raw HTML (no backticks,
  //    no bare <). Mermaid blocks keep <br/> literal; others escape <.
  // 2. Strip ALL remaining backticks (inline code -> plain text).
  // 3. Escape < in the remaining prose.
  // 4. Replace }} and {# with safe placeholders.
  // 5. Append the extracted <pre> blocks (raw HTML, outside markdownify).
  let [md, pres] = extractCodeBlocks(text);
  md = md.replace(/`/g, "");
  md = md.replace(/}}/g, PH_CLOSE).replace(/\{#/g, PH_COMMENT);
  md = md.replace(/</g, "&lt;");
  let result = md;
  if (pres.length) {
    result += "\n" + pres.join("\n");
  }
  return result;
}

function renderFile(filePath) {
  const raw = readFileSync(filePath, "utf-8");
  let body = stripFrontmatter(raw);
  const m = body.match(/^#\s+(.+)$/m);
  let title;
  if (m) {
    title = m[1].trim();
    body = body.replace(m[0], "");
  } else {
    title = basename(filePath, ".md");
  }
  // Remove "Guide map" / "Quick path" / "Start here" / "Next step" sections
  // (from the ## heading to the next ## or end of file).
  const sectionRe = /^##\s+(Guide map|Quick path|Start here|Next step)\s*$/m;
  let guard = 0;
  while (sectionRe.test(body) && guard++ < 100) {
    const m = body.match(sectionRe);
    const start = m.index;
    // Find the start of the NEXT ## heading after this one, or end of string.
    const rest = body.slice(start + m[0].length);
    const nextH2 = rest.search(/^##\s/m);
    const end = nextH2 === -1 ? body.length : start + m[0].length + nextH2;
    body = body.slice(0, start) + body.slice(end);
  }
  body = body.trim();
  return [title, body];
}

function main() {
  const parts = [];
  const missing = [];
  for (const fname of ORDER) {
    const p = join(DOCS, fname);
    let exists = true;
    try {
      readFileSync(p);
    } catch {
      exists = false;
    }
    if (!exists) {
      missing.push(fname);
      continue;
    }
    const [title, body] = renderFile(p);
    parts.push(`<h2 id="${slug(title)}">${htmlEscape(title)}</h2>`);
    parts.push(
      '<div class="doc-body">{{ `' + tplEscape(body) + '` | markdownify }}</div>'
    );
  }
  if (missing.length) {
    console.error(`WARN: missing docs: ${missing.join(", ")}`);
  }
  writeFileSync(OUT, parts.join("\n\n") + "\n", "utf-8");
  console.log(`wrote ${OUT} (${parts.length} blocks)`);
}

main();
