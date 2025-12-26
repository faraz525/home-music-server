import { tracksApi } from '../lib/api'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { usePlayer } from '../state/player'
import { Link, useSearchParams } from 'react-router-dom'
import { MoreHorizontal, ListPlus, Play, Pause, Music, Download } from 'lucide-react'
import { useToast } from '../hooks/useToast'
import { useCrates, useTracks, useDeleteTrack, useAddTracksToCrate, useRemoveTracksFromCrate } from '../hooks/useQueries'

export function LibraryPage() {
  const { play, isPlaying, toggle, queue, index, setCurrentCrate } = usePlayer()
  const current = queue[index]
  const toast = useToast()
  const [searchParams, setSearchParams] = useSearchParams()
  const [selectedCrate, setSelectedCrate] = useState<string>(() => searchParams.get('crate') || 'all')
  const [showCrateDropdown, setShowCrateDropdown] = useState(false)
  const [trackMenuOpen, setTrackMenuOpen] = useState<string | null>(null)
  const [selectedTrackIds, setSelectedTrackIds] = useState<Set<string>>(new Set())
  const [lastClickedIndex, setLastClickedIndex] = useState<number | null>(null)
  const [draggingTrackIds, setDraggingTrackIds] = useState<Set<string>>(new Set())

  // Debounced search state
  const [searchInput, setSearchInput] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')

  // Debounce search input
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(searchInput)
    }, 500)
    return () => clearTimeout(timer)
  }, [searchInput])

  // React Query hooks
  const { data: crates, isLoading: loadingCrates } = useCrates()
  const { data: tracks, isLoading: loadingTracks } = useTracks({
    q: debouncedSearch || undefined,
    selectedCrate,
  })

  // Mutations
  const deleteTrackMutation = useDeleteTrack()
  const addTracksMutation = useAddTracksToCrate()
  const removeTracksMutation = useRemoveTracksFromCrate()

  // Keep selected crate in sync with URL (?crate=<id>|unsorted or none => all)
  useEffect(() => {
    const c = searchParams.get('crate')
    const next = c || 'all'
    if (next !== selectedCrate) {
      setSelectedCrate(next)
    }
  }, [searchParams, selectedCrate])

  // Update player context when crate changes
  useEffect(() => {
    setCurrentCrate(selectedCrate)
  }, [selectedCrate, setCurrentCrate])

  // Only push to URL if the change originated from in-page actions (not from URL itself)
  const [lastUrlCrate, setLastUrlCrate] = useState<string | null>(searchParams.get('crate'))
  useEffect(() => {
    const curr = searchParams.get('crate') || 'all'
    if (selectedCrate !== curr && lastUrlCrate === curr) {
      const next = new URLSearchParams(searchParams)
      if (selectedCrate && selectedCrate !== 'all') next.set('crate', selectedCrate)
      else next.delete('crate')
      setSearchParams(next, { replace: true })
    }
    setLastUrlCrate(searchParams.get('crate'))
  }, [selectedCrate, searchParams, lastUrlCrate, setSearchParams])

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

  // Memoized handlers to prevent unnecessary re-renders
  const onDelete = useCallback(async (id: string) => {
    try {
      await deleteTrackMutation.mutateAsync(id)
      toast.success('Track deleted successfully')
    } catch (error) {
      console.error('Failed to delete track:', error)
      toast.error('Failed to delete track')
    }
  }, [deleteTrackMutation, toast])

  const addTrackToCrate = useCallback(async (trackId: string, crateId: string) => {
    try {
      await addTracksMutation.mutateAsync({ crateId, trackIds: [trackId] })
      toast.success('Track added to crate')
    } catch (error) {
      console.error('Failed to add track to crate:', error)
      toast.error('Failed to add track to crate')
    }
  }, [addTracksMutation, toast])

  const removeTrackFromCrate = useCallback(async (trackId: string, crateId: string) => {
    try {
      await removeTracksMutation.mutateAsync({ crateId, trackIds: [trackId] })
      toast.success('Track removed from crate')
    } catch (error) {
      console.error('Failed to remove track from crate:', error)
      toast.error('Failed to remove track from crate')
    }
  }, [removeTracksMutation, toast])

  const bulkAddToCrate = useCallback(async (crateId: string) => {
    try {
      const ids = Array.from(selectedTrackIds)
      if (ids.length === 0) return
      await addTracksMutation.mutateAsync({ crateId, trackIds: ids })
      setSelectedTrackIds(new Set())
      toast.success(`Added ${ids.length} track${ids.length > 1 ? 's' : ''} to crate`)
    } catch (error) {
      console.error('Failed bulk add:', error)
      toast.error('Failed to add tracks to crate')
    }
  }, [selectedTrackIds, addTracksMutation, toast])

  const bulkRemoveFromCurrentCrate = useCallback(async () => {
    try {
      if (!selectedCrate || selectedCrate === 'all' || selectedCrate === 'unsorted') return
      const ids = Array.from(selectedTrackIds)
      if (ids.length === 0) return
      await removeTracksMutation.mutateAsync({ crateId: selectedCrate, trackIds: ids })
      setSelectedTrackIds(new Set())
      toast.success(`Removed ${ids.length} track${ids.length > 1 ? 's' : ''} from crate`)
    } catch (error) {
      console.error('Failed bulk remove:', error)
      toast.error('Failed to remove tracks from crate')
    }
  }, [selectedCrate, selectedTrackIds, removeTracksMutation, toast])

  const toggleSelected = useCallback((id: string, shiftKey: boolean = false, idx: number) => {
    if (shiftKey && lastClickedIndex !== null && tracks?.tracks) {
      const start = Math.min(lastClickedIndex, idx)
      const end = Math.max(lastClickedIndex, idx)
      const rangeIds = tracks.tracks.slice(start, end + 1).map(t => t.id)
      setSelectedTrackIds((prev) => {
        const next = new Set(prev)
        rangeIds.forEach(id => next.add(id))
        return next
      })
    } else {
      setSelectedTrackIds((prev) => {
        const next = new Set(prev)
        if (next.has(id)) next.delete(id)
        else next.add(id)
        return next
      })
    }
    setLastClickedIndex(idx)
  }, [lastClickedIndex, tracks?.tracks])

  const clearSelection = useCallback(() => {
    setSelectedTrackIds(new Set())
    setLastClickedIndex(null)
  }, [])

  const formatDuration = useCallback((totalSeconds?: number) => {
    if (!totalSeconds || totalSeconds <= 0) return '—'
    const minutes = Math.floor(totalSeconds / 60)
    const seconds = Math.floor(totalSeconds % 60)
    return `${minutes}:${seconds.toString().padStart(2, '0')}`
  }, [])

  const handleDragStart = useCallback((e: React.DragEvent, trackId: string) => {
    const target = e.target as HTMLElement
    if (target.tagName === 'BUTTON' || target.tagName === 'INPUT' || target.closest('button')) {
      e.preventDefault()
      return
    }

    const tracksToDrag = selectedTrackIds.has(trackId)
      ? Array.from(selectedTrackIds)
      : [trackId]

    setDraggingTrackIds(new Set(tracksToDrag))
    e.dataTransfer.effectAllowed = 'copy'
    e.dataTransfer.setData('application/json', JSON.stringify({ trackIds: tracksToDrag }))
  }, [selectedTrackIds])

  const handleDragEnd = useCallback(() => {
    setDraggingTrackIds(new Set())
  }, [])

  // Get the current crate name for display
  const currentCrateName = useMemo(() => {
    if (!selectedCrate || selectedCrate === 'all') return 'Your Library'
    if (selectedCrate === 'unsorted') return 'Unsorted'
    const crate = crates?.crates?.find(c => c.id === selectedCrate)
    return crate?.name || 'Your Library'
  }, [selectedCrate, crates?.crates])

  const currentCrateDescription = useMemo(() => {
    if (!selectedCrate || selectedCrate === 'all') return 'All your music in one place'
    if (selectedCrate === 'unsorted') return 'Tracks not assigned to any crate'
    const crate = crates?.crates?.find(c => c.id === selectedCrate)
    return crate?.description || 'Browse your music'
  }, [selectedCrate, crates?.crates])

  return (
    <div className="space-y-6 overflow-visible">
      <div>
        <h1 className="text-2xl font-bold">{currentCrateName}</h1>
        <p className="text-[#A1A1A1] mt-1">{currentCrateDescription}</p>
      </div>

      {/* Bulk selection toolbar */}
      {selectedTrackIds.size > 0 && (
        <div className="card p-3 flex flex-wrap items-center gap-3">
          <div className="text-sm">{selectedTrackIds.size} selected</div>
          <div className="relative">
            <button className="btn btn-primary" onClick={() => setShowCrateDropdown(!showCrateDropdown)}>
              Add to crate
            </button>
            {showCrateDropdown && (
              <div className="crate-dropdown absolute top-full mt-1 w-56 bg-[#1A1A1A] rounded-lg shadow-lg border border-[#2A2A2A] py-1 z-[9999]">
                {loadingCrates ? (
                  <div className="px-3 py-2 text-sm text-[#A1A1A1]">Loading crates...</div>
                ) : (
                  crates?.crates?.filter(c => !c.is_default).map((c) => (
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
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            placeholder="Search your library"
            className="input flex-1"
          />
          {searchInput && (
            <button
              className="btn"
              onClick={() => setSearchInput('')}
            >
              Clear
            </button>
          )}
        </div>
      </div>


      <div className="card p-0">
        {/* Header row */}
        <div className="px-4 py-2 text-xs uppercase tracking-wide text-[#A1A1A1] grid grid-cols-[24px_1fr_1fr_120px_60px_32px] items-center gap-3 border-b border-[#2A2A2A] min-w-[800px]">
          <input
            type="checkbox"
            checked={tracks?.tracks && tracks.tracks.length > 0 && selectedTrackIds.size === tracks.tracks.length}
            onChange={(e) => {
              if (e.target.checked && tracks?.tracks) setSelectedTrackIds(new Set(tracks.tracks.map(t => t.id)))
              else clearSelection()
            }}
          />
          <div className="col-span-1">Title</div>
          <div>Album</div>
          <div className="text-right pr-4">Date added</div>
          <div className="text-right">Duration</div>
          <div></div>
        </div>

        {/* Loading state */}
        {loadingTracks && (
          <div className="px-4 py-8 text-center text-[#A1A1A1]">
            Loading tracks...
          </div>
        )}

        {/* Empty state */}
        {!loadingTracks && (!tracks?.tracks || tracks.tracks.length === 0) && (
          <div className="px-4 py-8 text-center">
            <Music size={48} className="mx-auto text-[#A1A1A1] mb-4" />
            <div className="text-lg font-semibold mb-2">No tracks found</div>
            <div className="text-[#A1A1A1] mb-4">
              {searchInput ? 'Try adjusting your search query' : 'Upload some music to get started'}
            </div>
            {!searchInput && (
              <Link to="/upload" className="btn btn-primary inline-flex">
                Upload Music
              </Link>
            )}
          </div>
        )}

        {/* Rows */}
        {!loadingTracks && tracks?.tracks && tracks.tracks.map((t, idx) => {
          const isCurrent = current?.id === t.id
          const isCurrentAndPlaying = isCurrent && isPlaying
          const isSelected = selectedTrackIds.has(t.id)
          const isDragging = draggingTrackIds.has(t.id)
          return (
            <div
              key={t.id}
              className={`px-4 py-2 grid grid-cols-[24px_1fr_1fr_120px_60px_32px] items-center gap-3 hover:bg-[#1A1A1A] min-w-[800px] cursor-move transition-opacity ${isSelected ? 'bg-[#2A2A2A]' : ''} ${isDragging ? 'opacity-50' : ''}`}
              draggable
              onDragStart={(e) => handleDragStart(e, t.id)}
              onDragEnd={handleDragEnd}
            >
              <input
                type="checkbox"
                checked={isSelected}
                onClick={(e) => {
                  e.stopPropagation()
                  toggleSelected(t.id, e.shiftKey, idx)
                }}
                onChange={() => { }}
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
                        play({ id: t.id, title: displayTitle, artist: displayArtist, streamUrl: `/api/tracks/${t.id}/stream`, durationSeconds: t.duration_seconds }, true)
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
                  <div className="track-menu absolute right-0 top-full mt-1 w-48 bg-[#1A1A1A] rounded-lg shadow-lg border border-[#2A2A2A] py-1 z-50">
                    <div className="px-3 py-2 text-xs font-semibold text-[#A1A1A1] border-b border-[#2A2A2A]">Add to Crate</div>
                    {loadingCrates ? (
                      <div className="px-3 py-2 text-sm text-[#A1A1A1]">Loading crates...</div>
                    ) : (
                      crates?.crates && crates.crates
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
                        tracksApi.download(t.id, t.original_filename)
                        setTrackMenuOpen(null)
                      }}
                      className="w-full text-left px-3 py-2 text-sm hover:bg-[#2A2A2A] flex items-center gap-2"
                    >
                      <Download size={14} />
                      Download
                    </button>
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
