import { api, cratesApi, normalizeCrateList, tracksApi } from '../lib/api'
import type { UnsortedParams } from '../lib/api'
import { ChangeEvent, FormEvent, useEffect, useMemo, useState } from 'react'
import { usePlayer } from '../state/player'
import { useSearchParams } from 'react-router-dom'
import { MoreHorizontal, ListPlus, Play, Pause } from 'lucide-react'
import type { CrateList, TrackList } from '../types/crates'

export function LibraryPage() {
  const { play, isPlaying, toggle, queue, index, setCurrentCrate } = usePlayer()
  const current = queue[index]
  const [q, setQ] = useState('')
  const [searchParams, setSearchParams] = useSearchParams()
  const [selectedCrate, setSelectedCrate] = useState<string>(() => searchParams.get('crate') || 'all')
  const [tracks, setTracks] = useState<TrackList>({ tracks: [], total: 0, limit: 20, offset: 0, has_next: false })
  const [crates, setCrates] = useState<CrateList>({ crates: [], total: 0, limit: 20, offset: 0, has_next: false })
  const [loadingCrates, setLoadingCrates] = useState(true)
  const [showCrateDropdown, setShowCrateDropdown] = useState(false)
  const [trackMenuOpen, setTrackMenuOpen] = useState<string | null>(null)
  const [selectedTrackIds, setSelectedTrackIds] = useState<Set<string>>(new Set())

  // Fetch crates
  const fetchCrates = async () => {
    try {
      setLoadingCrates(true)
      const { data } = await cratesApi.list()
      setCrates(normalizeCrateList(data))
    } catch (error) {
      console.error('Failed to fetch crates:', error)
      setCrates({ crates: [], total: 0, limit: 20, offset: 0, has_next: false })
    } finally {
      setLoadingCrates(false)
    }
  }

  // Fetch tracks based on selected crate
  const fetchTracks = useMemo(() => async () => {
    try {
      let data
      if (selectedCrate === 'unsorted') {
        const params: UnsortedParams = { q: q || undefined }
        const response = await tracksApi.getUnsorted(params)
        data = response.data
      } else if (selectedCrate && selectedCrate !== 'all') {
        const response = await api.get('/api/tracks', {
          params: { q: q || undefined, playlist_id: selectedCrate }
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
  }, [q, selectedCrate])

  useEffect(() => {
    fetchCrates()
  }, [])

  // Keep selected crate in sync with URL (?crate=<id>|unsorted or none => all)
  useEffect(() => {
    const c = searchParams.get('crate')
    const next = c || 'all'
    if (next !== selectedCrate) {
      setSelectedCrate(next)
    }
  }, [searchParams])

  // Update player context when crate changes
  useEffect(() => {
    setCurrentCrate(selectedCrate)
  }, [selectedCrate, setCurrentCrate])

  // Only push to URL if the change originated from in-page actions (not from URL itself)
  const [lastUrlCrate, setLastUrlCrate] = useState<string | null>(searchParams.get('crate'))
  useEffect(() => {
    const curr = searchParams.get('crate') || 'all'
    // If selectedCrate differs from the URL and the URL hasn't just changed, update URL
    if (selectedCrate !== curr && lastUrlCrate === curr) {
      const next = new URLSearchParams(searchParams)
      if (selectedCrate && selectedCrate !== 'all') next.set('crate', selectedCrate)
      else next.delete('crate')
      setSearchParams(next, { replace: true })
    }
    setLastUrlCrate(searchParams.get('crate'))
  }, [selectedCrate, searchParams])

  useEffect(() => {
    fetchTracks()
  }, [fetchTracks])

  // Close dropdowns when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (showCrateDropdown && !(event.target as Element).closest('.crate-dropdown')) {
        setShowCrateDropdown(false)
      }
      if (trackMenuOpen && !(event.target as Element).closest('.track-menu')) {
        setTrackMenuOpen(null)
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [showCrateDropdown, trackMenuOpen])


  async function onDelete(id: string) {
    await api.delete(`/api/tracks/${id}`)
    await fetchTracks()
  }

  async function addTrackToCrate(trackId: string, crateId: string) {
    try {
      await cratesApi.addTracks(crateId, [trackId])
      await fetchTracks()
      await fetchCrates()
    } catch (error) {
      console.error('Failed to add track to crate:', error)
    }
  }

  async function removeTrackFromCrate(trackId: string, crateId: string) {
    try {
      await cratesApi.removeTracks(crateId, [trackId])
      await fetchTracks()
      await fetchCrates()
    } catch (error) {
      console.error('Failed to remove track from crate:', error)
    }
  }

  // Bulk actions
  async function bulkAddToCrate(crateId: string) {
    try {
      const ids = Array.from(selectedTrackIds)
      if (ids.length === 0) return
      await cratesApi.addTracks(crateId, ids)
      setSelectedTrackIds(new Set())
      await fetchTracks()
      await fetchCrates()
    } catch (error) {
      console.error('Failed bulk add:', error)
    }
  }

  async function bulkRemoveFromCurrentCrate() {
    try {
      if (!selectedCrate || selectedCrate === 'all' || selectedCrate === 'unsorted') return
      const ids = Array.from(selectedTrackIds)
      if (ids.length === 0) return
      await cratesApi.removeTracks(selectedCrate, ids)
      setSelectedTrackIds(new Set())
      await fetchTracks()
      await fetchCrates()
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

  // selected crate is controlled by the URL and sidebar links

  return (
    <div className="space-y-6">
      {/* Bulk selection toolbar */}
      {selectedTrackIds.size > 0 && (
        <div className="card p-3 flex flex-wrap items-center gap-3">
          <div className="text-sm">{selectedTrackIds.size} selected</div>
          <div className="relative">
            <button className="btn btn-primary" onClick={() => setShowCrateDropdown(!showCrateDropdown)}>
              Add to crate
            </button>
            {showCrateDropdown && (
              <div className="crate-dropdown absolute top-full mt-1 w-56 bg-[#1A1A1A] rounded-lg shadow-lg border border-[#2A2A2A] py-1 z-50">
                {loadingCrates ? (
                  <div className="px-3 py-2 text-sm text-[#A1A1A1]">Loading crates...</div>
                ) : (
                  crates.crates?.filter(c => !c.is_default).map((c) => (
                    <button
                      key={c.id}
                      className="w-full text-left px-3 py-2 text-sm hover:bg-[#2A2A2A]"
                      onClick={() => { bulkAddToCrate(c.id); setShowCrateDropdown(false) }}
                    >
                      {c.name}
                    </button>
                  ))
                )}
              </div>
            )}
          </div>
          {selectedCrate && selectedCrate !== 'all' && selectedCrate !== 'unsorted' && (
            <button className="btn" onClick={bulkRemoveFromCurrentCrate}>Remove from this crate</button>
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
                  <div className="track-menu absolute right-0 top-full mt-1 w-48 bg-[#1A1A1A] rounded-lg shadow-lg border border-[#2A2A2A] py-1 z-40">
                    <div className="px-3 py-2 text-xs font-semibold text-[#A1A1A1] border-b border-[#2A2A2A]">Add to Crate</div>
                    {loadingCrates ? (
                      <div className="px-3 py-2 text-sm text-[#A1A1A1]">Loading crates...</div>
                    ) : (
                      crates.crates && crates.crates
                        .filter(c => !c.is_default)
                        .map((crate) => (
                          <button
                            key={crate.id}
                            onClick={() => {
                              addTrackToCrate(t.id, crate.id)
                              setTrackMenuOpen(null)
                            }}
                            className="w-full text-left px-3 py-2 text-sm hover:bg-[#2A2A2A] flex items-center gap-2"
                          >
                            <ListPlus size={14} />
                            {crate.name}
                          </button>
                        ))
                    )}
                    {selectedCrate && selectedCrate !== 'all' && selectedCrate !== 'unsorted' && (
                      <>
                        <div className="border-t border-[#2A2A2A] my-1"></div>
                        <button
                          onClick={() => {
                            removeTrackFromCrate(t.id, selectedCrate)
                            setTrackMenuOpen(null)
                          }}
                          className="w-full text-left px-3 py-2 text-sm hover:bg-[#2A2A2A] text-red-400 flex items-center gap-2"
                        >
                          <ListPlus size={14} className="rotate-45" />
                          Remove from Crate
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

