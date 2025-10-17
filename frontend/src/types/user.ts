// User types for TypeScript
export type User = {
  id: string
  email: string
  username?: string
  role: string
  created_at: string
  updated_at: string
}

export type UserSearchResult = {
  id: string
  username: string
  email?: string // Only shown to admins
}

export type UserSearchResponse = {
  users: UserSearchResult[]
  total: number
  limit: number
  offset: number
  has_next: boolean
}

export type UpdateUsernameRequest = {
  username: string
}

// Trade types
export type TradeRequest = {
  crate_id: string
  track_id: string
  offer_track_ids: string[]
}

export type TradeTransaction = {
  id: string
  requester_user_id: string
  owner_user_id: string
  crate_id: string
  requested_track_id: string
  given_track_ids: string
  trade_ratio: string
  created_at: string
}

export type TradeHistoryResponse = {
  trades: TradeTransaction[]
  total: number
  limit: number
  offset: number
  has_next: boolean
}
