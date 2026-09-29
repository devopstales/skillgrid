'use strict';

const { globSync } = require('glob');

// C1: no delta composition, no supersession. Every extracted feature runs.
// This is the collapsed form of the original openspec-effective-paths.cjs.
function effectivePaths() {
  return globSync('.extracted/**/*.feature', {
    cwd: __dirname,
    posix: true,
  }).sort();
}

module.exports = { effectivePaths };
