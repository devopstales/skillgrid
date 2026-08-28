---
name: github-issues
description: 'Create, update, and manage GitHub issues from OpenSpec changes using Strategy B mapping (Domain = Tracking Issue tagged Epic, Change = Issue tagged Task, delta spec = Issue tagged Sub-Task), wired together with native GitHub sub-issues. Use when users want to create GitHub issues from OpenSpec changes, create sub-tasks from delta specs, link issues as native sub-issues, sync task checklists, close issues on archive, or manage issue workflows via gh CLI. Triggers on requests like "create issue from change", "create sub-task from delta spec", "add sub-issue", "sync tasks to github", "track change as issue", "close issue on archive", or any OpenSpec-to-GitHub issue management task.'
---

# Skillgrid GitHub Issues

Map OpenSpec changes to GitHub issues using Strategy B: each domain gets a tracking issue tagged `Epic`, each change becomes an issue tagged `Task`, and each delta spec inside a change becomes an issue tagged `Sub-Task`. The three levels are wired together with **native GitHub sub-issues**, so GitHub renders the real hierarchy and rolls up progress automatically.

## Mapping

```
openspec/specs/auth/spec.md   ──► Tracking Issue: "Auth capability"          [Epic]
  changes/add-2fa/            ──► Sub-issue of #12: "Add 2FA"                [Task]
    specs/auth/spec.md        ──►   Sub-issue of #42: "add-2fa — auth"       [Sub-Task]
    tasks.md items            ──►   Checklist items (- [ ] 1-1, - [ ] 1-2, ...)
  changes/fix-session-expiry/ ──► Sub-issue of #12: "Fix session expiry"     [Task]
    specs/auth/spec.md        ──►   Sub-issue of #43: "fix-session-expiry …" [Sub-Task]
    tasks.md items            ──►   Checklist items
```

Two things are always applied together:

- **Hierarchy tag**: `Epic` on tracking issues, `Task` on change issues, `Sub-Task` on delta spec issues
- **Native sub-issue link**: `Task` is a sub-issue of its `Epic`, `Sub-Task` is a sub-issue of its `Task`

## Workflow

1. **Ensure labels exist**: Create required labels — including the `Epic`, `Task`, and `Sub-Task` hierarchy tags — before creating any issues
2. **Ensure tracking issue exists**: Create or find the domain tracking issue (tagged `Epic`)
3. **Create change issue**: Create issue from change (tagged `Task`) as a native sub-issue of the tracking issue
4. **Create sub-task issues**: One issue per delta spec `specs/<capability>/spec.md` (tagged `Sub-Task`) as a native sub-issue of the change issue
5. **Structure content**: Build issue body with Goal, Tasks checklist, Spec reference, tracking link
6. **Sync tasks**: Use `gh issue edit` to keep checklists in sync
7. **Close on archive**: Close the change issue and its sub-task issues when archived, leave tracking issue open

## Prerequisites — Labels

Always create labels before creating issues. The `openspec` label is mandatory. Domain labels match `openspec/specs/<domain>/` paths. The `Epic`, `Task`, and `Sub-Task` labels encode the issue hierarchy.

```bash
# Create openspec label
gh label create openspec --description "OpenSpec change" --color "0E8A16"

# Create domain label (example: auth)
gh label create auth --description "Authentication domain" --color "1D76DB"

# Create hierarchy labels
gh label create Epic --description "Domain tracking issue" --color "6F42C1"
gh label create Task --description "Issue created from an OpenSpec change" --color "C2E0C6"
gh label create Sub-Task --description "Issue created from an OpenSpec delta spec" --color "F9D0C4"
```

## Creating Tracking Issues

Create a long-lived tracking issue per domain/capability, tagged `Epic`. These never close — they accumulate linked changes indefinitely.

```bash
# Create tracking issue for a domain
DOMAIN="auth"
TITLE="Auth capability"
BODY=$(cat <<'EOF'
Tracking issue for the authentication domain.

## Active Changes
- Closes #<issue-number> — Change name

## Archived Changes
- None yet
EOF
)

gh issue create --title "$TITLE" --body "$BODY" --label "openspec,$DOMAIN,tracking,Epic"
```

