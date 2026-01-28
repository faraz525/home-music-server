import { useEffect, useState, useCallback } from 'react'
import { Music2, RefreshCw, Check, X, Disc, History, AlertCircle, Link2, Unlink } from 'lucide-react'
import { spotifyApi, SpotifyConfig, SpotifySyncHistory } from '../lib/api'
import { useAuth } from '../state/auth'
import { toast } from 'sonner'
import { useSearchParams } from 'react-router-dom'

// PKCE helpers
function generateRandomString(length: number): string {
  const possible = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
  const values = crypto.getRandomValues(new Uint8Array(length))
  return values.reduce((acc, x) => acc + possible[x % possible.length], '')
}

async function sha256(plain: string): Promise<ArrayBuffer> {
  const encoder = new TextEncoder()
  const data = encoder.encode(plain)
  return window.crypto.subtle.digest('SHA-256', data)
}

function base64encode(input: ArrayBuffer): string {
  return btoa(String.fromCharCode(...new Uint8Array(input)))
    .replace(/=/g, '')
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
}

async function generateCodeChallenge(codeVerifier: string): Promise<string> {
  const hashed = await sha256(codeVerifier)
  return base64encode(hashed)
}

const SPOTIFY_AUTH_URL = 'https://accounts.spotify.com/authorize'
const SPOTIFY_SCOPES = ['user-library-read', 'playlist-read-private', 'playlist-read-collaborative']

