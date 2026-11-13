export interface GhIssue {
  number: number;
  title: string;
  state: string;
  assignee?: string | null;
  body?: string;
}

export async function fetchIssues(token: string, owner: string, repo: string): Promise<GhIssue[]> {
  const res = await fetch(`https://api.github.com/repos/${owner}/${repo}/issues?state=all&per_page=100`, {
    headers: { Authorization: `token ${token}`, Accept: "application/vnd.github+json" },
  });
  if (!res.ok) throw new Error(`GitHub ${res.status}`);
  // the /issues endpoint returns pull requests too — drop them
  const all = (await res.json()) as (GhIssue & { pull_request?: unknown })[];
  return all.filter((i) => i.pull_request === undefined);
}