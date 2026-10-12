'use strict';

// Base URL resolution for skill-pi extension ports (Task 4) and mcp-bridge (Task 5).
//
// Mirrors the opencode plugins (plugins/opencode/mnemonic-memory.ts, etc.):
//   SKILLGRID_CHECKPOINT_URL > SKILLGRID_MNEMONIC_HTTP_URL > loopback default.
//
// The JSON config files (settings.json, mcp.json) carry NO URL literal — this
// helper is the single source of the default, so `mnemonic ctx wrap pi` can
// override the base URL purely via env (the loopback :8787 proxy).
//
// stdlib only; no pi import. CJS (require/module.exports).

const DEFAULT_BASE_URL = 'http://127.0.0.1:7438';

/**
 * Resolve the skillgrid HTTP base URL from the environment.
 *
 * @param {NodeJS.ProcessEnv} [env] - environment to read (defaults to process.env)
 * @returns {string} the effective base URL
 */
function resolveBaseUrl(env) {
  const e = env || process.env;
  return e.SKILLGRID_CHECKPOINT_URL || e.SKILLGRID_MNEMONIC_HTTP_URL || DEFAULT_BASE_URL;
}

module.exports = { resolveBaseUrl, DEFAULT_BASE_URL };
