import { api } from './client'

export const searchTracks = (q) => api.get(`/tracks/search?q=${encodeURIComponent(q)}`)

export const getTrack = (trackId) => api.get(`/tracks/${trackId}`)

export const playTrack = (trackId, userId) => api.post(`/tracks/${trackId}/play?userId=${userId}`)