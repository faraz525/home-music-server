import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import axios from 'axios'

function getCookie(name: string) {
  try {
    const match = document.cookie.match(new RegExp('(?:^|; )' + name.replace(/([.$?*|{}()\[\]\\\/\+^])/g, '\\$1') + '=([^;]*)'))
    const val = match ? decodeURIComponent(match[1]) : ''
    console.log(`[Auth] getCookie ${name}:`, val ? 'found' : 'empty')
    return val
  } catch (e) {
    console.error('[Auth] getCookie error:', e)
    return ''
  }
}

function setCookie(name: string, value: string, maxAgeSeconds: number) {
  document.cookie = `${name}=${encodeURIComponent(value)}; Path=/; Max-Age=${maxAgeSeconds}`
}

type User = {
  id: string
  email: string
  role: 'admin' | 'user'
}

type AuthContextValue = {
  user: User | null
  isAuthenticated: boolean
  ready: boolean
  login: (email: string, password: string) => Promise<void>
  signup: (email: string, password: string) => Promise<void>
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined)

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [ready, setReady] = useState(false)

  console.log('[Auth] Provider render. Ready:', ready, 'User:', user?.id)

  const client = useMemo(() => {
    const instance = axios.create({ withCredentials: true })
    instance.interceptors.request.use((config) => {
      const token = getCookie('access_token')
      if (token && token.trim() !== '') {
        config.headers.Authorization = `Bearer ${token}`
      } else {
        console.warn('[Auth] No token found for request')
      }
      return config
    })
    instance.interceptors.response.use(
      (res) => res,
      async (error) => {
        if (error.response?.status === 401) {
          console.log('[Auth] 401 detected, trying refresh')
          try {
            const { data } = await axios.post('/api/auth/refresh', {}, { withCredentials: true })
            if (data?.access_token) {
              console.log('[Auth] Refresh successful')
              setCookie('access_token', data.access_token, 60 * 15)
              return instance.request(error.config)
            }
          } catch (e) {
            console.error('[Auth] Refresh failed:', e)
            setUser(null)
          }
        }
        return Promise.reject(error)
      }
    )
    return instance
  }, [])

  const fetchMe = useCallback(async () => {
    console.log('[Auth] fetchMe started')
    try {
      const { data } = await client.get('/api/me')
      console.log('[Auth] fetchMe success:', data?.user?.id)
      setUser(data?.user || null)
    } catch (e) {
      console.error('[Auth] fetchMe failed:', e)
      setUser(null)
    } finally {
      console.log('[Auth] fetchMe finally, setting ready=true')
      setReady(true)
    }
  }, [client])

  useEffect(() => {
    fetchMe()
  }, [fetchMe])

  const login = useCallback(async (email: string, password: string) => {
    const { data } = await client.post('/api/auth/login', { email, password })
    if (data?.access_token) setCookie('access_token', data.access_token, 60 * 15)
    await fetchMe()
  }, [client, fetchMe])

  const signup = useCallback(async (email: string, password: string) => {
    const { data } = await client.post('/api/auth/signup', { email, password })
    if (data?.access_token) setCookie('access_token', data.access_token, 60 * 15)
    await fetchMe()
  }, [client, fetchMe])

  const logout = useCallback(async () => {
    await client.post('/api/auth/logout')
    setUser(null)
  }, [client])

  const value: AuthContextValue = useMemo(() => ({
    user,
    isAuthenticated: !!user,
    ready,
    login,
    signup,
    logout,
  }), [user, ready, login, signup, logout])

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
