// HTTP client helper for Mnemonic's REST API.
// Used by the OpenCode/Kilo plugins to call the local mnemonic server.

export interface MnemonicClientOptions {
  baseURL: string;
  token?: string;
  fetch?: typeof fetch;
}

export class MnemonicClient {
  private baseURL: string;
  private token?: string;
  private fetchFn: typeof fetch;

  constructor(opts: MnemonicClientOptions) {
    this.baseURL = opts.baseURL.replace(/\/$/, "");
    this.token = opts.token;
    this.fetchFn = opts.fetch || globalThis.fetch;
  }

  private headers(): Record<string, string> {
    const h: Record<string, string> = { "Content-Type": "application/json" };
    if (this.token) h["Authorization"] = "Bearer " + this.token;
    return h;
  }

  async health(): Promise<{ status: string; version: string }> {
    const r = await this.fetchFn(this.baseURL + "/health", { headers: this.headers() });
    return r.json();
  }

  async sessionStart(opts: { directory?: string; title?: string } = {}): Promise<{ session_id: string; project_id: string }> {
    const q = opts.directory
      ? "?directory=" + encodeURIComponent(opts.directory) +
        (opts.title ? "&title=" + encodeURIComponent(opts.title) : "")
      : opts.title
        ? "?title=" + encodeURIComponent(opts.title)
        : "";
    const r = await this.fetchFn(this.baseURL + "/sessions" + q, {
      method: "POST",
      headers: this.headers(),
    });
    return r.json();
  }

  async searchObservations(query: string, matchMode = "any", limit = 20): Promise<{ observations: any[] }> {
    const params = new URLSearchParams({ query, match_mode: matchMode, limit: String(limit) });
    const r = await this.fetchFn(this.baseURL + "/search?" + params, { headers: this.headers() });
    return r.json();
  }

  async codeSearch(query: string, limit = 20): Promise<{ hits: any[] }> {
    const params = new URLSearchParams({ query, limit: String(limit) });
    const r = await this.fetchFn(this.baseURL + "/code/search?" + params, { headers: this.headers() });
    return r.json();
  }
}

export default MnemonicClient;
