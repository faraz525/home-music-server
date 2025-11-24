import { Navigate, Route, Routes } from 'react-router-dom'
import { Layout } from './components/layout/Layout'
import { LoginPage } from './pages/LoginPage'
import { SignupPage } from './pages/SignupPage'
import { LibraryPage } from './pages/LibraryPage'
import { CratesPage } from './pages/CratesPage'
import { CommunityPage } from './pages/CommunityPage'
import { UploadPage } from './pages/UploadPage'
import { AdminPage } from './pages/AdminPage'
import { NotFound } from './pages/NotFound'
import { AuthProvider, useAuth } from './state/auth'

function PrivateRoute({ children }: { children: JSX.Element }) {
  const { isAuthenticated, ready } = useAuth()
  if (!ready) return null
  return isAuthenticated ? children : <Navigate to="/login" replace />
}

export default function App() {
  return (
    <AuthProvider>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/signup" element={<SignupPage />} />
        <Route
          path="/"
          element={
            <PrivateRoute>
              <Layout />
            </PrivateRoute>
          }
        >
          <Route index element={<LibraryPage />} />
          <Route path="/upload" element={<UploadPage />} />
          <Route path="/crates" element={<CratesPage />} />
          <Route path="/community" element={<CommunityPage />} />
          <Route path="/admin" element={<AdminPage />} />
        </Route>
        <Route path="*" element={<NotFound />} />
      </Routes>
    </AuthProvider>
  )
}

