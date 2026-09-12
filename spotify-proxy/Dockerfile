FROM golang:1.26-bookworm AS builder
WORKDIR /app
# deps de audio solo si se habilita player (CGO)
RUN apt-get update && apt-get install -y --no-install-recommends libogg-dev libvorbis-dev libflac-dev libmpg123-dev pkg-config && rm -rf /var/lib/apt/lists/*
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o /bin/spotify-proxy ./cmd/server

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates libogg0 libvorbis0a libFLAC12 libmpg123-0 && rm -rf /var/lib/apt/lists/*
COPY --from=builder /bin/spotify-proxy /usr/local/bin/spotify-proxy
EXPOSE 8081
ENV PORT=8081
CMD ["spotify-proxy"]
