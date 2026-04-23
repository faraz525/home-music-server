import { tracksApi } from '../lib/api'
import { useCallback, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import { usePlayer } from '../state/player'
import { Link, useSearchParams } from 'react-router-dom'
import { MoreHorizontal, ListPlus, Play, Pause, Music, Download, Disc, Search, X } from 'lucide-react'
import { useToast } from '../hooks/useToast'
import { useCrates, useTracks, useDeleteTrack, useAddTracksToCrate, useRemoveTracksFromCrate, useUpdateTrackAnalysis } from '../hooks/useQueries'

function MiniVinyl({ spinning = false }: { spinning?: boolean }) {
  return (
    <div className={`w-5 h-5 flex-shrink-0 ${spinning ? 'vinyl-spinning' : ''}`}>
      <div
        className="w-full h-full rounded-full"
        style={{
          background: `radial-gradient(circle at 50% 50%,
            #E5A000 0%,
            #E5A000 25%,
            #1A171F 26%,
            #0D0A14 40%,
            #1A171F 41%,
            #252130 100%
          )`,
        }}
      />
    </div>
  )
}

function ConfidenceDot({ status, confidence }: { status?: string; confidence?: number }) {
  let color = 'bg-transparent border border-crate-subtle' // pending default
  let title = 'Analyzing…'

  if (status === 'failed') {
    color = 'bg-transparent border border-crate-danger'
    title = 'Analysis failed'
  } else if (status === 'user_edited') {
    return null // no dot for user-edited values
  } else if (status === 'analyzed' && typeof confidence === 'number') {
    if (confidence > 0.7) {
      color = 'bg-green-500'
      title = `High confidence (${Math.round(confidence * 100)}%)`
    } else if (confidence >= 0.4) {
      color = 'bg-amber-500'
      title = `Medium confidence (${Math.round(confidence * 100)}%)`
    } else {
      color = 'bg-red-500'
      title = `Low confidence (${Math.round(confidence * 100)}%)`
    }
  }

  return <span className={`inline-block w-2 h-2 rounded-full ${color}`} title={title} />
}

function EditableCell({
  value,
  onSave,
  validate,
  display,
  align = 'right',
}: {
  value: string
  onSave: (next: string) => Promise<void> | void
  validate: (next: string) => boolean
  display: ReactNode
  align?: 'left' | 'right'
}) {
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(value)
  const [saving, setSaving] = useState(false)

  if (!editing) {
    return (
      <div
        onDoubleClick={(e) => {
          e.stopPropagation()
          setDraft(value)
          setEditing(true)
        }}
        className="cursor-text select-none"
        title="Double-click to edit"
      >
        {display}
      </div>
    )
  }

  const commit = async () => {
    if (!validate(draft)) {
      setEditing(false)
      return
    }
    setSaving(true)
    try {
      await onSave(draft)
    } finally {
      setSaving(false)
      setEditing(false)
    }
  }

  return (
    <input
      autoFocus
      value={draft}
      disabled={saving}
      onChange={(e) => setDraft(e.target.value)}
      onBlur={commit}
      onClick={(e) => e.stopPropagation()}
      onKeyDown={(e) => {
        if (e.key === 'Enter') commit()
        else if (e.key === 'Escape') setEditing(false)
      }}
      className={`input h-6 px-1 py-0 text-sm w-16 ${align === 'right' ? 'text-right' : ''}`}
    />
  )
}

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

  const [searchInput, setSearchInput] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(searchInput)
    }, 500)
    return () => clearTimeout(timer)
  }, [searchInput])

  const { data: crates, isLoading: loadingCrates } = useCrates()
  const {
    data: tracks,
    isLoading: loadingTracks,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useTracks({
    q: debouncedSearch || undefined,
    selectedCrate,
  })

  const [sentinelEl, setSentinelEl] = useState<HTMLDivElement | null>(null)
  useEffect(() => {
    if (!sentinelEl) return
    const io = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting && hasNextPage && !isFetchingNextPage) {
          fetchNextPage()
        }
      },
      { rootMargin: '200px' },
    )
    io.observe(sentinelEl)
    return () => io.disconnect()
  }, [sentinelEl, hasNextPage, isFetchingNextPage, fetchNextPage])

  const deleteTrackMutation = useDeleteTrack()
  const addTracksMutation = useAddTracksToCrate()
  const removeTracksMutation = useRemoveTracksFromCrate()
  const updateAnalysisMutation = useUpdateTrackAnalysis()

  const saveBpm = useCallback(async (trackId: string, raw: string) => {
    const bpm = parseFloat(raw)
    if (!isFinite(bpm) || bpm < 50 || bpm > 250) {
      toast.error('BPM must be between 50 and 250')
      return
    }
    try {
      await updateAnalysisMutation.mutateAsync({ id: trackId, payload: { bpm } })
      toast.success('BPM updated')
    } catch {
      toast.error('Failed to update BPM')
    }
  }, [updateAnalysisMutation, toast])

  const saveKey = useCallback(async (trackId: string, raw: string) => {
    const normalized = raw.trim().toUpperCase()
    if (!/^(1[0-2]|[1-9])[AB]$/.test(normalized)) {
      toast.error('Key must be Camelot notation (e.g. 8A, 12B)')
      return
    }
    try {
      await updateAnalysisMutation.mutateAsync({ id: trackId, payload: { musical_key: normalized } })
      toast.success('Key updated')
    } catch {
      toast.error('Failed to update key')
    }
  }, [updateAnalysisMutation, toast])

  useEffect(() => {
    const c = searchParams.get('crate')
    const next = c || 'all'
    if (next !== selectedCrate) {
      setSelectedCrate(next)
    }
  }, [searchParams, selectedCrate])

  useEffect(() => {
    setCurrentCrate(selectedCrate)
  }, [selectedCrate, setCurrentCrate])

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

  const bulkDownload = useCallback(() => {
    if (!tracks?.tracks) return
    const selectedTracks = tracks.tracks.filter(t => selectedTrackIds.has(t.id))
    if (selectedTracks.length === 0) return

    toast.success(`Starting download of ${selectedTracks.length} track${selectedTracks.length > 1 ? 's' : ''}...`)

    // Download tracks with a small delay between each to avoid overwhelming the browser
    selectedTracks.forEach((track, index) => {
      setTimeout(() => {
        tracksApi.download(track.id, track.original_filename)
      }, index * 300)
    })
  }, [tracks?.tracks, selectedTrackIds, toast])

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
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-end justify-between gap-4">
        <div>
          <h1 className="text-3xl font-display font-bold text-crate-cream">{currentCrateName}</h1>
          <p className="text-crate-muted mt-1">{currentCrateDescription}</p>
        </div>
        {tracks?.tracks && tracks.tracks.length > 0 && (
          <div className="text-sm text-crate-subtle">
            {tracks.total > tracks.tracks.length
              ? `${tracks.tracks.length} of ${tracks.total} tracks`
              : `${tracks.total} track${tracks.total !== 1 ? 's' : ''}`}
          </div>
        )}
      </div>

      {/* Bulk selection toolbar */}
      {selectedTrackIds.size > 0 && (
        <div className="card p-4 flex flex-wrap items-center gap-3 border-crate-amber/30 bg-crate-amber/5">
          <div className="text-sm font-medium text-crate-amber">
            {selectedTrackIds.size} selected
          </div>
          <div className="relative">
            <button className="btn btn-primary" onClick={() => setShowCrateDropdown(!showCrateDropdown)}>
              Add to crate
            </button>
            {showCrateDropdown && (
              <div className="crate-dropdown absolute top-full mt-2 w-56 bg-crate-elevated rounded-xl shadow-elevated border border-crate-border py-2 z-[9999]">
                {loadingCrates ? (
                  <div className="px-4 py-2 text-sm text-crate-muted">Loading crates...</div>
                ) : (
                  crates?.crates?.filter(c => !c.is_default).map((c) => (
                    <button
                      key={c.id}
                      className="w-full text-left px-4 py-2.5 text-sm text-crate-cream hover:bg-crate-border transition-colors"
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
            <button className="btn btn-danger" onClick={bulkRemoveFromCurrentCrate}>
              Remove from crate
            </button>
          )}
          <button className="btn flex items-center gap-2" onClick={bulkDownload}>
            <Download size={16} />
            Download
          </button>
          <button className="btn" onClick={clearSelection}>Clear</button>
        </div>
      )}

      {/* Search */}
      <div className="relative">
        <Search className="absolute left-3 sm:left-4 top-1/2 -translate-y-1/2 text-crate-subtle pointer-events-none" size={18} />
        <input
          value={searchInput}
          onChange={(e) => setSearchInput(e.target.value)}
          placeholder="Search..."
          className="input w-full pl-10 sm:pl-11 pr-10"
        />
        {searchInput && (
          <button
            className="absolute right-3 top-1/2 -translate-y-1/2 p-1 text-crate-muted hover:text-crate-cream transition-colors"
            onClick={() => setSearchInput('')}
          >
            <X size={16} />
          </button>
        )}
      </div>

      {/* Track list */}
      <div className="card overflow-visible">
        {/* Header row - hidden on mobile */}
        <div className="hidden sm:grid px-4 py-3 text-xs uppercase tracking-wider text-crate-subtle grid-cols-[28px_1fr_1fr_70px_70px_100px_60px_40px] items-center gap-3 border-b border-crate-border bg-crate-elevated/50">
          <input
            type="checkbox"
            checked={tracks?.tracks && tracks.tracks.length > 0 && selectedTrackIds.size === tracks.tracks.length}
            onChange={(e) => {
              if (e.target.checked && tracks?.tracks) setSelectedTrackIds(new Set(tracks.tracks.map(t => t.id)))
              else clearSelection()
            }}
            className="justify-self-center"
          />
          <div>Title</div>
          <div>Album</div>
          <div className="text-right">BPM</div>
          <div className="text-right">Key</div>
          <div className="text-right">Added</div>
          <div className="text-right">Time</div>
          <div></div>
        </div>

        {/* Loading state */}
        {loadingTracks && (
          <div className="px-4 py-12 text-center">
            <Disc className="mx-auto text-crate-amber vinyl-spinning-slow mb-4" size={48} />
            <div className="text-crate-muted">Loading tracks...</div>
          </div>
        )}

        {/* Empty state */}
        {!loadingTracks && (!tracks?.tracks || tracks.tracks.length === 0) && (
          <div className="px-4 py-12 text-center">
            <div className="w-20 h-20 mx-auto mb-4 rounded-full bg-crate-elevated flex items-center justify-center">
              <Music size={32} className="text-crate-subtle" />
            </div>
            <h3 className="text-lg font-display font-semibold text-crate-cream mb-2">No tracks found</h3>
            <p className="text-crate-muted mb-6">
              {searchInput ? 'Try adjusting your search query' : 'Upload some music to get started'}
            </p>
            {!searchInput && (
              <Link to="/upload" className="btn btn-primary">
                Upload Music
              </Link>
            )}
          </div>
        )}

        {/* Track rows */}
        {!loadingTracks && tracks?.tracks && tracks.tracks.map((t, idx) => {
          const isCurrent = current?.id === t.id
          const isCurrentAndPlaying = isCurrent && isPlaying
          const isSelected = selectedTrackIds.has(t.id)
          const isDragging = draggingTrackIds.has(t.id)

          return (
            <div
              key={t.id}
              className={`stagger-item group px-4 py-3 grid grid-cols-[1fr_auto] sm:grid-cols-[28px_1fr_1fr_70px_70px_100px_60px_40px] items-center gap-3 border-b border-crate-border/50 hover:bg-crate-elevated/50 transition-all cursor-move ${isSelected ? 'bg-crate-amber/5 border-l-2 border-l-crate-amber' : ''} ${isDragging ? 'opacity-50' : ''} ${isCurrent ? 'bg-crate-elevated/30' : ''}`}
              draggable
              onDragStart={(e) => handleDragStart(e, t.id)}
              onDragEnd={handleDragEnd}
            >
              {/* Mobile: Combined title/artist with play button */}
              <div className="sm:hidden flex items-center gap-3 min-w-0">
                <button
                  className={`hw-button p-2 flex-shrink-0 ${isCurrentAndPlaying ? 'hw-button-primary' : ''}`}
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
                  {isCurrentAndPlaying ? <Pause size={14} /> : <Play size={14} className="ml-0.5" />}
                </button>
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    {isCurrent && <MiniVinyl spinning={isPlaying} />}
                    <span className={`truncate text-sm ${isCurrent ? 'text-crate-amber font-medium' : 'text-crate-cream'}`}>
                      {t.title || t.original_filename}
                    </span>
                  </div>
                  <div className="text-xs text-crate-muted truncate">{t.artist || 'Unknown Artist'}</div>
                </div>
              </div>

              {/* Desktop: Checkbox */}
              <input
                type="checkbox"
                checked={isSelected}
                onClick={(e) => {
                  e.stopPropagation()
                  toggleSelected(t.id, e.shiftKey, idx)
                }}
                onChange={() => { }}
                className="hidden sm:block justify-self-center"
              />

              {/* Desktop: Title/Artist with play */}
              <div className="hidden sm:flex items-center gap-3 min-w-0">
                <button
                  className={`hw-button p-2 flex-shrink-0 opacity-0 group-hover:opacity-100 transition-opacity ${isCurrentAndPlaying ? 'hw-button-primary opacity-100' : ''}`}
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
                  {isCurrentAndPlaying ? <Pause size={14} /> : <Play size={14} className="ml-0.5" />}
                </button>
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    {isCurrent && <MiniVinyl spinning={isPlaying} />}
                    <span className={`truncate ${isCurrent ? 'text-crate-amber font-medium' : 'text-crate-cream'}`}>
                      {t.title || t.original_filename}
                    </span>
                  </div>
                  <div className="text-sm text-crate-muted truncate">{t.artist || 'Unknown Artist'}</div>
                </div>
              </div>

              {/* Desktop: Album */}
              <div className="hidden sm:block truncate text-sm text-crate-muted">{t.album || '—'}</div>

              {/* Desktop: BPM (editable) */}
              <div className="hidden sm:flex items-center justify-end gap-1.5 text-sm text-crate-subtle tabular-nums">
                <ConfidenceDot status={t.analysis_status} confidence={t.bpm_confidence} />
                <EditableCell
                  value={t.bpm ? t.bpm.toFixed(1) : ''}
                  display={<span>{t.bpm ? t.bpm.toFixed(1) : '—'}</span>}
                  validate={(v) => { const n = parseFloat(v); return isFinite(n) && n >= 50 && n <= 250 }}
                  onSave={(v) => saveBpm(t.id, v)}
                />
              </div>

              {/* Desktop: Key (editable) */}
              <div className="hidden sm:flex items-center justify-end gap-1.5 text-sm text-crate-subtle">
                <ConfidenceDot status={t.analysis_status} confidence={t.key_confidence} />
                <EditableCell
                  value={t.musical_key || ''}
                  display={<span>{t.musical_key || '—'}</span>}
                  validate={(v) => /^(1[0-2]|[1-9])[AB]$/i.test(v.trim())}
                  onSave={(v) => saveKey(t.id, v)}
                />
              </div>

              {/* Desktop: Date added */}
              <div className="hidden sm:block text-right text-sm text-crate-subtle">
                {t.created_at ? new Date(t.created_at).toLocaleDateString() : '—'}
              </div>

              {/* Desktop: Duration */}
              <div className="hidden sm:block text-right text-sm text-crate-subtle tabular-nums">
                {formatDuration(t.duration_seconds)}
              </div>

              {/* More menu */}
              <div className="relative justify-self-end">
                <button
                  className="btn-ghost p-2 rounded-lg"
                  onClick={() => setTrackMenuOpen(trackMenuOpen === t.id ? null : t.id)}
                >
                  <MoreHorizontal size={16} />
                </button>
                {trackMenuOpen === t.id && (
                  <>
                    {/* Mobile: Full screen overlay with bottom sheet style menu */}
                    <div className="sm:hidden fixed inset-0 bg-crate-black/60 z-[100]" onClick={() => setTrackMenuOpen(null)} />
                    <div className="track-menu sm:absolute sm:right-0 sm:top-full sm:mt-2 fixed sm:relative bottom-0 left-0 right-0 sm:bottom-auto sm:left-auto w-full sm:w-52 bg-crate-elevated sm:rounded-xl rounded-t-2xl shadow-elevated border border-crate-border py-2 z-[101] max-h-[70vh] overflow-y-auto">
                      <div className="sm:hidden w-12 h-1 bg-crate-border rounded-full mx-auto my-2" />
                      <div className="px-4 py-2 text-xs font-medium text-crate-subtle border-b border-crate-border mb-1">
                        Add to Crate
                      </div>
                      {loadingCrates ? (
                        <div className="px-4 py-2 text-sm text-crate-muted">Loading crates...</div>
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
                              className="w-full text-left px-4 py-3 sm:py-2 text-sm text-crate-cream hover:bg-crate-border active:bg-crate-border flex items-center gap-2 transition-colors"
                            >
                              <ListPlus size={14} className="text-crate-muted" />
                              {crate.name}
                            </button>
                          ))
                      )}
                      {selectedCrate && selectedCrate !== 'all' && selectedCrate !== 'unsorted' && (
                        <>
                          <div className="border-t border-crate-border my-1" />
                          <button
                            onClick={() => {
                              removeTrackFromCrate(t.id, selectedCrate)
                              setTrackMenuOpen(null)
                            }}
                            className="w-full text-left px-4 py-3 sm:py-2 text-sm text-crate-danger hover:bg-crate-danger/10 active:bg-crate-danger/10 flex items-center gap-2 transition-colors"
                          >
                            <ListPlus size={14} className="rotate-45" />
                            Remove from Crate
                          </button>
                        </>
                      )}
                      <div className="border-t border-crate-border my-1" />
                      <button
                        onClick={() => {
                          tracksApi.download(t.id, t.original_filename)
                          setTrackMenuOpen(null)
                        }}
                        className="w-full text-left px-4 py-3 sm:py-2 text-sm text-crate-cream hover:bg-crate-border active:bg-crate-border flex items-center gap-2 transition-colors"
                      >
                        <Download size={14} className="text-crate-muted" />
                        Download
                      </button>
                      <div className="border-t border-crate-border my-1" />
                      <button
                        onClick={() => {
                          onDelete(t.id)
                          setTrackMenuOpen(null)
                        }}
                        className="w-full text-left px-4 py-3 sm:py-2 text-sm text-crate-danger hover:bg-crate-danger/10 active:bg-crate-danger/10 flex items-center gap-2 transition-colors"
                      >
                        <MoreHorizontal size={14} />
                        Delete Track
                      </button>
                      {/* Extra padding for safe area on mobile */}
                      <div className="sm:hidden h-4" />
                    </div>
                  </>
                )}
              </div>
            </div>
          )
        })}

        {/* Infinite-scroll sentinel + status row */}
        {!loadingTracks && tracks?.tracks && tracks.tracks.length > 0 && hasNextPage && (
          <div
            ref={setSentinelEl}
            className="px-4 py-6 text-center text-sm text-crate-subtle"
          >
            {isFetchingNextPage ? 'Loading more…' : 'Scroll for more'}
          </div>
        )}
      </div>
    </div>
  )
}
