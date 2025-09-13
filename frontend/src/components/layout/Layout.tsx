import { NavLink, Outlet, Link, useLocation } from 'react-router-dom'
import { Library, LogOut, Settings, Music, Folder } from 'lucide-react'
import { useAuth } from '../../state/auth'
import { PlayerBar } from '../player/PlayerBar'
import { useEffect, useState } from 'react'
import { playlistsApi } from '../../lib/api'
import type { Playlist, PlaylistList } from '../../types/playlists'

export function Layout() {
  const { user, logout } = useAuth()
  const [playlists, setPlaylists] = useState<PlaylistList>({ playlists: [], total: 0, limit: 20, offset: 0, has_next: false })
  const location = useLocation()
  const selectedPlaylistId = new URLSearchParams(location.search).get('playlist')

  useEffect(() => {
    let mounted = true
    playlistsApi.list().then(({ data }) => {
      if (!mounted) return
      const safe = data && Array.isArray(data.playlists) ? data : { playlists: [], total: 0, limit: 20, offset: 0, has_next: false }
      setPlaylists(safe)
    }).catch(() => {})
    return () => { mounted = false }
  }, [location.pathname])
  return (
    <div className="min-h-screen grid grid-rows-[1fr_auto]">
      <div className="grid grid-cols-[220px_1fr] gap-6 p-6">
        <aside className="card p-4">
          <div className="mb-6 flex items-center justify-between">
            <div className="text-xl font-extrabold tracking-tight">CrateDrop</div>
          </div>
          <nav className="space-y-1">
            <NavLink
              to="/"
              end
              className={({ isActive }) =>
                `flex items-center gap-2 rounded-lg px-3 py-2 text-sm transition-colors ${
                  isActive && !selectedPlaylistId ? 'bg-[#2A2A2A] text-white' : 'text-[#C1C1C1] hover:text-white hover:bg-[#202020]'
                }`
              }
            >
              <span className="text-[#1DB954]"><Library size={18} /></span>
              Library
            </NavLink>
            {user?.role === 'admin' && (
              <NavItem to="/admin" label="Admin" icon={<Settings size={18} />} />
            )}
            <div className="mt-4 text-xs text-[#A1A1A1] px-3">Playlists</div>
            <div className="space-y-1">
              {playlists.playlists.map((p) => (
                <Link
                  key={p.id}
                  to={`/?playlist=${p.id}`}
                  className={`flex items-center gap-2 rounded-lg px-3 py-2 text-sm transition-colors ${selectedPlaylistId === p.id ? 'bg-[#2A2A2A] text-white' : 'text-[#C1C1C1] hover:text-white hover:bg-[#202020]'}`}
                >
                  <span className="text-[#1DB954]"><Folder size={16} /></span>
                  <span className="truncate">{p.name}</span>
                </Link>
              ))}
              <Link
                to={`/?playlist=unsorted`}
                className={`flex items-center gap-2 rounded-lg px-3 py-2 text-sm transition-colors ${selectedPlaylistId === 'unsorted' ? 'bg-[#2A2A2A] text-white' : 'text-[#C1C1C1] hover:text-white hover:bg-[#202020]'}`}
              >
                <span className="text-[#1DB954]"><Folder size={16} /></span>
                Unsorted
              </Link>
              <Link to="/playlists?create=1" className="flex items-center gap-2 rounded-lg px-3 py-2 text-sm text-[#A1A1A1] hover:text-white hover:bg-[#202020]">
                <span className="text-[#1DB954]"><Music size={16} /></span>
                Manage Playlists
              </Link>
            </div>
          </nav>
          <div className="mt-8 text-xs text-[#A1A1A1]">Signed in as</div>
          <div className="flex items-center justify-between mt-1">
            <div className="text-sm">{user?.email}</div>
            <button className="btn btn-primary" onClick={logout} title="Logout">
              <LogOut size={16} />
            </button>
          </div>
        </aside>
        <main className="space-y-6">
          <Outlet />
        </main>
      </div>
      <PlayerBar />
    </div>
  )
}

function NavItem({ to, label, icon }: { to: string; label: string; icon: React.ReactNode }) {
  return (
    <NavLink
      to={to}
      end
      className={({ isActive }) =>
        `flex items-center gap-2 rounded-lg px-3 py-2 text-sm transition-colors ${
          isActive ? 'bg-[#2A2A2A] text-white' : 'text-[#C1C1C1] hover:text-white hover:bg-[#202020]'
        }`
      }
    >
      <span className="text-[#1DB954]">{icon}</span>
      {label}
    </NavLink>
  )
}

