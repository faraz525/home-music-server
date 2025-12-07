import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Plus, Edit, Trash2, Music, MoreHorizontal, Globe, Lock } from 'lucide-react'
import { cratesApi, normalizeCrateList } from '../lib/api'
import { Crate, CrateList, CreateCrateRequest, UpdateCrateRequest } from '../types/crates'

export function CratesPage() {
  const [crates, setCrates] = useState<CrateList>({ crates: [], total: 0, limit: 20, offset: 0, has_next: false })
  const [loading, setLoading] = useState(true)
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [editingCrate, setEditingCrate] = useState<Crate | null>(null)
  const [createForm, setCreateForm] = useState<CreateCrateRequest>({ name: '', description: '', is_public: true })
  const [updateForm, setUpdateForm] = useState<UpdateCrateRequest>({ name: '', description: '', is_public: true })
  const [searchParams, setSearchParams] = useSearchParams()
  const [menuOpen, setMenuOpen] = useState<string | null>(null)
  const [dragOverCrateId, setDragOverCrateId] = useState<string | null>(null)

  useEffect(() => {
    fetchCrates()
  }, [])

  useEffect(() => {
    if (searchParams.get('create') === '1') setShowCreateModal(true)
  }, [searchParams])

  // Close menu when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (menuOpen && !(event.target as Element).closest('.crate-menu')) {
        setMenuOpen(null)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [menuOpen])

  const fetchCrates = async () => {
    try {
      const { data } = await cratesApi.list()
      // Use normalizeCrateList to handle both 'crates' and 'playlists' fields
      setCrates(normalizeCrateList(data))
    } catch (error) {
      console.error('Failed to fetch crates:', error)
      setCrates({ crates: [], total: 0, limit: 20, offset: 0, has_next: false })
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!createForm.name.trim()) return

    try {
      await cratesApi.create(createForm)
      setCreateForm({ name: '', description: '', is_public: true })
      setShowCreateModal(false)
      window.dispatchEvent(new CustomEvent('crates:updated'))
      const next = new URLSearchParams(searchParams)
      next.delete('create')
      setSearchParams(next, { replace: true })
      await fetchCrates()
    } catch (error) {
      console.error('Failed to create crate:', error)
    }
  }

  const handleUpdate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!editingCrate || !updateForm.name.trim()) return

    try {
      await cratesApi.update(editingCrate.id, updateForm)
      setEditingCrate(null)
      setUpdateForm({ name: '', description: '', is_public: true })
      window.dispatchEvent(new CustomEvent('crates:updated'))
      await fetchCrates()
    } catch (error) {
      console.error('Failed to update crate:', error)
    }
  }

  const handleDelete = async (crate: Crate) => {
    if (!confirm(`Are you sure you want to delete "${crate.name}"? This action cannot be undone.`)) {
      return
    }

    try {
      await cratesApi.delete(crate.id)
      window.dispatchEvent(new CustomEvent('crates:updated'))
      await fetchCrates()
    } catch (error) {
      console.error('Failed to delete crate:', error)
    }
  }

  const startEdit = (crate: Crate) => {
    setEditingCrate(crate)
    setUpdateForm({
      name: crate.name,
      description: crate.description || '',
      is_public: crate.is_public
    })
  }

  // Drag and drop handlers
  const handleDragOver = (e: React.DragEvent, crateId: string) => {
    e.preventDefault()
    e.stopPropagation()
    e.dataTransfer.dropEffect = 'copy'
    setDragOverCrateId(crateId)
  }

  const handleDragLeave = (e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setDragOverCrateId(null)
  }

  const handleDrop = async (e: React.DragEvent, crateId: string) => {
    e.preventDefault()
    e.stopPropagation()
    setDragOverCrateId(null)

    try {
      const data = e.dataTransfer.getData('application/json')
      if (data) {
        const { trackIds } = JSON.parse(data)
        if (trackIds && Array.isArray(trackIds) && trackIds.length > 0) {
          await cratesApi.addTracks(crateId, trackIds)
          window.dispatchEvent(new CustomEvent('crates:updated'))
          await fetchCrates()
        }
      }
    } catch (error) {
      console.error('Failed to add tracks to crate:', error)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-[#A1A1A1]">Loading crates...</div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Your Crates</h1>
          <p className="text-[#A1A1A1] mt-1">Organize your music into custom crates</p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="btn btn-primary flex items-center gap-2"
        >
          <Plus size={18} />
          Create Crate
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {Array.isArray(crates.crates) && crates.crates.filter(c => c.id !== 'unsorted').map((crate) => (
          <div
            key={crate.id}
            className={`card p-4 group transition-all relative ${dragOverCrateId === crate.id ? 'ring-2 ring-[#1DB954] bg-[#1DB954]/10' : ''}`}
            onDragOver={(e) => handleDragOver(e, crate.id)}
            onDragLeave={handleDragLeave}
            onDrop={(e) => handleDrop(e, crate.id)}
          >
            <div className="flex items-start justify-between mb-3">
              <div className="flex items-center gap-3 flex-1">
                <div className="w-12 h-12 bg-[#1DB954] rounded-lg flex items-center justify-center">
                  <Music size={24} className="text-black" />
                </div>
                <div className="flex-1 min-w-0">
                  <h3 className="font-semibold truncate">{crate.name}</h3>
                  <p className="text-sm text-[#A1A1A1] truncate">
                    {crate.description || 'No description'}
                  </p>
                  {crate.is_default && (
                    <span className="text-xs bg-[#2A2A2A] px-2 py-1 rounded mt-1 inline-block">
                      Default
                    </span>
                  )}
                </div>
              </div>

              <div className="absolute top-4 right-12">
                {crate.is_public ? (
                  <Globe size={16} className="text-[#A1A1A1]" />
                ) : (
                  <Lock size={16} className="text-[#A1A1A1]" />
                )}
              </div>

              {!crate.is_default && (
                <div className="relative">
                  <button
                    className="p-1 hover:bg-[#2A2A2A] rounded"
                    onClick={() => setMenuOpen(menuOpen === crate.id ? null : crate.id)}
                  >
                    <MoreHorizontal size={16} />
                  </button>

                  {menuOpen === crate.id && (
                    <div className="crate-menu absolute right-0 mt-1 w-32 bg-[#1A1A1A] rounded-lg shadow-lg border border-[#2A2A2A] py-1 z-50">
                      <button
                        onClick={() => {
                          startEdit(crate)
                          setMenuOpen(null)
                        }}
                        className="w-full text-left px-3 py-2 text-sm hover:bg-[#2A2A2A] flex items-center gap-2"
                      >
                        <Edit size={14} />
                        Edit
                      </button>
                      <button
                        onClick={() => {
                          handleDelete(crate)
                          setMenuOpen(null)
                        }}
                        className="w-full text-left px-3 py-2 text-sm hover:bg-[#2A2A2A] flex items-center gap-2 text-red-400"
                      >
                        <Trash2 size={14} />
                        Delete
                      </button>
                    </div>
                  )}
                </div>
              )}
            </div>

            <div className="text-xs text-[#A1A1A1]">
              Created {new Date(crate.created_at).toLocaleDateString()}
            </div>
          </div>
        ))}
      </div>

      {
        (!Array.isArray(crates.crates) || crates.crates.filter(c => c.id !== 'unsorted').length === 0) && (
          <div className="text-center py-12">
            <Music size={48} className="mx-auto text-[#A1A1A1] mb-4" />
            <h3 className="text-lg font-semibold mb-2">No crates yet</h3>
            <p className="text-[#A1A1A1] mb-4">Create your first crate to start organizing your music</p>
            <button
              onClick={() => setShowCreateModal(true)}
              className="btn btn-primary"
            >
              Create Your First Crate
            </button>
          </div>
        )
      }

      {/* Create Crate Modal */}
      {
        showCreateModal && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
            <div className="bg-[#1A1A1A] rounded-lg p-6 w-full max-w-md">
              <h2 className="text-xl font-bold mb-4">Create New Crate</h2>

              <form onSubmit={handleCreate} className="space-y-4">
                <div>
                  <label className="block text-sm font-medium mb-1">Name *</label>
                  <input
                    type="text"
                    value={createForm.name}
                    onChange={(e) => setCreateForm({ ...createForm, name: e.target.value })}
                    className="input w-full"
                    placeholder="My Awesome Playlist"
                    required
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">Description</label>
                  <textarea
                    value={createForm.description}
                    onChange={(e) => setCreateForm({ ...createForm, description: e.target.value })}
                    className="input w-full h-24 resize-none"
                    placeholder="Optional description..."
                  />
                </div>

                <div className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    id="create-public"
                    checked={createForm.is_public}
                    onChange={(e) => setCreateForm({ ...createForm, is_public: e.target.checked })}
                    className="rounded bg-[#2A2A2A] border-none text-[#1DB954] focus:ring-[#1DB954]"
                  />
                  <label htmlFor="create-public" className="text-sm font-medium cursor-pointer select-none flex items-center gap-2">
                    {createForm.is_public ? <Globe size={14} /> : <Lock size={14} />}
                    {createForm.is_public ? 'Public Crate' : 'Private Crate'}
                  </label>
                </div>

                <div className="flex gap-3 pt-2">
                  <button type="submit" className="btn btn-primary flex-1">
                    Create Crate
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      setShowCreateModal(false)
                      setCreateForm({ name: '', description: '', is_public: true })
                      const next = new URLSearchParams(searchParams)
                      next.delete('create')
                      setSearchParams(next, { replace: true })
                    }}
                    className="btn flex-1"
                  >
                    Cancel
                  </button>
                </div>
              </form>
            </div>
          </div>
        )
      }

      {/* Edit Crate Modal */}
      {
        editingCrate && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
            <div className="bg-[#1A1A1A] rounded-lg p-6 w-full max-w-md">
              <h2 className="text-xl font-bold mb-4">Edit Crate</h2>

              <form onSubmit={handleUpdate} className="space-y-4">
                <div>
                  <label className="block text-sm font-medium mb-1">Name *</label>
                  <input
                    type="text"
                    value={updateForm.name}
                    onChange={(e) => setUpdateForm({ ...updateForm, name: e.target.value })}
                    className="input w-full"
                    required
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">Description</label>
                  <textarea
                    value={updateForm.description}
                    onChange={(e) => setUpdateForm({ ...updateForm, description: e.target.value })}
                    className="input w-full h-24 resize-none"
                    placeholder="Optional description..."
                  />
                </div>

                <div className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    id="update-public"
                    checked={updateForm.is_public}
                    onChange={(e) => setUpdateForm({ ...updateForm, is_public: e.target.checked })}
                    className="rounded bg-[#2A2A2A] border-none text-[#1DB954] focus:ring-[#1DB954]"
                  />
                  <label htmlFor="update-public" className="text-sm font-medium cursor-pointer select-none flex items-center gap-2">
                    {updateForm.is_public ? <Globe size={14} /> : <Lock size={14} />}
                    {updateForm.is_public ? 'Public Crate' : 'Private Crate'}
                  </label>
                </div>

                <div className="flex gap-3 pt-2">
                  <button type="submit" className="btn btn-primary flex-1">
                    Update Crate
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      setEditingCrate(null)
                      setUpdateForm({ name: '', description: '', is_public: true })
                    }}
                    className="btn flex-1"
                  >
                    Cancel
                  </button>
                </div>
              </form>
            </div>
          </div>
        )
      }
    </div >
  )
}
