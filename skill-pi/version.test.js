const test = require('node:test');
const assert = require('node:assert');
const fs = require('node:fs');
const path = require('node:path');

const skillPiDir = path.join(__dirname);
const repoRoot = path.join(__dirname, '..');

test('skill-pi module skeleton is correct', () => {
  const pkgPath = path.join(skillPiDir, 'package.json');
  const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8'));

  const dep = pkg.dependencies && pkg.dependencies['@earendil-works/pi-coding-agent'];
  assert.ok(dep, 'dependencies["@earendil-works/pi-coding-agent"] must exist');
  assert.match(dep, /^\d+\.\d+\.\d+$/, 'version must be an exact pin, got: ' + dep);

  const piVersionPath = path.join(skillPiDir, 'pi-version.txt');
  const piVersion = fs.readFileSync(piVersionPath, 'utf8').trim();
  assert.strictEqual(piVersion, dep, 'pi-version.txt must equal package.json dependency string');

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
