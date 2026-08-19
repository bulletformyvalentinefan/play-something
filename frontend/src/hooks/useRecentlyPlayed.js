import { useCallback, useState } from 'react'

const KEY = 'recently_played'
const MAX = 10

export function useRecentlyPlayed() {
  const [tracks, setTracks] = useState(() => {
    try {
      return JSON.parse(localStorage.getItem(KEY)) || []
    } catch {
      return []
    }
  })

  const add = useCallback((track) => {
    if (!track) return
    setTracks((prev) => {
      const next = [track, ...prev.filter((t) => t.id !== track.id)].slice(0, MAX)
      localStorage.setItem(KEY, JSON.stringify(next))
      return next
    })
  }, [])

  return { tracks, add }
}