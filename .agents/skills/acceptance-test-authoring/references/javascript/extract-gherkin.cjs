'use strict';

// Extracts classic Gherkin from Markdown acceptance specs (acceptance.feature)
// into .feature files, synthesizing Feature:/Rule:/Scenario: from the Markdown
// headings and copying fenced step lines verbatim.
//
// ../EXTRACTION.md IS THE DEFINITION — the line-by-line mapping, the fence
// mechanics, the line-fidelity invariant and every hard error live there.
// Change the doc first, then this implementation, then re-verify.
//
// Skillgrid C1 model: one complete acceptance.feature per change under
// .skillgrid/specs/<id>/. There is no source-of-truth corpus and no delta
// composition — every change's file is the whole executable spec for that
// change. Discovery anchors to the literal basename acceptance.feature.

// Deliberately dependency-free (node:fs/node:path only): the CLI form must
// run from the skill's references/ directory, where no node_modules exists.

const fs = require('node:fs');
const path = require('node:path');

const GHERKIN_OPEN_RE = /^(`{3,})gherkin\s*$/;
const ANY_OPEN_RE = /^(`{3,})\S*\s*$/;
const INDENTED_GHERKIN_RE = /^\s+`{3,}gherkin\s*$/;

const HEADING_RE = /^#{1,6}\s+/;
const H1_RE = /^#\s+(.+?)\s*$/;
const REQUIREMENT_RE = /^###\s+Requirement:\s*(.+?)\s*$/i;
const SCENARIO_RE = /^####\s+(Scenario(?:\s+Outline)?):\s*(.+?)\s*$/i;

// Structure keywords are illegal inside a fence — they come from the
// headings. `Examples:` (the Scenario Outline table) and `Background:` are
// deliberately absent: both legitimately live in a fence.
const STRUCTURE_IN_FENCE_RE = /^\s*(Feature|Rule|Scenario\s+Outline|Scenario|Example):/;

function extractFile(mdPath) {
  const lines = fs.readFileSync(mdPath, 'utf8').split(/\r?\n/);
  const out = [];
  let state = 'prose'; // 'prose' | 'gherkin' | 'other-fence'
  let fenceTicks = 0;
  let openLine = 0;
  let h1Count = 0;
  let pendingScenario = null; // scenario heading still awaiting its fence

  lines.forEach((line, i) => {
    if (state === 'prose') {
      let m;
      if ((m = GHERKIN_OPEN_RE.exec(line))) {
        state = 'gherkin';
        fenceTicks = m[1].length;
        openLine = i + 1;
        pendingScenario = null;
        out.push('');
        return;
      }
      if (INDENTED_GHERKIN_RE.test(line)) {
        throw new Error(
          `${mdPath}:${i + 1}: indented \`\`\`gherkin fence — gherkin fences must start at column 0`
        );
      }
      if ((m = ANY_OPEN_RE.exec(line))) {
        state = 'other-fence';
        fenceTicks = m[1].length;
        openLine = i + 1;
        out.push('');
        return;
      }
      if (!HEADING_RE.test(line)) {
        out.push('');
        return;
      }
      // A heading ends any scenario still waiting for its steps — but only
      // once the file is known to have its H1 title. Gating on h1Count means a
      // missing-title error (checked at EOF) is not shadowed by a stray
      // pending-scenario error, and a structure-keyword-in-fence error (checked
      // inside the fence) is still surfaced for a well-formed spec.
      if (pendingScenario && h1Count > 0) {
        throw new Error(
          `${mdPath}:${pendingScenario.line}: "#### ${pendingScenario.keyword}: ${pendingScenario.name}" ` +
            `has no \`\`\`gherkin fence before the next heading`
        );
      }
      if ((m = H1_RE.exec(line))) {
        h1Count += 1;
        if (h1Count > 1) {
          throw new Error(
            `${mdPath}:${i + 1}: more than one H1 — an acceptance.feature has exactly one "# <capability>" title`
          );
        }
        out.push(`Feature: ${m[1]}`);
        return;
      }
      if ((m = REQUIREMENT_RE.exec(line))) {
        out.push(`  Rule: ${m[1]}`);
        return;
      }
      if ((m = SCENARIO_RE.exec(line))) {
        const keyword = /outline/i.test(m[1]) ? 'Scenario Outline' : 'Scenario';
        const name = m[2];
        pendingScenario = { line: i + 1, keyword, name };
        out.push(`    ${keyword}: ${name}`);
        return;
      }
      out.push('');
      return;
    }
    const close = new RegExp('^`{' + fenceTicks + ',}\\s*$');
    if (close.test(line)) {
      state = 'prose';
      out.push('');
      return;
    }
    if (state !== 'gherkin') {
      out.push('');
      return;
    }
    const kw = STRUCTURE_IN_FENCE_RE.exec(line);
    if (kw) {
      throw new Error(
        `${mdPath}:${i + 1}: "${kw[1]}:" inside a \`\`\`gherkin fence — structure comes from Markdown ` +
          `headings ("# title", "### Requirement:", "#### Scenario:"); fences hold only steps`
      );
    }
    out.push(line);
  });

  if (state !== 'prose') {
    throw new Error(`${mdPath}:${openLine}: unclosed fence`);
  }
  // Check the H1 title before the pending-scenario check so a missing-title
  // error is not shadowed by a scenario that happens to lack its fence.
  if (h1Count === 0) {
    throw new Error(`${mdPath}: no H1 title — an acceptance.feature must start with "# <capability>"`);
  }
  if (pendingScenario) {
    throw new Error(
      `${mdPath}:${pendingScenario.line}: "#### ${pendingScenario.keyword}: ${pendingScenario.name}" ` +
        `has no \`\`\`gherkin fence before the end of the file`
    );
  }
  if (out.length !== lines.length) {
    throw new Error(`${mdPath}: line-count invariant violated (extractor bug)`);
  }
  return out.join('\n');
}

