import { useEffect, useState } from 'react'
import { Globe, Music, Play, Pause, User } from 'lucide-react'
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
    const { play, queue, index } = usePlayer()

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

        // Fetch tracks if not already loaded
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
        play({
            id: track.id,
            title: track.title || track.original_filename,
            artist: track.artist || 'Unknown Artist',
            streamUrl: `/api/tracks/${track.id}/stream`,
            durationSeconds: track.duration_seconds
        }, true)
    }

    if (loading) {
        return (
            <div className="flex items-center justify-center py-12">
                <div className="text-[#A1A1A1]">Loading community crates...</div>
            </div>
        )
    }

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <div className="flex items-center gap-2">
                        <Globe size={24} className="text-[#1DB954]" />
                        <h1 className="text-2xl font-bold">Community</h1>
                    </div>
                    <p className="text-[#A1A1A1] mt-1">Browse public crates from other users</p>
                </div>
            </div>

            {crates.length === 0 ? (
                <div className="text-center py-12">
                    <Music size={48} className="mx-auto text-[#A1A1A1] mb-4" />
                    <h3 className="text-lg font-semibold mb-2">No public crates yet</h3>
                    <p className="text-[#A1A1A1]">No users have shared their crates publicly</p>
                </div>
            ) : (
                <div className="space-y-4">
                    {crates.map((crate) => (
                        <div key={crate.id} className="card p-4">
                            <div
                                className="flex items-start justify-between cursor-pointer"
                                onClick={() => toggleCrateExpansion(crate.id)}
                            >
                                <div className="flex items-center gap-3 flex-1">
                                    <div className="w-12 h-12 bg-gradient-to-br from-[#1DB954] to-[#1ed760] rounded-lg flex items-center justify-center">
                                        <Music size={24} className="text-black" />
                                    </div>
                                    <div className="flex-1 min-w-0">
                                        <h3 className="font-semibold truncate">{crate.name}</h3>
                                        <p className="text-sm text-[#A1A1A1] truncate">
                                            {crate.description || 'No description'}
                                        </p>
                                        <div className="flex items-center gap-2 mt-1">
                                            <User size={12} className="text-[#A1A1A1]" />
                                            <span className="text-xs text-[#A1A1A1]">{crate.owner_email}</span>
                                        </div>
                                    </div>
                                </div>
                                <span className="text-xs bg-[#1DB954]/20 text-[#1DB954] px-2 py-1 rounded">
                                    Public
                                </span>
                            </div>

                            {/* Expanded tracks view */}
                            {expandedCrate === crate.id && (
                                <div className="mt-4 pt-4 border-t border-[#2A2A2A]">
                                    {loadingTracks === crate.id ? (
                                        <div className="text-center py-4 text-[#A1A1A1]">
                                            Loading tracks...
                                        </div>
                                    ) : crate.tracks && crate.tracks.length > 0 ? (
                                        <div className="space-y-2">
                                            {crate.tracks.map((track: any) => (
                                                <div
                                                    key={track.id}
                                                    className="flex items-center justify-between p-2 hover:bg-[#2A2A2A] rounded group"
                                                >
                                                    <div className="flex-1 min-w-0">
                                                        <div className="font-medium truncate">
                                                            {track.title || track.original_filename}
                                                        </div>
                                                        <div className="text-sm text-[#A1A1A1] truncate">
                                                            {track.artist || 'Unknown Artist'}
                                                        </div>
                                                    </div>
                                                    <button
                                                        onClick={(e) => {
                                                            e.stopPropagation()
                                                            handlePlayTrack(track, crate.id)
                                                        }}
                                                        className="p-2 hover:bg-[#1DB954]/20 rounded-full transition-colors"
                                                    >
                                                        {queue[index]?.id === track.id ? (
                                                            <Pause size={18} className="text-[#1DB954]" />
                                                        ) : (
                                                            <Play size={18} className="text-[#1DB954]" />
                                                        )}
                                                    </button>
                                                </div>
                                            ))}
                                        </div>
                                    ) : (
                                        <div className="text-center py-4 text-[#A1A1A1]">
                                            No tracks in this crate
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
