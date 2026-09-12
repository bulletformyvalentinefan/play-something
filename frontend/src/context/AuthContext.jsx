import { createContext, useContext, useState, useCallback, useEffect } from 'react'
import { startSpotifyAuth, getSpotifyMe, getSpotifyToken, logoutSpotify as apiLogout } from '../api/spotify'

const AuthContext = createContext(null)
const STORAGE_KEY = 'spotify_user'
const TOKEN_KEY = 'spotify_token'

function getInitialUser() {
  try {
    const params = new URLSearchParams(window.location.search)
    if (params.get('spotify_linked') === '1') {
      const userId = params.get('userId')
      const displayName = params.get('display_name')
      const accessToken = params.get('access_token')
      if (userId) {
        const u = { id: userId, display_name: displayName || userId }
        localStorage.setItem(STORAGE_KEY, JSON.stringify(u))
        if (accessToken) localStorage.setItem(TOKEN_KEY, accessToken)
        // limpiar query antes de que ProtectedRoute evalue
        window.history.replaceState({}, '', window.location.pathname)
        return u
      }
    }
  } catch {}
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    return raw ? JSON.parse(raw) : null
  } catch {
    return null
  }
}

export function AuthProvider({ children }) {
  const [user, setUser] = useState(() => getInitialUser())
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

  // Enriquecer perfil en background si venimos de redirect (ya persistimos sync en getInitialUser)
  useEffect(() => {
    const token = localStorage.getItem(TOKEN_KEY)
    if (user?.id && token) {
      getSpotifyMe(user.id)
        .then((me) => {
          if (me?.id && me.display_name !== user.display_name) {
            persist({ id: me.id, display_name: me.display_name, email: me.email, image: me.images?.[0]?.url }, token)
          }
        })
        .catch(() => {})
    }
  }, [])

  return <AuthContext.Provider value={{ user, loading, loginWithSpotify, logout }}>{children}</AuthContext.Provider>
}

export const useAuth = () => useContext(AuthContext)
