import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Pause, Play, SkipBack, SkipForward } from 'lucide-react'
import { usePlayer } from '../../state/player'

function VinylRecord({ isPlaying, size = 48 }: { isPlaying: boolean; size?: number }) {
  return (
    <div
      className={`relative flex-shrink-0 ${isPlaying ? 'vinyl-spinning' : ''}`}
      style={{ width: size, height: size }}
    >
      {/* Outer ring */}
      <div
        className="absolute inset-0 rounded-full bg-crate-black"
        style={{
          background: `
            radial-gradient(circle at 50% 50%,
              #1A171F 0%,
              #0D0A14 15%,
              #1A171F 16%,
              #0D0A14 30%,
              #1A171F 31%,
              #0D0A14 45%,
              #1A171F 46%,
              #0D0A14 60%,
              #252130 61%,
              #1A171F 100%
            )
          `,
          boxShadow: 'inset 0 0 10px rgba(0,0,0,0.5), 0 2px 8px rgba(0,0,0,0.3)',
        }}
      />
      {/* Center label */}
      <div
        className="absolute rounded-full bg-crate-amber"
        style={{
          width: size * 0.35,
          height: size * 0.35,
          top: '50%',
          left: '50%',
          transform: 'translate(-50%, -50%)',
          boxShadow: isPlaying ? '0 0 12px rgba(229, 160, 0, 0.4)' : 'none',
        }}
      >
        {/* Center hole */}
        <div
          className="absolute rounded-full bg-crate-black"
          style={{
            width: size * 0.08,
            height: size * 0.08,
            top: '50%',
            left: '50%',
            transform: 'translate(-50%, -50%)',
          }}
        />
      </div>
      {/* Shine effect */}
      <div
        className="absolute inset-0 rounded-full pointer-events-none"
        style={{
          background: 'linear-gradient(135deg, rgba(255,255,255,0.1) 0%, transparent 50%)',
        }}
      />
    </div>
  )
}

