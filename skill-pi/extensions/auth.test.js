"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");

const factory = require("./auth.js");
const T = factory.__test;

// Fast device-flow polling for the test run (the real flow uses 3000ms).
T.setPollInterval(10);

// ---------------------------------------------------------------------------
// Registration
// ---------------------------------------------------------------------------

test("factory is a function exporting registerProvider-driven providers", () => {
  assert.equal(typeof factory, "function");
  assert.equal(typeof T, "object");
  const expected = ["kilo", "opencode-zen", "ollama-cloud", "local-openai"];
  for (const id of expected) {
    assert.ok(T.PROVIDERS[id], `PROVIDERS missing ${id}`);
  }
});

function makeFakePi() {
  const registered = new Map();
  const commands = new Map();
  return {
    registered,
    commands,
    registerProvider(id, config) {
      registered.set(id, config);
    },
    unregisterProvider(id) {
      registered.delete(id);
    },
    registerCommand(name, def) {
      commands.set(name, def);
    },
    on() {},
  };
}

test("registerAccounts registers all four base providers with correct ids", () => {
  const pi = makeFakePi();
  const { registered } = T.registerAccounts(pi);
  assert.deepEqual(
    registered.sort(),
    ["kilo", "local-openai", "ollama-cloud", "opencode-zen"].sort(),
  );
  assert.ok(pi.registered.has("kilo"));
  assert.ok(pi.registered.has("opencode-zen"));
  assert.ok(pi.registered.has("ollama-cloud"));
  assert.ok(pi.registered.has("local-openai"));
});

test("provider baseUrls match the verified endpoints", () => {
  const pi = makeFakePi();
  T.registerAccounts(pi);
  assert.equal(pi.registered.get("kilo").baseUrl, "https://api.kilo.ai");
  assert.equal(pi.registered.get("opencode-zen").baseUrl, "https://opencode.ai/zen/v1");
  assert.equal(pi.registered.get("ollama-cloud").baseUrl, "https://ollama.com/v1");
  assert.equal(pi.registered.get("local-openai").baseUrl, "http://localhost:11434/v1");
});

test("all providers use the openai-completions wire", () => {
  const pi = makeFakePi();
  T.registerAccounts(pi);
  for (const [id, cfg] of pi.registered) {
    assert.equal(cfg.api, "openai-completions", `${id} api`);
  }
});

test("ollama-cloud sends the OLLAMA_API_KEY as a bearer (authHeader)", () => {
  const cfg = T.makeOllamaCloudConfig();
  assert.equal(cfg.apiKey, "$OLLAMA_API_KEY");
  assert.equal(cfg.authHeader, true);
});

test("local-openai is keyless by default and honors SKILL_PI_LOCAL_BASE_URL", () => {
  const cfg = T.makeLocalOpenAIConfig({ SKILL_PI_LOCAL_BASE_URL: "http://localhost:8000/v1" });
  assert.equal(cfg.baseUrl, "http://localhost:8000/v1");
  assert.equal(cfg.authHeader, false);
  assert.equal(cfg.apiKey, undefined);
});

test("local-openai sends a bearer when SKILL_PI_LOCAL_API_KEY is set", () => {
  const cfg = T.makeLocalOpenAIConfig({ SKILL_PI_LOCAL_API_KEY: "sk-local" });
  assert.equal(cfg.apiKey, "$SKILL_PI_LOCAL_API_KEY");
  assert.equal(cfg.authHeader, true);
});

test("multi-account: registerOne registers kilo-work under a distinct id", () => {
  const pi = makeFakePi();
  const { registerOne } = T.registerAccounts(pi);
  const result = registerOne("kilo", "work");
  assert.equal(result.ok, true);
  assert.equal(result.id, "kilo-work");
  assert.ok(pi.registered.has("kilo"));
  assert.ok(pi.registered.has("kilo-work"));
  assert.notEqual(pi.registered.get("kilo"), pi.registered.get("kilo-work"), "independent config objects");
});

test("multi-account: unknown provider base returns not-ok", () => {
  const pi = makeFakePi();
  const { registerOne } = T.registerAccounts(pi);
  const result = registerOne("nope", "x");
  assert.equal(result.ok, false);
  assert.match(result.error, /unknown provider/);
});

test("factory wires the /skillgrid-account command", () => {
  const pi = makeFakePi();
  factory(pi);
  assert.ok(pi.commands.has("skillgrid-account"));
  // After the factory runs, all four base providers are registered.
  assert.ok(pi.registered.has("kilo"));
  assert.ok(pi.registered.has("opencode-zen"));
});

// ---------------------------------------------------------------------------
// OAuth login flows (device-code)
// ---------------------------------------------------------------------------

function makeDeviceCallbacks({ approvals, token = "device-token" } = {}) {
  let authUrl;
  let deviceCode;
  const cbs = {
    onAuth: (info) => {
      authUrl = info.url;
    },
    onDeviceCode: (info) => {
      deviceCode = info;
    },
    signal: undefined,
  };
  return { cbs, getAuthUrl: () => authUrl, getDeviceCode: () => deviceCode };
}

