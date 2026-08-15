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
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" 2>/dev/null && pwd || true)"

command_exists() { command -v "$1" >/dev/null 2>&1; }

# Map /etc/os-release (and package managers) onto a package family.
# Omarchy, Manjaro, EndeavourOS, CachyOS, etc. all report ID_LIKE=arch
# (or ship pacman), so they must not fall through as "unsupported".
detect_family() {
    local id="" like=""
    if [ -f /etc/os-release ]; then
        # shellcheck disable=SC1091
        . /etc/os-release
        id="${ID:-}"
        like="${ID_LIKE:-}"
    fi

    case "$id" in
        arch|omarchy|manjaro|endeavouros|cachyos|garuda|artix|archlinux|archcraft)
            echo "arch"; return ;;
        ubuntu|debian|linuxmint|pop|elementary|raspbian|kali)
            echo "debian"; return ;;
        fedora|rhel|centos|rocky|almalinux|nobara)
            echo "fedora"; return ;;
        opensuse*|suse*|sles)
            echo "opensuse"; return ;;
    esac

    case " $like " in
        *" arch "*) echo "arch"; return ;;
        *" debian "*|*" ubuntu "*) echo "debian"; return ;;
        *" fedora "*|*" rhel "*) echo "fedora"; return ;;
        *" suse "*) echo "opensuse"; return ;;
    esac

    if command_exists pacman; then
        echo "arch"
    elif command_exists apt-get || command_exists apt; then
        echo "debian"
    elif command_exists dnf; then
        echo "fedora"
    elif command_exists zypper; then
        echo "opensuse"
    else
        echo "unknown"
    fi
}

distro_id() {
    if [ -f /etc/os-release ]; then
        # shellcheck disable=SC1091
        . /etc/os-release
        echo "${ID:-unknown}"
    else
        echo "unknown"
    fi
}

run_root() {
    if [ "${EUID:-$(id -u)}" -eq 0 ]; then
        "$@"
    elif command_exists sudo; then
        sudo "$@"
    elif command_exists pkexec; then
        pkexec "$@"
    else
        echo -e "${RED}Need root to install packages. Re-run as root or install sudo.${NC}"
        exit 1
    fi
}

install_packages() {
    local family
    family=$(detect_family)
    local id
    id=$(distro_id)

    echo -e "${BLUE}Detected distribution: $id ($family)${NC}"

    case $family in
        arch)
            # Prefer Omarchy's package helper when present; it wraps pacman.
            if command_exists omarchy && command_exists omarchy-pkg-add; then
                omarchy pkg add "$@"
            else
                run_root pacman -S --needed --noconfirm "$@"
            fi
            ;;
        debian)
            run_root apt-get update
            run_root apt-get install -y "$@"
            ;;
        fedora)
            run_root dnf install -y "$@"
            ;;
        opensuse)
            run_root zypper install -y "$@"
            ;;
        *)
            echo -e "${RED}Unsupported distribution ($id). Please install manually: $*${NC}"
            exit 1
            ;;
    esac
}

# Runtime: libqalculate. Build tools only when we have to compile.
install_runtime_deps() {
    local family
    family=$(detect_family)

    case $family in
        arch) install_packages libqalculate ;;
        debian) install_packages libqalculate-dev pkg-config ;;
        fedora) install_packages libqalculate-devel pkgconfig ;;
        opensuse) install_packages libqalculate-devel pkg-config ;;
        *) install_packages libqalculate ;;
    esac
}

install_build_deps() {
    local family
    family=$(detect_family)

    case $family in
        arch) install_packages go libqalculate pkgconf gcc git ;;
        debian) install_packages golang libqalculate-dev pkg-config g++ git ;;
        fedora) install_packages golang libqalculate-devel pkgconfig gcc-c++ git ;;
        opensuse) install_packages go libqalculate-devel pkg-config gcc-c++ git ;;
        *) install_packages go libqalculate gcc git ;;
    esac
}

have_libqalculate() {
    if command_exists pkg-config && pkg-config --exists libqalculate; then
        return 0
    fi
    # Runtime-only systems may ship the .so without pkg-config metadata.
    ldconfig -p 2>/dev/null | grep -q 'libqalculate\.so' && return 0
    return 1
}

