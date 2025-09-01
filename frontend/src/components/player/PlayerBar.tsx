import { useEffect, useMemo, useRef, useState } from 'react'
import { Pause, Play, SkipBack, SkipForward } from 'lucide-react'
import { usePlayer } from '../../state/player'

export function PlayerBar() {
  const audioRef = useRef<HTMLAudioElement | null>(null)
  const { queue, index, next, prev } = usePlayer()
  const current = queue[index]
  const [isPlaying, setIsPlaying] = useState(false)
  const [progress, setProgress] = useState(0)
  const [duration, setDuration] = useState(0)

  useEffect(() => {
    if (!audioRef.current) return
    const el = audioRef.current
    const onTime = () => setProgress(el.currentTime)
    const onLoaded = () => setDuration(el.duration || 0)
    el.addEventListener('timeupdate', onTime)
    el.addEventListener('loadedmetadata', onLoaded)
    return () => {
      el.removeEventListener('timeupdate', onTime)
      el.removeEventListener('loadedmetadata', onLoaded)
    }
  }, [])

  useEffect(() => {
    if (!audioRef.current) return
    if (isPlaying) audioRef.current.play().catch(() => {})
    else audioRef.current.pause()
  }, [isPlaying, current])

  const pct = useMemo(() => (duration ? (progress / duration) * 100 : 0), [progress, duration])

  return (
    <div className="sticky bottom-0 w-full border-t border-[#2A2A2A] bg-[#121212]">
      <div className="mx-auto max-w-6xl px-6 py-3 flex items-center gap-4">
        <button className="btn" title="Previous" onClick={prev}>
          <SkipBack />
        </button>
        <button className="btn btn-primary" title={isPlaying ? 'Pause' : 'Play'} onClick={() => setIsPlaying((p) => !p)}>
          {isPlaying ? <Pause /> : <Play />}
        </button>
        <button className="btn" title="Next" onClick={next}>
          <SkipForward />
        </button>

        <div className="flex-1">
          <div className="h-1 rounded-full bg-[#2A2A2A]">
            <div className="h-1 rounded-full bg-[#1DB954]" style={{ width: `${pct}%` }} />
          </div>
        </div>

        <audio ref={audioRef} src={current?.streamUrl} preload="metadata" />
      </div>
    </div>
  )
}

