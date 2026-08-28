# Ticketing

## References

* https://github.com/fselich/dossier
* https://www.rushis.com/mapping-openspec-to-jira-sdd-without-abandoning-your-backlog/

## Mapping openspec SDD to GitHub

### Strategy A — Change = Issue, Tasks = Checklist

Map each OpenSpec change directly to a GitHub Issue. Tasks inside `tasks.md` become checklist items in the issue body. This is the simplest mapping and works well for small teams already using GitHub Issues.

```
openspec/specs/auth/spec.md   ──► (reference only, no issue)
  changes/add-2fa/            ──► Issue: "Add 2FA"
    tasks.md items            ──►   Checklist items (- [ ] 1-1, - [ ] 1-2, ...)
  changes/fix-session-expiry/ ──► Issue: "Fix session expiry"
    tasks.md items            ──►   Checklist items
```

### When to use

Small to medium teams using GitHub Issues for tracking. Works best when each change is a single deliverable that can be completed in a few days.

### Problems

* No native hierarchy — GitHub Issues have no parent/child linking (only "linked issues" via keywords).
* Checklist progress is manual — agents must edit the issue body to check items; no structured task state.
* Cross-repo changes require manual issue duplication or references.

### Mitigation

Use GitHub Projects (v2) to group changes by domain. A Project view can roll up checklist progress across issues. Use `gh issue edit` with a script to sync task state from `tasks.md` checkboxes.

### Strategy B — Domain = Tracking Issue, Change = Issue

Create a long-lived "tracking" issue per domain/capability, and link each change issue to it via `gh issue edit --add-label` or closing keywords.

```
openspec/specs/auth/spec.md   ──► Tracking Issue: "Auth capability"
  changes/add-2fa/            ──► Issue: "Add 2FA" (linked: closes #12)
    tasks.md items            ──►   Checklist items
  changes/fix-session-expiry/ ──► Issue: "Fix session expiry" (linked: closes #12)
    tasks.md items            ──►   Checklist items
```

### When to use

Teams that want visibility into which capabilities have active work without adopting a full project board.

### Problems

* Tracking issues never close — they accumulate linked changes indefinitely.
* No burndown or velocity metrics at the domain level.

### Mitigation

Use a GitHub Project with a "Status" field per domain, and automate issue creation with `gh issue create` from `openspec list --json`.

### gh CLI examples for issue management

```bash
# Create an issue from an OpenSpec change
CHANGE_NAME="add-2fa"
TITLE="Add 2FA"
BODY=$(cat <<'EOF'
## Goal
Implement two-factor authentication for user accounts.

## Tasks
- [ ] 1-1 Create internal/auth/ service
- [ ] 1-2 Implement TOTP generation
- [ ] 1-3 Add QR code enrollment UI
- [ ] 1-4 Write acceptance tests

Spec: openspec/changes/add-2fa/specs/auth/spec.md
EOF
)
gh issue create --title "$TITLE" --body "$BODY" --label "openspec,auth"

# Sync task state from tasks.md to issue checklist
ISSUE_NUMBER=42
TASKS_MD="openspec/changes/add-2fa/tasks.md"
# Extract checked/unchecked tasks and update issue body
CHECKED=$(grep '^\- \[x\]' "$TASKS_MD" | wc -l)
TOTAL=$(grep '^\- \[[ x]\]' "$TASKS_MD" | wc -l)
gh issue edit "$ISSUE_NUMBER" --body "$(gh issue view "$ISSUE_NUMBER" --json body -q '.body' | sed "s/Progress: .*/Progress: $CHECKED\/TOTAL/")"

# List all openspec-tracked issues
gh issue list --label "openspec" --state all --json number,title,state,labels

# Close an issue when change is archived
ISSUE_NUMBER=42
gh issue close "$ISSUE_NUMBER" --comment "Archived via openspec archive add-2fa"

# Create a tracking issue for a domain
gh issue create --title "Auth capability" --body "Tracking issue for authentication domain changes." --label "domain,auth,tracking"

# Bulk-create issues from all active changes
openspec list --json | jq -r '.changes[] | select(.status == "in-progress") | .name' | while read name; do
  gh issue create --title "$name" --label "openspec" --body "Change: $name"
done
```

## Mapping openspec SDD to Jira

### Strategy B — Domain = Epic, Change = Story

Shift everything up one level. The Epic represents the long-lived *domain* or *capability* (matching `openspec/specs/<domain>/`), and each change folder maps to a Story under that Epic.

```
openspec/specs/auth/spec.md   ──► Epic: "Authentication"
  changes/add-2fa/            ──►   Story: "Add 2FA"
    tasks.md items            ──►     Sub-tasks
  changes/fix-session-expiry/ ──►   Story: "Fix session expiry"
    tasks.md items            ──►     Sub-tasks
```

### When to use

Product-led teams where “Authentication,” “Payments,” “Notifications” are long-running investment areas with a continuous stream of small changes.

### Problems

* Epics never close — they’re effectively themes or components. This breaks Jira’s Epic Burndown report, which assumes Epics have a lifecycle.
* Cross-domain changes (a change that touches `auth/` *and* `payments/`) need to live under two Epics, which Jira does not support cleanly for a single Story.

### Mitigation

Use Jira Components or Labels for the domain, and let Epics remain genuine initiatives. This effectively moves you toward Strategy D.
