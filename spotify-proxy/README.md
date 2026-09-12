# spotify-proxy (Go + go-librespot)

Proxy que se hace pasar por cliente oficial Spotify usando `devgianlu/go-librespot`.
Finge `client_id` oficial y obtiene `login5` access_token para luego proxyear
a `api.spotify.com/v1` y `spclient.wg.spotify.com` con el token del usuario.

## Endpoints (fase 1 scaffold)

- `GET /health`
- `POST /api/v1/spotify/auth/start` -> devuelve URL redirect (TODO: OAuth real en :36842)
- `GET /api/v1/spotify/auth/callback?code=`
- `GET /api/v1/spotify/auth/status?userId=`
- `GET /api/v1/spotify/proxy/me` (requiere `Authorization: Bearer <token>`)
- `GET /api/v1/spotify/proxy/me/playlists`
- `GET /api/v1/spotify/proxy/search?q=&type=track`
- `GET /api/v1/spotify/player/status` (TODO: CGO player)

## Env

 Ver `internal/config/config.go` : `PORT`, `FRONTEND_ORIGIN`, `CREDENTIAL_KEY` (32 bytes), `ORACLE_DSN`.

## Dev

```bash
cd spotify-proxy
go mod tidy
go run ./cmd/server
```

Docker:
```bash
docker build -f spotify-proxy/Dockerfile -t spotify-proxy spotify-proxy
```

## Notas legales

 Usa APIs internas no documentadas (`spclient`, `dealer`, `login5`). Breakable, requiere Premium,
 viola ToS Spotify. Solo para lab. GPL-3.0 por `go-librespot` si se linkea como librería.
