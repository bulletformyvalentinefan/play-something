// Proxy a go-librespot Sonora-style: sin BDD, token en Authorization header
const getToken = () => localStorage.getItem('spotify_token')
const authHeader = () => {
  const t = getToken()
  return t ? { Authorization: `Bearer ${t}` } : {}
}

export const startSpotifyAuth = () =>
  fetch(`/api/v1/spotify/auth/start`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
  }).then(async (r) => {
    if (!r.ok) throw new Error(await r.text())
    return r.json()
  })

export const getSpotifyStatus = () => fetch(`/api/v1/spotify/auth/status`).then((r) => r.json())

export const getSpotifyMe = (userId) => {
  const qs = userId ? `?userId=${encodeURIComponent(userId)}` : ''
  const headers = authHeader()
  return fetch(`/api/v1/spotify/auth/me${qs}`, { headers }).then(async (r) => {
    if (!r.ok) throw new Error('not linked')
    return r.json()
  })
}

export const getSpotifyToken = (userId) => {
  const qs = userId ? `?userId=${encodeURIComponent(userId)}` : ''
  return fetch(`/api/v1/spotify/auth/token${qs}`).then(async (r) => {
    if (!r.ok) throw new Error('not linked')
    return r.json()
  })
}

export const logoutSpotify = (userId) => {
  const qs = userId ? `?userId=${encodeURIComponent(userId)}` : ''
  return fetch(`/api/v1/spotify/auth/logout${qs}`, { method: 'POST' }).then((r) => r.json())
}

export const spotifyMe = () => fetch(`/api/v1/spotify/proxy/me`, { headers: authHeader() }).then((r) => r.json())

export const spotifyPlaylists = (limit = 20, offset = 0) =>
  fetch(`/api/v1/spotify/proxy/me/playlists?limit=${limit}&offset=${offset}`, { headers: authHeader() }).then((r) => r.json())

export const spotifySearch = (q, type = 'track', limit = 20) =>
  fetch(`/api/v1/spotify/proxy/search?q=${encodeURIComponent(q)}&type=${type}&limit=${limit}`, {
    headers: authHeader(),
  }).then((r) => r.json())

export const spotifyPlayerStatus = () =>
  fetch(`/api/v1/spotify/player/status`, { headers: authHeader() }).then((r) => r.json())

export const spotifyPlay = (body, deviceId) => {
  const qs = deviceId ? `?device_id=${deviceId}` : ''
  return fetch(`/api/v1/spotify/player/play${qs}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', ...authHeader() },
    body: JSON.stringify(body),
  }).then((r) => ({ ok: r.ok, status: r.status }))
}
