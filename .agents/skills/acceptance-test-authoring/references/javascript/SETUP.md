# JavaScript Stack - cucumber-js

For `bdd.stack: javascript` in `.skillgrid/config.yaml`. Read the skill's `SKILL.md` first for the spec format and runner invariants, plus [../EXTRACTION.md](../EXTRACTION.md) — the normative contract this pack implements.

## Files To Copy

Copy into `acceptance-tests/` at the repo root. Destination filenames are load-bearing.

| Source | Destination | Why the name is fixed |
| --- | --- | --- |
| `javascript/extract-gherkin.cjs` | `extract-gherkin.cjs` | `cucumber.cjs` requires it as `./extract-gherkin.cjs` |
| `javascript/cucumber.cjs` | `cucumber.cjs` | cucumber-js discovers its configuration by this exact name |
| `javascript/skillgrid-paths.cjs` | `skillgrid-paths.cjs` | required by `cucumber.cjs` under this name |
| `../gherkin-lintrc.json` | `.gherkin-lintrc` | gherkin-lint auto-discovers this name and has no built-in defaults |

The fourth file is from the shared `references/` root, not this folder.

## Project Setup

`acceptance-tests/` is an independent Node project with its own `package.json`. Its hooks boot the application before the suite and shut it down after, so the suite runs with one command.

devDependencies: `@cucumber/cucumber`, `glob`, `cheerio`, `gherkin-lint`.

```json
"scripts": {
  "test": "cucumber-js",
  "lint:specs": "node extract-gherkin.cjs && gherkin-lint .extracted"
}
```

`npm test` stays a plain `cucumber-js` call: `cucumber.cjs` runs `extractAll()` synchronously at config load, so the suite always sees a freshly rebuilt `.extracted/`.

## Discovery

cucumber-js runs every `.feature` file under `.extracted/`. `effectivePaths()` in `skillgrid-paths.cjs` returns a simple glob (`.extracted/**/*.feature`) — no line-targeted filtering, no supersession. C1 is "everything runs."

## Verification

```sh
npx cucumber-js --dry-run
node -e "console.log(require('./cucumber.cjs').default.paths)"
```

Resolved paths must show only `.extracted/` entries.

For a name-level diff, read `testCase` envelopes, not `pickle` envelopes.

```sh
npx cucumber-js --dry-run -f message | \
  node -e "let p={},s=[];require('readline').createInterface({input:process.stdin}).on('line',l=>{try{const m=JSON.parse(l);if(m.pickle)p[m.pickle.id]=m.pickle.name;if(m.testCase)s.push(m.testCase.pickleId)}catch{}}).on('close',()=>console.log(s.map(i=>p[i]).sort().join('\\n')))"
```

## Linting

Pass `.extracted` as a directory argument. A quoted `'**'` glob silently matches nothing through the dot-directory. Reported line numbers are valid in source `acceptance.feature`.

Known limitation: gherkin-lint's AST rules do not descend into `Rule:` children, so `no-unnamed-scenarios` misses scenarios nested under a Rule. Line-based rules and feature-level rules work.

## Page Object Model

Page objects live in `acceptance-tests/support/pages/`, one per screen or flow. Parse responses with `cheerio`; never with regexes over raw HTML. The World stays a thin HTTP client holding page-object instances.

```js
When('they submit a valid email and password', async function () {
  this.result = await this.signupPage.submitSignup({
    email: 'user@example.com',
    password: 'correct-horse-battery-staple',
  });
});

Then('an error message is shown', function () {
  assert.ok(this.signupPage.errorMessage());
});
```
