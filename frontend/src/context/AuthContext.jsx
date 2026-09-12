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

  // Al volver del callback Go redirige a /?spotify_linked=1&userId=xxx&access_token=...&display_name=... (Sonora style)
  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    if (params.get('spotify_linked') === '1') {
      const userId = params.get('userId')
      const displayName = params.get('display_name')
      const accessToken = params.get('access_token')
      window.history.replaceState({}, '', window.location.pathname)
      // Persist irnmediato sin esperar fetch — corrige "vuelve a la web pero no pasa nada"
      if (accessToken) {
        persist({ id: userId, display_name: displayName || userId }, accessToken)
        // Enriquecer en background con /me si hace falta
        getSpotifyMe(userId)
          .then((me) => {
            if (me?.id) persist({ id: me.id, display_name: me.display_name, email: me.email, image: me.images?.[0]?.url }, accessToken)
          })
          .catch(() => {})
        return
      }
      // fallback si Go no mandó token (compat)
      Promise.all([getSpotifyMe(userId).catch(() => null), getSpotifyToken(userId).catch(() => null)]).then(
        ([me, tok]) => {
          const token = tok?.access_token
          if (me?.id) {
            persist({ id: me.id, display_name: me.display_name, email: me.email, image: me.images?.[0]?.url }, token)
          } else if (userId) {
            persist({ id: userId, display_name: displayName || userId }, token)
          }
        },
      )
    }
  }, [])

  return <AuthContext.Provider value={{ user, loading, loginWithSpotify, logout }}>{children}</AuthContext.Provider>
}

export const useAuth = () => useContext(AuthContext)