// Recursively collects files named <basename> under <dir>, returned as
// posix paths relative to <root>, sorted for deterministic output.
function walk(root, dir, basename, found) {
  if (!fs.existsSync(dir)) return found;
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const abs = path.join(dir, entry.name);
    if (entry.isDirectory()) walk(root, abs, basename, found);
    else if (entry.name === basename || (basename.startsWith('*.') && entry.name.endsWith(basename.slice(1)))) {
      found.push(path.relative(root, abs).split(path.sep).join('/'));
    }
  }
  return found;
}

// C1 discovery: every acceptance.feature under <specsDir>, one per change.
// No delta composition, no archive.
function collectSpecSources(specsDir, basename) {
  return walk(specsDir, specsDir, basename, []).sort();
}

// Extracts every acceptance.feature under <specsDir> into <outDir>, mirroring
// the specsDir-relative path with acceptance.feature -> acceptance.feature
// (the .md is the source; the .feature is the extracted Gherkin). The output
// dir is wiped first — a stale extraction would keep deleted or renamed
// capabilities executing.
function extractAll(specsDir, outDir) {
  specsDir = specsDir ? path.resolve(specsDir) : path.resolve(process.cwd(), '.skillgrid/specs');
  outDir = outDir ? path.resolve(outDir) : path.resolve(process.cwd(), 'acceptance-tests/.extracted');

  fs.rmSync(outDir, { recursive: true, force: true });

  const sources = collectSpecSources(specsDir, 'acceptance.feature');

  const written = [];
  for (const rel of sources) {
    const dest = path.join(outDir, rel.replace(/acceptance\.feature$/, 'acceptance.feature'));
    fs.mkdirSync(path.dirname(dest), { recursive: true });
    fs.writeFileSync(dest, extractFile(path.join(specsDir, rel)));
    written.push(dest);
  }
  return { outDir, written };
}

module.exports = { extractAll, extractFile };

// CLI: node extract-gherkin.cjs [specsDir] [outDir]
if (require.main === module) {
  try {
    const { outDir, written } = extractAll(process.argv[2], process.argv[3]);
    console.error(`[extract-gherkin] ${written.length} acceptance.feature file(s) extracted to ${outDir}`);
  } catch (err) {
    console.error(`[extract-gherkin] ${err.message}`);
    process.exit(1);
  }
}
