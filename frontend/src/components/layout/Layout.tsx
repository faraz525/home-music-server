import { NavLink, Outlet } from 'react-router-dom'
import { Library, LogOut, PlusCircle, Settings, UserPlus } from 'lucide-react'
import { useAuth } from '../../state/auth'
import { PlayerBar } from '../player/PlayerBar'

export function Layout() {
  const { user, logout } = useAuth()
  return (
    <div className="min-h-screen grid grid-rows-[1fr_auto]">
      <div className="grid grid-cols-[220px_1fr] gap-6 p-6">
        <aside className="card p-4">
          <div className="mb-6 flex items-center justify-between">
            <div className="text-xl font-extrabold tracking-tight">CrateDrop</div>
          </div>
          <nav className="space-y-1">
            <NavItem to="/" label="Library" icon={<Library size={18} />} />
            {user?.role === 'admin' && (
              <NavItem to="/admin" label="Admin" icon={<Settings size={18} />} />
            )}
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

