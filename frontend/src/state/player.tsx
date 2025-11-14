import React, { createContext, useContext, useMemo, useState } from 'react'
import { api, cratesApi, tracksApi } from '../lib/api'

export type QueueItem = {
  id: string
  title?: string
  artist?: string
  streamUrl: string
  durationSeconds?: number // Duration from database for immediate playback
}

type PlayerContextValue = {
  queue: QueueItem[]
  index: number
  currentCrate: string | null
  play: (item: QueueItem, replace?: boolean) => void
  next: () => void
  prev: () => void
  setCurrentCrate: (crateId: string | null) => void
  isPlaying: boolean
  toggle: () => void
}

const PlayerContext = createContext<PlayerContextValue | undefined>(undefined)

export function PlayerProvider({ children }: { children: React.ReactNode }) {
  const [queue, setQueue] = useState<QueueItem[]>([])
  const [index, setIndex] = useState(0)
  const [isPlaying, setIsPlaying] = useState(false)
  const [currentCrate, setCurrentCrate] = useState<string | null>(null)

  async function fetchCrateTracks(crateId: string) {
    try {
      let response
      if (crateId === 'unsorted') {
        response = await tracksApi.getUnsorted({ limit: 100 })
      } else if (crateId === 'all') {
        response = await api.get('/api/tracks', { params: { limit: 100 } })
      } else {
        response = await cratesApi.getTracks(crateId, { limit: 100 })
      }

      const tracks = response.data?.tracks || []
      return tracks.map((track: any) => ({
        id: track.id,
        title: track.title || track.original_filename,
        artist: track.artist || 'Unknown Artist',
        streamUrl: `/api/tracks/${track.id}/stream`,
        durationSeconds: track.duration_seconds
      }))
    } catch (error) {
      console.error('Failed to fetch playlist tracks:', error)
      return []
    }
  }

  async function populateQueueFromCrate() {
    if (!currentCrate) return false

    const tracks = await fetchCrateTracks(currentCrate)
    if (tracks.length > 0) {
      setQueue(tracks)
      setIndex(0)
      return true
    }
    return false
  }

  function play(item: QueueItem, replace = false) {
    if (replace || queue.length === 0) {
      setQueue([item])
      setIndex(0)
    } else {
      setQueue((q) => [...q, item])
    }
    setIsPlaying(true)
  }

  async function next() {
    const canAdvance = queue.length > 0 && index < queue.length - 1

    if (canAdvance) {
      setIndex((i) => i + 1)
    } else if (queue.length <= 1) {
      // Queue is empty or has only one track, try to populate from crate
      const populated = await populateQueueFromCrate()
      if (populated && queue.length > 1) {
        setIndex(1) // Move to the second track since we're advancing
      }
    }
  }

  async function prev() {
    if (index > 0) {
      setIndex((i) => i - 1)
    } else if (queue.length <= 1) {
      // At the beginning and queue is small, try to populate from crate
      await populateQueueFromCrate()
      // Stay at index 0 since we're going backwards
    }
  }

  function toggle() { setIsPlaying((p) => !p) }

  const value = useMemo(() => ({
    queue,
    index,
    currentCrate,
    play,
    next,
    prev,
    setCurrentCrate,
    isPlaying,
    toggle
  }), [queue, index, currentCrate, isPlaying])

  return <PlayerContext.Provider value={value}>{children}</PlayerContext.Provider>
}

export function usePlayer() {
  const ctx = useContext(PlayerContext)
  if (!ctx) throw new Error('usePlayer must be used within PlayerProvider')
  return ctx
}

