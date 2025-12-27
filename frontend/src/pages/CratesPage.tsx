import { useCallback, useEffect, useState } from 'react'
import { useSearchParams, useNavigate } from 'react-router-dom'
import { Plus, Edit, Trash2, Music, MoreHorizontal, Globe, Lock, Disc } from 'lucide-react'
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
      <div className="flex flex-col items-center justify-center py-16">
        <Disc className="text-crate-amber vinyl-spinning-slow mb-4" size={48} />
        <div className="text-crate-muted">Loading crates...</div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-display font-bold text-crate-cream">Your Crates</h1>
          <p className="text-crate-muted mt-1">Organize your music into custom crates</p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="btn btn-primary flex items-center gap-2"
        >
          <Plus size={18} />
          Create Crate
        </button>
      </div>

      {/* Crate grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {crates?.crates.filter(c => c.id !== 'unsorted').map((crate, idx) => (
          <div
            key={crate.id}
            className={`stagger-item card p-5 group transition-all relative cursor-pointer hover:shadow-glow ${dragOverCrateId === crate.id ? 'ring-2 ring-crate-cyan shadow-glow-cyan' : ''}`}
            style={{ animationDelay: `${idx * 50}ms` }}
            onDragOver={(e) => handleDragOver(e, crate.id)}
            onDragLeave={handleDragLeave}
            onDrop={(e) => handleDrop(e, crate.id)}
            onClick={() => navigate(`/?crate=${crate.id}`)}
          >
            <div className="flex items-start gap-4">
              {/* Crate icon */}
              <div className="w-14 h-14 rounded-xl bg-gradient-to-br from-crate-amber to-crate-amberDark flex items-center justify-center flex-shrink-0 shadow-glow">
                <Music size={24} className="text-crate-black" />
              </div>

              {/* Content */}
              <div className="flex-1 min-w-0">
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0">
                    <h3 className="font-display font-semibold text-crate-cream truncate">
                      {crate.name}
                    </h3>
                    <p className="text-sm text-crate-muted truncate mt-0.5">
                      {crate.description || 'No description'}
                    </p>
                  </div>

                  {/* Status badges */}
                  <div className="flex items-center gap-2 flex-shrink-0">
                    <div className={`p-1.5 rounded-lg ${crate.is_public ? 'bg-crate-cyan/10 text-crate-cyan' : 'bg-crate-elevated text-crate-subtle'}`}>
                      {crate.is_public ? <Globe size={14} /> : <Lock size={14} />}
                    </div>

                    {!crate.is_default && (
                      <div className="relative">
                        <button
                          className="p-1.5 rounded-lg hover:bg-crate-elevated text-crate-muted hover:text-crate-cream transition-colors"
                          onClick={(e) => {
                            e.stopPropagation()
                            setMenuOpen(menuOpen === crate.id ? null : crate.id)
                          }}
                        >
                          <MoreHorizontal size={14} />
                        </button>

                        {menuOpen === crate.id && (
                          <div className="crate-menu absolute right-0 mt-1 w-36 bg-crate-elevated rounded-xl shadow-elevated border border-crate-border py-1 z-50">
                            <button
                              onClick={(e) => {
                                e.stopPropagation()
                                startEdit(crate)
                                setMenuOpen(null)
                              }}
                              className="w-full text-left px-3 py-2 text-sm text-crate-cream hover:bg-crate-border flex items-center gap-2 transition-colors"
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
                              className="w-full text-left px-3 py-2 text-sm text-crate-danger hover:bg-crate-danger/10 flex items-center gap-2 transition-colors"
                            >
                              <Trash2 size={14} />
                              Delete
                            </button>
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                </div>

                {/* Metadata */}
                <div className="flex items-center gap-3 mt-3 text-xs text-crate-subtle">
                  {crate.is_default && (
                    <span className="px-2 py-1 rounded-md bg-crate-elevated">Default</span>
                  )}
                  <span>Created {new Date(crate.created_at).toLocaleDateString()}</span>
                </div>
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* Empty state */}
      {(!crates?.crates.length || crates.crates.filter(c => c.id !== 'unsorted').length === 0) && (
        <div className="text-center py-16">
          <div className="w-24 h-24 mx-auto mb-6 rounded-full bg-crate-elevated flex items-center justify-center">
            <Music size={40} className="text-crate-subtle" />
          </div>
          <h3 className="text-xl font-display font-semibold text-crate-cream mb-2">No crates yet</h3>
          <p className="text-crate-muted mb-6">Create your first crate to start organizing your music</p>
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
        <div className="fixed inset-0 bg-crate-black/80 backdrop-blur-sm flex items-center justify-center p-4 z-50">
          <div className="bg-crate-surface border border-crate-border rounded-2xl p-6 w-full max-w-md shadow-elevated animate-slide-up">
            <h2 className="text-xl font-display font-bold text-crate-cream mb-6">Create New Crate</h2>

            <form onSubmit={handleCreate} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-crate-cream mb-2">Name *</label>
                <input
                  type="text"
                  value={createForm.name}
                  onChange={(e) => setCreateForm({ ...createForm, name: e.target.value })}
                  className="input w-full"
                  placeholder="My Awesome Playlist"
                  required
                  autoFocus
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-crate-cream mb-2">Description</label>
                <textarea
                  value={createForm.description}
                  onChange={(e) => setCreateForm({ ...createForm, description: e.target.value })}
                  className="input w-full h-24 resize-none"
                  placeholder="Optional description..."
                />
              </div>

              <div className="flex items-center gap-3 p-3 rounded-xl bg-crate-elevated">
                <input
                  type="checkbox"
                  id="create-public"
                  checked={createForm.is_public}
                  onChange={(e) => setCreateForm({ ...createForm, is_public: e.target.checked })}
                />
                <label htmlFor="create-public" className="flex items-center gap-2 text-sm font-medium text-crate-cream cursor-pointer select-none">
                  {createForm.is_public ? (
                    <>
                      <Globe size={16} className="text-crate-cyan" />
                      Public Crate
                    </>
                  ) : (
                    <>
                      <Lock size={16} className="text-crate-muted" />
                      Private Crate
                    </>
                  )}
                </label>
              </div>

              <div className="flex gap-3 pt-4">
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
      )}

      {/* Edit Crate Modal */}
      {editingCrate && (
        <div className="fixed inset-0 bg-crate-black/80 backdrop-blur-sm flex items-center justify-center p-4 z-50">
          <div className="bg-crate-surface border border-crate-border rounded-2xl p-6 w-full max-w-md shadow-elevated animate-slide-up">
            <h2 className="text-xl font-display font-bold text-crate-cream mb-6">Edit Crate</h2>

            <form onSubmit={handleUpdate} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-crate-cream mb-2">Name *</label>
                <input
                  type="text"
                  value={updateForm.name}
                  onChange={(e) => setUpdateForm({ ...updateForm, name: e.target.value })}
                  className="input w-full"
                  required
                  autoFocus
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-crate-cream mb-2">Description</label>
                <textarea
                  value={updateForm.description}
                  onChange={(e) => setUpdateForm({ ...updateForm, description: e.target.value })}
                  className="input w-full h-24 resize-none"
                  placeholder="Optional description..."
                />
              </div>

              <div className="flex items-center gap-3 p-3 rounded-xl bg-crate-elevated">
                <input
                  type="checkbox"
                  id="update-public"
                  checked={updateForm.is_public}
                  onChange={(e) => setUpdateForm({ ...updateForm, is_public: e.target.checked })}
                />
                <label htmlFor="update-public" className="flex items-center gap-2 text-sm font-medium text-crate-cream cursor-pointer select-none">
                  {updateForm.is_public ? (
                    <>
                      <Globe size={16} className="text-crate-cyan" />
                      Public Crate
                    </>
                  ) : (
                    <>
                      <Lock size={16} className="text-crate-muted" />
                      Private Crate
                    </>
                  )}
                </label>
              </div>

              <div className="flex gap-3 pt-4">
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
      )}
    </div>
  )
}
