import { useCallback, useEffect, useState } from 'react'
import { useSearchParams, useNavigate } from 'react-router-dom'
import { Plus, Edit, Trash2, Music, MoreHorizontal, Globe, Lock } from 'lucide-react'
import { Crate, CreateCrateRequest, UpdateCrateRequest } from '../types/crates'
import { useToast } from '../hooks/useToast'
import { useCrates, useCreateCrate, useUpdateCrate, useDeleteCrate, useAddTracksToCrate } from '../hooks/useQueries'

export function CratesPage() {
  const { data: crates, isLoading: loading } = useCrates()
  const createCrateMutation = useCreateCrate()
  const updateCrateMutation = useUpdateCrate()
  const deleteCrateMutation = useDeleteCrate()
  const addTracksMutation = useAddTracksToCrate()

  const [showCreateModal, setShowCreateModal] = useState(false)
  const [editingCrate, setEditingCrate] = useState<Crate | null>(null)
  const [createForm, setCreateForm] = useState<CreateCrateRequest>({ name: '', description: '', is_public: true })
  const [updateForm, setUpdateForm] = useState<UpdateCrateRequest>({ name: '', description: '', is_public: true })
  const [searchParams, setSearchParams] = useSearchParams()
  const [menuOpen, setMenuOpen] = useState<string | null>(null)
  const [dragOverCrateId, setDragOverCrateId] = useState<string | null>(null)
  const toast = useToast()
  const navigate = useNavigate()

  useEffect(() => {
    if (searchParams.get('create') === '1') setShowCreateModal(true)
  }, [searchParams])

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (menuOpen && !(event.target as Element).closest('.crate-menu')) {
        setMenuOpen(null)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [menuOpen])

  const handleCreate = useCallback(async (e: React.FormEvent) => {
    e.preventDefault()
    if (!createForm.name.trim()) return

    try {
      await createCrateMutation.mutateAsync(createForm)
      toast.success(`Crate "${createForm.name}" created successfully`)
      setCreateForm({ name: '', description: '', is_public: true })
      setShowCreateModal(false)
      const next = new URLSearchParams(searchParams)
      next.delete('create')
      setSearchParams(next, { replace: true })
    } catch (error) {
      console.error('Failed to create crate:', error)
      toast.error('Failed to create crate')
    }
  }, [createForm, createCrateMutation, searchParams, setSearchParams, toast])

  const handleUpdate = useCallback(async (e: React.FormEvent) => {
    e.preventDefault()
    if (!editingCrate || !updateForm.name.trim()) return

    try {
      await updateCrateMutation.mutateAsync({ id: editingCrate.id, data: updateForm })
      toast.success(`Crate "${updateForm.name}" updated successfully`)
      setEditingCrate(null)
      setUpdateForm({ name: '', description: '', is_public: true })
    } catch (error) {
      console.error('Failed to update crate:', error)
      toast.error('Failed to update crate')
    }
  }, [editingCrate, updateForm, updateCrateMutation, toast])

  const handleDelete = useCallback(async (crate: Crate) => {
    if (!confirm(`Are you sure you want to delete "${crate.name}"? This action cannot be undone.`)) {
      return
    }

    try {
      await deleteCrateMutation.mutateAsync(crate.id)
      toast.success(`Crate "${crate.name}" deleted successfully`)
    } catch (error) {
      console.error('Failed to delete crate:', error)
      toast.error('Failed to delete crate')
    }
  }, [deleteCrateMutation, toast])

  const startEdit = useCallback((crate: Crate) => {
    setEditingCrate(crate)
    setUpdateForm({
      name: crate.name,
      description: crate.description || '',
      is_public: crate.is_public
    })
  }, [])

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
        {crates?.crates.filter(c => c.id !== 'unsorted').map((crate) => (
          <div
            key={crate.id}
            className={`card p-4 group transition-all relative cursor-pointer hover:bg-[#202020] ${dragOverCrateId === crate.id ? 'ring-2 ring-[#1DB954] bg-[#1DB954]/10' : ''}`}
            onDragOver={(e) => handleDragOver(e, crate.id)}
            onDragLeave={handleDragLeave}
            onDrop={(e) => handleDrop(e, crate.id)}
            onClick={() => navigate(`/?crate=${crate.id}`)}
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
                    onClick={(e) => {
                      e.stopPropagation()
                      setMenuOpen(menuOpen === crate.id ? null : crate.id)
                    }}
                  >
                    <MoreHorizontal size={16} />
                  </button>

                  {menuOpen === crate.id && (
                    <div className="crate-menu absolute right-0 mt-1 w-32 bg-[#1A1A1A] rounded-lg shadow-lg border border-[#2A2A2A] py-1 z-50">
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          startEdit(crate)
                          setMenuOpen(null)
                        }}
                        className="w-full text-left px-3 py-2 text-sm hover:bg-[#2A2A2A] flex items-center gap-2"
                      >
                        <Edit size={14} />
                        Edit
                      </button>
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
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
        (!crates?.crates.length || crates.crates.filter(c => c.id !== 'unsorted').length === 0) && (
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
