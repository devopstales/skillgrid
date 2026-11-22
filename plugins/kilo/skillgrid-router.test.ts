import fs from 'fs';
import os from 'os';
import path from 'path';
import { fileURLToPath, pathToFileURL } from 'url';
import { SkillgridRouterPlugin, __resetBootstrapCacheForTests } from './skillgrid-router.ts';

type TestPart = { type?: string; text?: string; prompt?: string; description?: string; agent?: string };
type TestMessage = { info?: { role?: string; mode?: string }; parts?: TestPart[] };
type TestMessagesOutput = { messages?: TestMessage[] };

const FIRST_ACTION_KEY = 'first action';
const FALLBACK_MARKER = 'fallback';
const LIVE_MARKER = 'This project is configured with Skillgrid';

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
const assetPath = path.join(repoRoot, '.agents', 'skills', '_shared', 'agent-config', 'router-bootstrap.md');

const makeUserMessage = (text: string): TestMessage => ({
  info: { role: 'user' },
  parts: [{ type: 'text', text }],
});

const makeMessagesOutput = (text: string): TestMessagesOutput => ({
  messages: [makeUserMessage(text)],
});

const bootstrapPartsFor = (messages: TestMessage[] | undefined): TestPart[] =>
  (messages?.[0]?.parts || []).filter(
    (p) => p.type === 'text' && (p.text || '').toLowerCase().includes(FIRST_ACTION_KEY),
  );

const makeProjectDir = (): string => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'skillgrid-kilo-router-'));
  fs.mkdirSync(path.join(dir, '.skillgrid'), { recursive: true });
  fs.writeFileSync(path.join(dir, '.skillgrid', 'config.yaml'), '# test\n');
  return dir;
};

const makeNonProjectDir = (): string => fs.mkdtempSync(path.join(os.tmpdir(), 'skillgrid-kilo-router-'));

