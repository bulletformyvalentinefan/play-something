# play-something

Web que consume Spotify con token Premium propio vía `go-librespot` (estilo Sonora):
sin Web API para búsqueda — todo por `spclient` (`context-resolve` + `extended-metadata`).

## Servicios

| Servicio      | Código            | Local (compose)                          |
|---------------|-------------------|------------------------------------------|
| frontend      | `frontend/`       | http://localhost:3200                    |
| spotify-proxy | `spotify-proxy/`  | http://localhost:3210 (`/health`)        |
| OAuth loopback| —                 | http://127.0.0.1:8989/login (fijo Spotify) |

El puerto `8989` **debe** exponerse tal cual en el host: el client oficial de
Spotify solo tiene whitelisteado `http://127.0.0.1:8989/login` como redirect.

## Dev local

```bash
docker compose up -d --build
# UI: http://localhost:3200 → "continuar con spotify"
```

Sin Docker:

```bash
cd spotify-proxy && go run ./cmd/server   # :8081 + loopback :8989
cd frontend && pnpm install && pnpm dev
```

## Tests backend

```bash
cd spotify-proxy
go test ./internal/spotify/ -run 'TestEscapeQuery|TestSearchSpclientEmptyQuery' -v  # offline
go test ./internal/spotify/ -run TestSpclientLogin -v                                # integración: abre browser, login Premium
```

## Deploy (server)

Cada push a `main` construye y sube las imágenes a GHCR. En el server:

```bash
docker compose pull && docker compose up -d
```

El workflow dispara el webhook de Watchtower para auto-actualizar.
