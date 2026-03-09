#!/bin/sh
set -e

REPO="mikolajbadyl/clauductor"
INSTALL_DIR="/usr/local/bin"
BINARY="clauductor"
DRY_RUN=false

# Colors
if [ -t 1 ]; then
    BOLD='\033[1m'
    DIM='\033[2m'
    CYAN='\033[36m'
    GREEN='\033[32m'
    YELLOW='\033[33m'
    RED='\033[31m'
    RESET='\033[0m'
else
    BOLD='' DIM='' CYAN='' GREEN='' YELLOW='' RED='' RESET=''
fi

info()    { printf "  ${DIM}%s${RESET}\n" "$1"; }
success() { printf "  ${GREEN}%s${RESET}\n" "$1"; }
warn()    { printf "  ${YELLOW}%s${RESET}\n" "$1"; }
err()     { printf "  ${RED}%s${RESET}\n" "$1"; }
step()    { printf "\n  ${BOLD}%s${RESET}\n" "$1"; }
run()     {
    if [ "$DRY_RUN" = true ]; then
        printf "  ${DIM}\$ %s${RESET}\n" "$*"
    else
        "$@"
    fi
}

for arg in "$@"; do
    case "$arg" in
        --dry-run) DRY_RUN=true ;;
    esac
done

header() {
    printf "  🚂 ${BOLD}Clauductor${RESET}\n"
    printf "  ${DIM}Your AI needs a conductor${RESET}\n"
    if [ "$DRY_RUN" = true ]; then
        printf "  ${YELLOW}DRY RUN — no changes will be made${RESET}\n"
    fi
}

detect_platform() {
    step "Detecting platform"

    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    arch=$(uname -m)

    case "$arch" in
        x86_64|amd64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *) err "Unsupported architecture: $arch" && exit 1 ;;
    esac

    case "$os" in
        linux|darwin) ;;
        *) err "Unsupported OS: $os (use install.ps1 for Windows)" && exit 1 ;;
    esac

    info "Platform: $os/$arch"

    pkg_manager="none"
    if [ "$os" = "linux" ]; then
        if command -v dpkg >/dev/null 2>&1 && command -v apt-get >/dev/null 2>&1; then
            pkg_manager="deb"
        elif command -v rpm >/dev/null 2>&1 && (command -v dnf >/dev/null 2>&1 || command -v yum >/dev/null 2>&1); then
            pkg_manager="rpm"
        fi
    fi

    if [ "$pkg_manager" != "none" ]; then
        info "Package manager: $pkg_manager"
    else
        info "Install method: binary"
    fi
}

fetch_version() {
    step "Fetching latest release"

    if [ "$DRY_RUN" = true ]; then
        tag="v1.0.0"
        info "Version: $tag (simulated)"
        return
    fi

    tag=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | head -1 | cut -d'"' -f4)
    if [ -z "$tag" ]; then
        err "Failed to fetch latest release"
        exit 1
    fi
    info "Version: $tag"
}

