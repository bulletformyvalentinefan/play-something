import { createContext, useContext, useState, useCallback, useEffect } from 'react'
import { startSpotifyAuth, getSpotifyMe, logoutSpotify as apiLogout } from '../api/spotify'

const AuthContext = createContext(null)
const STORAGE_KEY = 'spotify_user'

export function AuthProvider({ children }) {
  const [user, setUser] = useState(() => {
    try {
      const raw = localStorage.getItem(STORAGE_KEY)
      return raw ? JSON.parse(raw) : null
    } catch {
      return null
    }
  })
  const [loading, setLoading] = useState(false)

  const persist = (u) => {
    setUser(u)
    if (u) localStorage.setItem(STORAGE_KEY, JSON.stringify(u))
    else localStorage.removeItem(STORAGE_KEY)
    return u
  }

  // Login directo con Spotify (go-librespot): no hay form local
  const loginWithSpotify = useCallback(async () => {
    setLoading(true)
    try {
      const { url } = await startSpotifyAuth()
      // Redirige directo a Spotify (accounts.spotify.com) con client_id oficial
      window.location.href = url
    } finally {
      setLoading(false)
    }
  }, [])

  const logout = useCallback(async () => {
    try {
      if (user?.id) await apiLogout(user.id)
    } catch {
      // ignore
    }
    persist(null)
  }, [user])

  const refresh = useCallback(async () => {
    try {
      const me = await getSpotifyMe(user?.id)
      if (me?.id) {
        const u = { id: me.id, display_name: me.display_name, email: me.email, image: me.images?.[0]?.url }
        persist(u)
        return u
      }
    } catch {
      // not linked
    }
    return null
  }, [user])

  // Al volver del callback Go redirige a /?spotify_linked=1&userId=xxx
  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    if (params.get('spotify_linked') === '1') {
      const userId = params.get('userId')
      window.history.replaceState({}, '', window.location.pathname)
      // fetchear perfil real y persistir
      getSpotifyMe(userId)
        .then((me) => {
          if (me?.id) {
            persist({ id: me.id, display_name: me.display_name, email: me.email, image: me.images?.[0]?.url })
          } else if (userId) {
            persist({ id: userId, display_name: userId })
          }
        })
        .catch(() => {
          if (userId) persist({ id: userId, display_name: userId })
        })
    }
  }, [])

  // Validar sesión al montar si hay user guardado
  useEffect(() => {
    if (user?.id) {
      getSpotifyMe(user.id).then((me) => {
        if (!me?.id) {
          // token expirado o revocado
          // no auto-logout agresivo, solo refrescar
        }
      }).catch(() => {})
    }
  }, [])

  return <AuthContext.Provider value={{ user, loading, loginWithSpotify, logout, refresh }}>{children}</AuthContext.Provider>
}

export const useAuth = () => useContext(AuthContext)
