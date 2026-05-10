import { Routes, Route } from 'react-router-dom'
import { AppShell } from './components/layout/AppShell'
import { HomePage } from './pages/HomePage'
import { LoginPage } from './pages/LoginPage'
import { RegisterPage } from './pages/RegisterPage'
import { NewRepoPage } from './pages/NewRepoPage'
import { RepoPage } from './pages/RepoPage'
import { RepoFilePage } from './pages/RepoFilePage'
import { RepoCommitsPage } from './pages/RepoCommitsPage'
import { NotFoundPage } from './pages/NotFoundPage'
import { useAuthRehydration } from './hooks/useAuth'

export function App() {
  useAuthRehydration()

  return (
    <Routes>
      <Route element={<AppShell />}>
        <Route index element={<HomePage />} />
        <Route path="login" element={<LoginPage />} />
        <Route path="register" element={<RegisterPage />} />
        <Route path="new" element={<NewRepoPage />} />
        <Route path=":owner/:repo" element={<RepoPage />} />
        <Route path=":owner/:repo/blob/:ref/*" element={<RepoFilePage />} />
        <Route path=":owner/:repo/commits/:ref" element={<RepoCommitsPage />} />
        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  )
}
