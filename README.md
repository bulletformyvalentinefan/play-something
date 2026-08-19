# 🎵 Spotify Clone — Fullstack Event-Driven

> Plataforma de streaming de música con búsqueda de canciones (previews), gestión de playlists, reproducción con eventos asíncronos y caché distribuida.

Clon funcional de Spotify construido con una arquitectura de **microservicios orientada a eventos**: el backend en Spring Boot sirve una API REST que consulta el catálogo de Deezer, cachea respuestas en Redis y emite métricas de reproducción hacia Kafka; el frontend en React ofrece una interfaz editorial en blanco y negro con tema claro/oscuro.

[![CI Pipeline](https://github.com/bulletformyvalentinefan/play-something/actions/workflows/ci-build.yml/badge.svg)](https://github.com/bulletformyvalentinefan/play-something/actions/workflows/ci-build.yml)
![Java](https://img.shields.io/badge/Java-21-orange)
![Spring Boot](https://img.shields.io/badge/Spring_Boot-3.4.3-green)
![React](https://img.shields.io/badge/React-19-blue)
![Vite](https://img.shields.io/badge/Vite-8-purple)
![Oracle](https://img.shields.io/badge/Oracle-23c-red)
![Redis](https://img.shields.io/badge/Redis-7-red)
![Kafka](https://img.shields.io/badge/Kafka-Event_Driven-black)

---

## 📐 Arquitectura del Sistema

```mermaid
graph TD
    A[Frontend React 19 / Vite] -->|HTTP / REST /api| B[Spring Boot API :8080]
    B -->|JPA / Hibernate| C[(Oracle DB 23c)]
    B -->|Cache TTL 1h| D[(Redis 7)]
    B -->|RestClient| E[Deezer API]
    B -->|TrackPlayedEvent| F[Kafka Broker]
    F -->|Consumidor de analítica| G[TrackEventConsumer]
```

El flujo de reproducción es **asíncrono**: el frontend solicita el track, el backend devuelve la URL de preview y, en paralelo, publica un `TrackPlayedEvent` en el tópico `track-played-topic` que un consumidor procesa para analítica sin bloquear la respuesta.

---

## 🛠️ Stack Tecnológico

| Capa | Tecnología |
| --- | --- |
| **Frontend** | React 19, Vite 8, React Router 7, JavaScript (oxlint) |
| **Backend** | Java 21, Spring Boot 3.4.3 (Web, Data JPA, Validation), Lombok |
| **Base de Datos** | Oracle Database 23c Free (JDBC thin) |
| **Caché** | Redis 7 — TTL de 1 hora e invalidación declarativa (`@Cacheable`, `@CacheEvict`) |
| **Mensajería / Streaming** | Apache Kafka 7.6 (modo KRaft) |
| **Cliente externo** | Deezer API (búsqueda y previews de tracks) |
| **Contenedores** | Docker Compose (Oracle, Redis, Kafka) |
| **CI/CD** | GitHub Actions (compilación y empaquetado con Maven) |

---

## 🗂️ Estructura del Repositorio

```text
.
├── src/main/java/…              # Backend Spring Boot (dominios: user, track, playlist)
├── src/main/resources/          # application.yml (configuración de arranque)
├── frontend/                    # Aplicación React + Vite
├── docker-compose.yml           # Oracle, Redis y Kafka
└── .github/workflows/           # Pipeline de CI (Java CI/CD Pipeline)
```

---

## 🚀 Puesta en Marcha Local

### 1. Requisitos Previos

- **Docker** y **Docker Compose** instalados.
- **JDK 21+** instalado.
- **Node.js v20+** instalado (se usa npm).

### 2. Clonar el Repositorio

```bash
git clone https://github.com/bulletformyvalentinefan/play-something.git
cd play-something
```

### 3. Levantar la Infraestructura (Oracle, Redis y Kafka)

```bash
docker compose up -d
```

Se levantan tres servicios en segundo plano:

| Servicio | Imagen | Puerto |
| --- | --- | --- |
| Oracle DB | `gvenzl/oracle-free:23-slim` | `1521` |
| Redis | `redis:7-alpine` | `6379` |
| Kafka | `confluentinc/cp-kafka:7.6.0` | `9092` |

> La configuración de arranque del backend vive en `src/main/resources/application.yml`, con credenciales por defecto que coinciden con el `docker-compose.yml` (base `FREEPDB1`, usuario `spotify_user`). No es necesario crear archivos `.env` para un arranque local.

### 4. Ejecutar el Backend

```bash
# Linux / macOS
./mvnw spring-boot:run

# Windows
mvnw.cmd spring-boot:run
```

El backend quedará expuesto en **http://localhost:8080** con base de rutas `/api/v1/spotify`.

### 5. Ejecutar el Frontend

```bash
cd frontend
npm install
npm run dev
```

El frontend quedará accesible en **http://localhost:5173**. Vite redirige las peticiones `/api` hacia el backend en el puerto 8080 (no se requiere configuración CORS).

---

## 📡 Endpoints Principales (Contratos de API)

Base: `/api/v1/spotify` · Formato de respuesta: JSON · Errores: `400` validación, `404` no encontrado.

| Método | Ruta | Descripción | Caché / Asíncrono |
| --- | --- | --- | --- |
| `POST` | `/auth/register` | Registrar un usuario | — |
| `POST` | `/auth/login` | Iniciar sesión | — |
| `GET` | `/user/{id}` | Obtener un usuario | — |
| `GET` | `/tracks/search?q={query}` | Buscar tracks en el catálogo de Deezer | Redis (`@Cacheable` `track_searches`) |
| `GET` | `/tracks/{trackId}` | Detalle de un track | Redis (`@Cacheable` `tracks`) |
| `POST` | `/tracks/{trackId}/play?userId={userId}` | Obtener el audio y emitir la métrica de reproducción | Kafka (`track-played-topic`) |
| `POST` | `/playlists` | Crear una playlist | Redis (`@CachePut` / `@CacheEvict`) |
| `GET` | `/users/{userId}/playlists` | Listar playlists de un usuario | Redis (`@Cacheable` `user_playlists`) |
| `GET` | `/playlists/{playlistId}` | Detalle de una playlist | Redis (`@Cacheable` `playlist`) |
| `POST` | `/playlists/{playlistId}/tracks` | Añadir un track a la playlist | Redis (invalidación) |
| `DELETE` | `/playlists/{playlistId}/tracks/{trackId}` | Quitar un track de la playlist | Redis (invalidación) |
| `DELETE` | `/playlists/{playlistId}` | Eliminar una playlist | Redis (invalidación) |

**Ejemplo — crear una playlist:**

```bash
curl -X POST http://localhost:8080/api/v1/spotify/playlists \
  -H "Content-Type: application/json" \
  -d '{"userId":"<uuid>","titulo":"Mi playlist","descripcion":"favoritas","esPublica":true}'
```

**Ejemplo — registrar un usuario:**

```bash
curl -X POST http://localhost:8080/api/v1/spotify/auth/register \
  -H "Content-Type: application/json" \
  -d '{"nombre":"Ana","email":"ana@mail.com"}'
```

> ⚠️ No hay Swagger/OpenAPI habilitado actualmente; los contratos se documentan en esta tabla.

---

## 🧠 Caché y Eventos

### Caché (Redis)

| Caché | Clave | TTL | Uso |
| --- | --- | --- | --- |
| `track_searches` | query | 1 h | Respuestas de búsqueda de tracks |
| `tracks` | trackId | 1 h | Detalle individual de tracks |
| `playlist` | playlistId | 1 h | Detalle de playlists |
| `user_playlists` | userId | 1 h | Listado de playlists por usuario |

Las escrituras invalidan las entradas afectadas mediante `@CachePut`/`@CacheEvict` para mantener consistencia (por ejemplo, añadir/quitar tracks o eliminar una playlist limpia `playlist` y `user_playlists`).

### Eventos (Kafka)

- **Tópico:** `track-played-topic`
- **Productor:** `TrackEventProducer` — emite `TrackPlayedEvent` (`trackId`, `userId`, `playedAt`) al reproducir un track.
- **Consumidor:** `TrackEventConsumer` — procesa la analítica de reproducción sin afectar la latencia de la API.

---

## 🧩 Buenas Prácticas

1. **Sin secretos en el repositorio:** las credenciales viven en `docker-compose.yml` y `application.yml` (solo valores de desarrollo). Nunca se suben tokens ni claves de API.
2. **Commits semánticos:** la historia usa [Conventional Commits](https://www.conventionalcommits.org/) (`feat:`, `fix:`, `chore:`, `docs:`).
3. **CI automático:** cada push a `main`/`features` compila y empaqueta el backend con GitHub Actions.

---

## 🗺️ Próximos Pasos

- Habilitar Swagger UI / OpenAPI para documentación viva de la API.
- Añadir autenticación con JWT y protección por usuario de las playlists.
- Pipeline de CI para el frontend (lint + build + deploy).
- Tests de integración con Testcontainers (Oracle, Redis y Kafka).
- Definir `LICENSE` y `CONTRIBUTING.md`.

---

## 📄 Licencia

Proyecto de carácter educativo; licencia aún por definir.