Save the issue number returned for linking change issues.

## Native Sub-Issues

GitHub sub-issues are a real parent/child relationship (not a text convention): the parent shows a sub-issue list with a progress bar, and `sub_issues_summary` exposes the rollup via API. Use them for both hierarchy edges — `Task` under `Epic`, and `Sub-Task` under `Task`.

```bash
# Create an issue directly as a sub-issue of a parent
gh issue create --title "Add 2FA" --body-file body.md --label "openspec,auth,Task" --parent 12

# Attach existing issues to a parent (comma-separated numbers or URLs)
gh issue edit 12 --add-sub-issue 42,43

# Or set the parent from the child side
gh issue edit 42 --parent 12

# Detach
gh issue edit 12 --remove-sub-issue 42
gh issue edit 42 --remove-parent
```

Inspect the hierarchy and rollup progress:

```bash
REPO="owner/repo"

# Progress rollup on a parent: {"total":8,"completed":3,"percent_completed":37}
gh api "repos/$REPO/issues/12" --jq '.sub_issues_summary'

# List a parent's sub-issues
gh api "repos/$REPO/issues/12/sub_issues" --jq '.[] | "\(.number)\t\(.state)\t\(.title)"'

# Get a child's parent
gh api "repos/$REPO/issues/42/parent" --jq '"#\(.number) \(.title)"'
```

### REST Fallback

If the `gh issue` flags are unavailable (verified working on gh 2.98.0), use the REST API. Note the payload takes the sub-issue's **database `id`**, not its issue number:

```bash
REPO="owner/repo"
PARENT=12
CHILD=42

SUB_ID=$(gh api "repos/$REPO/issues/$CHILD" --jq '.id')
gh api --method POST "repos/$REPO/issues/$PARENT/sub_issues" -F sub_issue_id="$SUB_ID"

# Remove, and reprioritize within the parent's list
gh api --method DELETE "repos/$REPO/issues/$PARENT/sub_issue" -F sub_issue_id="$SUB_ID"
gh api --method PATCH "repos/$REPO/issues/$PARENT/sub_issues/priority" -F sub_issue_id="$SUB_ID" -F after_id=<other-sub-issue-id>
```

Use `-F` (typed) not `-f` (string) so `sub_issue_id` is sent as an integer.

### Limits

- Max **100 sub-issues** per parent issue
- Max **8 levels** of nesting (Epic → Task → Sub-Task uses 3)
- A sub-issue must belong to the same repository owner as its parent
- An issue has at most one parent — pass `replace_parent: true` (REST) to move it
- Rapid creation/removal can trip secondary rate limits; add a small sleep in bulk loops

## Creating Issues from Changes

Use `gh issue create` with structured body from the OpenSpec change, tagged `Task`, created directly as a native sub-issue of the tracking issue via `--parent`.

```bash
CHANGE_NAME="add-2fa"
CHANGE_DIR="openspec/changes/$CHANGE_NAME"
DOMAIN="auth"
TRACKING_ISSUE=12

TITLE=$(cat "$CHANGE_DIR/proposal.md" | head -1 | sed 's/^## //')
BODY=$(cat <<EOF
## Goal
$(sed -n '/^## Why$/,/^## What Changes/p' "$CHANGE_DIR/proposal.md" | sed '1d;$d')

## Tasks
$(grep '^\- \[[ x]\]' "$CHANGE_DIR/tasks.md")

## Spec
$CHANGE_DIR/specs/**/spec.md

## Sub-Tasks
- Part of #<sub-task-issue-number> — capability name

## Tracking
Closes #$TRACKING_ISSUE

---
*Tracked by OpenSpec change: $CHANGE_NAME*
EOF
)

gh issue create --title "$TITLE" --body "$BODY" --label "openspec,$DOMAIN,Task" --parent "$TRACKING_ISSUE"
```

