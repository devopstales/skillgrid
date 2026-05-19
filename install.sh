#!/bin/bash
# =============================================================================
# install.sh — AI IDE Config Installer
# =============================================================================
#
# PURPOSE:
#   Copies AI assistant configuration folders (Cursor, Copilot, Kilo, OpenCode,
#   Antigravity) from this hub repo into a target project directory.
#   Hub .agents/ (rules, workflows, skills) is the cross-IDE Skillgrid standard — synced
#   to the project root for every install when present. Hub .agent/ (e.g. agents/ prompts)
#   is for Google Antigravity (and future Gemini CLI integration) only — synced when that
#   IDE is selected, not for Cursor/Kilo/OpenCode/Copilot-only installs.
#   Syncs hub `.skillgrid/` (project docs + scripts) and merges MCP configs from
#   .configs/mcp/ into each product's native MCP shape (Cursor, VS Code servers,
#   Kilo/OpenCode mcp, Antigravity serverUrl).
#
# USAGE:
#   ./install.sh [OPTIONS]
#
#   Common workflows:
#     # Interactive — pick IDEs and MCP servers via prompts
#     ./install.sh -p /path/to/project
#
#     # Non-interactive — all IDEs, all MCP servers
#     ./install.sh -p /path/to/project -A -y
#
#     # All MCP servers without MCP prompt (subset filter cleared); combine with -y for no IDE prompt
#     ./install.sh -p /path/to/project -A -y --all-mcp
#     ./install.sh -p /path/to/project -AA -y
#
#     # Dry run — preview changes without writing
#     ./install.sh -p /path/to/project -n
#
#     # Check/install dependencies only
#     ./install.sh -d
#
#     # Read-only sanity check for hub dependencies and expected files
#     ./install.sh --sanity-check
#
#     # Uninstall managed folders from a project
#     ./install.sh -p /path/to/project -u
#
# ARCHITECTURE:
#   1. Argument parsing  →  populate SELECTED_IDES, flags, PROJECT_PATH
#   2. Interactive prompts → IDE selection, MCP server selection (if eligible)
#   3. Dependency check  → optional install of missing tools
#   4. MCP merge         → jq-merge .configs/mcp/**/*.json → canonical mcpServers
#   5. IDE setup         → per-IDE setup_*() maps merged MCP to native JSON schemas
#   6. Optional tools    → gitnexus + engram CLIs always; -t adds openspec/dmux/brave-search-cli/cocoindex-code; npm/uv/brew installs, then copy + openspec init when selected
#
# DEPENDENCIES:
#   Runtime:  bash 3.2+ (incl. macOS /bin/bash), rsync, jq
#   Optional: node, npx, npm, python3, pip3, uv (Python CLIs; hub Node deps via npm ci)
#   IDE CLIs: opencode, kilo, semgrep, trivy (+ trivy plugin mcp when hub trivy MCP is selected)
#
# VERSION: 1.0.0
# =============================================================================

set -e

VERSION="1.0.0"
PROJECT_PATH=""
SELECTED_IDES=()
SELECTED_TOOLS=()
TOOLS_INTERACTIVE=false
DRY_RUN=false
UNINSTALL=false
CHECK_DEPS=false
SANITY_CHECK=false
ALL_IDES=false
NON_INTERACTIVE=false
MERGE_MCP=true
ALL_MCP=false
MCP_KEY_FILTER_JSON=""
SYNC_ASSETS=false
INSTALL_SDD_HOOKS=true

# =============================================================================
# GLOBALS: Colors, IDE mappings, dependency declarations
# =============================================================================

# Color support — only emit ANSI codes when stdout is a terminal
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    CYAN='\033[0;36m'
    NC='\033[0m'
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    CYAN=''
    NC=''
fi

# IDE folder name mapping (case-based — bash 3.2 on macOS has no declare -A)
ide_folder_for() {
    case "$1" in
        cursor) echo ".cursor" ;;
        copilot) echo ".vscode" ;;
        kilo) echo ".kilo" ;;
        opencode) echo ".opencode" ;;
        antigravity) echo ".agents" ;;
        *) echo "" ;;
    esac
}

# Core dependencies — always checked regardless of IDE selection
# Format: "name:brew_package:pip_package:npm_package:check_command"
declare -a CORE_DEPENDENCIES=(
    # Core tools
    "rsync:rsync:::rsync --version"
    "jq:jq:::jq --version"
    "node:node:::node --version"
    "python3:python@3.12:::python3 --version"
    "pip3::::pip3 --version"
    "npx::::npx --version"
)

# IDE-specific dependencies — only checked when the corresponding IDE is selected
# Format: "name|brew_package|pip_package|npm_package|check_command|ide_flag"
declare -a IDE_DEPENDENCIES=(
    "opencode|opencode|||opencode --version|opencode"
    "openspec|openspec|||openspec --version|openspec"
    "kilo|Kilo-Org/homebrew-tap/kilo|||kilo --version|kilo"
    "semgrep|semgrep|||semgrep --version|semgrep"
    "trivy|trivy|||trivy --version|trivy"
    "trivy-mcp|trivy|||trivy plugin list|trivy-mcp"
)

# =============================================================================
# UTILITY FUNCTIONS: logging, help, version
# =============================================================================

show_help() {
    cat << EOF
Usage:
  $(basename "$0") [OPTIONS]

Options:
  -p, --path <dir>      Install to a specific project path
  -c, --cursor          Setup configuration for Cursor
  -C, --copilot         Setup configuration for Copilot
  -k, --kilo            Setup configuration for Kilocode
  -o, --opencode        Setup configuration for opencode
  -a, --antigravity     Setup configuration for Google Antigravity
  -A, --all             Setup for all supported IDEs (Default if none selected)
  -AA, --all-mcp        Merge every hub MCP server (skip MCP prompt; clears any subset filter). Implies MCP merge on unless later --no-mcp
  -t, --tools           Interactive prompt for extra optional tools (openspec, dmux, brave-search-cli, cocoindex-code); gitnexus + engram CLIs are always installed when possible
  -d, --deps            Check and install dependencies before install
  --sanity-check        Verify hub dependencies and expected files without installing or writing
  -y, --yes             Non-interactive mode (skip prompts)
  --no-mcp              Skip MCP server configuration
  -n, --dry-run         Show what would be installed without making changes
  -s, --sync-assets     Run IDE asset sync (render hub commands/prompts/agents/rules into each selected IDE after install)
  --no-sdd-hooks        Skip installing SDD gate pre-commit/pre-push git hooks
  -u, --uninstall       Remove .ai-config and managed IDE dirs from target
  -v, --version         Print Version
  -h, --help            Show this help message

Interactive mode: On TTY with no IDE flags, choose IDEs (1-5 or a=all) and MCP servers.
Use -AA or --all-mcp to skip the MCP prompt and install all hub MCP servers (still respects --no-mcp if it appears later on the command line).
Use -t to pick optional tools interactively (openspec, dmux, brave-search-cli, cocoindex-code/ccc). GitNexus and Engram CLIs are always attempted (hub MCP); see docs/01-installation.md.
EOF
}

show_version() {
    echo "install.sh version $VERSION"
}

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[OK]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# =============================================================================
# INTERACTIVE PROMPTS: IDE selection, MCP selection
# =============================================================================

