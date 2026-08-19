import { api } from './client'

export const createPlaylist = (payload) => api.post('/playlists', payload)

export const getUserPlaylists = (userId) => api.get(`/users/${userId}/playlists`)

export const getPlaylist = (playlistId) => api.get(`/playlists/${playlistId}`)

export const addTrackToPlaylist = (playlistId, deezerTrackId) =>
  api.post(`/playlists/${playlistId}/tracks`, { deezerTrackId })

export const removeTrackFromPlaylist = (playlistId, trackId) =>
  api.del(`/playlists/${playlistId}/tracks/${trackId}`)

export const deletePlaylist = (playlistId) => api.del(`/playlists/${playlistId}`)