ensure_runtime_deps() {
    local missing=()

    command_exists curl || missing+=("curl")
    have_libqalculate || missing+=("libqalculate")

    if [ ${#missing[@]} -gt 0 ]; then
        echo -e "${YELLOW}Missing runtime dependencies: ${missing[*]}${NC}"
        echo -e "${BLUE}Installing dependencies...${NC}"
        install_runtime_deps
        command_exists curl || install_packages curl
    else
        echo -e "${GREEN}✓ Runtime dependencies found${NC}"
    fi
}

ensure_build_deps() {
    local missing=()

    command_exists go || missing+=("go")
    command_exists g++ || command_exists gcc || missing+=("g++")
    command_exists git || missing+=("git")
    command_exists pkg-config || missing+=("pkg-config")
    have_libqalculate || missing+=("libqalculate")

    if [ ${#missing[@]} -gt 0 ]; then
        echo -e "${YELLOW}Missing build dependencies: ${missing[*]}${NC}"
        echo -e "${BLUE}Installing build dependencies...${NC}"
        install_build_deps
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
    curl -fsSL "https://api.github.com/repos/$GITHUB_REPO/releases/latest" |
        grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' | head -n1
}

map_arch() {
    case "$(uname -m)" in
        x86_64|amd64) echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *) echo "" ;;
    esac
}

is_elf() {
    [ "$(od -An -N4 -tx1 "$1" 2>/dev/null | tr -d ' \n')" = "7f454c46" ]
}

install_path() {
    if [ -n "${PREFIX:-}" ]; then
        mkdir -p "$PREFIX/bin"
        echo "$PREFIX/bin/nasc"
        return
    fi

    if [ "${EUID:-$(id -u)}" -eq 0 ]; then
        echo "/usr/local/bin/nasc"
        return
    fi

    if [ -w /usr/local/bin ] 2>/dev/null; then
        echo "/usr/local/bin/nasc"
        return
    fi

    mkdir -p "$HOME/.local/bin"
    echo "$HOME/.local/bin/nasc"
}

place_binary() {
    local src=$1
    local dest
    dest=$(install_path)

    chmod +x "$src"
    if [ -w "$(dirname "$dest")" ]; then
        install -m 755 "$src" "$dest"
    else
        run_root install -m 755 "$src" "$dest"
    fi

    if [ "$(dirname "$dest")" = "$HOME/.local/bin" ] && [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
        if [ -f "$HOME/.bashrc" ] && ! grep -q 'export PATH="$HOME/.local/bin:$PATH"' "$HOME/.bashrc"; then
            echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.bashrc"
            echo -e "${YELLOW}Added $HOME/.local/bin to PATH in ~/.bashrc${NC}"
        fi
    fi

    echo "$dest"
}

local_source_dir() {
    if [ -n "${SCRIPT_DIR:-}" ] && [ -f "$SCRIPT_DIR/src/calculator.go" ] && [ -f "$SCRIPT_DIR/src/calc_wrapper.cpp" ]; then
        echo "$SCRIPT_DIR"
    fi
}

build_in_dir() {
    local dir=$1
    local version=$2

    (
        cd "$dir"
        if [ -f Makefile ]; then
            make VERSION="${version:-dev}"
        else
            cd src
            go build -ldflags "-X main.version=${version:-dev}" -o ../nasc
        fi
    )
}

build_from_source() {
    local version=$1
    local local_dir
    local_dir=$(local_source_dir)

    ensure_build_deps

    if [ -n "$local_dir" ]; then
        echo -e "${BLUE}Building from local source ($local_dir)...${NC}"
        build_in_dir "$local_dir" "${version:-dev}"
        local dest
        dest=$(place_binary "$local_dir/nasc")
        echo -e "${GREEN}✓ Built and installed nasc ${version:-dev} → $dest${NC}"
        return
    fi

    local temp_dir
    temp_dir=$(mktemp -d)
    trap 'rm -rf "$temp_dir"' RETURN

    echo -e "${BLUE}Building from source...${NC}"
    if [ -n "$version" ] && git clone --depth 1 --branch "$version" "https://github.com/$GITHUB_REPO.git" "$temp_dir/src" 2>/dev/null; then
        :
    else
        git clone --depth 1 "https://github.com/$GITHUB_REPO.git" "$temp_dir/src"
    fi

    build_in_dir "$temp_dir/src" "${version:-dev}"
    local dest
    dest=$(place_binary "$temp_dir/src/nasc")
    echo -e "${GREEN}✓ Built and installed nasc ${version:-dev} → $dest${NC}"
}

install_binary() {
    local version=$1
    local arch
    arch=$(map_arch)

    if [ -z "$arch" ]; then
        echo -e "${YELLOW}No prebuilt binary for $(uname -m), building from source...${NC}"
        build_from_source "$version"
        return
    fi

    local binary_name="nasc-linux-$arch"
    local download_url="https://github.com/$GITHUB_REPO/releases/download/$version/$binary_name"
    local temp_dir
    temp_dir=$(mktemp -d)

    echo -e "${BLUE}Downloading $binary_name...${NC}"
    if ! curl -fSL "$download_url" -o "$temp_dir/nasc" || ! is_elf "$temp_dir/nasc"; then
        rm -rf "$temp_dir"
        echo -e "${YELLOW}Binary download failed, building from source...${NC}"
        build_from_source "$version"
        return
    fi

    local dest
    dest=$(place_binary "$temp_dir/nasc")
    rm -rf "$temp_dir"
    echo -e "${GREEN}✓ Installed nasc $version → $dest${NC}"
}

main() {
    ensure_runtime_deps

    local current_version
    current_version=$(get_current_version)
    local latest_version
    latest_version=$(get_latest_version || true)
    local local_dir
    local_dir=$(local_source_dir)

    echo -e "${BLUE}Current version: $current_version${NC}"

    if [ -z "$latest_version" ]; then
        echo -e "${YELLOW}Could not fetch latest release, building from source...${NC}"
        build_from_source ""
    elif [ "$current_version" = "$latest_version" ] && [ -z "$local_dir" ]; then
        echo -e "${GREEN}✓ nasc is up to date${NC}"
        exit 0
    elif [ -n "$local_dir" ] && [ "$current_version" != "not_installed" ] && [ "${1:-}" != "--force" ]; then
        # Running from a git checkout: rebuild so local fixes land.
        echo -e "${BLUE}Source checkout detected, rebuilding...${NC}"
        build_from_source "${latest_version:-dev}"
    elif [ "$current_version" = "not_installed" ]; then
        echo -e "${BLUE}Installing nasc $latest_version...${NC}"
        install_binary "$latest_version"
    else
        echo -e "${BLUE}Updating nasc $current_version → $latest_version...${NC}"
        install_binary "$latest_version"
    fi

    hash -r 2>/dev/null || true
    if command_exists nasc; then
        echo -e "${GREEN}🎉 nasc $(nasc --version 2>/dev/null || echo installed)! Run 'nasc' to start.${NC}"
    else
        echo -e "${YELLOW}⚠ nasc not found in PATH. You may need to restart your shell.${NC}"
        echo -e "${YELLOW}  Or run: export PATH=\"\$HOME/.local/bin:\$PATH\"${NC}"
    fi
}

main "$@"
