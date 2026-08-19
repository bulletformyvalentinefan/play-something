import { useEffect, useRef, useState } from 'react'

const DEBOUNCE_MS = 400

export default function SearchBar({ onSearch, query }) {
  const [q, setQ] = useState(query ?? '')

  useEffect(() => {
    if (query !== undefined && query !== q) setQ(query)
  }, [query])

  const timerRef = useRef(null)

  useEffect(() => () => clearTimeout(timerRef.current), [])

  const handleChange = (value) => {
    setQ(value)
    clearTimeout(timerRef.current)
    if (!value.trim()) {
      onSearch('')
      return
    }
    timerRef.current = setTimeout(() => onSearch(value), DEBOUNCE_MS)
  }

  const submit = (e) => {
    e.preventDefault()
    clearTimeout(timerRef.current)
    onSearch(q)
  }

  return (
    <form className="search" role="search" onSubmit={submit}>
      <input
        className="search-input"
        type="search"
        placeholder="buscar canción o artista…"
        value={q}
        onChange={(e) => handleChange(e.target.value)}
        aria-label="Buscar canciones"
      />
      <button type="submit" className="theme-btn">
        buscar
      </button>
    </form>
  )
}