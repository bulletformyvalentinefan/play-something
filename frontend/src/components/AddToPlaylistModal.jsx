import { useEffect, useState } from 'react'
import { getUserPlaylists, addTrackToPlaylist } from '../api/playlists'
import { useAuth } from '../context/AuthContext'
import Modal from './Modal'

export default function AddToPlaylistModal({ track, onClose, onAdded }) {
  const { user } = useAuth()
  const [playlists, setPlaylists] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [busyId, setBusyId] = useState(null)

  useEffect(() => {
    getUserPlaylists(user.id)
      .then((data) => setPlaylists(data))
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false))
  }, [user.id])

  const add = async (playlist) => {
    setBusyId(playlist.id)
    setError(null)
    try {
      await addTrackToPlaylist(playlist.id, track.id)
      onAdded()
    } catch (e) {
      setError(e.message)
    } finally {
      setBusyId(null)
    }
  }

  return (
    <Modal title={`agregar a playlist — ${track.title}`} onClose={onClose}>
      {error && <p className="error">{error}</p>}
      {loading && <p className="muted">cargando…</p>}
      {!loading && playlists.length === 0 && <p className="muted">aún no tienes playlists.</p>}
      <ul className="modal-list">
        {playlists.map((p) => (
          <li key={p.id}>
            <button type="button" className="link" onClick={() => add(p)} disabled={busyId === p.id}>
              {busyId === p.id ? 'agregando…' : p.titulo}
            </button>
          </li>
        ))}
      </ul>
    </Modal>
  )
}