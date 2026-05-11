import { useEffect, useRef } from 'react'
import { Routes, Route } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { AppShell } from './components/layout/AppShell'
import { HomePage } from './pages/HomePage'
import { LoginPage } from './pages/LoginPage'
import { RegisterPage } from './pages/RegisterPage'
import { NewRepoPage } from './pages/NewRepoPage'
import { RepoPage } from './pages/RepoPage'
import { RepoFilePage } from './pages/RepoFilePage'
import { RepoCommitsPage } from './pages/RepoCommitsPage'
import { UserProfilePage } from './pages/UserProfilePage'
import { NotFoundPage } from './pages/NotFoundPage'
import { useAuthRehydration, useIsRehydrated } from './hooks/useAuth'
import { useAuthStore } from './store/auth'
import { Spinner } from './components/ui/Spinner'

function AuthGate({ children }: { children: React.ReactNode }) {
  const isRehydrated = useIsRehydrated()

  if (!isRehydrated) {
    return (
      <div className="min-h-screen bg-base flex items-center justify-center">
        <Spinner size="md" className="text-secondary" />
      </div>
    )
  }

  return <>{children}</>
}

export function App() {
  useAuthRehydration()

  const queryClient = useQueryClient()
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  // Track previous auth state so we detect the true→false transition.
  const prevAuthRef = useRef(isAuthenticated)

  useEffect(() => {
    if (prevAuthRef.current && !isAuthenticated) {
      // User just logged out — wipe all cached API data so stale data
      // from the previous session never shows on the next login.
      queryClient.clear()
    }
    prevAuthRef.current = isAuthenticated
  }, [isAuthenticated, queryClient])

  return (
    <AuthGate>
      <Routes>
        <Route element={<AppShell />}>
          <Route index element={<HomePage />} />
          {/* Auth */}
          <Route path="login" element={<LoginPage />} />
          <Route path="register" element={<RegisterPage />} />
          <Route path="new" element={<NewRepoPage />} />
          {/* Repo — specific routes BEFORE the catch-all /:owner/:repo */}
          <Route path=":owner/:repo/blob/:ref/*" element={<RepoFilePage />} />
          <Route path=":owner/:repo/tree/:ref/*" element={<RepoPage />} />
          <Route path=":owner/:repo/commits/:ref" element={<RepoCommitsPage />} />
          <Route path=":owner/:repo" element={<RepoPage />} />
          {/* User profile — AFTER all specific two-segment routes */}
          <Route path=":username" element={<UserProfilePage />} />
          <Route path="*" element={<NotFoundPage />} />
        </Route>
      </Routes>
    </AuthGate>
  )
}
