# Skills

Skills are the unit of behavior agents can load per task (brainstorming, TDD, systematic debugging, etc.). skillgrid installs them from `config.d/skills.yaml` (schema in [03-config-reference](03-config-reference.md)).

They are distinct from plugins — a skill is behavior, a plugin is a bundle of skills/hooks that gets *registered* with the agent. See [07-plugins](07-plugins.md) for the superpowers and engram plugin mechanics.

## Install mechanics

The CLI invokes the local `skills` CLI — not `npx` at runtime, because that would pull a version that may drift:

```bash
~/.skillgrid/npm/node_modules/.bin/skills add <repo> --agent <agent> -g -s <skill> -y
```

Installed by the default `skills.yaml`:

| Skill | Source | Target |
|-------|--------|--------|
| Superpowers skill set | `obra/superpowers` | all (`*`) |
| engram-memory | `gentleman-programming/engram` | `amp` (default) |
| engram-memory-protocol | `gentleman-programming/engram` | `amp` |
| engram-testing-coverage | `gentleman-programming/engram` | `amp` |

A skill add failure **warns and continues** — the install does not abort.

## Uninstall / Update

skillgrid is idempotent but does not yet track "skills I installed" for removal. To update to the latest skill, bump the entry in `config.d/skills.yaml` and re-run `install` — the `skills` CLI handles add/replace semantics for its own registry.
