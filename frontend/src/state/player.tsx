import React, { createContext, useContext, useMemo, useState } from 'react'

export type QueueItem = {
  id: string
  title?: string
  artist?: string
  streamUrl: string
}

type PlayerContextValue = {
  queue: QueueItem[]
  index: number
  play: (item: QueueItem, replace?: boolean) => void
  next: () => void
  prev: () => void
}

const PlayerContext = createContext<PlayerContextValue | undefined>(undefined)

export function PlayerProvider({ children }: { children: React.ReactNode }) {
  const [queue, setQueue] = useState<QueueItem[]>([])
  const [index, setIndex] = useState(0)

  function play(item: QueueItem, replace = false) {
    if (replace || queue.length === 0) {
      setQueue([item])
      setIndex(0)
    } else {
      setQueue((q) => [...q, item])
    }
  }
  function next() { setIndex((i) => Math.min(queue.length - 1, i + 1)) }
  function prev() { setIndex((i) => Math.max(0, i - 1)) }

  const value = useMemo(() => ({ queue, index, play, next, prev }), [queue, index])
  return <PlayerContext.Provider value={value}>{children}</PlayerContext.Provider>
}

export function usePlayer() {
  const ctx = useContext(PlayerContext)
  if (!ctx) throw new Error('usePlayer must be used within PlayerProvider')
  return ctx
}

