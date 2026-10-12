"use strict";

// skill-pi auth extension — extends pi's provider list with multi-account OAuth + API-key
// auth for four targets: Kilo Code (device-flow OAuth), opencode zen (device-flow OAuth),
// Ollama Cloud (API-key), and a local OpenAI-compatible endpoint (Ollama / vLLM).
//
// Multi-account model: pi stores one credential per provider id. Multiple accounts for one
// provider = register the provider under multiple ids (kilo, kilo-2, kilo-work, ...). This
// is the pattern pi-multi-auth uses. The /skillgrid-account <provider> [suffix] command
// re-registers <provider>-<suffix> with the shared config so several logged-in identities
// coexist and are switched via the normal /model flow. No rotation/quota/failover engine —
// pi handles credential storage, refresh, and rotation natively once a provider is registered.
//
// All four targets use the openai-completions wire. No fetch happens at registration time;
// refreshModels closures fetch the catalog lazily when pi requests it. CJS throughout.

// ---------------------------------------------------------------------------
// Endpoints (verified from source; opencode zen URLs are placeholders pending
// anomalyco/opencode source confirmation — see blueprint Addendum).
// ---------------------------------------------------------------------------

const KILO_BASE = "https://api.kilo.ai";
const KILO_DEVICE_CODE_PATH = "/api/device-auth/codes";
const KILO_POLL_INTERVAL_MS = 3000;
const KILO_TOKEN_TTL_MS = 365 * 24 * 60 * 60 * 1000; // ~1 year, access==refresh

const ZEN_BASE = "https://opencode.ai/zen/v1"; // placeholder base (opencode zen)
const ZEN_DEVICE_CODE_PATH = "/device-auth/codes"; // placeholder — confirm against opencode source
const ZEN_POLL_INTERVAL_MS = 3000;
const ZEN_TOKEN_TTL_MS = 365 * 24 * 60 * 60 * 1000;

const OLLAMA_CLOUD_BASE = "https://ollama.com/v1";
const OLLAMA_CLOUD_API_BASE = "https://ollama.com";

// Local OpenAI-compatible (Ollama / vLLM). Base URL is configurable via env; defaults to
// the local Ollama OpenAI-compat path. Keyless by default (Ollama/vLLM do not require auth),
// but an optional $SKILL_PI_LOCAL_API_KEY is sent as a Bearer when present.
const LOCAL_OPENAI_DEFAULT_BASE = "http://localhost:11434/v1";

// ---------------------------------------------------------------------------
// Small fetch helper with an injectable fetcher (so tests can stub the network).
// ---------------------------------------------------------------------------

async function jsonGet(url, headers, signal) {
  const fetcher = (module.exports.__test && module.exports.__test.fetcher) || globalThis.fetch;
  const res = await fetcher(url, { headers: headers || {}, signal });
  if (!res.ok) {
    // device-flow polling uses 202 (pending) and 403 (denied) as meaningful codes; let the
    // caller decide. We only surface a throw for other non-2xx.
    if (res.status === 202 || res.status === 403 || res.status === 410) {
      return { status: res.status, body: await safeJson(res) };
    }
    throw new Error(`GET ${url} -> HTTP ${res.status}`);
  }
  return { status: res.status, body: await safeJson(res) };
}

async function jsonPost(url, payload, headers, signal) {
  const fetcher = (module.exports.__test && module.exports.__test.fetcher) || globalThis.fetch;
  const res = await fetcher(url, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...(headers || {}) },
    body: JSON.stringify(payload),
    signal,
  });
  if (!res.ok) throw new Error(`POST ${url} -> HTTP ${res.status}`);
  return { status: res.status, body: await safeJson(res) };
}

async function safeJson(res) {
  try {
    return await res.json();
  } catch {
    return {};
  }
}

function sleep(ms, signal) {
  return new Promise((resolve, reject) => {
    const t = setTimeout(resolve, ms);
    if (signal) {
      signal.addEventListener("abort", () => {
        clearTimeout(t);
        reject(new Error("aborted"));
      }, { once: true });
    }
  });
}

// ---------------------------------------------------------------------------
// OAuth login flows. Both Kilo and opencode zen use the device-code pattern:
//   1. POST <base>/<devicePath> -> { code, verificationUrl, expiresIn }
//   2. callbacks.onAuth({ url: verificationUrl }) + onDeviceCode({ code, ... })
//   3. poll GET <base>/<devicePath>/<code> until the body carries an access token
//   4. return OAuthCredentials { access, refresh, expires }
// ---------------------------------------------------------------------------

