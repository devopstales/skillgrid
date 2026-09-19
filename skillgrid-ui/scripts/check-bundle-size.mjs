// Phase 7.3 bundle-size budget check. Run after `vite build`. Asserts the
// initial (index) JS chunk stays under the budget so route code-splitting
// hasn't regressed. Exits non-zero over budget (CI gate).
import { readdirSync, statSync } from 'node:fs'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const here = dirname(fileURLToPath(import.meta.url))
// scripts/ → skillgrid-ui/ → repo root → skillgrid-cli/internal/.../dist/assets
const dist = join(here, '../../skillgrid-cli/internal/mnemonic/http/ui/dist/assets')

// Budgets (kB, raw size). The initial index chunk must stay under this —
// per-feature chunks are lazy and exempt (they only load on route visit).
const INDEX_BUDGET_KB = 400

function listJs() {
  return readdirSync(dist)
    .filter((f) => f.endsWith('.js'))
    .map((f) => ({ name: f, kb: statSync(join(dist, f)).size / 1024 }))
    .sort((a, b) => b.kb - a.kb)
}

const chunks = listJs()
let failed = false

console.log('\n=== Bundle-size budget (Phase 7.3) ===')
for (const c of chunks.slice(0, 15)) {
  console.log(`  ${c.kb.toFixed(1).padStart(8)} kB  ${c.name}`)
}
console.log(`  ... ${Math.max(0, chunks.length - 15)} more chunks`)

const index = chunks.find((c) => /^index-.*\.js$/.test(c.name))
if (!index) {
  console.log('  (no index-*.js chunk found)')
} else {
  const over = index.kb > INDEX_BUDGET_KB
  console.log(
    `  ${index.kb.toFixed(1)} kB  index (budget ${INDEX_BUDGET_KB} kB) — ${over ? 'OVER BUDGET' : 'OK'}`,
  )
  if (over) failed = true
}

if (failed) {
  console.error(`\nBundle-size budget EXCEEDED. Increase the budget in check-bundle-size.mjs or reduce the index chunk.`)
  process.exit(1)
}
console.log('\nBundle-size budget OK.\n')
