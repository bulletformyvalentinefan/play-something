import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import {
  addTrackToPlaylist,
  deletePlaylist,
  getPlaylist,
  removeTrackFromPlaylist,
} from '../api/playlists'
import { getTrack, searchTracks } from '../api/tracks'
import TrackRow from '../components/TrackRow'
import SearchBar from '../components/SearchBar'
import ConfirmModal from '../components/ConfirmModal'

export default function PlaylistDetail() {
  const { id } = useParams()
  const navigate = useNavigate()
  const [playlist, setPlaylist] = useState(null)
  const [tracks, setTracks] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [results, setResults] = useState(null)
  const [query, setQuery] = useState('')
  const [searching, setSearching] = useState(false)
  const [busyId, setBusyId] = useState(null)
  const [addedIds, setAddedIds] = useState(() => new Set())
  const [showDelete, setShowDelete] = useState(false)
  const [deleting, setDeleting] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const p = await getPlaylist(id)
      setPlaylist(p)
      const resolved = await Promise.all(p.trackIds.map((tid) => getTrack(tid)))
      setTracks(resolved)
    } catch (e) {
      setError(e.message)
    } finally {
      setLoading(false)
    }
  }, [id])

  useEffect(() => {
    load()
  }, [load])

  const remove = async (trackId) => {
    try {
      await removeTrackFromPlaylist(id, trackId)
      await load()
    } catch (e) {
      setError(e.message)
    }
  }

  const onSearch = async (q) => {
    if (!q.trim()) {
      setResults(null)
      setQuery('')
      setError(null)
      return
    }
    setSearching(true)
    setQuery(q)
    setError(null)
    try {
      setResults(await searchTracks(q))
    } catch (e) {
      setError(e.message)
      setResults(null)
    } finally {
      setSearching(false)
    }
  }

  const addDirect = async (track) => {
    setBusyId(track.id)
    setError(null)
    try {
      await addTrackToPlaylist(id, track.id)
      setAddedIds((prev) => new Set(prev).add(track.id))
      await load()
    } catch (e) {
      setError(e.message)
    } finally {
      setBusyId(null)
    }
  }

  const confirmDelete = async () => {
    setDeleting(true)
    try {
      await deletePlaylist(id)
      navigate('/library')
    } catch (e) {
      setError(e.message)
      setShowDelete(false)
    } finally {
      setDeleting(false)
    }
  }

  const existing = new Set(playlist?.trackIds ?? [])
  const isEmpty = playlist ? playlist.trackIds.length === 0 : false

  return (
    <section className="section">
      <Link to="/library" className="link">
        ← biblioteca
      </Link>

      {playlist && (
        <>
          <div className="section-head" style={{ marginTop: '2rem' }}>
            <h2 className="section-title">
              {playlist.titulo} {playlist.esPublica ? '· pública' : '· privada'}
            </h2>
            <button type="button" className="theme-btn" onClick={() => setShowDelete(true)}>
              eliminar
            </button>
          </div>
          <p className="row-desc">{playlist.descripcion || 'sin descripción'}</p>
        </>
      )}

      {error && <p className="error">{error}</p>}
      {loading && !error && <p className="muted">cargando…</p>}

      {!loading && !error && isEmpty && (
        <div className="empty-state">
          <p className="muted">playlist vacía. busca y agrega música:</p>
          <SearchBar onSearch={onSearch} />
          {searching && <p className="muted">buscando…</p>}
        </div>
      )}

      {!loading && isEmpty && results && results.length > 0 && (
        <section className="section">
          <h2 className="section-title">resultados · {query}</h2>
          {results.map((t) => (
            <TrackRow
              key={t.id}
              track={t}
              isAdded={existing.has(t.id) || addedIds.has(t.id)}
              isBusy={busyId === t.id}
              onAdd={() => addDirect(t)}
            />
          ))}
        </section>
      )}

      {!loading && isEmpty && results && results.length === 0 && !searching && !error && (
        <p className="muted">sin resultados.</p>
      )}

      {!loading && playlist && !isEmpty && (
        <>
          <p className="muted" style={{ marginBottom: '1rem' }}>
            {playlist.trackIds.length} canciones
          </p>
          {tracks.map((t) => (
            <TrackRow
              key={t.id}
              track={t}
              isAdded={existing.has(t.id) || addedIds.has(t.id)}
              isBusy={busyId === t.id}
              onAdd={() => addDirect(t)}
              onRemove={remove}
            />
          ))}
        </>
      )}

      {showDelete && playlist && (
        <ConfirmModal
          title="eliminar playlist"
          message={`se eliminará '${playlist.titulo}' junto con sus canciones.`}
          busy={deleting}
          onConfirm={confirmDelete}
          onClose={() => setShowDelete(false)}
        />
      )}
    </section>
  )
}