export function SpotifyPage() {
  const { user } = useAuth()
  const [searchParams, setSearchParams] = useSearchParams()
  const [config, setConfig] = useState<SpotifyConfig | null>(null)
  const [history, setHistory] = useState<SpotifySyncHistory[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [syncing, setSyncing] = useState(false)
  const [connecting, setConnecting] = useState(false)
  const [enabled, setEnabled] = useState(false)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const [configRes, historyRes] = await Promise.all([
        spotifyApi.getConfig(),
        spotifyApi.getHistory(),
      ])
      setConfig(configRes.data)
      setHistory(historyRes.data?.history || [])
      setEnabled(configRes.data?.enabled || false)
    } catch {
      toast.error('Failed to load Spotify configuration')
    } finally {
      setLoading(false)
    }
  }, [])

  // Handle OAuth callback
  useEffect(() => {
    const code = searchParams.get('code')
    const state = searchParams.get('state')
    const error = searchParams.get('error')

    if (error) {
      toast.error(`Spotify authorization failed: ${error}`)
      searchParams.delete('error')
      setSearchParams(searchParams)
      return
    }

    if (code && state) {
      const storedState = sessionStorage.getItem('spotify_auth_state')
      const codeVerifier = sessionStorage.getItem('spotify_code_verifier')

      if (state !== storedState) {
        toast.error('State mismatch. Please try connecting again.')
      } else if (!codeVerifier) {
        toast.error('Code verifier not found. Please try connecting again.')
      } else {
        // Exchange code for token
        setConnecting(true)
        spotifyApi.exchangeToken({
          code,
          code_verifier: codeVerifier,
          redirect_uri: window.location.origin + '/spotify',
        })
          .then(() => {
            toast.success('Connected to Spotify!')
            sessionStorage.removeItem('spotify_auth_state')
            sessionStorage.removeItem('spotify_code_verifier')
            fetchData()
          })
          .catch(() => {
            toast.error('Failed to connect to Spotify')
          })
          .finally(() => {
            setConnecting(false)
          })
      }

      // Clean up URL
      searchParams.delete('code')
      searchParams.delete('state')
      setSearchParams(searchParams)
    }
  }, [searchParams, setSearchParams, fetchData])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  async function handleConnect() {
    if (!config?.client_id) {
      toast.error('Spotify client ID not configured on server')
      return
    }

    const codeVerifier = generateRandomString(64)
    const state = generateRandomString(16)
    const codeChallenge = await generateCodeChallenge(codeVerifier)

    // Store for later verification
    sessionStorage.setItem('spotify_code_verifier', codeVerifier)
    sessionStorage.setItem('spotify_auth_state', state)

    const params = new URLSearchParams({
      response_type: 'code',
      client_id: config.client_id,
      scope: SPOTIFY_SCOPES.join(' '),
      redirect_uri: window.location.origin + '/spotify',
      state: state,
      code_challenge_method: 'S256',
      code_challenge: codeChallenge,
    })

    window.location.href = `${SPOTIFY_AUTH_URL}?${params.toString()}`
  }

  async function handleDisconnect() {
    try {
      await spotifyApi.disconnect()
      toast.success('Disconnected from Spotify')
      fetchData()
    } catch {
      toast.error('Failed to disconnect')
    }
  }

  async function handleSave() {
    setSaving(true)
    try {
      await spotifyApi.updateConfig({ enabled })
      toast.success('Spotify configuration saved')
      fetchData()
    } catch {
      toast.error('Failed to save configuration')
    } finally {
      setSaving(false)
    }
  }

  async function handleSync() {
    setSyncing(true)
    try {
      await spotifyApi.triggerSync()
      toast.success('Sync started! Check back in a few minutes.')
      setTimeout(fetchData, 5000)
    } catch {
      toast.error('Failed to trigger sync')
    } finally {
      setSyncing(false)
    }
  }

  function formatDate(dateString?: string) {
    if (!dateString) return 'Never'
    return new Date(dateString).toLocaleString()
  }

  if (user?.role !== 'admin') {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-display font-bold text-crate-cream">Spotify Sync</h1>
          <p className="text-crate-muted mt-1">Admin access required</p>
        </div>
        <div className="card p-8 text-center">
          <AlertCircle size={48} className="mx-auto text-crate-subtle mb-4" />
          <p className="text-crate-muted">Only admins can configure Spotify sync.</p>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-display font-bold text-crate-cream">Spotify Sync</h1>
        <p className="text-crate-muted mt-1">Auto-sync your Spotify liked songs to CrateDrop</p>
      </div>

      {loading || connecting ? (
        <div className="flex items-center justify-center gap-3 py-12">
          <Disc className="text-crate-amber vinyl-spinning" size={24} />
          <span className="text-crate-muted">{connecting ? 'Connecting to Spotify...' : 'Loading...'}</span>
        </div>
      ) : (
        <>
          {/* Status Card */}
          <div className="card p-5">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-4">
                <div className="p-3 rounded-xl bg-green-500/10">
                  <Music2 size={24} className="text-green-500" />
                </div>
                <div>
                  <div className="text-sm text-crate-muted">Status</div>
                  <div className="flex items-center gap-2">
                    {config?.configured ? (
                      <>
                        {config?.enabled ? (
                          <span className="flex items-center gap-1.5 text-green-500">
                            <Check size={16} /> Active (auto-sync on)
                          </span>
                        ) : (
                          <span className="flex items-center gap-1.5 text-crate-amber">
                            <Check size={16} /> Connected (auto-sync off)
                          </span>
                        )}
                      </>
                    ) : (
                      <span className="text-crate-muted">Not connected</span>
                    )}
                  </div>
                </div>
              </div>
              <div className="flex items-center gap-2">
                {config?.configured && (
                  <>
                    <button
                      onClick={handleDisconnect}
                      className="btn-ghost flex items-center gap-2 text-crate-muted hover:text-red-500"
                    >
                      <Unlink size={16} />
                      Disconnect
                    </button>
                    <button
                      onClick={handleSync}
                      disabled={syncing}
                      className="btn-secondary flex items-center gap-2"
                    >
                      <RefreshCw size={16} className={syncing ? 'animate-spin' : ''} />
                      {syncing ? 'Syncing...' : 'Sync Now'}
                    </button>
                  </>
                )}
              </div>
            </div>
            {config?.last_sync_at && (
              <div className="mt-4 pt-4 border-t border-crate-border">
                <span className="text-sm text-crate-muted">
                  Last sync: {formatDate(config.last_sync_at)}
                </span>
              </div>
            )}
          </div>

          {/* Connect or Configuration Card */}
          <div className="card overflow-hidden">
            <div className="p-5 border-b border-crate-border">
              <h2 className="text-lg font-display font-semibold text-crate-cream">
                {config?.configured ? 'Configuration' : 'Connect to Spotify'}
              </h2>
            </div>
            <div className="p-5 space-y-4">
              {!config?.configured ? (
                <div className="text-center py-6">
                  <Music2 size={48} className="mx-auto text-green-500 mb-4" />
                  <p className="text-crate-muted mb-6">
                    Connect your Spotify account to sync your liked songs automatically.
                  </p>
                  {!config?.client_id ? (
                    <div className="p-4 bg-crate-amber/10 border border-crate-amber/20 rounded-xl text-sm text-crate-amber">
                      <AlertCircle size={16} className="inline mr-2" />
                      SPOTIFY_CLIENT_ID environment variable not set on server
                    </div>
                  ) : (
                    <button
                      onClick={handleConnect}
                      className="btn-primary bg-green-600 hover:bg-green-700 flex items-center gap-2 mx-auto"
                    >
                      <Link2 size={16} />
                      Connect with Spotify
                    </button>
                  )}
                </div>
              ) : (
                <>
                  <div className="flex items-center gap-3">
                    <button
                      onClick={() => setEnabled(!enabled)}
                      className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                        enabled ? 'bg-green-500' : 'bg-crate-elevated'
                      }`}
                    >
                      <span
                        className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                          enabled ? 'translate-x-6' : 'translate-x-1'
                        }`}
                      />
                    </button>
                    <span className="text-sm text-crate-cream">
                      Enable automatic daily sync
                    </span>
                  </div>

                  <div className="pt-4">
                    <button
                      onClick={handleSave}
                      disabled={saving}
                      className="btn-primary"
                    >
                      {saving ? 'Saving...' : 'Save Configuration'}
                    </button>
                  </div>
                </>
              )}
            </div>
          </div>

          {/* Sync History */}
          {config?.configured && (
            <div className="card overflow-hidden">
              <div className="p-5 border-b border-crate-border flex items-center gap-2">
                <History size={18} className="text-crate-muted" />
                <h2 className="text-lg font-display font-semibold text-crate-cream">Sync History</h2>
              </div>

              {history.length === 0 ? (
                <div className="text-center py-12">
                  <History size={32} className="mx-auto text-crate-subtle mb-3" />
                  <p className="text-crate-muted">No sync history yet</p>
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full">
                    <thead>
                      <tr className="border-b border-crate-border bg-crate-elevated/50">
                        <th className="text-left py-3 px-5 text-xs uppercase tracking-wider text-crate-subtle font-medium">
                          Date
                        </th>
                        <th className="text-right py-3 px-5 text-xs uppercase tracking-wider text-crate-subtle font-medium">
                          Added
                        </th>
                        <th className="text-right py-3 px-5 text-xs uppercase tracking-wider text-crate-subtle font-medium">
                          Skipped
                        </th>
                        <th className="text-left py-3 px-5 text-xs uppercase tracking-wider text-crate-subtle font-medium">
                          Status
                        </th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-crate-border/50">
                      {history.map((h, idx) => (
                        <tr
                          key={h.id}
                          className="stagger-item hover:bg-crate-elevated/30 transition-colors"
                          style={{ animationDelay: `${idx * 30}ms` }}
                        >
                          <td className="py-4 px-5 text-crate-cream">
                            {formatDate(h.started_at)}
                          </td>
                          <td className="py-4 px-5 text-right text-green-500 tabular-nums">
                            +{h.tracks_added}
                          </td>
                          <td className="py-4 px-5 text-right text-crate-muted tabular-nums">
                            {h.tracks_skipped}
                          </td>
                          <td className="py-4 px-5">
                            {h.error_message ? (
                              <span className="text-red-500 text-sm" title={h.error_message}>
                                Failed
                              </span>
                            ) : h.completed_at ? (
                              <span className="text-green-500 text-sm">Success</span>
                            ) : (
                              <span className="text-crate-amber text-sm">Running</span>
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          )}

          {/* How It Works */}
          <div className="card p-5">
            <h3 className="text-sm font-medium text-crate-cream mb-3">How It Works</h3>
            <ul className="text-sm text-crate-muted space-y-2">
              <li>1. Connect your Spotify account using the button above</li>
              <li>2. Enable automatic sync to run daily</li>
              <li>3. Liked songs will be searched and downloaded as MP3s</li>
              <li>4. Tracks are added to a "Spotify Liked Songs" crate</li>
              <li>5. Already-synced tracks are skipped automatically</li>
            </ul>
          </div>
        </>
      )}
    </div>
  )
}
