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

  assert.strictEqual(pkg.piConfig && pkg.piConfig.name, 'skill-pi');
  assert.strictEqual(pkg.piConfig && pkg.piConfig.configDir, '.skill-pi');
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
