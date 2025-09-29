import { api, playlistsApi, tracksApi } from '../lib/api'
import type { UnsortedParams } from '../lib/api'
import { ChangeEvent, FormEvent, useEffect, useMemo, useState } from 'react'
import { usePlayer } from '../state/player'
import { useSearchParams } from 'react-router-dom'
import { MoreHorizontal, ListPlus, Play, Pause } from 'lucide-react'
import { PlaylistList, TrackList } from '../types/playlists'

export function LibraryPage() {
  const { play, isPlaying, toggle, queue, index, setCurrentPlaylist } = usePlayer()
  const current = queue[index]
  const [q, setQ] = useState('')
  const [searchParams, setSearchParams] = useSearchParams()
  const [selectedPlaylist, setSelectedPlaylist] = useState<string>(() => searchParams.get('playlist') || 'all')
  const [tracks, setTracks] = useState<TrackList>({ tracks: [], total: 0, limit: 20, offset: 0, has_next: false })
  const [playlists, setPlaylists] = useState<PlaylistList>({ playlists: [], total: 0, limit: 20, offset: 0, has_next: false })
  const [loadingPlaylists, setLoadingPlaylists] = useState(true)
  const [file, setFile] = useState<File | null>(null)
  const [uploading, setUploading] = useState(false)
  const [progress, setProgress] = useState(0)
  const [showPlaylistDropdown, setShowPlaylistDropdown] = useState(false)
  const [trackMenuOpen, setTrackMenuOpen] = useState<string | null>(null)
  const [selectedTrackIds, setSelectedTrackIds] = useState<Set<string>>(new Set())

  // Fetch playlists
  const fetchPlaylists = async () => {
    try {
      setLoadingPlaylists(true)
      const { data } = await playlistsApi.list()
      // Ensure we always have a valid structure
      setPlaylists(data && data.playlists ? data : { playlists: [], total: 0, limit: 20, offset: 0, has_next: false })
    } catch (error) {
      console.error('Failed to fetch playlists:', error)
      // Set empty playlists on error
      setPlaylists({ playlists: [], total: 0, limit: 20, offset: 0, has_next: false })
    } finally {
      setLoadingPlaylists(false)
    }
  }

  // Fetch tracks based on selected playlist
  const fetchTracks = useMemo(() => async () => {
    try {
      let data
      if (selectedPlaylist === 'unsorted') {
        const params: UnsortedParams = { q: q || undefined }
        const response = await tracksApi.getUnsorted(params)
        data = response.data
      } else if (selectedPlaylist && selectedPlaylist !== 'all') {
        const response = await api.get('/api/tracks', {
          params: { q: q || undefined, playlist_id: selectedPlaylist }
        })
        data = response.data
      } else {
        const response = await api.get('/api/tracks', { params: { q: q || undefined } })
        data = response.data
      }

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
  }, [q, selectedPlaylist])

  useEffect(() => {
    fetchPlaylists()
  }, [])

  // Keep selected playlist in sync with URL (?playlist=<id>|unsorted or none => all)
  useEffect(() => {
    const p = searchParams.get('playlist')
    const next = p || 'all'
    if (next !== selectedPlaylist) {
      setSelectedPlaylist(next)
    }
  }, [searchParams])

  // Update player context when playlist changes
  useEffect(() => {
    setCurrentPlaylist(selectedPlaylist)
  }, [selectedPlaylist, setCurrentPlaylist])

  // Only push to URL if the change originated from in-page actions (not from URL itself)
  const [lastUrlPlaylist, setLastUrlPlaylist] = useState<string | null>(searchParams.get('playlist'))
  useEffect(() => {
    const curr = searchParams.get('playlist') || 'all'
    // If selectedPlaylist differs from the URL and the URL hasn't just changed, update URL
    if (selectedPlaylist !== curr && lastUrlPlaylist === curr) {
      const next = new URLSearchParams(searchParams)
      if (selectedPlaylist && selectedPlaylist !== 'all') next.set('playlist', selectedPlaylist)
      else next.delete('playlist')
      setSearchParams(next, { replace: true })
    }
    setLastUrlPlaylist(searchParams.get('playlist'))
  }, [selectedPlaylist, searchParams])

  useEffect(() => {
    fetchTracks()
  }, [fetchTracks])

  // Close dropdowns when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (showPlaylistDropdown && !(event.target as Element).closest('.playlist-dropdown')) {
        setShowPlaylistDropdown(false)
      }
      if (trackMenuOpen && !(event.target as Element).closest('.track-menu')) {
        setTrackMenuOpen(null)
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [showPlaylistDropdown, trackMenuOpen])

  async function onUpload(e: FormEvent) {
    e.preventDefault()
    if (!file) return
    const form = new FormData()
    form.append('file', file)

    // Add playlist assignment if a playlist is selected (not 'all' or 'unsorted')
    if (selectedPlaylist && selectedPlaylist !== 'all' && selectedPlaylist !== 'unsorted') {
      form.append('playlist_id', selectedPlaylist)
    }

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
      await fetchPlaylists() // Refresh playlists in case new tracks were added
    } finally {
      setUploading(false)
    }
  }

  async function onDelete(id: string) {
    await api.delete(`/api/tracks/${id}`)
    await fetchTracks()
  }

  async function addTrackToPlaylist(trackId: string, playlistId: string) {
    try {
      await playlistsApi.addTracks(playlistId, [trackId])
      await fetchTracks()
      await fetchPlaylists()
    } catch (error) {
      console.error('Failed to add track to playlist:', error)
    }
  }

  async function removeTrackFromPlaylist(trackId: string, playlistId: string) {
    try {
      await playlistsApi.removeTracks(playlistId, [trackId])
      await fetchTracks()
      await fetchPlaylists()
    } catch (error) {
      console.error('Failed to remove track from playlist:', error)
    }
  }

  // Bulk actions
  async function bulkAddToPlaylist(playlistId: string) {
    try {
      const ids = Array.from(selectedTrackIds)
      if (ids.length === 0) return
      await playlistsApi.addTracks(playlistId, ids)
      setSelectedTrackIds(new Set())
      await fetchTracks()
      await fetchPlaylists()
    } catch (error) {
      console.error('Failed bulk add:', error)
    }
  }

  async function bulkRemoveFromCurrentPlaylist() {
    try {
      if (!selectedPlaylist || selectedPlaylist === 'all' || selectedPlaylist === 'unsorted') return
      const ids = Array.from(selectedTrackIds)
      if (ids.length === 0) return
      await playlistsApi.removeTracks(selectedPlaylist, ids)
      setSelectedTrackIds(new Set())
      await fetchTracks()
      await fetchPlaylists()
    } catch (error) {
      console.error('Failed bulk remove:', error)
    }
  }

  function toggleSelected(id: string) {
    setSelectedTrackIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  function clearSelection() { setSelectedTrackIds(new Set()) }

  function formatDuration(totalSeconds?: number) {
    if (!totalSeconds || totalSeconds <= 0) return '—'
    const minutes = Math.floor(totalSeconds / 60)
    const seconds = Math.floor(totalSeconds % 60)
    return `${minutes}:${seconds.toString().padStart(2, '0')}`
  }

  // selected playlist is controlled by the URL and sidebar links

  return (
    <div className="space-y-6">
      {/* Bulk selection toolbar */}
      {selectedTrackIds.size > 0 && (
        <div className="card p-3 flex flex-wrap items-center gap-3">
          <div className="text-sm">{selectedTrackIds.size} selected</div>
          <div className="relative">
            <button className="btn btn-primary" onClick={() => setShowPlaylistDropdown(!showPlaylistDropdown)}>
              Add to playlist
            </button>
            {showPlaylistDropdown && (
              <div className="playlist-dropdown absolute top-full mt-1 w-56 bg-[#1A1A1A] rounded-lg shadow-lg border border-[#2A2A2A] py-1 z-10">
                {loadingPlaylists ? (
                  <div className="px-3 py-2 text-sm text-[#A1A1A1]">Loading playlists...</div>
                ) : (
                  playlists.playlists?.filter(p => !p.is_default).map((p) => (
                    <button
                      key={p.id}
                      className="w-full text-left px-3 py-2 text-sm hover:bg-[#2A2A2A]"
                      onClick={() => { bulkAddToPlaylist(p.id); setShowPlaylistDropdown(false) }}
                    >
                      {p.name}
                    </button>
                  ))
                )}
              </div>
            )}
          </div>
          {selectedPlaylist && selectedPlaylist !== 'all' && selectedPlaylist !== 'unsorted' && (
            <button className="btn" onClick={bulkRemoveFromCurrentPlaylist}>Remove from this playlist</button>
          )}
          <button className="btn" onClick={clearSelection}>Clear selection</button>
        </div>
      )}

      <div className="flex flex-col sm:flex-row gap-3">
        {/* Search Input */}
        <div className="flex items-center gap-3 flex-1">
          <input
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder="Search your library"
            className="input flex-1"
          />
          <button className="btn btn-primary" onClick={fetchTracks}>Search</button>
        </div>
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

      <div className="card p-0 overflow-hidden">
        {/* Header row */}
        <div className="px-4 py-2 text-xs uppercase tracking-wide text-[#A1A1A1] grid grid-cols-[24px_1fr_1fr_120px_60px_32px] items-center gap-3 border-b border-[#2A2A2A]">
          <input
            type="checkbox"
            checked={tracks.tracks.length > 0 && selectedTrackIds.size === tracks.tracks.length}
            onChange={(e) => {
              if (e.target.checked) setSelectedTrackIds(new Set(tracks.tracks.map(t => t.id)))
              else clearSelection()
            }}
          />
          <div className="col-span-1">Title</div>
          <div>Album</div>
          <div className="text-right pr-4">Date added</div>
          <div className="text-right">Duration</div>
          <div></div>
        </div>

        {/* Rows */}
        {Array.isArray(tracks.tracks) && tracks.tracks.map((t, idx) => {
          const isCurrent = current?.id === t.id
          const isCurrentAndPlaying = isCurrent && isPlaying
          return (
            <div key={t.id} className="px-4 py-2 grid grid-cols-[24px_1fr_1fr_120px_60px_32px] items-center gap-3 hover:bg-[#1A1A1A]">
              <input
                type="checkbox"
                checked={selectedTrackIds.has(t.id)}
                onChange={() => toggleSelected(t.id)}
              />
              <div className="min-w-0">
                <div className="flex items-center gap-3 min-w-0">
                  <button
                    className="btn btn-primary p-2 flex items-center justify-center"
                    title={isCurrentAndPlaying ? 'Pause' : 'Play'}
                    onClick={() => {
                      if (isCurrent) toggle()
                      else {
                        const fallbackFromFilename = (() => {
                          const name = t.original_filename || ''
                          const noExt = name.includes('.') ? name.substring(0, name.lastIndexOf('.')) : name
                          return noExt.slice(0, 60)
                        })()
                        const displayTitle = t.title || fallbackFromFilename || 'Unknown track'
                        const displayArtist = t.artist || 'Unknown artist'
                        play({ id: t.id, title: displayTitle, artist: displayArtist, streamUrl: `/api/tracks/${t.id}/stream` }, true)
                      }
                    }}
                  >
                    {isCurrentAndPlaying ? <Pause /> : <Play />}
                  </button>
                  <div className="min-w-0">
                    <div className="truncate">{t.title || t.original_filename}</div>
                    <div className="text-sm text-[#A1A1A1] truncate">{t.artist || 'Unknown Artist'}</div>
                  </div>
                </div>
              </div>
              <div className="truncate">{t.album || '—'}</div>
              <div className="text-right pr-4 text-sm text-[#A1A1A1]">{t.created_at ? new Date(t.created_at).toLocaleDateString() : '—'}</div>
              <div className="text-right text-sm text-[#A1A1A1]">{formatDuration(t.duration_seconds)}</div>
              <div className="relative justify-self-end">
                <button
                  className="btn p-2"
                  onClick={() => setTrackMenuOpen(trackMenuOpen === t.id ? null : t.id)}
                >
                  <MoreHorizontal size={16} />
                </button>
                {trackMenuOpen === t.id && (
                  <div className="track-menu absolute right-0 top-full mt-1 w-48 bg-[#1A1A1A] rounded-lg shadow-lg border border-[#2A2A2A] py-1 z-20">
                    <div className="px-3 py-2 text-xs font-semibold text-[#A1A1A1] border-b border-[#2A2A2A]">Add to Playlist</div>
                    {loadingPlaylists ? (
                      <div className="px-3 py-2 text-sm text-[#A1A1A1]">Loading playlists...</div>
                    ) : (
                      playlists.playlists && playlists.playlists
                        .filter(p => !p.is_default)
                        .map((playlist) => (
                          <button
                            key={playlist.id}
                            onClick={() => {
                              addTrackToPlaylist(t.id, playlist.id)
                              setTrackMenuOpen(null)
                            }}
                            className="w-full text-left px-3 py-2 text-sm hover:bg-[#2A2A2A] flex items-center gap-2"
                          >
                            <ListPlus size={14} />
                            {playlist.name}
                          </button>
                        ))
                    )}
                    {selectedPlaylist && selectedPlaylist !== 'all' && selectedPlaylist !== 'unsorted' && (
                      <>
                        <div className="border-t border-[#2A2A2A] my-1"></div>
                        <button
                          onClick={() => {
                            removeTrackFromPlaylist(t.id, selectedPlaylist)
                            setTrackMenuOpen(null)
                          }}
                          className="w-full text-left px-3 py-2 text-sm hover:bg-[#2A2A2A] text-red-400 flex items-center gap-2"
                        >
                          <ListPlus size={14} className="rotate-45" />
                          Remove from Playlist
                        </button>
                      </>
                    )}
                    <div className="border-t border-[#2A2A2A] my-1"></div>
                    <button
                      onClick={() => {
                        onDelete(t.id)
                        setTrackMenuOpen(null)
                      }}
                      className="w-full text-left px-3 py-2 text-sm hover:bg-[#2A2A2A] text-red-400 flex items-center gap-2"
                    >
                      <MoreHorizontal size={14} />
                      Delete Track
                    </button>
                  </div>
                )}
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}

