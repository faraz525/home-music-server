import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
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

// Tracks query hook
export function useTracks(params: { q?: string; selectedCrate?: string }) {
  const { q, selectedCrate } = params

  return useQuery({
    queryKey: queryKeys.tracks(params),
    queryFn: async () => {
      let data
      if (selectedCrate === 'unsorted') {
        const response = await tracksApi.getUnsorted({ q: q || undefined })
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

      if (data && data.tracks && Array.isArray(data.tracks)) {
        return data as TrackList
      }
      return emptyTrackList
    },
    placeholderData: emptyTrackList,
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
