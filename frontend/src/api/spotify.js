// Proxy a go-librespot en :8081 (vite lo proxea)
export const startSpotifyAuth = (userId) => {
  const body = userId ? JSON.stringify({ userId }) : undefined
  return fetch(`/api/v1/spotify/auth/start`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    ...(body ? { body } : {}),
  }).then(async (r) => {
    if (!r.ok) throw new Error(await r.text())
    return r.json()
  })
}

export const getSpotifyStatus = (userId) => {
  const qs = userId ? `?userId=${encodeURIComponent(userId)}` : ''
  return fetch(`/api/v1/spotify/auth/status${qs}`).then((r) => r.json())
}

export const getSpotifyMe = (userId) => {
  const qs = userId ? `?userId=${encodeURIComponent(userId)}` : ''
  return fetch(`/api/v1/spotify/auth/me${qs}`).then(async (r) => {
    if (!r.ok) throw new Error('not linked')
    return r.json()
  })
}

export const getSpotifyToken = (userId) =>
  fetch(`/api/v1/spotify/auth/token?userId=${encodeURIComponent(userId)}`).then(async (r) => {
    if (!r.ok) throw new Error('not linked')
    return r.json()
  })

export const logoutSpotify = (userId) => {
  const qs = userId ? `?userId=${encodeURIComponent(userId)}` : ''
  return fetch(`/api/v1/spotify/auth/logout${qs}`, { method: 'POST' }).then((r) => r.json())
}

export const spotifyMe = (userId) => {
  const qs = userId ? `?userId=${encodeURIComponent(userId)}` : ''
  return fetch(`/api/v1/spotify/proxy/me${qs}`).then((r) => r.json())
}

export const spotifyPlaylists = (userId, limit = 20, offset = 0) => {
  const qs = userId ? `userId=${encodeURIComponent(userId)}&` : ''
  return fetch(`/api/v1/spotify/proxy/me/playlists?${qs}limit=${limit}&offset=${offset}`).then((r) => r.json())
}

export const spotifySearch = (userId, q, type = 'track', limit = 20) => {
  const qs = userId ? `userId=${encodeURIComponent(userId)}&` : ''
  return fetch(`/api/v1/spotify/proxy/search?${qs}q=${encodeURIComponent(q)}&type=${type}&limit=${limit}`).then((r) => r.json())
}

export const spotifyPlayerStatus = (userId) => {
  const qs = userId ? `?userId=${encodeURIComponent(userId)}` : ''
  return fetch(`/api/v1/spotify/player/status${qs}`).then((r) => r.json())
}

export const spotifyPlay = (userId, body, deviceId) => {
  const base = userId ? `userId=${userId}` : ''
  const dev = deviceId ? `&device_id=${deviceId}` : ''
  const qs = base || dev ? `?${base}${dev}` : ''
  return fetch(`/api/v1/spotify/player/play${qs}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  }).then((r) => ({ ok: r.ok, status: r.status }))
}
