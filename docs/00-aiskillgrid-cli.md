# aiskillgrid cli

It is a go based single binari to install and configire the ai agents with preselected tools.

## Install subcommand

1) Clone this git repo at home

```bash
mkdir ~/.aiskillgrid
cd ~/.aiskillgrid
mkdir ~/.aiskillgrid/repos
git clone -b release/2 https://github.com/devopstales/aiskillgrid.git repos/
cp -r repos/aiskillgrid/config.d .
```

2) check and install node

```bash
based on scripts/install_node.sh
```

3) install engram binary into `~/.aiskillgrid/bin`

```bash
ENGram_VERSION=$(curl -s https://api.github.com/repos/Gentleman-Programming/engram/releases/latest | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/')
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$OS-$ARCH" in
  darwin-arm64) PLATFORM="darwin_arm64" ;;
  darwin-x86_64) PLATFORM="darwin_amd64" ;;
  linux-arm64) PLATFORM="linux_arm64" ;;
  linux-x86_64) PLATFORM="linux_amd64" ;;
esac
curl -L "https://github.com/Gentleman-Programming/engram/releases/download/v${ENGram_VERSION}/engram_${ENGram_VERSION}_${PLATFORM}.tar.gz" -o /tmp/engram.tar.gz
tar -xzf /tmp/engram.tar.gz -C ~/.aiskillgrid/bin
chmod +x ~/.aiskillgrid/bin/engram
```

4) Install selected agents and tools into to `~/.aiskillgrid` based on `~/.aiskillgrid/config.d/tools.yaml`

```bash
npm install @kilocode/cli --prefix "$HOME/.aiskillgrid"
npm install opencode-ai --prefix "$HOME/.aiskillgrid"

npm install vercel-labs/skills --prefix "$HOME/.aiskillgrid"
npm install @playwright/cli@latest --prefix "$HOME/.aiskillgrid"
npm install @playwright/mcp@latest --prefix "$HOME/.aiskillgrid"
npm install agent-browser --prefix "$HOME/.aiskillgrid"
```

4) install plugins

```bash
npm install superpowers@git+https://github.com/obra/superpowers.git --prefix "$HOME/.config/kilo"

npm install superpowers@git+https://github.com/obra/superpowers.git --prefix "$HOME/.config/opencode"

nano ~/.config/kilo/kilo.jsonc
{
  "plugin": ["~/.config/kilo/node_modules/superpowers"]
}
nano ~/.config/opencode/opencode.json
{
  "plugin": ["~/.config/opencode/node_modules/superpowers"]
}
```

5) install skills based on config file `~/.aiskillgrid/config.d/skills.yaml`

```bash
npx skills add obra/superpowers --agent amp -g -s '*' -y
```

7) install mcp based on config file `~/.aiskillgrid/config.d/mcp.yaml`


7) print paths that the user shoud add to $PATH variable

### Optional selectorts

* `--skip-clone` - do not git clone repo
* `--sync-repo` - add path that will be synced into  `~/.aiskillgrid/repos/aiskillgrid`