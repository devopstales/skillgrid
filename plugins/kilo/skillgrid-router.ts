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

// Module-level cache for bootstrap content.
// The bootstrap asset does not change during a session, so reading + resolving
// it once eliminates redundant fs.existsSync + fs.readFileSync + regex work on
// every agent step. See #1202 for the full analysis.
let _bootstrapCache: string | null | undefined = undefined; // undefined = not yet loaded, null = missing, string = content
let _bootstrapCacheKey: string | null = null; // session cwd the cache was resolved for

export const SkillgridRouterPlugin = async ({ client, directory }: { client?: unknown; directory?: string }) => {
  const homeDir = os.homedir();
  const cwd = directory || process.cwd();

  // Project-conditional (R3): a skillgrid project has a `.skillgrid/config.yaml`
  // at or above the working directory. Bounded walk-up stops at the filesystem root.
  // R3: an unreadable or non-regular-file config is "no project" (fail-open).
  const isReadableProjectConfig = (p: string): boolean => {
    try {
      if (!fs.statSync(p).isFile()) return false;
      fs.accessSync(p, fs.constants.R_OK);
      return true;
    } catch {
      return false;
    }
  };

  const findProjectRoot = (dir: string): string | null => {
    let current = path.resolve(dir || '.');
    while (true) {
      if (isReadableProjectConfig(path.join(current, '.skillgrid', 'config.yaml'))) return current;
      const parent = path.dirname(current);
      if (parent === current) return null;
      current = parent;
    }
  };

  const projectRoot = findProjectRoot(cwd);
  const repoBootstrapPath = projectRoot
    ? path.join(projectRoot, '.agents', 'skills', '_shared', 'agent-config', 'router-bootstrap.md')
    : null;
  const installedBootstrapPath = path.join(homeDir, '.skillgrid', 'router-bootstrap.md');
  const skillsDir = projectRoot
    ? path.join(projectRoot, '.agents', 'skills')
    : path.join(homeDir, '.skillgrid', '.agents', 'skills');

  // Subagent/child sessions are skipped (R3, mirrors `<SUBAGENT-STOP>` and the
  // superpowers `parentID` skip): the router bootstrap is main-agent context.
  const isSubagentMessage = (msg: RouterMessage): boolean => {
    const parts = msg.parts || [];
    const subagentPart = parts.some((p) => p.type === 'subtask');
    const mode = msg.info?.mode;
    return Boolean(subagentPart) || mode === 'subagent';
  };

  const resolveBootstrap = (): string => {
    if (_bootstrapCacheKey === cwd && _bootstrapCache !== undefined) return _bootstrapCache as string;

    let bootstrap: string | null = null;
    const candidates = [installedBootstrapPath, repoBootstrapPath].filter(Boolean) as string[];
    for (const candidate of candidates) {
      try {
        const raw = fs.readFileSync(candidate, 'utf8');
        if (raw.trim()) {
          bootstrap = raw; // whitespace-only copy -> next source
          break;
        }
      } catch {
        // fail-open: missing/unreadable leg -> next resolution leg
      }
    }
    const content = bootstrap ?? STATIC_BOOTSTRAP;
    _bootstrapCacheKey = cwd;
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
    if (!projectRoot) return;
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
      try {
        // Kilo may hand us a frozen/readonly config: fail-open like the other hooks.
        config.skills = config.skills || {};
        config.skills.paths = config.skills.paths || [];
        if (!config.skills.paths.includes(skillsDir)) {
          config.skills.paths.push(skillsDir);
        }
      } catch {
        // fail-open
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
  _bootstrapCacheKey = null;
};

export default { id: 'skillgrid-router', server: SkillgridRouterPlugin };
