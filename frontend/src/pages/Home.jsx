import { useRef, useState } from 'react'
import { spotifySearch } from '../api/spotify'
import SearchBar from '../components/SearchBar'
import TrackRow from '../components/TrackRow'
import AddToPlaylistModal from '../components/AddToPlaylistModal'
import { PlayIcon, PauseIcon } from '../components/icons'
import { usePlayer } from '../context/PlayerContext'

function toRow(t) {
  return {
    id: t.id,
    title: t.name,
    artistName: t.artists?.map((a) => a.name).join(', '),
    albumCover: t.album?.images?.[0]?.url,
    duration: Math.round((t.duration_ms || 0) / 1000),
    previewUrl: t.preview_url,
    spotifyUri: t.uri,
    explicit: !!t.explicit,
  }
}

function HeroCard({ track, onAdd, isAdded }) {
  const { play, current, isPlaying } = usePlayer()
  const playing = current?.id === track.id && isPlaying
  return (
    <article className="hero-card">
      {track.albumCover ? (
        <img className="hero-art" src={track.albumCover} alt="" />
      ) : (
        <span className="hero-art placeholder" aria-hidden="true" />
      )}
      <div className="hero-body">
        <span className="hero-kicker">destacado</span>
        <h2 className="hero-title">{track.title}</h2>
        <p className="hero-artist">{track.artistName}</p>
        <div className="row-actions">
          <button type="button" className="play-btn" onClick={() => play(track)} aria-label="Reproducir destacado">
            {playing ? <PauseIcon /> : <PlayIcon />}
          </button>
          <button type="button" className="theme-btn" onClick={() => onAdd(track)} disabled={isAdded}>
            {isAdded ? 'agregada' : 'agregar'}
          </button>
        </div>
      </div>
    </article>
  )
}

export default function Home() {
  const [results, setResults] = useState(null)
  const [query, setQuery] = useState('')
  const [error, setError] = useState(null)
  const [busy, setBusy] = useState(false)
  const [addTrack, setAddTrack] = useState(null)
  const [addedIds, setAddedIds] = useState(() => new Set())
  const { recentlyPlayed } = usePlayer()
  const searchSeq = useRef(0)

  const onSearch = async (q) => {
    const term = q.trim()
    const seq = ++searchSeq.current
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
      const data = await spotifySearch(term)
      if (seq !== searchSeq.current) return
      // Spotify Web API: { tracks: { items: [...] } }
      const items = data?.tracks?.items ?? data?.items ?? []
      setResults(items.map(toRow))
    } catch (err) {
      if (seq !== searchSeq.current) return
      setError(err.message)
      setResults(null)
    } finally {
      if (seq === searchSeq.current) setBusy(false)
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
          {results.length > 0 && (
            <HeroCard track={results[0]} onAdd={() => setAddTrack(results[0])} isAdded={addedIds.has(results[0].id)} />
          )}
          {results.slice(1).map((t) => (
            <TrackRow key={t.id} track={t} isAdded={addedIds.has(t.id)} onAdd={() => setAddTrack(t)} />
          ))}
        </section>
      )}

      {addTrack && <AddToPlaylistModal track={addTrack} onClose={() => setAddTrack(null)} onAdded={onAdded} />}
    </>
  )
}