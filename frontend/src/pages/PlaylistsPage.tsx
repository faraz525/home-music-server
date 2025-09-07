import { useEffect, useState } from 'react'
import { Plus, Edit, Trash2, Music, MoreHorizontal } from 'lucide-react'
import { playlistsApi } from '../lib/api'
import { Playlist, PlaylistList, CreatePlaylistRequest, UpdatePlaylistRequest } from '../types/playlists'

export function PlaylistsPage() {
  const [playlists, setPlaylists] = useState<PlaylistList>({ playlists: [], total: 0, limit: 20, offset: 0, has_next: false })
  const [loading, setLoading] = useState(true)
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [editingPlaylist, setEditingPlaylist] = useState<Playlist | null>(null)
  const [createForm, setCreateForm] = useState<CreatePlaylistRequest>({ name: '', description: '' })
  const [updateForm, setUpdateForm] = useState<UpdatePlaylistRequest>({ name: '', description: '' })

  useEffect(() => {
    fetchPlaylists()
  }, [])

  const fetchPlaylists = async () => {
    try {
      const { data } = await playlistsApi.list()
      setPlaylists(data)
    } catch (error) {
      console.error('Failed to fetch playlists:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!createForm.name.trim()) return

    try {
      await playlistsApi.create(createForm)
      setCreateForm({ name: '', description: '' })
      setShowCreateModal(false)
      await fetchPlaylists()
    } catch (error) {
      console.error('Failed to create playlist:', error)
    }
  }

  const handleUpdate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!editingPlaylist || !updateForm.name.trim()) return

    try {
      await playlistsApi.update(editingPlaylist.id, updateForm)
      setEditingPlaylist(null)
      setUpdateForm({ name: '', description: '' })
      await fetchPlaylists()
    } catch (error) {
      console.error('Failed to update playlist:', error)
    }
  }

  const handleDelete = async (playlist: Playlist) => {
    if (!confirm(`Are you sure you want to delete "${playlist.name}"? This action cannot be undone.`)) {
      return
    }

    try {
      await playlistsApi.delete(playlist.id)
      await fetchPlaylists()
    } catch (error) {
      console.error('Failed to delete playlist:', error)
    }
  }

  const startEdit = (playlist: Playlist) => {
    setEditingPlaylist(playlist)
    setUpdateForm({
      name: playlist.name,
      description: playlist.description || ''
    })
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-[#A1A1A1]">Loading playlists...</div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Your Playlists</h1>
          <p className="text-[#A1A1A1] mt-1">Organize your music into custom playlists</p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="btn btn-primary flex items-center gap-2"
        >
          <Plus size={18} />
          Create Playlist
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {playlists.playlists.map((playlist) => (
          <div key={playlist.id} className="card p-4 group">
            <div className="flex items-start justify-between mb-3">
              <div className="flex items-center gap-3 flex-1">
                <div className="w-12 h-12 bg-[#1DB954] rounded-lg flex items-center justify-center">
                  <Music size={24} className="text-black" />
                </div>
                <div className="flex-1 min-w-0">
                  <h3 className="font-semibold truncate">{playlist.name}</h3>
                  <p className="text-sm text-[#A1A1A1] truncate">
                    {playlist.description || 'No description'}
                  </p>
                  {playlist.is_default && (
                    <span className="text-xs bg-[#2A2A2A] px-2 py-1 rounded mt-1 inline-block">
                      Default
                    </span>
                  )}
                </div>
              </div>

              {!playlist.is_default && (
                <div className="opacity-0 group-hover:opacity-100 transition-opacity">
                  <div className="relative">
                    <button className="p-1 hover:bg-[#2A2A2A] rounded">
                      <MoreHorizontal size={16} />
                    </button>

                    <div className="absolute right-0 mt-1 w-32 bg-[#1A1A1A] rounded-lg shadow-lg border border-[#2A2A2A] py-1 opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none group-hover:pointer-events-auto">
                      <button
                        onClick={() => startEdit(playlist)}
                        className="w-full text-left px-3 py-2 text-sm hover:bg-[#2A2A2A] flex items-center gap-2"
                      >
                        <Edit size={14} />
                        Edit
                      </button>
                      <button
                        onClick={() => handleDelete(playlist)}
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
              Created {new Date(playlist.created_at).toLocaleDateString()}
            </div>
          </div>
        ))}
      </div>

      {playlists.playlists.length === 0 && (
        <div className="text-center py-12">
          <Music size={48} className="mx-auto text-[#A1A1A1] mb-4" />
          <h3 className="text-lg font-semibold mb-2">No playlists yet</h3>
          <p className="text-[#A1A1A1] mb-4">Create your first playlist to start organizing your music</p>
          <button
            onClick={() => setShowCreateModal(true)}
            className="btn btn-primary"
          >
            Create Your First Playlist
          </button>
        </div>
      )}

      {/* Create Playlist Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
          <div className="bg-[#1A1A1A] rounded-lg p-6 w-full max-w-md">
            <h2 className="text-xl font-bold mb-4">Create New Playlist</h2>

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
                  Create Playlist
                </button>
                <button
                  type="button"
                  onClick={() => {
                    setShowCreateModal(false)
                    setCreateForm({ name: '', description: '' })
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

      {/* Edit Playlist Modal */}
      {editingPlaylist && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
          <div className="bg-[#1A1A1A] rounded-lg p-6 w-full max-w-md">
            <h2 className="text-xl font-bold mb-4">Edit Playlist</h2>

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
                  Update Playlist
                </button>
                <button
                  type="button"
                  onClick={() => {
                    setEditingPlaylist(null)
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
