import fs from 'fs';
import os from 'os';
import path from 'path';
import { fileURLToPath } from 'url';
import { SkillgridRouterPlugin, __resetBootstrapCacheForTests } from './skillgrid-router.ts';

type TestPart = { type?: string; text?: string; prompt?: string; description?: string; agent?: string };
type TestMessage = { info?: { role?: string; mode?: string }; parts?: TestPart[] };
type TestMessagesOutput = { messages?: TestMessage[] };

const FIRST_ACTION_KEY = 'first action';
const FALLBACK_MARKER = 'fallback';
const LIVE_MARKER = 'This project is configured with Skillgrid';

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
const assetPath = path.join(repoRoot, '.agents', 'skills', '_shared', 'agent-config', 'router-bootstrap.md');
const skillsDir = path.join(repoRoot, '.agents', 'skills');

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

  // Case 5 — missing asset: static fallback.
  __resetBootstrapCacheForTests();
  fs.renameSync(assetPath, `${assetPath}.bak`);
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
    fs.renameSync(`${assetPath}.bak`, assetPath);
  }

  // Config hook: pushes the repo .agents/skills dir into config.skills.paths.
  const pluginCfg = await SkillgridRouterPlugin({ client: {}, directory: projectDir });
  const cfg: { skills?: { paths?: string[] } } = { skills: { paths: [] } };
  await pluginCfg.config(cfg);
  check(
    Array.isArray(cfg.skills?.paths) && cfg.skills?.paths.includes(skillsDir),
    `config: skills.paths pushed with ${skillsDir}`,
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
