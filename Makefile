# Makefile for APA Agent

BINARY_NAME=agentd
BUILD_DIR=bin
CMD_PATH=./cmd/agentd/main.go
MATRIX_PLATFORMS?=linux/amd64 linux/arm64 linux/arm linux/386 linux/riscv64 windows/amd64 windows/arm64 darwin/amd64 darwin/arm64 freebsd/amd64
GOFLAGS?=
DIST_DIR=dist

.PHONY: all build clean test build-linux build-windows build-darwin build-matrix build-matrix-minimal ci-local

all: build

build:
	@echo "Building for current OS..."
	CGO_ENABLED=0 go build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)

build-linux:
	@echo "Building for Linux (amd64)..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_PATH)
	@echo "Building for Linux (arm64)..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_PATH)
	@echo "Building for Linux (arm)..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-armv7 $(CMD_PATH)
	@echo "Building for Linux (386)..."
	CGO_ENABLED=0 GOOS=linux GOARCH=386 go build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-386 $(CMD_PATH)

build-windows:
	@echo "Building for Windows (amd64)..."
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_PATH)
	@echo "Building for Windows (arm64)..."
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.exe $(CMD_PATH)

build-darwin:
	@echo "Building for macOS (amd64)..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_PATH)
	@echo "Building for macOS (arm64)..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_PATH)

build-matrix:
	@mkdir -p $(BUILD_DIR)
	@for platform in $(MATRIX_PLATFORMS); do \
		OS=$${platform%/*}; ARCH=$${platform#*/}; \
		EXT=""; [ "$$OS" = "windows" ] && EXT=".exe"; \
		OUT="$(BUILD_DIR)/$(BINARY_NAME)-$$OS-$$ARCH$$EXT"; \
		echo "Building $$OS/$$ARCH..."; \
		CGO_ENABLED=0 GOOS=$$OS GOARCH=$$ARCH go build $(GOFLAGS) -o $$OUT $(CMD_PATH) || exit $$?; \
	done

build-matrix-minimal:
	@mkdir -p $(BUILD_DIR)
	@for platform in $(MATRIX_PLATFORMS); do \
		OS=$${platform%/*}; ARCH=$${platform#*/}; \
		EXT=""; [ "$$OS" = "windows" ] && EXT=".exe"; \
		OUT="$(BUILD_DIR)/$(BINARY_NAME)-minimal-$$OS-$$ARCH$$EXT"; \
		echo "Building minimal $$OS/$$ARCH..."; \
		CGO_ENABLED=0 GOOS=$$OS GOARCH=$$ARCH go build $(GOFLAGS) -tags minimal -o $$OUT $(CMD_PATH) || exit $$?; \
	done

test:
	@echo "Running tests..."
	go test -v ./...

ci-local:
	@echo "Validating local GitHub workflows..."
	bash scripts/validate-workflows-local.sh

clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR) $(DIST_DIR)

.PHONY: dist package-matrix checksums

dist: package-matrix checksums

# Build matrix and create distribution archives (tar.gz for Unix, zip for Windows)
package-matrix: build-matrix
	@mkdir -p $(DIST_DIR)
	@for artifact in $(BUILD_DIR)/$(BINARY_NAME)-*; do \
		fname=$$(basename $$artifact); \
		case $$fname in \
			*-windows-*.exe) \
				zipname=$(DIST_DIR)/$$fname.zip; \
				zip -j $$zipname $$artifact >/dev/null; \
				echo "Packaged $$zipname"; \
				;; \
			*) \
				tarname=$(DIST_DIR)/$$fname.tar.gz; \
				tar -C $(BUILD_DIR) -czf $$tarname $$fname; \
				echo "Packaged $$tarname"; \
				;; \
		esac; \
	done

# Generate SHA256SUMS for all dist artifacts
checksums:
	@cd $(DIST_DIR) && shasum -a 256 * > SHA256SUMS
	@echo "Wrote $(DIST_DIR)/SHA256SUMS"
