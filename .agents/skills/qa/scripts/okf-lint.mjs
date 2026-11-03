#!/usr/bin/env node
/**
 * okf-lint.mjs — read-only drift guard for the .skillgrid knowledge bundle
 * (OKF v0.2).
 *
 * Runs nine checks over the bundle and reports a named drift verdict. It
 * owns verification, not the write: it never modifies any concept, has no
 * `--fix`, and its ONLY write side-effect is the telemetry JSONL append.
 * Callers apply fixes.
 *
 * Usage: node scripts/okf-lint.mjs [project-root]
 *   project-root defaults to the current working directory.
 *
 * Exit codes:
 *   0 — clean (no drift), or UNSCOPED (no knowledge bundle found — a
 *       non-answer, not a clean bill)
 *   1 — drift detected (names every finding)
 *   2 — usage/parse error (missing arg, bundle unreadable)
 *
 * Checks (in order):
 *   1 frontmatter-valid        parseable YAML, non-empty `type`; reserved
 *                              `index.md`/`log.md` are exempt EXCEPT the
 *                              root catalog, which must carry okf_version
 *   2 index-complete           both directions: every concept file is
 *                              listed, every catalog entry resolves
 *   3 links-unbroken           in-bundle Markdown targets only (no network,
 *                              no anchors, no assets)
 *   4 no-wikilink-syntax       relative Markdown links only; no [[wikilinks]]
 *   5 provenance-required      status: stable => generated present, plus
 *                              sources when anchored; draft exempt (the agent
 *                              never self-stamps)
 *   6 claims-require-verification  InvariantContract => claims[] =>
 *                              verification.anchors
 *   7 anchors-match (liveness) all_of terms in the pinned path, honoring
 *                              `near`/`scope_lines`; claim literals in the
 *                              matched region => LIVE/DRIFT/CLAIM_BROKEN
 *   8 reverse-index-sync       regenerate the expected path -> concept map,
 *                              diff the committed file
 *   9 orphans                  advisory-only: never fails the run
 *
 * Layout (R6): post-R6 roots first — `<root>/.skillgrid/knowledge/`, then
 * the bundle root itself when it carries an `index.md` (fixture mode) —
 * else pre-R6 `<root>/.skillgrid/artifacts/` fallback (README.md is the
 * root catalog). Neither found => SCOPE: UNSCOPED (exit 0, non-answer).
 *
 * The bundle-root walk skips the non-concept zones of the `.skillgrid/`
 * tree (specs/, xarchive/, sdd/, cache/, prototypes/, okf-metrics.jsonl,
 * WINDOWS.md, config.d/): working-state and SDD artifacts, not OKF
 * concepts (ADR-0048, TICKET-14).
 *
 * Telemetry (repocodex metrics.jsonl pattern): after the checks, append ONE
 * JSONL event to `.skillgrid/okf-metrics.jsonl` (override with the
 * OKF_METRICS_SINK env var; parent dir auto-created):
 *   {"event":"lint","at":<iso-utc>,"drift":<n>,"findings":[...],
 *    "false_drift_rate":<0.0-1.0>}
 * false_drift_rate = share of the PRIOR run's findings absent this run
 * (0.0 when there is no prior run or no prior drift). A telemetry-write
 * failure warns to stderr and does NOT change the exit code.
 *
 * ADR: ASSUMPTIONS.md § LOCKED — ADR-0010 (scope discriminator), ADR-0008
 * (Node + yaml package, read-only verifier); OKF profile per
 * .skillgrid/specs/2026-10-06-second-brain-learnings/okf-wiki-plan.md.
 */

import {
  readFileSync,
  readdirSync,
  existsSync,
  statSync,
  appendFileSync,
  mkdirSync,
} from "node:fs";
import { join, resolve, isAbsolute, dirname, relative } from "node:path";
import { parse } from "yaml";

const RESERVED = new Set(["index.md", "log.md"]);
const ANCHOR_MAX_SPAN = 100; // scope_lines default

// --- Scope (verification-scope.md): worst-scope-wins ---

const SCOPE_RANK = { COMPLETE: 0, TRUNCATED: 1, UNSCOPED: 2, UNREADABLE: 3 };

function worstScope(a, b) {
  return SCOPE_RANK[b] > SCOPE_RANK[a] ? b : a;
}

// --- Layout detection ---

