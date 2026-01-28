import axios from 'axios'
import type { CrateList } from '../types/crates'

function getCookie(name: string) {
  try {
    const match = document.cookie.match(new RegExp('(?:^|; )' + name.replace(/([.$?*|{}()\[\]\\\/\+^])/g, '\\$1') + '=([^;]*)'))
    const val = match ? decodeURIComponent(match[1]) : ''
    // console.log(`[API] getCookie ${name}:`, val ? 'found' : 'empty') // Commented out to avoid spam
    return val
  } catch (e) {
    console.error('[API] getCookie error:', e)
    return ''
  }
}

export const api = axios.create({ withCredentials: true })

api.interceptors.request.use((config) => {
  const token = getCookie('access_token')
  if (token && token.trim() !== '') {
    config.headers.Authorization = `Bearer ${token}`
  }
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
      console.log('[API] 401 detected, trying refresh')
      try {
        const { data } = await axios.post('/api/auth/refresh', {}, { withCredentials: true })
        if (data?.access_token) {
          console.log('[API] Refresh successful')
          document.cookie = `access_token=${encodeURIComponent(data.access_token)}; Path=/; Max-Age=${60 * 15}`
          return api.request(error.config)
        }
      } catch (_) {
        console.error('[API] Refresh failed')
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

  updateVisibility: (id: string, isPublic: boolean) =>
    api.patch(`/api/playlists/${id}/visibility`, { is_public: isPublic }),
}

// Community API functions
export const communityApi = {
  getPublicCrates: (params?: { limit?: number; offset?: number }) =>
    api.get('/api/community/crates', { params }),
}

// Backwards compatibility
export const playlistsApi = cratesApi

// Unsorted tracks
export type UnsortedParams = { limit?: number; offset?: number; q?: string }
export const tracksApi = {
  getUnsorted: (params?: UnsortedParams) =>
    api.get('/api/tracks', { params: { ...(params || {}), playlist_id: 'unsorted' } }),

  download: (id: string, filename: string) => {
    // Create a temporary link to trigger download
    // We use the API endpoint directly. Since we use cookies for auth, this works.
    const link = document.createElement('a')
    link.href = `/api/tracks/${id}/download`
    link.download = filename
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
  },
}

// SoundCloud sync API
export type SoundCloudConfig = {
  configured: boolean
  enabled: boolean
  owner_user_id?: string
  playlist_id?: string
  last_sync_at?: string
  masked_token?: string
}

export type SoundCloudSyncHistory = {
  id: string
  started_at: string
  completed_at?: string
  tracks_added: number
  tracks_skipped: number
  error_message?: string
}

export const soundcloudApi = {
  getConfig: () =>
    api.get<SoundCloudConfig>('/api/soundcloud/config'),

  updateConfig: (data: { oauth_token: string; owner_user_id: string; enabled: boolean }) =>
    api.put('/api/soundcloud/config', data),

  triggerSync: () =>
    api.post('/api/soundcloud/sync'),

  getHistory: () =>
    api.get<{ history: SoundCloudSyncHistory[] }>('/api/soundcloud/history'),
}

// Spotify sync API
export type SpotifyConfig = {
  configured: boolean
  enabled: boolean
  owner_user_id?: string
  liked_songs_playlist_id?: string
  last_sync_at?: string
  client_id?: string
}

export type SpotifySyncHistory = {
  id: string
  started_at: string
  completed_at?: string
  tracks_added: number
  tracks_skipped: number
  error_message?: string
}

export const spotifyApi = {
  getConfig: () =>
    api.get<SpotifyConfig>('/api/spotify/config'),

  updateConfig: (data: { enabled: boolean }) =>
    api.put('/api/spotify/config', data),

  exchangeToken: (data: { code: string; code_verifier: string; redirect_uri: string }) =>
    api.post('/api/spotify/token', data),

  disconnect: () =>
    api.post('/api/spotify/disconnect'),

  triggerSync: () =>
    api.post('/api/spotify/sync'),

  getHistory: () =>
    api.get<{ history: SpotifySyncHistory[] }>('/api/spotify/history'),

  getPlaylists: () =>
    api.get<{ playlists: Array<{ id: string; name: string; tracks: { total: number } }> }>('/api/spotify/playlists'),

  getSyncedPlaylists: () =>
    api.get<{ playlists: Array<{ id: string; spotify_playlist_id: string; local_playlist_id: string; name: string; enabled: boolean }> }>('/api/spotify/synced-playlists'),
}