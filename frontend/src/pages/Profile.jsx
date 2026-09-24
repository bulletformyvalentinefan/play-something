import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { usePlayer } from '../context/PlayerContext'
import { spotifyPlaylists } from '../api/spotify'
import PlaylistRow from '../components/PlaylistRow'

function toRow(p) {
  return {
    id: p.id,
    titulo: p.name,
    descripcion: p.description || '',
    esPublica: p.public,
    trackIds: [],
    total: p.tracks?.total ?? 0,
  }
}

export default function Profile() {
  const { user } = useAuth()
  const { recentlyPlayed } = usePlayer()
  const [playlists, setPlaylists] = useState([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    spotifyPlaylists(6)
      .then((data) => {
        setPlaylists((data?.items ?? []).map(toRow))
        setTotal(data?.total ?? 0)
        setError(null)
      })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false))
  }, [])

  const initial = (user?.display_name || user?.id || '?').slice(0, 1).toUpperCase()

  return (
    <>
      <section className="section profile-head">
        {user?.image ? (
          <img className="avatar" src={user.image} alt="" />
        ) : (
          <span className="avatar placeholder" aria-hidden="true">
            {initial}
          </span>
        )}
        <div>
          <p className="section-title">perfil</p>
          <h2 className="profile-name">{user?.display_name || user?.id}</h2>
          {user?.email && <p className="muted">{user.email}</p>}
        </div>
      </section>

      <section className="section">
        <div className="stats">
          <div className="stat">
            <span className="stat-num">{total}</span>
            <span className="stat-label">playlists</span>
          </div>
          <div className="stat">
            <span className="stat-num">{recentlyPlayed.length}</span>
            <span className="stat-label">recientes</span>
          </div>
        </div>
      </section>

      <section className="section">
        <div className="section-head">
          <h3 className="section-title">mis playlists</h3>
          <Link to="/library" className="theme-btn">
            ver todas
          </Link>
        </div>
        {loading && <p className="muted">cargando…</p>}
        {error && <p className="error">{error}</p>}
        {!loading && !error && playlists.map((p) => <PlaylistRow key={p.id} playlist={p} />)}
      </section>
    </>
  )
}
