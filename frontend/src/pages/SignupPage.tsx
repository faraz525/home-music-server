import { FormEvent, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { useAuth } from '../state/auth'

export function SignupPage() {
  const { signup } = useAuth()
  const nav = useNavigate()
  const [params] = useSearchParams()
  const [invite, setInvite] = useState(params.get('invite') || '')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setLoading(true)
    setError('')
    try {
      await signup(invite, email, password)
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
        <div className="text-sm text-[#A1A1A1] mb-6">Enter invite code to continue</div>
        <form className="space-y-4" onSubmit={onSubmit}>
          <div>
            <label className="block text-sm mb-1">Invite code</label>
            <input className="input w-full" value={invite} onChange={(e) => setInvite(e.target.value)} required />
          </div>
          <div>
            <label className="block text-sm mb-1">Email</label>
            <input className="input w-full" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
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

