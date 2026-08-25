
# Notes

## Agent Configs

* [ ] kilo
  * `~/.kiro/settings/mcp.json`
  * `~/.config/kilo/kilo.jsonc`
* [ ] opencode
  * `~/.config/opencode/opencode.json`
* cursor
* claud
* codex
* gemini cli
* antigravity

## Tools

* [ ] skills
* [ ] playwright
* [ ] agent-browser
* [ ] engram
* [ ] [gryph](https://github.com/safedep/gryph)
* graphify ??
* gitnexus ??
* ccc - coconaut code index ???
* [ ] [cucumber]()

## Plugins

```
npm install superpowers@git+https://github.com/obra/superpowers.git --prefix "$HOME/.config/kilo"

npm install superpowers@git+https://github.com/obra/superpowers.git --prefix "$HOME/.config/opencode"

{
  "plugin": ["~/.config/opencode/node_modules/superpowers"]
}
```

## Skills

```
skills add obra/superpowers --agent amp -g -s '*' -y
skills add gentleman-programming/engram --agent amp -g -s 'engram-memory' -y
skills add gentleman-programming/engram --agent amp -g -s 'engram-memory-protocol' -y
skills add gentleman-programming/engram --agent amp -g -s 'engram-testing-coverag' -y
```

* SDD - Spec-Driven Development
* TDD - Test-Driven Development
* DDD - [Behaviour-Driven Development](https://cucumber.io/docs/bdd)

## Rules

Copy `~/.aiskillgrid/config.d/AGENTS.md` to `~/.agents/AGENTS.md` and add it to the configs:

* `~/.config/kilo/kilo.jsonc`
* `~/.config/opencode/opencode.json`
