# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-04-06

### Added
- Complete rewrite of the autonomous agent platform with enhanced capabilities
- Multi-protocol networking support (TCP, UDP, HTTP, WebSocket, libp2p)
- Cross-platform compatibility for Linux, macOS, and Windows
- Multi-architecture support (AMD64, ARM64, ARM)
- Advanced robustness and self-healing mechanisms
- Intelligent algorithms and adaptive decision-making
- Comprehensive testing framework
- Standalone agent implementation for demonstration
- Enhanced CI/CD pipeline with security scanning
- Comprehensive documentation and examples
- Local workflow validation utility: `scripts/validate-workflows-local.sh`
- `Makefile` target `ci-local` to run local workflow validation
- Explicit opt-in guard for local P2P integration tests via `APA_RUN_P2P_INTEGRATION=1`

### Changed
- Refactored core architecture for better modularity
- Improved error handling and recovery systems
- Enhanced security features and authentication
- Updated build system and deployment processes
- Modernized codebase with Go 1.24+ features
- Hardened CI workflows with bounded job timeouts and workflow-level concurrency controls
- Made CI formatting checks deterministic and fail-fast (no in-CI auto-formatting)
- Strengthened lint enforcement to fail on issues in CI
- Improved docs generation determinism by resetting/sorting generated API index output
- Updated release workflow action pin from `softprops/action-gh-release@v1` to `@v2`
- Corrected release packaging tarball path handling for docs inclusion
- Fixed README local development typo in build command and documented local workflow validation

### Fixed
- Reduced networking test flakiness by skipping environment-dependent local P2P tests unless explicitly enabled
- Removed brittle release short-circuit behavior tied to latest release asset count

### Removed
- Legacy components that were not maintained
- Outdated dependencies and unused code
- Experimental features that were not production-ready