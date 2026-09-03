#!/usr/bin/env node
/**
 * extract_skills.js — Skill registry indexer for sdd-init
 *
 * Scans configured skill directories (project-level and user-level), extracts
 * the frontmatter `description` (trigger text) from each SKILL.md, and emits
 * a markdown table suitable for `docs/agents/skill-registry.md`.
 *
 * Called by the sdd-init skill as:
 *   node .agents/skills/sdd-init/scripts/extract_skills.js [--root <project-root>]
 *
 * Scan rules (from references/init-details.md):
 * - Project skills: <root>/.agents/skills/, <root>/.claude/skills/, <root>/.github/skills/, <root>/skills/
 * - User skills:    ~/.agents/skills/, ~/.config/kilo/skills/, ~/.claude/skills/,
 *                   ~/.gemini/skills/, ~/.cursor/skills/, ~/.config/agents/skills/
 * - Skip: `_shared`, `sdd-*` prefixes, `skill-registry` (workflow machinery, not project skills)
 * - Deduplicate by name, preferring project-level over user-level
 * - Convention files (AGENTS.md, CLAUDE.md, etc.) indexed separately by the skill
 */

const fs = require('fs');
const path = require('path');

const SKIP_NAMES = new Set(['_shared', 'skill-registry']);
const SKIP_PREFIXES = ['sdd-'];

function shouldSkipDir(name) {
  if (SKIP_NAMES.has(name)) return true;
  return SKIP_PREFIXES.some(p => name.startsWith(p));
}

/**
 * Extract the `description:` field from YAML frontmatter at the top of a file.
 * Handles plain, quoted, and folded (`>`) scalars, plus literal (`|`) blocks.
 */
function extractDescription(filepath) {
  const content = fs.readFileSync(filepath, 'utf-8');
  const lines = content.split('\n');
  let inFrontmatter = false;

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    if (line === '---' && i === 0) {
      inFrontmatter = true;
      continue;
    }
    if (line === '---' && inFrontmatter) {
      break;
    }
    if (!inFrontmatter) break;

    const match = line.match(/^(\w+):\s*(.*)$/);
    if (match && match[1] === 'description') {
      const val = match[2].trim();

      if (val.startsWith('"') && val.endsWith('"')) {
        return val.slice(1, -1).replace(/\s+/g, ' ').trim();
      }

      if (val.startsWith('>')) {
        // Folded scalar — collect indented continuation lines.
        const collected = [];
        let j = val.startsWith('>') && val.length > 1 ? i : i + 1;
        if (val.startsWith('>') && val.length > 1) {
          // `>` with inline content on same line is not valid YAML, but handle gracefully
          collected.push(val.slice(1).trim());
        }
        j = i + 1;
        while (j < lines.length && (lines[j].startsWith('  ') || lines[j].trim() === '')) {
          collected.push(lines[j].trim());
          j++;
        }
        return collected.join(' ').replace(/\s+/g, ' ').trim();
      }

      if (val.startsWith('|')) {
        // Literal block scalar — collect indented lines.
        const collected = [];
        const j = i + 1;
        while (j < lines.length && (lines[j].startsWith('  ') || lines[j].trim() === '')) {
          collected.push(lines[j]);
          j++;
        }
        return collected.map(l => l.trim()).join('\n').replace(/\s+/g, ' ').trim();
      }

      if (val === '') {
        // description: with content on continuation lines
        const collected = [];
        let j = i + 1;
        while (j < lines.length && (lines[j].startsWith('  ') || lines[j].trim() === '')) {
          const subMatch = lines[j].match(/^\s+-?\s+(.*)$/);
          if (subMatch && subMatch[1]) {
            collected.push(subMatch[1].trim());
          }
          j++;
        }
        if (collected.length > 0) {
          return collected.join(' ').replace(/\s+/g, ' ').trim();
        }
        break;
      }

      return val.replace(/\s+/g, ' ').trim();
    }
  }
  return '';
}

/**
 * Scan a directory for skills, returning entries with name, trigger, path, scope.
 */
function scanSkillsDir(skillsDir, scope, rootDir) {
  const results = [];
  if (!fs.existsSync(skillsDir)) return results;

  for (const name of fs.readdirSync(skillsDir)) {
    const fullPath = path.join(skillsDir, name);
    try {
      if (!fs.statSync(fullPath).isDirectory()) continue;
    } catch {
      continue;
    }
    if (shouldSkipDir(name)) continue;

    const skillMd = path.join(fullPath, 'SKILL.md');
    if (!fs.existsSync(skillMd)) continue;

    let desc;
    try {
      desc = extractDescription(skillMd);
    } catch {
      desc = '';
    }

    let entryPath;
    if (scope === 'project' && rootDir) {
      entryPath = path.relative(rootDir, skillMd);
    } else if (scope === 'project') {
      entryPath = skillMd;
    } else {
      entryPath = fullPath;
    }

    results.push({ name, trigger: desc, path: entryPath, scope });
  }
  return results;
}

// --- Determine root ---
let rootDir = process.cwd();
const args = process.argv.slice(2);
for (let i = 0; i < args.length; i++) {
  if (args[i] === '--root' && i + 1 < args.length) {
    rootDir = path.resolve(args[i + 1]);
    i++;
  }
}

// If no --root given, try git root
if (!args.includes('--root')) {
  try {
    const { execSync } = require('child_process');
    const gitRoot = execSync('git rev-parse --show-toplevel', { cwd: process.cwd(), stdio: 'pipe' }).toString().trim();
    if (gitRoot) rootDir = gitRoot;
  } catch {
    // Not in a git repo, stay with cwd
  }
}

const homeDir = require('os').homedir();

const projectDirs = [
  path.join(rootDir, '.agents', 'skills'),
  path.join(rootDir, '.claude', 'skills'),
  path.join(rootDir, '.github', 'skills'),
  path.join(rootDir, 'skills'),
];

const userDirs = [
  path.join(homeDir, '.agents', 'skills'),
  path.join(homeDir, '.config', 'kilo', 'skills'),
  path.join(homeDir, '.claude', 'skills'),
  path.join(homeDir, '.gemini', 'skills'),
  path.join(homeDir, '.cursor', 'skills'),
  path.join(homeDir, '.config', 'agents', 'skills'),
];

// --- Scan project-level first (preferred) ---
const entries = [];
for (const dir of projectDirs) {
  for (const e of scanSkillsDir(dir, 'project', rootDir)) {
    if (!entries.some(existing => existing.name === e.name)) {
      entries.push(e);
    }
  }
}

// --- Scan user-level, deduplicate by name ---
for (const dir of userDirs) {
  for (const e of scanSkillsDir(dir, 'user', rootDir)) {
    if (!entries.some(existing => existing.name === e.name)) {
      entries.push(e);
    }
  }
}

entries.sort((a, b) => a.name.localeCompare(b.name));

// --- Output markdown table ---
console.log('| name | trigger | path | scope |');
console.log('|---|---|---|---|');
for (const e of entries) {
  console.log(`| ${e.name} | ${e.trigger} | ${e.path} | ${e.scope} |`);
}
console.log(`\nTotal: ${entries.length} skills`);
