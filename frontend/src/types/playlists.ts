// Playlist types for TypeScript
export type Playlist = {
  id: string
  owner_user_id: string
  name: string
  description?: string
  is_default: boolean
  created_at: string
  updated_at: string
}

export type PlaylistWithTracks = {
  playlist: Playlist
  tracks: Track[]
  total: number
  limit: number
  offset: number
  has_next: boolean
}

export type PlaylistList = {
  playlists: Playlist[]
  total: number
  limit: number
  offset: number
  has_next: boolean
}

// API request/response types
export type CreatePlaylistRequest = {
  name: string
  description?: string
}

export type UpdatePlaylistRequest = {
  name: string
  description?: string
}

export type AddTracksToPlaylistRequest = {
  track_ids: string[]
}

export type RemoveTracksFromPlaylistRequest = {
  track_ids: string[]
}

// Extended Track type with playlist info
export type Track = {
  id: string
  title?: string
  artist?: string
  album?: string
  original_filename: string
  owner_user_id?: string
  content_type?: string
  size_bytes?: number
  duration_seconds?: number
  genre?: string
  year?: number
  sample_rate?: number
  bitrate?: number
  file_path?: string
  created_at?: string
  updated_at?: string
}

export type TrackList = {
  tracks: Track[]
  total: number
  limit: number
  offset: number
  has_next: boolean
}
