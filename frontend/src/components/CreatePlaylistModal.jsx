import { useState } from 'react'
import { createPlaylist } from '../api/playlists'
import Modal from './Modal'

export default function CreatePlaylistModal({ user, onClose, onCreated }) {
  const [titulo, setTitulo] = useState('')
  const [descripcion, setDescripcion] = useState('')
  const [esPublica, setEsPublica] = useState(true)
  const [error, setError] = useState(null)
  const [busy, setBusy] = useState(false)

  const submit = async (e) => {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      const playlist = await createPlaylist({ userId: user.id, titulo, descripcion, esPublica })
      onCreated(playlist)
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal title="nueva playlist" onClose={onClose}>
      <form className="form" onSubmit={submit}>
        <label className="field">
          <span className="field-label">título</span>
          <input
            className="search-input"
            placeholder="nombre de la playlist"
            value={titulo}
            onChange={(e) => setTitulo(e.target.value)}
            required
            autoFocus
            aria-label="Título de la playlist"
          />
        </label>

        <label className="field">
          <span className="field-label">descripción</span>
          <input
            className="search-input"
            placeholder="opcional"
            value={descripcion}
            onChange={(e) => setDescripcion(e.target.value)}
            aria-label="Descripción de la playlist"
          />
        </label>

        <div className="field">
          <span className="field-label">visibilidad</span>
          <div className="segment" role="group" aria-label="Visibilidad de la playlist">
            <button
              type="button"
              className={`theme-btn${esPublica ? ' active' : ''}`}
              onClick={() => setEsPublica(true)}
            >
              pública
            </button>
            <button
              type="button"
              className={`theme-btn${!esPublica ? ' active' : ''}`}
              onClick={() => setEsPublica(false)}
            >
              privada
            </button>
          </div>
        </div>

        {error && <p className="error">{error}</p>}
        <button type="submit" className="primary-btn" disabled={busy}>
          {busy ? 'creando…' : 'crear playlist'}
        </button>
      </form>
    </Modal>
  )
}