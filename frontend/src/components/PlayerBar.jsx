import { useState } from 'react'
import { usePlayer } from '../context/PlayerContext'
import { formatDuration } from '../utils/format'
import { PlayIcon, PauseIcon } from './icons'

export default function PlayerBar() {
  const { current, isPlaying, progress, duration, toggle, seek } = usePlayer()
  const [dragging, setDragging] = useState(false)
  const [dragValue, setDragValue] = useState(null)

  if (!current) return null

  const shown = dragging && dragValue !== null ? dragValue : progress
  const fillPct = duration > 0 ? Math.min(100, Math.max(0, (shown / duration) * 100)) : 0

  const onScrub = (e) => {
    setDragging(true)
    setDragValue(Number(e.target.value))
  }

  const onScrubEnd = (e) => {
    seek(Number(e.target.value))
    setDragging(false)
    setDragValue(null)
  }

  return (
    <footer className="player">
      <div className="player-inner">
        <button
          type="button"
          className="icon-btn"
          onClick={toggle}
          aria-label={isPlaying ? 'Pausar' : 'Reproducir'}
        >
          {isPlaying ? <PauseIcon /> : <PlayIcon />}
        </button>
        {current.albumCover ? (
          <img className="player-cover" src={current.albumCover} alt="" />
        ) : (
          <span className="player-cover placeholder" aria-hidden="true" />
        )}
        <div className="player-info">
          <span className="player-title">{current.title}</span>
          <span className="player-artist">{current.artistName}</span>
        </div>
        <div className="player-progress">
          <span className="mono-small">{formatDuration(shown)}</span>
          <input
            className="progress"
            type="range"
            min="0"
            max={duration || 0}
            step="0.1"
            value={shown}
            onChange={onScrub}
            onPointerUp={onScrubEnd}
            onKeyUp={onScrubEnd}
            onPointerCancel={() => setDragging(false)}
            style={{ '--fill': `${fillPct}%` }}
            aria-label="Progreso de la canción"
          />
          <span className="mono-small">{formatDuration(duration)}</span>
        </div>
      </div>
    </footer>
  )
}