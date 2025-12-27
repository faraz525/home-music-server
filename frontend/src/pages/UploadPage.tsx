import { ChangeEvent, FormEvent, useEffect, useState } from 'react'
import { api, cratesApi, normalizeCrateList } from '../lib/api'
import type { Crate, CrateList } from '../types/crates'
import { Upload, Music, X, Check, Disc } from 'lucide-react'

export function UploadPage() {
  const [files, setFiles] = useState<File[]>([])
  const [selectedCrate, setSelectedCrate] = useState<string>('unsorted')
  const [crates, setCrates] = useState<CrateList>({ crates: [], total: 0, limit: 20, offset: 0, has_next: false })
  const [loadingCrates, setLoadingCrates] = useState(true)
  const [uploading, setUploading] = useState(false)
  const [progress, setProgress] = useState<{ [key: string]: number }>({})
  const [completed, setCompleted] = useState<Set<string>>(new Set())
  const [dragActive, setDragActive] = useState(false)

  const fetchCrates = async () => {
    try {
      setLoadingCrates(true)
      const { data } = await cratesApi.list()
      setCrates(normalizeCrateList(data))
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

  const handleDrag = (e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (e.type === 'dragenter' || e.type === 'dragover') {
      setDragActive(true)
    } else if (e.type === 'dragleave') {
      setDragActive(false)
    }
  }

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setDragActive(false)
    if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
      const droppedFiles = Array.from(e.dataTransfer.files)
      const audioFiles = droppedFiles.filter(file => file.type.startsWith('audio/'))
      setFiles(prev => [...prev, ...audioFiles])
    }
  }

  const removeFile = (index: number) => {
    setFiles(prev => prev.filter((_, i) => i !== index))
  }

  const uploadFile = async (file: File, index: number) => {
    const form = new FormData()
    form.append('file', file)

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
      for (let i = 0; i < files.length; i++) {
        await uploadFile(files[i], i)
      }
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
      {/* Header */}
      <div>
        <h1 className="text-3xl font-display font-bold text-crate-cream">Upload Music</h1>
        <p className="text-crate-muted mt-1">Upload songs to your library and organize them into crates</p>
      </div>

      <div className="card p-6">
        <form onSubmit={handleUpload} className="space-y-6">
          {/* File Selection */}
          <div>
            <label className="block text-sm font-medium text-crate-cream mb-3">Select Files</label>
            <div
              className={`relative border-2 border-dashed rounded-2xl p-10 text-center transition-all ${dragActive
                ? 'border-crate-amber bg-crate-amber/5'
                : 'border-crate-border hover:border-crate-subtle'
                }`}
              onDragEnter={handleDrag}
              onDragLeave={handleDrag}
              onDragOver={handleDrag}
              onDrop={handleDrop}
            >
              <div className="w-16 h-16 mx-auto mb-4 rounded-full bg-crate-elevated flex items-center justify-center">
                <Upload size={28} className={dragActive ? 'text-crate-amber' : 'text-crate-muted'} />
              </div>
              <div className="space-y-2">
                <p className="text-lg font-medium text-crate-cream">
                  {dragActive ? 'Drop files here' : 'Drop audio files here or click to browse'}
                </p>
                <p className="text-sm text-crate-muted">Supports MP3, WAV, FLAC, AAC, and more</p>
              </div>
              <input
                type="file"
                multiple
                accept="audio/*"
                onChange={handleFileSelect}
                className="absolute inset-0 w-full h-full opacity-0 cursor-pointer"
              />
            </div>
          </div>

          {/* File List */}
          {files.length > 0 && (
            <div>
              <label className="block text-sm font-medium text-crate-cream mb-3">
                Selected Files ({files.length})
              </label>
              <div className="space-y-2 max-h-72 overflow-y-auto">
                {files.map((file, index) => {
                  const fileKey = `${file.name}-${index}`
                  const uploadProgress = progress[fileKey]
                  const isCompleted = completed.has(fileKey)

                  return (
                    <div
                      key={index}
                      className={`stagger-item flex items-center gap-3 p-4 rounded-xl transition-colors ${isCompleted ? 'bg-crate-success/5 border border-crate-success/20' : 'bg-crate-elevated'
                        }`}
                      style={{ animationDelay: `${index * 30}ms` }}
                    >
                      <div className={`w-10 h-10 rounded-lg flex items-center justify-center flex-shrink-0 ${isCompleted ? 'bg-crate-success/20' : 'bg-crate-amber/10'
                        }`}>
                        {isCompleted ? (
                          <Check size={18} className="text-crate-success" />
                        ) : (
                          <Music size={18} className="text-crate-amber" />
                        )}
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="truncate font-medium text-crate-cream">{file.name}</div>
                        <div className="text-sm text-crate-muted">{formatFileSize(file.size)}</div>
                        {uploadProgress !== undefined && !isCompleted && (
                          <div className="mt-2">
                            <div className="vu-meter">
                              <div
                                className="vu-meter-fill"
                                style={{ width: `${uploadProgress}%` }}
                              />
                            </div>
                            <div className="text-xs text-crate-muted mt-1">{uploadProgress}% uploaded</div>
                          </div>
                        )}
                        {isCompleted && (
                          <div className="text-sm text-crate-success mt-1">Uploaded successfully</div>
                        )}
                      </div>
                      {!uploading && (
                        <button
                          type="button"
                          onClick={() => removeFile(index)}
                          className="p-2 rounded-lg hover:bg-crate-border text-crate-muted hover:text-crate-cream transition-colors"
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
            <label className="block text-sm font-medium text-crate-cream mb-3">Add to Crate (Optional)</label>
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
                (crates.crates || []).filter((c) => c.id !== 'unsorted').map((crate: Crate) => (
                  <option key={crate.id} value={crate.id}>
                    {crate.name}
                  </option>
                ))
              )}
            </select>
            <p className="text-xs text-crate-subtle mt-2">
              Files will be added to the selected crate. If no crate is selected, they'll go to the unsorted collection.
            </p>
          </div>

          {/* Upload Button */}
          <div className="flex justify-end pt-2">
            <button
              type="submit"
              disabled={files.length === 0 || uploading}
              className="btn btn-primary px-8"
            >
              {uploading ? (
                <span className="flex items-center gap-2">
                  <Disc className="vinyl-spinning" size={18} />
                  Uploading...
                </span>
              ) : (
                `Upload ${files.length} File${files.length !== 1 ? 's' : ''}`
              )}
            </button>
          </div>
        </form>
      </div>

      {/* Help Text */}
      <div className="card p-5">
        <h3 className="font-display font-semibold text-crate-cream mb-3">Upload Tips</h3>
        <ul className="text-sm text-crate-muted space-y-2">
          <li className="flex items-start gap-2">
            <span className="text-crate-amber">•</span>
            You can upload multiple files at once by selecting them or dragging and dropping
          </li>
          <li className="flex items-start gap-2">
            <span className="text-crate-amber">•</span>
            Supported formats: MP3, WAV, FLAC, AAC, OGG, and more
          </li>
          <li className="flex items-start gap-2">
            <span className="text-crate-amber">•</span>
            Files without crate assignment will appear in the "Unsorted" collection
          </li>
          <li className="flex items-start gap-2">
            <span className="text-crate-amber">•</span>
            You can organize uploaded files into crates from the Library page
          </li>
        </ul>
      </div>
    </div>
  )
}
