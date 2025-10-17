import { FormEvent, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../state/auth'

export function SignupPage() {
  const { signup } = useAuth()
  const nav = useNavigate()
  const [email, setEmail] = useState('')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setLoading(true)
    setError('')
    try {
      await signup(email, username, password)
      nav('/')
    } catch (e: any) {
      setError(e?.response?.data?.error?.message || 'Signup failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen grid place-items-center p-6">
      <div className="card w-full max-w-md p-6">
        <div className="text-2xl font-extrabold mb-1">Create your account</div>
        <div className="text-sm text-[#A1A1A1] mb-6">Sign up to start building your music library</div>
        <form className="space-y-4" onSubmit={onSubmit}>
          <div>
            <label className="block text-sm mb-1">Email</label>
            <input className="input w-full" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
          </div>
          <div>
            <label className="block text-sm mb-1">Username</label>
            <input 
              className="input w-full" 
              type="text" 
              value={username} 
              onChange={(e) => setUsername(e.target.value)} 
              minLength={3}
              maxLength={30}
              pattern="[a-z0-9_-]+"
              title="Username can only contain lowercase letters, numbers, underscores, and hyphens"
              required 
            />
            <p className="text-xs text-[#A1A1A1] mt-1">3-30 characters, lowercase letters, numbers, _ or -</p>
          </div>
          <div>
            <label className="block text-sm mb-1">Password</label>
            <input className="input w-full" type="password" value={password} onChange={(e) => setPassword(e.target.value)} required />
          </div>
          {error && <div className="text-red-400 text-sm">{error}</div>}
          <button className="btn btn-primary w-full" disabled={loading}>
            {loading ? 'Creating account...' : 'Sign up'}
          </button>
        </form>
        <div className="mt-4 text-sm text-[#A1A1A1]">
          Already have an account? <Link className="link" to="/login">Sign in</Link>
        </div>
      </div>
    </div>
  )
}

