import { Link } from 'react-router-dom'
import { Music } from 'lucide-react'

export function NotFound() {
  return (
    <div className="min-h-screen grid place-items-center p-6 bg-[#121212]">
      <div className="text-center space-y-6 max-w-md">
        <Music size={64} className="mx-auto text-[#1DB954]" />
        <h1 className="text-6xl font-bold text-[#1DB954]">404</h1>
        <div className="space-y-2">
          <h2 className="text-2xl font-bold">Page not found</h2>
          <p className="text-[#A1A1A1]">The page you're looking for doesn't exist or has been moved.</p>
        </div>
        <Link to="/" className="btn btn-primary inline-flex">
          Back to Library
        </Link>
      </div>
    </div>
  )
}

