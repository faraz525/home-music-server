import { NavLink, Outlet, Link, useLocation } from 'react-router-dom'
import { Library, LogOut, Settings, UploadCloud, Folder, Menu, X, FolderOpen, Globe, Disc } from 'lucide-react'
import { useAuth } from '../../state/auth'
import { PlayerBar } from '../player/PlayerBar'
import { useCallback, useEffect, useState } from 'react'
import { useToast } from '../../hooks/useToast'
import { useCrates, useAddTracksToCrate } from '../../hooks/useQueries'

export function Layout() {
  const { user, logout } = useAuth()
  const toast = useToast()
  const { data: crates } = useCrates()
  const addTracksMutation = useAddTracksToCrate()
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)
  const [dragOverCrateId, setDragOverCrateId] = useState<string | null>(null)
  const location = useLocation()
  const search = new URLSearchParams(location.search)
  const selectedCrateId = search.get('crate') || search.get('playlist')

  useEffect(() => {
    setMobileMenuOpen(false)
  }, [location])

  const handleDragOver = useCallback((e: React.DragEvent, crateId: string) => {
    e.preventDefault()
    e.stopPropagation()
    e.dataTransfer.dropEffect = 'copy'
    setDragOverCrateId(crateId)
  }, [])

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setDragOverCrateId(null)
  }, [])

  const handleDrop = useCallback(async (e: React.DragEvent, crateId: string) => {
    e.preventDefault()
    e.stopPropagation()
    setDragOverCrateId(null)

    try {
      const data = e.dataTransfer.getData('application/json')
      if (data) {
        const { trackIds } = JSON.parse(data)
        if (trackIds && Array.isArray(trackIds) && trackIds.length > 0) {
          await addTracksMutation.mutateAsync({ crateId, trackIds })
          const count = trackIds.length
          toast.success(`Added ${count} track${count > 1 ? 's' : ''} to crate`)
        }
      }
    } catch (error) {
      console.error('Failed to add tracks to crate:', error)
      toast.error('Failed to add tracks to crate')
    }
  }, [addTracksMutation, toast])

  return (
    <div className="min-h-screen grid grid-rows-[1fr_auto] overflow-visible bg-crate-black">
      {/* Mobile Header */}
      <div className="lg:hidden fixed top-0 left-0 right-0 bg-crate-surface/95 backdrop-blur-md border-b border-crate-border px-4 py-3 flex items-center justify-between z-40">
        <div className="flex items-center gap-2">
          <Disc className="text-crate-amber" size={24} />
          <span className="text-xl font-display font-bold text-crate-cream">CrateDrop</span>
        </div>
        <button
          onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
          className="hw-button p-2"
          aria-label={mobileMenuOpen ? 'Close menu' : 'Open menu'}
          aria-expanded={mobileMenuOpen}
        >
          {mobileMenuOpen ? <X size={20} /> : <Menu size={20} />}
        </button>
      </div>

      {/* Mobile Menu Overlay */}
      {mobileMenuOpen && (
        <div
          className="lg:hidden fixed inset-0 bg-crate-black/80 backdrop-blur-sm z-30"
          onClick={() => setMobileMenuOpen(false)}
        />
      )}

      {/* Mobile Sidebar */}
      <aside className={`lg:hidden fixed top-[57px] left-0 bottom-0 w-72 bg-crate-surface border-r border-crate-border p-5 overflow-y-auto z-40 transform transition-transform duration-200 ${mobileMenuOpen ? 'translate-x-0' : '-translate-x-full'}`}>
        <nav className="space-y-1">
          <NavLink
            to="/"
            end
            className={({ isActive }) =>
              `flex items-center gap-3 rounded-xl px-4 py-3 text-sm font-medium transition-all ${isActive && !selectedCrateId
                ? 'bg-crate-amber/10 text-crate-amber border border-crate-amber/20'
                : 'text-crate-muted hover:text-crate-cream hover:bg-crate-elevated'
              }`
            }
          >
            <Library size={18} />
            Library
          </NavLink>
          <NavItem to="/upload" label="Upload" icon={<UploadCloud size={18} />} />
          <NavItem to="/community" label="Community" icon={<Globe size={18} />} />
          {user?.role === 'admin' && (
            <NavItem to="/admin" label="Admin" icon={<Settings size={18} />} />
          )}

          <div className="pt-6 pb-2">
            <div className="text-xs font-medium text-crate-subtle uppercase tracking-wider px-4">
              Your Crates
            </div>
          </div>

          <div className="space-y-1">
            {crates?.crates.map((p) => (
              <Link
                key={p.id}
                to={`/?crate=${p.id}`}
                className={`flex items-center gap-3 rounded-xl px-4 py-2.5 text-sm transition-all ${selectedCrateId === p.id
                  ? 'bg-crate-amber/10 text-crate-amber border border-crate-amber/20'
                  : 'text-crate-muted hover:text-crate-cream hover:bg-crate-elevated'
                } ${dragOverCrateId === p.id && p.id !== 'unsorted'
                  ? 'ring-2 ring-crate-cyan bg-crate-cyan/10'
                  : ''
                }`}
                onDragOver={p.id !== 'unsorted' ? (e) => handleDragOver(e, p.id) : undefined}
                onDragLeave={p.id !== 'unsorted' ? handleDragLeave : undefined}
                onDrop={p.id !== 'unsorted' ? (e) => handleDrop(e, p.id) : undefined}
              >
                <Folder size={16} className={selectedCrateId === p.id ? 'text-crate-amber' : ''} />
                <span className="truncate">{p.name}</span>
              </Link>
            ))}
            <Link
              to="/crates"
              className="flex items-center gap-3 rounded-xl px-4 py-2.5 text-sm text-crate-subtle hover:text-crate-cream hover:bg-crate-elevated transition-all"
            >
              <FolderOpen size={16} />
              Manage Crates
            </Link>
          </div>
        </nav>

        <div className="mt-8 pt-6 border-t border-crate-border">
          <div className="text-xs text-crate-subtle mb-2">Signed in as</div>
          <div className="flex items-center justify-between">
            <div className="text-sm text-crate-cream truncate">{user?.email}</div>
            <button
              className="btn-ghost p-2 rounded-lg text-crate-muted hover:text-crate-danger"
              onClick={logout}
              title="Logout"
            >
              <LogOut size={16} />
            </button>
          </div>
        </div>
      </aside>

      <div className="grid grid-cols-1 lg:grid-cols-[260px_1fr] gap-6 p-4 sm:p-6 pt-20 lg:pt-6 overflow-visible">
        {/* Desktop Sidebar */}
        <aside className="card p-5 hidden lg:block h-fit sticky top-6">
          <div className="mb-8 flex items-center gap-3">
            <div className="p-2 rounded-xl bg-crate-amber/10">
              <Disc className="text-crate-amber" size={24} />
            </div>
            <span className="text-xl font-display font-bold text-crate-cream">CrateDrop</span>
          </div>

          <nav className="space-y-1">
            <NavLink
              to="/"
              end
              className={({ isActive }) =>
                `flex items-center gap-3 rounded-xl px-4 py-3 text-sm font-medium transition-all ${isActive && !selectedCrateId
                  ? 'bg-crate-amber/10 text-crate-amber border border-crate-amber/20'
                  : 'text-crate-muted hover:text-crate-cream hover:bg-crate-elevated'
                }`
              }
            >
              <Library size={18} />
              Library
            </NavLink>
            <NavItem to="/upload" label="Upload" icon={<UploadCloud size={18} />} />
            <NavItem to="/community" label="Community" icon={<Globe size={18} />} />
            {user?.role === 'admin' && (
              <NavItem to="/admin" label="Admin" icon={<Settings size={18} />} />
            )}

            <div className="pt-6 pb-2">
              <div className="text-xs font-medium text-crate-subtle uppercase tracking-wider px-4">
                Your Crates
              </div>
            </div>

            <div className="space-y-1 max-h-64 overflow-y-auto">
              {crates?.crates.map((p) => (
                <Link
                  key={p.id}
                  to={`/?crate=${p.id}`}
                  className={`flex items-center gap-3 rounded-xl px-4 py-2.5 text-sm transition-all ${selectedCrateId === p.id
                    ? 'bg-crate-amber/10 text-crate-amber border border-crate-amber/20'
                    : 'text-crate-muted hover:text-crate-cream hover:bg-crate-elevated'
                  } ${dragOverCrateId === p.id && p.id !== 'unsorted'
                    ? 'ring-2 ring-crate-cyan bg-crate-cyan/10'
                    : ''
                  }`}
                  onDragOver={p.id !== 'unsorted' ? (e) => handleDragOver(e, p.id) : undefined}
                  onDragLeave={p.id !== 'unsorted' ? handleDragLeave : undefined}
                  onDrop={p.id !== 'unsorted' ? (e) => handleDrop(e, p.id) : undefined}
                >
                  <Folder size={16} className={selectedCrateId === p.id ? 'text-crate-amber' : ''} />
                  <span className="truncate">{p.name}</span>
                </Link>
              ))}
              <Link
                to="/crates"
                className="flex items-center gap-3 rounded-xl px-4 py-2.5 text-sm text-crate-subtle hover:text-crate-cream hover:bg-crate-elevated transition-all"
              >
                <FolderOpen size={16} />
                Manage Crates
              </Link>
            </div>
          </nav>

          <div className="mt-8 pt-6 border-t border-crate-border">
            <div className="text-xs text-crate-subtle mb-2">Signed in as</div>
            <div className="flex items-center justify-between">
              <div className="text-sm text-crate-cream truncate">{user?.email}</div>
              <button
                className="btn-ghost p-2 rounded-lg text-crate-muted hover:text-crate-danger transition-colors"
                onClick={logout}
                title="Logout"
              >
                <LogOut size={16} />
              </button>
            </div>
          </div>
        </aside>

        <main className="space-y-6 overflow-visible animate-fade-in">
          <Outlet />
        </main>
      </div>

      {/* Mobile drawer overlay */}
      <div
        className={`fixed inset-0 z-40 bg-crate-black/60 backdrop-blur-sm transition-opacity duration-200 lg:hidden ${mobileMenuOpen ? 'opacity-100 pointer-events-auto' : 'opacity-0 pointer-events-none'}`}
        onClick={() => setMobileMenuOpen(false)}
      />

      {/* Mobile drawer */}
      <div
        className={`fixed inset-y-0 left-0 z-50 w-72 bg-crate-surface border-r border-crate-border p-5 transform transition-transform duration-200 lg:hidden ${mobileMenuOpen ? 'translate-x-0' : '-translate-x-full'}`}
      >
        <div className="mb-8 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-xl bg-crate-amber/10">
              <Disc className="text-crate-amber" size={24} />
            </div>
            <span className="text-xl font-display font-bold text-crate-cream">CrateDrop</span>
          </div>
          <button className="hw-button p-2" onClick={() => setMobileMenuOpen(false)} aria-label="Close menu">
            <X size={18} />
          </button>
        </div>
        <SidebarContent user={user} logout={logout} onNavigate={() => setMobileMenuOpen(false)} crates={crates} selectedCrateId={selectedCrateId} />
      </div>

      <PlayerBar />
    </div>
  )
}

