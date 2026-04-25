import { useState } from 'react'

type TrackCoverProps = {
  trackId: string
  hasCover: boolean
  size?: number
  className?: string
  fallback: React.ReactNode
  alt?: string
}

export function TrackCover({ trackId, hasCover, size = 40, className = '', fallback, alt }: TrackCoverProps) {
  const [errored, setErrored] = useState(false)

  if (!hasCover || errored) {
    return <>{fallback}</>
  }

  return (
    <img
      src={`/api/tracks/${trackId}/cover`}
      alt={alt || 'Cover art'}
      loading="lazy"
      width={size}
      height={size}
      className={`flex-shrink-0 rounded-md object-cover ${className}`}
      style={{ width: size, height: size }}
      onError={() => setErrored(true)}
    />
  )
}
