import { Navigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'

export default function AuthPage() {
  const { user, loginWithSpotify, loading } = useAuth()

  if (user) return <Navigate to="/" replace />

  return (
    <section className="auth">
      <h1 className="name">play something</h1>
      <p style={{ opacity: 0.7, marginBottom: '1.5rem', maxWidth: 420 }}>
        Inicia sesión con tu cuenta real de Spotify. Usamos{' '}
        <code>go-librespot</code> para hacernos pasar por cliente oficial y tomar
        tu token (ingeniería inversa) — luego usamos tu token para mostrar tus
        playlists y reproducir con tu cuenta.
      </p>
      <button
        type="button"
        className="theme-btn"
        onClick={loginWithSpotify}
        disabled={loading}
        style={{ padding: '0.9rem 1.6rem', fontSize: '1rem' }}
      >
        {loading ? '…' : 'continuar con spotify'}
      </button>
      <p style={{ opacity: 0.5, marginTop: '1rem', fontSize: '0.8rem' }}>
        Serás redirigido a accounts.spotify.com · scopes: playlist-read, streaming, user-read
      </p>
    </section>
  )
}
