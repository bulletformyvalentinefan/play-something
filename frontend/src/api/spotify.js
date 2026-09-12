// Proxy a go-librespot (ing. inversa) en :8081
// Si vite proxy está configurado, esto va por /api; si no, directo a 8081 con CORS.
const PROXY_BASE = '' // relativo, vite lo proxea a 8081 para /auth y /proxy

// Inicia flujo OAuth redirect (requiere userId local)
export const startSpotifyAuth = (userId) =>
  fetch(`/api/v1/spotify/auth/start`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ userId }),
  }).then(async (r) => {
    if (!r.ok) throw new Error(await r.text())
    return r.json()
  })

export const getSpotifyStatus = (userId) =>
  fetch(`/api/v1/spotify/auth/status?userId=${encodeURIComponent(userId)}`).then((r) => r.json())

export const getSpotifyToken = (userId) =>
  fetch(`/api/v1/spotify/auth/token?userId=${encodeURIComponent(userId)}`).then(async (r) => {
    if (!r.ok) throw new Error('no vinculado')
    return r.json()
  })

export const logoutSpotify = (userId) =>
  fetch(`/api/v1/spotify/auth/logout?userId=${encodeURIComponent(userId)}`, { method: 'POST' }).then((r) => r.json())

// Proxy a api.spotify.com vía Go (token resuelto por userId en backend)
export const spotifyMe = (userId) =>
  fetch(`/api/v1/spotify/proxy/me?userId=${encodeURIComponent(userId)}`).then((r) => r.json())

export const spotifyPlaylists = (userId, limit = 20, offset = 0) =>
  fetch(`/api/v1/spotify/proxy/me/playlists?userId=${encodeURIComponent(userId)}&limit=${limit}&offset=${offset}`).then((r) => r.json())

export const spotifySearch = (userId, q, type = 'track', limit = 20) =>
  fetch(`/api/v1/spotify/proxy/search?userId=${encodeURIComponent(userId)}&q=${encodeURIComponent(q)}&type=${type}&limit=${limit}`).then((r) => r.json())

export const spotifyPlayerStatus = (userId) =>
  fetch(`/api/v1/spotify/player/status?userId=${encodeURIComponent(userId)}`).then((r) => r.json())

export const spotifyPlay = (userId, body, deviceId) => {
  const qs = deviceId ? `?userId=${userId}&device_id=${deviceId}` : `?userId=${userId}`
  return fetch(`/api/v1/spotify/player/play${qs}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  }).then((r) => ({ ok: r.ok, status: r.status }))
}
