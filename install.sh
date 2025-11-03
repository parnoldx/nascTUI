#!/bin/bash

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}"
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

GITHUB_REPO="parnoldx/nascTUI"

command_exists() { command -v "$1" >/dev/null 2>&1; }

detect_distro() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        echo "$ID"
    elif command_exists pacman; then
        echo "arch"
    elif command_exists apt; then
        echo "ubuntu"
    elif command_exists dnf; then
        echo "fedora"
    elif command_exists zypper; then
        echo "opensuse"
    else
        echo "unknown"
    fi
}

install_deps() {
    local distro=$(detect_distro)
    echo -e "${BLUE}Detected distribution: $distro${NC}"
    
    case $distro in
        arch)
            sudo pacman -S --needed go libqalculate pkgconf gcc git
            ;;
        ubuntu|debian)
            sudo apt update
            sudo apt install -y golang libqalculate-dev pkg-config gcc git
            ;;
        fedora)
            sudo dnf install -y golang libqalculate-devel pkgconfig gcc git
            ;;
        opensuse)
            sudo zypper install -y go libqalculate-devel pkg-config gcc git
            ;;
        *)
            echo -e "${RED}Unsupported distribution. Please install manually: go, libqalculate, gcc, git${NC}"
            exit 1
            ;;
    esac
}

check_deps() {
    local missing=()
    
    command_exists curl || missing+=("curl")
    command_exists go || missing+=("go")
    command_exists gcc || missing+=("gcc")
    command_exists git || missing+=("git")
    pkg-config --exists libqalculate || missing+=("libqalculate-dev")
    
    if [ ${#missing[@]} -gt 0 ]; then
        echo -e "${YELLOW}Missing dependencies: ${missing[*]}${NC}"
        echo -e "${BLUE}Installing dependencies...${NC}"
        install_deps
    else
        echo -e "${GREEN}✓ All dependencies found${NC}"
    fi
}

get_current_version() {
    if command_exists nasc; then
        nasc --version 2>/dev/null || echo "unknown"
    else
        echo "not_installed"
    fi
}

get_latest_version() {
    # Use GitHub API for reliable version detection
    curl -s "https://api.github.com/repos/$GITHUB_REPO/releases/latest" | \
    grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' | head -n1
}

version_compare() {
    local current=$1 latest=$2
    
    if [ "$current" = "not_installed" ]; then
        echo "install"
    elif [ "$current" = "$latest" ]; then
        echo "current"
    else
        echo "update"
    fi
}

install_binary() {
    local version=$1
    local os="linux"
    local arch=$(uname -m)
    
    case $arch in
        x86_64) arch="amd64" ;;
        *) echo -e "${RED}Unsupported architecture: $arch${NC}"; exit 1 ;;
    esac
    
    local binary_name="nasc-$os-$arch"
    local download_url="https://github.com/$GITHUB_REPO/releases/download/$version/$binary_name"
    local temp_dir=$(mktemp -d)
    
    echo -e "${BLUE}Downloading $binary_name...${NC}"
    curl -L "$download_url" -o "$temp_dir/nasc" || {
        echo -e "${YELLOW}Binary download failed, building from source...${NC}"
        build_from_source "$version"
        return
    }
    
    chmod +x "$temp_dir/nasc"
    
    if [ -w "/usr/local/bin" ] && [ "$EUID" -ne 0 ]; then
        sudo cp "$temp_dir/nasc" /usr/local/bin/nasc
    elif [ "$EUID" -eq 0 ]; then
        cp "$temp_dir/nasc" /usr/local/bin/nasc
    else
        mkdir -p "$HOME/.local/bin"
        cp "$temp_dir/nasc" "$HOME/.local/bin/nasc"
        
        if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
            echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.bashrc"
            echo -e "${YELLOW}Added $HOME/.local/bin to PATH in ~/.bashrc${NC}"
        fi
    fi
    
    rm -rf "$temp_dir"
    echo -e "${GREEN}✓ Installed nasc $version${NC}"
}

build_from_source() {
    local version=$1
    local temp_dir=$(mktemp -d)
    
    echo -e "${BLUE}Building from source...${NC}"
    git clone --depth 1 --branch "$version" "https://github.com/$GITHUB_REPO.git" "$temp_dir"
    cd "$temp_dir"
    
    g++ -c -std=c++11 $(pkg-config --cflags libqalculate) src/calc_wrapper.cpp -o src/calc_wrapper.o
    cd src && go build -ldflags "-X main.version=$version" -o ../nasc
    
    if [ -w "/usr/local/bin" ] && [ "$EUID" -ne 0 ]; then
        sudo cp ../nasc /usr/local/bin/nasc
    elif [ "$EUID" -eq 0 ]; then
        cp ../nasc /usr/local/bin/nasc
    else
        mkdir -p "$HOME/.local/bin"
        cp ../nasc "$HOME/.local/bin/nasc"
    fi
    
    cd / && rm -rf "$temp_dir"
    echo -e "${GREEN}✓ Built and installed nasc $version${NC}"
}

main() {
    check_deps
    
    local current_version=$(get_current_version)
    local latest_version=$(get_latest_version)
    
    if [ -z "$latest_version" ]; then
        echo -e "${RED}Failed to fetch latest version${NC}"
        exit 1
    fi
    
    echo -e "${BLUE}Current version: $current_version${NC}"
    echo -e "${BLUE}Latest version: $latest_version${NC}"
    
    local action=$(version_compare "$current_version" "$latest_version")
    
    case $action in
        install)
            echo -e "${BLUE}Installing nasc...${NC}"
            install_binary "$latest_version"
            ;;
        update)
            echo -e "${BLUE}Updating nasc...${NC}"
            install_binary "$latest_version"
            ;;
        current)
            echo -e "${GREEN}✓ nasc is up to date${NC}"
            exit 0
            ;;
    esac
    
    if command_exists nasc; then
        echo -e "${GREEN}🎉 nasc installation complete! Run 'nasc' to start.${NC}"
    else
        echo -e "${YELLOW}⚠ nasc not found in PATH. You may need to restart your shell.${NC}"
    fi
}

main "$@"