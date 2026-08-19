import { useCallback, useEffect, useState } from 'react'
import { deletePlaylist, getUserPlaylists } from '../api/playlists'
import { useAuth } from '../context/AuthContext'
import PlaylistRow from '../components/PlaylistRow'
import CreatePlaylistModal from '../components/CreatePlaylistModal'
import ConfirmModal from '../components/ConfirmModal'

export default function Library() {
  const { user } = useAuth()
  const [playlists, setPlaylists] = useState([])
  const [loading, setLoading] = useState(true)
  const [showCreate, setShowCreate] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState(null)
  const [deleting, setDeleting] = useState(false)
  const [error, setError] = useState(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      setPlaylists(await getUserPlaylists(user.id))
      setError(null)
    } catch (e) {
      setError(e.message)
    } finally {
      setLoading(false)
    }
  }, [user.id])

  useEffect(() => {
    load()
  }, [load])

  const confirmDelete = async () => {
    if (!deleteTarget) return
    setDeleting(true)
    try {
      await deletePlaylist(deleteTarget.id)
      setDeleteTarget(null)
      await load()
    } catch (e) {
      setError(e.message)
      setDeleteTarget(null)
    } finally {
      setDeleting(false)
    }
  }

  return (
    <>
      <section className="section">
        <div className="section-head">
          <h2 className="section-title">
            biblioteca{playlists.length > 0 ? ` · ${playlists.length} playlists` : ''}
          </h2>
          {playlists.length > 0 && (
            <button type="button" className="theme-btn" onClick={() => setShowCreate(true)}>
              + nueva playlist
            </button>
          )}
        </div>

        {error && <p className="error">{error}</p>}
        {loading && !error && <p className="muted">cargando…</p>}

        {!loading && !error && playlists.length === 0 && (
          <div className="empty-state">
            <p className="muted">todavía no tienes playlists.</p>
            <button type="button" className="primary-btn" onClick={() => setShowCreate(true)}>
              crear tu primera playlist
            </button>
          </div>
        )}

        {playlists.map((p) => (
          <PlaylistRow key={p.id} playlist={p} onDelete={setDeleteTarget} />
        ))}
      </section>

      {showCreate && (
        <CreatePlaylistModal
          user={user}
          onClose={() => setShowCreate(false)}
          onCreated={() => {
            setShowCreate(false)
            load()
          }}
        />
      )}

      {deleteTarget && (
        <ConfirmModal
          title="eliminar playlist"
          message={`se eliminará '${deleteTarget.titulo}' junto con sus canciones.`}
          busy={deleting}
          onConfirm={confirmDelete}
          onClose={() => setDeleteTarget(null)}
        />
      )}
    </>
  )
}