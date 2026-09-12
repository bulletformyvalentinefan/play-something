import { useEffect } from 'react'
import { Link, NavLink, useLocation } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { usePlayer } from '../context/PlayerContext'
import { useTheme } from '../hooks/useTheme'

export default function Header() {
  const { user, logout, spotifyLinked, linkSpotify, refreshSpotifyStatus } = useAuth()
  const { theme, toggle } = useTheme()
  const { current } = usePlayer()
  const location = useLocation()
  const isHome = location.pathname === '/'

  useEffect(() => {
    document.title = current ? `playing ${current.title}` : 'play something'
  }, [current])

  return (
    <header className="header">
      <div className="header-inner">
        <Link to="/" className="brand">
          {current ? `playing ${current.title}` : 'play something'}
        </Link>
        <nav className="nav">
          {!isHome && (
            <NavLink to="/" className={({ isActive }) => `nav-link${isActive ? ' active' : ''}`} end>
              buscar
            </NavLink>
          )}
          <NavLink to="/library" className={({ isActive }) => `nav-link${isActive ? ' active' : ''}`}>
            biblioteca
          </NavLink>
        </nav>
        <div className="header-actions">
          <button type="button" className="theme-btn" onClick={toggle} aria-label="Cambiar tema">
            {theme === 'dark' ? 'claro' : 'oscuro'}
          </button>
          {user && !spotifyLinked?.linked && (
            <button
              type="button"
              className="theme-btn"
              onClick={async () => {
                try {
                  await linkSpotify()
                } catch (e) {
                  alert(e.message)
                }
              }}
              title="Conectar tu cuenta real de Spotify (ingeniería inversa go-librespot)"
            >
              conectar spotify
            </button>
          )}
          {user && spotifyLinked?.linked && (
            <button type="button" className="theme-btn" onClick={refreshSpotifyStatus} title={spotifyLinked.spotifyUsername}>
              ✓ {spotifyLinked.spotifyUsername}
            </button>
          )}
          {user && (
            <button type="button" className="theme-btn" onClick={logout} title={user.email}>
              salir
            </button>
          )}
        </div>
      </div>
    </header>
  )
}