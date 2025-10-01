import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Plus, Edit, Trash2, Music, MoreHorizontal } from 'lucide-react'
import { cratesApi } from '../lib/api'
import { Crate, CrateList, CreateCrateRequest, UpdateCrateRequest } from '../types/crates'

export function CratesPage() {
  const [crates, setCrates] = useState<CrateList>({ crates: [], total: 0, limit: 20, offset: 0, has_next: false })
  const [loading, setLoading] = useState(true)
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [editingCrate, setEditingCrate] = useState<Crate | null>(null)
  const [createForm, setCreateForm] = useState<CreateCrateRequest>({ name: '', description: '' })
  const [updateForm, setUpdateForm] = useState<UpdateCrateRequest>({ name: '', description: '' })
  const [searchParams, setSearchParams] = useSearchParams()

  useEffect(() => {
    fetchCrates()
  }, [])

  useEffect(() => {
    if (searchParams.get('create') === '1') setShowCreateModal(true)
  }, [searchParams])

  const fetchCrates = async () => {
    try {
      const { data } = await cratesApi.list()
      // Ensure safe structure
      const safe: CrateList = data && Array.isArray(data.crates)
        ? data
        : { crates: [], total: 0, limit: 20, offset: 0, has_next: false }
      setCrates(safe)
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
      setCreateForm({ name: '', description: '' })
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
      setUpdateForm({ name: '', description: '' })
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
      description: crate.description || ''
    })
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
        {Array.isArray(crates.crates) && crates.crates.map((crate) => (
          <div key={crate.id} className="card p-4 group">
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

              {!crate.is_default && (
                <div className="opacity-0 group-hover:opacity-100 transition-opacity">
                  <div className="relative">
                    <button className="p-1 hover:bg-[#2A2A2A] rounded">
                      <MoreHorizontal size={16} />
                    </button>

                    <div className="absolute right-0 mt-1 w-32 bg-[#1A1A1A] rounded-lg shadow-lg border border-[#2A2A2A] py-1 opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none group-hover:pointer-events-auto z-30">
                      <button
                        onClick={() => startEdit(crate)}
                        className="w-full text-left px-3 py-2 text-sm hover:bg-[#2A2A2A] flex items-center gap-2"
                      >
                        <Edit size={14} />
                        Edit
                      </button>
                      <button
                        onClick={() => handleDelete(crate)}
                        className="w-full text-left px-3 py-2 text-sm hover:bg-[#2A2A2A] flex items-center gap-2 text-red-400"
                      >
                        <Trash2 size={14} />
                        Delete
                      </button>
                    </div>
                  </div>
                </div>
              )}
            </div>

            <div className="text-xs text-[#A1A1A1]">
              Created {new Date(crate.created_at).toLocaleDateString()}
            </div>
          </div>
        ))}
      </div>

      {(!Array.isArray(crates.crates) || crates.crates.length === 0) && (
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
      )}

      {/* Create Crate Modal */}
      {showCreateModal && (
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

              <div className="flex gap-3 pt-2">
                <button type="submit" className="btn btn-primary flex-1">
                  Create Crate
                </button>
                <button
                  type="button"
                  onClick={() => {
                    setShowCreateModal(false)
                    setCreateForm({ name: '', description: '' })
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
      )}

      {/* Edit Crate Modal */}
      {editingCrate && (
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

              <div className="flex gap-3 pt-2">
                <button type="submit" className="btn btn-primary flex-1">
                  Update Crate
                </button>
                <button
                  type="button"
                  onClick={() => {
                    setEditingCrate(null)
                    setUpdateForm({ name: '', description: '' })
                  }}
                  className="btn flex-1"
                >
                  Cancel
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
