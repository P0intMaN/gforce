import { useState, useEffect } from 'react'
import { useParams, useNavigate, Navigate, Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { AlertTriangle } from 'lucide-react'
import { apiClient, unwrap } from '../api/client'
import { deleteRepo } from '../api/repos'
import { useAuth } from '../hooks/useAuth'
import { Button } from '../components/ui/Button'
import { Input, Textarea } from '../components/ui/Input'
import { Spinner } from '../components/ui/Spinner'
import type { Repository } from '../types/api'

async function getRepo(owner: string, repo: string): Promise<Repository> {
  return unwrap(await apiClient.get<{ data: Repository }>(`/repos/${owner}/${repo}`))
}

async function updateRepo(
  owner: string,
  repo: string,
  body: { name?: string; description?: string; is_private?: boolean; default_branch?: string }
): Promise<Repository> {
  return unwrap(await apiClient.patch<{ data: Repository }>(`/repos/${owner}/${repo}`, body))
}

export function RepoSettingsPage() {
  const { owner = '', repo: repoName = '' } = useParams<{ owner: string; repo: string }>()
  const navigate = useNavigate()
  const { user } = useAuth()
  const qc = useQueryClient()

  const { data: repo, isLoading } = useQuery<Repository>({
    queryKey: ['repo', owner, repoName],
    queryFn: () => getRepo(owner, repoName),
  })

  const [activeSection, setActiveSection] = useState<'general' | 'danger'>('general')

  // General form state
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [isPrivate, setIsPrivate] = useState(false)
  const [defaultBranch, setDefaultBranch] = useState('')
  const [saveError, setSaveError] = useState<string | null>(null)
  const [saveOk, setSaveOk] = useState(false)

  useEffect(() => {
    if (repo) {
      setName(repo.name)
      setDescription(repo.description ?? '')
      setIsPrivate(repo.is_private)
      setDefaultBranch(repo.default_branch)
    }
  }, [repo])

  const saveMutation = useMutation({
    mutationFn: () => updateRepo(owner, repoName, { name, description, is_private: isPrivate, default_branch: defaultBranch }),
    onSuccess: (updated) => {
      qc.invalidateQueries({ queryKey: ['repo', owner, repoName] })
      setSaveOk(true)
      setSaveError(null)
      setTimeout(() => setSaveOk(false), 3000)
      // If name changed, navigate to new URL
      if (updated.name !== repoName) {
        navigate(`/${owner}/${updated.name}/settings`, { replace: true })
      }
    },
    onError: (err: unknown) => {
      const e = err as { response?: { data?: { error?: string } } }
      setSaveError(e.response?.data?.error ?? 'Failed to save settings')
    },
  })

  // Delete flow
  const [deleteConfirm, setDeleteConfirm] = useState('')
  const [showDeleteModal, setShowDeleteModal] = useState(false)
  const deleteMutation = useMutation({
    mutationFn: () => deleteRepo(owner, repoName),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['my-repos'] })
      navigate(`/${user?.username ?? owner}`, { replace: true })
    },
  })

  if (isLoading) return <div className="flex justify-center py-12"><Spinner size="md" className="text-secondary" /></div>
  if (!repo) return <div className="max-w-4xl mx-auto px-4 py-8"><p className="text-accent-red font-mono text-sm">Repository not found.</p></div>

  // Redirect non-owners
  if (user && repo.owner.id !== user.id) {
    return <Navigate to={`/${owner}/${repoName}`} replace />
  }

  const sidebar = [
    { id: 'general', label: 'General' },
    { id: 'danger', label: 'Danger Zone' },
  ] as const

  return (
    <div className="max-w-4xl mx-auto px-4 py-6 flex gap-6">
      {/* Sidebar */}
      <nav className="w-48 flex-shrink-0">
        {sidebar.map((s) => (
          <button
            key={s.id}
            onClick={() => setActiveSection(s.id)}
            className={`w-full text-left px-3 py-2 text-sm font-mono transition-colors ${
              activeSection === s.id
                ? 'bg-elevated text-primary border-l-2 border-accent-orange'
                : 'text-secondary hover:text-primary hover:bg-elevated border-l-2 border-transparent'
            }`}
          >
            {s.label}
          </button>
        ))}
      </nav>

      {/* Content */}
      <div className="flex-1 min-w-0">
        <div className="mb-4 flex items-center gap-2 text-sm">
          <Link to={`/${owner}/${repoName}`} className="text-accent-blue hover:underline no-underline">{owner}/{repoName}</Link>
          <span className="text-muted">/</span>
          <span className="text-primary">Settings</span>
        </div>

        {activeSection === 'general' && (
          <div className="space-y-6">
            <h1 className="font-mono text-base text-primary font-semibold">General</h1>

            {saveError && <div className="border border-accent-red p-3 text-accent-red text-sm font-mono">{saveError}</div>}
            {saveOk && <div className="border border-accent-green p-3 text-accent-green text-sm font-mono">Settings saved.</div>}

            <Input label="Repository name" value={name} onChange={(e) => setName(e.target.value)} monospace />
            <Textarea label="Description" value={description} onChange={(e) => setDescription(e.target.value)} rows={3} />
            <Input label="Default branch" value={defaultBranch} onChange={(e) => setDefaultBranch(e.target.value)} monospace />

            <div>
              <p className="text-xs text-secondary font-mono mb-2">Visibility</p>
              <div className="space-y-2">
                {[
                  { value: false, label: 'Public', desc: 'Anyone can see this repository' },
                  { value: true, label: 'Private', desc: 'Only you can see this repository' },
                ].map((opt) => (
                  <label key={opt.label} className={`flex items-start gap-3 p-3 border cursor-pointer transition-colors ${isPrivate === opt.value ? 'border-accent-blue bg-elevated' : 'border-line hover:border-[#3a3d45]'}`}>
                    <input type="radio" checked={isPrivate === opt.value} onChange={() => setIsPrivate(opt.value)} className="mt-0.5" />
                    <div>
                      <p className="text-sm text-primary font-medium">{opt.label}</p>
                      <p className="text-xs text-secondary">{opt.desc}</p>
                    </div>
                  </label>
                ))}
              </div>
            </div>

            <Button variant="primary" loading={saveMutation.isPending} onClick={() => saveMutation.mutate()} className="font-mono">
              Save changes
            </Button>
          </div>
        )}

        {activeSection === 'danger' && (
          <div className="border border-accent-red p-6 space-y-4">
            <h2 className="font-mono text-sm text-accent-red font-semibold flex items-center gap-2">
              <AlertTriangle size={14} /> Danger Zone
            </h2>
            <div className="flex items-start justify-between gap-4">
              <div>
                <p className="text-sm text-primary font-medium">Delete this repository</p>
                <p className="text-xs text-secondary">Once deleted, there is no going back. All git data will be lost.</p>
              </div>
              <Button variant="danger" size="sm" onClick={() => setShowDeleteModal(true)} className="flex-shrink-0 font-mono">
                Delete repository
              </Button>
            </div>
          </div>
        )}
      </div>

      {/* Delete confirmation modal */}
      {showDeleteModal && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 px-4">
          <div className="bg-overlay border border-line w-full max-w-md p-6">
            <h3 className="font-mono text-sm text-primary font-semibold mb-3">Delete {owner}/{repoName}</h3>
            <p className="text-sm text-secondary mb-4">
              This action <span className="text-accent-red font-medium">cannot be undone</span>. Type the repository name to confirm.
            </p>
            <input
              type="text"
              value={deleteConfirm}
              onChange={(e) => setDeleteConfirm(e.target.value)}
              placeholder={repoName}
              className="w-full h-8 bg-base border border-line text-primary font-mono text-sm px-3 mb-4"
            />
            <div className="flex justify-end gap-2">
              <Button variant="secondary" size="sm" onClick={() => { setShowDeleteModal(false); setDeleteConfirm('') }}>Cancel</Button>
              <Button
                variant="danger"
                size="sm"
                disabled={deleteConfirm !== repoName}
                loading={deleteMutation.isPending}
                onClick={() => deleteMutation.mutate()}
                className="font-mono"
              >
                I understand, delete it
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