function makeDeviceOAuth({ label, base, devicePath, pollIntervalMs, tokenTtlMs, accessTokenKey = "access_token" }) {
  return {
    name: label,
    isSubscription: false,
    async login(callbacks) {
      const signal = callbacks.signal;
      const testInterval = module.exports.__test.pollInterval;
      const pollMs = testInterval != null ? testInterval : pollIntervalMs;
      const started = Date.now();
      const created = await jsonPost(base + devicePath, { editorName: "Pi" }, { "X-KILOCODE-EDITORNAME": "Pi" }, signal);
      const code = created.body.code || created.body.device_code || created.body.verifier;
      const verificationUrl = created.body.verificationUrl || created.body.verification_uri || created.body.verification_url;
      if (!code) throw new Error(`${label}: device-code response missing code`);

      callbacks.onAuth({ url: verificationUrl || `${base}/login` });
      if (verificationUrl) callbacks.onDeviceCode({ code, verificationUrl, expiresIn: created.body.expiresIn || created.body.expires_in });

      // Poll until a token is issued or the flow errors / times out.
      for (;;) {
        if (signal && signal.aborted) throw new Error(`${label}: login aborted`);
        await sleep(pollMs, signal);
        const poll = await jsonGet(`${base}${devicePath}/${code}`, { "X-KILOCODE-EDITORNAME": "Pi" }, signal);
        const b = poll.body || {};
        const token = b[accessTokenKey] || b.access_token || b.token || b.accessToken;
        if (token) {
          const expires = Date.now() + (Number(b.expires_in) * 1000 || tokenTtlMs);
          return { access: token, refresh: token, expires };
        }
        if (poll.status === 403) throw new Error(`${label}: authorization denied`);
        if (poll.status === 410) throw new Error(`${label}: device code expired`);
        if (Date.now() - started > (created.body.expiresIn || created.body.expires_in || 600) * 1000) {
          throw new Error(`${label}: device code expired (timeout)`);
        }
      }
    },
    async refreshToken(credentials, signal) {
      // Kilo/zen tokens are self-contained: reuse the access token, extend expiry.
      void signal;
      return { access: credentials.access, refresh: credentials.refresh, expires: Date.now() + tokenTtlMs };
    },
    getApiKey(credentials) {
      return credentials.access;
    },
  };
}

// ---------------------------------------------------------------------------
// Model catalog helpers. Each refreshModels maps a GET /v1/models body to the
// ProviderChatModelConfig[] shape pi expects. Cost is 0 for local/self-hosted; a
// conservative default for remote until a real price table is confirmed.
// ---------------------------------------------------------------------------

const DEFAULT_CONTEXT_WINDOW = 128000;
const DEFAULT_MAX_TOKENS = 8192;

function modelFromBody(entry, { input = ["text"], reasoning = false, contextWindow = DEFAULT_CONTEXT_WINDOW, maxTokens = DEFAULT_MAX_TOKENS, cost = ZERO_COST } = {}) {
  const id = entry.id || entry.model;
  if (!id) return null;
  return {
    id,
    name: entry.name || id,
    api: "openai-completions",
    input,
    reasoning,
    contextWindow: Number(entry.context_window || entry.contextLength || contextWindow),
    maxTokens: Number(entry.max_tokens || entry.maxTokens || maxTokens),
    cost: entry.cost || cost,
  };
}

const ZERO_COST = { input: 0, output: 0, cacheRead: 0, cacheWrite: 0 };

// Map a standard OpenAI /v1/models response body -> ProviderModelConfig[].
function mapModelsList(body, opts) {
  const list = Array.isArray(body) ? body : body.data || body.models || [];
  const out = [];
  for (const entry of list) {
    const m = modelFromBody(entry, opts);
    if (m) out.push(m);
  }
  return out;
}

async function refreshOpenAIModels(base, { input, reasoning, contextWindow, maxTokens, cost, headers, apiBase } = {}) {
  // OpenAI-compat discovery endpoint is <base>/models (base already includes /v1).
  const modelsUrl = base.endsWith("/models") ? base : `${base}/models`;
  const got = await jsonGet(modelsUrl, headers);
  return mapModelsList(got.body, { input, reasoning, contextWindow, maxTokens, cost });
}

// ---------------------------------------------------------------------------
// Provider configs. Each returns a ProviderConfig for pi.registerProvider(name, config).
// ---------------------------------------------------------------------------

function makeKiloConfig() {
  return {
    name: "Kilo Code",
    baseUrl: KILO_BASE,
    api: "openai-completions",
    // A dummy key is required by pi when models are defined and no oauth login has run yet;
    // the real bearer comes from oauth.getApiKey once logged in.
    apiKey: "$KILO_API_KEY",
    authHeader: true,
    oauth: makeDeviceOAuth({
      label: "Kilo Code",
      base: KILO_BASE,
      devicePath: KILO_DEVICE_CODE_PATH,
      pollIntervalMs: KILO_POLL_INTERVAL_MS,
      tokenTtlMs: KILO_TOKEN_TTL_MS,
    }),
    async refreshModels() {
      return refreshOpenAIModels(`${KILO_BASE}/v1`, { input: ["text", "image"] });
    },
  };
}

