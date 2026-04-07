# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go module files first for layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')" \
    -o /app/portwatch ./main.go

# Final stage — minimal runtime image
FROM scratch

# Copy CA certificates for HTTPS webhook/email notifications
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the compiled binary
COPY --from=builder /app/portwatch /portwatch

# portwatch reads /proc/net directly, so no special ports needed
# Config file is expected at /etc/portwatch/config.yaml by default
VOLUME ["/etc/portwatch", "/var/lib/portwatch"]

# Run as non-root where possible (note: reading /proc/net/tcp may require
# elevated privileges depending on the host kernel configuration)
USER 65534:65534

ENTRYPOINT ["/portwatch"]
CMD ["--config", "/etc/portwatch/config.yaml"]
