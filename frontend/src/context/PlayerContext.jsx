import { createContext, useContext, useEffect, useRef, useState } from 'react'
import {
  spotifyPlayerState,
  spotifyPlayUris,
  spotifyResume,
  spotifyPausePlayback,
  spotifySeekTo,
} from '../api/spotify'
import { useRecentlyPlayed } from '../hooks/useRecentlyPlayed'

const PlayerContext = createContext(null)
const NO_DEVICE_MSG = 'abrí Spotify en tu celu o compu y volvé a intentar'

export function PlayerProvider({ children }) {
  const { tracks: recentlyPlayed, add: addRecent } = useRecentlyPlayed()
  const audioRef = useRef(null)
  const [current, setCurrent] = useState(null)
  const [isPlaying, setIsPlaying] = useState(false)
  const [progress, setProgress] = useState(0)
  const [duration, setDuration] = useState(0)
  // remote=true: suena en tu dispositivo Spotify (el track no trae preview)
  const [remote, setRemote] = useState(false)
  const [playError, setPlayError] = useState(null)

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

  // En modo remoto el progreso lo reporta tu dispositivo: polling cada 5s
  useEffect(() => {
    if (!remote || !isPlaying || !current) return
    const t = setInterval(async () => {
      try {
        const st = await spotifyPlayerState()
        if (!st) {
          setIsPlaying(false)
          return
        }
        setProgress((st.progress_ms || 0) / 1000)
        if (st.item?.duration_ms) setDuration(st.item.duration_ms / 1000)
        setIsPlaying(!!st.is_playing)
      } catch {
        /* mantiene el último estado conocido */
      }
    }, 5000)
    return () => clearInterval(t)
  }, [remote, isPlaying, current])

  const playRemote = async (track) => {
    const uri = track.spotifyUri || `spotify:track:${track.id}`
    const res = await spotifyPlayUris([uri])
    if (!res.ok) {
      const reason = res.body?.error?.reason
      setPlayError(reason === 'NO_ACTIVE_DEVICE' ? NO_DEVICE_MSG : `no se pudo reproducir (${res.status})`)
      setIsPlaying(false)
      return
    }
    audioRef.current.pause()
    setCurrent(track)
    setProgress(0)
    setDuration(track.duration || 0)
    setRemote(true)
    setIsPlaying(true)
    setPlayError(null)
    addRecent(track)
  }

  const play = (track) => {
    const isSame = current && current.id === track.id
    if (isSame) {
      toggle()
      return
    }
    setPlayError(null)
    if (track.previewUrl) {
      const audio = audioRef.current
      setRemote(false)
      setCurrent(track)
      setProgress(0)
      setDuration(0)
      audio.src = track.previewUrl
      audio.play().catch(() => setIsPlaying(false))
      setIsPlaying(true)
      addRecent(track)
      return
    }
    void playRemote(track)
  }

  const toggle = () => {
    if (!current) return
    if (remote) {
      if (isPlaying) {
        setIsPlaying(false)
        spotifyPausePlayback().catch(() => setIsPlaying(true))
      } else {
        setIsPlaying(true)
        spotifyResume().catch(() => setIsPlaying(false))
      }
      return
    }
    const audio = audioRef.current
    if (!audio) return
    if (audio.paused) {
      audio.play().catch(() => setIsPlaying(false))
      setIsPlaying(true)
    } else {
      audio.pause()
      setIsPlaying(false)
    }
  }

  const seek = (value) => {
    if (remote && current) {
      setProgress(value)
      spotifySeekTo(value * 1000).catch(() => {})
      return
    }
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
    <PlayerContext.Provider
      value={{ current, isPlaying, progress, duration, play, toggle, seek, recentlyPlayed, remote, playError }}
    >
      {children}
    </PlayerContext.Provider>
  )
}

export const usePlayer = () => useContext(PlayerContext)