### Body Template

Every issue created from an OpenSpec change uses this structure:

```markdown
## Goal
[One sentence from proposal.md Why section]

## Tasks
- [ ] 1-1 Task description
- [ ] 1-2 Task description
- [ ] 2-1 Task description

## Spec
Path to spec file(s)

## Sub-Tasks
- Part of #<sub-task-issue-number> — capability name

## Tracking
Closes #<tracking-issue-number>

---
*Tracked by OpenSpec change: <change-name>*
```

The `## Sub-Tasks` section sits after `## Spec` so task syncing (which rewrites only the block between `## Tasks` and `## Spec`) never clobbers sub-task links.

## Creating Sub-Task Issues from Delta Specs

Each delta spec — `openspec/changes/<change>/specs/<capability>/spec.md` — becomes one issue tagged `Sub-Task`, created as a native sub-issue of its parent change issue via `--parent`. Delta specs contain the `## ADDED Requirements` / `## MODIFIED Requirements` / `## REMOVED Requirements` operations, and each `### Requirement:` heading becomes a checklist item.

```bash
CHANGE_NAME="add-2fa"
CHANGE_DIR="openspec/changes/$CHANGE_NAME"
DOMAIN="auth"
TRACKING_ISSUE=12
CHANGE_ISSUE=42

for SPEC in "$CHANGE_DIR"/specs/*/spec.md; do
  CAPABILITY=$(basename "$(dirname "$SPEC")")
  BODY=$(cat <<EOF
## Goal
Deliver the \`$CAPABILITY\` capability delta for the \`$CHANGE_NAME\` change.

## Requirements
$(grep '^### Requirement:' "$SPEC" | sed 's/^### Requirement: /- [ ] /')

## Delta Operations
$(grep -E '^## (ADDED|MODIFIED|REMOVED|RENAMED) Requirements' "$SPEC" | sed 's/^## /- /')

## Spec
$SPEC

## Tracking
Part of #$CHANGE_ISSUE
Epic: #$TRACKING_ISSUE

---
*Tracked by OpenSpec delta spec: $CHANGE_NAME/$CAPABILITY*
EOF
)

  gh issue create --title "$CHANGE_NAME — $CAPABILITY" --body "$BODY" \
    --label "openspec,$DOMAIN,Sub-Task" --parent "$CHANGE_ISSUE"
  sleep 1  # avoid secondary rate limits in bulk loops
done

# Verify the rollup on the parent change issue
gh api "repos/$(gh repo view --json nameWithOwner -q .nameWithOwner)/issues/$CHANGE_ISSUE" \
  --jq '.sub_issues_summary'
```

### Sub-Task Body Template

```markdown
## Goal
Deliver the `<capability>` capability delta for the `<change-name>` change.

## Requirements
- [ ] Requirement name from `### Requirement:` heading
- [ ] Requirement name from `### Requirement:` heading

## Delta Operations
- ADDED Requirements

## Spec
openspec/changes/<change-name>/specs/<capability>/spec.md

## Tracking
Part of #<change-issue-number>
Epic: #<tracking-issue-number>

---
*Tracked by OpenSpec delta spec: <change-name>/<capability>*
```

Sub-task issues use `Part of #<change-issue>` rather than `Closes` — only the change issue closes its tracking `Epic` reference. After creating sub-tasks, back-link them in the parent change issue's `## Sub-Tasks` section.

## Syncing Task State

Sync checkbox state from `tasks.md` to the GitHub issue body, replacing only the block between the `## Tasks` and `## Spec` markers.

