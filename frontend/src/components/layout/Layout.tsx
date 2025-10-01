import { NavLink, Outlet, Link, useLocation } from 'react-router-dom'
import { Library, LogOut, Settings, UploadCloud, Folder } from 'lucide-react'
import { useAuth } from '../../state/auth'
import { PlayerBar } from '../player/PlayerBar'
import { useEffect, useState } from 'react'
import { cratesApi, normalizeCrateList } from '../../lib/api'
import type { CrateList } from '../../types/crates'

export function Layout() {
  const { user, logout } = useAuth()
  const [crates, setCrates] = useState<CrateList>({ crates: [], total: 0, limit: 20, offset: 0, has_next: false })
  const location = useLocation()
  const search = new URLSearchParams(location.search)
  const selectedCrateId = search.get('crate') || search.get('playlist')

  useEffect(() => {
    let mounted = true
    const load = () =>
      cratesApi
        .list()
        .then(({ data }) => {
          if (!mounted) return
          setCrates(normalizeCrateList(data))
        })
        .catch(() => {
          if (!mounted) return
          setCrates({ crates: [], total: 0, limit: 20, offset: 0, has_next: false })
        })

    load()
    const handler = () => load()
    window.addEventListener('crates:updated', handler)
    window.addEventListener('playlists:updated', handler)
    return () => {
      mounted = false
      window.removeEventListener('crates:updated', handler)
      window.removeEventListener('playlists:updated', handler)
    }
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
                  isActive && !selectedCrateId ? 'bg-[#2A2A2A] text-white' : 'text-[#C1C1C1] hover:text-white hover:bg-[#202020]'
                }`
              }
            >
              <span className="text-[#1DB954]"><Library size={18} /></span>
              Library
            </NavLink>
            <NavItem to="/upload" label="Upload" icon={<UploadCloud size={18} />} />
            {user?.role === 'admin' && (
              <NavItem to="/admin" label="Admin" icon={<Settings size={18} />} />
            )}
            <div className="mt-4 text-xs text-[#A1A1A1] px-3">Crates</div>
            <div className="space-y-1">
              {crates.crates.map((p) => (
                <Link
                  key={p.id}
                  to={`/?crate=${p.id}`}
                  className={`flex items-center gap-2 rounded-lg px-3 py-2 text-sm transition-colors ${selectedCrateId === p.id ? 'bg-[#2A2A2A] text-white' : 'text-[#C1C1C1] hover:text-white hover:bg-[#202020]'}`}
                >
                  <span className="text-[#1DB954]"><Folder size={16} /></span>
                  <span className="truncate">{p.name}</span>
                </Link>
              ))}
              <Link
                to="/?crate=unsorted"
                className={`flex items-center gap-2 rounded-lg px-3 py-2 text-sm transition-colors ${selectedCrateId === 'unsorted' ? 'bg-[#2A2A2A] text-white' : 'text-[#C1C1C1] hover:text-white hover:bg-[#202020]'}`}
              >
                <span className="text-[#1DB954]"><Folder size={16} /></span>
                Unsorted
              </Link>
              <Link to="/crates" className="flex items-center gap-2 rounded-lg px-3 py-2 text-sm text-[#A1A1A1] hover:text-white hover:bg-[#202020]">
                <span className="text-[#1DB954]"><Folder size={16} /></span>
                Manage Crates
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

