# use Go 1.24 alpine image
FROM --platform=$BUILDPLATFORM golang:1.24.6-alpine AS builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG GOOS=linux
ARG GOARCH=amd64

WORKDIR /src
COPY . .

# build dependencies
RUN apk add --no-cache build-base git libc-dev

# build binary
RUN CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH} go build -o /out/agentd ./cmd/agentd

FROM scratch
COPY --from=builder /out/agentd /agentd
COPY configs /configs
COPY examples /examples
EXPOSE 8080 9090
ENTRYPOINT ["/agentd", "-config", "/configs/agent-config.yaml"]

