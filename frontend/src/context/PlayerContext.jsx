import { createContext, useContext, useEffect, useRef, useState } from 'react'
import { playTrack } from '../api/tracks'
import { useAuth } from './AuthContext'
import { useRecentlyPlayed } from '../hooks/useRecentlyPlayed'

const PlayerContext = createContext(null)

export function PlayerProvider({ children }) {
  const { user } = useAuth()
  const { tracks: recentlyPlayed, add: addRecent } = useRecentlyPlayed()
  const audioRef = useRef(null)
  const [current, setCurrent] = useState(null)
  const [isPlaying, setIsPlaying] = useState(false)
  const [progress, setProgress] = useState(0)
  const [duration, setDuration] = useState(0)

  useEffect(() => {
    const audio = new Audio()
    audioRef.current = audio

    const onTime = () => setProgress(audio.currentTime)
    const onMeta = () => setDuration(audio.duration || 0)
    const onEnd = () => {
      setIsPlaying(false)
      setProgress(0)
    }

    audio.addEventListener('timeupdate', onTime)
    audio.addEventListener('loadedmetadata', onMeta)
    audio.addEventListener('ended', onEnd)

    return () => {
      audio.pause()
      audio.src = ''
      audio.removeEventListener('timeupdate', onTime)
      audio.removeEventListener('loadedmetadata', onMeta)
      audio.removeEventListener('ended', onEnd)
    }
  }, [])

  const play = async (track) => {
    const audio = audioRef.current
    const isSame = current && current.id === track.id

    if (isSame) {
      if (isPlaying) {
        audio.pause()
        setIsPlaying(false)
        return
      }
      audio.play().catch(() => setIsPlaying(false))
      setIsPlaying(true)
      return
    }

    setCurrent(track)
    setProgress(0)
    setDuration(0)
    audio.src = track.previewUrl
    audio.play().catch(() => setIsPlaying(false))
    setIsPlaying(true)
    addRecent(track)

    if (user) {
      try {
        await playTrack(track.id, user.id)
      } catch {
        /* el evento de reproducción no bloquea el audio */
      }
    }
  }

  const toggle = () => {
    const audio = audioRef.current
    if (!audio || !current) return
    if (audio.paused) {
      audio.play().catch(() => setIsPlaying(false))
      setIsPlaying(true)
    } else {
      audio.pause()
      setIsPlaying(false)
    }
  }

  const seek = (value) => {
    const audio = audioRef.current
    if (!audio) return
    audio.currentTime = value
    setProgress(value)
  }

  const toggleRef = useRef(toggle)
  useEffect(() => {
    toggleRef.current = toggle
  })

  useEffect(() => {
    const onKeyDown = (e) => {
      if (e.code !== 'Space') return
      const tag = e.target.tagName
      if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || tag === 'BUTTON' || e.target.isContentEditable) {
        return
      }
      e.preventDefault()
      toggleRef.current()
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [])

  return (
    <PlayerContext.Provider value={{ current, isPlaying, progress, duration, play, toggle, seek, recentlyPlayed }}>
      {children}
    </PlayerContext.Provider>
  )
}

export const usePlayer = () => useContext(PlayerContext)