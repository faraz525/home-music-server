import axios from 'axios'
import { FormEvent, useEffect, useState } from 'react'

type Invite = {
  code: string
  used_by?: string | null
  expires_at?: string | null
}

type User = {
  id: string
  email: string
  role: string
}

export function AdminPage() {
  const [invites, setInvites] = useState<Invite[]>([])
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)
  const [expiresAt, setExpiresAt] = useState('')

  async function fetchData() {
    setLoading(true)
    try {
      const [usersRes] = await Promise.all([
        axios.get('/api/users'),
      ])
      // Ensure we always set an array, even if the response is malformed
      const usersData = usersRes.data?.users || usersRes.data || []
      setUsers(Array.isArray(usersData) ? usersData : [])
      // invites endpoints not exposed in backend repo; placeholder for future
      setInvites([])
    } catch (error) {
      console.error('Failed to fetch admin data:', error)
      // Set empty arrays on error to prevent map errors
      setUsers([])
      setInvites([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [])

  async function createInvite(e: FormEvent) {
    e.preventDefault()
    try {
      // Placeholder; backend invite endpoints not implemented in provided code
      alert('Invite creation is not available in backend yet')
    } finally {
      // noop
    }
  }

  return (
    <div className="space-y-6">
      <div className="text-xl font-bold">Admin</div>
      <div className="grid md:grid-cols-2 gap-6">
        <div className="card p-4">
          <div className="font-semibold mb-2">Users</div>
          {loading ? (
            <div className="text-sm text-[#A1A1A1]">Loading…</div>
          ) : (
            <ul className="space-y-2">
              {Array.isArray(users) && users.map((u) => (
                <li key={u.id} className="flex items-center justify-between text-sm">
                  <span>{u.email}</span>
                  <span className="text-[#A1A1A1]">{u.role}</span>
                </li>
              ))}
            </ul>
          )}
        </div>
        <div className="card p-4">
          <div className="font-semibold mb-2">Invites</div>
          <form className="flex gap-2 mb-3" onSubmit={createInvite}>
            <input className="input flex-1" type="datetime-local" value={expiresAt} onChange={(e) => setExpiresAt(e.target.value)} />
            <button className="btn btn-primary">Create</button>
          </form>
          <ul className="space-y-2">
            {Array.isArray(invites) && invites.map((i) => (
              <li key={i.code} className="flex items-center justify-between text-sm">
                <span>{i.code}</span>
                <span className="text-[#A1A1A1]">{i.used_by ? 'used' : 'unused'}</span>
              </li>
            ))}
          </ul>
        </div>
      </div>
    </div>
  )
}

