// Mnemonic OpenCode plugin — installs MCP-backed memory tools.
// This file is copied into ~/.config/opencode/plugins/ by `skillgrid setup opencode`.

export const MnemonicPlugin = {
  name: "skillgrid-mnemonic",
  version: "0.1.0",
  description: "Local-first persistent memory (mem_*, code_*, web_* tools)",

  // The MCP server is registered separately via opencode.json mcp config.
  // This plugin provides UI hooks and HTTP client helpers.
};

export default MnemonicPlugin;
