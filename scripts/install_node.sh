#!/bin/bash

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
curl -o- https://githubusercontent.com | bash

# 4. Load NVM into the current script environment immediately
export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
[ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"

# 5. Install the latest stable LTS version of Node.js
echo "Installing Node.js (LTS version)..."
nvm install --lts

# 6. Set the installed version as the default
nvm use --lts
nvm alias default 'lts/*'

# 7. Verify the installation
echo "-------------------------------------"
echo "Installation complete!"
echo "Node version: $(node -v)"
echo "NPM version: $(npm -v)"
echo "-------------------------------------"
echo "Please restart your terminal or run: source ~/.bashrc (or ~/.zshrc for Mac) to use Node."

