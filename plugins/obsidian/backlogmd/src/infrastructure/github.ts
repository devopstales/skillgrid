export interface GhIssue {
  number: number;
  title: string;
  state: string;
  /** Real GitHub API shape: a user object, or null when unassigned. */
  assignee?: { login: string } | null;
  body?: string;
}

const PAGE_SIZE = 100;
const MAX_PAGES = 10; // 1000 issues cap
const TIMEOUT_MS = 30000;

/** Fetch all issues (open + closed) of a repo, paging while a full page
 *  comes back (cap 10 pages / 1000 issues), with a 30s timeout per page.
 *  Pull requests mixed into the /issues response are dropped. */
export async function fetchIssues(token: string, owner: string, repo: string): Promise<GhIssue[]> {
  const issues: GhIssue[] = [];
  for (let page = 1; page <= MAX_PAGES; page++) {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), TIMEOUT_MS);
    let res: Response;
    try {
      res = await fetch(`https://api.github.com/repos/${owner}/${repo}/issues?state=all&per_page=${PAGE_SIZE}&page=${page}`, {
        headers: { Authorization: `token ${token}`, Accept: "application/vnd.github+json" },
        signal: controller.signal,
      });
    } finally {
      clearTimeout(timer);
    }
    if (!res.ok) throw new Error(`GitHub ${res.status}`);
    const batch = (await res.json()) as (GhIssue & { pull_request?: unknown })[];
    issues.push(...batch.filter((i) => i.pull_request === undefined));
    if (batch.length < PAGE_SIZE) break; // last page
  }
  return issues;
}
