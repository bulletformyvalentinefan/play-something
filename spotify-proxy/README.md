# spotify-proxy (Go + go-librespot)

Backend que se hace pasar por cliente oficial Spotify usando
`devgianlu/go-librespot` (puro Go, sin CGO). La búsqueda va por `spclient`:
`context-resolve/v1/spotify:search:…` + `extended-metadata` (`TRACK_V4`).

## Endpoints

- `GET /health`
- `POST /api/v1/spotify/auth/start` → URL de login Spotify (PKCE)
- `GET /api/v1/spotify/auth/callback` y loopback `:8989/login`
- `GET /api/v1/spotify/auth/status` · `/token` · `/me` · `POST /logout`
- `GET /api/v1/spotify/proxy/search?q=&limit=` → `{tracks:{items:[…]}}` (header `X-Source: spclient`)
- `GET /api/v1/spotify/proxy/me/playlists`, `/playlists/{id}`, `/playlists/{id}/tracks`, `/tracks/{id}`
- `PUT /api/v1/spotify/player/*` (proxy a Web API con tu token)

## Env

Ver `internal/config/config.go`: `PORT`, `FRONTEND_ORIGIN`, `OAUTH_CALLBACK_URL`,
`SPOTIFY_CLIENT_ID` (default: client oficial `65b70807…`), `SPOTIFY_CACHE_DIR`.

## Estructura

- `cmd/server/main.go` — wiring + loopback `:8989` para el client oficial
- `internal/auth/` — config OAuth + helpers PKCE
- `internal/handler/` — `auth.go`, `proxy.go`, `player.go`
- `internal/spotify/` — `manager.go` (sesiones spclient cacheadas),
  `spclient.go` (sesión pura-Go + búsqueda + enriquecimiento),
  `spclient_unit_test.go` (offline), `spclient_integration_test.go` (browser)

## Dev

```bash
go vet ./... && go build ./cmd/server/
```

## Notas legales

Usa APIs internas no documentadas (`spclient`, `login5`). Requiere Premium,
viola ToS Spotify. Solo para lab. GPL-3.0 por `go-librespot`.
