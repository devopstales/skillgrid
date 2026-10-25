#!/usr/bin/env node
/**
 * sources.mjs — citation ledger for grounded answers and documents.
 *
 * The Node port of sources.py (plan item J: single deterministic runtime).
 * Owns the url → [n] mapping used by the grounded-citations skill. Ids are
 * assigned at retrieval time and never change, so a draft's [3] always
 * resolves to the same page. The model only ever emits integers the ledger
 * handed it, which is what makes the citations verifiable.
 *
 * Subcommands (identical contract to sources.py):
 *   reset                            start a clean ledger
 *   add URL [URL ...]                register source(s), print their ids
 *   ingest FILE|-                    register every url found in JSON tool output
 *   quote ID --text T --from FILE|-  attach verbatim supporting evidence to a source
 *   list                             show the ledger
 *   render                           render a Sources block
 *   verify DRAFT                     check a draft's citations against the ledger
 *
 * Ledger path resolution (first wins):
 *   --ledger PATH
 *   $SKILLGRID_CITATION_LEDGER
 *   <project>/.skillgrid/sdd/citations/ledger.json (gitignored scratch)
 *
 * Ported from the Hermes grounded-citations skill under MIT (ADR-0015);
 * the Python original is retained as sources.py until this port is proven
 * equivalent, then the skill switches to `node sources.mjs`.
 */

import {
  readFileSync,
  writeFileSync,
  existsSync,
  mkdirSync,
  renameSync,
  openSync,
  closeSync,
  unlinkSync,
  readdirSync,
  statSync,
  readSync,
} from "node:fs";
import { resolve, dirname, join } from "node:path";
import { cwd } from "node:process";
import { performance } from "node:perf_hooks";

const SCHEMA_VERSION = 1;

