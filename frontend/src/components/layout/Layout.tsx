import { NavLink, Outlet } from 'react-router-dom'
import { Library, LogOut, PlusCircle, Settings, UserPlus, Menu, X } from 'lucide-react'
import { useAuth } from '../../state/auth'
import { PlayerBar } from '../player/PlayerBar'
import { useState } from 'react'

export function Layout() {
  const { user, logout } = useAuth()
  const [mobileOpen, setMobileOpen] = useState(false)
  return (
    <div className="min-h-screen grid grid-rows-[1fr_auto]">
      <div className="grid grid-cols-1 lg:grid-cols-[220px_1fr] gap-6 p-3 sm:p-6">
        {/* Desktop sidebar */}
        <aside className="card p-4 hidden lg:block">
          <SidebarContent user={user} logout={logout} />
        </aside>
        <main className="space-y-6">
          {/* Mobile header */}
          <div className="lg:hidden flex items-center justify-between">
            <div className="text-xl font-extrabold tracking-tight">CrateDrop</div>
            <button className="btn" onClick={() => setMobileOpen(true)} aria-label="Open menu">
              <Menu size={18} />
            </button>
          </div>
          <Outlet />
        </main>
      </div>

      {/* Mobile drawer */}
      {(
        <>
          <div
            className={`fixed inset-0 z-40 bg-black/60 transition-opacity duration-200 lg:hidden ${mobileOpen ? 'opacity-100 pointer-events-auto' : 'opacity-0 pointer-events-none'}`}
            onClick={() => setMobileOpen(false)}
          />
          <div
            className={`fixed inset-y-0 left-0 z-50 w-64 bg-[#181818] border-r border-[#2A2A2A] p-4 transform transition-transform duration-200 lg:hidden ${mobileOpen ? 'translate-x-0' : '-translate-x-full'}`}
          >
            <div className="mb-6 flex items-center justify-between">
              <div className="text-xl font-extrabold tracking-tight">CrateDrop</div>
              <button className="btn" onClick={() => setMobileOpen(false)} aria-label="Close menu">
                <X size={18} />
              </button>
            </div>
            <SidebarContent user={user} logout={logout} onNavigate={() => setMobileOpen(false)} />
          </div>
        </>
      )}

      <PlayerBar />
    </div>
  )
}

function SidebarContent({ user, logout, onNavigate }: { user: any; logout: () => void; onNavigate?: () => void }) {
  return (
    <>
      <nav className="space-y-1">
        <NavItem to="/" label="Library" icon={<Library size={18} />} onClick={onNavigate} />
        {user?.role === 'admin' && (
          <NavItem to="/admin" label="Admin" icon={<Settings size={18} />} onClick={onNavigate} />
        )}
      </nav>
      <div className="mt-8 text-xs text-[#A1A1A1]">Signed in as</div>
      <div className="flex items-center justify-between mt-1">
        <div className="text-sm">{user?.email}</div>
        <button
          className="btn btn-primary"
          onClick={() => {
            logout()
            onNavigate?.()
          }}
          title="Logout"
        >
          <LogOut size={16} />
        </button>
      </div>
    </>
  )
}

function NavItem({ to, label, icon, onClick }: { to: string; label: string; icon: React.ReactNode; onClick?: () => void }) {
  return (
    <NavLink
      to={to}
      end
      className={({ isActive }) =>
        `flex items-center gap-2 rounded-lg px-3 py-2 text-sm transition-colors ${
          isActive ? 'bg-[#2A2A2A] text-white' : 'text-[#C1C1C1] hover:text-white hover:bg-[#202020]'
        }`
      }
      onClick={onClick}
    >
      <span className="text-[#1DB954]">{icon}</span>
      {label}
    </NavLink>
  )
}

