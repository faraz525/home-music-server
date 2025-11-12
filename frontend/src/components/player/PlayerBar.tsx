import { useEffect, useMemo, useRef, useState } from 'react'
import { Pause, Play, SkipBack, SkipForward } from 'lucide-react'
import { usePlayer } from '../../state/player'

export function PlayerBar() {
  const audioRef = useRef<HTMLAudioElement | null>(null)
  const barRef = useRef<HTMLDivElement | null>(null)
  const prevTrackIdRef = useRef<string | undefined>(undefined)
  const { queue, index, next, prev, isPlaying, toggle } = usePlayer()
  const current = queue[index]
  const [progress, setProgress] = useState(0)
  const [duration, setDuration] = useState(0)
  const [dragging, setDragging] = useState(false)

  useEffect(() => {
    if (!audioRef.current) return
    const el = audioRef.current
    const onTime = () => setProgress(el.currentTime)
    const onLoaded = () => setDuration(isFinite(el.duration) ? el.duration : 0)
    const onEnded = () => {
      // Auto-play next track when current track ends
      next()
    }
    const onError = async () => {
      try {
        await fetch('/api/auth/refresh', { method: 'POST', credentials: 'include' })
      } catch (_) {
        // ignore
      }
      // Retry loading the current source
      try {
        el.load()
        if (isPlaying) {
          await el.play().catch(() => {})
        }
      } catch (_) {
        // ignore
      }
    }
    el.addEventListener('timeupdate', onTime)
    el.addEventListener('loadedmetadata', onLoaded)
    el.addEventListener('ended', onEnded)
    el.addEventListener('error', onError)
    return () => {
      el.removeEventListener('timeupdate', onTime)
      el.removeEventListener('loadedmetadata', onLoaded)
      el.removeEventListener('ended', onEnded)
      el.removeEventListener('error', onError)
    }
  }, [next])

  useEffect(() => {
    if (!audioRef.current) return
    const el = audioRef.current
    
    // Handle play/pause state changes - preserve currentTime
    if (isPlaying) {
      // Resume playback from current position
      el.play().catch(() => {})
    } else {
      el.pause()
    }
  }, [isPlaying])

  // Handle track changes when index changes (next/prev navigation)
  useEffect(() => {
    if (!audioRef.current || !current) return
    const el = audioRef.current
    
    // Only reload if the track actually changed (not just play/pause state)
    const trackChanged = prevTrackIdRef.current !== current.id
    if (trackChanged) {
      // Load the new track when current changes
      el.load()
      // Reset progress and duration for new track
      setProgress(0)
      setDuration(0)
      prevTrackIdRef.current = current.id
      
      // Auto-play if we were playing before - wait for metadata to load
      if (isPlaying) {
        const onCanPlay = () => {
          el.play().catch(() => {})
          el.removeEventListener('canplay', onCanPlay)
        }
        el.addEventListener('canplay', onCanPlay)
      }
    }
  }, [current, isPlaying]) // Track both, but only reload on track change

  const pct = useMemo(() => (duration ? (progress / duration) * 100 : 0), [progress, duration])

  function onSeek(e: React.MouseEvent<HTMLDivElement>) {
    if (!audioRef.current || !barRef.current || !duration) return
    const rect = barRef.current.getBoundingClientRect()
    const x = Math.min(Math.max(e.clientX - rect.left, 0), rect.width)
    const ratio = rect.width ? x / rect.width : 0
    const nextTime = ratio * duration
    audioRef.current.currentTime = nextTime
    setProgress(nextTime)
  }

  useEffect(() => {
    if (!dragging) return

    function handleMove(ev: PointerEvent) {
      ev.preventDefault()
      if (!audioRef.current || !barRef.current || !duration) return
      const rect = barRef.current.getBoundingClientRect()
      const x = Math.min(Math.max(ev.clientX - rect.left, 0), rect.width)
      const ratio = rect.width ? x / rect.width : 0
      const nextTime = ratio * duration
      audioRef.current.currentTime = nextTime
      setProgress(nextTime)
    }

    function handleUp(ev: PointerEvent) {
      ev.preventDefault()
      setDragging(false)
    }

    // Prevent text selection during drag
    document.body.style.userSelect = 'none'
    document.body.style.cursor = 'grabbing'

    window.addEventListener('pointermove', handleMove, { passive: false })
    window.addEventListener('pointerup', handleUp, { passive: false })

    return () => {
      window.removeEventListener('pointermove', handleMove)
      window.removeEventListener('pointerup', handleUp)
      document.body.style.userSelect = ''
      document.body.style.cursor = ''
    }
  }, [dragging, duration])

  function formatTime(totalSeconds: number) {
    if (!isFinite(totalSeconds) || totalSeconds <= 0) return '0:00'
    const minutes = Math.floor(totalSeconds / 60)
    const seconds = Math.floor(totalSeconds % 60)
    return `${minutes}:${seconds.toString().padStart(2, '0')}`
  }

  return (
    <div className="sticky bottom-0 w-full border-t border-[#2A2A2A] bg-[#121212] z-30">
      <div className="mx-auto max-w-6xl px-3 sm:px-6 py-2 sm:py-3 flex flex-col sm:flex-row items-center gap-2 sm:gap-4">
        <button className="btn" title="Previous" onClick={() => prev()}>
          <SkipBack />
        </button>
        <button className="btn btn-primary" title={isPlaying ? 'Pause' : 'Play'} onClick={toggle}>
          {isPlaying ? <Pause /> : <Play />}
        </button>
        <button className="btn" title="Next" onClick={() => next()}>
          <SkipForward />
        </button>

        <div className="flex flex-col sm:flex-row items-center gap-2 sm:gap-4 flex-1 min-w-0 w-full sm:w-auto">
          {/* Mobile: Show track info above controls */}
          <div className="sm:hidden w-full text-center mb-1">
            <div className="truncate text-xs">{current?.title || 'Unknown track'}</div>
            <div className="truncate text-xs text-[#A1A1A1]">{current?.artist || 'Unknown artist'}</div>
          </div>
          
          {/* Desktop: Show track info inline */}
          <div className="min-w-0 hidden sm:block">
            <div className="truncate text-sm">{current?.title || 'Unknown track'}</div>
            <div className="truncate text-xs text-[#A1A1A1]">{current?.artist || 'Unknown artist'}</div>
          </div>
          
          <div className="text-xs tabular-nums text-[#A1A1A1] w-10 sm:w-12 text-right">{formatTime(progress)}</div>
          <div
            ref={barRef}
            className="h-1 rounded-full bg-[#2A2A2A] flex-1 cursor-pointer"
            onClick={onSeek}
            onPointerDown={(e) => {
              e.preventDefault()
              setDragging(true)
            }}
          >
            <div className="h-1 rounded-full bg-[#1DB954]" style={{ width: `${pct}%` }} />
          </div>
          <div className="text-xs tabular-nums text-[#A1A1A1] w-10 sm:w-12">{formatTime(duration)}</div>
        </div>

        <audio 
          ref={audioRef} 
          src={current?.streamUrl} 
          preload="metadata"
          crossOrigin="use-credentials"
        />
      </div>
    </div>
  )
}