install_package() {
    step "Installing"

    case "$pkg_manager" in
        deb)
            url="https://github.com/${REPO}/releases/download/${tag}/${BINARY}_${tag#v}_linux_${arch}.deb"
            info "Downloading .deb package..."
            if [ "$DRY_RUN" = true ]; then
                run curl -fsSL "$url" -o /tmp/clauductor.deb
                run sudo dpkg -i /tmp/clauductor.deb
            else
                tmp=$(mktemp /tmp/clauductor-XXXXXX.deb)
                curl -fsSL "$url" -o "$tmp"
                info "Installing (requires sudo)..."
                sudo dpkg -i "$tmp"
                rm -f "$tmp"
            fi
            ;;
        rpm)
            url="https://github.com/${REPO}/releases/download/${tag}/${BINARY}_${tag#v}_linux_${arch}.rpm"
            info "Downloading .rpm package..."
            if [ "$DRY_RUN" = true ]; then
                run curl -fsSL "$url" -o /tmp/clauductor.rpm
                if command -v dnf >/dev/null 2>&1; then
                    run sudo dnf install -y /tmp/clauductor.rpm
                else
                    run sudo yum install -y /tmp/clauductor.rpm
                fi
            else
                tmp=$(mktemp /tmp/clauductor-XXXXXX.rpm)
                curl -fsSL "$url" -o "$tmp"
                info "Installing (requires sudo)..."
                if command -v dnf >/dev/null 2>&1; then
                    sudo dnf install -y "$tmp"
                else
                    sudo yum install -y "$tmp"
                fi
                rm -f "$tmp"
            fi
            ;;
        *)
            archive="${BINARY}-${os}-${arch}.tar.gz"
            url="https://github.com/${REPO}/releases/download/${tag}/${archive}"
            info "Downloading binary..."
            if [ "$DRY_RUN" = true ]; then
                run curl -fsSL "$url" -o "/tmp/${archive}"
                run tar -xzf "/tmp/${archive}" -C /tmp
                run sudo install -m 755 "/tmp/${BINARY}" "${INSTALL_DIR}/${BINARY}"
            else
                tmp=$(mktemp -d /tmp/clauductor-XXXXXX)
                curl -fsSL "$url" -o "${tmp}/${archive}"
                info "Extracting..."
                tar -xzf "${tmp}/${archive}" -C "$tmp"
                info "Installing to ${INSTALL_DIR} (requires sudo)..."
                sudo install -m 755 "${tmp}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
                rm -rf "$tmp"
            fi
            ;;
    esac

    success "Installed to ${INSTALL_DIR}/${BINARY}"
}

setup_mcp() {
    step "MCP Permission Server"

    bin_path=$(command -v clauductor 2>/dev/null || echo "${INSTALL_DIR}/${BINARY}")

    if command -v claude >/dev/null 2>&1; then
        info "Configuring MCP server via Claude CLI..."

        if [ "$DRY_RUN" = true ]; then
            run claude mcp remove clauductor-mcp -s user
            run claude mcp add --scope user clauductor-mcp -- "$bin_path" --mcp
            return
        fi

        claude mcp remove clauductor-mcp -s user 2>/dev/null || true
        claude mcp add --scope user clauductor-mcp -- "$bin_path" --mcp
        success "MCP server configured"
    else
        warn "Claude CLI not found, configuring manually..."
        config="$HOME/.claude.json"

        mcp_entry=$(cat <<EOF
{
  "type": "stdio",
  "command": "${bin_path}",
  "args": ["--mcp"]
}
EOF
)

        if [ "$DRY_RUN" = true ]; then
            info "Would add to $config:"
            printf "${DIM}"
            printf '  "clauductor-mcp": %s\n' "$mcp_entry"
            printf "${RESET}"
            return
        fi

        if [ -f "$config" ]; then
            if command -v jq >/dev/null 2>&1; then
                tmp_config=$(mktemp)
                jq --argjson mcp "$mcp_entry" '.mcpServers["clauductor-mcp"] = $mcp' "$config" > "$tmp_config"
                mv "$tmp_config" "$config"
                success "Added MCP server to $config"
            else
                warn "jq not found. Add this manually to $config under \"mcpServers\":"
                printf "\n${DIM}  \"clauductor-mcp\": %s${RESET}\n" "$mcp_entry"
            fi
        else
            cat > "$config" <<CONF
{
  "mcpServers": {
    "clauductor-mcp": $mcp_entry
  }
}
CONF
            success "Created $config with MCP server config"
        fi
    fi
}

summary() {
    printf "\n  ${GREEN}${BOLD}All aboard!${RESET}\n\n"
    printf "  ${BOLD}clauductor${RESET}              Start the server\n"
    printf "  ${BOLD}clauductor --help${RESET}       Show all options\n\n"
}

header
detect_platform
fetch_version
install_package
setup_mcp
summary
