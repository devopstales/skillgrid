"use strict";
// Root entry for jiti import("<skill-pi-root>").
// pi's --extension <dir> path does a direct jiti import() of the directory, which
// resolves package.json -> main -> index.js. A valid pi extension exports a
// factory that receives the ExtensionAPI. T4 replaces this no-op body with the
// real meta-extension that registers the five opencode-behavior ports.
module.exports = (api) => {};
