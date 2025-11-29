#!/bin/bash

# Release script for creating and signing deployment artifacts for the APA agent
# This script automates the entire release process including building, packaging, signing, and verification

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
DIST_DIR="$PROJECT_ROOT/dist"
RELEASE_DIR="$PROJECT_ROOT/release"
VERSION=""
GPG_KEY_ID=""
GITHUB_TOKEN=""

# Colors for output
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

# Display usage information
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo "Options:"
    echo "  -v, --version VERSION    Specify the release version (required)"
    echo "  -k, --key-id KEY_ID      Specify the GPG key ID for signing (required)"
    echo "  -t, --token TOKEN        Specify the GitHub token for release upload (optional)"
    echo "  -h, --help               Display this help message"
    echo ""
    echo "Example:"
    echo "  $0 -v 1.2.3 -k ABC123DEF456"
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -v|--version)
                VERSION="$2"
                shift 2
                ;;
            -k|--key-id)
                GPG_KEY_ID="$2"
                shift 2
                ;;
            -t|--token)
                GITHUB_TOKEN="$2"
                shift 2
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done

    # Validate required arguments
    if [[ -z "$VERSION" ]]; then
        log_error "Version is required"
        usage
        exit 1
    fi

    if [[ -z "$GPG_KEY_ID" ]]; then
        log_error "GPG key ID is required"
        usage
        exit 1
    fi
}

# Create directories
prepare_directories() {
    log_info "Preparing directories..."
    
    # Clean previous release directory
    rm -rf "$RELEASE_DIR"
    
    # Create directories
    mkdir -p "$DIST_DIR"
    mkdir -p "$RELEASE_DIR"
    
    log_info "Directories prepared successfully"
}

# Run the packaging script
run_packaging() {
    log_info "Running packaging process..."
    
    # Execute the packaging script
    "$SCRIPT_DIR/package.sh"
    
    log_info "Packaging completed successfully"
}

# Sign packages with GPG
sign_packages() {
    log_info "Signing packages with GPG key: $GPG_KEY_ID..."
    
    # Check if GPG key exists
    if ! gpg --list-keys "$GPG_KEY_ID" >/dev/null 2>&1; then
        log_error "GPG key $GPG_KEY_ID not found"
        exit 1
    fi
    
    # Sign all packages in the dist directory
    for file in "$DIST_DIR"/*; do
        if [[ -f "$file" ]]; then
            log_info "Signing: $(basename "$file")"
            gpg --detach-sign --armor --local-user "$GPG_KEY_ID" "$file"
        fi
    done
    
    log_info "All packages signed successfully"
}

# Create checksums for packages
create_checksums() {
    log_info "Creating checksums for packages..."
    
    # Create SHA256 checksums
    (cd "$DIST_DIR" && sha256sum * > "$DIST_DIR/checksums.txt")
    
    # Sign the checksums file
    gpg --detach-sign --armor --local-user "$GPG_KEY_ID" "$DIST_DIR/checksums.txt"
    
    log_info "Checksums created and signed successfully"
}

# Verify signatures
verify_signatures() {
    log_info "Verifying package signatures..."
    
    # Verify all signatures
    for file in "$DIST_DIR"/*.asc; do
        if [[ -f "$file" ]]; then
            log_info "Verifying signature: $(basename "$file")"
            gpg --verify "$file" "${file%.asc}"
        fi
    done
    
    # Verify checksums signature
    gpg --verify "$DIST_DIR/checksums.txt.asc" "$DIST_DIR/checksums.txt"
    
    log_info "All signatures verified successfully"
}

# Create release notes
create_release_notes() {
    log_info "Creating release notes..."
    
    # Create a basic release notes template
    cat > "$RELEASE_DIR/release-notes.md" << EOF
# APA Agent Release $VERSION

## Changelog

- Feature 1
- Feature 2
- Bug fix 1
- Bug fix 2

## Installation

Download the appropriate package for your platform:

- **Linux (tar.gz)**: \`apa-$VERSION-linux-amd64.tar.gz\`
- **Linux (Debian)**: \`apa_$VERSION_amd64.deb\`
- **Linux (RPM)**: \`apa-$VERSION-1.x86_64.rpm\`
- **macOS**: \`apa-$VERSION-darwin-amd64.tar.gz\`
- **Windows**: \`apa-$VERSION-windows-amd64.zip\`
- **Container**: \`apa-container-$VERSION-linux-amd64.tar.gz\`

## Verification

All packages are signed with GPG key $GPG_KEY_ID. To verify:

\`\`\`bash
gpg --verify <package>.asc <package>
\`\`\`

Checksums are available in \`checksums.txt\` and \`checksums.txt.asc\`.

## Security

For security concerns, please contact our security team at security@example.com.

EOF
    
    log_info "Release notes created successfully"
}

# Organize release files
organize_release_files() {
    log_info "Organizing release files..."
    
    # Copy all distribution files to release directory
    cp "$DIST_DIR"/* "$RELEASE_DIR/"
    
    # Copy release notes
    cp "$RELEASE_DIR/release-notes.md" "$RELEASE_DIR/"
    
    log_info "Release files organized successfully"
}

# Upload to GitHub Releases (if token is provided)
upload_to_github() {
    if [[ -n "$GITHUB_TOKEN" ]]; then
        log_info "Uploading to GitHub Releases..."
        
        # This would typically use the GitHub API or gh CLI
        # For now, we'll just log what would be uploaded
        log_info "Would upload the following files to GitHub Releases:"
        for file in "$RELEASE_DIR"/*; do
            if [[ -f "$file" ]]; then
                log_info "  - $(basename "$file")"
            fi
        done
        
        log_info "GitHub upload completed (simulated)"
    else
        log_info "No GitHub token provided, skipping upload"
    fi
}

# Create SBOM (Software Bill of Materials)
create_sbom() {
    log_info "Creating Software Bill of Materials (SBOM)..."
    
    # This would typically use tools like syft or cyclonedx-gomod
    # For now, we'll create a basic SBOM template
    cat > "$RELEASE_DIR/sbom.json" << EOF
{
  "bomFormat": "CycloneDX",
  "specVersion": "1.4",
  "version": 1,
  "components": [
    {
      "type": "application",
      "name": "APA Agent",
      "version": "$VERSION",
      "description": "Autonomous Polymorphic Agent",
      "licenses": [
        {
          "license": {
            "id": "MIT"
          }
        }
      ],
      "purl": "pkg:github/naviNBRuas/APA@$VERSION"
    }
  ]
}
EOF
    
    # Sign the SBOM
    gpg --detach-sign --armor --local-user "$GPG_KEY_ID" "$RELEASE_DIR/sbom.json"
    
    log_info "SBOM created and signed successfully"
}

# Main function
main() {
    log_info "Starting APA release process..."
    
    # Parse command line arguments
    parse_args "$@"
    
    # Prepare directories
    prepare_directories
    
    # Run packaging
    run_packaging
    
    # Sign packages
    sign_packages
    
    # Create checksums
    create_checksums
    
    # Verify signatures
    verify_signatures
    
    # Create release notes
    create_release_notes
    
    # Create SBOM
    create_sbom
    
    # Organize release files
    organize_release_files
    
    # Upload to GitHub (if token provided)
    upload_to_github
    
    log_info "Release process completed successfully!"
    log_info "Release files are available in: $RELEASE_DIR"
}

# Run main function with all arguments
main "$@"