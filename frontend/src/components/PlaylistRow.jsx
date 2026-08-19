import { Link } from 'react-router-dom'
import { NoteIcon } from './icons'

export default function PlaylistRow({ playlist, onDelete }) {
  return (
    <article className="row">
      <div className="row-meta">
        <span className="year">{playlist.esPublica ? 'pública' : 'privada'}</span>
        <span className="tag">{playlist.trackIds.length} canciones</span>
      </div>
      <div className="row-content">
        <div className="track-main">
          <span className="row-cover" aria-hidden="true">
            <NoteIcon />
          </span>
          <Link to={`/playlists/${playlist.id}`} className="row-link">
            <h3 className="row-title">{playlist.titulo}</h3>
            <p className="row-desc">{playlist.descripcion || 'sin descripción'}</p>
          </Link>
        </div>
        {onDelete && (
          <div className="row-actions">
            <button type="button" className="theme-btn" onClick={() => onDelete(playlist)}>
              eliminar
            </button>
          </div>
        )}
      </div>
    </article>
  )
}