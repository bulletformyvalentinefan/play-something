import { usePlayer } from '../context/PlayerContext'
import { formatDuration } from '../utils/format'
import { PlayIcon, PauseIcon } from './icons'

export default function TrackRow({ track, isAdded, isBusy, onAdd, onRemove }) {
  const { current, isPlaying, play } = usePlayer()
  const isCurrent = current?.id === track.id
  const playing = isCurrent && isPlaying

  return (
    <article className="row">
      <div className="row-meta">
        <span className="year">{formatDuration(track.duration)}</span>
      </div>
      <div className="row-content">
        <div className="track-main">
          {track.albumCover ? (
            <img className="cover" src={track.albumCover} alt="" loading="lazy" />
          ) : (
            <span className="cover placeholder" aria-hidden="true" />
          )}
          <div className="row-text">
            <h3 className="row-title">{track.title}</h3>
            <p className="row-desc">{track.artistName}</p>
          </div>
        </div>
        <div className="row-actions">
          <button
            type="button"
            className="icon-btn"
            onClick={() => play(track)}
            aria-label={playing ? 'Pausar' : 'Reproducir'}
          >
            {playing ? <PauseIcon /> : <PlayIcon />}
          </button>
          {onAdd && (
            <button
              type="button"
              className="theme-btn"
              onClick={() => onAdd(track)}
              disabled={isAdded || isBusy}
            >
              {isBusy ? 'agregando…' : isAdded ? 'agregada' : 'agregar'}
            </button>
          )}
          {onRemove && (
            <button type="button" className="theme-btn" onClick={() => onRemove(track.id)}>
              quitar
            </button>
          )}
        </div>
      </div>
    </article>
  )
}