```bash
ISSUE_NUMBER=42
CHANGE_NAME="add-2fa"
CHANGE_DIR="openspec/changes/$CHANGE_NAME"
BODY_FILE=$(mktemp)

# Build task checklist from tasks.md (verbatim — preserves 1-1 numbering and [x] state)
TASKS=$(grep '^- \[[ x]\] ' "$CHANGE_DIR/tasks.md")

# Fetch current body, replace only the Tasks section
gh issue view "$ISSUE_NUMBER" --json body -q '.body' > "$BODY_FILE"
python3 - "$BODY_FILE" "$TASKS" <<'PY'
import sys
path, tasks = sys.argv[1], sys.argv[2]
body = open(path).read()
start = body.index("## Tasks")
end = body.index("## Spec")
new = body[:start] + "## Tasks\n" + tasks.rstrip() + "\n\n" + body[end:]
open(path, "w").write(new.rstrip() + "\n")   # normalize tail: see note below
PY

gh issue edit "$ISSUE_NUMBER" --body-file "$BODY_FILE"
```

Do **not** try this with `sed "/## Tasks/,/## Spec/c\\..."`. Task lines begin with `-` and contain backticks, brackets, and slashes, so `sed` aborts with `unknown command: '-'` and can blank the issue body. Marker-based slicing in `python3` is safe and idempotent.

Always write `new.rstrip() + "\n"`. GitHub appends a trailing newline to every issue body it stores, so a sync that preserves the fetched tail verbatim grows the body by one blank line on every run (83 → 84 → 85 …). Normalizing the tail keeps repeated syncs byte-stable.


## Closing Issues on Archive

When `openspec archive` moves a change to archive, close the linked change issue and all its sub-task issues. Do NOT close the tracking issue.

```bash
ISSUE_NUMBER=42
CHANGE_NAME="add-2fa"
TRACKING_ISSUE=12
REPO=$(gh repo view --json nameWithOwner -q .nameWithOwner)

# Close sub-task issues, enumerated from the native sub-issue list
for SUB in $(gh api "repos/$REPO/issues/$ISSUE_NUMBER/sub_issues" --jq '.[] | select(.state=="open") | .number'); do
  gh issue close "$SUB" \
    --comment "Archived via \`openspec archive $CHANGE_NAME\`. Parent change issue: #$ISSUE_NUMBER." \
    --reason "completed"
done

gh issue close "$ISSUE_NUMBER" \
  --comment "Archived via \`openspec archive $CHANGE_NAME\`. Change moved to openspec/changes/archive/." \
  --reason "completed"

# Move the change from Active Changes to Archived Changes on the Epic
TRACK_FILE=$(mktemp)
gh issue view "$TRACKING_ISSUE" --json body -q '.body' > "$TRACK_FILE"
python3 - "$TRACK_FILE" "$ISSUE_NUMBER" "$CHANGE_NAME" <<'PY'
import sys
path, num, name = sys.argv[1], sys.argv[2], sys.argv[3]
lines = open(path).read().splitlines()
a = lines.index("## Active Changes")
z = lines.index("## Archived Changes")
end = z + 1
while end < len(lines) and not lines[end].startswith("---"):
    end += 1
active, archived = lines[a + 1 : z], lines[z + 1 : end]

moved = [l for l in active if l.startswith("- ") and f"#{num} " in l]
active = [l for l in active if l not in moved]
already = [l for l in archived if l.startswith("- ") and f"#{num} " in l]
if not moved and not already:          # entry missing from both sections
    moved = [f"- Closes #{num} — {name}"]
archived = [l for l in archived if l.strip() != "- None yet"]

def tidy(block, empty="- None"):
    block = [l for l in block if l.strip()]
    return (block if block else [empty]) + [""]

open(path, "w").write("\n".join(
    lines[: a + 1] + tidy(active) + ["## Archived Changes"] + tidy(archived + moved) + lines[end:]
).rstrip() + "\n")
PY
gh issue edit "$TRACKING_ISSUE" --body-file "$TRACK_FILE"
```

Enumerating children from `sub_issues` is more reliable than searching issue bodies, and the closed sub-issues immediately show in the parent's progress bar.

Do **not** use `sed 's/## Active Changes/## Archived Changes\n- Closes #N — name\n\n## Active Changes/'` for this step. It inserts a *second* `## Archived Changes` heading above the active list and leaves the entry in Active Changes instead of moving it. The `python3` version above actually moves the line, drops the `- None yet` placeholder, and is idempotent when re-run.