export function PlayerBar() {
  const audioRef = useRef<HTMLAudioElement | null>(null)
  const barRef = useRef<HTMLDivElement | null>(null)
  const prevTrackIdRef = useRef<string | undefined>(undefined)
  const lastProgressUpdateRef = useRef<number>(0)
  const { queue, index, next, prev, isPlaying, toggle } = usePlayer()
  const current = queue[index]
  const [progress, setProgress] = useState(0)
  const [duration, setDuration] = useState(0)
  const [dragging, setDragging] = useState(false)

  useEffect(() => {
    if (!audioRef.current) return
    const el = audioRef.current
    const onTime = () => {
      const now = Date.now()
      if (now - lastProgressUpdateRef.current >= 250) {
        setProgress(el.currentTime)
        lastProgressUpdateRef.current = now
      }
    }
    const onLoaded = () => setDuration(isFinite(el.duration) ? el.duration : 0)
    const onEnded = () => {
      next()
    }
    const onError = async () => {
      try {
        await fetch('/api/auth/refresh', { method: 'POST', credentials: 'include' })
      } catch (_) {
      }
      try {
        el.load()
        if (isPlaying) {
          await el.play().catch(() => {})
        }
      } catch (_) {
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

    if (isPlaying) {
      el.play().catch(() => {})
    } else {
      el.pause()
    }
  }, [isPlaying])

  useEffect(() => {
    if (!audioRef.current || !current) return
    const el = audioRef.current

    const trackChanged = prevTrackIdRef.current !== current.id
    if (trackChanged) {
      if (current.durationSeconds && current.durationSeconds > 0) {
        Object.defineProperty(el, 'duration', {
          value: current.durationSeconds,
          writable: true,
          configurable: true
        })
        setDuration(current.durationSeconds)
      }

      el.load()
      setProgress(0)
      prevTrackIdRef.current = current.id

      if (isPlaying) {
        let playbackStarted = false
        const tryPlay = () => {
          if (playbackStarted) return
          if (el.readyState >= 2) {
            el.play().catch(() => {})
            playbackStarted = true
            return true
          }
          return false
        }

        if (tryPlay()) return

        const pollInterval = setInterval(() => {
          if (tryPlay()) {
            clearInterval(pollInterval)
          }
        }, 50)

        const onLoadedData = () => {
          if (tryPlay()) {
            clearInterval(pollInterval)
            el.removeEventListener('loadeddata', onLoadedData)
            el.removeEventListener('canplay', onCanPlay)
            el.removeEventListener('loadedmetadata', onLoadedMetadata)
          }
        }

        const onLoadedMetadata = () => {
          if (!current.durationSeconds && isFinite(el.duration)) {
            setDuration(el.duration)
          }
          if (tryPlay()) {
            clearInterval(pollInterval)
            el.removeEventListener('loadeddata', onLoadedData)
            el.removeEventListener('canplay', onCanPlay)
            el.removeEventListener('loadedmetadata', onLoadedMetadata)
          }
        }

        const onCanPlay = () => {
          if (tryPlay()) {
            clearInterval(pollInterval)
            el.removeEventListener('loadeddata', onLoadedData)
            el.removeEventListener('canplay', onCanPlay)
            el.removeEventListener('loadedmetadata', onLoadedMetadata)
          }
        }

        el.addEventListener('loadeddata', onLoadedData)
        el.addEventListener('loadedmetadata', onLoadedMetadata)
        el.addEventListener('canplay', onCanPlay)

        setTimeout(() => {
          clearInterval(pollInterval)
          el.removeEventListener('loadeddata', onLoadedData)
          el.removeEventListener('canplay', onCanPlay)
          el.removeEventListener('loadedmetadata', onLoadedMetadata)
        }, 10000)
      }
    }
  }, [current, isPlaying])

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

    function handleMove(ev: PointerEvent | TouchEvent) {
      ev.preventDefault()
      if (!audioRef.current || !barRef.current || !duration) return
      const rect = barRef.current.getBoundingClientRect()
      const clientX = 'touches' in ev ? ev.touches[0]?.clientX ?? 0 : ev.clientX
      const x = Math.min(Math.max(clientX - rect.left, 0), rect.width)
      const ratio = rect.width ? x / rect.width : 0
      const nextTime = ratio * duration
      audioRef.current.currentTime = nextTime
      setProgress(nextTime)
    }

    function handleUp(ev: PointerEvent | TouchEvent) {
      ev.preventDefault()
      setDragging(false)
    }

    document.body.style.userSelect = 'none'
    document.body.style.cursor = 'grabbing'

    window.addEventListener('pointermove', handleMove, { passive: false })
    window.addEventListener('pointerup', handleUp, { passive: false })
    window.addEventListener('touchmove', handleMove, { passive: false })
    window.addEventListener('touchend', handleUp, { passive: false })

    return () => {
      window.removeEventListener('pointermove', handleMove)
      window.removeEventListener('pointerup', handleUp)
      window.removeEventListener('touchmove', handleMove)
      window.removeEventListener('touchend', handleUp)
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
    <div className="sticky bottom-0 w-full border-t border-crate-border bg-crate-surface/95 backdrop-blur-md z-30">
      {/* Ambient glow when playing */}
      {isPlaying && (
        <div
          className="absolute inset-x-0 -top-8 h-8 pointer-events-none"
          style={{
            background: 'linear-gradient(to top, rgba(229, 160, 0, 0.08), transparent)',
          }}
        />
      )}

      <div className="mx-auto max-w-6xl px-4 sm:px-6 py-3 sm:py-4">
        {/* Mobile layout */}
        <div className="sm:hidden space-y-3">
          {/* Track info with vinyl */}
          <div className="flex items-center gap-3">
            <VinylRecord isPlaying={isPlaying} size={40} />
            <div className="flex-1 min-w-0">
              <div className="truncate text-sm font-medium text-crate-cream">
                {current?.title || 'No track selected'}
              </div>
              <div className="truncate text-xs text-crate-muted">
                {current?.artist || 'Select a track to play'}
              </div>
            </div>
          </div>

          {/* Progress bar - larger touch target on mobile */}
          <div className="flex items-center gap-2">
            <div className="text-xs tabular-nums text-crate-muted w-10 text-right">
              {formatTime(progress)}
            </div>
            <div
              ref={barRef}
              className="flex-1 cursor-pointer py-3 -my-3 touch-none"
              onClick={onSeek}
              onPointerDown={(e) => {
                e.preventDefault()
                setDragging(true)
              }}
              onTouchStart={(e) => {
                e.preventDefault()
                setDragging(true)
                const touch = e.touches[0]
                if (barRef.current && duration) {
                  const rect = barRef.current.getBoundingClientRect()
                  const x = Math.min(Math.max(touch.clientX - rect.left, 0), rect.width)
                  const ratio = rect.width ? x / rect.width : 0
                  const nextTime = ratio * duration
                  if (audioRef.current) {
                    audioRef.current.currentTime = nextTime
                    setProgress(nextTime)
                  }
                }
              }}
            >
              <div className="vu-meter">
                <div className="vu-meter-fill" style={{ width: `${pct}%` }} />
              </div>
            </div>
            <div className="text-xs tabular-nums text-crate-muted w-10">
              {formatTime(duration)}
            </div>
          </div>

          {/* Transport controls */}
          <div className="flex items-center justify-center gap-4">
            <button className="hw-button" title="Previous" onClick={() => prev()}>
              <SkipBack size={18} />
            </button>
            <button
              className={`hw-button ${isPlaying ? 'hw-button-primary' : ''}`}
              title={isPlaying ? 'Pause' : 'Play'}
              onClick={toggle}
              style={{ padding: '14px' }}
            >
              {isPlaying ? <Pause size={22} /> : <Play size={22} className="ml-0.5" />}
            </button>
            <button className="hw-button" title="Next" onClick={() => next()}>
              <SkipForward size={18} />
            </button>
          </div>
        </div>

        {/* Desktop layout */}
        <div className="hidden sm:flex items-center gap-6">
          {/* Left: Transport controls */}
          <div className="flex items-center gap-2">
            <button className="hw-button" title="Previous" onClick={() => prev()}>
              <SkipBack size={18} />
            </button>
            <button
              className={`hw-button ${isPlaying ? 'hw-button-primary' : ''}`}
              title={isPlaying ? 'Pause' : 'Play'}
              onClick={toggle}
              style={{ padding: '14px' }}
            >
              {isPlaying ? <Pause size={22} /> : <Play size={22} className="ml-0.5" />}
            </button>
            <button className="hw-button" title="Next" onClick={() => next()}>
              <SkipForward size={18} />
            </button>
          </div>

          {/* Center: Vinyl + Track info + Progress */}
          <div className="flex items-center gap-4 flex-1 min-w-0">
            <VinylRecord isPlaying={isPlaying} size={48} />

            <div className="min-w-0 w-48">
              <div className="truncate text-sm font-medium text-crate-cream">
                {current?.title || 'No track selected'}
              </div>
              <div className="truncate text-xs text-crate-muted">
                {current?.artist || 'Select a track to play'}
              </div>
            </div>

            <div className="text-xs tabular-nums text-crate-muted w-12 text-right">
              {formatTime(progress)}
            </div>

            <div
              ref={barRef}
              className="vu-meter flex-1 cursor-pointer"
              onClick={onSeek}
              onPointerDown={(e) => {
                e.preventDefault()
                setDragging(true)
              }}
            >
              <div className="vu-meter-fill" style={{ width: `${pct}%` }} />
            </div>

            <div className="text-xs tabular-nums text-crate-muted w-12">
              {formatTime(duration)}
            </div>
          </div>
        </div>
      </div>

      <audio
        ref={audioRef}
        src={current?.streamUrl}
        preload="metadata"
        crossOrigin="use-credentials"
      />
    </div>
  )
}
