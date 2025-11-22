import axios from 'axios'
import { useEffect, useState } from 'react'

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
      // Ensure we always set an array, even if the response is malformed
      const usersData = usersRes.data?.users || usersRes.data || []
      setUsers(Array.isArray(usersData) ? usersData : [])
    } catch (error) {
      console.error('Failed to fetch admin data:', error)
      // Set empty arrays on error to prevent map errors
      setUsers([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [])



  // Helper function to format storage bytes to human-readable format
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

  return (
    <div className="space-y-6">
      <div className="text-xl font-bold">Admin</div>
      <div className="card p-4">
        <div className="font-semibold mb-2">Users</div>
        {loading ? (
          <div className="text-sm text-[#A1A1A1]">Loading…</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-[#2A2A2A]">
                  <th className="text-left py-2 px-2">Email</th>
                  <th className="text-left py-2 px-2">Role</th>
                  <th className="text-right py-2 px-2">Storage</th>
                </tr>
              </thead>
              <tbody>
                {Array.isArray(users) && users.map((u) => (
                  <tr key={u.id} className="border-b border-[#1A1A1A]">
                    <td className="py-2 px-2">{u.email}</td>
                    <td className="py-2 px-2 text-[#A1A1A1]">{u.role}</td>
                    <td className="py-2 px-2 text-right text-[#A1A1A1]">{formatStorage(u.storage_bytes)}</td>
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

