# Ticketing

## References

* https://github.com/fselich/dossier
* https://www.rushis.com/mapping-openspec-to-jira-sdd-without-abandoning-your-backlog/

## Mapping openspec SDD to GitHub

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
