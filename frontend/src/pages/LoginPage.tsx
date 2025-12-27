import { FormEvent, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../state/auth'
import { Disc } from 'lucide-react'

export function LoginPage() {
  const { login } = useAuth()
  const nav = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setLoading(true)
    setError('')
    try {
      await login(email, password)
      nav('/')
    } catch (e: any) {
      setError(e?.response?.data?.error?.message || 'Login failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen grid place-items-center p-6 bg-crate-black relative overflow-hidden">
      {/* Background decoration */}
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div
          className="absolute -top-1/4 -left-1/4 w-1/2 h-1/2 rounded-full opacity-20"
          style={{
            background: 'radial-gradient(circle, rgba(229, 160, 0, 0.3) 0%, transparent 70%)',
          }}
        />
        <div
          className="absolute -bottom-1/4 -right-1/4 w-1/2 h-1/2 rounded-full opacity-10"
          style={{
            background: 'radial-gradient(circle, rgba(0, 212, 255, 0.3) 0%, transparent 70%)',
          }}
        />
      </div>

      <div className="card w-full max-w-md p-8 relative z-10 animate-slide-up">
        {/* Logo */}
        <div className="flex items-center justify-center gap-3 mb-8">
          <div className="p-3 rounded-xl bg-crate-amber/10">
            <Disc className="text-crate-amber" size={32} />
          </div>
          <span className="text-2xl font-display font-bold text-crate-cream">CrateDrop</span>
        </div>

        <div className="text-center mb-8">
          <h1 className="text-2xl font-display font-bold text-crate-cream mb-2">Welcome back</h1>
          <p className="text-crate-muted">Sign in to your library</p>
        </div>

        <form className="space-y-5" onSubmit={onSubmit}>
          <div>
            <label className="block text-sm font-medium text-crate-cream mb-2">Email</label>
            <input
              className="input w-full"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="you@example.com"
              required
              autoFocus
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-crate-cream mb-2">Password</label>
            <input
              className="input w-full"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              required
            />
          </div>

          {error && (
            <div className="p-3 rounded-xl bg-crate-danger/10 border border-crate-danger/20 text-crate-danger text-sm">
              {error}
            </div>
          )}

          <button className="btn btn-primary w-full" disabled={loading}>
            {loading ? (
              <span className="flex items-center gap-2">
                <Disc className="vinyl-spinning" size={18} />
                Signing in...
              </span>
            ) : (
              'Sign in'
            )}
          </button>
        </form>

        <div className="mt-6 pt-6 border-t border-crate-border text-center">
          <span className="text-sm text-crate-muted">
            Don't have an account?{' '}
            <Link className="link" to="/signup">Sign up</Link>
          </span>
        </div>
      </div>
    </div>
  )
}
