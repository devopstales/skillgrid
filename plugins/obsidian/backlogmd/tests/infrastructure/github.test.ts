import { describe, it, expect, vi, afterEach } from "vitest";
import { fetchIssues, GhIssue } from "../../src/infrastructure/github";

afterEach(() => {
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

const issue = (n: number, extra: Partial<GhIssue> = {}): GhIssue & { pull_request?: unknown } => ({
  number: n,
  title: `Issue ${n}`,
  state: "open",
  ...extra,
});

function mockPages(pages: (GhIssue & { pull_request?: unknown })[][]) {
  const fetchMock = vi.fn(async (url: string) => ({
    ok: true,
    status: 200,
    json: async () => pages[Number(new URL(url).searchParams.get("page") ?? "1") - 1],
  }));
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

describe("fetchIssues", () => {
  it("returns issues from a single page when fewer than 100 come back", async () => {
    const fetchMock = mockPages([[issue(1), issue(2), issue(3, { pull_request: {} })]]);
    const issues = await fetchIssues("tok", "owner", "repo");
    expect(issues.map((i) => i.number)).toEqual([1, 2]); // PR dropped
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("pages while full pages (100) come back, stopping on a short page", async () => {
    const page1 = Array.from({ length: 100 }, (_, i) => issue(i + 1));
    const page2 = [issue(101), issue(102)];
    const fetchMock = mockPages([page1, page2]);
    const issues = await fetchIssues("tok", "owner", "repo");
    expect(issues).toHaveLength(102);
    expect(issues.map((i) => i.number)).toEqual([...page1.map((i) => i.number), 101, 102]);
    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(String(fetchMock.mock.calls[0][0])).toContain("page=1");
    expect(String(fetchMock.mock.calls[1][0])).toContain("page=2");
  });

  it("caps at 10 pages (1000 issues) even if every page is full", async () => {
    const fetchMock = mockPages(Array.from({ length: 12 }, () => Array.from({ length: 100 }, (_, i) => issue(i + 1))));
    const issues = await fetchIssues("tok", "owner", "repo");
    expect(issues).toHaveLength(1000);
    expect(fetchMock).toHaveBeenCalledTimes(10);
  });

  it("throws on a non-ok response", async () => {
    vi.stubGlobal("fetch", async () => ({ ok: false, status: 403, json: async () => [] }));
    await expect(fetchIssues("bad", "owner", "repo")).rejects.toThrow("GitHub 403");
  });

  it("aborts a page fetch after the 30s timeout", async () => {
    vi.useFakeTimers();
    vi.stubGlobal(
      "fetch",
      (_url: unknown, init: RequestInit) =>
        new Promise((_resolve, reject) => {
          init.signal?.addEventListener("abort", () => reject(new Error("aborted")));
        }),
    );
    const pending = fetchIssues("tok", "owner", "repo");
    const assertion = expect(pending).rejects.toThrow("aborted");
    await vi.advanceTimersByTimeAsync(30000);
    await assertion;
  });
});