## Querying Issues

```bash
REPO=$(gh repo view --json nameWithOwner -q .nameWithOwner)

# List all openspec-tracked issues
gh issue list --label "openspec" --state all --json number,title,state,labels

# List open tracking issues for a specific domain
gh issue list --label "openspec,tracking" --state open

# List open change issues for a specific domain
gh issue list --label "openspec,auth" --state open

# List by hierarchy level
gh issue list --label "openspec,Epic" --state open
gh issue list --label "openspec,Task" --state open
gh issue list --label "openspec,Sub-Task" --state open

# Walk the native hierarchy: Epic -> Task -> Sub-Task
gh api "repos/$REPO/issues/12/sub_issues" --jq '.[] | "\(.number)\t\(.title)"'
gh api "repos/$REPO/issues/42/sub_issues" --jq '.[] | "\(.number)\t\(.title)"'
gh api "repos/$REPO/issues/42/parent" --jq '"#\(.number) \(.title)"'

# Progress rollup per Epic
for E in $(gh issue list --label "openspec,Epic" --state open --json number -q '.[].number'); do
  gh api "repos/$REPO/issues/$E" --jq '"#\(.number) \(.title): \(.sub_issues_summary.completed)/\(.sub_issues_summary.total) (\(.sub_issues_summary.percent_completed)%)"'
done

# Get tracking issue details with linked changes
gh issue view 12 --json number,title,body,state,labels,assignees,closedBy
```

## Labels

| Label | Use For |
|-------|---------|
| `openspec` | All issues created from OpenSpec changes |
| `Epic` | Domain tracking issues (hierarchy level 1) |
| `Task` | Issues created from an OpenSpec change (hierarchy level 2) |
| `Sub-Task` | Issues created from a delta spec `specs/<capability>/spec.md` (hierarchy level 3) |
| `tracking` | Long-lived domain tracking issues |
| `<domain>` | Domain tag matching `openspec/specs/<domain>/` (e.g., `auth`, `payments`) |
| `bug` | Change fixes a defect |
| `enhancement` | Change adds new capability |

Always create and apply the `openspec` label. Add the domain label from the change's spec path. Apply exactly one hierarchy tag per issue: `Epic` for domain tracking issues (alongside `tracking`), `Task` for change issues, `Sub-Task` for delta spec issues.


## Examples

### Example 1: Create Tracking Issue and Change Issue

**User**: "Create a GitHub tracking issue for auth and an issue from add-2fa"

**Action**:
```bash
# Step 1: Ensure labels exist
gh label create openspec --description "OpenSpec change" --color "0E8A16"
gh label create auth --description "Authentication domain" --color "1D76DB"
gh label create tracking --description "Domain tracking issue" --color "FF7F00"
gh label create Epic --description "Domain tracking issue" --color "6F42C1"
gh label create Task --description "Issue created from an OpenSpec change" --color "C2E0C6"
gh label create Sub-Task --description "Issue created from an OpenSpec delta spec" --color "F9D0C4"

# Step 2: Create tracking issue for auth domain (Epic)
TRACKING_BODY=$(cat <<'EOF'
Tracking issue for the authentication domain.

## Active Changes
- Closes #<issue-number> — Change name

## Archived Changes
- None yet
EOF
)
TRACKING_ISSUE=$(gh issue create --title "Auth capability" --body "$TRACKING_BODY" --label "openspec,auth,tracking,Epic" --json number -q '.number')

# Step 3: Create change issue linked to tracking issue (Task)
CHANGE_DIR="openspec/changes/add-2fa"
BODY=$(cat <<'EOF'
## Goal
Implement two-factor authentication for user accounts.

## Tasks
- [ ] 1-1 Create internal/auth/ service
- [ ] 1-2 Implement TOTP generation
- [ ] 1-3 Add QR code enrollment UI
- [ ] 1-4 Write acceptance tests

## Spec
openspec/changes/add-2fa/specs/auth/spec.md

## Sub-Tasks
- Part of #<sub-task-issue-number> — auth

## Tracking
Closes #<tracking-issue-number>

---
*Tracked by OpenSpec change: add-2fa*
EOF
)
# Replace placeholder with actual tracking issue number
BODY="${BODY/<tracking-issue-number>/$TRACKING_ISSUE}"

gh issue create --title "Add 2FA" --body "$BODY" --label "openspec,auth,Task" --parent "$TRACKING_ISSUE"
```

