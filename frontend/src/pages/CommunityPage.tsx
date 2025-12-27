import { useEffect, useState } from 'react'
import { Globe, Music, Play, Pause, User, ChevronDown, ChevronUp, Disc } from 'lucide-react'
import { communityApi, cratesApi } from '../lib/api'
import type { PlaylistWithOwner } from '../types/crates'
import { usePlayer } from '../state/player'

type CrateWithTracks = PlaylistWithOwner & {
    tracks?: any[]
    tracksExpanded?: boolean
}

export function CommunityPage() {
    const [crates, setCrates] = useState<CrateWithTracks[]>([])
    const [loading, setLoading] = useState(true)
    const [expandedCrate, setExpandedCrate] = useState<string | null>(null)
    const [loadingTracks, setLoadingTracks] = useState<string | null>(null)
    const { play, queue, index, isPlaying, toggle } = usePlayer()

    useEffect(() => {
        fetchPublicCrates()
    }, [])

    const fetchPublicCrates = async () => {
        try {
            const { data } = await communityApi.getPublicCrates()
            const normalizedCrates = (data?.crates || []).map((crate: PlaylistWithOwner) => ({
                ...crate,
                tracksExpanded: false,
            }))
            setCrates(normalizedCrates)
        } catch (error) {
            console.error('Failed to fetch public crates:', error)
            setCrates([])
        } finally {
            setLoading(false)
        }
    }

    const toggleCrateExpansion = async (crateId: string) => {
        if (expandedCrate === crateId) {
            setExpandedCrate(null)
            return
        }

        setExpandedCrate(crateId)

        const crate = crates.find(c => c.id === crateId)
        if (crate && !crate.tracks) {
            setLoadingTracks(crateId)
            try {
                const { data } = await cratesApi.getTracks(crateId)
                setCrates(prevCrates =>
                    prevCrates.map(c =>
                        c.id === crateId ? { ...c, tracks: data.tracks || [] } : c
                    )
                )
            } catch (error) {
                console.error('Failed to fetch tracks:', error)
            } finally {
                setLoadingTracks(null)
            }
        }
    }

    const handlePlayTrack = (track: any, crateId: string) => {
        const currentTrack = queue[index]
        if (currentTrack?.id === track.id) {
            toggle()
        } else {
            play({
                id: track.id,
                title: track.title || track.original_filename,
                artist: track.artist || 'Unknown Artist',
                streamUrl: `/api/tracks/${track.id}/stream`,
                durationSeconds: track.duration_seconds
            }, true)
        }
    }

    if (loading) {
        return (
            <div className="flex flex-col items-center justify-center py-16">
                <Disc className="text-crate-amber vinyl-spinning-slow mb-4" size={48} />
                <div className="text-crate-muted">Loading community crates...</div>
            </div>
        )
    }

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex items-center gap-4">
                <div className="p-3 rounded-xl bg-crate-cyan/10">
                    <Globe size={24} className="text-crate-cyan" />
                </div>
                <div>
                    <h1 className="text-3xl font-display font-bold text-crate-cream">Community</h1>
                    <p className="text-crate-muted mt-1">Browse public crates from other users</p>
                </div>
            </div>

            {/* Empty state */}
            {crates.length === 0 ? (
                <div className="text-center py-16">
                    <div className="w-24 h-24 mx-auto mb-6 rounded-full bg-crate-elevated flex items-center justify-center">
                        <Music size={40} className="text-crate-subtle" />
                    </div>
                    <h3 className="text-xl font-display font-semibold text-crate-cream mb-2">No public crates yet</h3>
                    <p className="text-crate-muted">No users have shared their crates publicly</p>
                </div>
            ) : (
                <div className="space-y-4">
                    {crates.map((crate, idx) => (
                        <div
                            key={crate.id}
                            className="stagger-item card overflow-hidden transition-all"
                            style={{ animationDelay: `${idx * 50}ms` }}
                        >
                            {/* Crate header */}
                            <div
                                className="p-5 flex items-start justify-between cursor-pointer hover:bg-crate-elevated/30 transition-colors"
                                onClick={() => toggleCrateExpansion(crate.id)}
                            >
                                <div className="flex items-center gap-4 flex-1 min-w-0">
                                    <div className="w-14 h-14 rounded-xl bg-gradient-to-br from-crate-cyan to-crate-cyanDark flex items-center justify-center flex-shrink-0 shadow-glow-cyan">
                                        <Music size={24} className="text-crate-black" />
                                    </div>
                                    <div className="flex-1 min-w-0">
                                        <h3 className="font-display font-semibold text-crate-cream truncate">
                                            {crate.name}
                                        </h3>
                                        <p className="text-sm text-crate-muted truncate mt-0.5">
                                            {crate.description || 'No description'}
                                        </p>
                                        <div className="flex items-center gap-2 mt-2">
                                            <User size={12} className="text-crate-subtle" />
                                            <span className="text-xs text-crate-subtle">{crate.owner_email}</span>
                                        </div>
                                    </div>
                                </div>

                                <div className="flex items-center gap-3 flex-shrink-0">
                                    <span className="text-xs bg-crate-cyan/10 text-crate-cyan px-3 py-1.5 rounded-lg font-medium">
                                        Public
                                    </span>
                                    {expandedCrate === crate.id ? (
                                        <ChevronUp size={18} className="text-crate-muted" />
                                    ) : (
                                        <ChevronDown size={18} className="text-crate-muted" />
                                    )}
                                </div>
                            </div>

                            {/* Expanded tracks view */}
                            {expandedCrate === crate.id && (
                                <div className="border-t border-crate-border bg-crate-elevated/20">
                                    {loadingTracks === crate.id ? (
                                        <div className="flex items-center justify-center gap-3 py-8">
                                            <Disc className="text-crate-amber vinyl-spinning" size={24} />
                                            <span className="text-crate-muted">Loading tracks...</span>
                                        </div>
                                    ) : crate.tracks && crate.tracks.length > 0 ? (
                                        <div className="divide-y divide-crate-border/50">
                                            {crate.tracks.map((track: any) => {
                                                const isCurrent = queue[index]?.id === track.id
                                                const isCurrentAndPlaying = isCurrent && isPlaying

                                                return (
                                                    <div
                                                        key={track.id}
                                                        className={`flex items-center justify-between px-5 py-3 hover:bg-crate-elevated/50 transition-colors group ${isCurrent ? 'bg-crate-amber/5' : ''}`}
                                                    >
                                                        <div className="flex items-center gap-3 flex-1 min-w-0">
                                                            <button
                                                                onClick={(e) => {
                                                                    e.stopPropagation()
                                                                    handlePlayTrack(track, crate.id)
                                                                }}
                                                                className={`hw-button p-2 flex-shrink-0 ${isCurrentAndPlaying ? 'hw-button-primary' : 'opacity-0 group-hover:opacity-100'} transition-opacity`}
                                                            >
                                                                {isCurrentAndPlaying ? (
                                                                    <Pause size={14} />
                                                                ) : (
                                                                    <Play size={14} className="ml-0.5" />
                                                                )}
                                                            </button>
                                                            <div className="min-w-0">
                                                                <div className={`font-medium truncate ${isCurrent ? 'text-crate-amber' : 'text-crate-cream'}`}>
                                                                    {track.title || track.original_filename}
                                                                </div>
                                                                <div className="text-sm text-crate-muted truncate">
                                                                    {track.artist || 'Unknown Artist'}
                                                                </div>
                                                            </div>
                                                        </div>
                                                    </div>
                                                )
                                            })}
                                        </div>
                                    ) : (
                                        <div className="text-center py-8">
                                            <Music size={24} className="mx-auto text-crate-subtle mb-2" />
                                            <p className="text-crate-muted text-sm">No tracks in this crate</p>
                                        </div>
                                    )}
                                </div>
                            )}
                        </div>
                    ))}
                </div>
            )}
        </div>
    )
}
