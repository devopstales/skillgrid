/**
 * Skillgrid router plugin for Kilo
 *
 * Injects the skillgrid router bootstrap via:
 *   - experimental.chat.messages.transform (primary; TUI / interactive)
 *   - experimental.chat.system.transform (fallback for non-interactive
 *     `kilo run` paths where messages.transform may not fire — see #2197)
 * Auto-registers skills directory via config hook (no symlinks needed).
 */

import path from 'path';
import fs from 'fs';
import os from 'os';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

// Static fallback bootstrap (R4): used when neither the installed copy nor the
// repo clone is resolvable. Mirrors the cursor hook's fallback string.
const STATIC_BOOTSTRAP = 'skillgrid: router bootstrap fallback - first action: read `.agents/skills/using-skillgrid/SKILL.md`; announce `Using skillgrid:using-skillgrid to {purpose}`.';

type RouterPart = { type?: string; text?: string };
type RouterMessage = { info?: { role?: string; mode?: string }; parts?: RouterPart[] };
type RouterMessages = { messages?: RouterMessage[] };
type RouterSystem = { system?: string[] };
type SkillgridPluginConfig = { skills?: { paths?: string[] } };

// Simple frontmatter extraction (avoid dependency on skills-core for bootstrap)
const extractAndStripFrontmatter = (content: string) => {
  const match = content.match(/^---\n([\s\S]*?)\n---\n([\s\S]*)$/);
  if (!match) return { frontmatter: {}, content };

  const frontmatterStr = match[1];
  const body = match[2];
  const frontmatter: Record<string, string> = {};

  for (const line of frontmatterStr.split('\n')) {
    const colonIdx = line.indexOf(':');
    if (colonIdx > 0) {
      const key = line.slice(0, colonIdx).trim();
      const value = line.slice(colonIdx + 1).trim().replace(/^["']|["']$/g, '');
      frontmatter[key] = value;
    }
  }

  return { frontmatter, content: body };
};

// Normalize a path: trim whitespace, expand ~, resolve to absolute
const normalizePath = (p: string | null | undefined, homeDir: string): string | null => {
  if (!p || typeof p !== 'string') return null;
  let normalized = p.trim();
  if (!normalized) return null;
  if (normalized.startsWith('~/')) {
    normalized = path.join(homeDir, normalized.slice(2));
  } else if (normalized === '~') {
    normalized = homeDir;
  }
  return path.resolve(normalized);
};

// Module-level cache for bootstrap content.
// The bootstrap asset does not change during a session, so reading + resolving
// it once eliminates redundant fs.existsSync + fs.readFileSync + regex work on
// every agent step. See #1202 for the full analysis.
let _bootstrapCache: string | null | undefined = undefined; // undefined = not yet loaded, null = missing, string = content

export const SkillgridRouterPlugin = async ({ client, directory }: { client?: unknown; directory?: string }) => {
  const homeDir = os.homedir();
  const repoRoot = path.resolve(__dirname, '../..');
  const skillsDir = path.join(repoRoot, '.agents', 'skills');
  const repoBootstrapPath = path.join(repoRoot, '.agents', 'skills', '_shared', 'agent-config', 'router-bootstrap.md');
  const installedBootstrapPath = path.join(homeDir, '.skillgrid', 'router-bootstrap.md');
  const envConfigDir = normalizePath(process.env.KILO_CONFIG_DIR || process.env.KILO_CONFIG, homeDir);
  const configDir = envConfigDir || path.join(homeDir, '.config/kilo');
  const cwd = directory || process.cwd();

  // Project-conditional (R3): a skillgrid project has a `.skillgrid/config.yaml`
  // at or above the working directory. Bounded walk-up stops at the filesystem root.
  const isSkillgridProject = (dir: string): boolean => {
    let current = path.resolve(dir || '.');
    while (true) {
      if (fs.existsSync(path.join(current, '.skillgrid', 'config.yaml'))) return true;
      const parent = path.dirname(current);
      if (parent === current) return false;
      current = parent;
    }
  };

  // Subagent/child sessions are skipped (R3, mirrors `<SUBAGENT-STOP>` and the
  // superpowers `parentID` skip): the router bootstrap is main-agent context.
  const isSubagentMessage = (msg: RouterMessage): boolean => {
    const parts = msg.parts || [];
    const subagentPart = parts.some((p) => p.type === 'subtask');
    const mode = msg.info?.mode;
    return Boolean(subagentPart) || mode === 'subagent';
  };

  const resolveBootstrap = (): string => {
    if (_bootstrapCache !== undefined) return _bootstrapCache as string;

    let bootstrap: string | null = null;
    const candidates = [installedBootstrapPath, repoBootstrapPath];
    for (const candidate of candidates) {
      try {
        if (fs.existsSync(candidate)) {
          bootstrap = fs.readFileSync(candidate, 'utf8');
          break;
        }
      } catch {
        // fail-open: try the next resolution leg
      }
    }
    const content = bootstrap ?? STATIC_BOOTSTRAP;
    _bootstrapCache = content;
    return content;
  };

  const injectIntoFirstUserMessage = (output: RouterMessages): void => {
    const bootstrap = resolveBootstrap();
    const messages = output?.messages;
    if (!bootstrap || !messages || messages.length === 0) return;
    const firstUser = messages.find((m) => m.info?.role === 'user');
    if (!firstUser || !firstUser.parts || firstUser.parts.length === 0) return;
    if (isSubagentMessage(firstUser)) return;
    if (!isSkillgridProject(cwd)) return;
    const alreadyInjected = firstUser.parts.some((p) => p.type === 'text' && p.text?.includes(bootstrap));
    if (alreadyInjected) return;
    const ref = firstUser.parts[0];
    firstUser.parts.unshift({ ...ref, type: 'text', text: bootstrap });
  };

  return {
    // Inject skills path into live config so Kilo discovers the skillgrid
    // skills directory without requiring manual skills.paths edits when the
    // config hook is honored. OpenCode relies on Config.get() returning a
    // cached singleton; Kilo may treat config as read-only — keep explicit
    // skills.paths in INSTALL as fallback.
    config: async (config: SkillgridPluginConfig): Promise<void> => {
      config.skills = config.skills || {};
      config.skills.paths = config.skills.paths || [];
      const skills = config.skills;
      if (!skills.paths.includes(skillsDir)) {
        skills.paths.push(skillsDir);
      }
    },

    // Fallback bootstrap for non-interactive run (#2197): that path may
    // register skills via `config` but never fire messages.transform.
    // Mutate output.system in place (reassignment is a silent no-op).
    // Each channel guards on the bootstrap content independently (approach A).
    'experimental.chat.system.transform': async (_input: unknown, output: RouterSystem): Promise<void> => {
      const bootstrap = resolveBootstrap();
      const system = output?.system;
      if (!bootstrap || !system) return;
      if (system.some((s) => typeof s === 'string' && s.includes(bootstrap))) return;
      if (system.length > 0) {
        system.splice(1, 0, bootstrap);
      } else {
        system.push(bootstrap);
      }
    },

    // Inject bootstrap into the first user message of each session.
    // Using a user message instead of a system message avoids:
    //   1. Token bloat from system messages repeated every turn (#750)
    //   2. Multiple system messages breaking Qwen and other models (#894)
    //
    // The hook fires on every agent step (not just every turn) because
    // Kilo reloads messages from DB each step. Fresh message arrays may
    // need injection again, so resolveBootstrap() must not do repeated
    // disk work.
    'experimental.chat.messages.transform': async (_input: unknown, output: RouterMessages): Promise<void> => {
      injectIntoFirstUserMessage(output);
    },
  };
};

// Test-only reset of the module-level bootstrap cache (undefined = not loaded).
export const __resetBootstrapCacheForTests = (): void => {
  _bootstrapCache = undefined;
};

export default { id: 'skillgrid-router', server: SkillgridRouterPlugin };
