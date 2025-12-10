import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { cratesApi, normalizeCrateList } from '../lib/api'
import type { Crate, CrateList, CreateCrateRequest, UpdateCrateRequest } from '../types/crates'

type CratesContextValue = {
  crates: CrateList
  loading: boolean
  error: string | null
  fetchCrates: () => Promise<void>
  createCrate: (data: CreateCrateRequest) => Promise<Crate>
  updateCrate: (id: string, data: UpdateCrateRequest) => Promise<void>
  deleteCrate: (id: string) => Promise<void>
  addTracksToCrate: (crateId: string, trackIds: string[]) => Promise<void>
  removeTracksFromCrate: (crateId: string, trackIds: string[]) => Promise<void>
}

const CratesContext = createContext<CratesContextValue | undefined>(undefined)

const emptyCrateList: CrateList = { crates: [], total: 0, limit: 20, offset: 0, has_next: false }

export function CratesProvider({ children }: { children: React.ReactNode }) {
  const [crates, setCrates] = useState<CrateList>(emptyCrateList)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchCrates = useCallback(async () => {
    try {
      setError(null)
      const { data } = await cratesApi.list()
      setCrates(normalizeCrateList(data))
    } catch (err) {
      console.error('Failed to fetch crates:', err)
      setError('Failed to fetch crates')
      setCrates(emptyCrateList)
    } finally {
      setLoading(false)
    }
  }, [])

  // Initial fetch on mount
  useEffect(() => {
    fetchCrates()
  }, [fetchCrates])

  const createCrate = useCallback(async (data: CreateCrateRequest): Promise<Crate> => {
    const { data: responseData } = await cratesApi.create(data)
    // Refresh crates list after creation
    await fetchCrates()
    return responseData
  }, [fetchCrates])

  const updateCrate = useCallback(async (id: string, data: UpdateCrateRequest) => {
    await cratesApi.update(id, data)
    // Refresh crates list after update
    await fetchCrates()
  }, [fetchCrates])

  const deleteCrate = useCallback(async (id: string) => {
    await cratesApi.delete(id)
    // Refresh crates list after deletion
    await fetchCrates()
  }, [fetchCrates])

  const addTracksToCrate = useCallback(async (crateId: string, trackIds: string[]) => {
    await cratesApi.addTracks(crateId, trackIds)
    // Refresh crates list to update any counts/metadata
    await fetchCrates()
  }, [fetchCrates])

  const removeTracksFromCrate = useCallback(async (crateId: string, trackIds: string[]) => {
    await cratesApi.removeTracks(crateId, trackIds)
    // Refresh crates list to update any counts/metadata
    await fetchCrates()
  }, [fetchCrates])

  const value: CratesContextValue = useMemo(() => ({
    crates,
    loading,
    error,
    fetchCrates,
    createCrate,
    updateCrate,
    deleteCrate,
    addTracksToCrate,
    removeTracksFromCrate,
  }), [crates, loading, error, fetchCrates, createCrate, updateCrate, deleteCrate, addTracksToCrate, removeTracksFromCrate])

  return <CratesContext.Provider value={value}>{children}</CratesContext.Provider>
}

export function useCrates() {
  const ctx = useContext(CratesContext)
  if (!ctx) throw new Error('useCrates must be used within CratesProvider')
  return ctx
}

