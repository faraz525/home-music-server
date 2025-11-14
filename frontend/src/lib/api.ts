import axios from 'axios'
import type { CrateList } from '../types/crates'

function getCookie(name: string) {
  const match = document.cookie.match(new RegExp('(?:^|; )' + name.replace(/([.$?*|{}()\[\]\\\/\+^])/g, '\\$1') + '=([^;]*)'))
  return match ? decodeURIComponent(match[1]) : ''
}

export const api = axios.create({ withCredentials: true })

api.interceptors.request.use((config) => {
  const token = getCookie('access_token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

api.interceptors.response.use(
  (res) => {
    const payload = res?.data
    if (payload && typeof payload === 'object' && 'success' in payload && 'data' in payload) {
      return { ...res, data: (payload as any).data }
    }
    return res
  },
  async (error) => {
    if (error.response?.status === 401) {
      try {
        const { data } = await axios.post('/api/auth/refresh', {}, { withCredentials: true })
        if (data?.access_token) document.cookie = `access_token=${encodeURIComponent(data.access_token)}; Path=/; Max-Age=${60 * 15}`
        return api.request(error.config)
      } catch (_) {
        // fallthrough
      }
    }
    return Promise.reject(error)
  }
)

export function normalizeCrateList(raw: any): CrateList {
  const crates = Array.isArray(raw?.crates)
    ? raw.crates
    : Array.isArray(raw?.playlists)
      ? raw.playlists
      : []

  return {
    crates,
    total: typeof raw?.total === 'number' ? raw.total : crates.length,
    limit: typeof raw?.limit === 'number' ? raw.limit : 20,
    offset: typeof raw?.offset === 'number' ? raw.offset : 0,
    has_next: Boolean(raw?.has_next),
  }
}

// Crate (playlist) API functions
export const cratesApi = {
  create: (data: { name: string; description?: string }) =>
    api.post('/api/playlists', data),

  list: (params?: { limit?: number; offset?: number }) =>
    api.get('/api/playlists', { params }),

  get: (id: string) =>
    api.get(`/api/playlists/${id}`),

  update: (id: string, data: { name: string; description?: string }) =>
    api.put(`/api/playlists/${id}`, data),

  delete: (id: string) =>
    api.delete(`/api/playlists/${id}`),

  addTracks: (id: string, trackIds: string[]) =>
    api.post(`/api/playlists/${id}/tracks`, { track_ids: trackIds }),

  removeTracks: (id: string, trackIds: string[]) =>
    api.delete(`/api/playlists/${id}/tracks`, { data: { track_ids: trackIds } }),

  getTracks: (id: string, params?: { limit?: number; offset?: number }) =>
    api.get(`/api/playlists/${id}/tracks`, { params }),
}

// Backwards compatibility
export const playlistsApi = cratesApi

// Unsorted tracks
export type UnsortedParams = { limit?: number; offset?: number; q?: string }
export const tracksApi = {
  getUnsorted: (params?: UnsortedParams) =>
    api.get('/api/tracks', { params: { ...(params || {}), playlist_id: 'unsorted' } }),
}