function makeZenConfig() {
  return {
    name: "opencode zen",
    baseUrl: ZEN_BASE,
    api: "openai-completions",
    apiKey: "$OPENCODE_API_KEY",
    authHeader: true,
    oauth: makeDeviceOAuth({
      label: "opencode zen",
      base: ZEN_BASE,
      devicePath: ZEN_DEVICE_CODE_PATH,
      pollIntervalMs: ZEN_POLL_INTERVAL_MS,
      tokenTtlMs: ZEN_TOKEN_TTL_MS,
    }),
    async refreshModels() {
      return refreshOpenAIModels(ZEN_BASE, { input: ["text", "image"] });
    },
  };
}

function makeOllamaCloudConfig() {
  return {
    name: "Ollama Cloud",
    baseUrl: OLLAMA_CLOUD_BASE,
    api: "openai-completions",
    apiKey: "$OLLAMA_API_KEY",
    authHeader: true,
    async refreshModels() {
      // Catalog keyless on the OpenAI-compat surface; /api/show gives per-model detail.
      const models = await refreshOpenAIModels(OLLAMA_CLOUD_BASE, { input: ["text"], cost: ZERO_COST });
      return models;
    },
  };
}

function makeLocalOpenAIConfig(env = process.env) {
  const base = env.SKILL_PI_LOCAL_BASE_URL || LOCAL_OPENAI_DEFAULT_BASE;
  const cfg = {
    name: "Local OpenAI",
    baseUrl: base,
    api: "openai-completions",
    authHeader: false, // keyless by default (Ollama/vLLM need no auth)
    async refreshModels() {
      return refreshOpenAIModels(base, { input: ["text"], cost: ZERO_COST });
    },
  };
  // Optional bearer for vLLM deployments that require a key.
  if (env.SKILL_PI_LOCAL_API_KEY) {
    cfg.apiKey = "$SKILL_PI_LOCAL_API_KEY";
    cfg.authHeader = true;
  }
  return cfg;
}

// ---------------------------------------------------------------------------
// Registration.
// ---------------------------------------------------------------------------

const PROVIDERS = {
  kilo: makeKiloConfig,
  "opencode-zen": makeZenConfig,
  "ollama-cloud": makeOllamaCloudConfig,
  "local-openai": () => makeLocalOpenAIConfig(),
};

function providerId(base, suffix) {
  return suffix ? `${base}-${suffix}` : base;
}

function registerAccounts(pi, { env = process.env } = {}) {
  const registered = [];
  const registerOne = (baseId, suffix) => {
    const factory = PROVIDERS[baseId];
    if (!factory) return { id: providerId(baseId, suffix), ok: false, error: `unknown provider ${baseId}` };
    const id = providerId(baseId, suffix);
    const config = baseId === "local-openai" ? makeLocalOpenAIConfig(env) : factory();
    pi.registerProvider(id, config);
    registered.push(id);
    return { id, ok: true };
  };

  // Base registrations (one per provider).
  for (const baseId of Object.keys(PROVIDERS)) {
    registerOne(baseId);
  }
  return { registered, registerOne };
}

// The extension factory. Registers all four base providers on load and exposes the
// /skillgrid-account command for multi-account registration.
function factory(pi) {
  const { registerOne } = registerAccounts(pi);

  pi.registerCommand("skillgrid-account", {
    description: "Register a second account for a provider: /skillgrid-account <kilo|opencode-zen|ollama-cloud|local-openai> [suffix]",
    handler: async (args, ctx) => {
      const [baseId, suffix] = (args || "").trim().split(/\s+/);
      if (!baseId || !PROVIDERS[baseId]) {
        ctx.ui.notify("usage: /skillgrid-account <kilo|opencode-zen|ollama-cloud|local-openai> [suffix]", "error");
        return;
      }
      const result = registerOne(baseId, suffix);
      if (!result.ok) {
        ctx.ui.notify(result.error, "error");
        return;
      }
      ctx.ui.notify(`registered ${result.id} — use /model to switch to it`, "info");
    },
  });
}

if (typeof module !== "undefined") {
  module.exports = factory;
  module.exports.__test = {
    KILO_BASE,
    KILO_DEVICE_CODE_PATH,
    ZEN_BASE,
    ZEN_DEVICE_CODE_PATH,
    OLLAMA_CLOUD_BASE,
    OLLAMA_CLOUD_API_BASE,
    LOCAL_OPENAI_DEFAULT_BASE,
    ZERO_COST,
    DEFAULT_CONTEXT_WINDOW,
    DEFAULT_MAX_TOKENS,
    jsonGet,
    jsonPost,
    sleep,
    safeJson,
    makeDeviceOAuth,
    modelFromBody,
    mapModelsList,
    refreshOpenAIModels,
    makeKiloConfig,
    makeZenConfig,
    makeOllamaCloudConfig,
    makeLocalOpenAIConfig,
    PROVIDERS,
    providerId,
    registerAccounts,
    factory,
    setFetcher(fn) {
      module.exports.__test.fetcher = fn;
    },
    clearFetcher() {
      module.exports.__test.fetcher = undefined;
    },
    // Tests override the device-flow poll interval so the suite runs without real 3s waits.
    setPollInterval(ms) {
      module.exports.__test.pollInterval = ms;
    },
  };
}