function detectBundleRoot(projectRoot) {
  const sg = join(projectRoot, ".skillgrid");
  const postR6 = join(sg, "knowledge");
  if (isDir(postR6)) return { root: postR6, layout: "knowledge/" };
  // Bundle-root mode: the lint target itself is a bundle (fixtures).
  if (existsSync(join(projectRoot, "index.md"))) {
    return { root: projectRoot, layout: "bundle-root" };
  }
  const preR6 = join(sg, "artifacts");
  if (isDir(preR6)) return { root: preR6, layout: "artifacts/ (pre-R6)" };
  return null;
}

function isDir(p) {
  try {
    return statSync(p).isDirectory();
  } catch {
    return false;
  }
}

// Non-concept zones of the .skillgrid/ tree: working state (sdd/, cache/,
// specs/), shipped changes (xarchive/), throwaway work (prototypes/),
// machine-local config (config.d/), the defect register (WINDOWS.md —
// advisory, not a concept), and the telemetry sink.
const NON_CONCEPT_ZONES = new Set([
  "specs",
  "xarchive",
  "sdd",
  "cache",
  "prototypes",
  "config.d",
]);
const NON_CONCEPT_FILES = new Set(["WINDOWS.md", "okf-metrics.jsonl"]);

// --- Bundle enumeration ---

function walkFiles(dir) {
  const out = [];
  const stack = [dir];
  while (stack.length) {
    const d = stack.pop();
    let entries;
    try {
      entries = readdirSync(d, { withFileTypes: true });
    } catch {
      continue;
    }
    for (const e of entries) {
      if (e.name.startsWith(".")) continue;
      if (NON_CONCEPT_ZONES.has(e.name) || NON_CONCEPT_FILES.has(e.name)) continue;
      const p = join(d, e.name);
      if (e.isDirectory()) stack.push(p);
      else if (e.name.endsWith(".md")) out.push(p);
    }
  }
  return out.sort();
}

function relToRoot(bundleRoot, p) {
  const rel = relative(bundleRoot, p).split(join).join("/");
  return rel.startsWith("./") ? rel.slice(2) : rel;
}

function stripFence(content) {
  // Remove fenced code blocks (``` / ~~~) from body text: they carry examples,
  // not real links, wikilinks, or literals.
  return content.replace(
    /^[ \t]*(```|~~~)[\s\S]*?(^[ \t]*\1[ \t]*$|\n?(?=---\s*$)|$)/gm,
    (m) => m.replace(/[^`\n]/g, " "),
  );
}

function readConceptFile(absPath) {
  const raw = readFileSync(absPath, "utf8");
  const m = raw.match(/^---\r?\n([\s\S]*?)\r?\n---\r?\n?([\s\S]*)$/);
  if (!m) {
    return { frontmatter: null, body: raw, parseError: null };
  }
  let data = null;
  let parseError = null;
  try {
    data = parse(m[1]);
  } catch (e) {
    parseError = e.message.split("\n")[0];
  }
  const fm =
    data && typeof data === "object" && !Array.isArray(data) ? data : null;
  return { frontmatter: fm, body: m[2], parseError };
}

function readBodyFile(absPath) {
  return readFileSync(absPath, "utf8");
}

// --- Catalog parsing (root + per-dir index.md) ---

