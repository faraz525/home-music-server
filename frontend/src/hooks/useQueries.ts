import { useQuery, useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, cratesApi, normalizeCrateList, tracksApi } from '../lib/api'
import type { CrateList, TrackList } from '../types/crates'

// Query keys
export const queryKeys = {
  crates: ['crates'] as const,
  tracks: (params: { q?: string; selectedCrate?: string }) => ['tracks', params] as const,
}

// Default empty states
const emptyCrateList: CrateList = { crates: [], total: 0, limit: 20, offset: 0, has_next: false }
const emptyTrackList: TrackList = { tracks: [], total: 0, limit: 20, offset: 0, has_next: false }

// Backend caps per-request track lists at 100; we page through via offset.
const TRACKS_PAGE_SIZE = 100

// Crates query hook
export function useCrates() {
  return useQuery({
    queryKey: queryKeys.crates,
    queryFn: async () => {
      const { data } = await cratesApi.list()
      return normalizeCrateList(data)
    },
    placeholderData: emptyCrateList,
  })
}

// Tracks query hook — paginates via useInfiniteQuery, exposes a flattened
// TrackList via `data` so consumers keep their existing shape.
export function useTracks(params: { q?: string; selectedCrate?: string }) {
  const { q, selectedCrate } = params

  return useInfiniteQuery({
    queryKey: queryKeys.tracks(params),
    initialPageParam: 0,
    queryFn: async ({ pageParam }) => {
      const offset = pageParam as number
      const pageParams = { q: q || undefined, limit: TRACKS_PAGE_SIZE, offset }

      let response
      if (selectedCrate === 'unsorted') {
        response = await tracksApi.getUnsorted(pageParams)
      } else if (selectedCrate && selectedCrate !== 'all') {
        response = await api.get('/api/tracks', {
          params: { ...pageParams, playlist_id: selectedCrate },
        })
      } else {
        response = await api.get('/api/tracks', { params: pageParams })
      }

      const data = response.data
      if (data && Array.isArray(data.tracks)) {
        return data as TrackList
      }
      return emptyTrackList
    },
    getNextPageParam: (lastPage) => {
      if (!lastPage.has_next) return undefined
      return lastPage.offset + lastPage.tracks.length
    },
    select: (data) => {
      const pages = data.pages as TrackList[]
      const tracks = pages.flatMap((p) => p.tracks)
      const last = pages[pages.length - 1]
      return {
        tracks,
        total: last?.total ?? tracks.length,
        limit: last?.limit ?? TRACKS_PAGE_SIZE,
        offset: 0,
        has_next: Boolean(last?.has_next),
      } as TrackList
    },
  })
}

// Mutation hooks
export function useDeleteTrack() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (trackId: string) => api.delete(`/api/tracks/${trackId}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tracks'] })
      queryClient.invalidateQueries({ queryKey: queryKeys.crates })
    },
  })
}

export function useAddTracksToCrate() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ crateId, trackIds }: { crateId: string; trackIds: string[] }) =>
      cratesApi.addTracks(crateId, trackIds),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tracks'] })
      queryClient.invalidateQueries({ queryKey: queryKeys.crates })
      window.dispatchEvent(new CustomEvent('crates:updated'))
    },
  })
}

export function useRemoveTracksFromCrate() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ crateId, trackIds }: { crateId: string; trackIds: string[] }) =>
      cratesApi.removeTracks(crateId, trackIds),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tracks'] })
      queryClient.invalidateQueries({ queryKey: queryKeys.crates })
      window.dispatchEvent(new CustomEvent('crates:updated'))
    },
  })
}

export function useCreateCrate() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: { name: string; description?: string; is_public?: boolean }) =>
      cratesApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.crates })
      window.dispatchEvent(new CustomEvent('crates:updated'))
    },
  })
}

export function useUpdateCrate() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: { name: string; description?: string; is_public?: boolean } }) =>
      cratesApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.crates })
      window.dispatchEvent(new CustomEvent('crates:updated'))
    },
  })
}

export function useDeleteCrate() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (crateId: string) => cratesApi.delete(crateId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.crates })
      window.dispatchEvent(new CustomEvent('crates:updated'))
    },
  })
}

export function useUpdateTrackAnalysis() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: { bpm?: number; musical_key?: string } }) =>
      tracksApi.patch(id, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tracks'] })
    },
  })
}
