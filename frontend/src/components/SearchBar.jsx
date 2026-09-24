import { useEffect, useState } from 'react'

export default function SearchBar({ onSearch, query }) {
  const [q, setQ] = useState(query ?? '')

  useEffect(() => {
    if (query !== undefined && query !== q) setQ(query)
  }, [query])

  const handleChange = (value) => {
    setQ(value)
    if (value.trim().length < 2) {
      onSearch('')
      return
    }
    onSearch(value)
  }

  const submit = (e) => {
    e.preventDefault()
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
