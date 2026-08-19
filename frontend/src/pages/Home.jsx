import { useState } from 'react'
import { searchTracks } from '../api/tracks'
import SearchBar from '../components/SearchBar'
import TrackRow from '../components/TrackRow'
import AddToPlaylistModal from '../components/AddToPlaylistModal'
import { usePlayer } from '../context/PlayerContext'

export default function Home() {
  const [results, setResults] = useState(null)
  const [query, setQuery] = useState('')
  const [error, setError] = useState(null)
  const [busy, setBusy] = useState(false)
  const [addTrack, setAddTrack] = useState(null)
  const [addedIds, setAddedIds] = useState(() => new Set())
  const { recentlyPlayed } = usePlayer()

  const onSearch = async (q) => {
    const term = q.trim()
    if (!term) {
      setResults(null)
      setQuery('')
      setError(null)
      return
    }
    setBusy(true)
    setError(null)
    setQuery(term)
    try {
      setResults(await searchTracks(term))
    } catch (err) {
      setError(err.message)
      setResults(null)
    } finally {
      setBusy(false)
    }
  }

  const onAdded = () => {
    if (addTrack) setAddedIds((prev) => new Set(prev).add(addTrack.id))
    setAddTrack(null)
  }

  return (
    <>
      <section className="hero">
        <SearchBar onSearch={onSearch} query={query} />
      </section>

      {error && <p className="error">{error}</p>}
      {busy && <p className="muted">buscando…</p>}

      {!results && !query.trim() && recentlyPlayed.length > 0 && (
        <section className="section recents">
          <h2 className="section-title">reproducidas recientemente</h2>
          {recentlyPlayed.map((t) => (
            <TrackRow key={t.id} track={t} isAdded={addedIds.has(t.id)} onAdd={() => setAddTrack(t)} />
          ))}
        </section>
      )}

      {results && (
        <section className="section">
          <h2 className="section-title">resultados · {query}</h2>
          {results.length === 0 && <p className="muted">sin resultados.</p>}
          {results.map((t) => (
            <TrackRow key={t.id} track={t} isAdded={addedIds.has(t.id)} onAdd={() => setAddTrack(t)} />
          ))}
        </section>
      )}

      {addTrack && <AddToPlaylistModal track={addTrack} onClose={() => setAddTrack(null)} onAdded={onAdded} />}
    </>
  )
}