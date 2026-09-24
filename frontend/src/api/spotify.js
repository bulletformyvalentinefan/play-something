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

export const logoutSpotify = (userId) => {
  const qs = userId ? `?userId=${encodeURIComponent(userId)}` : ''
  return fetch(`/api/v1/spotify/auth/logout${qs}`, { method: 'POST' }).then((r) => r.json())
}

const checkOk = async (r) => {
  if (r.status === 429) {
    const ra = r.headers.get('Retry-After') || '30'
    throw new Error(`Spotify rate limit — reintenta en ${ra}s`)
  }
  if (!r.ok) throw new Error(await r.text().then((t) => t || `Error ${r.status}`))
  return r.json()
}

export const spotifyPlaylists = (limit = 20, offset = 0) =>
  fetch(`/api/v1/spotify/proxy/me/playlists?limit=${limit}&offset=${offset}`, { headers: authHeader() }).then(checkOk)

export const spotifySearch = (q, type = 'track', limit = 20) =>
  fetch(`/api/v1/spotify/proxy/search?q=${encodeURIComponent(q)}&type=${type}&limit=${limit}`, {
    headers: authHeader(),
  }).then(checkOk)

// Playback en tu dispositivo Spotify (Connect). Se usa cuando el track no
// trae preview_url (spclient no devuelve previews). Requiere Premium y la
// app de Spotify abierta en algún dispositivo.
export const spotifyPlayerState = () =>
  fetch(`/api/v1/spotify/player/status`, { headers: authHeader() }).then(async (r) => {
    if (r.status === 204) return null
    if (!r.ok) throw new Error(await r.text().then((t) => t || `Error ${r.status}`))
    return r.json()
  })

const playResult = async (r) => {
  const text = await r.text()
  let body = null
  try {
    body = text ? JSON.parse(text) : null
  } catch {
    /* respuesta sin JSON (204/404 de Spotify) */
  }
  return { ok: r.ok, status: r.status, retryAfter: r.headers.get('Retry-After'), body }
}

export const spotifyPlayUris = (uris, deviceId) => {
  const qs = deviceId ? `?device_id=${encodeURIComponent(deviceId)}` : ''
  return fetch(`/api/v1/spotify/player/play${qs}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', ...authHeader() },
    body: JSON.stringify({ uris }),
  }).then(playResult)
}

export const spotifyResume = () =>
  fetch(`/api/v1/spotify/player/play`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', ...authHeader() },
    body: JSON.stringify({}),
  }).then(playResult)

export const spotifyPausePlayback = () =>
  fetch(`/api/v1/spotify/player/pause`, {
    method: 'PUT',
    headers: authHeader(),
  }).then(playResult)

export const spotifySeekTo = (ms) =>
  fetch(`/api/v1/spotify/player/seek?position_ms=${Math.round(ms)}`, {
    method: 'PUT',
    headers: authHeader(),
  }).then(playResult)

// URL de audio completo vía nuestro backend (spclient + audio key con tu
// token Premium). El <audio> no manda headers, así que el token va en query.
export const trackStreamUrl = (track) => {
  const t = getToken()
  const uri = track.spotifyUri || (track.id ? `spotify:track:${track.id}` : null)
  if (!t || !uri) return null
  return `/api/v1/spotify/proxy/stream?uri=${encodeURIComponent(uri)}&access_token=${encodeURIComponent(t)}`
}

// Playlists CRUD via token — sin BDD, todo en Spotify con tu token
export const createSpotifyPlaylist = (name, description, isPublic) =>
  fetch(`/api/v1/spotify/proxy/me/playlists`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...authHeader() },
    body: JSON.stringify({ name, description, public: isPublic }),
  }).then(async (r) => {
    if (!r.ok) throw new Error(await r.text())
    return r.json()
  })

export const getSpotifyPlaylist = (id) =>
  fetch(`/api/v1/spotify/proxy/playlists/${encodeURIComponent(id)}`, { headers: authHeader() }).then((r) => r.json())

export const getSpotifyPlaylistTracks = (id) =>
  fetch(`/api/v1/spotify/proxy/playlists/${encodeURIComponent(id)}/tracks`, { headers: authHeader() }).then((r) => r.json())

export const addSpotifyTrack = (playlistId, trackId) =>
  fetch(`/api/v1/spotify/proxy/playlists/${encodeURIComponent(playlistId)}/tracks`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...authHeader() },
    body: JSON.stringify({ uris: [`spotify:track:${trackId}`] }),
  }).then(async (r) => {
    if (!r.ok) throw new Error(await r.text())
    return r.json()
  })

export const removeSpotifyTrack = (playlistId, trackId) =>
  fetch(`/api/v1/spotify/proxy/playlists/${encodeURIComponent(playlistId)}/tracks`, {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json', ...authHeader() },
    body: JSON.stringify({ tracks: [{ uri: `spotify:track:${trackId}` }] }),
  }).then(async (r) => {
    if (!r.ok) throw new Error(await r.text())
    return r.json()
  })

export const deleteSpotifyPlaylist = (id) =>
  fetch(`/api/v1/spotify/proxy/playlists/${encodeURIComponent(id)}`, {
    method: 'DELETE',
    headers: authHeader(),
  }).then(async (r) => {
    if (!r.ok) throw new Error(await r.text())
    return r.json()
  })
