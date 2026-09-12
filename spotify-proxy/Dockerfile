FROM golang:1.26-alpine AS builder

# Install build dependencies (libc, audio libs)
RUN apk add --no-cache gcc musl-dev pkgconfig \
    libogg-dev libvorbis-dev flac-dev mpg123-dev alsa-lib-dev

WORKDIR /app

# Cache Go modules layer
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source code
COPY . .

# Build with optimization flags
# -w: strip DWARF debug info
# -s: strip all symbol table
# This reduces binary size by ~40-50%
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=1 go build \
    -ldflags="-s -w" \
    -a \
    -installsuffix cgo \
    -o /bin/spotify-proxy ./cmd/server

# Runtime stage: Alpine (5MB vs debian 50MB+)
FROM alpine:3.21

# Install only runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    libogg \
    libvorbis \
    flac \
    mpg123 \
    alsa-lib \
    curl

# Add non-root user for security
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /home/appuser

COPY --from=builder --chown=appuser:appuser /bin/spotify-proxy /usr/local/bin/spotify-proxy

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8081/health || exit 1

USER appuser

EXPOSE 8081 8989

ENV PORT=8081

CMD ["spotify-proxy"]
