FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 go build \
    -ldflags="-s -w" \
    -o /bin/spotify-proxy ./cmd/server

FROM alpine:3.21

RUN apk add --no-cache ca-certificates curl

RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /home/appuser

COPY --from=builder --chown=appuser:appuser /bin/spotify-proxy /usr/local/bin/spotify-proxy

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8081/health || exit 1

USER appuser

EXPOSE 8081 8989

ENV PORT=8081

CMD ["spotify-proxy"]
