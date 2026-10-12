const test = require('node:test');
const assert = require('node:assert');
const fs = require('node:fs');
const path = require('node:path');

const skillPiDir = path.join(__dirname);
const repoRoot = path.join(__dirname, '..');

test('skill-pi module skeleton is correct', () => {
  const pkgPath = path.join(skillPiDir, 'package.json');
  const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8'));

  // skill-pi is a self-contained pi distribution: pi is a bundled exact
  // dependency (not a peer), so `npm i -g skill-pi` ships the pinned runtime
  // under the package's own node_modules. The exact pin (R12 / ADR-0042) is
  // recorded in pi-version.txt and must equal the dependency pin.
  const dep = pkg.dependencies && pkg.dependencies['@earendil-works/pi-coding-agent'];
  assert.ok(dep, 'dependencies["@earendil-works/pi-coding-agent"] must exist');

  const piVersionPath = path.join(skillPiDir, 'pi-version.txt');
  const piVersion = fs.readFileSync(piVersionPath, 'utf8').trim();
  assert.match(piVersion, /^\d+\.\d+\.\d+$/, 'pi-version.txt must be an exact pin, got: ' + piVersion);
  assert.strictEqual(dep, piVersion, 'dependency pin must equal pi-version.txt, got: ' + dep);

  // Bundled, so a global install self-contains pi.
  assert.ok(
    Array.isArray(pkg.bundleDependencies) && pkg.bundleDependencies.includes('@earendil-works/pi-coding-agent'),
    'bundleDependencies must include pi so a global install ships it'
  );

  // pi is no longer host-provided: it must not remain a peerDependency.
  assert.strictEqual(
    pkg.peerDependencies && pkg.peerDependencies['@earendil-works/pi-coding-agent'],
    undefined,
    'pi must not be a peerDependency now that it is a bundled dependency'
  );

  // Both bins ship: skill-pi (canonical) and pi (distribution pass-through).
  assert.strictEqual(pkg.bin && pkg.bin['pi'], './bin/pi.mjs');
  assert.ok(fs.existsSync(path.join(skillPiDir, 'bin', 'pi.mjs')), 'bin/pi.mjs must exist');

  // pi 1.1.0 reads piConfig.configDir from the package dir the launcher points
  // at via PI_PACKAGE_DIR (= skill-pi/). That is what sets the project config
  // dir to ".skill-pi". The `pi` key is the resource manifest (extensions,
  // skills, prompts, themes). pi is now a bundled runtime dependency (exact
  // 1.1.0) — the peer/host-provided model was superseded 2026-10-12, see
  // ADR-0042 "Supersession" section.
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
