import React, { createContext, useContext, useMemo, useState } from 'react'
import { api, playlistsApi, tracksApi } from '../lib/api'

export type QueueItem = {
  id: string
  title?: string
  artist?: string
  streamUrl: string
}

type PlayerContextValue = {
  queue: QueueItem[]
  index: number
  currentPlaylist: string | null
  play: (item: QueueItem, replace?: boolean) => void
  next: () => void
  prev: () => void
  setCurrentPlaylist: (playlistId: string | null) => void
  isPlaying: boolean
  toggle: () => void
}

const PlayerContext = createContext<PlayerContextValue | undefined>(undefined)

export function PlayerProvider({ children }: { children: React.ReactNode }) {
  const [queue, setQueue] = useState<QueueItem[]>([])
  const [index, setIndex] = useState(0)
  const [isPlaying, setIsPlaying] = useState(false)
  const [currentPlaylist, setCurrentPlaylist] = useState<string | null>(null)

  async function fetchPlaylistTracks(playlistId: string) {
    try {
      let response
      if (playlistId === 'unsorted') {
        response = await tracksApi.getUnsorted({ limit: 100 })
      } else if (playlistId === 'all') {
        response = await api.get('/api/tracks', { params: { limit: 100 } })
      } else {
        response = await playlistsApi.getTracks(playlistId, { limit: 100 })
      }

      const tracks = response.data?.tracks || []
      return tracks.map((track: any) => ({
        id: track.id,
        title: track.title || track.original_filename,
        artist: track.artist || 'Unknown Artist',
        streamUrl: `/api/tracks/${track.id}/stream`
      }))
    } catch (error) {
      console.error('Failed to fetch playlist tracks:', error)
      return []
    }
  }

  async function populateQueueFromPlaylist() {
    if (!currentPlaylist) return false

    const tracks = await fetchPlaylistTracks(currentPlaylist)
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
      // Queue is empty or has only one track, try to populate from playlist
      const populated = await populateQueueFromPlaylist()
      if (populated && queue.length > 1) {
        setIndex(1) // Move to the second track since we're advancing
      }
    }
  }

  async function prev() {
    if (index > 0) {
      setIndex((i) => i - 1)
    } else if (queue.length <= 1) {
      // At the beginning and queue is small, try to populate from playlist
      await populateQueueFromPlaylist()
      // Stay at index 0 since we're going backwards
    }
  }

  function toggle() { setIsPlaying((p) => !p) }

  const value = useMemo(() => ({
    queue,
    index,
    currentPlaylist,
    play,
    next,
    prev,
    setCurrentPlaylist,
    isPlaying,
    toggle
  }), [queue, index, currentPlaylist, isPlaying])

  return <PlayerContext.Provider value={value}>{children}</PlayerContext.Provider>
}

export function usePlayer() {
  const ctx = useContext(PlayerContext)
  if (!ctx) throw new Error('usePlayer must be used within PlayerProvider')
  return ctx
}