const runCases = async (): Promise<string[]> => {
  const failures: string[] = [];
  const check = (ok: boolean, label: string): void => {
    if (!ok) failures.push(label);
  };

  // Case 1 — first user message + project cwd + main agent: first-action line exactly once.
  const projectDir = makeProjectDir();
  const plugin1 = await SkillgridRouterPlugin({ client: {}, directory: projectDir });
  const transform1 = plugin1['experimental.chat.messages.transform'];
  check(typeof transform1 === 'function', 'case1: experimental.chat.messages.transform registered');
  const output1 = makeMessagesOutput('hello');
  await transform1({}, output1);
  const bootstrap1 = bootstrapPartsFor(output1.messages);
  check(bootstrap1.length === 1, `case1: expected exactly one bootstrap part, got ${bootstrap1.length}`);
  check(
    typeof bootstrap1[0]?.text === 'string' && (bootstrap1[0].text || '').toLowerCase().includes(FIRST_ACTION_KEY),
    'case1: bootstrap part carries the first-action line',
  );

  // Case 2 — second transform: no-op (present-guard).
  const transform2 = plugin1['experimental.chat.messages.transform'];
  await transform2({}, output1);
  const bootstrap2 = bootstrapPartsFor(output1.messages);
  check(bootstrap2.length === 1, `case2: second transform is a no-op (bootstrap parts=${bootstrap2.length})`);

  // Case 3 — subagent: unchanged.
  const subagentOutput = {
    messages: [
      {
        info: { role: 'user', mode: 'subagent' },
        parts: [{ type: 'subtask', prompt: 'do the thing', description: 'do the thing', agent: 'general' }],
      },
    ],
  };
  const transform3 = plugin1['experimental.chat.messages.transform'];
  await transform3({}, subagentOutput);
  const bootstrap3 = bootstrapPartsFor(subagentOutput.messages);
  check(bootstrap3.length === 0, `case3: subagent session unchanged (bootstrap parts=${bootstrap3.length})`);

  // Case 4 — non-project cwd: unchanged.
  const nonProjectDir = makeNonProjectDir();
  const plugin4 = await SkillgridRouterPlugin({ client: {}, directory: nonProjectDir });
  const transform4 = plugin4['experimental.chat.messages.transform'];
  const output4 = makeMessagesOutput('hello');
  await transform4({}, output4);
  const bootstrap4 = bootstrapPartsFor(output4.messages);
  check(bootstrap4.length === 0, `case4: non-project cwd unchanged (bootstrap parts=${bootstrap4.length})`);

  // Case 5 — missing asset (BOTH sources): static fallback.
  // Hermetic: the installed copy (~/.skillgrid/router-bootstrap.md) takes
  // precedence over the checkout asset, so a machine with a completed
  // `skillgrid install` would otherwise serve the live installed asset. Point
  // HOME at an empty temp dir so the installed path is absent too.
  __resetBootstrapCacheForTests();
  fs.renameSync(assetPath, `${assetPath}.bak`);
  const realHome = process.env.HOME;
  const tmpHome = fs.mkdtempSync(path.join(os.tmpdir(), 'skillgrid-kilo-home-'));
  process.env.HOME = tmpHome;
  try {
    const plugin5 = await SkillgridRouterPlugin({ client: {}, directory: projectDir });
    const transform5 = plugin5['experimental.chat.messages.transform'];
    const output5 = makeMessagesOutput('hello');
    await transform5({}, output5);
    const bootstrap5 = bootstrapPartsFor(output5.messages);
    check(bootstrap5.length === 1, `case5: missing asset -> static fallback injected (bootstrap parts=${bootstrap5.length})`);
    const fallbackText = bootstrap5[0]?.text || '';
    check(fallbackText.toLowerCase().includes(FIRST_ACTION_KEY), 'case5: static fallback carries the first-action line');
    check(fallbackText.includes(FALLBACK_MARKER), 'case5: static fallback used (not the live asset)');
    check(!fallbackText.includes(LIVE_MARKER), 'case5: static fallback is the fallback, not the live asset');
  } finally {
    process.env.HOME = realHome;
    fs.renameSync(`${assetPath}.bak`, assetPath);
    fs.rmSync(tmpHome, { recursive: true, force: true });
  }

  // Case 6 — installed layout: plugin loaded from ~/.config/kilo/plugin/ (the
  // installer's distribution location) resolves the checkout asset from the
  // session's project root — __dirname is ~/.config/kilo/plugin/ there.
  {
    const instHome = fs.mkdtempSync(path.join(os.tmpdir(), 'skillgrid-kilo-inst-home-'));
    const instPluginDir = path.join(instHome, '.config', 'kilo', 'plugin');
    fs.mkdirSync(instPluginDir, { recursive: true });
    const instPluginPath = path.join(instPluginDir, 'skillgrid-router.ts');
    fs.writeFileSync(instPluginPath, fs.readFileSync(path.join(path.dirname(fileURLToPath(import.meta.url)), 'skillgrid-router.ts')));
    const instProject = makeProjectDir();
    const instAssetDir = path.join(instProject, '.agents', 'skills', '_shared', 'agent-config');
    fs.mkdirSync(instAssetDir, { recursive: true });
    fs.writeFileSync(path.join(instAssetDir, 'router-bootstrap.md'), fs.readFileSync(assetPath));
    const instRealHome = process.env.HOME;
    process.env.HOME = instHome; // installed copy (~/.skillgrid/...) absent
    try {
      const instMod = await import(pathToFileURL(instPluginPath).href);
      const instPlugin = await instMod.SkillgridRouterPlugin({ client: {}, directory: instProject });
      const instOut = makeMessagesOutput('hello');
      await instPlugin['experimental.chat.messages.transform']({}, instOut);
      const instParts = bootstrapPartsFor(instOut.messages);
      check(instParts.length === 1, 'case6 (installed layout): bootstrap injected once');
      check(
        (instParts[0]?.text || '').includes(LIVE_MARKER),
        'case6 (installed layout): project checkout asset served (not static)',
      );
    } finally {
      process.env.HOME = instRealHome;
      fs.rmSync(instHome, { recursive: true, force: true });
      fs.rmSync(instProject, { recursive: true, force: true });
    }
  }

  // Case 7 — walk-up: session cwd is a subdirectory of the project; the config
  // is found levels above cwd (R3: ".skillgrid/config.yaml at or above cwd").
  const subDir = path.join(projectDir, 'src', 'deep');
  fs.mkdirSync(subDir, { recursive: true });
  __resetBootstrapCacheForTests();
  const plugin7 = await SkillgridRouterPlugin({ client: {}, directory: subDir });
  const output7 = makeMessagesOutput('hello');
  await plugin7['experimental.chat.messages.transform']({}, output7);
  const bootstrap7 = bootstrapPartsFor(output7.messages);
  check(bootstrap7.length === 1, `case7: walk-up from subdir injects bootstrap (parts=${bootstrap7.length})`);

  // Config hook: pushes the session project's .agents/skills dir into
  // config.skills.paths (project-scoped — the old __dirname derivation was
  // dead in the installed layout).
  const pluginCfg = await SkillgridRouterPlugin({ client: {}, directory: projectDir });
  const cfg: { skills?: { paths?: string[] } } = { skills: { paths: [] } };
  await pluginCfg.config(cfg);
  const expectedSkillsDir = path.join(projectDir, '.agents', 'skills');
  check(
    Array.isArray(cfg.skills?.paths) && cfg.skills?.paths.includes(expectedSkillsDir),
    `config: skills.paths pushed with ${expectedSkillsDir}`,
  );

  return failures;
};

const main = async (): Promise<void> => {
  const failures = await runCases();
  if (failures.length > 0) {
    for (const f of failures) console.error(`FAIL: ${f}`);
    process.exit(1);
  }
  console.log('PASS: all kilo router plugin cases green');
};

void main();