### Example 2: Sync Task Progress

**User**: "Sync the task checklist to GitHub issue #42"

**Action**:
```bash
ISSUE_NUMBER=42
CHANGE_DIR="openspec/changes/add-2fa"
BODY_FILE=$(mktemp)

TASKS=$(grep '^- \[[ x]\] ' "$CHANGE_DIR/tasks.md")
gh issue view "$ISSUE_NUMBER" --json body -q '.body' > "$BODY_FILE"
python3 - "$BODY_FILE" "$TASKS" <<'PY'
import sys
path, tasks = sys.argv[1], sys.argv[2]
body = open(path).read()
start = body.index("## Tasks")
end = body.index("## Spec")
new = body[:start] + "## Tasks\n" + tasks.rstrip() + "\n\n" + body[end:]
open(path, "w").write(new.rstrip() + "\n")
PY
gh issue edit "$ISSUE_NUMBER" --body-file "$BODY_FILE"
```

### Example 3: Close on Archive

**User**: "Archive the add-2fa change and close the issue"

**Action**:
```bash
# Get the issue number for add-2fa (assuming 42)
ISSUE_NUMBER=42
TRACKING_ISSUE=12
CHANGE_NAME="add-2fa"

# Close sub-task issues belonging to this change
REPO=$(gh repo view --json nameWithOwner -q .nameWithOwner)
for SUB in $(gh api "repos/$REPO/issues/$ISSUE_NUMBER/sub_issues" --jq '.[] | select(.state=="open") | .number'); do
  gh issue close "$SUB" --comment "Archived via \`openspec archive $CHANGE_NAME\`. Parent change issue: #$ISSUE_NUMBER." --reason "completed"
done

# Close the change issue
gh issue close "$ISSUE_NUMBER" --comment "Archived via \`openspec archive $CHANGE_NAME\`. Change moved to openspec/changes/archive/." --reason "completed"

# Move the change to the Archived Changes section (Epic stays open)
# scripts/archive-move.py is relative to this skill's directory
TRACK_FILE=$(mktemp)
gh issue view "$TRACKING_ISSUE" --json body -q '.body' > "$TRACK_FILE"
python3 scripts/archive-move.py "$TRACK_FILE" "$ISSUE_NUMBER" "$CHANGE_NAME"
gh issue edit "$TRACKING_ISSUE" --body-file "$TRACK_FILE"
```

`scripts/archive-move.py` holds the same section-move logic shown inline above.

### Example 4: Create Sub-Task Issues from Delta Specs

**User**: "Create sub-task issues for each delta spec in add-2fa"

