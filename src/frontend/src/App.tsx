import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import type { ReactNode } from 'react'
import { AuthProvider } from './context/AuthContext'
import { useAuth } from './context/useAuth'
import { DevHealthPage } from './pages/dev-health/DevHealthPage'
import { LoginPage } from './pages/login/LoginPage'
import { OperationsPage } from './pages/operations/OperationsPage'

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<LoginPage />} />
          <Route
            path="/operations"
            element={
              <RequireAuth>
                <OperationsPage />
              </RequireAuth>
            }
          />
          <Route path="/dev/health" element={<DevHealthPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  )
}

function RequireAuth({ children }: { children: ReactNode }) {
  const auth = useAuth()

  if (auth.status === 'loading') {
    return <main className="ops-auth-loading">同步中</main>
  }

  if (auth.status !== 'authenticated') {
    return <Navigate to="/" replace />
  }

  return <>{children}</>
}
