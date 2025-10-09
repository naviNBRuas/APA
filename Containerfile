# use Go 1.24 alpine image
FROM golang:1.24.6-alpine AS builder

WORKDIR /src
COPY . .

# build dependencies
RUN apk add --no-cache build-base git libc-dev

# build binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/agentd ./cmd/agentd

FROM scratch
COPY --from=builder /out/agentd /agentd
COPY configs /configs
COPY examples /examples
EXPOSE 8080 9090
ENTRYPOINT ["/agentd", "-config", "/configs/agent-config.yaml"]

