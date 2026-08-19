import { createContext, useContext, useState } from 'react'
import { register as apiRegister, login as apiLogin } from '../api/users'

const AuthContext = createContext(null)

const STORAGE_KEY = 'spotify_user'

export function AuthProvider({ children }) {
  const [user, setUser] = useState(() => {
    try {
      return JSON.parse(localStorage.getItem(STORAGE_KEY))
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
  }

  return <AuthContext.Provider value={{ user, login, register, logout }}>{children}</AuthContext.Provider>
}

export const useAuth = () => useContext(AuthContext)