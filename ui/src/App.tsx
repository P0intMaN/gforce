import { useEffect, useRef } from 'react'
import { Routes, Route } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { useAuthRehydration } from './hooks/useAuthRehydration'
import { useAuthStore } from './store/auth'
import { Spinner } from './components/ui/Spinner'
import { AppShell } from './components/layout/AppShell'
import { HomePage } from './pages/HomePage'
import { LoginPage } from './pages/LoginPage'
import { RegisterPage } from './pages/RegisterPage'
import { NewRepoPage } from './pages/NewRepoPage'
import { RepoPage } from './pages/RepoPage'
import { RepoFilePage } from './pages/RepoFilePage'
import { RepoCommitsPage } from './pages/RepoCommitsPage'
import { RepoSettingsPage } from './pages/RepoSettingsPage'
import { SettingsPage } from './pages/SettingsPage'
import { SSHKeysPage } from './pages/SSHKeysPage'
import { UserProfilePage } from './pages/UserProfilePage'
import { NotFoundPage } from './pages/NotFoundPage'

export function App() {
  // Single source of truth for auth readiness.
  // Blocks ALL route rendering until localStorage is read and token is
  // validated with the server. No child component ever needs to do this.
  const isReady = useAuthRehydration()

  // Clear TanStack Query cache when the user logs out.
  const queryClient = useQueryClient()
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const prevAuthRef = useRef(isAuthenticated)
  useEffect(() => {
    if (prevAuthRef.current && !isAuthenticated) {
      queryClient.clear()
    }
    prevAuthRef.current = isAuthenticated
  }, [isAuthenticated, queryClient])

  if (!isReady) {
    return (
      <div style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        height: '100vh',
        background: '#0a0b0d',
      }}>
        <Spinner size="md" className="text-secondary" />
      </div>
    )
  }

  return (
    <Routes>
      <Route element={<AppShell />}>
        <Route index element={<HomePage />} />
        {/* Auth */}
        <Route path="login" element={<LoginPage />} />
        <Route path="register" element={<RegisterPage />} />
        <Route path="new" element={<NewRepoPage />} />
        {/* User settings */}
        <Route path="settings" element={<SettingsPage />} />
        <Route path="settings/keys" element={<SSHKeysPage />} />
        {/* Repo — specific routes BEFORE the catch-all /:owner/:repo */}
        <Route path=":owner/:repo/settings" element={<RepoSettingsPage />} />
        <Route path=":owner/:repo/blob/:ref/*" element={<RepoFilePage />} />
        <Route path=":owner/:repo/tree/:ref/*" element={<RepoPage />} />
        <Route path=":owner/:repo/commits/:ref" element={<RepoCommitsPage />} />
        <Route path=":owner/:repo" element={<RepoPage />} />
        {/* User profile — AFTER all specific two-segment routes */}
        <Route path=":username" element={<UserProfilePage />} />
        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  )
}