# Check if interactive mode is eligible
interactive_eligible() {
    # Not interactive if: NON_INTERACTIVE, CI env, not a TTY, or IDE flags already set
    [ "$NON_INTERACTIVE" != true ] || return 1
    case "${CI:-}" in
        true|1|yes|YES) return 1 ;;
    esac
    [ -t 0 ] && [ -t 1 ] || return 1
    [ ${#SELECTED_IDES[@]} -eq 0 ] || return 1
    return 0
}

# Interactive IDE selection
interactive_ide_selection() {
    interactive_eligible || return 0

    echo ""
    echo -e "${CYAN}IDE integration${NC} — symlink the hub into which tools?"
    echo "  1) Cursor (.cursor/)"
    echo "  2) Copilot (.vscode/)"
    echo "  3) Kilocode (.kilo/)"
    echo "  4) OpenCode (.opencode/)"
    echo "  5) Antigravity (.agents/)"
    echo ""
    echo "  a — all five   |   e.g. 1,3,5 — only those numbers"
    echo ""

    local choice
    while true; do
        if ! read -r -p "IDE choice [a]: " choice; then
            echo ""
            ALL_IDES=true
            SELECTED_IDES=("cursor" "copilot" "kilo" "opencode" "antigravity")
            log_info "IDE: all five (default)"
            return 0
        fi
        choice=$(printf '%s' "$choice" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        if [ -z "$choice" ]; then
            ALL_IDES=true
            SELECTED_IDES=("cursor" "copilot" "kilo" "opencode" "antigravity")
            log_info "IDE: all five (default)"
            return 0
        fi
        local lower
        lower=$(printf '%s' "$choice" | tr '[:upper:]' '[:lower:]')
        case "$lower" in
            a|all)
                ALL_IDES=true
                SELECTED_IDES=("cursor" "copilot" "kilo" "opencode" "antigravity")
                log_info "IDE: all five"
                return 0
                ;;
        esac

        SELECTED_IDES=()
        local -a parts
        local tok invalid=""
        IFS=',' read -ra parts <<< "$choice"
        for tok in "${parts[@]}"; do
            tok=$(printf '%s' "$tok" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
            [ -z "$tok" ] && continue
            case "$tok" in
                1) SELECTED_IDES+=("cursor") ;;
                2) SELECTED_IDES+=("copilot") ;;
                3) SELECTED_IDES+=("kilo") ;;
                4) SELECTED_IDES+=("opencode") ;;
                5) SELECTED_IDES+=("antigravity") ;;
                *) invalid="invalid index: $tok (use 1–5 or a)"; break ;;
            esac
        done

        if [ -n "$invalid" ]; then
            log_warn "$invalid"
            continue
        fi
        if [ ${#SELECTED_IDES[@]} -eq 0 ]; then
            log_warn "Pick at least one number (1–5) or a for all"
            continue
        fi
        log_info "IDE: selected ${#SELECTED_IDES[@]} tool(s)"
        return 0
    done
}

# Get available MCP servers from .configs/mcp/
get_available_mcp_servers() {
    local mcp_dir="$SCRIPT_DIR/.configs/mcp"
    local servers=()

    if [ -d "$mcp_dir" ]; then
        # Collect all JSON files recursively
        local -a merge_paths=()
        local main_mcp="$SCRIPT_DIR/.configs/mcp.json"
        [ -f "$main_mcp" ] && merge_paths+=("$main_mcp")

        local f
        if [ -d "$mcp_dir" ]; then
            while IFS= read -r -d '' f; do
                merge_paths+=("$f")
            done < <(find "$mcp_dir" -name '*.json' -type f -print0 2>/dev/null | sort -z)
        fi

        # Extract unique server keys from all files, one per line
        if [ ${#merge_paths[@]} -gt 0 ] && command -v jq &>/dev/null; then
            jq -s '[.[] | (if (type == "object") and has("mcpServers") then .mcpServers else . end) | keys[]] | unique | reverse | .[]' "${merge_paths[@]}" 2>/dev/null
        fi
    fi
}

# Interactive MCP server selection
interactive_mcp_selection() {
    # Skip if --no-mcp was passed
    [ "$MERGE_MCP" != false ] || return 0
    # Explicit all MCP: no prompt, no subset filter (useful with interactive IDE pick)
    if [ "$ALL_MCP" = true ]; then
        MERGE_MCP=true
        MCP_KEY_FILTER_JSON=""
        log_info "MCP: all servers (--all-mcp)"
        return 0
    fi
    # Not interactive if: NON_INTERACTIVE, CI env, not a TTY
    [ "$NON_INTERACTIVE" != true ] || return 0
    case "${CI:-}" in
        true|1|yes|YES) return 0 ;;
    esac
    [ -t 0 ] && [ -t 1 ] || return 0

    local mcp_dir="$SCRIPT_DIR/.configs/mcp"
    local main_mcp="$SCRIPT_DIR/.configs/mcp.json"

    # Check if any MCP configs exist
    local has_configs=false
    [ -f "$main_mcp" ] && has_configs=true
    [ -d "$mcp_dir" ] && [ -n "$(ls -A "$mcp_dir" 2>/dev/null)" ] && has_configs=true

    if [ "$has_configs" = false ]; then
        log_info "MCP: no .configs/mcp/ directory found — skipping MCP selection"
        return 0
    fi

    # Check for jq
    if ! command -v jq &>/dev/null; then
        log_warn "MCP: jq not found — skipping MCP selection (install jq or use --no-mcp)"
        return 0
    fi

    echo ""
    echo -e "${CYAN}MCP Servers${NC} — which servers to enable?"
    echo "  a — all servers   |   n — skip MCP   |   e.g. 1,3,5 — subset"
    echo ""

    # List available servers
    local -a servers
    local i=1
    while IFS= read -r server; do
        if [ -n "$server" ] && [ "$server" != "null" ]; then
            servers+=("$server")
            echo "  $i) $server"
            i=$((i + 1))
        fi
    done < <(get_available_mcp_servers)

    if [ ${#servers[@]} -eq 0 ]; then
        log_info "MCP: no servers found — skipping"
        return 0
    fi

    echo ""

    local choice
    while true; do
        if ! read -r -p "MCP choice [a]: " choice; then
            echo ""
            log_info "MCP: all servers (default)"
            return 0
        fi
        choice=$(printf '%s' "$choice" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        if [ -z "$choice" ]; then
            log_info "MCP: all servers (default)"
            return 0
        fi
        local lower
        lower=$(printf '%s' "$choice" | tr '[:upper:]' '[:lower:]')
        case "$lower" in
            a|all)
                log_info "MCP: all servers"
                return 0
                ;;
            n|no|skip)
                MERGE_MCP=false
                log_info "MCP: skipped"
                return 0
                ;;
        esac

        local -a selected_indices
        local -a parts
        local tok invalid=""
        IFS=',' read -ra parts <<< "$choice"
        for tok in "${parts[@]}"; do
            tok=$(printf '%s' "$tok" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
            [ -z "$tok" ] && continue
            if [[ "$tok" =~ ^[0-9]+$ ]] && [ "$tok" -ge 1 ] && [ "$tok" -le "${#servers[@]}" ]; then
                selected_indices+=("$tok")
            else
                invalid="invalid index: $tok (use 1–${#servers[@]}, a, or n)"; break
            fi
        done

        if [ -n "$invalid" ]; then
            log_warn "$invalid"
            continue
        fi
        if [ ${#selected_indices[@]} -eq 0 ]; then
            log_warn "Pick at least one number, a for all, or n to skip"
            continue
        fi

        # Build JSON filter for selected servers
        local -a selected_names
        for idx in "${selected_indices[@]}"; do
            local name="${servers[$((idx-1))]}"
            selected_names+=("$name")
        done

        # JSON array via jq so keys with quotes/special chars are valid for --argjson
        MCP_KEY_FILTER_JSON=$(printf '%s\n' "${selected_names[@]}" | jq -R -s 'split("\n") | map(select(length > 0))') || {
            log_error "MCP: could not encode server key list (jq) — try again"
            continue
        }

        log_info "MCP: selected ${#selected_names[@]} server(s)"
        return 0
    done
}

# Return 0 if optional tool id is in SELECTED_TOOLS
tool_is_selected() {
    local id="$1"
    local t
    for t in "${SELECTED_TOOLS[@]}"; do
        [ "$t" = "$id" ] && return 0
    done
    return 1
}

# Return 0 if merged MCP will include the hub server key "trivy" (.configs/mcp/trivy.json)
mcp_trivy_is_selected() {
    [ "$MERGE_MCP" = true ] || return 1
    if [ -z "$MCP_KEY_FILTER_JSON" ]; then
        return 0
    fi
    command -v jq &>/dev/null || return 1
    printf '%s' "$MCP_KEY_FILTER_JSON" | jq -e 'type == "array" and index("trivy") != null' >/dev/null 2>&1
}

# Install Trivy's MCP plugin when the hub trivy server is part of the merged MCP selection
ensure_trivy_mcp_plugin() {
    mcp_trivy_is_selected || return 0
    command -v trivy &>/dev/null || return 0
    if [ "$DRY_RUN" = true ]; then
        echo "[DRY-RUN] trivy plugin install mcp"
        echo ""
        return 0
    fi
    echo "Installing trivy MCP plugin..."
    trivy plugin install mcp
    echo ""
}

# Ensure uv is available (for cocoindex-code[full] → ccc: uv tool install)
ensure_uv() {
    command -v uv &>/dev/null && return 0
    if [ "$DRY_RUN" = true ]; then
        echo "[DRY-RUN] brew install uv  (or: curl -LsSf https://astral.sh/uv/install.sh | sh)"
        return 0
    fi
    log_info "Installing uv..."
    if command -v brew &>/dev/null; then
        brew install uv || true
    elif command -v curl &>/dev/null; then
        curl -LsSf https://astral.sh/uv/install.sh | sh || true
    else
        log_warn "uv not found — install with: brew install uv (or https://docs.astral.sh/uv/)"
        return 1
    fi
    command -v uv &>/dev/null
}

# Run npm ci in the hub when package.json exists (pins openspec, dmux, MCP packages; see package.json)
hub_ensure_npm_install() {
    [ -n "${SCRIPT_DIR:-}" ] && [ -f "$SCRIPT_DIR/package.json" ] || return 1
    if [ "$DRY_RUN" = true ]; then
        echo "[DRY-RUN] (cd \"$SCRIPT_DIR\" && npm ci)"
        return 0
    fi
    log_info "Hub Node dependencies (npm ci in $SCRIPT_DIR)..."
    (cd "$SCRIPT_DIR" && npm ci) 2>/dev/null || (cd "$SCRIPT_DIR" && npm install) || {
        log_warn "hub npm ci failed — check Node, network, and package-lock.json (see docs/01-installation.md)"
        return 1
    }
    return 0
}

# Install openspec CLI: prefer PATH, else Homebrew, else hub node_modules, else npm -g
install_openspec_cli() {
    if command -v openspec &>/dev/null; then
        log_info "openspec CLI already on PATH"
        return 0
    fi
    if [ "$DRY_RUN" = true ]; then
        echo "[DRY-RUN] brew install openspec  ||  hub npm ci  ||  npm install -g @fission-ai/openspec@latest"
        return 0
    fi
    if command -v brew &>/dev/null; then
        if brew install openspec; then
            log_success "openspec installed (Homebrew)"
            return 0
        fi
    fi
    if hub_ensure_npm_install && [ -x "$SCRIPT_DIR/node_modules/.bin/openspec" ]; then
        log_success "openspec available: cd \"$SCRIPT_DIR\" && npx openspec (or add node_modules/.bin to PATH)"
        return 0
    fi
    if command -v npm &>/dev/null; then
        log_info "Installing openspec via npm global (@fission-ai/openspec)..."
        if npm install -g @fission-ai/openspec@latest; then
            log_success "openspec installed (npm -g)"
            return 0
        fi
    fi
    log_warn "openspec: install manually — brew install openspec  OR  npm ci in hub  OR  npm install -g @fission-ai/openspec@latest"
    return 1
}

# Install CLIs for SELECTED_TOOLS (uv / hub npm / brew).
# GitNexus + Engram match hub MCP fragments (.configs/mcp/); their CLIs are always reconciled.
install_optional_tool_clis() {
    tool_is_selected gitnexus || SELECTED_TOOLS+=("gitnexus")
    tool_is_selected engram || SELECTED_TOOLS+=("engram")

    [ ${#SELECTED_TOOLS[@]} -eq 0 ] && return 0

    echo ""
    echo "Optional tools — installing CLIs (includes gitnexus + engram for bundled MCP)..."
    echo ""

    if tool_is_selected gitnexus; then
        if command -v gitnexus &>/dev/null 2>&1; then
            log_info "gitnexus CLI already present"
        elif [ "$DRY_RUN" = true ]; then
            echo "[DRY-RUN] npm install -g gitnexus@1.3.11"
        elif command -v npm &>/dev/null; then
            log_info "Installing GitNexus (npm install -g gitnexus@1.3.11)..."
            if npm install -g gitnexus@1.3.11; then
                log_success "gitnexus installed"
            else
                log_warn "gitnexus: npm install -g failed"
            fi
        else
            log_warn "gitnexus: npm not found — install Node.js, then run: npm install -g gitnexus@1.3.11"
        fi
    fi

    # CocoIndex Code — publishes the `ccc` CLI (https://github.com/cocoindex-io/cocoindex-code)
    if tool_is_selected cocoindex-code; then
        ensure_uv || true
        if [ "$DRY_RUN" = true ]; then
            echo "[DRY-RUN] uv tool install --upgrade 'cocoindex-code[full]'"
        elif command -v uv &>/dev/null; then
            log_info "Installing/upgrading cocoindex-code[full] (ccc) via uv tool install..."
            if uv tool install --upgrade 'cocoindex-code[full]'; then
                log_success "cocoindex-code (ccc) installed or upgraded"
            else
                log_warn "cocoindex-code: uv tool install --upgrade failed"
            fi
        else
            log_warn "cocoindex-code: uv missing — run: uv tool install --upgrade 'cocoindex-code[full]'"
        fi
    fi

    if tool_is_selected openspec; then
        install_openspec_cli || true
    fi

    if tool_is_selected dmux; then
        if command -v dmux &>/dev/null; then
            log_info "dmux CLI already on PATH"
        elif [ "$DRY_RUN" = true ]; then
            echo "[DRY-RUN] hub npm ci  ||  npm install -g dmux"
        elif hub_ensure_npm_install && [ -x "$SCRIPT_DIR/node_modules/.bin/dmux" ]; then
            log_success "dmux available: cd \"$SCRIPT_DIR\" && npx dmux (or add node_modules/.bin to PATH)"
        elif command -v npm &>/dev/null; then
            log_info "Installing dmux (npm -g fallback)..."
            if npm install -g dmux; then
                log_success "dmux installed (npm -g)"
            else
                log_warn "dmux: npm install -g failed"
            fi
        else
            log_warn "dmux: npm not found — install Node.js"
        fi
    fi

    if tool_is_selected engram; then
        if command -v engram &>/dev/null; then
            local engram_path
            engram_path=$(command -v engram)
            log_info "engram CLI already installed: $engram_path"
        elif [ "$DRY_RUN" = true ]; then
            echo "[DRY-RUN] brew install gentleman-programming/tap/engram"
            echo "[DRY-RUN] verify MCP fragment: .configs/mcp/command/engram.json"
        elif command -v brew &>/dev/null; then
            log_info "Installing engram (Homebrew)..."
            if brew install gentleman-programming/tap/engram; then
                log_success "engram installed"
            else
                log_warn "engram: brew install failed — install manually with: brew install gentleman-programming/tap/engram"
            fi
        else
            log_warn "engram: Homebrew not found — run: brew install gentleman-programming/tap/engram"
        fi
    fi

    # Brave Search CLI — installs `bx` (https://github.com/brave/brave-search-cli)
    if tool_is_selected brave-search-cli; then
        if command -v bx &>/dev/null; then
            log_info "brave-search-cli (bx) already on PATH"
        elif [ "$DRY_RUN" = true ]; then
            echo "[DRY-RUN] curl -fsSL https://raw.githubusercontent.com/brave/brave-search-cli/main/scripts/install.sh | sh"
        elif command -v curl &>/dev/null; then
            log_info "Installing brave-search-cli (official install.sh → bx)..."
            if curl -fsSL https://raw.githubusercontent.com/brave/brave-search-cli/main/scripts/install.sh | sh; then
                log_success "brave-search-cli installed (bx)"
            else
                log_warn "brave-search-cli: install script failed"
            fi
        else
            log_warn "brave-search-cli: curl not found — install curl or run the install command manually"
        fi
    fi

    echo ""
}

# Interactive optional tools (openspec, dmux, brave-search-cli, cocoindex-code; gitnexus+engram always installed)
interactive_tools_selection() {
    [ "$TOOLS_INTERACTIVE" = true ] || return 0
    [ "$NON_INTERACTIVE" != true ] || {
        log_info "Optional tools: skipping -t prompt (--yes); gitnexus + engram CLIs still run with the rest of install"
        return 0
    }
    case "${CI:-}" in
        true|1|yes|YES)
            log_info "Optional tools: skipping -t prompt (CI); gitnexus + engram CLIs still run with install"
            return 0
            ;;
    esac
    [ -t 0 ] && [ -t 1 ] || {
        log_warn "Optional tools: not a TTY — skipping tool selection (use a terminal for -t)"
        return 0
    }

    echo ""
    echo -e "${CYAN}Optional tools${NC} — CLIs via npm, uv, hub npm ci, brew, or Brave install (see docs/01-installation.md)"
    echo "  1) openspec — OpenSpec (brew, hub npx, or npm -g)"
    echo "  2) dmux — tmux pane manager (hub npx, or npm -g fallback)"
    echo "  3) brave-search-cli — Brave Search CLI, bx (curl | sh from brave/brave-search-cli)"
    echo "  4) cocoindex-code — CocoIndex Code, ccc (uv tool install --upgrade 'cocoindex-code[full]')"
    echo ""
    echo "  (gitnexus + engram are installed automatically for hub MCP — not listed here.)"
    echo ""
    echo "  a — all four   |   n — none   |   e.g. 1,2 — pick by number"
    echo ""

    local choice
    while true; do
        if ! read -r -p "Tool choice [n]: " choice; then
            echo ""
            log_info "Optional tools: none (default)"
            return 0
        fi
        choice=$(printf '%s' "$choice" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        if [ -z "$choice" ]; then
            log_info "Optional tools: none (default)"
            return 0
        fi
        local lower
        lower=$(printf '%s' "$choice" | tr '[:upper:]' '[:lower:]')
        case "$lower" in
            a|all)
                SELECTED_TOOLS=("openspec" "dmux" "brave-search-cli" "cocoindex-code")
                log_info "Optional tools: openspec, dmux, brave-search-cli, cocoindex-code (gitnexus + engram always)"
                return 0
                ;;
            n|no|none|skip)
                log_info "Optional tools: none"
                return 0
                ;;
        esac

        SELECTED_TOOLS=()
        local tok invalid=""
        local -a parts
        IFS=',' read -ra parts <<< "$choice"
        for tok in "${parts[@]}"; do
            tok=$(printf '%s' "$tok" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
            [ -z "$tok" ] && continue
            case "$tok" in
                1) SELECTED_TOOLS+=("openspec") ;;
                2) SELECTED_TOOLS+=("dmux") ;;
                3) SELECTED_TOOLS+=("brave-search-cli") ;;
                4) SELECTED_TOOLS+=("cocoindex-code") ;;
                *) invalid="invalid index: $tok (use 1–4, a, or n)"; break ;;
            esac
        done

        if [ -n "$invalid" ]; then
            log_warn "$invalid"
            continue
        fi
        if [ ${#SELECTED_TOOLS[@]} -eq 0 ]; then
            log_warn "Pick at least one number (1–4), a for all, or n for none"
            continue
        fi
        log_info "Optional tools: selected ${#SELECTED_TOOLS[@]} tool(s)"
        return 0
    done
}

# Merge all MCP configs from .configs/mcp.json and .configs/mcp/**/*.json
# Returns merged JSON on stdout (log messages go to stderr)
merge_mcp_configs() {
    local mcp_dir="$SCRIPT_DIR/.configs/mcp"
    local main_mcp="$SCRIPT_DIR/.configs/mcp.json"

    if [ "$MERGE_MCP" = false ]; then
        return 0
    fi

    # Collect all JSON files
    local -a merge_paths=()
    [ -f "$main_mcp" ] && merge_paths+=("$main_mcp")
    
    local f
    if [ -d "$mcp_dir" ]; then
        while IFS= read -r -d '' f; do
            merge_paths+=("$f")
        done < <(find "$mcp_dir" -name '*.json' -type f -print0 2>/dev/null | sort -z)
    fi

    if [ ${#merge_paths[@]} -eq 0 ]; then
        log_info "No MCP configs found — skip MCP merge" >&2
        return 0
    fi

    if ! command -v jq &>/dev/null; then
        log_error "jq is required to merge MCP configs (install jq or pass --no-mcp)" >&2
        return 1
    fi

    log_info "Merging MCP configs (${#merge_paths[@]} file(s))" >&2

    # Merge all configs
    local merged_json
    merged_json=$(jq -s '
      def normalize:
        if (type == "object") and has("mcpServers") then { mcpServers: .mcpServers }
        else { mcpServers: . }
        end;
      reduce .[] as $item ({"mcpServers": {}}; ($item | normalize) as $n | { mcpServers: (.mcpServers * $n.mcpServers) })
    ' "${merge_paths[@]}" 2>/dev/null) || {
        log_error "MCP merge failed (jq)" >&2
        return 1
    }

    # Apply filter if subset selected (--argjson name must not shadow jq builtins like "keys")
    if [ -n "$MCP_KEY_FILTER_JSON" ]; then
        local jq_filter_err
        jq_filter_err=$(mktemp)
        merged_json=$(printf '%s' "$merged_json" | jq --argjson keep "$MCP_KEY_FILTER_JSON" '
          { mcpServers: ((.mcpServers // {}) | with_entries(select(.key as $k | ($keep | index($k) != null)))) }
        ' 2>"$jq_filter_err") || {
            log_error "MCP subset filter failed (jq): $(tr '\n' ' ' < "$jq_filter_err")" >&2
            rm -f "$jq_filter_err"
            return 1
        }
        rm -f "$jq_filter_err"
        local count
        count=$(echo "$merged_json" | jq '.mcpServers | keys | length')
        log_info "Applied MCP server subset ($count server(s))" >&2
    fi

    local server_count
    server_count=$(echo "$merged_json" | jq '.mcpServers | keys | length')
    log_success "Merged MCP: $server_count server(s)" >&2

    # Return the merged JSON
    echo "$merged_json"
}

# -----------------------------------------------------------------------------
# MCP emitters: hub fragments are Cursor-style mcpServers (stdio: command+args;
# remote: type streamable-http + url). Each consumer gets a valid native shape.
# -----------------------------------------------------------------------------

# .cursor/mcp.json — remote: url (+headers/auth); stdio: type + command + args
mcp_emit_for_cursor() {
    jq '
      { mcpServers: (
          .mcpServers // {} | to_entries | map(
            .key as $k | .value as $v |
            if ($v | type != "object") then {key: $k, value: $v}
            elif ($v | has("url")) and ($v | has("command") | not) then
              {key: $k, value: (
                {url: $v.url}
                + (if ($v | has("headers")) then {headers: $v.headers} else {} end)
                + (if ($v | has("auth")) then {auth: $v.auth} else {} end)
              )}
            elif ($v | has("command")) then
              {key: $k, value: (
                {type: "stdio", command: $v.command, args: ($v.args // [])}
                + (if ($v | has("env")) then {env: $v.env} else {} end)
                + (if ($v | has("envFile")) then {envFile: $v.envFile} else {} end)
                + (if ($v | has("cwd")) then {cwd: $v.cwd} else {} end)
              )}
            else {key: $k, value: $v}
            end
          ) | from_entries
        )
      }
    '
}

# .vscode/mcp.json — VS Code / Copilot: top-level "servers", explicit type (http|stdio|sse)
mcp_emit_for_vscode() {
    jq '
      { servers: (
          .mcpServers // {} | to_entries | map(
            .key as $k | .value as $v |
            if ($v | type != "object") then {key: $k, value: $v}
            elif ($v | has("url")) and ($v | has("command") | not) then
              {key: $k, value: (
                {type: (if ($v.type == "sse") then "sse" else "http" end), url: $v.url}
                + (if ($v | has("headers")) then {headers: $v.headers} else {} end)
              )}
            elif ($v | has("command")) then
              {key: $k, value: (
                {type: "stdio", command: $v.command, args: ($v.args // [])}
                + (if ($v | has("env")) then {env: $v.env} else {} end)
                + (if ($v | has("cwd")) then {cwd: $v.cwd} else {} end)
              )}
            else {key: $k, value: $v}
            end
          ) | from_entries
        )
      }
    '
}

# OpenCode .mcp / Kilo .kilo/kilo.json "mcp" — schema https://opencode.ai/config.json
mcp_emit_opencode_style_mcp_object() {
    jq -c '
      (.mcpServers // {}) | to_entries | map(
        .key as $k | .value as $v |
        if ($v | type != "object") then {key: $k, value: $v}
        elif ($v | has("url")) and ($v | has("command") | not) then
          {key: $k, value: (
            {enabled: true, type: "remote", url: $v.url}
            + (if ($v | has("headers")) then {headers: $v.headers} else {} end)
          )}
        elif ($v | has("command")) then
          {key: $k, value: (
            {enabled: true, type: "local", command: ([$v.command] + ($v.args // []))}
            + (if ($v | has("env")) then {environment: $v.env} else {} end)
          )}
        else {key: $k, value: $v}
        end
      ) | from_entries
    '
}

# ~/.gemini/antigravity/mcp_config.json — HTTP: serverUrl; stdio: command + args
mcp_emit_for_antigravity() {
    jq '
      { mcpServers: (
          .mcpServers // {} | to_entries | map(
            .key as $k | .value as $v |
            if ($v | type != "object") then {key: $k, value: $v}
            elif ($v | has("serverUrl")) then {key: $k, value: $v}
            elif ($v | has("url")) and ($v | has("command") | not) then
              {key: $k, value: (
                {serverUrl: $v.url}
                + (if ($v | has("headers")) then {headers: $v.headers} else {} end)
                + (if ($v | has("env")) then {env: $v.env} else {} end)
              )}
            elif ($v | has("command")) then
              {key: $k, value: (
                {command: $v.command, args: ($v.args // [])}
                + (if ($v | has("env")) then {env: $v.env} else {} end)
                + (if ($v | has("cwd")) then {cwd: $v.cwd} else {} end)
              )}
            else {key: $k, value: $v}
            end
          ) | from_entries
        )
      }
    '
}

verify_engram_setup() {
    [ "$MERGE_MCP" = true ] || {
        log_info "Engram MCP: skipped because --no-mcp was used"
        return 0
    }

    local merged_mcp="$1"
    local fragment="$SCRIPT_DIR/.configs/mcp/command/engram.json"

    if [ ! -f "$fragment" ]; then
        log_warn "Engram MCP: missing fragment .configs/mcp/command/engram.json"
        return 0
    fi

    if command -v engram &>/dev/null; then
        log_success "Engram CLI available: $(command -v engram)"
    else
        log_warn "Engram CLI not on PATH — install with: brew install gentleman-programming/tap/engram"
    fi

    if [ -z "$merged_mcp" ]; then
        log_warn "Engram MCP: no merged MCP config was produced"
        return 0
    fi

    if ! command -v jq &>/dev/null; then
        log_warn "Engram MCP: jq missing, cannot verify merged server list"
        return 0
    fi

    if printf '%s' "$merged_mcp" | jq -e '.mcpServers.engram' >/dev/null 2>&1; then
        log_success "Engram MCP server included in merged config"
    else
        log_warn "Engram MCP server not included in merged config — select it during MCP setup or use all MCP servers"
    fi
}

show_dependencies() {
    echo "=== Dependency Check ==="
    echo ""
    
    local missing=()
    local installed=()
    
    # Check core dependencies
    echo "Core Dependencies:"
    for dep in "${CORE_DEPENDENCIES[@]}"; do
        IFS=':' read -r name brew pip npm check_cmd <<< "$dep"
        
        # Skip empty check commands
        if [ -z "$check_cmd" ]; then
            continue
        fi
        
        # Check if dependency is installed
        if eval "$check_cmd" &>/dev/null; then
            installed+=("$name")
            echo "  ✓ $name"
        else
            missing+=("$name|$brew|$pip|$npm|core")
            echo "  ✗ $name"
        fi
    done
    
    echo ""
    
    # Check IDE-specific dependencies
    echo "IDE-Specific Dependencies:"
    for dep in "${IDE_DEPENDENCIES[@]}"; do
        IFS='|' read -r name brew pip npm check_cmd ide_flag <<< "$dep"
        
        # Skip if this IDE is not selected
        local is_selected=false
        for selected in "${SELECTED_IDES[@]}"; do
            if [ "$selected" = "$ide_flag" ]; then
                is_selected=true
                break
            fi
        done
        
        # Special handling for openspec (optional tools)
        if [ "$ide_flag" = "openspec" ] && tool_is_selected openspec; then
            is_selected=true
        fi

        # Trivy CLI + MCP plugin when hub "trivy" MCP server is selected
        if { [ "$ide_flag" = "trivy" ] || [ "$ide_flag" = "trivy-mcp" ]; } && mcp_trivy_is_selected; then
            is_selected=true
        fi

        # Check if running with -A (all)
        if [ "$ALL_IDES" = true ]; then
            is_selected=true
        fi

        if [ "$is_selected" = false ]; then
            echo "  - $name (skipped, not selected)"
            continue
        fi
        
        # Skip empty check commands
        if [ -z "$check_cmd" ]; then
            continue
        fi
        
        # Check if dependency is installed
        if eval "$check_cmd" &>/dev/null; then
            installed+=("$name")
            echo "  ✓ $name"
        else
            missing+=("$name|$brew|$pip|$npm|$ide_flag")
            echo "  ✗ $name"
        fi
    done
    
    echo ""
    echo "Installed: ${#installed[@]}"
    echo "Missing: ${#missing[@]}"
    echo ""
    
    if [ ${#missing[@]} -gt 0 ]; then
        echo "Missing dependencies:"
        for dep in "${missing[@]}"; do
            IFS='|' read -r name brew pip npm ide_flag <<< "$dep"
            local install_cmd=""
            
            if [ -n "$brew" ]; then
                install_cmd="brew install $brew"
            elif [ -n "$npm" ]; then
                install_cmd="npm install -g $npm"
            elif [ -n "$pip" ]; then
                install_cmd="pip3 install $pip"
            fi
            
            if [ -n "$install_cmd" ]; then
                echo "  - $name → $install_cmd"
            else
                echo "  - $name → (manual install required)"
            fi
        done
        echo ""
    fi
}

sanity_ok() {
    echo "  ✓ $1"
}

sanity_fail() {
    echo "  ✗ $1"
    SANITY_FAILURES=$((SANITY_FAILURES + 1))
}

sanity_check_command() {
    local name="$1"
    local check_cmd="$2"
    local hint="$3"

    if eval "$check_cmd" &>/dev/null; then
        sanity_ok "$name"
    else
        if [ -n "$hint" ]; then
            sanity_fail "$name — $hint"
        else
            sanity_fail "$name"
        fi
    fi
}

sanity_check_file() {
    local label="$1"
    local file_path="$2"

    if [ -e "$file_path" ]; then
        sanity_ok "$label"
    else
        sanity_fail "$label — missing $file_path"
    fi
}

run_sanity_check() {
    SANITY_FAILURES=0

    echo "=== install.sh Sanity Check ==="
    echo ""

    echo "Core commands:"
    for dep in "${CORE_DEPENDENCIES[@]}"; do
        IFS=':' read -r name brew pip npm check_cmd <<< "$dep"
        [ -z "$check_cmd" ] && continue
        local hint=""
        if [ -n "$brew" ]; then
            hint="install with: brew install $brew"
        elif [ -n "$npm" ]; then
            hint="install with: npm install -g $npm"
        elif [ -n "$pip" ]; then
            hint="install with: pip3 install $pip"
        fi
        sanity_check_command "$name" "$check_cmd" "$hint"
    done

    echo ""
    echo "IDE and security CLIs:"
    for dep in "${IDE_DEPENDENCIES[@]}"; do
        IFS='|' read -r name brew pip npm check_cmd ide_flag <<< "$dep"
        [ -z "$check_cmd" ] && continue
        local hint=""
        if [ -n "$brew" ]; then
            hint="install with: brew install $brew"
        elif [ -n "$npm" ]; then
            hint="install with: npm install -g $npm"
        elif [ -n "$pip" ]; then
            hint="install with: pip3 install $pip"
        fi
        sanity_check_command "$name" "$check_cmd" "$hint"
    done

    echo ""
    echo "Optional Skillgrid tools:"
    sanity_check_command "uv" "command -v uv" "install with: brew install uv"
    sanity_check_command "gitnexus" "command -v gitnexus || command -v npx" "install with: npm install -g gitnexus@1.3.11"
    sanity_check_command "cocoindex-code (ccc)" "command -v ccc" "install with: uv tool install --upgrade 'cocoindex-code[full]'"
    sanity_check_command "dmux" "command -v dmux || [ -x \"$SCRIPT_DIR/node_modules/.bin/dmux\" ]" "run npm ci or install dmux"
    sanity_check_command "engram" "command -v engram" "install with: brew install gentleman-programming/tap/engram"
    sanity_check_command "brave-search-cli (bx)" "command -v bx" "install from brave-search-cli"

    echo ""
    echo "Hub files:"
    sanity_check_file "AGENTS.md template" "$SCRIPT_DIR/.configs/AGENTS.md"
    sanity_check_file "MCP config fragments" "$SCRIPT_DIR/.configs/mcp"
    sanity_check_file "Engram MCP fragment" "$SCRIPT_DIR/.configs/mcp/command/engram.json"
    sanity_check_file "Skill catalog" "$SCRIPT_DIR/.agents/skills"
    sanity_check_file "Skillgrid UI script" "$SCRIPT_DIR/.skillgrid/scripts/skillgrid-ui.mjs"
    sanity_check_file "Preview script" "$SCRIPT_DIR/.skillgrid/scripts/preview.sh"
    sanity_check_file "Node package manifest" "$SCRIPT_DIR/package.json"

    echo ""
    echo "Hub script checks:"
    sanity_check_command "skillgrid-ui.mjs syntax" "node --check \"$SCRIPT_DIR/.skillgrid/scripts/skillgrid-ui.mjs\"" "check Node and the dashboard script"

    echo ""
    if [ "$SANITY_FAILURES" -eq 0 ]; then
        log_success "Sanity check passed"
        return 0
    fi

    log_error "Sanity check found $SANITY_FAILURES issue(s)"
    return 1
}

install_dependencies() {
    echo "=== Installing Dependencies ==="
    echo ""
    
    local brew_pkgs=()
    local pip_pkgs=()
    local npm_pkgs=()
    
    # Find missing core dependencies
    for dep in "${CORE_DEPENDENCIES[@]}"; do
        IFS=':' read -r name brew pip npm check_cmd <<< "$dep"
        
        # Skip empty check commands
        if [ -z "$check_cmd" ]; then
            continue
        fi
        
        # Check if dependency is installed
        if ! eval "$check_cmd" &>/dev/null; then
            if [ -n "$brew" ]; then
                brew_pkgs+=("$brew")
            elif [ -n "$pip" ]; then
                pip_pkgs+=("$pip")
            elif [ -n "$npm" ]; then
                npm_pkgs+=("$npm")
            fi
        fi
    done
    
    # Find missing IDE-specific dependencies
    for dep in "${IDE_DEPENDENCIES[@]}"; do
        IFS='|' read -r name brew pip npm check_cmd ide_flag <<< "$dep"
        
        # Skip if this IDE is not selected
        local is_selected=false
        for selected in "${SELECTED_IDES[@]}"; do
            if [ "$selected" = "$ide_flag" ]; then
                is_selected=true
                break
            fi
        done
        
        # Special handling for openspec (optional tools)
        if [ "$ide_flag" = "openspec" ] && tool_is_selected openspec; then
            is_selected=true
        fi

        # Trivy CLI + MCP plugin when hub "trivy" MCP server is selected
        if { [ "$ide_flag" = "trivy" ] || [ "$ide_flag" = "trivy-mcp" ]; } && mcp_trivy_is_selected; then
            is_selected=true
        fi

        # Check if running with -A (all)
        if [ "$ALL_IDES" = true ]; then
            is_selected=true
        fi

        if [ "$is_selected" = false ]; then
            continue
        fi

        # Skip empty check commands
        if [ -z "$check_cmd" ]; then
            continue
        fi

        # Check if dependency is installed
        if ! eval "$check_cmd" &>/dev/null; then
            if [ -n "$brew" ]; then
                brew_pkgs+=("$brew")
            elif [ -n "$pip" ]; then
                pip_pkgs+=("$pip")
            elif [ -n "$npm" ]; then
                npm_pkgs+=("$npm")
            fi
        fi
    done
    
    # Install brew packages
    if [ ${#brew_pkgs[@]} -gt 0 ]; then
        echo "Installing with Homebrew: ${brew_pkgs[*]}"
        if [ "$DRY_RUN" = true ]; then
            echo "[DRY-RUN] brew install ${brew_pkgs[*]}"
        else
            # Check if brew is available
            if command -v brew &>/dev/null; then
                brew install "${brew_pkgs[@]}"
            else
                echo "  ⚠ Homebrew not found. Skipping brew packages."
                echo "  Install Homebrew: https://brew.sh"
            fi
        fi
        echo ""
    fi

    # Trivy MCP plugin when hub "trivy" MCP server is selected (after brew install, if any)
    ensure_trivy_mcp_plugin
    
    # Install pip packages
    if [ ${#pip_pkgs[@]} -gt 0 ]; then
        echo "Installing with pip: ${pip_pkgs[*]}"
        if [ "$DRY_RUN" = true ]; then
            echo "[DRY-RUN] pip3 install ${pip_pkgs[*]}"
        else
            if command -v pip3 &>/dev/null; then
                pip3 install --user "${pip_pkgs[@]}"
            else
                echo "  ⚠ pip3 not found. Skipping pip packages."
            fi
        fi
        echo ""
    fi
    
    # Install npm packages
    if [ ${#npm_pkgs[@]} -gt 0 ]; then
        echo "Installing with npm: ${npm_pkgs[*]}"
        if [ "$DRY_RUN" = true ]; then
            echo "[DRY-RUN] npm install -g ${npm_pkgs[*]}"
        else
            if command -v npm &>/dev/null; then
                npm install -g "${npm_pkgs[@]}"
            else
                echo "  ⚠ npm not found. Skipping npm packages."
            fi
        fi
        echo ""
    fi
    
    if [ ${#brew_pkgs[@]} -eq 0 ] && [ ${#pip_pkgs[@]} -eq 0 ] && [ ${#npm_pkgs[@]} -eq 0 ]; then
        echo "All dependencies are already installed!"
        echo ""
    fi
}

prompt_install_deps() {
    local missing_count=0

    # Count missing core dependencies
    for dep in "${CORE_DEPENDENCIES[@]}"; do
        IFS=':' read -r name brew pip npm check_cmd <<< "$dep"

        if [ -z "$check_cmd" ]; then
            continue
        fi

        if ! eval "$check_cmd" &>/dev/null; then
            missing_count=$((missing_count + 1))
        fi
    done
    
    # Count missing IDE-specific dependencies
    for dep in "${IDE_DEPENDENCIES[@]}"; do
        IFS='|' read -r name brew pip npm check_cmd ide_flag <<< "$dep"
        
        # Skip if this IDE is not selected
        local is_selected=false
        for selected in "${SELECTED_IDES[@]}"; do
            if [ "$selected" = "$ide_flag" ]; then
                is_selected=true
                break
            fi
        done
        
        # Special handling for openspec (optional tools)
        if [ "$ide_flag" = "openspec" ] && tool_is_selected openspec; then
            is_selected=true
        fi

        # Trivy CLI + MCP plugin when hub "trivy" MCP server is selected
        if { [ "$ide_flag" = "trivy" ] || [ "$ide_flag" = "trivy-mcp" ]; } && mcp_trivy_is_selected; then
            is_selected=true
        fi
        
        # Check if running with -A (all)
        if [ "$ALL_IDES" = true ]; then
            is_selected=true
        fi
        
        if [ "$is_selected" = false ]; then
            continue
        fi
        
        if [ -z "$check_cmd" ]; then
            continue
        fi

        if ! eval "$check_cmd" &>/dev/null; then
            missing_count=$((missing_count + 1))
        fi
    done

    if [ $missing_count -gt 0 ]; then
        echo "Found $missing_count missing dependencies."
        echo ""
        read -p "Would you like to install missing dependencies? [y/N] " -n 1 -r
        echo ""
        
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            install_dependencies
        else
            echo "Skipping dependency installation."
            echo ""
        fi
    fi
}

# =============================================================================
# ARGUMENT PARSING
# =============================================================================

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -p|--path)
            PROJECT_PATH="$2"
            shift 2
            ;;
        -c|--cursor)
            SELECTED_IDES+=("cursor")
            shift
            ;;
        -C|--copilot)
            SELECTED_IDES+=("copilot")
            shift
            ;;
        -k|--kilo)
            SELECTED_IDES+=("kilo")
            shift
            ;;
        -o|--opencode)
            SELECTED_IDES+=("opencode")
            shift
            ;;
        -a|--antigravity)
            SELECTED_IDES+=("antigravity")
            shift
            ;;
        -AA|--all-mcp)
            ALL_MCP=true
            MERGE_MCP=true
            MCP_KEY_FILTER_JSON=""
            shift
            ;;
        -A|--all-ides)
            SELECTED_IDES=("cursor" "copilot" "kilo" "opencode" "antigravity")
            ALL_IDES=true
            shift
            ;;
        -t|--tools)
            TOOLS_INTERACTIVE=true
            shift
            ;;
        -d|--deps)
            CHECK_DEPS=true
            shift
            ;;
        --sanity-check)
            SANITY_CHECK=true
            NON_INTERACTIVE=true
            shift
            ;;
        -y|--yes)
            NON_INTERACTIVE=true
            shift
            ;;
        --no-mcp)
            MERGE_MCP=false
            shift
            ;;
        -n|--dry-run)
            DRY_RUN=true
            shift
            ;;
        -s|--sync-assets)
            SYNC_ASSETS=true
            shift
            ;;
        --no-sdd-hooks)
            INSTALL_SDD_HOOKS=false
            shift
            ;;
        -u|--uninstall)
            UNINSTALL=true
            shift
            ;;
        -v|--version)
            show_version
            exit 0
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            echo "Error: Unknown option: $1"
            echo ""
            show_help
            exit 1
            ;;
    esac
done

# Get the directory where this script is located (needed for interactive MCP selection)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [ "$SANITY_CHECK" = true ]; then
    run_sanity_check
    exit $?
fi

# Interactive IDE selection (if eligible)
interactive_ide_selection

# If no IDEs selected after interactive, default to all
if [ ${#SELECTED_IDES[@]} -eq 0 ]; then
    SELECTED_IDES=("cursor" "copilot" "kilo" "opencode" "antigravity")
    ALL_IDES=true
fi

# Interactive MCP selection (if eligible)
interactive_mcp_selection

# Optional tools — must run before --deps counts (openspec / dmux / brave-search-cli / cocoindex-code; gitnexus+engram always in install_optional_tool_clis)
interactive_tools_selection

# Handle --deps flag (check and optionally install)
if [ "$CHECK_DEPS" = true ]; then
    show_dependencies

    # Count missing core dependencies
    missing_count=0
    for dep in "${CORE_DEPENDENCIES[@]}"; do
        IFS=':' read -r name brew pip npm check_cmd <<< "$dep"
        if [ -z "$check_cmd" ]; then continue; fi
        if ! eval "$check_cmd" &>/dev/null; then
            missing_count=$((missing_count + 1))
        fi
    done

    # Count missing IDE-specific dependencies
    for dep in "${IDE_DEPENDENCIES[@]}"; do
        IFS='|' read -r name brew pip npm check_cmd ide_flag <<< "$dep"

        # Skip if this IDE is not selected
        is_selected=false
        for selected in "${SELECTED_IDES[@]}"; do
            if [ "$selected" = "$ide_flag" ]; then
                is_selected=true
                break
            fi
        done

        # Special handling for openspec (optional tools)
        if [ "$ide_flag" = "openspec" ] && tool_is_selected openspec; then
            is_selected=true
        fi

        # Trivy CLI + MCP plugin when hub "trivy" MCP server is selected
        if { [ "$ide_flag" = "trivy" ] || [ "$ide_flag" = "trivy-mcp" ]; } && mcp_trivy_is_selected; then
            is_selected=true
        fi

        # Check if running with -A (all)
        if [ "$ALL_IDES" = true ]; then
            is_selected=true
        fi

        if [ "$is_selected" = false ]; then continue; fi
        if [ -z "$check_cmd" ]; then continue; fi

        if ! eval "$check_cmd" &>/dev/null; then
            missing_count=$((missing_count + 1))
        fi
    done

    if [ $missing_count -gt 0 ]; then
        read -p "Would you like to install missing dependencies? [y/N] " -n 1 -r
        echo ""
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            install_dependencies
        fi
    fi

    # If only checking deps, exit here
    if [ -z "$PROJECT_PATH" ]; then
        exit 0
    fi
fi

# Validate project path
if [ -z "$PROJECT_PATH" ]; then
    echo "Error: Project path is required."
    echo ""
    show_help
    exit 1
fi

if [ ! -d "$PROJECT_PATH" ]; then
    echo "Error: Directory '$PROJECT_PATH' does not exist"
    exit 1
fi

# Resolve to absolute path
PROJECT_PATH=$(cd "$PROJECT_PATH" && pwd)

# =============================================================================
# IDE SETUP FUNCTIONS: per-IDE config generation
# =============================================================================

# Setup Cursor: write mcp.json directly
setup_cursor() {
    local target="$1"
    local merged_mcp="$2"
    local tool_dir="$target/.cursor"
    local mcp_file="$tool_dir/mcp.json"

    log_info "Setting up Cursor configuration..."
    ensure_dir "$tool_dir"

    if [ "$MERGE_MCP" = true ] && [ -n "$merged_mcp" ]; then
        printf '%s' "$merged_mcp" | mcp_emit_for_cursor | jq '.' > "$mcp_file"
        local count
        count=$(jq '.mcpServers | keys | length' "$mcp_file")
        log_success "Generated: $mcp_file ($count server(s))"
    else
        log_info "Skipping Cursor mcp.json (--no-mcp or no merged data)"
    fi

    log_success "Cursor setup complete"
}

# Setup Copilot (VSCode): write mcp.json directly
setup_copilot() {
    local target="$1"
    local merged_mcp="$2"
    local tool_dir="$target/.vscode"
    local mcp_file="$tool_dir/mcp.json"

    log_info "Setting up Copilot configuration..."
    ensure_dir "$tool_dir"

    if [ "$MERGE_MCP" = true ] && [ -n "$merged_mcp" ]; then
        printf '%s' "$merged_mcp" | mcp_emit_for_vscode | jq '.' > "$mcp_file"
        local count
        count=$(jq '.servers | keys | length' "$mcp_file")
        log_success "Generated: $mcp_file ($count server(s))"
    else
        log_info "Skipping Copilot mcp.json (--no-mcp or no merged data)"
    fi

    log_success "Copilot setup complete"
}

# Setup Kilo: merge MCP into .kilo/kilo.jsonc (or kilo.json) under "mcp"
setup_kilo() {
    local target="$1"
    local merged_mcp="$2"
    local tool_dir="$target/.kilo"
    local kilo_cfg=""
    if [ -f "$tool_dir/kilo.jsonc" ]; then
        kilo_cfg="$tool_dir/kilo.jsonc"
    elif [ -f "$tool_dir/kilo.json" ]; then
        kilo_cfg="$tool_dir/kilo.json"
    else
        kilo_cfg="$tool_dir/kilo.jsonc"
    fi

    log_info "Setting up Kilocode configuration..."
    ensure_dir "$tool_dir"

    # Install KiloCode CLI if not already installed
    if ! command -v kilo &>/dev/null; then
        log_info "Installing KiloCode CLI (@kilocode/cli)..."
        if [ "$DRY_RUN" = true ]; then
            echo "[DRY-RUN] npm install -g @kilocode/cli"
        else
            if command -v npm &>/dev/null; then
                npm install -g @kilocode/cli
                if [ $? -eq 0 ]; then
                    log_success "KiloCode CLI installed successfully"
                else
                    log_warn "KiloCode CLI installation failed — you may need to install it manually: npm install -g @kilocode/cli"
                fi
            else
                log_warn "npm not found — skipping KiloCode CLI installation"
            fi
        fi
    else
        log_info "KiloCode CLI already installed"
    fi

    if [ "$MERGE_MCP" = true ] && [ -n "$merged_mcp" ]; then
        local mcp_obj tmp_kilo
        mcp_obj=$(printf '%s' "$merged_mcp" | mcp_emit_opencode_style_mcp_object)
        if [ -f "$kilo_cfg" ]; then
            tmp_kilo=$(mktemp)
            if jq --argjson mcp "$mcp_obj" '.mcp = $mcp' "$kilo_cfg" > "$tmp_kilo" 2>/dev/null; then
                mv "$tmp_kilo" "$kilo_cfg"
            else
                rm -f "$tmp_kilo"
                log_warn "Could not merge MCP into $kilo_cfg — replacing with MCP-only config"
                jq -n --argjson mcp "$mcp_obj" '{mcp: $mcp}' > "$kilo_cfg"
            fi
        else
            jq -n --argjson mcp "$mcp_obj" '{mcp: $mcp}' > "$kilo_cfg"
        fi
        local count
        count=$(jq '.mcp | keys | length' "$kilo_cfg")
        log_success "Wrote Kilo MCP: $kilo_cfg ($count server(s); OpenCode-style local/remote)"
        if [ -f "$tool_dir/mcp.json" ]; then
            log_info "Note: legacy $tool_dir/mcp.json exists — Kilo reads $kilo_cfg; remove mcp.json if unused"
        fi
    else
        log_info "Skipping Kilo MCP (--no-mcp or no merged data)"
    fi

    log_success "Kilocode setup complete"
}

# Setup OpenCode: copy .configs/opencode.json → .opencode/opencode.json, optional MCP merge, then mirror to project root opencode.json
setup_opencode() {
    local target="$1"
    local merged_mcp="$2"
    local tool_dir="$target/.opencode"
    local mcp_file="$tool_dir/opencode.json"
    local hub_opencode="$SCRIPT_DIR/.configs/opencode.json"
    local root_opencode="$target/opencode.json"

    ensure_dir "$tool_dir"

    if [ -f "$hub_opencode" ]; then
        cp "$hub_opencode" "$mcp_file"
        log_success "Copied hub config -> $mcp_file"
    fi

    if [ ! -f "$mcp_file" ]; then
        log_info "Skipping opencode: $mcp_file missing (add .configs/opencode.json to hub or .opencode/)"
        return 0
    fi

    if [ "$MERGE_MCP" = true ] && [ -n "$merged_mcp" ] && command -v jq &>/dev/null; then
        local tmp_file=$(mktemp)
        local opencode_mcp
        opencode_mcp=$(printf '%s' "$merged_mcp" | mcp_emit_opencode_style_mcp_object)

        if jq --argjson mcp "$opencode_mcp" '.mcp = $mcp' "$mcp_file" > "$tmp_file" 2>/dev/null; then
            mv "$tmp_file" "$mcp_file"
            local count
            count=$(jq '.mcp | keys | length' "$mcp_file")
            log_success "Updated MCP config: $mcp_file ($count server(s))"
        else
            rm -f "$tmp_file"
            log_warn "Could not update $mcp_file (jq failed)"
        fi
    elif [ "$MERGE_MCP" != true ] || [ -z "$merged_mcp" ]; then
        log_info "Skipping opencode MCP (--no-mcp or no merged data)"
    else
        log_warn "jq not found - cannot patch $mcp_file"
    fi

    cp "$mcp_file" "$root_opencode"
    log_success "Wrote project root opencode.json (same content as .opencode/opencode.json)"
}

# Setup Antigravity: write mcp_config.json directly
setup_antigravity() {
    local target="$1"
    local merged_mcp="$2"
    local mcp_dir="$HOME/.gemini/antigravity"
    local mcp_file="$mcp_dir/mcp_config.json"

    log_info "Setting up Google Antigravity configuration..."
    ensure_dir "$mcp_dir"

    if [ "$MERGE_MCP" = true ] && [ -n "$merged_mcp" ]; then
        printf '%s' "$merged_mcp" | mcp_emit_for_antigravity | jq '.' > "$mcp_file"
        local count
        count=$(jq '.mcpServers | keys | length' "$mcp_file")
        log_success "Generated: $mcp_file ($count server(s); Antigravity transport shape)"
    else
        log_info "Skipping Antigravity mcp_config.json (--no-mcp or no merged data)"
    fi

    log_success "Antigravity setup complete"
}

# SDD gate git hooks (inline — install via install.sh only; skillgrid-cli has TS parity).
_sdd_hook_git_hooks_dir() {
    local project="$1"
    local git_dir
    git_dir="$(git -C "$project" rev-parse --git-dir 2>/dev/null)" || return 1
    case "$git_dir" in
        /*) echo "${git_dir}/hooks" ;;
        *) echo "${project}/${git_dir}/hooks" ;;
    esac
}

_sdd_hook_contains_sdd_gate() {
    local f="$1"
    [ -f "$f" ] && grep -q "sdd-gate" "$f" 2>/dev/null
}

install_sdd_gate_hooks() {
    local project="$1"
    local gate_script hooks_dir

    [ "$INSTALL_SDD_HOOKS" = true ] || return 0

    if ! git -C "$project" rev-parse --git-dir &>/dev/null; then
        log_info "SDD gate hooks: skipped (not a git repository)"
        return 0
    fi

    gate_script="$project/.skillgrid/scripts/sdd-gate.sh"
    if [ ! -f "$gate_script" ]; then
        log_warn "SDD gate hooks: skipped — missing $gate_script"
        return 0
    fi

    if [ "$DRY_RUN" = true ]; then
        log_info "[DRY-RUN] Would install SDD gate hooks in $project"
        return 0
    fi

    hooks_dir="$(_sdd_hook_git_hooks_dir "$project")" || {
        log_warn "SDD gate hooks: could not resolve git hooks directory"
        return 0
    }

    log_info "Installing SDD gate git hooks..."
    chmod +x "$gate_script"
    mkdir -p "$hooks_dir"

    cat > "$hooks_dir/pre-commit" <<'SDD_PRE_COMMIT'
#!/usr/bin/env bash
# Pre-commit hook: run sdd-gate.sh on SDD changes when openspec/ changes are staged.

set -uo pipefail

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root" || exit 1

staged_files="$(git diff --cached --name-only 2>/dev/null || true)"
if [[ -z "$staged_files" ]]; then
  exit 0
fi

change_dirs=()
while IFS= read -r f; do
  [[ -z "$f" ]] && continue
  if [[ "$f" == openspec/changes/* ]] || [[ "$f" == .openspec/changes/* ]]; then
    dir="$(echo "$f" | cut -d'/' -f1-3)"
    change_dirs+=("$dir")
  fi
done <<< "$staged_files"

if [[ ${#change_dirs[@]} -eq 0 ]]; then
  exit 0
fi

unique_dirs=()
for d in "${change_dirs[@]}"; do
  found=false
  for u in "${unique_dirs[@]+"${unique_dirs[@]}"}"; do
    [[ "$u" == "$d" ]] && found=true && break
  done
  "$found" || unique_dirs+=("$d")
done

sdd_detect_phase() {
  local change_name="$1"
  local prefix="openspec/changes/${change_name}/"
  local staged
  staged="$(git diff --cached --name-only -- "${prefix}" 2>/dev/null || true)"
  if echo "$staged" | grep -qE 'tasks\.md$'; then
    echo "apply"
  elif echo "$staged" | grep -qE 'design\.md$'; then
    echo "design"
  elif echo "$staged" | grep -qE 'specs/.*/spec\.md$'; then
    echo "spec"
  elif echo "$staged" | grep -qE 'proposal\.md$'; then
    echo "propose"
  elif echo "$staged" | grep -qE 'ui-(wireframes|decisions)\.md$'; then
    echo "design"
  else
    echo "tasks"
  fi
}

errors=0
for d in "${unique_dirs[@]}"; do
  dir_name="$(basename "$d")"
  [[ "$dir_name" == "archive" ]] && continue
  phase="$(sdd_detect_phase "$dir_name")"
  echo "[sdd-gate] Running gate for: ${d} (phase=${phase})" >&2

  if ! "${repo_root}/.skillgrid/scripts/sdd-gate.sh" "$phase" --change "$dir_name" 2>&1; then
    echo "" >&2
    echo "=== sdd-gate PRE-COMMIT BLOCKED ===" >&2
    echo "Phase: $phase | Change: $dir_name" >&2
    echo "Fix gate failures before committing. Run manually:" >&2
    echo "  .skillgrid/scripts/sdd-gate.sh $phase --change $dir_name" >&2
    errors=1
  fi
done

exit $errors
SDD_PRE_COMMIT

    cat > "$hooks_dir/pre-push" <<'SDD_PRE_PUSH'
#!/usr/bin/env bash
# Pre-push hook: verify only SDD changes included in this push.

set -uo pipefail

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root" || exit 1
errors=0

change_names=()
sdd_collect_change_from_path() {
  local f="$1"
  local name=""
  if [[ "$f" == openspec/changes/*/* ]]; then
    name="$(echo "$f" | cut -d'/' -f3)"
  elif [[ "$f" == .openspec/changes/*/* ]]; then
    name="$(echo "$f" | cut -d'/' -f3)"
  else
    return 0
  fi
  [[ -z "$name" || "$name" == "archive" ]] && return 0
  local found=false
  for c in "${change_names[@]+"${change_names[@]}"}"; do
    [[ "$c" == "$name" ]] && found=true && break
  done
  "$found" || change_names+=("$name")
}

while read -r local_ref local_sha remote_ref remote_sha; do
  [[ -z "$local_sha" ]] && continue
  if [[ "$local_sha" == "0000000000000000000000000000000000000000" ]]; then
    continue
  fi
  if [[ "$remote_sha" == "0000000000000000000000000000000000000000" ]]; then
    range="$local_sha"
  else
    range="${remote_sha}..${local_sha}"
  fi
  while IFS= read -r f; do
    [[ -z "$f" ]] && continue
    sdd_collect_change_from_path "$f"
  done < <(git diff --name-only "$range" 2>/dev/null || true)
done

if [[ ${#change_names[@]} -eq 0 ]]; then
  exit 0
fi

for change_name in "${change_names[@]}"; do
  change_dir="$repo_root/openspec/changes/$change_name"
  [[ -f "${change_dir}/tasks.md" ]] || continue

  echo "[sdd-gate-pre-push] Checking: $change_name" >&2

  if ! "${repo_root}/.skillgrid/scripts/sdd-gate.sh" verify --change "$change_name" 2>&1; then
    echo "" >&2
    echo "=== sdd-gate PRE-PUSH BLOCKED ===" >&2
    echo "Change: $change_name" >&2
    echo "Resolve gate failures before pushing. Run manually:" >&2
    echo "  .skillgrid/scripts/sdd-gate.sh verify --change $change_name" >&2
    errors=1
  fi
done

exit $errors
SDD_PRE_PUSH

    chmod +x "$hooks_dir/pre-commit" "$hooks_dir/pre-push"
    log_success "SDD gate hooks installed ($hooks_dir/pre-commit, $hooks_dir/pre-push)"
}

uninstall_sdd_gate_hooks() {
    local project="$1"
    local hooks_dir

    [ "$INSTALL_SDD_HOOKS" = true ] || return 0

    if ! git -C "$project" rev-parse --git-dir &>/dev/null; then
        return 0
    fi

    if [ "$DRY_RUN" = true ]; then
        echo "[DRY-RUN] Would remove SDD gate git hooks from $project"
        return 0
    fi

    hooks_dir="$(_sdd_hook_git_hooks_dir "$project")" || return 0

    if _sdd_hook_contains_sdd_gate "$hooks_dir/pre-commit"; then
        rm -f "$hooks_dir/pre-commit"
        log_info "Removed $hooks_dir/pre-commit (sdd-gate)"
    fi
    if _sdd_hook_contains_sdd_gate "$hooks_dir/pre-push"; then
        rm -f "$hooks_dir/pre-push"
        log_info "Removed $hooks_dir/pre-push (sdd-gate)"
    fi
}

# Helper function to ensure directory exists
ensure_dir() {
    local dir="$1"
    if [ ! -d "$dir" ]; then
        if [ "$DRY_RUN" = true ]; then
            echo "[DRY-RUN] Would create directory: $dir"
        else
            mkdir -p "$dir"
        fi
    fi
}

# =============================================================================
# IDE ASSET SYNC: render hub commands/prompts/agents/rules to each selected IDE
# =============================================================================
# Inline rendering so install.sh is the single orchestrator for IDE setup.
# Enabled via --sync-assets / -s.

run_ide_asset_sync() {
    log_info "Syncing IDE assets — rendering commands, prompts, agents, and rules..."

    if [ "$DRY_RUN" = true ]; then
        log_info "[DRY-RUN] run_ide_asset_sync: targets = ${SELECTED_IDES[*]}, no files would be changed"
        return 0
    fi

    if ! command -v python3 &>/dev/null; then
        log_warn "IDE asset sync skipped: python3 not found (install Python 3 to enable asset sync)"
        return 0
    fi

    SRC_WORKFLOWS="$SCRIPT_DIR/.agents/workflows"

    # --- 1. Render commands from hub workflows → .cursor/commands/, .kilo/commands/, .opencode/commands/ ---
    # IDE command path mapping
    declare -A IDE_CMD_DIRS=(
        [cursor]="$PROJECT_PATH/.cursor/commands"
        [kilo]="$PROJECT_PATH/.kilo/commands"
        [opencode]="$PROJECT_PATH/.opencode/commands"
    )
    for ide in "${SELECTED_IDES[@]}"; do
        dest="${IDE_CMD_DIRS[$ide]}"
        [ -z "$dest" ] && continue
        mkdir -p "$dest"
        for src in "$SRC_WORKFLOWS"/*.md; do
            [ -f "$src" ] || continue
            python3 - "$src" "$dest" <<'PYINLINE'
import sys, re
from pathlib import Path

src = Path(sys.argv[1])
dest = Path(sys.argv[2])

def frontmatter(text):
    m = re.match(r"^---\n(.*?)\n---\n?", text, re.DOTALL)
    if not m:
        return {}, text
    fields = []
    for line in m.group(1).splitlines():
        if not line.strip() or ":" not in line:
            continue
        k, v = line.split(":", 1)
        fields.append((k.strip(), v.strip()))
    return {k: v for k, v in fields}, text[m.end():]

text = src.read_text(encoding="utf-8")
vals, body = frontmatter(text)
desc = vals.get("description", src.stem)

lines = [
    "---",
    f"name: {vals.get('id') or src.stem}",
    f"id: {vals.get('id') or src.stem}",
    f"category: {vals.get('category') or 'Workflow'}",
    f"description: {desc}",
]
for k, v in vals.items():
    if k not in {"name", "id", "category", "description"}:
        lines.append(f"{k}: {v}")
lines.append("---")
out = "\n".join(lines) + "\n" + body
(dest / src.name).write_text(out, encoding="utf-8")
PYINLINE
        done
        # prune stale mirrors
        for f in "$dest"/*.md; do
            [ -f "$f" ] || continue
            base="$(basename "$f")"
            [ -f "$SRC_WORKFLOWS/$base" ] || rm -f "$f"
        done
        log_success "Rendered commands → $dest"
    done

    # --- 2. Render Copilot prompts from hub workflows → .github/prompts/ ---
    if [[ " ${SELECTED_IDES[*]} " =~ " copilot " ]]; then
        gh_dest="$PROJECT_PATH/.github/prompts"
        mkdir -p "$gh_dest"
        for src in "$SRC_WORKFLOWS"/*.md; do
            [ -f "$src" ] || continue
            # extract frontmatter description
            desc="$(sed -n '1,/^---$/p' "$src" | grep '^description:' | head -1 | sed 's/^description:[[:space:]]*//')"
            [ -z "$desc" ] && desc="$(basename "$src" .md)"
            # strip frontmatter, keep body
            body="$(sed -e '1,/^---$/d' "$src")"
            out="---\ndescription: $desc\n---\n$body"
            echo "$out" > "$gh_dest/$(basename "$src" .md).prompt.md"
        done
        # prune stale mirrors
        for f in "$gh_dest"/*.prompt.md; do
            [ -f "$f" ] || continue
            sname="$(basename "$f" .prompt.md).md"
            [ -f "$SRC_WORKFLOWS/$sname" ] || rm -f "$f"
        done
        log_success "Rendered prompts → $gh_dest"
    fi

    # --- 3. Mirror agents from .cursor/agents → other IDE agent dirs ---
    if [ -d "$SCRIPT_DIR/.cursor/agents" ]; then
        AGENT_DEST_DIRS=(
            "$PROJECT_PATH/.github/agents"
            "$PROJECT_PATH/.kilo/agents"
            "$PROJECT_PATH/.opencode/agents"
        )
        for ad in "${AGENT_DEST_DIRS[@]}"; do
            mkdir -p "$ad"
            for f in "$SCRIPT_DIR/.cursor/agents"/*.md; do
                [ -f "$f" ] || continue
                cp "$f" "$ad/"
            done
            # prune stale
            for f in "$ad"/*.md; do
                [ -f "$f" ] || continue
                base="$(basename "$f")"
                [ -f "$SCRIPT_DIR/.cursor/agents/$base" ] || rm -f "$f"
            done
        done
        # prune from IDE dirs where the source file is deleted
        for target in "${AGENT_DEST_DIRS[@]}"; do
            for f in "$target"/*.md; do
                [ -f "$f" ] || continue
                base="$(basename "$f")"
                [ -f "$SCRIPT_DIR/.cursor/agents/$base" ] || rm -f "$f"
            done
        done
        log_success "Mirrored agents → .github/agents .kilo/agents .opencode/agents"
    fi

    # --- 4. Mirror rules from hub (.agents/rules + .cursor/rules) → .kilo/rules + .opencode/rules ---
    RULE_DEST_DIRS=("$PROJECT_PATH/.kilo/rules" "$PROJECT_PATH/.opencode/rules")
    for rule_dir in "${RULE_DEST_DIRS[@]}"; do
        mkdir -p "$rule_dir"
        for src in "$SCRIPT_DIR/.agents/rules"/*.mdc "$SCRIPT_DIR/.cursor/rules"/*.mdc; do
            [ -f "$src" ] || continue
            cp "$src" "$rule_dir/"
        done
        # prune stale
        for f in "$rule_dir"/*.mdc; do
            [ -f "$f" ] || continue
            base="$(basename "$f")"
            [ -f "$SCRIPT_DIR/.agents/rules/$base" ] || [ -f "$SCRIPT_DIR/.cursor/rules/$base" ] || rm -f "$f"
        done
    done
    log_success "Mirrored rules → .kilo/rules .opencode/rules"

    log_success "IDE asset sync complete"
}

# =============================================================================
# MAIN: orchestration — wiring, validation, execution
# =============================================================================

main() {
    # Items to exclude from copying (within each config folder)
    EXCLUDES=(
        ".git"
        ".gitignore"
        "docs"
        "configs"
    )

    # Build rsync exclude arguments
    RSYNC_EXCLUDES=""
    for exclude in "${EXCLUDES[@]}"; do
        RSYNC_EXCLUDES="$RSYNC_EXCLUDES --exclude '$exclude'"
    done

    # Uninstall mode
    if [ "$UNINSTALL" = true ]; then
        echo "Uninstalling AI config folders from: $PROJECT_PATH"
        echo ""

        # Remove managed IDE folders
        for ide in "${SELECTED_IDES[@]}"; do
            folder=$(ide_folder_for "$ide")
            target="$PROJECT_PATH/$folder"
            if [ -d "$target" ]; then
                if [ "$DRY_RUN" = true ]; then
                    echo "[DRY-RUN] Would remove: $folder"
                else
                    echo "Removing: $folder"
                    rm -rf "$target"
                fi
            else
                echo "Skipping: $folder (not found)"
            fi
        done

        # Root opencode.json (mirrors hub .configs/opencode.json via setup_opencode)
        if [[ " ${SELECTED_IDES[*]} " =~ " opencode " ]]; then
            target="$PROJECT_PATH/opencode.json"
            if [ -f "$target" ]; then
                if [ "$DRY_RUN" = true ]; then
                    echo "[DRY-RUN] Would remove: opencode.json (project root)"
                else
                    echo "Removing: opencode.json (project root)"
                    rm -f "$target"
                fi
            fi
        fi

        # Remove optional tool paths when selected via -t (same session)
        if tool_is_selected openspec; then
            target="$PROJECT_PATH/openspec"
            if [ -d "$target" ]; then
                if [ "$DRY_RUN" = true ]; then
                    echo "[DRY-RUN] Would remove: openspec"
                else
                    echo "Removing: openspec"
                    rm -rf "$target"
                fi
            fi
        fi

        if tool_is_selected gitnexus; then
            target="$PROJECT_PATH/.gitnexus"
            if [ -d "$target" ]; then
                if [ "$DRY_RUN" = true ]; then
                    echo "[DRY-RUN] Would remove: .gitnexus"
                else
                    echo "Removing: .gitnexus"
                    rm -rf "$target"
                fi
            fi
        fi

        # Remove .github / .copilot if copilot was selected
        if [[ " ${SELECTED_IDES[*]} " =~ " copilot " ]]; then
            for extra in ".github" ".copilot"; do
                target="$PROJECT_PATH/$extra"
                if [ -d "$target" ]; then
                    if [ "$DRY_RUN" = true ]; then
                        echo "[DRY-RUN] Would remove: $extra"
                    else
                        echo "Removing: $extra"
                        rm -rf "$target"
                    fi
                fi
            done
        fi

        uninstall_sdd_gate_hooks "$PROJECT_PATH"

        echo ""
        echo "Done!"
        exit 0
    fi

    # Check dependencies first (interactive prompt)
    if [ "$CHECK_DEPS" = false ]; then
        # Always check deps but don't auto-install
        echo "Checking dependencies..."
        echo ""

        missing_count=0
        for dep in "${CORE_DEPENDENCIES[@]}"; do
            IFS=':' read -r name brew pip npm check_cmd <<< "$dep"
            if [ -z "$check_cmd" ]; then continue; fi
            if ! eval "$check_cmd" &>/dev/null; then
                missing_count=$((missing_count + 1))
            fi
        done

        # Count missing IDE-specific dependencies
        for dep in "${IDE_DEPENDENCIES[@]}"; do
            IFS='|' read -r name brew pip npm check_cmd ide_flag <<< "$dep"

            # Skip if this IDE is not selected
            is_selected=false
            for selected in "${SELECTED_IDES[@]}"; do
                if [ "$selected" = "$ide_flag" ]; then
                    is_selected=true
                    break
                fi
            done

            # Special handling for openspec (optional tools)
            if [ "$ide_flag" = "openspec" ] && tool_is_selected openspec; then
                is_selected=true
            fi

            # Trivy CLI + MCP plugin when hub "trivy" MCP server is selected
            if { [ "$ide_flag" = "trivy" ] || [ "$ide_flag" = "trivy-mcp" ]; } && mcp_trivy_is_selected; then
                is_selected=true
            fi

            # Check if running with -A (all)
            if [ "$ALL_IDES" = true ]; then
                is_selected=true
            fi

            if [ "$is_selected" = false ]; then continue; fi
            if [ -z "$check_cmd" ]; then continue; fi

            if ! eval "$check_cmd" &>/dev/null; then
                missing_count=$((missing_count + 1))
            fi
        done

        if [ $missing_count -gt 0 ]; then
            echo "Found $missing_count missing dependencies."
            if [ "$NON_INTERACTIVE" = true ]; then
                echo "Continuing without installing dependencies (non-interactive mode)..."
                echo ""
            else
                read -p "Would you like to install missing dependencies? [y/N] " -n 1 -r
                echo ""
                if [[ $REPLY =~ ^[Yy]$ ]]; then
                    install_dependencies
                else
                    echo "Continuing without installing dependencies..."
                    echo ""
                fi
            fi
        else
            echo "All dependencies are installed!"
            echo ""
        fi
    fi

    # Optional tool CLIs (uv / hub npm / brew) — after dep prompt, before copying configs
    install_optional_tool_clis

    ensure_trivy_mcp_plugin

    # Dry run info
    if [ "$DRY_RUN" = true ]; then
        echo "=== DRY RUN MODE ==="
        echo ""
    fi

    echo "Installing AI config folders to: $PROJECT_PATH"
    echo ""

    # Copy selected IDE folders
    for ide in "${SELECTED_IDES[@]}"; do
        folder=$(ide_folder_for "$ide")
        src="$SCRIPT_DIR/$folder"
        dst="$PROJECT_PATH/$folder"

        if [ -d "$src" ]; then
            if [ -d "$dst" ]; then
                if [ "$DRY_RUN" = true ]; then
                    echo "[DRY-RUN] Would update: $folder"
                else
                    echo "Updating: $folder"
                    eval rsync -av --delete $RSYNC_EXCLUDES "$src/" "$dst/"
                fi
            else
                if [ "$DRY_RUN" = true ]; then
                    echo "[DRY-RUN] Would create: $folder"
                else
                    echo "Creating: $folder"
                    eval rsync -av --delete $RSYNC_EXCLUDES "$src/" "$dst/"
                fi
            fi
        else
            echo "Skipping: $folder (not found in source)"
        fi
    done

    # Copy hub .configs/AGENTS.md → project root and each selected IDE config dir
    HUB_AGENTS="$SCRIPT_DIR/.configs/AGENTS.md"
    if [ -f "$HUB_AGENTS" ] && [ ${#SELECTED_IDES[@]} -gt 0 ]; then
        echo ""
        if [ "$DRY_RUN" = true ]; then
            echo "[DRY-RUN] Would copy .configs/AGENTS.md -> $PROJECT_PATH/AGENTS.md"
            for ide in "${SELECTED_IDES[@]}"; do
                folder=$(ide_folder_for "$ide")
                echo "[DRY-RUN] Would copy .configs/AGENTS.md -> $PROJECT_PATH/$folder/AGENTS.md"
            done
        else
            echo "Copying AGENTS.md from hub (.configs/AGENTS.md)..."
            cp "$HUB_AGENTS" "$PROJECT_PATH/AGENTS.md"
            log_success "Wrote AGENTS.md (project root)"
            for ide in "${SELECTED_IDES[@]}"; do
                folder=$(ide_folder_for "$ide")
                dst_dir="$PROJECT_PATH/$folder"
                ensure_dir "$dst_dir"
                cp "$HUB_AGENTS" "$dst_dir/AGENTS.md"
                log_success "Wrote $folder/AGENTS.md"
            done
        fi
    elif [ ! -f "$HUB_AGENTS" ] && [ ${#SELECTED_IDES[@]} -gt 0 ]; then
        log_info ".configs/AGENTS.md not found in hub — skipping AGENTS.md copy"
    fi

    # Sync hub .skillgrid/ → project root (no --delete: preserve target prd/, preview/, etc.)
    HUB_SKILLGRID="$SCRIPT_DIR/.skillgrid"
    if [ -d "$HUB_SKILLGRID" ]; then
        echo ""
        dst_skillgrid="$PROJECT_PATH/.skillgrid"
        if [ "$DRY_RUN" = true ]; then
            if [ -d "$dst_skillgrid" ]; then
                echo "[DRY-RUN] Would update: .skillgrid"
            else
                echo "[DRY-RUN] Would create: .skillgrid"
            fi
        else
            if [ -d "$dst_skillgrid" ]; then
                echo "Updating: .skillgrid"
            else
                echo "Creating: .skillgrid"
            fi
            rsync -av "$HUB_SKILLGRID/" "$dst_skillgrid/"
            log_success "Synced .skillgrid/ (hub templates → project)"
        fi
    else
        log_info ".skillgrid/ not found in hub — skipping"
    fi

    # Sync hub .agents/ → project root — Skillgrid standard for all IDEs (rules, workflows, skills)
    # --delete: destination mirrors hub; files removed from hub disappear here (keep bespoke work outside .agents/)
    HUB_AGENTS_ROOT="$SCRIPT_DIR/.agents"
    if [ -d "$HUB_AGENTS_ROOT" ]; then
        echo ""
        dst_agents="$PROJECT_PATH/.agents"
        if [ "$DRY_RUN" = true ]; then
            if [ -d "$dst_agents" ]; then
                echo "[DRY-RUN] Would update: .agents (rules, workflows, skills)"
            else
                echo "[DRY-RUN] Would create: .agents (rules, workflows, skills)"
            fi
        else
            if [ -d "$dst_agents" ]; then
                echo "Updating: .agents (rules, workflows, skills)"
            else
                echo "Creating: .agents (rules, workflows, skills)"
            fi
            ensure_dir "$dst_agents"
            rsync -av --delete "$HUB_AGENTS_ROOT/" "$dst_agents/"
            log_success "Synced .agents/ (hub → project root)"
        fi
    else
        log_info ".agents/ not found in hub — skipping root .agents sync"
    fi

    # Sync hub .agent/ → project root only for Antigravity (and future Gemini; not implemented yet)
    # --delete: mirror hub fragment (removed hub files disappear)
    HUB_AGENT_SINGULAR="$SCRIPT_DIR/.agent"
    if [[ " ${SELECTED_IDES[*]} " =~ " antigravity " ]]; then
        if [ -d "$HUB_AGENT_SINGULAR" ]; then
            echo ""
            dst_agent="$PROJECT_PATH/.agent"
            if [ "$DRY_RUN" = true ]; then
                if [ -d "$dst_agent" ]; then
                    echo "[DRY-RUN] Would update: .agent (Antigravity / Gemini-style hub mirror)"
                else
                    echo "[DRY-RUN] Would create: .agent (Antigravity / Gemini-style hub mirror)"
                fi
            else
                if [ -d "$dst_agent" ]; then
                    echo "Updating: .agent (Antigravity / Gemini-style hub mirror)"
                else
                    echo "Creating: .agent (Antigravity / Gemini-style hub mirror)"
                fi
                ensure_dir "$dst_agent"
                rsync -av --delete "$HUB_AGENT_SINGULAR/" "$dst_agent/"
                log_success "Synced .agent/ (hub → project root)"
            fi
        else
            log_info ".agent/ not found in hub — skipping (Antigravity selected but no hub .agent/)"
        fi
    else
        if [ -d "$HUB_AGENT_SINGULAR" ]; then
            log_info ".agent/: skipped — hub has .agent/ but Antigravity is not selected (-a or -A); Gemini CLI TBD"
        fi
    fi

    # Sync .agents/skills/ to each IDE's skills directory
    echo ""
    echo "Syncing skills configurations..."
    SKILLS_SRC="$SCRIPT_DIR/.agents/skills"
    if [ -d "$SKILLS_SRC" ]; then
        for ide in "${SELECTED_IDES[@]}"; do
            case "$ide" in
                antigravity)
                    # Antigravity: sync .agents/skills/ → .agents/skills/
                    target="$PROJECT_PATH/.agents/skills"
                    if [ "$DRY_RUN" = true ]; then
                        echo "[DRY-RUN] Would sync skills to: .agents/skills"
                    else
                        ensure_dir "$target"
                        echo "Syncing skills to: .agents/skills"
                        rsync -av --delete "$SKILLS_SRC/" "$target/"
                    fi
                    ;;
                cursor)
                    # Cursor: sync .agents/skills/ → .cursor/.agents/skills/
                    target="$PROJECT_PATH/.cursor/.agents/skills"
                    if [ "$DRY_RUN" = true ]; then
                        echo "[DRY-RUN] Would sync skills to: .cursor/.agents/skills"
                    else
                        ensure_dir "$target"
                        echo "Syncing skills to: .cursor/.agents/skills"
                        rsync -av --delete "$SKILLS_SRC/" "$target/"
                    fi
                    ;;
                kilo)
                    # Kilo: sync .agents/skills/ → .kilo/skills/
                    target="$PROJECT_PATH/.kilo/skills"
                    if [ "$DRY_RUN" = true ]; then
                        echo "[DRY-RUN] Would sync skills to: .kilo/skills"
                    else
                        ensure_dir "$target"
                        echo "Syncing skills to: .kilo/skills"
                        rsync -av --delete "$SKILLS_SRC/" "$target/"
                    fi
                    ;;
                opencode)
                    # OpenCode: sync .agents/skills/ → .opencode/skills/
                    target="$PROJECT_PATH/.opencode/skills"
                    if [ "$DRY_RUN" = true ]; then
                        echo "[DRY-RUN] Would sync skills to: .opencode/skills"
                    else
                        ensure_dir "$target"
                        echo "Syncing skills to: .opencode/skills"
                        rsync -av --delete "$SKILLS_SRC/" "$target/"
                    fi
                    ;;
            esac
        done
    else
        log_info "Skills source not found: $SKILLS_SRC — skipping skills sync"
    fi

    # Sync .github/ and .copilot/ when Copilot is selected
    if [[ " ${SELECTED_IDES[*]} " =~ " copilot " ]]; then
        for extra in ".github" ".copilot"; do
            src="$SCRIPT_DIR/$extra"
            dst="$PROJECT_PATH/$extra"
            if [ -d "$src" ]; then
                if [ -d "$dst" ]; then
                    if [ "$DRY_RUN" = true ]; then
                        echo "[DRY-RUN] Would update: $extra"
                    else
                        echo "Updating: $extra"
                        rsync -av --delete "$src/" "$dst/"
                    fi
                else
                    if [ "$DRY_RUN" = true ]; then
                        echo "[DRY-RUN] Would create: $extra"
                    else
                        echo "Creating: $extra"
                        rsync -av --delete "$src/" "$dst/"
                    fi
                fi
            else
                echo "Skipping: $extra (not found in source)"
            fi
        done
    fi

    # Run IDE asset sync — render hub commands/prompts/agents/rules into selected IDE folders
    # Invoked when --sync-assets (-s) is passed.
    if [ "$SYNC_ASSETS" = true ]; then
        run_ide_asset_sync
    fi

    # SDD gate git hooks (requires .skillgrid/scripts/sdd-gate.sh in target)
    install_sdd_gate_hooks "$PROJECT_PATH"

    # Merge MCP configs
    echo ""
    echo "Merging MCP configurations..."
    MERGED_MCP=""
    if [ "$MERGE_MCP" = true ]; then
        MERGED_MCP=$(merge_mcp_configs)
    fi
    verify_engram_setup "$MERGED_MCP"

    # Setup selected IDEs
    echo ""
    echo "Setting up IDE configurations..."
    for ide in "${SELECTED_IDES[@]}"; do
        case "$ide" in
            cursor)
                if [ "$DRY_RUN" = true ]; then
                    echo "[DRY-RUN] Would setup: Cursor"
                else
                    setup_cursor "$PROJECT_PATH" "$MERGED_MCP"
                fi
                ;;
            copilot)
                if [ "$DRY_RUN" = true ]; then
                    echo "[DRY-RUN] Would setup: Copilot"
                else
                    setup_copilot "$PROJECT_PATH" "$MERGED_MCP"
                fi
                ;;
            kilo)
                if [ "$DRY_RUN" = true ]; then
                    echo "[DRY-RUN] Would setup: Kilocode"
                else
                    setup_kilo "$PROJECT_PATH" "$MERGED_MCP"
                fi
                ;;
            opencode)
                if [ "$DRY_RUN" = true ]; then
                    echo "[DRY-RUN] Would setup: OpenCode"
                else
                    setup_opencode "$PROJECT_PATH" "$MERGED_MCP"
                fi
                ;;
            antigravity)
                if [ "$DRY_RUN" = true ]; then
                    echo "[DRY-RUN] Would setup: Antigravity"
                else
                    setup_antigravity "$PROJECT_PATH" "$MERGED_MCP"
                fi
                ;;
        esac
    done

    # Optional tool: openspec (hub → project)
    if tool_is_selected openspec && [ -d "$SCRIPT_DIR/openspec" ]; then
        target="$PROJECT_PATH/openspec"
        if [ -d "$target" ]; then
            if [ "$DRY_RUN" = true ]; then
                echo "[DRY-RUN] Would update: openspec"
            else
                echo "Updating: openspec"
                eval rsync -av --delete $RSYNC_EXCLUDES "$SCRIPT_DIR/openspec/" "$target/"
            fi
        else
            if [ "$DRY_RUN" = true ]; then
                echo "[DRY-RUN] Would create: openspec"
            else
                echo "Creating: openspec"
                eval rsync -av --delete $RSYNC_EXCLUDES "$SCRIPT_DIR/openspec/" "$target/"
            fi
        fi
    elif tool_is_selected openspec && [ ! -d "$SCRIPT_DIR/openspec" ]; then
        log_info "openspec: hub has no openspec/ — skip copy"
    fi

    # Run openspec init when openspec tool selected
    if tool_is_selected openspec; then
        echo ""
        if [ "$DRY_RUN" = true ]; then
            echo "[DRY-RUN] Would run: openspec init in $PROJECT_PATH/openspec"
        else
            echo "Running openspec init..."
            if [ -d "$PROJECT_PATH/openspec" ]; then
                cd "$PROJECT_PATH/openspec" && openspec init
            fi
        fi
    fi

    echo ""
    if [ "$DRY_RUN" = true ]; then
        echo "=== DRY RUN COMPLETE ==="
        echo "(No changes were made)"
    else
        echo "Done! IDE config folders have been copied to $PROJECT_PATH"
    fi
}

# =============================================================================
# ENTRY POINT
# =============================================================================
main "$@"