// A citation marker in prose: [12]. Markdown links ([text](url)) and
// reference-style labels are excluded by requiring digits only and no
// following "(" or ":".
// Global form for matchAll; TEST form (no /g) for .test() so lastIndex never
// advances. Both must stay in sync.
const CITE_RE = /\[(\d{1,4})\](?![(:])/g;
const CITE_TEST_RE = /\[(\d{1,4})\](?![(:])/;
const SOURCES_HEADER_RE = /^\s*(?:#{1,6}\s*)?(?:\*\*)?sources:?(?:\*\*)?\s*$/i;
const SOURCE_LINE_RE = /^\s*\[(\d{1,4})\]\s*[-–:]?\s*(\S+)/;
const URL_IN_TEXT_RE = /https?:\/\/[^\s"'<>)\]}]+/;
const FENCE_RE = /^\s*(?:```|~~~)/;
const UNVERIFIED_RE = /\[unverified\]/i;
// Markdown artifacts that retrieval tools inject into otherwise-identical prose.
const MD_LINK_RE = /\[([^\]]*)\]\((?:[^()\s]|\([^()]*\))*\)/g;
const MD_NOISE_RE = /[*_`~]|\\(?=[^\w\s])/g;

// ---------------------------------------------------------------------------
// Scratch / ledger path resolution (port of _skillgrid_scratch.py)
// ---------------------------------------------------------------------------

function projectRoot() {
  let cur = resolve(cwd());
  while (true) {
    if (
      (existsSync(join(cur, ".skillgrid")) && statSync(join(cur, ".skillgrid")).isDirectory()) ||
      existsSync(join(cur, ".git"))
    ) {
      return cur;
    }
    const parent = dirname(cur);
    if (parent === cur) return cur;
    cur = parent;
  }
}

function projectScratch() {
  return join(projectRoot(), ".skillgrid", "sdd");
}

function resolveLedgerPath(explicit) {
  if (explicit) return expandHome(explicit);
  const env = (process.env.SKILLGRID_CITATION_LEDGER || "").trim();
  if (env) return expandHome(env);
  return join(projectScratch(), "citations", "ledger.json");
}

function expandHome(p) {
  if (p.startsWith("~/")) return join(resolve(cwd()), p.slice(2));
  return p;
}

// ---------------------------------------------------------------------------
// Ledger I/O
// ---------------------------------------------------------------------------

function normalizeUrl(url) {
  let u = (url || "").trim();
  if (u.includes("#")) u = u.split("#")[0];
  const stripped = u.replace(/\/+$/, "");
  return stripped || u;
}

function fail(msg) {
  console.error(msg);
  process.exit(2);
}

function loadLedger(path) {
  if (!existsSync(path)) return { version: SCHEMA_VERSION, sources: [] };
  let data;
  try {
    data = JSON.parse(readFileSync(path, "utf8"));
  } catch (exc) {
    fail(`error: ledger at ${path} is unreadable (${exc.message}); run \`reset\` to start over`);
  }
  if (typeof data !== "object" || data === null || !Array.isArray(data.sources)) {
    fail(`error: ledger at ${path} has an unexpected shape; run \`reset\``);
  }
  return data;
}

function saveLedger(path, data) {
  mkdirSync(dirname(path), { recursive: true });
  const tmp = path + `.tmp${process.pid}`;
  writeFileSync(tmp, JSON.stringify(data, null, 2) + "\n", "utf8");
  renameSync(tmp, path);
}

// Best-effort cross-process lock (O_EXCL lockfile). Parallel subagents can
// share one ledger via --ledger; without a lock two concurrent add calls can
// assign the same id. Falls through after the timeout — a stale lock must
// never wedge the ledger.
class LedgerLock {
  constructor(path, timeoutMs = 5000) {
    this.lockPath = path + ".lock";
    this.timeout = timeoutMs;
  }
  acquire() {
    mkdirSync(dirname(this.lockPath), { recursive: true });
    const deadline = performance.now() + this.timeout;
    for (;;) {
      try {
        openSync(this.lockPath, "wx");
        return;
      } catch {
        if (performance.now() >= deadline) {
          try { unlinkSync(this.lockPath); } catch { return; }
          continue;
        }
        Atomics.wait(new Int32Array(new SharedArrayBuffer(4)), 0, 0, 50);
      }
    }
  }
  release() {
    try { unlinkSync(this.lockPath); } catch {}
  }
}

function byUrl(sources) {
  const m = {};
  for (const s of sources) m[s.url] = s;
  return m;
}

function addSources(path, urls, title = null, accessed = null) {
  urls = urls.map((u) => String(u).trim()).filter(Boolean);
  if (!urls.length) return [];
  const lock = new LedgerLock(path);
  lock.acquire();
  try {
    const data = loadLedger(path);
    const sources = data.sources;
    const index = byUrl(sources);
    const out = [];
    let changed = false;
    for (const raw of urls) {
      const key = normalizeUrl(raw);
      const existing = index[key];
      if (existing) {
        if (title && !existing.title) {
          existing.title = title;
          changed = true;
        }
        out.push(existing);
        continue;
      }
      const entry = {
        id: sources.length + 1,
        url: key,
        title: (title || "").trim(),
        accessed: accessed || today(),
      };
      sources.push(entry);
      index[key] = entry;
      out.push(entry);
      changed = true;
    }
    if (changed) saveLedger(path, data);
    return out;
  } finally {
    lock.release();
  }
}

function today() {
  return new Date().toISOString().slice(0, 10);
}

function urlsFromJson(payload) {
  const found = [];
  const seen = new Set();
  function walk(node) {
    if (node && typeof node === "object" && !Array.isArray(node)) {
      const url = node.url || node.link || node.source_url;
      if (typeof url === "string" && (url.startsWith("http://") || url.startsWith("https://"))) {
        const key = normalizeUrl(url);
        if (!seen.has(key)) {
          seen.add(key);
          const t = node.title || node.name || "";
          found.push([url, typeof t === "string" ? t : ""]);
        }
      }
      for (const value of Object.values(node)) walk(value);
    } else if (Array.isArray(node)) {
      for (const item of node) walk(item);
    }
  }
  walk(payload);
  return found;
}

// ---------------------------------------------------------------------------
// Evidence quotes (fact-checking)
// ---------------------------------------------------------------------------

function normalizeWs(text) {
  return (text || "").split(/\s+/).filter(Boolean).join(" ");
}

function matchKey(text) {
  const collapsed = (text || "").replace(MD_LINK_RE, "$1");
  return normalizeWs(collapsed.replace(MD_NOISE_RE, "")).toLowerCase();
}

function quoteInEvidence(quote, evidence) {
  const q = matchKey(quote);
  return q.length > 0 && matchKey(evidence).includes(q);
}

function attachQuote(path, sourceId, quote, evidence) {
  quote = (quote || "").trim();
  if (normalizeWs(quote).split(/\s+/).filter(Boolean).length < 3) {
    fail("error: quote too short — use at least 3 words of verbatim text");
  }
  if (!quoteInEvidence(quote, evidence)) {
    fail("error: quote not found verbatim in the evidence text — copy the exact wording from the fetched page, do not paraphrase");
  }
  const lock = new LedgerLock(path);
  lock.acquire();
  try {
    const data = loadLedger(path);
    const entry = data.sources.find((s) => s.id === sourceId) || null;
    if (!entry) fail(`error: no source [${sourceId}] in the ledger`);
    if (!Array.isArray(entry.quotes)) entry.quotes = [];
    const norm = matchKey(quote);
    if (!entry.quotes.some((q) => matchKey(q.text || "") === norm)) {
      entry.quotes.push({ text: quote, added: today() });
      saveLedger(path, data);
    }
    return entry;
  } finally {
    lock.release();
  }
}

function renderSources(sources, style = "markdown", only = null) {
  const picked = sources.filter((s) => only === null || only.has(s.id));
  picked.sort((a, b) => a.id - b.id);
  if (!picked.length) return "";
  const lines = [];
  if (style === "bibtex") {
    for (const s of picked) {
      const key = `source${s.id}`;
      const title = s.title || s.url;
      lines.push(`@misc{${key},\n  title = {${title}},\n  howpublished = {\\url{${s.url}}},\n  note = {Accessed ${s.accessed || ""}}\n}`);
    }
    return lines.join("\n");
  }
  if (style === "footnotes") {
    for (const s of picked) {
      const title = s.title;
      const suffix = title ? ` — ${title}` : "";
      lines.push(`[^${s.id}]: ${s.url}${suffix}`);
    }
    return lines.join("\n");
  }
  const header = style === "plain" ? "Sources:" : "## Sources";
  lines.push(header);
  if (style !== "plain") lines.push("");
  for (const s of picked) {
    const title = s.title;
    const suffix = title ? ` — ${title}` : "";
    lines.push(`[${s.id}] ${s.url}${suffix}`);
    if (style === "evidence") {
      for (const q of s.quotes || []) lines.push(`    > "${q.text || ""}"`);
    }
  }
  return lines.join("\n");
}

// ---------------------------------------------------------------------------
// Verification
// ---------------------------------------------------------------------------

function splitDraft(text) {
  const lines = text.split("\n");
  let headerIdx = -1;
  for (let i = 0; i < lines.length; i++) {
    if (SOURCES_HEADER_RE.test(lines[i])) headerIdx = i;
  }
  const listed = {};
  if (headerIdx >= 0) {
    for (const line of lines.slice(headerIdx + 1)) {
      const m = line.match(SOURCE_LINE_RE);
      if (m) {
        const urlMatch = line.match(URL_IN_TEXT_RE);
        listed[parseInt(m[1], 10)] = urlMatch ? urlMatch[0] : m[2];
      }
    }
  }
  const bodyLines = headerIdx >= 0 ? lines.slice(0, headerIdx) : lines;
  const prose = [];
  let inFence = false;
  for (const line of bodyLines) {
    if (FENCE_RE.test(line)) {
      inFence = !inFence;
      continue;
    }
    if (!inFence) prose.push(line);
  }
  return [prose.join("\n"), listed];
}

function stripSourcesBlock(text) {
  const lines = text.split("\n");
  let headerIdx = -1;
  for (let i = 0; i < lines.length; i++) {
    if (SOURCES_HEADER_RE.test(lines[i])) headerIdx = i;
  }
  if (headerIdx < 0) return text;
  return lines.slice(0, headerIdx).join("\n");
}

function sentences(prose) {
  const out = [];
  for (const line of prose.split("\n")) {
    let stripped = line.trim();
    if (!stripped || stripped.startsWith("#") || stripped.startsWith("|")) continue;
    if (stripped.startsWith(">")) stripped = stripped.replace(/^>+\s*/, "").trim();
    for (const part of stripped.split(/(?<=[.!?])\s+/)) {
      const p = part.trim();
      if (p.split(/\s+/).filter(Boolean).length >= 4) out.push(p);
    }
  }
  return out;
}

function verifyDraft(draftPath, sources, { strict = false, minCoverage = null, requireEvidence = false } = {}) {
  const text = readFileSync(draftPath, "utf8");
  const [prose, listed] = splitDraft(text);
  const byId = {};
  for (const s of sources) byId[s.id] = s;

  const errors = [];
  const warnings = [];

  // CITE_RE has a capture group; String.match(global) returns the full match
  // (e.g. "[1]"), so re-match to pull the digits. Use a non-global clone so
  // .test() calls below aren't affected by lastIndex state.
  const cited = [...prose.matchAll(/\[(\d{1,4})\](?![(:])/g)].map((m) => parseInt(m[1], 10));
  const citedSet = new Set(cited);

  const unknown = [...citedSet].filter((i) => !(i in byId)).sort((a, b) => a - b);
  if (unknown.length) {
    errors.push("citations not in the ledger (hallucinated or renumbered): " + unknown.map((i) => `[${i}]`).join(", "));
  }

  if (citedSet.size && !Object.keys(listed).length) {
    errors.push("draft cites sources but has no `Sources:` block — run `render --cited-in`");
  }

  const listedKeys = Object.keys(listed).map((k) => parseInt(k, 10));
  // Cited in prose but absent from the Sources block.
  const missingFromBlock = [...citedSet].filter((i) => !listedKeys.includes(i)).sort((a, b) => a - b);
  if (missingFromBlock.length) {
    errors.push("cited but absent from the Sources block: " + missingFromBlock.map((i) => `[${i}]`).join(", "));
  }

  for (const sid of [...listedKeys].sort((a, b) => a - b)) {
    const url = listed[sid];
    const entry = byId[sid];
    if (!entry) {
      errors.push(`Sources block lists [${sid}], which is not in the ledger`);
      continue;
    }
    if (normalizeUrl(url) !== entry.url) {
      errors.push(`Sources block URL for [${sid}] does not match the ledger (block: ${url} / ledger: ${entry.url}) — re-run \`render\``);
    }
  }

  const extraInBlock = listedKeys.filter((i) => !citedSet.has(i)).sort((a, b) => a - b);
  if (extraInBlock.length) {
    warnings.push("listed in Sources but never cited inline: " + extraInBlock.map((i) => `[${i}]`).join(", "));
  }

  const registeredUncited = Object.keys(byId).map((k) => parseInt(k, 10)).filter((i) => !citedSet.has(i)).sort((a, b) => a - b);
  if (registeredUncited.length) {
    warnings.push("registered in the ledger but not cited in this draft: " + registeredUncited.map((i) => `[${i}]`).join(", "));
  }

  const allSentences = sentences(prose);
  const citedSentences = allSentences.filter((s) => CITE_TEST_RE.test(s));
  const unverifiedSentences = allSentences.filter((s) => UNVERIFIED_RE.test(s));
  const covered = allSentences.filter((s) => CITE_TEST_RE.test(s) || UNVERIFIED_RE.test(s));
  const coverage = allSentences.length ? covered.length / allSentences.length : 0.0;
  if (minCoverage !== null && allSentences.length && coverage < minCoverage) {
    errors.push(
      `citation coverage ${Math.round(coverage * 100)}% is below the required ${Math.round(minCoverage * 100)}% ` +
      `(${covered.length}/${allSentences.length} sentences cited or marked [unverified])`
    );
  }

  if (requireEvidence) {
    const unevidenced = [...citedSet].filter((i) => i in byId && !byId[i].quotes).sort((a, b) => a - b);
    if (unevidenced.length) {
      errors.push("cited sources carry no verbatim evidence quote (run `quote` with the fetched page text): " + unevidenced.map((i) => `[${i}]`).join(", "));
    }
  }

  const overCited = allSentences.filter((s) => [...s.matchAll(CITE_RE)].length > 3);
  if (overCited.length) {
    warnings.push(`${overCited.length} sentence(s) carry more than 3 citations`);
  }

  const code = errors.length ? 1 : strict && warnings.length ? 1 : 0;
  const quoted = sources.filter((s) => s.quotes).length;
  const stats =
    `${allSentences.length} prose sentence(s), ${covered.length} with declared provenance ` +
    `(${Math.round(coverage * 100)}%) — ${citedSentences.length} cited, ` +
    `${unverifiedSentences.length} marked [unverified] (a sentence may be both); ` +
    `${citedSet.size} distinct source(s) cited, ` +
    `${Object.keys(byId).length} in ledger (${quoted} with evidence quotes)`;
  warnings.unshift(`stats: ${stats}`);
  return [code, errors, warnings];
}

function parseOnly(spec) {
  if (!spec) return null;
  const out = new Set();
  for (const chunk of spec.replace(/ /g, "").split(",")) {
    if (!chunk) continue;
    if (chunk.includes("-")) {
      const [lo, hi] = chunk.split("-");
      for (let i = parseInt(lo, 10); i <= parseInt(hi, 10); i++) out.add(i);
    } else {
      out.add(parseInt(chunk, 10));
    }
  }
  return out;
}

function readMaybeStdin(file) {
  if (file === "-") {
    const fd = 0;
    const buf = Buffer.alloc(1024 * 1024);
    let out = "";
    let bytes;
    while ((bytes = readSync(fd, buf, 0, buf.length, null)) !== 0) {
      out += buf.toString("utf8", 0, bytes);
    }
    return out;
  }
  return readFileSync(file, "utf8");
}

// ---------------------------------------------------------------------------
// CLI
// ---------------------------------------------------------------------------

function main() {
  const argv = process.argv.slice(2);
  // --ledger <path> is a global flag (may precede the subcommand).
  let ledgerArg = null;
  let rest = [];
  for (let i = 0; i < argv.length; i++) {
    if (argv[i] === "--ledger") {
      ledgerArg = argv[++i];
    } else {
      rest.push(argv[i]);
    }
  }
  const cmd = rest[0];
  const args = rest.slice(1);

  if (!cmd) {
    console.error("Usage: sources.mjs [--ledger PATH] <reset|add|ingest|quote|list|render|verify> …");
    process.exit(2);
  }

  // Collect flags + positionals for this subcommand.
  const flags = {};
  const positional = [];
  for (let i = 0; i < args.length; i++) {
    const a = args[i];
    if (a === "--title") flags.title = args[++i];
    else if (a === "--accessed") flags.accessed = args[++i];
    else if (a === "--text") flags.text = args[++i];
    else if (a === "--from") flags.evidence = args[++i];
    else if (a === "--style") flags.style = args[++i];
    else if (a === "--only") flags.only = args[++i];
    else if (a === "--cited-in") flags.citedIn = args[++i];
    else if (a === "--replace-in") flags.replaceIn = args[++i];
    else if (a === "--min-coverage") flags.minCoverage = parseFloat(args[++i]);
    else if (a === "--strict") flags.strict = true;
    else if (a === "--evidence") flags.evidenceFlag = true;
    else if (a === "--json") flags.json = true;
    else positional.push(a);
  }

  const path = resolveLedgerPath(ledgerArg);

  if (cmd === "reset") {
    const lock = new LedgerLock(path);
    lock.acquire();
    try {
      saveLedger(path, { version: SCHEMA_VERSION, sources: [] });
    } finally {
      lock.release();
    }
    console.log(`ledger reset: ${path}`);
    process.exit(0);
  }

  if (cmd === "add") {
    const title = positional.length === 1 ? flags.title || null : null;
    const entries = addSources(path, positional, title, flags.accessed || null);
    if (flags.json) console.log(JSON.stringify(entries, null, 2));
    else for (const e of entries) console.log(`[${e.id}] ${e.url}`);
    process.exit(0);
  }

  if (cmd === "ingest") {
    const raw = readMaybeStdin(positional[0]);
    let payload;
    try {
      payload = JSON.parse(raw);
    } catch (exc) {
      console.error(`error: input is not valid JSON (${exc.message})`);
      process.exit(2);
    }
    const pairs = urlsFromJson(payload);
    if (!pairs.length) {
      console.error("no urls found in input");
      process.exit(1);
    }
    for (const [url, title] of pairs) {
      const entry = addSources(path, [url], title || null)[0];
      console.log(`[${entry.id}] ${entry.url}`);
    }
    process.exit(0);
  }

  if (cmd === "quote") {
    const id = parseInt(positional[0], 10);
    const raw = readMaybeStdin(flags.evidence);
    const entry = attachQuote(path, id, flags.text, raw);
    console.log(`[${entry.id}] evidence attached (${(entry.quotes || []).length} quote(s))`);
    process.exit(0);
  }

  const data = loadLedger(path);
  const sources = [...data.sources].sort((a, b) => a.id - b.id);

  if (cmd === "list") {
    if (flags.json) console.log(JSON.stringify(sources, null, 2));
    else if (!sources.length) console.log(`ledger is empty: ${path}`);
    else {
      for (const s of sources) {
        const title = s.title ? `  ${s.title}` : "";
        const nq = (s.quotes || []).length;
        const mark = nq ? `  (${nq} quote${nq !== 1 ? "s" : ""})` : "";
        console.log(`[${s.id}] ${s.url}${title}${mark}`);
      }
    }
    process.exit(0);
  }

  if (cmd === "render") {
    let only = parseOnly(flags.only);
    const draftForIds = flags.replaceIn || flags.citedIn;
    if (draftForIds) {
      const draft = readFileSync(draftForIds, "utf8");
      const [prose] = splitDraft(draft);
      const cited = new Set([...prose.matchAll(CITE_RE)].map((m) => parseInt(m[1], 10)));
      only = only === null ? cited : new Set([...only].filter((i) => cited.has(i)));
    }
    const block = renderSources(sources, flags.style || "markdown", only);
    if (!block) {
      console.error("no sources to render");
      process.exit(1);
    }
    if (flags.replaceIn) {
      const target = flags.replaceIn;
      const body = stripSourcesBlock(readFileSync(target, "utf8"));
      writeFileSync(target, body.replace(/\n*$/, "") + "\n\n" + block + "\n", "utf8");
      console.log(`Sources block rewritten in ${target}`);
      process.exit(0);
    }
    console.log(block);
    process.exit(0);
  }

  if (cmd === "verify") {
    const draftPath = positional[0];
    if (!existsSync(draftPath)) {
      console.error(`error: no such draft: ${draftPath}`);
      process.exit(2);
    }
    const [code, errors, warnings] = verifyDraft(draftPath, sources, {
      strict: !!flags.strict,
      minCoverage: flags.minCoverage ?? null,
      requireEvidence: !!flags.evidenceFlag,
    });
    for (const w of warnings) {
      const prefix = w.startsWith("stats: ") ? "info" : "warn";
      console.log(`${prefix}: ${w}`);
    }
    for (const e of errors) console.error(`FAIL: ${e}`);
    console.log(code === 0 ? "citations OK" : "verification failed");
    process.exit(code);
  }

  process.exit(2);
}

main();
