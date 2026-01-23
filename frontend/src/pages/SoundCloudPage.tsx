import { useEffect, useState } from 'react'
import { Cloud, RefreshCw, Check, X, Disc, History, AlertCircle } from 'lucide-react'
import { soundcloudApi, SoundCloudConfig, SoundCloudSyncHistory } from '../lib/api'
import { useAuth } from '../state/auth'
import { toast } from 'sonner'

export function SoundCloudPage() {
  const { user } = useAuth()
  const [config, setConfig] = useState<SoundCloudConfig | null>(null)
  const [history, setHistory] = useState<SoundCloudSyncHistory[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [syncing, setSyncing] = useState(false)

  const [oauthToken, setOauthToken] = useState('')
  const [enabled, setEnabled] = useState(false)

  async function fetchData() {
    setLoading(true)
    try {
      const [configRes, historyRes] = await Promise.all([
        soundcloudApi.getConfig(),
        soundcloudApi.getHistory(),
      ])
      setConfig(configRes.data)
      setHistory(historyRes.data?.history || [])
      setEnabled(configRes.data?.enabled || false)
    } catch {
      toast.error('Failed to load SoundCloud configuration')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [])

  async function handleSave() {
    if (!user?.id) return

    setSaving(true)
    try {
      await soundcloudApi.updateConfig({
        oauth_token: oauthToken || '',
        owner_user_id: user.id,
        enabled,
      })
      toast.success('SoundCloud configuration saved')
      setOauthToken('')
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
      await soundcloudApi.triggerSync()
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
          <h1 className="text-3xl font-display font-bold text-crate-cream">SoundCloud Sync</h1>
          <p className="text-crate-muted mt-1">Admin access required</p>
        </div>
        <div className="card p-8 text-center">
          <AlertCircle size={48} className="mx-auto text-crate-subtle mb-4" />
          <p className="text-crate-muted">Only admins can configure SoundCloud sync.</p>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-display font-bold text-crate-cream">SoundCloud Sync</h1>
        <p className="text-crate-muted mt-1">Auto-sync your SoundCloud likes to CrateDrop</p>
      </div>

      {loading ? (
        <div className="flex items-center justify-center gap-3 py-12">
          <Disc className="text-crate-amber vinyl-spinning" size={24} />
          <span className="text-crate-muted">Loading...</span>
        </div>
      ) : (
        <>
          {/* Status Card */}
          <div className="card p-5">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-4">
                <div className="p-3 rounded-xl bg-orange-500/10">
                  <Cloud size={24} className="text-orange-500" />
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
                            <Check size={16} /> Configured (auto-sync off)
                          </span>
                        )}
                      </>
                    ) : (
                      <span className="text-crate-muted">Not configured</span>
                    )}
                  </div>
                </div>
              </div>
              {config?.configured && (
                <button
                  onClick={handleSync}
                  disabled={syncing}
                  className="btn-secondary flex items-center gap-2"
                >
                  <RefreshCw size={16} className={syncing ? 'animate-spin' : ''} />
                  {syncing ? 'Syncing...' : 'Sync Now'}
                </button>
              )}
            </div>
            {config?.last_sync_at && (
              <div className="mt-4 pt-4 border-t border-crate-border">
                <span className="text-sm text-crate-muted">
                  Last sync: {formatDate(config.last_sync_at)}
                </span>
              </div>
            )}
          </div>

          {/* Configuration Card */}
          <div className="card overflow-hidden">
            <div className="p-5 border-b border-crate-border">
              <h2 className="text-lg font-display font-semibold text-crate-cream">Configuration</h2>
            </div>
            <div className="p-5 space-y-4">
              <div>
                <label className="block text-sm font-medium text-crate-cream mb-2">
                  OAuth Token
                </label>
                <input
                  type="text"
                  value={oauthToken}
                  onChange={(e) => setOauthToken(e.target.value)}
                  placeholder={config?.masked_token || (config?.configured ? '••••••••••••••••' : 'Paste your SoundCloud OAuth token')}
                  className="input w-full"
                />
                <p className="text-xs text-crate-muted mt-2">
                  To get your token: Open SoundCloud in browser → DevTools (F12) → Application → Cookies → soundcloud.com → oauth_token
                </p>
              </div>

              <div className="flex items-center gap-3">
                <button
                  onClick={() => setEnabled(!enabled)}
                  className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                    enabled ? 'bg-crate-amber' : 'bg-crate-elevated'
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
                  disabled={saving || (!oauthToken && !config?.configured)}
                  className="btn-primary"
                >
                  {saving ? 'Saving...' : 'Save Configuration'}
                </button>
              </div>
            </div>
          </div>

          {/* Sync History */}
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

          {/* How It Works */}
          <div className="card p-5">
            <h3 className="text-sm font-medium text-crate-cream mb-3">How It Works</h3>
            <ul className="text-sm text-crate-muted space-y-2">
              <li>1. Paste your SoundCloud OAuth token above</li>
              <li>2. Enable automatic sync to run daily</li>
              <li>3. New liked tracks will be downloaded as MP3s</li>
              <li>4. Tracks are added to a "SoundCloud Likes" crate</li>
              <li>5. Already-synced tracks are skipped automatically</li>
            </ul>
          </div>
        </>
      )}
    </div>
  )
}
