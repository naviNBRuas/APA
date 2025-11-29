#!/bin/bash

# Package script for creating deployment artifacts for the APA agent
# Supports multiple platforms and package formats

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BUILD_DIR="$PROJECT_ROOT/build"
DIST_DIR="$PROJECT_ROOT/dist"
VERSION="1.0.0"

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

# Create directories
mkdir -p "$BUILD_DIR"
mkdir -p "$DIST_DIR"

# Function to build binaries for all platforms
build_binaries() {
    log_info "Building binaries for all platforms..."
    
    # Platforms to build for
    PLATFORMS=(
        "linux/amd64"
        "linux/arm64"
        "darwin/amd64"
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
        
        # Build the binary
        CGO_ENABLED=0 GOOS="$OS" GOARCH="$ARCH" \
            go build -v -ldflags="-s -w" -o "$BUILD_DIR/$OUTPUT_NAME" "$PROJECT_ROOT/cmd/agentd"
        
        # Build WASM modules
        MODULES_DIR="$BUILD_DIR/modules-$OS-$ARCH"
        mkdir -p "$MODULES_DIR"
        
        MODULES=(config-watcher crypto-hasher data-logger message-broker net-monitor simple-adder system-info)
        for module in "${MODULES[@]}"; do
            log_info "Building WASM module: $module for $OS/$ARCH..."
            GOOS=wasip1 GOARCH=wasm go build -o "$MODULES_DIR/$module.wasm" "$PROJECT_ROOT/examples/modules/$module/main.go"
        done
        
        log_info "Built binaries for $OS/$ARCH"
    done
    
    log_info "All binaries built successfully"
}

# Function to create tarball packages
create_tarballs() {
    log_info "Creating tarball packages..."
    
    # Platforms to package
    PLATFORMS=(
        "linux/amd64"
        "linux/arm64"
        "darwin/amd64"
    )
    
    for platform in "${PLATFORMS[@]}"; do
        OS=$(echo "$platform" | cut -d'/' -f1)
        ARCH=$(echo "$platform" | cut -d'/' -f2)
        
        log_info "Creating tarball for $OS/$ARCH..."
        
        PACKAGE_NAME="apa-$VERSION-$OS-$ARCH.tar.gz"
        PACKAGE_DIR="$DIST_DIR/$OS-$ARCH"
        
        # Create package directory
        mkdir -p "$PACKAGE_DIR"
        
        # Copy binaries
        cp "$BUILD_DIR/agentd-$OS-$ARCH" "$PACKAGE_DIR/"
        
        # Copy modules
        cp -r "$BUILD_DIR/modules-$OS-$ARCH" "$PACKAGE_DIR/modules"
        
        # Copy configs
        cp -r "$PROJECT_ROOT/configs" "$PACKAGE_DIR/"
        
        # Copy documentation
        cp "$PROJECT_ROOT/README.md" "$PACKAGE_DIR/" 2>/dev/null || true
        cp "$PROJECT_ROOT/LICENSE" "$PACKAGE_DIR/" 2>/dev/null || true
        
        # Create tarball
        tar -czf "$DIST_DIR/$PACKAGE_NAME" -C "$DIST_DIR" "$(basename "$PACKAGE_DIR")"
        
        # Clean up temp directory
        rm -rf "$PACKAGE_DIR"
        
        log_info "Created tarball: $PACKAGE_NAME"
    done
    
    log_info "Tarball packages created successfully"
}

# Function to create Windows ZIP packages
create_windows_zip() {
    log_info "Creating Windows ZIP package..."
    
    OS="windows"
    ARCH="amd64"
    
    PACKAGE_NAME="apa-$VERSION-$OS-$ARCH.zip"
    PACKAGE_DIR="$DIST_DIR/windows-amd64"
    
    # Create package directory
    mkdir -p "$PACKAGE_DIR"
    
    # Copy binaries
    cp "$BUILD_DIR/agentd-$OS-$ARCH.exe" "$PACKAGE_DIR/"
    
    # Copy modules
    cp -r "$BUILD_DIR/modules-$OS-$ARCH" "$PACKAGE_DIR/modules"
    
    # Copy configs
    cp -r "$PROJECT_ROOT/configs" "$PACKAGE_DIR/"
    
    # Copy documentation
    cp "$PROJECT_ROOT/README.md" "$PACKAGE_DIR/" 2>/dev/null || true
    cp "$PROJECT_ROOT/LICENSE" "$PACKAGE_DIR/" 2>/dev/null || true
    
    # Create ZIP file
    (cd "$DIST_DIR" && zip -r "$PACKAGE_NAME" "windows-amd64")
    
    # Clean up temp directory
    rm -rf "$PACKAGE_DIR"
    
    log_info "Created Windows ZIP: $PACKAGE_NAME"
}

# Function to create Debian packages
create_deb_packages() {
    log_info "Creating Debian packages..."
    
    # Platforms to package
    PLATFORMS=(
        "linux/amd64"
        "linux/arm64"
    )
    
    for platform in "${PLATFORMS[@]}"; do
        OS=$(echo "$platform" | cut -d'/' -f1)
        ARCH=$(echo "$platform" | cut -d'/' -f2)
        
        # Map Go architecture to Debian architecture
        case "$ARCH" in
            "amd64")
                DEB_ARCH="amd64"
                ;;
            "arm64")
                DEB_ARCH="arm64"
                ;;
            *)
                log_warn "Unsupported architecture for Debian package: $ARCH"
                continue
                ;;
        esac
        
        log_info "Creating Debian package for $OS/$DEB_ARCH..."
        
        PACKAGE_NAME="apa_$VERSION_$DEB_ARCH.deb"
        DEB_ROOT="$DIST_DIR/deb-root-$DEB_ARCH"
        
        # Create Debian package structure
        mkdir -p "$DEB_ROOT/DEBIAN"
        mkdir -p "$DEB_ROOT/usr/bin"
        mkdir -p "$DEB_ROOT/usr/lib/apa/modules"
        mkdir -p "$DEB_ROOT/etc/apa"
        mkdir -p "$DEB_ROOT/usr/share/doc/apa"
        
        # Copy binaries
        cp "$BUILD_DIR/agentd-$OS-$ARCH" "$DEB_ROOT/usr/bin/agentd"
        chmod 755 "$DEB_ROOT/usr/bin/agentd"
        
        # Copy modules
        cp "$BUILD_DIR/modules-$OS-$ARCH"/* "$DEB_ROOT/usr/lib/apa/modules/"
        
        # Copy configs
        cp "$PROJECT_ROOT/configs/agent-config.yaml" "$DEB_ROOT/etc/apa/"
        
        # Copy documentation
        cp "$PROJECT_ROOT/README.md" "$DEB_ROOT/usr/share/doc/apa/" 2>/dev/null || true
        cp "$PROJECT_ROOT/LICENSE" "$DEB_ROOT/usr/share/doc/apa/copyright" 2>/dev/null || true
        
        # Create control file
        cat > "$DEB_ROOT/DEBIAN/control" << EOF
Package: apa
Version: $VERSION
Section: utils
Priority: optional
Architecture: $DEB_ARCH
Maintainer: APA Team <support@example.com>
Description: Autonomous Polymorphic Agent
 A state-of-the-art, self-healing, and decentralized agent platform.
EOF
        
        # Create postinst script
        cat > "$DEB_ROOT/DEBIAN/postinst" << EOF
#!/bin/bash
set -e

# Create apa user if it doesn't exist
if ! id -u apa >/dev/null 2>&1; then
    useradd -r -s /bin/false -d /var/lib/apa apa
fi

# Set permissions
chown -R apa:apa /etc/apa
chmod -R 750 /etc/apa

# Enable and start service (if systemd is available)
if command -v systemctl >/dev/null 2>&1; then
    # Create systemd service file
    cat > /etc/systemd/system/apa.service << SERVICE_EOF
[Unit]
Description=Autonomous Polymorphic Agent
After=network.target

[Service]
Type=simple
User=apa
ExecStart=/usr/bin/agentd -config /etc/apa/agent-config.yaml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
SERVICE_EOF
    
    systemctl daemon-reload
    systemctl enable apa.service
fi

exit 0
EOF
        
        chmod 755 "$DEB_ROOT/DEBIAN/postinst"
        
        # Create prerm script
        cat > "$DEB_ROOT/DEBIAN/prerm" << EOF
#!/bin/bash
set -e

# Stop and disable service (if systemd is available)
if command -v systemctl >/dev/null 2>&1; then
    systemctl stop apa.service || true
    systemctl disable apa.service || true
    rm -f /etc/systemd/system/apa.service
    systemctl daemon-reload
fi

exit 0
EOF
        
        chmod 755 "$DEB_ROOT/DEBIAN/prerm"
        
        # Build Debian package
        dpkg-deb --build "$DEB_ROOT" "$DIST_DIR/$PACKAGE_NAME"
        
        # Clean up
        rm -rf "$DEB_ROOT"
        
        log_info "Created Debian package: $PACKAGE_NAME"
    done
    
    log_info "Debian packages created successfully"
}

# Function to create RPM packages
create_rpm_packages() {
    log_info "Creating RPM packages..."
    
    # Platforms to package
    PLATFORMS=(
        "linux/amd64"
        "linux/arm64"
    )
    
    for platform in "${PLATFORMS[@]}"; do
        OS=$(echo "$platform" | cut -d'/' -f1)
        ARCH=$(echo "$platform" | cut -d'/' -f2)
        
        # Map Go architecture to RPM architecture
        case "$ARCH" in
            "amd64")
                RPM_ARCH="x86_64"
                ;;
            "arm64")
                RPM_ARCH="aarch64"
                ;;
            *)
                log_warn "Unsupported architecture for RPM package: $ARCH"
                continue
                ;;
        esac
        
        log_info "Creating RPM package for $OS/$RPM_ARCH..."
        
        # Create RPM build structure
        RPM_ROOT="$DIST_DIR/rpm-root-$RPM_ARCH"
        mkdir -p "$RPM_ROOT/BUILD"
        mkdir -p "$RPM_ROOT/RPMS"
        mkdir -p "$RPM_ROOT/SOURCES"
        mkdir -p "$RPM_ROOT/SPECS"
        mkdir -p "$RPM_ROOT/SRPMS"
        
        # Create source tarball
        SOURCE_TAR="apa-$VERSION.tar.gz"
        (cd "$PROJECT_ROOT" && tar -czf "$RPM_ROOT/SOURCES/$SOURCE_TAR" --exclude='.git' .)
        
        # Create spec file
        cat > "$RPM_ROOT/SPECS/apa.spec" << EOF
Name: apa
Version: $VERSION
Release: 1
Summary: Autonomous Polymorphic Agent
License: MIT
BuildArch: $RPM_ARCH

%description
A state-of-the-art, self-healing, and decentralized agent platform.

%prep
%setup -q

%build
# Build is done externally, just copy files

%install
mkdir -p %{buildroot}/usr/bin
mkdir -p %{buildroot}/usr/lib/apa/modules
mkdir -p %{buildroot}/etc/apa
mkdir -p %{buildroot}/usr/share/doc/apa

cp $BUILD_DIR/agentd-$OS-$ARCH %{buildroot}/usr/bin/agentd
chmod 755 %{buildroot}/usr/bin/agentd

cp $BUILD_DIR/modules-$OS-$ARCH/* %{buildroot}/usr/lib/apa/modules/

cp configs/agent-config.yaml %{buildroot}/etc/apa/

cp README.md %{buildroot}/usr/share/doc/apa/ || true
cp LICENSE %{buildroot}/usr/share/doc/apa/ || true

%files
/usr/bin/agentd
/usr/lib/apa/modules/*
/etc/apa/agent-config.yaml
/usr/share/doc/apa/*

%post
# Create apa user if it doesn't exist
if ! id -u apa >/dev/null 2>&1; then
    useradd -r -s /bin/false -d /var/lib/apa apa
fi

# Set permissions
chown -R apa:apa /etc/apa
chmod -R 750 /etc/apa

# Enable and start service (if systemd is available)
if command -v systemctl >/dev/null 2>&1; then
    # Create systemd service file
    cat > /etc/systemd/system/apa.service << SERVICE_EOF
[Unit]
Description=Autonomous Polymorphic Agent
After=network.target

[Service]
Type=simple
User=apa
ExecStart=/usr/bin/agentd -config /etc/apa/agent-config.yaml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
SERVICE_EOF
    
    systemctl daemon-reload
    systemctl enable apa.service
fi

%preun
# Stop and disable service (if systemd is available)
if command -v systemctl >/dev/null 2>&1; then
    systemctl stop apa.service || true
    systemctl disable apa.service || true
    rm -f /etc/systemd/system/apa.service
    systemctl daemon-reload
fi

%changelog
* \$(date +"%a %b %d %Y") APA Team <support@example.com> - $VERSION-1
- Initial release

EOF
        
        # Build RPM package
        (cd "$RPM_ROOT" && rpmbuild --define "_topdir $(pwd)" -bb SPECS/apa.spec)
        
        # Copy RPM to dist directory
        cp "$RPM_ROOT/RPMS/$RPM_ARCH/apa-$VERSION-1.$RPM_ARCH.rpm" "$DIST_DIR/"
        
        # Clean up
        rm -rf "$RPM_ROOT"
        
        log_info "Created RPM package: apa-$VERSION-1.$RPM_ARCH.rpm"
    done
    
    log_info "RPM packages created successfully"
}

# Function to create container images
create_container_images() {
    log_info "Creating container images..."
    
    # Platforms to build for
    PLATFORMS=(
        "linux/amd64"
        "linux/arm64"
    )
    
    for platform in "${PLATFORMS[@]}"; do
        OS=$(echo "$platform" | cut -d'/' -f1)
        ARCH=$(echo "$platform" | cut -d'/' -f2)
        
        log_info "Building container image for $OS/$ARCH..."
        
        # Build container image
        docker build --platform "$platform" -t "apa:$VERSION-$OS-$ARCH" -f "$PROJECT_ROOT/Containerfile" "$PROJECT_ROOT"
        
        # Save image as tar file
        docker save "apa:$VERSION-$OS-$ARCH" | gzip > "$DIST_DIR/apa-container-$VERSION-$OS-$ARCH.tar.gz"
        
        log_info "Created container image: apa-container-$VERSION-$OS-$ARCH.tar.gz"
    done
    
    log_info "Container images created successfully"
}

# Main function
main() {
    log_info "Starting APA packaging process..."
    
    # Parse command line arguments
    if [ $# -eq 0 ]; then
        log_info "No arguments provided, building all package types"
        build_binaries
        create_tarballs
        create_windows_zip
        create_deb_packages
        create_rpm_packages
        create_container_images
    else
        for arg in "$@"; do
            case "$arg" in
                "binaries")
                    build_binaries
                    ;;
                "tarballs")
                    build_binaries
                    create_tarballs
                    ;;
                "windows")
                    build_binaries
                    create_windows_zip
                    ;;
                "debian")
                    build_binaries
                    create_deb_packages
                    ;;
                "rpm")
                    build_binaries
                    create_rpm_packages
                    ;;
                "container")
                    build_binaries
                    create_container_images
                    ;;
                *)
                    log_error "Unknown argument: $arg"
                    log_info "Supported arguments: binaries, tarballs, windows, debian, rpm, container"
                    exit 1
                    ;;
            esac
        done
    fi
    
    log_info "Packaging process completed successfully!"
    log_info "Packages are available in: $DIST_DIR"
}

# Run main function with all arguments
main "$@"