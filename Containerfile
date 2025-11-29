# use Go 1.24 alpine image
FROM --platform=$BUILDPLATFORM golang:1.24.6-alpine AS builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG GOOS=linux
ARG GOARCH=amd64

# Install build dependencies including cross-compilation tools
RUN apk add --no-cache build-base git libc-dev gcc-aarch64-none-elf gcc-arm-none-eabi

WORKDIR /src
COPY . .

# Build dependencies
RUN apk add --no-cache build-base git libc-dev

# Build binary with cross-compilation support
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-$GOOS} GOARCH=${TARGETARCH:-$GOARCH} go build -o /out/agentd ./cmd/agentd

# Build WASM modules
RUN GOOS=wasip1 GOARCH=wasm go build -o /out/examples/modules/simple-adder/simple-adder.wasm ./examples/modules/simple-adder/main.go && \
    GOOS=wasip1 GOARCH=wasm go build -o /out/examples/modules/system-info/system-info.wasm ./examples/modules/system-info/main.go && \
    GOOS=wasip1 GOARCH=wasm go build -o /out/examples/modules/data-logger/data-logger.wasm ./examples/modules/data-logger/main.go && \
    GOOS=wasip1 GOARCH=wasm go build -o /out/examples/modules/net-monitor/net-monitor.wasm ./examples/modules/net-monitor/main.go && \
    GOOS=wasip1 GOARCH=wasm go build -o /out/examples/modules/crypto-hasher/crypto-hasher.wasm ./examples/modules/crypto-hasher/main.go && \
    GOOS=wasip1 GOARCH=wasm go build -o /out/examples/modules/message-broker/message-broker.wasm ./examples/modules/message-broker/main.go && \
    GOOS=wasip1 GOARCH=wasm go build -o /out/examples/modules/config-watcher/config-watcher.wasm ./examples/modules/config-watcher/main.go

# Multi-stage build for different platforms
FROM --platform=$TARGETPLATFORM alpine:latest AS linux-amd64
COPY --from=builder /out/agentd /agentd
COPY --from=builder /out/examples /examples
COPY configs /configs
EXPOSE 8080 9090
ENTRYPOINT ["/agentd", "-config", "/configs/agent-config.yaml"]

FROM --platform=$TARGETPLATFORM alpine:latest AS linux-arm64
COPY --from=builder /out/agentd /agentd
COPY --from=builder /out/examples /examples
COPY configs /configs
EXPOSE 8080 9090
ENTRYPOINT ["/agentd", "-config", "/configs/agent-config.yaml"]

# Default target
FROM linux-amd64