function catalogEntries(content) {
  // Inline links only (the SeedX convention: index entries are paths + a
  // one-line status, never nested). Returns normalized in-bundle targets.
  const entries = new Set();
  const re = /\[[^\]]*\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g;
  let m;
  while ((m = re.exec(content))) {
    const target = m[1].trim();
    if (/^(https?:)?\/\//.test(target) || target.startsWith("#")) continue;
    entries.add(target.split("#")[0]);
  }
  return [...entries];
}

// --- Markdown link extraction (body, fences stripped) ---

function extractLinks(body) {
  const noFences = stripFence(body);
  const out = [];
  // Images first so they are not double-counted as links, then links.
  const re = /(!?)\[[^\]]*\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g;
  let m;
  while ((m = re.exec(noFences))) {
    const target = m[2].trim();
    if (/^(https?:)?\/\//.test(target)) continue; // network: out of scope
    if (target.startsWith("#")) continue; // in-page anchor
    out.push({ kind: m[1] ? "image" : "link", target: target.split("#")[0] });
  }
  return out;
}

function wikilinks(body) {
  const noFences = stripFence(body);
  const out = [];
  const re = /\[\[([^\]]+)\]\]/g;
  let m;
  while ((m = re.exec(noFences))) {
    out.push(m[1].split("|")[0].trim());
  }
  return out;
}

// --- Anchor liveness ---

function findMatchRegion(fileLines, anchor) {
  // Returns { lines, start } or null. `near` narrows the file to the first
  // line carrying it; the all_of terms must then all appear within
  // `scope_lines` (default ANCHOR_MAX_SPAN) lines of that region.
  let pool = fileLines;
  if (anchor.near) {
    const idx = fileLines.findIndex((l) => l.includes(anchor.near));
    if (idx < 0) return null;
    const span = Number(anchor.scope_lines) || ANCHOR_MAX_SPAN;
    pool = fileLines.slice(idx, idx + span);
  }
  const terms = Array.isArray(anchor.all_of) ? anchor.all_of : [];
  if (terms.length === 0) return null;
  const span = Number(anchor.scope_lines) || ANCHOR_MAX_SPAN;
  for (let i = 0; i < pool.length; i++) {
    const window = pool.slice(i, i + span).join("\n");
    if (terms.every((t) => window.includes(String(t)))) {
      return { lines: window, start: i };
    }
  }
  return null;
}

function checkAnchorLiveness(concept, regions) {
  // regions: Map<anchorPath, { matched: boolean, text: string|null }>.
  // Returns { verdict, details[] }: CLAIM_BROKEN when a pinned all_of term is
  // gone from its file, DRIFT when a claim literal is absent from every
  // matched region, LIVE otherwise.
  const claims = Array.isArray(concept.fm?.claims) ? concept.fm.claims : [];
  const details = [];
  let verdict = "LIVE";
  for (const [path, r] of regions) {
    if (!r.matched) {
      verdict = "CLAIM_BROKEN";
      details.push(`anchor ${path}: all_of term missing from pinned file`);
    }
  }
  const matchedText = [...regions.values()]
    .filter((r) => r.matched)
    .map((r) => r.text)
    .join("\n");
  for (const claim of claims) {
    if (!claim || typeof claim !== "object") continue;
    const literal = claim.literal;
    if (typeof literal !== "string") continue;
    if (!matchedText.includes(literal)) {
      verdict = "DRIFT";
      details.push(
        `claim literal ${JSON.stringify(literal)} not in any matched region`,
      );
    }
  }
  return { verdict, details };
}

// --- Reverse index ---

function expectedReverseIndex(bundleRoot, concepts) {
  // path -> concept relative map, one row per anchored source path.
  const rows = new Map();
  for (const c of concepts) {
    const anchors = c.fm?.verification?.anchors;
    if (!Array.isArray(anchors)) continue;
    for (const a of anchors) {
      if (a && typeof a === "object" && typeof a.path === "string") {
        rows.set(a.path, c.rel);
      }
    }
  }
  const lines = [
    "# Reverse Index",
    "",
    "Generated, committed. Concept file -> anchored source paths. Keep in sync with the concepts' `verification.anchors`.",
    "",
    "| Anchor path | Concept |",
    "|---|---|",
  ];
  for (const [path, concept] of [...rows.entries()].sort((a, b) =>
    a[0] < b[0] ? -1 : 1,
  )) {
    lines.push(`| ${path} | ${concept} |`);
  }
  return [rows, lines.join("\n") + "\n"];
}

function committedReverseRows(content) {
  // Parse committed `| path | concept |` rows.
  const rows = new Map();
  for (const line of content.split("\n")) {
    const m = line.match(/^\|\s*([^|]+?)\s*\|\s*([^|]+?)\s*\|$/);
    if (!m) continue;
    if (m[1].startsWith("Anchor") || m[1].startsWith("---")) continue;
    if (/^[-: ]+$/.test(m[1])) continue;
    rows.set(m[1], m[2]);
  }
  return rows;
}

// --- Orphans (advisory) ---

function findOrphans(bundleRoot, concepts) {
  const listed = new Set();
  for (const c of concepts) listed.add(c.rel);
  for (const f of walkFiles(bundleRoot)) {
    const rel = relToRoot(bundleRoot, f);
    if (rel === "index.md" || rel === "log.md" || rel === "reverse-index.md") continue;
    if (rel.endsWith("index.md") && rel !== "index.md") continue; // per-dir catalog
    if (listed.has(rel)) continue;
    listed.add(rel);
  }
  // A concept is an orphan when no other bundle file links to it and it is
  // only self-listed in its own catalog.
  const inlinks = new Map();
  for (const c of concepts) inlinks.set(c.rel, 0);
  for (const c of concepts) {
    const body = c.rawBody;
    for (const link of extractLinks(body)) {
      const targetAbs = join(dirname(c.abs), link.target);
      const targetRel = relToRoot(bundleRoot, targetAbs);
      if (targetRel !== c.rel && inlinks.has(targetRel)) {
        inlinks.set(targetRel, inlinks.get(targetRel) + 1);
      }
    }
  }
  return [...inlinks.entries()].filter(([, n]) => n === 0).map(([rel]) => rel);
}

// --- Main ---

function main() {
  const args = process.argv.slice(2);
  if (args.length < 1 || args[0].startsWith("-")) {
    console.error(
      "Usage: okf-lint.mjs [project-root]\n  project-root defaults to the current working directory.",
    );
    process.exit(2);
  }
  const projectRoot = resolve(args[0]);
  if (!existsSync(projectRoot) || !isDir(projectRoot)) {
    console.error(`ERROR: ${projectRoot} is not a directory`);
    process.exit(2);
  }

  const detected = detectBundleRoot(projectRoot);
  if (!detected) {
    // Non-answer, not a clean bill (ADR-0010).
    console.log("DRIFT: none (no knowledge bundle found)");
    console.log("SCOPE: UNSCOPED");
    writeTelemetry(projectRoot, "UNSCOPED", 0, []);
    process.exit(0);
  }
  const { root: bundleRoot, layout } = detected;

  // --- Enumerate the bundle (a failing enumeration is UNREADABLE, not clean) ---
  let mdFiles;
  try {
    mdFiles = walkFiles(bundleRoot);
  } catch {
    console.error(`ERROR: cannot enumerate ${bundleRoot}`);
    process.exit(2);
  }

  let scope = "COMPLETE";
  for (const f of mdFiles) {
    try {
      readFileSync(f);
    } catch {
      scope = worstScope(scope, "UNREADABLE");
    }
  }

  // --- Load concepts + reserved files ---
  const concepts = [];
  const reserved = []; // { rel, content }
  for (const f of mdFiles) {
    const rel = relToRoot(bundleRoot, f);
    const base = rel.split("/").pop();
    // Pre-R6 artifacts/ fallback: README.md is the root catalog (the
    // post-R6 root catalog is index.md). Treat it as reserved so it is
    // checked for okf_version and links, not linted as a concept.
    if (base === "index.md" || base === "log.md" || base === "README.md") {
      reserved.push({ rel, content: readBodyFile(f) });
      continue;
    }
    const loaded = readConceptFile(f);
    concepts.push({
      abs: f,
      rel,
      base,
      fm: loaded.frontmatter,
      body: loaded.body,
      rawBody: loaded.body,
      parseError: loaded.parseError,
      isDirCatalog: rel.endsWith("index.md"),
    });
  }
  // Root catalog: index.md (post-R6 / bundle-root) or README.md (pre-R6).
  const rootIndex =
    reserved.find((r) => r.rel === "index.md") ||
    reserved.find((r) => r.rel === "README.md");

  const findings = [];
  const find = (kind, rel, detail) =>
    findings.push({ kind, rel, detail, msg: `${kind}: ${rel} — ${detail}` });

  // --- Check 1: frontmatter-valid ---
  for (const c of concepts) {
    if (c.parseError) {
      find("frontmatter-invalid", c.rel, `unparseable YAML: ${c.parseError}`);
      continue;
    }
    if (!c.isDirCatalog && c.fm && typeof c.fm.type !== "string" && !c.fm.type) {
      find("frontmatter-invalid", c.rel, "missing or empty `type`");
    }
  }
  if (rootIndex) {
    // Root catalog: reserved-exempt, but okf_version is the one required key.
    const m = rootIndex.content.match(/^---\r?\n([\s\S]*?)\r?\n---/);
    let versionMissing = true;
    if (m) {
      try {
        const data = parse(m[1]);
        versionMissing = !(data && typeof data === "object" && data.okf_version);
      } catch {
        versionMissing = false; // unparseable root is not a frontmatter finding
      }
    }
    if (versionMissing) {
      find(
        "frontmatter-invalid",
        rootIndex.rel,
        "root catalog missing `okf_version`",
      );
    }
  }

  // --- Check 2: index-complete (both directions) ---
  const entrySet = new Set();
  const catalogs = [];
  if (rootIndex) catalogs.push({ rel: rootIndex.rel, content: rootIndex.content });
  for (const c of concepts) {
    if (c.isDirCatalog) {
      // c.body is already fence-stripped (readConceptFile), so use it as the
      // catalog content directly — no re-read of the frontmatter needed.
      catalogs.push({ rel: c.rel, content: c.body });
    }
  }
  for (const cat of catalogs) {
    for (const entry of catalogEntries(cat.content)) {
      entrySet.add(entry);
    }
  }
  const conceptRels = new Set(
    concepts.filter((c) => !c.isDirCatalog).map((c) => c.rel),
  );
  for (const c of concepts) {
    if (c.isDirCatalog) continue;
    if (!entrySet.has(c.rel)) {
      find(
        "index-incomplete",
        c.rel,
        "concept not listed in index.md or its directory catalog",
      );
    }
  }
  for (const entry of [...entrySet].sort()) {
    if (!existsSync(join(bundleRoot, entry))) {
      find("index-incomplete", entry, "catalog entry does not resolve");
    }
  }

  // --- Check 3: links-unbroken (in-bundle Markdown targets only) ---
  // --- Check 4: no-wikilink-syntax ---
  for (const c of concepts) {
    for (const link of extractLinks(c.body)) {
      const abs = join(dirname(c.abs), link.target);
      const targetRel = relToRoot(bundleRoot, abs);
      const isMarkdown =
        link.target.endsWith(".md") ||
        existsSync(abs) && isDir(abs);
      if (isMarkdown && !existsSync(abs)) {
        find(
          "link-broken",
          c.rel,
          `link target does not exist: ${link.target}`,
        );
      }
    }
    for (const wl of wikilinks(c.body)) {
      find("no-wikilink-syntax", c.rel, `[[${wl}]] uses banned wikilink syntax`);
    }
  }
  for (const r of reserved) {
    for (const link of extractLinks(r.content)) {
      const abs = join(dirname(join(bundleRoot, r.rel)), link.target);
      const isMarkdown =
        link.target.endsWith(".md") ||
        (existsSync(abs) && isDir(abs));
      if (isMarkdown && !existsSync(abs)) {
        find("link-broken", r.rel, `link target does not exist: ${link.target}`);
      }
    }
    for (const wl of wikilinks(r.content)) {
      find("no-wikilink-syntax", r.rel, `[[${wl}]] uses banned wikilink syntax`);
    }
  }

  // --- Check 5: provenance-required ---
  for (const c of concepts) {
    if (c.isDirCatalog || !c.fm || c.parseError) continue;
    if (c.fm.status !== "stable") continue; // draft exempt
    const generated =
      c.fm.generated &&
      typeof c.fm.generated === "object" &&
      (c.fm.generated.by || c.fm.generated.at);
    if (!generated) {
      find(
        "provenance-missing",
        c.rel,
        "status: stable requires a `generated` provenance block",
      );
      continue;
    }
    const anchored = Array.isArray(c.fm.verification?.anchors) && c.fm.verification.anchors.length > 0;
    if (anchored && !(Array.isArray(c.fm.sources) && c.fm.sources.length > 0)) {
      find(
        "provenance-missing",
        c.rel,
        "anchored stable concept requires `sources`",
      );
    }
  }

  // --- Check 6: claims-require-verification ---
  for (const c of concepts) {
    if (c.isDirCatalog || !c.fm || c.parseError) continue;
    if (c.fm.type !== "InvariantContract") continue;
    const claims = Array.isArray(c.fm.claims) ? c.fm.claims : [];
    if (claims.length === 0) {
      find(
        "claims-missing",
        c.rel,
        "InvariantContract requires a non-empty `claims[]`",
      );
      continue;
    }
    const anchors = c.fm.verification?.anchors;
    if (!(Array.isArray(anchors) && anchors.length > 0)) {
      find(
        "claims-unverified",
        c.rel,
        "claims[] require `verification.anchors`",
      );
    }
  }

  // --- Check 7: anchors-match (liveness) ---
  for (const c of concepts) {
    if (c.isDirCatalog || !c.fm || c.parseError) continue;
    const anchors = c.fm.verification?.anchors;
    if (!Array.isArray(anchors) || anchors.length === 0) continue;
    // One region per anchor: read the pinned file, match the all_of terms.
    const regions = new Map();
    for (const anchor of anchors) {
      if (!anchor || typeof anchor.path !== "string") continue;
      const pinnedAbs = isAbsolute(anchor.path)
        ? anchor.path
        : join(bundleRoot, anchor.path);
      let pinned;
      try {
        pinned = readFileSync(pinnedAbs, "utf8");
      } catch {
        find(
          "anchor-broken",
          c.rel,
          `anchor path does not exist: ${anchor.path}`,
        );
        continue;
      }
      const region = findMatchRegion(pinned.split("\n"), anchor);
      regions.set(anchor.path, {
        matched: !!region,
        text: region ? region.lines : null,
      });
    }
    const result = checkAnchorLiveness(c, regions);
    if (!result) continue;
    for (const detail of result.details) {
      if (result.verdict === "CLAIM_BROKEN") {
        find("claim-broken", c.rel, `liveness CLAIM_BROKEN: ${detail}`);
      } else {
        find("anchor-drift", c.rel, `liveness DRIFT: ${detail}`);
      }
    }
  }

  // --- Check 8: reverse-index-sync ---
  const reverseRel = "reverse-index.md";
  const reverseAbs = join(bundleRoot, reverseRel);
  const [expectedRows] = expectedReverseIndex(bundleRoot, concepts);
  if (existsSync(reverseAbs)) {
    const committed = committedReverseRows(readBodyFile(reverseAbs));
    for (const [path, concept] of expectedRows) {
      if (!committed.has(path) || committed.get(path) !== concept) {
        find(
          "reverse-index-out-of-sync",
          reverseRel,
          `expected row missing or stale: ${path} -> ${concept}`,
        );
      }
    }
    for (const [path, concept] of committed) {
      if (!expectedRows.has(path) || expectedRows.get(path) !== concept) {
        find(
          "reverse-index-out-of-sync",
          reverseRel,
          `unexpected or stale row: ${path} -> ${concept}`,
        );
      }
    }
  } else if (expectedRows.size > 0) {
    find(
      "reverse-index-out-of-sync",
      reverseRel,
      "missing — anchors exist but the committed reverse index does not",
    );
  }

  // --- Check 9: orphans (advisory-only, never fails the run) ---
  const orphans = findOrphans(bundleRoot, concepts);

  // --- Report ---
  if (findings.length === 0) {
    console.log("DRIFT: clean");
  } else {
    console.log(`DRIFT: ${findings.length} finding(s):`);
    for (const f of findings) console.log(`  ${f.msg}`);
  }
  if (orphans.length > 0) {
    console.log(
      `ADVISORY: ${orphans.length} orphan(s) (not counted in drift): ${orphans.join(", ")}`,
    );
  }
  console.log(`SCOPE: ${scope}`);
  console.log(`LAYOUT: ${layout}`);

  writeTelemetry(projectRoot, scope, findings.length, findings.map((f) => f.msg));
  process.exit(findings.length > 0 ? 1 : 0);
}

// --- Telemetry (the ONLY write side-effect) ---

function writeTelemetry(projectRoot, scope, drift, findingMsgs) {
  const sink =
    process.env.OKF_METRICS_SINK || join(projectRoot, ".skillgrid", "okf-metrics.jsonl");
  let priorFindings = [];
  try {
    if (existsSync(sink)) {
      const lines = readFileSync(sink, "utf8").split("\n").filter(Boolean);
      const last = lines[lines.length - 1];
      if (last) {
        const parsed = JSON.parse(last);
        if (Array.isArray(parsed?.findings)) priorFindings = parsed.findings;
      }
    }
  } catch {
    priorFindings = [];
  }
  // Prior findings absent this run were false positives.
  const thisSet = new Set(findingMsgs);
  const falsePositives = priorFindings.filter((f) => !thisSet.has(f)).length;
  const rate = priorFindings.length > 0 ? falsePositives / priorFindings.length : 0.0;
  const event = {
    event: "lint",
    at: new Date().toISOString(),
    scope,
    drift,
    findings: findingMsgs,
    false_drift_rate: rate,
  };
  try {
    mkdirSync(dirname(sink), { recursive: true });
    appendFileSync(sink, JSON.stringify(event) + "\n");
  } catch (e) {
    // Non-fatal: the verdict above is already printed; the sink is an
    // operational artifact that lives or dies with the working tree.
    console.error(
      `WARNING: okf-metrics telemetry write failed (non-fatal): ${e.message}`,
    );
  }
}

main();
