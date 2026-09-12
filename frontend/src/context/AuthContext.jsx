import { createContext, useContext, useState, useCallback } from 'react'
import { register as apiRegister, login as apiLogin } from '../api/users'
import { startSpotifyAuth, getSpotifyStatus } from '../api/spotify'

const AuthContext = createContext(null)

const STORAGE_KEY = 'spotify_user'
const SPOTIFY_KEY = 'spotify_linked'

export function AuthProvider({ children }) {
  const [user, setUser] = useState(() => {
    try {
      return JSON.parse(localStorage.getItem(STORAGE_KEY))
    } catch {
      return null
    }
  })
  const [spotifyLinked, setSpotifyLinked] = useState(() => {
    try {
      return JSON.parse(localStorage.getItem(SPOTIFY_KEY))
    } catch {
      return null
    }
  })

  const persist = (u) => {
    setUser(u)
    localStorage.setItem(STORAGE_KEY, JSON.stringify(u))
    return u
  }

  const register = async (nombre, email) => persist(await apiRegister({ nombre, email }))

  const login = async (email) => persist(await apiLogin({ email }))

  const logout = () => {
    setUser(null)
    localStorage.removeItem(STORAGE_KEY)
    setSpotifyLinked(null)
    localStorage.removeItem(SPOTIFY_KEY)
  }

  const linkSpotify = useCallback(async () => {
    if (!user?.id) throw new Error('Debes loguearte primero')
    const { url } = await startSpotifyAuth(user.id)
    // abrir popup OAuth (redirect con client_id oficial 65b...)
    const w = window.open(url, 'spotify-auth', 'width=500,height=700')
    if (!w) window.location.href = url
    return url
  }, [user])

  const refreshSpotifyStatus = useCallback(async () => {
    if (!user?.id) return null
    const s = await getSpotifyStatus(user.id)
    setSpotifyLinked(s)
    localStorage.setItem(SPOTIFY_KEY, JSON.stringify(s))
    return s
  }, [user])

  // auto-check al montar y cuando vuelve de redirect ?spotify_linked=1
  // el callback del Go redirige a /?spotify_linked=1&userId=...
  if (typeof window !== 'undefined' && window.location.search.includes('spotify_linked=1') && user?.id) {
    // defer para no tocar estado durante render
    setTimeout(() => refreshSpotifyStatus(), 0)
    window.history.replaceState({}, '', window.location.pathname)
  }

  return <AuthContext.Provider value={{ user, login, register, logout, spotifyLinked, linkSpotify, refreshSpotifyStatus }}>{children}</AuthContext.Provider>
}

export const useAuth = () => useContext(AuthContext)