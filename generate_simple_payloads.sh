#!/bin/bash

# Simple Payload Generator Script for APA Agent
# Creates basic payloads to execute the agent on various devices

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$SCRIPT_DIR"
PAYLOADS_DIR="$PROJECT_ROOT/payloads"
BUILD_DIR="$PROJECT_ROOT/build"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_debug() {
    echo -e "${BLUE}[DEBUG]${NC} $1"
}

# Create directories
mkdir -p "$PAYLOADS_DIR"
mkdir -p "$BUILD_DIR"

# Function to build basic binaries for common platforms (no root required)
build_basic_binaries() {
    log_info "Building basic binaries for common platforms..."
    
    # Platforms to build for
    PLATFORMS=(
        "linux/amd64"
        "linux/arm64"
        "darwin/amd64"
        "darwin/arm64"
        "windows/amd64"
    )
    
    for platform in "${PLATFORMS[@]}"; do
        OS=$(echo "$platform" | cut -d'/' -f1)
        ARCH=$(echo "$platform" | cut -d'/' -f2)
        
        log_info "Building for $OS/$ARCH..."
        
        # Set output filename
        if [ "$OS" = "windows" ]; then
            OUTPUT_NAME="agentd-$OS-$ARCH.exe"
        else
            OUTPUT_NAME="agentd-$OS-$ARCH"
        fi
        
        # Build the binary (simplified version)
        CGO_ENABLED=0 GOOS="$OS" GOARCH="$ARCH" \
            go build -v -ldflags="-s -w" -o "$PAYLOADS_DIR/$OUTPUT_NAME" "$PROJECT_ROOT/cmd/agentd"
        
        log_info "Built binary for $OS/$ARCH: $OUTPUT_NAME"
    done
    
    log_info "Basic binaries built successfully"
}

# Function to create simple shell payload for Unix-like systems
create_simple_shell_payload() {
    log_info "Creating simple shell payload for Unix-like systems..."
    
    SH_PAYLOAD_DIR="$PAYLOADS_DIR/shell"
    mkdir -p "$SH_PAYLOAD_DIR"
    
    # Create a simple shell script that downloads and executes the agent
    cat > "$SH_PAYLOAD_DIR/simple-install.sh" << 'EOF'
#!/bin/bash

# Simple Shell Payload for APA Agent
# This script downloads and runs the APA agent on Unix-like systems

set -e

# Configuration
DEFAULT_URL="http://example.com/agentd-linux-amd64"
DEFAULT_INSTALL_PATH="/tmp/apa"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Detect system architecture
detect_architecture() {
    local arch=$(uname -m)
    case $arch in
        x86_64)
            echo "amd64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        armv7l|arm)
            echo "arm"
            ;;
        *)
            echo "$arch"
            ;;
    esac
}

# Detect operating system
detect_os() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    case $os in
        linux)
            echo "linux"
            ;;
        darwin)
            echo "darwin"
            ;;
        *)
            echo "$os"
            ;;
    esac
}

# Download file
download_file() {
    local url=$1
    local dest=$2
    
    log_info "Downloading $url to $dest"
    
    if command -v curl >/dev/null 2>&1; then
        curl -L -o "$dest" "$url"
    elif command -v wget >/dev/null 2>&1; then
        wget -O "$dest" "$url"
    else
        log_error "Neither curl nor wget is available"
        exit 1
    fi
}

# Main function
main() {
    local url=${1:-$DEFAULT_URL}
    local install_path=${2:-$DEFAULT_INSTALL_PATH}
    
    log_info "Starting APA Agent installation..."
    
    # Detect platform
    local os=$(detect_os)
    local arch=$(detect_architecture)
    
    log_info "Detected platform: $os/$arch"
    
    # Adjust URL based on detected platform
    case "$os/$arch" in
        linux/amd64)
            url="http://example.com/agentd-linux-amd64"
            ;;
        linux/arm64)
            url="http://example.com/agentd-linux-arm64"
            ;;
        darwin/amd64)
            url="http://example.com/agentd-darwin-amd64"
            ;;
        darwin/arm64)
            url="http://example.com/agentd-darwin-arm64"
            ;;
        *)
            log_warn "Unsupported platform: $os/$arch, using default URL"
            ;;
    esac
    
    # Create installation directory
    mkdir -p "$install_path"
    
    # Determine binary name
    local binary_name="agentd-$os-$arch"
    if [[ "$os" == "windows" ]]; then
        binary_name="${binary_name}.exe"
    fi
    
    local binary_path="$install_path/$binary_name"
    
    # Download the agent
    download_file "$url" "$binary_path"
    
    # Make binary executable
    chmod +x "$binary_path"
    
    log_info "Agent downloaded to: $binary_path"
    log_info "You can now run the agent with: $binary_path"
}

# Run main function with arguments
main "$@"
EOF

    # Make the shell script executable
    chmod +x "$SH_PAYLOAD_DIR/simple-install.sh"
    
    log_info "Simple shell payload created successfully"
}