function SidebarContent({ user, logout, onNavigate, crates, selectedCrateId }: {
  user: any;
  logout: () => void;
  onNavigate?: () => void;
  crates?: { crates: Array<{ id: string; name: string }> };
  selectedCrateId?: string | null;
}) {
  return (
    <>
      <nav className="space-y-1">
        <NavItem to="/" label="Library" icon={<Library size={18} />} onClick={onNavigate} />
        <NavItem to="/upload" label="Upload" icon={<UploadCloud size={18} />} onClick={onNavigate} />
        <NavItem to="/community" label="Community" icon={<Globe size={18} />} onClick={onNavigate} />
        {user?.role === 'admin' && (
          <NavItem to="/admin" label="Admin" icon={<Settings size={18} />} onClick={onNavigate} />
        )}

        <div className="pt-6 pb-2">
          <div className="text-xs font-medium text-crate-subtle uppercase tracking-wider px-4">
            Your Crates
          </div>
        </div>

        <div className="space-y-1 max-h-48 overflow-y-auto">
          {crates?.crates.map((p) => (
            <Link
              key={p.id}
              to={`/?crate=${p.id}`}
              onClick={onNavigate}
              className={`flex items-center gap-3 rounded-xl px-4 py-2.5 text-sm transition-all ${selectedCrateId === p.id
                ? 'bg-crate-amber/10 text-crate-amber border border-crate-amber/20'
                : 'text-crate-muted hover:text-crate-cream hover:bg-crate-elevated'
              }`}
            >
              <Folder size={16} className={selectedCrateId === p.id ? 'text-crate-amber' : ''} />
              <span className="truncate">{p.name}</span>
            </Link>
          ))}
          <Link
            to="/crates"
            onClick={onNavigate}
            className="flex items-center gap-3 rounded-xl px-4 py-2.5 text-sm text-crate-subtle hover:text-crate-cream hover:bg-crate-elevated transition-all"
          >
            <FolderOpen size={16} />
            Manage Crates
          </Link>
        </div>
      </nav>
      <div className="mt-8 pt-6 border-t border-crate-border">
        <div className="text-xs text-crate-subtle mb-2">Signed in as</div>
        <div className="flex items-center justify-between">
          <div className="text-sm text-crate-cream truncate">{user?.email}</div>
          <button
            className="btn-ghost p-2 rounded-lg text-crate-muted hover:text-crate-danger"
            onClick={() => {
              logout()
              onNavigate?.()
            }}
            title="Logout"
          >
            <LogOut size={16} />
          </button>
        </div>
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
        `flex items-center gap-3 rounded-xl px-4 py-3 text-sm font-medium transition-all ${isActive
          ? 'bg-crate-amber/10 text-crate-amber border border-crate-amber/20'
          : 'text-crate-muted hover:text-crate-cream hover:bg-crate-elevated'
        }`
      }
      onClick={onClick}
    >
      {icon}
      {label}
    </NavLink>
  )
}
