import axios from 'axios'
import { useEffect, useState } from 'react'
import { Users, HardDrive, Disc } from 'lucide-react'

type User = {
  id: string
  email: string
  role: string
  storage_bytes?: number
}

export function AdminPage() {
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)

  async function fetchData() {
    setLoading(true)
    try {
      const [usersRes] = await Promise.all([
        axios.get('/api/users'),
      ])
      const usersData = usersRes.data?.users || usersRes.data || []
      setUsers(Array.isArray(usersData) ? usersData : [])
    } catch (error) {
      console.error('Failed to fetch admin data:', error)
      setUsers([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [])

  function formatStorage(bytes?: number): string {
    if (!bytes || bytes === 0) return '0 B'
    const gb = bytes / (1024 * 1024 * 1024)
    if (gb >= 1) return `${gb.toFixed(2)} GB`
    const mb = bytes / (1024 * 1024)
    if (mb >= 1) return `${mb.toFixed(2)} MB`
    const kb = bytes / 1024
    if (kb >= 1) return `${kb.toFixed(2)} KB`
    return `${bytes} B`
  }

  const totalStorage = users.reduce((acc, u) => acc + (u.storage_bytes || 0), 0)

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-display font-bold text-crate-cream">Admin</h1>
        <p className="text-crate-muted mt-1">Manage users and monitor system usage</p>
      </div>

      {/* Stats cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div className="card p-5">
          <div className="flex items-center gap-4">
            <div className="p-3 rounded-xl bg-crate-cyan/10">
              <Users size={24} className="text-crate-cyan" />
            </div>
            <div>
              <div className="text-sm text-crate-muted">Total Users</div>
              <div className="text-2xl font-display font-bold text-crate-cream">
                {loading ? '—' : users.length}
              </div>
            </div>
          </div>
        </div>

        <div className="card p-5">
          <div className="flex items-center gap-4">
            <div className="p-3 rounded-xl bg-crate-amber/10">
              <HardDrive size={24} className="text-crate-amber" />
            </div>
            <div>
              <div className="text-sm text-crate-muted">Total Storage</div>
              <div className="text-2xl font-display font-bold text-crate-cream">
                {loading ? '—' : formatStorage(totalStorage)}
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Users table */}
      <div className="card overflow-hidden">
        <div className="p-5 border-b border-crate-border">
          <h2 className="text-lg font-display font-semibold text-crate-cream">Users</h2>
        </div>

        {loading ? (
          <div className="flex items-center justify-center gap-3 py-12">
            <Disc className="text-crate-amber vinyl-spinning" size={24} />
            <span className="text-crate-muted">Loading users...</span>
          </div>
        ) : users.length === 0 ? (
          <div className="text-center py-12">
            <Users size={32} className="mx-auto text-crate-subtle mb-3" />
            <p className="text-crate-muted">No users found</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-crate-border bg-crate-elevated/50">
                  <th className="text-left py-3 px-5 text-xs uppercase tracking-wider text-crate-subtle font-medium">
                    Email
                  </th>
                  <th className="text-left py-3 px-5 text-xs uppercase tracking-wider text-crate-subtle font-medium">
                    Role
                  </th>
                  <th className="text-right py-3 px-5 text-xs uppercase tracking-wider text-crate-subtle font-medium">
                    Storage
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-crate-border/50">
                {users.map((u, idx) => (
                  <tr
                    key={u.id}
                    className="stagger-item hover:bg-crate-elevated/30 transition-colors"
                    style={{ animationDelay: `${idx * 30}ms` }}
                  >
                    <td className="py-4 px-5 text-crate-cream">{u.email}</td>
                    <td className="py-4 px-5">
                      <span className={`inline-flex px-2.5 py-1 rounded-lg text-xs font-medium ${u.role === 'admin'
                        ? 'bg-crate-amber/10 text-crate-amber'
                        : 'bg-crate-elevated text-crate-muted'
                        }`}>
                        {u.role}
                      </span>
                    </td>
                    <td className="py-4 px-5 text-right text-crate-muted tabular-nums">
                      {formatStorage(u.storage_bytes)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}