# Function to create simple PowerShell payload for Windows
create_simple_powershell_payload() {
    log_info "Creating simple PowerShell payload for Windows..."
    
    PS_PAYLOAD_DIR="$PAYLOADS_DIR/powershell"
    mkdir -p "$PS_PAYLOAD_DIR"
    
    # Create a simple PowerShell script that downloads and executes the agent
    cat > "$PS_PAYLOAD_DIR/simple-install.ps1" << 'EOF'
# Simple PowerShell Payload for APA Agent
# This script downloads and runs the APA agent on Windows systems

param(
    [string]$DownloadURL = "http://example.com/agentd-windows-amd64.exe",
    [string]$InstallPath = "$env:TEMP\apa"
)

function Write-Log {
    param([string]$Message)
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    Write-Host "[$timestamp] $Message"
}

try {
    Write-Log "Starting APA Agent installation..."
    
    # Create installation directory
    if (!(Test-Path $InstallPath)) {
        New-Item -ItemType Directory -Path $InstallPath | Out-Null
        Write-Log "Created installation directory: $InstallPath"
    }
    
    # Determine binary name based on architecture
    $arch = (Get-WmiObject Win32_Processor).AddressWidth
    if ($arch -eq 64) {
        $binaryName = "agentd-windows-amd64.exe"
    } else {
        $binaryName = "agentd-windows-386.exe"
    }
    
    # Update download URL based on architecture
    $DownloadURL = "http://example.com/$binaryName"
    
    # Download the agent
    Write-Log "Downloading APA agent from: $DownloadURL"
    $binaryPath = "$InstallPath\$binaryName"
    Invoke-WebRequest -Uri $DownloadURL -OutFile $binaryPath
    
    Write-Log "Agent downloaded to: $binaryPath"
    Write-Log "You can now run the agent with: $binaryPath"
    
} catch {
    Write-Log "Installation failed: $($_.Exception.Message)"
    exit 1
}
EOF

    log_info "Simple PowerShell payload created successfully"
}

# Function to create a simple Python payload
create_simple_python_payload() {
    log_info "Creating simple Python payload..."
    
    PY_PAYLOAD_DIR="$PAYLOADS_DIR/python"
    mkdir -p "$PY_PAYLOAD_DIR"
    
    # Create a simple Python script that downloads and executes the agent
    cat > "$PY_PAYLOAD_DIR/simple-install.py" << 'EOF'
#!/usr/bin/env python3

"""
Simple Python Payload for APA Agent
This script downloads and runs the APA agent on various platforms
"""

import os
import sys
import platform
import urllib.request
import tempfile
import subprocess

def log(message):
    """Print a log message with timestamp"""
    import datetime
    timestamp = datetime.datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    print(f"[{timestamp}] {message}")

def get_platform_info():
    """Get platform-specific information"""
    system = platform.system().lower()
    machine = platform.machine().lower()
    
    # Normalize architecture names
    arch_map = {
        'x86_64': 'amd64',
        'x86': '386',
        'i386': '386',
        'i686': '386',
        'aarch64': 'arm64',
        'armv7l': 'arm'
    }
    
    arch = arch_map.get(machine, machine)
    
    return system, arch

def download_file(url, destination):
    """Download a file from URL to destination"""
    log(f"Downloading {url} to {destination}")
    urllib.request.urlretrieve(url, destination)

def main():
    log("Starting APA Agent installation...")
    
    # Get platform information
    system, arch = get_platform_info()
    log(f"Detected platform: {system}/{arch}")
    
    # Determine download URL based on platform
    if system == "windows":
        binary_name = f"agentd-{system}-{arch}.exe"
    else:
        binary_name = f"agentd-{system}-{arch}"
    
    download_url = f"http://example.com/{binary_name}"
    
    # Create temporary directory for download
    with tempfile.TemporaryDirectory() as temp_dir:
        binary_path = os.path.join(temp_dir, binary_name)
        
        # Download the agent package
        download_file(download_url, binary_path)
        
        # Make binary executable (Unix-like systems)
        if system != "windows":
            os.chmod(binary_path, 0o755)
        
        log(f"Agent downloaded to: {binary_path}")
        log(f"You can now run the agent with: {binary_path}")

if __name__ == "__main__":
    main()
EOF

    # Make the Python script executable
    chmod +x "$PY_PAYLOAD_DIR/simple-install.py"
    
    log_info "Simple Python payload created successfully"
}

# Main function
main() {
    log_info "Starting simple APA payload generation process..."
    
    # Parse command line arguments
    if [ $# -eq 0 ]; then
        log_info "No arguments provided, generating all simple payload types"
        build_basic_binaries
        create_simple_shell_payload
        create_simple_powershell_payload
        create_simple_python_payload
    else
        for arg in "$@"; do
            case "$arg" in
                "binaries")
                    build_basic_binaries
                    ;;
                "shell")
                    create_simple_shell_payload
                    ;;
                "powershell")
                    create_simple_powershell_payload
                    ;;
                "python")
                    create_simple_python_payload
                    ;;
                *)
                    log_error "Unknown argument: $arg"
                    log_info "Supported arguments: binaries, shell, powershell, python"
                    exit 1
                    ;;
            esac
        done
    fi
    
    log_info "Simple payload generation process completed successfully!"
    log_info "Payloads are available in: $PAYLOADS_DIR"
}

# Run main function with all arguments
main "$@"