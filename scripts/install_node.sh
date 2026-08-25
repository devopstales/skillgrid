#!/bin/bash

NVM_VERSION="v0.40.1"
NODE_VERSION="24"

# 1. Detect the operating system
OS_TYPE="$(uname)"

echo "Detected OS: $OS_TYPE"

# 2. Install pre-requisites based on OS
if [ "$OS_TYPE" == "Darwin" ]; then
    echo "Configuring for macOS..."
    # Ensure Homebrew is installed or Xcode tools are ready if needed
    if ! command -v curl &> /dev/null; then
        echo "curl is missing. Please install Xcode Command Line Tools."
        exit 1
    fi
elif [ "$OS_TYPE" == "Linux" ]; then
    echo "Configuring for Ubuntu/Linux..."
    # Update system packages and install curl if missing
    sudo apt-get update -y
    sudo apt-get install -y curl build-essential
else
    echo "Unsupported Operating System."
    exit 1
fi

# 3. Download and install NVM (Node Version Manager)
echo "Installing NVM..."
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/$NVM_VERSION/install.sh | bash

# 3b. Ensure NVM is sourced in shell rc files
NVM_RC_LINES='export NVM_DIR="$HOME/.config/nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
[ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"'

USER_SHELL="$(basename "$SHELL")"
if [ "$USER_SHELL" = "zsh" ]; then
    RC_FILE="$HOME/.zshrc"
elif [ "$USER_SHELL" = "bash" ]; then
    RC_FILE="$HOME/.bashrc"
else
    RC_FILE="$HOME/.profile"
fi

if [ -f "$RC_FILE" ] && ! grep -q 'NVM_DIR="$HOME/.config/nvm"' "$RC_FILE"; then
    echo "" >> "$RC_FILE"
    echo "# Load NVM" >> "$RC_FILE"
    echo "$NVM_RC_LINES" >> "$RC_FILE"
    echo "Added NVM source lines to $RC_FILE"

    echo 'export PATH="$HOME/.skillgrid/bin:$PATH"'  >> "$RC_FILE"
    echo 'export PATH="$HOME/.skillgrid/npm/bin:$PATH"'  >> "$RC_FILE"
    echo "Added skillgrid source lines to $RC_FILE"
fi

# 4. Load NVM into the current script environment immediately
export NVM_DIR="$HOME/.config/nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
[ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"

# 5. Install Node.js
echo "Installing Node.js v$NODE_VERSION..."
nvm install $NODE_VERSION

# 6. Set Node.js as the default
nvm use $NODE_VERSION
nvm alias default $NODE_VERSION

# 7. Verify the installation
echo "-------------------------------------"
echo "Installation complete!"
echo "Node version: $(node -v)"
echo "NPM version: $(npm -v)"
echo "-------------------------------------"
echo "Please restart your terminal or run: source ~/.bashrc (or ~/.zshrc for Mac) to use Node."