**Action**:
```bash
CHANGE_NAME="add-2fa"
CHANGE_DIR="openspec/changes/$CHANGE_NAME"
DOMAIN="auth"
TRACKING_ISSUE=12
CHANGE_ISSUE=42

SUBTASK_LINKS=""
for SPEC in "$CHANGE_DIR"/specs/*/spec.md; do
  CAPABILITY=$(basename "$(dirname "$SPEC")")
  BODY=$(cat <<EOF
## Goal
Deliver the \`$CAPABILITY\` capability delta for the \`$CHANGE_NAME\` change.

## Requirements
$(grep '^### Requirement:' "$SPEC" | sed 's/^### Requirement: /- [ ] /')

## Delta Operations
$(grep -E '^## (ADDED|MODIFIED|REMOVED|RENAMED) Requirements' "$SPEC" | sed 's/^## /- /')

## Spec
$SPEC

## Tracking
Part of #$CHANGE_ISSUE
Epic: #$TRACKING_ISSUE

---
*Tracked by OpenSpec delta spec: $CHANGE_NAME/$CAPABILITY*
EOF
)
  URL=$(gh issue create --title "$CHANGE_NAME — $CAPABILITY" --body "$BODY" --label "openspec,$DOMAIN,Sub-Task" --parent "$CHANGE_ISSUE")
  SUBTASK_LINKS="$SUBTASK_LINKS- Part of #${URL##*/} — $CAPABILITY"$'\n'
  sleep 1
done

# Back-link sub-tasks in the parent change issue's Sub-Tasks section
gh issue view "$CHANGE_ISSUE" --json body -q '.body' > /tmp/change-body.md
python3 - /tmp/change-body.md "$SUBTASK_LINKS" <<'PY'
import sys
path, links = sys.argv[1], sys.argv[2]
body = open(path).read()
start = body.index("## Sub-Tasks")
end = body.index("## Tracking")
open(path, "w").write(body[:start] + "## Sub-Tasks\n" + links + "\n" + body[end:])
PY
gh issue edit "$CHANGE_ISSUE" --body-file /tmp/change-body.md

# Confirm the native hierarchy and rollup
gh api "repos/$(gh repo view --json nameWithOwner -q .nameWithOwner)/issues/$CHANGE_ISSUE" --jq '.sub_issues_summary'
```

### Example 5: Link Existing Issues into the Hierarchy

**User**: "Wire the existing change issues under their tracking issues as sub-issues"

**Action**:
```bash
# Epic #12 (auth) gets change issues #42 and #43 as native sub-issues
gh issue edit 12 --add-sub-issue 42,43

# Equivalent, from the child side
gh issue edit 42 --parent 12
gh issue edit 43 --parent 12

# Verify
REPO=$(gh repo view --json nameWithOwner -q .nameWithOwner)
gh api "repos/$REPO/issues/12/sub_issues" --jq '.[] | "\(.number)\t\(.title)"'
gh api "repos/$REPO/issues/12" --jq '.sub_issues_summary'
```

## Tips

- Always create required labels before creating issues
- Apply exactly one hierarchy tag per issue: `Epic` (tracking), `Task` (change), `Sub-Task` (delta spec)
- Prefer `gh issue create --parent <n>` so the native sub-issue link exists from creation — no second call
- Use `gh issue edit <parent> --add-sub-issue <n>` to retrofit existing issues into the hierarchy
- Hierarchy tags and native sub-issue links are complementary: labels make issues filterable, sub-issues give GitHub's tree view and progress rollup
- Read progress with `gh api repos/OWNER/REPO/issues/<n> --jq '.sub_issues_summary'`
- Respect the limits: 100 sub-issues per parent, 8 nesting levels, one parent per issue, same repository owner
- Add `sleep 1` in bulk sub-issue loops to avoid secondary rate limits
- Preserve checklist item numbering (e.g., `- [ ] 1-1`) so task references stay stable
- When syncing, replace only the Tasks section between `## Tasks` and `## Spec` markers
- Keep the `## Sub-Tasks` section after `## Spec` so task syncing never overwrites sub-task links
- Use `gh issue list --label "openspec,tracking"` to find all tracking issues
- Use `gh issue list --label "openspec,auth"` to find all change issues for a domain
- Use `gh issue list --label "openspec,Epic"` / `Task` / `Sub-Task` to walk one hierarchy level by label
- Enumerate children from `repos/OWNER/REPO/issues/<n>/sub_issues` rather than searching issue bodies
- `Closes #<n>` / `Part of #<n>` in an issue body is a human-readable trail only — it does not auto-close or auto-link; the native sub-issue relationship is the machine-readable link
- The `openspec` label is mandatory for discoverability
- Tracking issues (`Epic`) are long-lived and should not be closed
- Closing a change issue on archive also closes its `Sub-Task` issues


