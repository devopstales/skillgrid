
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

### Agent Orchectration

* [ ] [opencode-4hol](https://dev.to/uenyioha/porting-claude-codes-agent-teams-to-opencode-4hol)
* [ ] [Cleave](https://cleave.dev/)

### Memory

* [?] [CocoIndex-Code](https://github.com/cocoindex-io/cocoindex-code)
* [?] [GitNexus](https://github.com/abhigyanpatwari/GitNexus)
* [X] [codegraph](https://github.com/colbymchenry/codegraph)
* [?] [graphify]()
* [?] [deepseek-harness](https://github.com/deepseek-ai/deepseek-harness)
* [X] [Engram](https://github.com/Gentleman-Programming/engram)

### Spec driven development

* PRD
* [-] [openspec](https://openspec.dev/)
* [X] [Superpowers](https://github.com/obra/superpowers)

### Ticketingt

* Atlassian MCP
* GitHub MCP
* Gitlab MCP
* Vercel
* backlog.md

### checkoint

* Superpowers - uses commits

### Design Tools

* design.md
* [ ] [taste-skill](https://github.com/Leonxlnx/taste-skill)
* [ ] [npxskillui](https://github.com/amaancoderx/npxskillui)
* [ ] [impeccable](https://github.com/pbakaus/impeccable)
* [ ] [open-design](https://github.com/nexu-io/open-design)
* [ ] [kombai](https://kombai.com/)
* [ ] [penpot](https://penpot.app/self-host#options)
* [ ] [penpot-desktop](https://github.com/author-more/penpot-desktop/wiki/Installation)

### Usage Data

* [ ] [context-mode](https://github.com/mksglu/context-mode)
* [ ] [gryph](https://github.com/safedep/gryph)


### Testing

* [X] playwright
* [X] agent-browser
* [ ] [cucumber](https://cucumber.io/docs/cucumber/)
* [ ] [gherkin](https://cucumber.io/docs/gherkin/reference)

### Security

* [ ] Trivy
* [ ] [secure-rules](https://github.com/TikiTribe/claude-secure-coding-rules/tree/main)

## Plugins

```
npm install superpowers@git+https://github.com/obra/superpowers.git --prefix "$HOME/.config/kilo"

npm install superpowers@git+https://github.com/obra/superpowers.git --prefix "$HOME/.config/opencode"

{
  "plugin": ["~/.config/opencode/node_modules/superpowers"]
}
```

* `https://github.com/kdcokenny/opencode-background-agents`

## Skills

* [X] skills

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

Copy `~/.skillgrid/config.d/AGENTS.md` to `~/.agents/AGENTS.md` and add it to the configs:

* `~/.config/kilo/kilo.jsonc`
* `~/.config/opencode/opencode.json`
* `https://github.com/obra/superpowers/blob/main/CLAUDE.md`
* `https://github.com/multica-ai/andrej-karpathy-skills/blob/main/CLAUDE.md`

## Installers

* bash script
  * use prebuilt binary
* brew
* nix flake