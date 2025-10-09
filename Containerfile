# Build a minimal Go agent runtime image (adjust if using Rust/other)
FROM golang:1.21-alpine AS builder
WORKDIR /src
COPY . .
# build static, small binary (example target)
RUN apk add --no-cache build-base git libc-dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/agentd ./cmd/agentd

FROM scratch
COPY --from=builder /out/agentd /agentd
EXPOSE 8080 9090
ENTRYPOINT ["/agentd"]
