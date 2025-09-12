#!/bin/bash

# NaSC Installation Script
# Downloads pre-built binaries from GitHub releases

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# ASCII Art
echo -e "${CYAN}"
cat << "EOF"
  _   _        _____  _____ 
 | \ | |      / ____|/ ____|
 |  \| | __ _| (___ | |     
 | . ` |/ _` |\___ \| |     
 | |\  | (_| |____) | |____ 
 |_| \_|\__,_|_____/ \_____|
                           
Do maths like a normal person
EOF
echo -e "${NC}"

# Configuration
GITHUB_REPO="parnoldx/nascTUI"
VERSION=${NASC_VERSION:-"latest"}

# Get system information
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Normalize architecture names
case $ARCH in
    x86_64|amd64)
        ARCH="amd64"
        ;;
    *)
        echo -e "${RED}Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

# Only support Linux for now
if [ "$OS" != "linux" ]; then
    echo -e "${RED}This installer only supports Linux. Your OS: $OS${NC}"
    exit 1
fi

echo -e "${BLUE}Detected OS: $OS${NC}"
echo -e "${BLUE}Detected Architecture: $ARCH${NC}"

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check for wget or curl
if command_exists curl; then
    DOWNLOAD_CMD="curl -L"
    echo -e "${GREEN}✓ curl found${NC}"
elif command_exists wget; then
    DOWNLOAD_CMD="wget -O-"
    echo -e "${YELLOW}⚠ Using wget (curl recommended)${NC}"
else
    echo -e "${RED}✗ Neither curl nor wget found. Please install curl or wget.${NC}"
    exit 1
fi

if command_exists pkg-config && pkg-config --exists libqalculate >/dev/null 2>&1; then
    LIBQALC_VERSION=$(pkg-config --modversion libqalculate)
    echo -e "${GREEN}✓ libqalculate $LIBQALC_VERSION found${NC}"
else
    echo -e "${YELLOW}⚠ libqalculate not found${NC}"
    echo -e "${YELLOW}NaSC requires libqalculate to work properly.${NC}"
    echo
    echo -e "${CYAN}To install libqalculate on various distributions:${NC}"
    echo -e "${CYAN}Ubuntu/Debian: sudo apt install libqalculate22-dev${NC}"
    echo -e "${CYAN}Arch Linux:    sudo pacman -S libqalculate${NC}"
    echo -e "${CYAN}Fedora:        sudo dnf install libqalculate-devel${NC}"
    echo -e "${CYAN}openSUSE:      sudo zypper install libqalculate-devel${NC}"
    echo
fi
echo
# Determine installation directory
if [ -w "/usr/local/bin" ] && [ "$EUID" -ne 0 ]; then
    INSTALL_DIR="/usr/local/bin"
    NEEDS_SUDO="false"
elif [ "$EUID" -eq 0 ]; then
    INSTALL_DIR="/usr/local/bin"
    NEEDS_SUDO="false"
else
    INSTALL_DIR="$HOME/.local/bin"
    NEEDS_SUDO="false"
    mkdir -p "$INSTALL_DIR"
    
    # Check if ~/.local/bin is in PATH
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        echo -e "${YELLOW}⚠ $INSTALL_DIR is not in your PATH${NC}"
        echo -e "${YELLOW}Adding it to your shell configuration...${NC}"
        
        # Add to appropriate shell config
        if [ -n "$ZSH_VERSION" ] || [ "$SHELL" = "/bin/zsh" ] || [ "$SHELL" = "/usr/bin/zsh" ]; then
            echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.zshrc"
            echo -e "${GREEN}Added to ~/.zshrc${NC}"
        elif [ -n "$BASH_VERSION" ] || [ "$SHELL" = "/bin/bash" ] || [ "$SHELL" = "/usr/bin/bash" ]; then
            echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.bashrc"
            echo -e "${GREEN}Added to ~/.bashrc${NC}"
        else
            echo -e "${YELLOW}Please add $INSTALL_DIR to your PATH manually${NC}"
        fi
    fi
fi

echo -e "${BLUE}Installing to: $INSTALL_DIR${NC}"
echo

# Create temporary directory
TEMP_DIR=$(mktemp -d)
if [ $? -ne 0 ]; then
    echo -e "${RED}Failed to create temporary directory${NC}"
    exit 1
fi

# Cleanup function
cleanup() {
    rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

# Get latest release info or use specified version
if [ "$VERSION" = "latest" ]; then
    if command_exists curl; then
        LATEST_RELEASE=$(curl -s "https://api.github.com/repos/$GITHUB_REPO/releases/latest")
    else
        LATEST_RELEASE=$(wget -qO- "https://api.github.com/repos/$GITHUB_REPO/releases/latest")
    fi
    
    VERSION=$(echo "$LATEST_RELEASE" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' | head -n1)
    if [ -z "$VERSION" ]; then
        echo -e "${RED}Failed to get latest release version${NC}"
        exit 1
    fi
fi

echo -e "${BLUE}Installing NaSC version: $VERSION${NC}"

# Construct download URL
BINARY_NAME="nasc-$OS-$ARCH"
DOWNLOAD_URL="https://github.com/$GITHUB_REPO/releases/download/$VERSION/$BINARY_NAME"

# Download binary
cd "$TEMP_DIR"
if command_exists curl; then
    curl -L "$DOWNLOAD_URL" -o nasc
else
    wget "$DOWNLOAD_URL" -O nasc
fi

if [ $? -ne 0 ]; then
    echo -e "${RED}Failed to download binary${NC}"
    echo -e "${YELLOW}Please check if the release exists for your platform${NC}"
    echo -e "${YELLOW}Available at: https://github.com/$GITHUB_REPO/releases${NC}"
    exit 1
fi

# Make executable
chmod +x nasc

if [ "$INSTALL_DIR" = "/usr/local/bin" ] && [ "$EUID" -ne 0 ]; then
    if sudo cp nasc "$INSTALL_DIR/nasc"; then
        echo -e "${GREEN}✓ Installed to $INSTALL_DIR${NC}"
    else
        echo -e "${YELLOW}Sudo failed, installing to user directory...${NC}"
        INSTALL_DIR="$HOME/.local/bin"
        mkdir -p "$INSTALL_DIR"
        cp nasc "$INSTALL_DIR/nasc"
        echo -e "${GREEN}✓ Installed to $INSTALL_DIR${NC}"
    fi
else
    cp nasc "$INSTALL_DIR/nasc"
    echo -e "${GREEN}✓ Installed to $INSTALL_DIR${NC}"
fi

# Test if command is available
if command_exists nasc; then
    echo
    echo -e "${GREEN}🎉 NaSC installation complete!${NC}"
    echo
    echo -e "${CYAN}Usage:${NC}"
    echo -e "${CYAN}  nasc${NC}"
else
    echo -e "${YELLOW}⚠ nasc command not found in PATH${NC}"
    echo -e "${YELLOW}You may need to restart your shell or run:${NC}"
    echo -e "${YELLOW}  source ~/.bashrc  # or ~/.zshrc${NC}"
fi


