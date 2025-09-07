import { api, playlistsApi, tracksApi } from '../lib/api'
import { ChangeEvent, FormEvent, useEffect, useMemo, useState } from 'react'
import { usePlayer } from '../state/player'
import { ChevronDown, Plus, Music, MoreHorizontal, ListPlus } from 'lucide-react'
import { Playlist, PlaylistList, Track, TrackList } from '../types/playlists'

export function LibraryPage() {
  const { play } = usePlayer()
  const [q, setQ] = useState('')
  const [selectedPlaylist, setSelectedPlaylist] = useState<string>('all')
  const [tracks, setTracks] = useState<TrackList>({ tracks: [], total: 0, limit: 20, offset: 0, has_next: false })
  const [playlists, setPlaylists] = useState<PlaylistList>({ playlists: [], total: 0, limit: 20, offset: 0, has_next: false })
  const [loadingPlaylists, setLoadingPlaylists] = useState(true)
  const [file, setFile] = useState<File | null>(null)
  const [uploading, setUploading] = useState(false)
  const [progress, setProgress] = useState(0)
  const [showPlaylistDropdown, setShowPlaylistDropdown] = useState(false)
  const [trackMenuOpen, setTrackMenuOpen] = useState<string | null>(null)

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
        const response = await tracksApi.getUnsorted({ q: q || undefined })
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

  // Get the selected playlist name for display
  const selectedPlaylistName = selectedPlaylist === 'all' ? 'All Tracks' :
    selectedPlaylist === 'unsorted' ? 'Unsorted' :
    playlists.playlists?.find(p => p.id === selectedPlaylist)?.name || 'All Tracks'

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row gap-3">
        {/* Playlist Selector */}
        <div className="relative">
          <button
            onClick={() => setShowPlaylistDropdown(!showPlaylistDropdown)}
            className="btn flex items-center gap-2 min-w-[200px]"
          >
            <Music size={16} />
            {selectedPlaylistName}
            <ChevronDown size={14} className="ml-auto" />
          </button>

          {showPlaylistDropdown && (
            <div className="playlist-dropdown absolute top-full mt-1 w-full bg-[#1A1A1A] rounded-lg shadow-lg border border-[#2A2A2A] py-1 z-10">
              <button
                onClick={() => {
                  setSelectedPlaylist('all')
                  setShowPlaylistDropdown(false)
                }}
                className="w-full text-left px-3 py-2 text-sm hover:bg-[#2A2A2A] flex items-center gap-2"
              >
                <Music size={14} />
                All Tracks
              </button>

              <button
                onClick={() => {
                  setSelectedPlaylist('unsorted')
                  setShowPlaylistDropdown(false)
                }}
                className="w-full text-left px-3 py-2 text-sm hover:bg-[#2A2A2A] flex items-center gap-2"
              >
                <Music size={14} />
                Unsorted
              </button>

              <div className="border-t border-[#2A2A2A] my-1"></div>

              {loadingPlaylists ? (
                <div className="px-3 py-2 text-sm text-[#A1A1A1]">Loading playlists...</div>
              ) : (
                playlists.playlists?.map((playlist) => (
                <button
                  key={playlist.id}
                  onClick={() => {
                    setSelectedPlaylist(playlist.id)
                    setShowPlaylistDropdown(false)
                  }}
                  className="w-full text-left px-3 py-2 text-sm hover:bg-[#2A2A2A] flex items-center gap-2"
                >
                  <Music size={14} />
                  {playlist.name}
                  {playlist.is_default && <span className="text-xs text-[#A1A1A1]">(Default)</span>}
                </button>
              ))
              )}
            </div>
          )}
        </div>

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

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {Array.isArray(tracks.tracks) && tracks.tracks.map((t) => (
          <div key={t.id} className="card p-4 flex flex-col gap-2 group">
            <div className="text-base font-semibold truncate">{t.title || t.original_filename}</div>
            <div className="text-sm text-[#A1A1A1] truncate">{t.artist || 'Unknown Artist'}</div>

            <div className="mt-2 flex gap-2 items-center">
              <button
                className="btn btn-primary flex-1"
                onClick={() => play({ id: t.id, title: t.title, artist: t.artist, streamUrl: `/api/tracks/${t.id}/stream` }, true)}
              >
                Play
              </button>

              {/* Track Menu */}
              <div className="relative">
                <button
                  className="btn p-2"
                  onClick={() => setTrackMenuOpen(trackMenuOpen === t.id ? null : t.id)}
                >
                  <MoreHorizontal size={16} />
                </button>

                {trackMenuOpen === t.id && (
                  <div className="track-menu absolute right-0 top-full mt-1 w-48 bg-[#1A1A1A] rounded-lg shadow-lg border border-[#2A2A2A] py-1 z-20">
                    {/* Add to Playlist Options */}
                    <div className="px-3 py-2 text-xs font-semibold text-[#A1A1A1] border-b border-[#2A2A2A]">
                      Add to Playlist
                    </div>

                    {loadingPlaylists ? (
                      <div className="px-3 py-2 text-sm text-[#A1A1A1]">Loading playlists...</div>
                    ) : (
                      playlists.playlists && playlists.playlists
                        .filter(p => !p.is_default) // Don't show default playlist
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

                    {/* Remove from Playlist (if viewing a specific playlist) */}
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
          </div>
        ))}
      </div>
    </div>
  )
}

