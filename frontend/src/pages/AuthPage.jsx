import { useState } from 'react'
import { Navigate, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'

export default function AuthPage() {
  const { user, login, register } = useAuth()
  const navigate = useNavigate()
  const [mode, setMode] = useState('login')
  const [nombre, setNombre] = useState('')
  const [email, setEmail] = useState('')
  const [error, setError] = useState(null)
  const [busy, setBusy] = useState(false)

  if (user) return <Navigate to="/" replace />

  const switchMode = (next) => {
    setMode(next)
    setError(null)
  }

  const submit = async (e) => {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      if (mode === 'register') await register(nombre, email)
      else await login(email)
      navigate('/')
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <section className="auth">
      <h1 className="name">{mode === 'login' ? 'ingresar' : 'crear cuenta'}</h1>
      <div className="auth-tabs">
        <button
          type="button"
          className={`theme-btn${mode === 'login' ? ' active' : ''}`}
          onClick={() => switchMode('login')}
        >
          ingresar
        </button>
        <button
          type="button"
          className={`theme-btn${mode === 'register' ? ' active' : ''}`}
          onClick={() => switchMode('register')}
        >
          crear cuenta
        </button>
      </div>
      <form className="form" onSubmit={submit}>
        {mode === 'register' && (
          <input
            className="search-input"
            placeholder="nombre"
            value={nombre}
            onChange={(e) => setNombre(e.target.value)}
            minLength={2}
            maxLength={50}
            required
            aria-label="Nombre"
          />
        )}
        <input
          className="search-input"
          type="email"
          placeholder="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
          aria-label="Email"
        />
        {error && <p className="error">{error}</p>}
        <button type="submit" className="theme-btn" disabled={busy}>
          {busy ? '…' : mode === 'login' ? 'ingresar' : 'crear cuenta'}
        </button>
      </form>
    </section>
  )
}