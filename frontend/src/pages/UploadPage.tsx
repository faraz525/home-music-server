import { ChangeEvent, FormEvent, useEffect, useState } from 'react'
import { api, cratesApi, CrateList } from '../lib/api'
import type { Crate } from '../types/crates'
import { Upload, Music, X } from 'lucide-react'

export function UploadPage() {
  const [files, setFiles] = useState<File[]>([])
  const [selectedCrate, setSelectedCrate] = useState<string>('unsorted')
  const [crates, setCrates] = useState<CrateList>({ crates: [], total: 0, limit: 20, offset: 0, has_next: false })
  const [loadingCrates, setLoadingCrates] = useState(true)
  const [uploading, setUploading] = useState(false)
  const [progress, setProgress] = useState<{ [key: string]: number }>({})
  const [completed, setCompleted] = useState<Set<string>>(new Set())

  // Fetch crates
  const fetchCrates = async () => {
    try {
      setLoadingCrates(true)
      const { data } = await cratesApi.list()
      setCrates(data)
    } catch (error) {
      console.error('Failed to fetch crates:', error)
      setCrates({ crates: [], total: 0, limit: 20, offset: 0, has_next: false })
    } finally {
      setLoadingCrates(false)
    }
  }

  useEffect(() => {
    fetchCrates()
  }, [])

  const handleFileSelect = (e: ChangeEvent<HTMLInputElement>) => {
    const selectedFiles = Array.from(e.target.files || [])
    const audioFiles = selectedFiles.filter(file => file.type.startsWith('audio/'))
    setFiles(prev => [...prev, ...audioFiles])
  }

  const removeFile = (index: number) => {
    setFiles(prev => prev.filter((_, i) => i !== index))
  }

  const uploadFile = async (file: File, index: number) => {
    const form = new FormData()
    form.append('file', file)

    // Add crate assignment if a crate is selected (not 'unsorted')
    if (selectedCrate && selectedCrate !== 'unsorted') {
      form.append('playlist_id', selectedCrate)
    }

    const fileKey = `${file.name}-${index}`
    setProgress(prev => ({ ...prev, [fileKey]: 0 }))

    try {
      await api.post('/api/tracks', form, {
        headers: { 'Content-Type': 'multipart/form-data' },
        onUploadProgress: (ev) => {
          if (ev.total) {
            const percent = Math.round((ev.loaded * 100) / ev.total)
            setProgress(prev => ({ ...prev, [fileKey]: percent }))
          }
        },
      })
      setCompleted(prev => new Set([...prev, fileKey]))
    } catch (error) {
      console.error('Failed to upload file:', error)
    } finally {
      setProgress(prev => {
        const next = { ...prev }
        delete next[fileKey]
        return next
      })
    }
  }

  const handleUpload = async (e: FormEvent) => {
    e.preventDefault()
    if (files.length === 0) return

    setUploading(true)
    setCompleted(new Set())

    try {
      // Upload files sequentially to avoid overwhelming the server
      for (let i = 0; i < files.length; i++) {
        await uploadFile(files[i], i)
      }

      // Clear files after successful upload
      setFiles([])
      setCompleted(new Set())
    } finally {
      setUploading(false)
    }
  }

  const formatFileSize = (bytes: number) => {
    if (bytes === 0) return '0 Bytes'
    const k = 1024
    const sizes = ['Bytes', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Upload Music</h1>
        <p className="text-[#A1A1A1] mt-1">Upload songs to your library and organize them into crates</p>
      </div>

      <div className="card p-6">
        <form onSubmit={handleUpload} className="space-y-6">
          {/* File Selection */}
          <div>
            <label className="block text-sm font-medium mb-3">Select Files</label>
            <div className="relative border-2 border-dashed border-[#2A2A2A] rounded-lg p-8 text-center hover:border-[#1DB954] transition-colors" style={{ pointerEvents: 'none' }}>
              <Upload size={48} className="mx-auto text-[#A1A1A1] mb-4" />
              <div className="space-y-2">
                <p className="text-lg font-medium">Drop audio files here or click to browse</p>
                <p className="text-sm text-[#A1A1A1]">Supports MP3, WAV, FLAC, and other audio formats</p>
              </div>
              <input
                type="file"
                multiple
                accept="audio/*"
                onChange={handleFileSelect}
                onClick={(e) => e.stopPropagation()}
                className="absolute inset-0 w-full h-full opacity-0 cursor-pointer"
                style={{ pointerEvents: 'auto' }}
              />
            </div>
          </div>

          {/* File List */}
          {files.length > 0 && (
            <div>
              <label className="block text-sm font-medium mb-3">Selected Files ({files.length})</label>
              <div className="space-y-2 max-h-60 overflow-y-auto">
                {files.map((file, index) => {
                  const fileKey = `${file.name}-${index}`
                  const uploadProgress = progress[fileKey]
                  const isCompleted = completed.has(fileKey)

                  return (
                    <div key={index} className="flex items-center gap-3 p-3 bg-[#1A1A1A] rounded-lg">
                      <Music size={20} className="text-[#1DB954] flex-shrink-0" />
                      <div className="flex-1 min-w-0">
                        <div className="truncate font-medium">{file.name}</div>
                        <div className="text-sm text-[#A1A1A1]">{formatFileSize(file.size)}</div>
                        {uploadProgress !== undefined && !isCompleted && (
                          <div className="mt-1">
                            <div className="w-full bg-[#2A2A2A] rounded-full h-2">
                              <div
                                className="bg-[#1DB954] h-2 rounded-full transition-all duration-300"
                                style={{ width: `${uploadProgress}%` }}
                              />
                            </div>
                            <div className="text-xs text-[#A1A1A1] mt-1">{uploadProgress}% uploaded</div>
                          </div>
                        )}
                        {isCompleted && (
                          <div className="text-sm text-green-400 mt-1">✓ Uploaded successfully</div>
                        )}
                      </div>
                      {!uploading && (
                        <button
                          type="button"
                          onClick={() => removeFile(index)}
                          className="p-1 hover:bg-[#2A2A2A] rounded"
                        >
                          <X size={16} />
                        </button>
                      )}
                    </div>
                  )
                })}
              </div>
            </div>
          )}

          {/* Crate Selection */}
          <div>
            <label className="block text-sm font-medium mb-3">Add to Crate (Optional)</label>
            <select
              value={selectedCrate}
              onChange={(e) => setSelectedCrate(e.target.value)}
              className="input w-full"
              disabled={uploading}
            >
              <option value="unsorted">Unsorted (Default)</option>
              {loadingCrates ? (
                <option disabled>Loading crates...</option>
              ) : (
                crates.crates?.filter(c => !c.is_default).map((crate) => (
                  <option key={crate.id} value={crate.id}>
                    {crate.name}
                  </option>
                ))
              )}
            </select>
            <p className="text-xs text-[#A1A1A1] mt-1">
              Files will be added to the selected crate. If no crate is selected, they'll go to the unsorted collection.
            </p>
          </div>

          {/* Upload Button */}
          <div className="flex justify-end">
            <button
              type="submit"
              disabled={files.length === 0 || uploading}
              className="btn btn-primary px-8 py-3"
            >
              {uploading ? 'Uploading...' : `Upload ${files.length} File${files.length !== 1 ? 's' : ''}`}
            </button>
          </div>
        </form>
      </div>

      {/* Help Text */}
      <div className="card p-4">
        <h3 className="font-semibold mb-2">Upload Tips</h3>
        <ul className="text-sm text-[#A1A1A1] space-y-1">
          <li>• You can upload multiple files at once by selecting them or dragging and dropping</li>
          <li>• Supported formats: MP3, WAV, FLAC, AAC, OGG, and more</li>
          <li>• Files without crate assignment will appear in the "Unsorted" collection</li>
          <li>• You can organize uploaded files into crates from the Library page</li>
        </ul>
      </div>
    </div>
  )
}
