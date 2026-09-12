import { createContext, useContext, useState, useCallback, useEffect } from 'react'
import { startSpotifyAuth, getSpotifyMe, getSpotifyToken, logoutSpotify as apiLogout } from '../api/spotify'

const AuthContext = createContext(null)
const STORAGE_KEY = 'spotify_user'
const TOKEN_KEY = 'spotify_token'

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

  const persist = (u, token) => {
    setUser(u)
    if (u) localStorage.setItem(STORAGE_KEY, JSON.stringify(u))
    else localStorage.removeItem(STORAGE_KEY)
    if (token) localStorage.setItem(TOKEN_KEY, token)
    else if (!u) localStorage.removeItem(TOKEN_KEY)
    return u
  }

  const loginWithSpotify = useCallback(async () => {
    setLoading(true)
    try {
      const { url } = await startSpotifyAuth()
      window.location.href = url
    } finally {
      setLoading(false)
    }
  }, [])

  const logout = useCallback(async () => {
    try {
      if (user?.id) await apiLogout(user.id)
    } catch {}
    localStorage.removeItem(TOKEN_KEY)
    persist(null)
  }, [user])

  // Al volver del callback Go redirige a /?spotify_linked=1&userId=xxx (Sonora style)
  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    if (params.get('spotify_linked') === '1') {
      const userId = params.get('userId')
      window.history.replaceState({}, '', window.location.pathname)
      // Sonora guarda token en cache file; nosotros lo pedimos a /auth/token y lo guardamos en localStorage
      Promise.all([getSpotifyMe(userId).catch(() => null), getSpotifyToken(userId).catch(() => null)]).then(
        ([me, tok]) => {
          const token = tok?.access_token
          if (me?.id) {
            persist({ id: me.id, display_name: me.display_name, email: me.email, image: me.images?.[0]?.url }, token)
          } else if (userId) {
            persist({ id: userId, display_name: userId }, token)
          }
        },
      )
    }
  }, [])

  return <AuthContext.Provider value={{ user, loading, loginWithSpotify, logout }}>{children}</AuthContext.Provider>
}

export const useAuth = () => useContext(AuthContext)
