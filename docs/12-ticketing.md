# Ticketing

## References

* https://github.com/fselich/dossier
* https://www.rushis.com/mapping-openspec-to-jira-sdd-without-abandoning-your-backlog/

## Mapping openspec SDD to GitHub

### Strategy A — Change = Issue, Tasks = Checklist

Map each OpenSpec change directly to a GitHub Issue. Tasks inside `tasks.md` become checklist items in the issue body.

```
openspec/specs/auth/spec.md   ──► (reference only, no issue)
  changes/add-2fa/            ──► Issue: "Add 2FA"
    tasks.md items            ──►   Checklist items (- [ ] 1-1, - [ ] 1-2, ...)
  changes/fix-session-expiry/ ──► Issue: "Fix session expiry"
    tasks.md items            ──►   Checklist items
```

### When to use

Small to medium teams using GitHub Issues for tracking.

### Problems

* No native hierarchy — GitHub Issues have no parent/child linking.
* Checklist progress is manual — agents must edit the issue body to check items.
* Cross-repo changes require manual issue duplication or references.

### Mitigation

Use GitHub Projects (v2) to group changes by domain.

### Strategy B — Domain = Tracking Issue, Change = Issue

Create a long-lived "tracking" issue per domain/capability, and link each change issue to it.

```
openspec/specs/auth/spec.md   ──► Tracking Issue: "Auth capability"
  changes/add-2fa/            ──► Issue: "Add 2FA" (linked: closes #12)
    tasks.md items            ──►   Checklist items
  changes/fix-session-expiry/ ──► Issue: "Fix session expiry" (linked: closes #12)
    tasks.md items            ──►   Checklist items
```

### gh CLI examples

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

Spec: openspec/changes/add-2fa/specs/auth/spec.md
EOF
)
gh issue create --title "$TITLE" --body "$BODY" --label "openspec,auth"
```

## Mapping openspec SDD to Jira

### Strategy B — Domain = Epic, Change = Story

Shift everything up one level. The Epic represents the long-lived domain or capability.

```
openspec/specs/auth/spec.md   ──► Epic: "Authentication"
  changes/add-2fa/            ──►   Story: "Add 2FA"
    tasks.md items            ──►     Sub-tasks
  changes/fix-session-expiry/ ──►   Story: "Fix session expiry"
    tasks.md items            ──►     Sub-tasks
```
