#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

REPO_URL="https://github.com/faradey/madock.git"
INSTALL_DIR="/usr/local/bin"
MADOCK_DIR="/opt/madock"
GO_VERSION="1.21.11"

echo -e "${GREEN}"
echo "  __  __   _   ____   ___   ____ _  __"
echo " |  \/  | / \ |  _ \ / _ \ / ___| |/ /"
echo " | |\/| |/ _ \| | | | | | | |   | ' / "
echo " | |  | / ___ \ |_| | |_| | |___| . \ "
echo " |_|  |_/_/   \_\____/\___/ \____|_|\_\\"
echo -e "${NC}"
echo "Madock Installer"
echo "================"
echo ""

# Function to print error and exit
error() {
    echo -e "${RED}Error: $1${NC}" >&2
    exit 1
}

# Function to print warning
warn() {
    echo -e "${YELLOW}Warning: $1${NC}"
}

# Function to print success
success() {
    echo -e "${GREEN}$1${NC}"
}

# Function to print info
info() {
    echo -e "${BLUE}$1${NC}"
}

# Check if running as root or can use sudo
check_root() {
    if [ "$EUID" -ne 0 ]; then
        if command -v sudo &> /dev/null; then
            SUDO="sudo"
        else
            error "This script requires root privileges. Please run as root or install sudo."
        fi
    else
        SUDO=""
    fi
}

# Detect OS
detect_os() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    case "$OS" in
        linux)
            OS="linux"
            # Detect Linux distribution
            if [ -f /etc/os-release ]; then
                . /etc/os-release
                DISTRO=$ID
            else
                DISTRO="unknown"
            fi
            ;;
        darwin)
            OS="darwin"
            DISTRO="macos"
            ;;
        *)
            error "Unsupported operating system: $OS"
            ;;
    esac
    echo "Detected OS: $OS ($DISTRO)"
}

# Detect architecture
detect_arch() {
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64|amd64)
            ARCH="amd64"
            GO_ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            GO_ARCH="arm64"
            ;;
        *)
            error "Unsupported architecture: $ARCH"
            ;;
    esac
    echo "Detected architecture: $ARCH"
}

# Check for required dependencies
check_dependencies() {
    echo ""
    echo "Checking dependencies..."

    # Check git
    if command -v git &> /dev/null; then
        success "  Git: installed"
    else
        error "Git is required but not installed. Please install git first."
    fi

    # Check Docker
    if command -v docker &> /dev/null; then
        success "  Docker: installed ($(docker --version | cut -d' ' -f3 | tr -d ','))"
    else
        warn "  Docker: not installed - required for madock to work"
        echo "    Install from: https://docs.docker.com/get-docker/"
    fi

    # Check Docker Compose
    if command -v docker-compose &> /dev/null; then
        success "  Docker Compose: installed"
    elif docker compose version &> /dev/null 2>&1; then
        success "  Docker Compose: installed (plugin)"
    else
        warn "  Docker Compose: not installed - required for madock to work"
        echo "    Install from: https://docs.docker.com/compose/install/"
    fi

    echo ""
}

# Install Go if not present
install_go() {
    echo "Checking Go installation..."

    if command -v go &> /dev/null; then
        GO_INSTALLED_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
        success "Go is already installed (version $GO_INSTALLED_VERSION)"

        # Check if version is sufficient (1.21+)
        GO_MAJOR=$(echo "$GO_INSTALLED_VERSION" | cut -d. -f1)
        GO_MINOR=$(echo "$GO_INSTALLED_VERSION" | cut -d. -f2)

        if [ "$GO_MAJOR" -lt 1 ] || ([ "$GO_MAJOR" -eq 1 ] && [ "$GO_MINOR" -lt 21 ]); then
            warn "Go version 1.21+ is recommended. Current version: $GO_INSTALLED_VERSION"
            info "Updating Go to version $GO_VERSION..."
            do_install_go
        fi
        return 0
    fi

    info "Go is not installed. Installing Go $GO_VERSION..."
    do_install_go
}

# Perform Go installation
do_install_go() {
    GO_TAR="go${GO_VERSION}.${OS}-${GO_ARCH}.tar.gz"
    GO_URL="https://go.dev/dl/${GO_TAR}"

    echo "Downloading Go from $GO_URL..."

    TMP_DIR=$(mktemp -d)
    cd "$TMP_DIR"

    # Download Go
    if command -v curl &> /dev/null; then
        curl -sLO "$GO_URL" || error "Failed to download Go"
    elif command -v wget &> /dev/null; then
        wget -q "$GO_URL" || error "Failed to download Go"
    else
        error "curl or wget is required to download Go"
    fi

    # Remove old Go installation if exists
    if [ -d /usr/local/go ]; then
        echo "Removing old Go installation..."
        $SUDO rm -rf /usr/local/go
    fi

    # Extract Go
    echo "Extracting Go..."
    $SUDO tar -C /usr/local -xzf "$GO_TAR"

    # Cleanup
    cd - > /dev/null
    rm -rf "$TMP_DIR"

    # Setup Go environment
    setup_go_env

    success "Go $GO_VERSION installed successfully!"
}

