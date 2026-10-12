const test = require('node:test');
const assert = require('node:assert');
const fs = require('node:fs');
const path = require('node:path');

const skillPiDir = path.join(__dirname);
const repoRoot = path.join(__dirname, '..');

test('skill-pi module skeleton is correct', () => {
  const pkgPath = path.join(skillPiDir, 'package.json');
  const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8'));

  // pi is host-provided: it must be a peerDependency with "*" (not a runtime
  // dependency, which would create a duplicate module — see pi's loader warning).
  const peer = pkg.peerDependencies && pkg.peerDependencies['@earendil-works/pi-coding-agent'];
  assert.ok(peer, 'peerDependencies["@earendil-works/pi-coding-agent"] must exist');
  assert.strictEqual(peer, '*', 'peer dependency range must be "*", got: ' + peer);

  // The exact pin (R12 / ADR-0042) is enforced at install time (Taskfile
  // skill-pi:install) and recorded in pi-version.txt.
  const piVersionPath = path.join(skillPiDir, 'pi-version.txt');
  const piVersion = fs.readFileSync(piVersionPath, 'utf8').trim();
  assert.match(piVersion, /^\d+\.\d+\.\d+$/, 'pi-version.txt must be an exact pin, got: ' + piVersion);

  assert.strictEqual(
    pkg.dependencies && pkg.dependencies['@earendil-works/pi-coding-agent'],
    undefined,
    'pi must not also be a runtime dependency (duplicate module risk)'
  );

  // pi 1.1.0 reads piConfig.configDir from the package dir the launcher points
  // at via PI_PACKAGE_DIR (= skill-pi/). That is what sets the project config
  // dir to ".skill-pi". The `pi` key is the resource manifest (extensions,
  // skills, prompts, themes). pi must not be a runtime dependency (ADR-0042).
  assert.strictEqual(pkg.piConfig && pkg.piConfig.name, 'skill-pi');
  assert.strictEqual(pkg.piConfig && pkg.piConfig.configDir, '.skill-pi');
  assert.ok(
    Array.isArray(pkg.pi && pkg.pi.extensions),
    'pi.extensions must be an array manifest entry'
  );
  assert.strictEqual(pkg.bin && pkg.bin['skill-pi'], './bin/skill-pi.js');

  const readme = fs.readFileSync(path.join(skillPiDir, 'README.md'), 'utf8');
  assert.ok(readme.includes('1.1.0'), 'README.md must name the min pi version 1.1.0');

  assert.ok(
    fs.existsSync(path.join(repoRoot, '.skillgrid', 'artifacts', '04-adr-0042-skill-pi-dependencies.md')),
    'ADR-0042 file must exist'
  );

  const assumptions = fs.readFileSync(path.join(repoRoot, '.skillgrid', 'ASSUMPTIONS.md'), 'utf8');
  assert.ok(assumptions.includes('0042'), 'ASSUMPTIONS.md must index ADR-0042');
});

// --- Task 3: global config + system prompt delta (R16, R17) ---

const CONFIG_FILES = ['settings.json', 'mcp.json'];
const PROVIDER_URL_RE = /https?:\/\/[a-z0-9.-]+\/(v1|api)/i;

test('config: no remote provider base URL literal in settings.json and mcp.json', () => {
  for (const file of CONFIG_FILES) {
    const p = path.join(skillPiDir, file);
    assert.ok(fs.existsSync(p), file + ' must exist');
    const text = fs.readFileSync(p, 'utf8');
    const hits = text.match(PROVIDER_URL_RE);
    assert.strictEqual(
      hits,
      null,
      file + ' must not contain a remote provider base URL (https?://host/v1|api); found: ' +
        (hits ? JSON.stringify(hits) : 'none')
    );
  }
});

test('config: base URL resolution reads env, not a literal (default + override)', () => {
  const helperPath = path.join(skillPiDir, 'lib', 'base-url.js');
  assert.ok(fs.existsSync(helperPath), 'lib/base-url.js must exist');

  // Delete the require cache so the helper re-reads process.env on each call.
  delete require.cache[require.resolve(helperPath)];
  const { resolveBaseUrl } = require(helperPath);

  const savedCheckpoint = process.env.SKILLGRID_CHECKPOINT_URL;
  const savedMnemonic = process.env.SKILLGRID_MNEMONIC_HTTP_URL;
  try {
    // Default: no env set.
    delete process.env.SKILLGRID_CHECKPOINT_URL;
    delete process.env.SKILLGRID_MNEMONIC_HTTP_URL;
    assert.strictEqual(
      resolveBaseUrl(),
      'http://127.0.0.1:7438',
      'default base URL must be http://127.0.0.1:7438 when no env is set'
    );

    // Override: SKILLGRID_MNEMONIC_HTTP_URL wins.
    delete process.env.SKILLGRID_CHECKPOINT_URL;
    process.env.SKILLGRID_MNEMONIC_HTTP_URL = 'http://127.0.0.1:8787';
    assert.strictEqual(
      resolveBaseUrl(),
      'http://127.0.0.1:8787',
      'SKILLGRID_MNEMONIC_HTTP_URL must override the default'
    );

    // Override: SKILLGRID_CHECKPOINT_URL wins over SKILLGRID_MNEMONIC_HTTP_URL.
    process.env.SKILLGRID_CHECKPOINT_URL = 'http://127.0.0.1:9999';
    assert.strictEqual(
      resolveBaseUrl(),
      'http://127.0.0.1:9999',
      'SKILLGRID_CHECKPOINT_URL must take precedence over SKILLGRID_MNEMONIC_HTTP_URL'
    );
  } finally {
    if (savedCheckpoint === undefined) delete process.env.SKILLGRID_CHECKPOINT_URL;
    else process.env.SKILLGRID_CHECKPOINT_URL = savedCheckpoint;
    if (savedMnemonic === undefined) delete process.env.SKILLGRID_MNEMONIC_HTTP_URL;
    else process.env.SKILLGRID_MNEMONIC_HTTP_URL = savedMnemonic;
  }
});

test('config: SYSTEM.md is a pure delta (zero grep hits for persona/engram markers)', () => {
  const p = path.join(skillPiDir, 'system', 'SYSTEM.md');
  assert.ok(fs.existsSync(p), 'system/SYSTEM.md must exist');
  const text = fs.readFileSync(p, 'utf8');
  const hits = text.match(/gentle-ai:persona|gentle-ai:engram-protocol/g);
  assert.strictEqual(
    hits,
    null,
    'SYSTEM.md must not contain the gentle-ai:persona or gentle-ai:engram-protocol markers; found: ' +
      (hits ? JSON.stringify(hits) : 'none')
  );

  // The delta sections must be present.
  assert.ok(
    /zone\s+discipline/i.test(text),
    'SYSTEM.md must contain a zone discipline section'
  );
  assert.ok(
    /commit\s+contract/i.test(text),
    'SYSTEM.md must contain a commit contract section'
  );
  // A pointer to AGENTS.md for the persona/engram blocks must be present.
  assert.ok(
    /AGENTS\.md/.test(text),
    'SYSTEM.md must reference AGENTS.md for the persona/engram blocks'
  );
});
