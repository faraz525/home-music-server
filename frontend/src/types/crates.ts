export type Crate = {
  id: string
  owner_user_id: string
  name: string
  description?: string
  is_default: boolean
  is_public: boolean
  created_at: string
  updated_at: string
}

export type PlaylistWithOwner = Crate & {
  owner_email: string
}

export type CrateWithTracks = {
  crate: Crate
  tracks: Track[]
  total: number
  limit: number
  offset: number
  has_next: boolean
}

export type CrateList = {
  crates: Crate[]
  total: number
  limit: number
  offset: number
  has_next: boolean
}

// API request/response types
export type CreateCrateRequest = {
  name: string
  description?: string
  is_public?: boolean
}

export type UpdateCrateRequest = {
  name: string
  description?: string
  is_public?: boolean
}

export type AddTracksToCrateRequest = {
  track_ids: string[]
}

export type RemoveTracksFromCrateRequest = {
  track_ids: string[]
}

// Extended Track type with crate info
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