# Setup Go environment variables
setup_go_env() {
    GO_PATH_LINE='export PATH=$PATH:/usr/local/go/bin'

    # Determine shell config file
    if [ -n "$BASH_VERSION" ]; then
        SHELL_RC="$HOME/.bashrc"
    elif [ -n "$ZSH_VERSION" ]; then
        SHELL_RC="$HOME/.zshrc"
    else
        SHELL_RC="$HOME/.profile"
    fi

    # Check if already in PATH
    if ! echo "$PATH" | grep -q "/usr/local/go/bin"; then
        # Add to current session
        export PATH=$PATH:/usr/local/go/bin

        # Add to shell config if not already there
        if [ -f "$SHELL_RC" ]; then
            if ! grep -q "/usr/local/go/bin" "$SHELL_RC"; then
                echo "" >> "$SHELL_RC"
                echo "# Go" >> "$SHELL_RC"
                echo "$GO_PATH_LINE" >> "$SHELL_RC"
                info "Added Go to PATH in $SHELL_RC"
            fi
        else
            echo "$GO_PATH_LINE" >> "$SHELL_RC"
            info "Created $SHELL_RC with Go PATH"
        fi
    fi

    # Also add to /etc/profile.d for system-wide availability
    if [ -d /etc/profile.d ]; then
        echo "$GO_PATH_LINE" | $SUDO tee /etc/profile.d/go.sh > /dev/null
        $SUDO chmod +x /etc/profile.d/go.sh
    fi
}

# Clone or update madock repository
clone_or_update_repo() {
    echo ""
    echo "Setting up madock repository..."

    if [ -d "$MADOCK_DIR" ]; then
        info "Madock directory exists. Updating from master..."
        cd "$MADOCK_DIR"
        $SUDO git fetch origin
        $SUDO git checkout master
        $SUDO git pull origin master
    else
        info "Cloning madock repository..."
        $SUDO git clone "$REPO_URL" "$MADOCK_DIR"
        cd "$MADOCK_DIR"
    fi

    success "Repository ready at $MADOCK_DIR"
}

# Build madock
build_madock() {
    echo ""
    echo "Building madock..."

    cd "$MADOCK_DIR"

    # Set GOARCH based on architecture
    if [ "$ARCH" = "arm64" ]; then
        $SUDO env PATH="$PATH:/usr/local/go/bin" GOARCH=arm64 /usr/local/go/bin/go build -o madock
    else
        $SUDO env PATH="$PATH:/usr/local/go/bin" /usr/local/go/bin/go build -o madock
    fi

    if [ ! -f "$MADOCK_DIR/madock" ]; then
        error "Build failed. madock binary not created."
    fi

    success "Build completed!"
}

# Install madock binary
install_madock() {
    echo ""
    echo "Installing madock..."

    # Remove old symlink if exists
    if [ -L "${INSTALL_DIR}/madock" ]; then
        $SUDO rm "${INSTALL_DIR}/madock"
    elif [ -f "${INSTALL_DIR}/madock" ]; then
        $SUDO rm "${INSTALL_DIR}/madock"
    fi

    # Create symlink
    $SUDO ln -s "$MADOCK_DIR/madock" "${INSTALL_DIR}/madock"

    success "Symlink created: ${INSTALL_DIR}/madock -> $MADOCK_DIR/madock"
}

# Verify installation
verify_installation() {
    echo ""
    echo "Verifying installation..."

    # Make sure the path is available
    export PATH=$PATH:/usr/local/go/bin:${INSTALL_DIR}

    if [ -x "${INSTALL_DIR}/madock" ]; then
        success "madock is installed and executable"
        echo ""
        echo "========================================"
        success "Installation complete!"
        echo "========================================"
        echo ""
        echo "Quick start:"
        echo "  cd <your_project>"
        echo "  madock setup"
        echo "  madock start"
        echo ""
        echo "For more information, visit: https://github.com/faradey/madock"
        echo ""
        info "Note: You may need to open a new terminal or run 'source ~/.bashrc' for PATH changes to take effect."
    else
        error "Installation verification failed. madock is not executable."
    fi
}

# Main installation flow
main() {
    detect_os
    detect_arch
    check_root
    check_dependencies
    install_go
    clone_or_update_repo
    build_madock
    install_madock
    verify_installation
}

# Run main function
main