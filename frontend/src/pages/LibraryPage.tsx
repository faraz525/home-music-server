import { api } from '../lib/api'
import { ChangeEvent, FormEvent, useEffect, useMemo, useState } from 'react'
import { usePlayer } from '../state/player'

type Track = {
  id: string
  title?: string
  artist?: string
  album?: string
  original_filename: string
}

type TrackList = {
  tracks: Track[]
  total: number
  limit: number
  offset: number
  has_next: boolean
}

export function LibraryPage() {
  const { play } = usePlayer()
  const [q, setQ] = useState('')
  const [tracks, setTracks] = useState<TrackList>({ tracks: [], total: 0, limit: 20, offset: 0, has_next: false })
  const [file, setFile] = useState<File | null>(null)
  const [uploading, setUploading] = useState(false)
  const [progress, setProgress] = useState(0)

  const fetchTracks = useMemo(() => async () => {
    try {
      const { data } = await api.get('/api/tracks', { params: { q } })
      // Ensure data has the expected structure, fallback to empty tracks if not
      if (data && data.tracks && Array.isArray(data.tracks)) {
        setTracks(data)
      } else {
        // If the response doesn't have the expected structure, set empty tracks
        setTracks({ tracks: [], total: 0, limit: 20, offset: 0, has_next: false })
      }
    } catch (error) {
      console.error('Failed to fetch tracks:', error)
      // On error, set empty tracks to prevent the map error
      setTracks({ tracks: [], total: 0, limit: 20, offset: 0, has_next: false })
    }
  }, [q])

  useEffect(() => {
    fetchTracks()
  }, [fetchTracks])

  async function onUpload(e: FormEvent) {
    e.preventDefault()
    if (!file) return
    const form = new FormData()
    form.append('file', file)
    setUploading(true)
    setProgress(0)
    try {
      await api.post('/api/tracks', form, {
        headers: { 'Content-Type': 'multipart/form-data' },
        onUploadProgress: (ev) => {
          if (ev.total) setProgress(Math.round((ev.loaded * 100) / ev.total))
        },
      })
      setFile(null)
      await fetchTracks()
    } finally {
      setUploading(false)
    }
  }

  async function onDelete(id: string) {
    await api.delete(`/api/tracks/${id}`)
    await fetchTracks()
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <input
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="Search your library"
          className="input flex-1"
        />
        <button className="btn btn-primary" onClick={fetchTracks}>Search</button>
      </div>

      <div className="card p-4">
        <form className="flex items-center gap-3" onSubmit={onUpload}>
          <input
            type="file"
            className="input file:mr-4 file:rounded-full file:border-0 file:bg-[#1DB954] file:text-black file:px-3 file:py-1"
            onChange={(e: ChangeEvent<HTMLInputElement>) => setFile(e.target.files?.[0] || null)}
            accept="audio/*"
          />
          <button className="btn btn-primary" disabled={!file || uploading}>
            {uploading ? `Uploading ${progress}%` : 'Upload'}
          </button>
        </form>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {Array.isArray(tracks.tracks) && tracks.tracks.map((t) => (
          <div key={t.id} className="card p-4 flex flex-col gap-2">
            <div className="text-base font-semibold truncate">{t.title || t.original_filename}</div>
            <div className="text-sm text-[#A1A1A1] truncate">{t.artist || 'Unknown Artist'}</div>
            <div className="mt-2 flex gap-2">
              <button
                className="btn btn-primary"
                onClick={() => play({ id: t.id, title: t.title, artist: t.artist, streamUrl: `/api/tracks/${t.id}/stream` }, true)}
              >
                Play
              </button>
              <button className="btn" onClick={() => onDelete(t.id)}>Delete</button>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

