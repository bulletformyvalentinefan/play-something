Clon de Spotify con **ingeniería inversa**: el backend está **100% en Go** usando `devgianlu/go-librespot` para hacerse pasar por cliente oficial Spotify, capturar tu `access_token` y proxear a `api.spotify.com`/`spclient.wg.spotify.com`; el frontend en React (pnpm + Vite) ofrece login directo con Spotify.

[![CI Pipeline](https://github.com/bulletformyvalentinefan/play-something/actions/workflows/ci-build.yml/badge.svg)](https://github.com/bulletformyvalentinefan/play-something/actions/workflows/ci-build.yml)
![Go](https://img.shields.io/badge/Go-1.26-blue)
![React](https://img.shields.io/badge/React-19-blue)
![Vite](https://img.shields.io/badge/Vite-8-purple)
![Redis](https://img.shields.io/badge/Redis-7-red)

---

## Arquitectura del Sistema

```mermaid
graph TD
    A[Frontend React 19 / Vite :5173] -->|/auth → :8081| B[Go spotify-proxy :8081]
    B -->|login5 + spclient| C[Spotify (accounts.spotify.com / spclient.wg.spotify.com / api.spotify.com)]
    B -->|Cifrado AES-GCM| D[(Store memoria / Redis)]
    A -->|Cache| E[(Redis 7 :6379)]
```

Login ya no es local: `Continuar con Spotify` redirige a `accounts.spotify.com` con el `client_id` oficial de `go-librespot` (`65b708073fc0480ea92a077233ca87bd`), el Go intercambia `code → tokens`, los cifra (`CREDENTIAL_KEY` 32B) y proxea tus playlists/búsqueda/reproducción con tu token.

---

## Stack Tecnológico

| Capa | Tecnología |
| --- | --- |
| **Frontend** | React 19, Vite 8, React Router 7, pnpm, oxlint |
| **Backend** | Go 1.26, chi, `go-librespot v0.9.1` (ingeniería inversa), AES-GCM |
| **Caché** | Redis 7 |
| **Cliente externo** | Spotify (vía login5/spclient + Web API con tu token) |
| **Contenedores** | Docker Compose (Redis + spotify-proxy) |
| **CI/CD** | GitHub Actions (Go + pnpm) |

---

## Estructura del Repositorio

```text
.
├── spotify-proxy/               # Go service go-librespot
│   ├── cmd/server/              # entrypoint :8081
│   └── internal/{config,crypto,store,auth,handler,spotify}
├── frontend/                    # React 19 + Vite + pnpm
├── docker-compose.yml           # Redis + spotify-proxy
└── .github/workflows/           # CI (Go + pnpm)
```

---

## Puesta en Marcha Local

### 1. Requisitos Previos

- **Docker** y **Docker Compose** instalados.
- **Go 1.26+** instalado.
- **Node.js v20+** y **pnpm 9+** instalados (`npm i -g pnpm`).

### 2. Clonar

```bash
git clone https://github.com/bulletformyvalentinefan/play-something.git
cd play-something
```

### 3. Infra (Redis)

```bash
docker compose up -d redis
```

### 4. Proxy Go (Spotify)

```bash
cd spotify-proxy
# PowerShell (Windows)
$env:PORT="8081"
$env:CREDENTIAL_KEY="0123456789abcdef0123456789abcdef"
$env:FRONTEND_ORIGIN="http://localhost:5173"
# usar 127.0.0.1 NO localhost (Spotify lo bloquea desde 2025)
$env:OAUTH_CALLBACK_URL="http://127.0.0.1:8081/api/v1/spotify/auth/callback"
go run ./cmd/server

# Linux/macOS
PORT=8081 CREDENTIAL_KEY=0123456789abcdef0123456789abcdef OAUTH_CALLBACK_URL=http://127.0.0.1:8081/api/v1/spotify/auth/callback go run ./cmd/server
# con audio CGO: CGO_ENABLED=1 go run ./cmd/server
```

Queda en **http://localhost:8081** (+ loopback `:8989` si usas client oficial) Endpoints: `POST /api/v1/spotify/auth/start`, `GET /auth/callback`, `GET /login` (whitelisted), `GET /auth/me`, `GET /proxy/me`, `/proxy/search`, `/player/*`.

#### ⚠️ redirect_uri: Not matching configuration

Si ves ese error es porque el `client_id` oficial `65b708073fc0480ea92a077233ca87bd` (de `go-librespot`/`Sonora`) **solo** tiene whitelisteado `http://127.0.0.1:8989/login` (ver `Sonora crates/music/src/spotify/auth.rs:9` y `librespot oauth_sync.rs`). `localhost` está bloqueado por Spotify desde 2025 y `/api/v1/...` no está whitelisteado.

**Dos opciones:**

| Opción | Qué hacer |
| --- | --- |
| **A. Seguir con client oficial (sin registrar nada)** | No toques env. El proxy ya levanta `:8989/login` además de `:8081` y fuerza `redirect_uri=http://127.0.0.1:8989/login` automáticamente cuando detecta el client oficial. Solo asegúrate de no tener `:8989` ocupado. |
| **B. Crear tu propia app (recomendado para prod)** | Ve a https://developer.spotify.com/dashboard → Create App → Redirect URIs → añade **exactamente** `http://127.0.0.1:8081/api/v1/spotify/auth/callback` (¡con `127.0.0.1`, no `localhost`!) → copia `Client ID` → arranca con `SPOTIFY_CLIENT_ID=tu_id OAUTH_CALLBACK_URL=http://127.0.0.1:8081/api/v1/spotify/auth/callback` |

Ref: https://developer.spotify.com/documentation/web-api/concepts/redirect_uri

### 5. Frontend

```bash
cd frontend
pnpm install
pnpm dev
```

Frontend en **http://localhost:5173** (Vite proxea `/api/v1/spotify/*` → `:8081`).

> Flujo login: botón **Continuar con Spotify** → `accounts.spotify.com` → Go intercambia `code` y redirige `/?spotify_linked=1&userId=<spotifyId>` → frontend guarda `spotify_user` y ya puedes ver tus playlists reales.

---

## Endpoints (Go)

Base: `/api/v1/spotify` en `:8081`

| Método | Ruta | Descripción |
| --- | --- | --- |
| `POST` | `/auth/start` | Inicia OAuth (devuelve `url` a Spotify) |
| `GET` | `/auth/callback?code=&state=` | Intercambia code, cifra tokens, redirect frontend |
| `GET` | `/auth/me?userId=` | Perfil `/v1/me` proxado |
| `GET` | `/auth/status` | Estado vínculo |
| `GET` | `/proxy/me` | Proxy `api.spotify.com/v1/me` |
| `GET` | `/proxy/me/playlists` | Tus playlists reales |
| `GET` | `/proxy/search?q=&type=track` | Búsqueda real |
| `PUT` | `/player/play` | Proxy `api.spotify.com/v1/me/player/play` |

---

## Notas Legales

Usa APIs internas no documentadas (`spclient`, `login5`, `dealer`) de Spotify — pueden romperse sin aviso y viola ToS. Requiere cuenta Spotify (Premium para Connect). Solo para lab. `go-librespot` es GPL-3.0.