test("kilo device-flow login drives onAuth + onDeviceCode and returns credentials", async () => {
  let createCalls = 0;
  let pollCalls = 0;
  T.setFetcher(async (url, init) => {
    if (init && init.method === "POST") {
      createCalls += 1;
      return { ok: true, status: 200, json: async () => ({ code: "c1", verificationUrl: "https://kilo.ai/login?c=c1", expiresIn: 600 }) };
    }
    pollCalls += 1;
    if (pollCalls < 2) {
      return { ok: true, status: 202, json: async () => ({ status: "pending" }) };
    }
    return { ok: true, status: 200, json: async () => ({ access_token: "kilo-access" }) };
  });

  const oauth = T.makeKiloConfig().oauth;
  const { cbs, getAuthUrl, getDeviceCode } = makeDeviceCallbacks();
  const creds = await oauth.login(cbs);

  assert.equal(getAuthUrl(), "https://kilo.ai/login?c=c1");
  assert.equal(getDeviceCode().code, "c1");
  assert.equal(creds.access, "kilo-access");
  assert.equal(creds.refresh, "kilo-access");
  assert.ok(creds.expires > Date.now());
  assert.equal(createCalls, 1);
  T.clearFetcher();
});

test("kilo device-flow login surfaces denial (403) as an error", async () => {
  T.setFetcher(async (url, init) => {
    if (init && init.method === "POST") {
      return { ok: true, status: 200, json: async () => ({ code: "c1", verificationUrl: "https://kilo.ai/login", expiresIn: 600 }) };
    }
    return { ok: false, status: 403, json: async () => ({ status: "denied" }) };
  });
  const oauth = T.makeKiloConfig().oauth;
  const { cbs } = makeDeviceCallbacks();
  await assert.rejects(() => oauth.login(cbs), /denied/);
  T.clearFetcher();
});

test("getApiKey maps the access token to the bearer key", () => {
  const oauth = T.makeKiloConfig().oauth;
  assert.equal(oauth.getApiKey({ access: "abc", refresh: "abc", expires: 0 }), "abc");
});

test("refreshToken reuses the token and extends expiry", async () => {
  const oauth = T.makeKiloConfig().oauth;
  const before = Date.now();
  const creds = await oauth.refreshToken({ access: "abc", refresh: "abc", expires: 0 }, undefined);
  assert.equal(creds.access, "abc");
  assert.ok(creds.expires >= before);
});

test("zen device-flow login uses the opencode zen endpoints and returns credentials", async () => {
  let sawUrl = "";
  T.setFetcher(async (url, init) => {
    sawUrl = url;
    if (init && init.method === "POST") {
      return { ok: true, status: 200, json: async () => ({ code: "zc", verificationUrl: "https://opencode.ai/zen/login?c=zc", expiresIn: 600 }) };
    }
    return { ok: true, status: 200, json: async () => ({ access_token: "zen-access" }) };
  });
  const oauth = T.makeZenConfig().oauth;
  const { cbs, getAuthUrl } = makeDeviceCallbacks();
  const creds = await oauth.login(cbs);
  assert.equal(creds.access, "zen-access");
  assert.match(sawUrl, /opencode\.ai\/zen\/v1/);
  assert.equal(getAuthUrl(), "https://opencode.ai/zen/login?c=zc");
  T.clearFetcher();
});

// ---------------------------------------------------------------------------
// refreshModels (catalog)
// ---------------------------------------------------------------------------

test("refreshOpenAIModels maps a standard /v1/models body to ProviderModelConfig[]", async () => {
  T.setFetcher(async () => ({
    ok: true,
    status: 200,
    json: async () => ({ data: [{ id: "gpt-4o", name: "GPT-4o" }, { id: "llama-3", name: "Llama 3" }] }),
  }));
  const models = await T.refreshOpenAIModels("https://example.com/v1", { input: ["text", "image"] });
  assert.equal(models.length, 2);
  for (const m of models) {
    assert.ok(m.id && m.name && m.input && m.contextWindow && m.maxTokens);
    assert.equal(m.api, "openai-completions");
    assert.ok(m.cost);
  }
  assert.equal(models[0].id, "gpt-4o");
  T.clearFetcher();
});

test("ollama-cloud refreshModels hits ollama.com/v1/models", async () => {
  let hit = "";
  T.setFetcher(async (url) => {
    hit = url;
    return { ok: true, status: 200, json: async () => ({ data: [{ id: "llama3.1", name: "llama3.1" }] }) };
  });
  const models = await T.makeOllamaCloudConfig().refreshModels();
  assert.match(hit, /ollama\.com\/v1\/models/);
  assert.equal(models.length, 1);
  assert.equal(models[0].cost.input, 0);
  T.clearFetcher();
});

test("local-openai refreshModels honors SKILL_PI_LOCAL_BASE_URL", async () => {
  let hit = "";
  T.setFetcher(async (url) => {
    hit = url;
    return { ok: true, status: 200, json: async () => ({ data: [{ id: "qwen2.5", name: "qwen2.5" }] }) };
  });
  const cfg = T.makeLocalOpenAIConfig({ SKILL_PI_LOCAL_BASE_URL: "http://localhost:8000/v1" });
  const models = await cfg.refreshModels();
  assert.match(hit, /localhost:8000\/v1\/models/);
  assert.equal(models[0].id, "qwen2.5");
  T.clearFetcher();
});

test("mapModelsList ignores entries with no id and defaults cost", () => {
  const out = T.mapModelsList({ data: [{ id: "a" }, { name: "no-id" }, { id: "b" }] }, {});
  assert.deepEqual(out.map((m) => m.id), ["a", "b"]);
  assert.deepEqual(out[0].cost, T.ZERO_COST);
});
