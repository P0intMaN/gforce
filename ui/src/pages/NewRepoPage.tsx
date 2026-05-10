import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { createRepo } from '../api/repos'
import { useAuth } from '../hooks/useAuth'
import { Button } from '../components/ui/Button'
import { Input } from '../components/ui/Input'
import { Check, X } from 'lucide-react'

const SLUG_RE = /^[a-z0-9]([a-z0-9-]*[a-z0-9])?$/

export function NewRepoPage() {
  const navigate = useNavigate()
  const { user } = useAuth()

  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [isPrivate, setIsPrivate] = useState(false)
  const [initRepo, setInitRepo] = useState(true)
  const [defaultBranch, setDefaultBranch] = useState('main')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const nameValid = SLUG_RE.test(name)
  const nameError = name && !nameValid ? 'Only lowercase letters, numbers, and hyphens' : null

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!nameValid) return
    setError(null)
    setLoading(true)
    try {
      const repo = await createRepo({
        name,
        description,
        is_private: isPrivate,
        init: initRepo,
        default_branch: defaultBranch,
      })
      navigate(`/${repo.owner.username}/${repo.name}`)
    } catch (err: unknown) {
      const axErr = err as { response?: { data?: { error?: string } } }
      setError(axErr.response?.data?.error ?? 'Failed to create repository')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="max-w-2xl mx-auto px-4 py-8">
      <h1 className="font-mono text-lg text-primary mb-1">Create a new repository</h1>
      <p className="text-sm text-secondary mb-8">
        A repository contains all project files, including the revision history.
      </p>

      {error && (
        <div className="mb-6 p-3 border border-accent-red bg-[rgba(248,81,73,0.1)] text-accent-red text-sm">
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Owner / name */}
        <div>
          <label className="block text-xs text-secondary mb-1.5 font-mono">
            Owner / Repository name
          </label>
          <div className="flex items-center gap-2">
            <div className="h-8 px-3 flex items-center bg-elevated border border-line text-primary font-mono text-sm flex-shrink-0">
              {user?.username ?? '—'}
            </div>
            <span className="text-secondary">/</span>
            <div className="flex-1 relative">
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value.toLowerCase())}
                placeholder="repository-name"
                required
                className={`w-full h-8 bg-base border px-3 font-mono text-sm text-primary placeholder:text-muted outline-none focus:border-accent-blue transition-colors ${
                  nameError ? 'border-accent-red' : 'border-line'
                }`}
              />
              {name && (
                <span className="absolute right-2 top-1/2 -translate-y-1/2">
                  {nameValid ? (
                    <Check size={13} className="text-accent-green" />
                  ) : (
                    <X size={13} className="text-accent-red" />
                  )}
                </span>
              )}
            </div>
          </div>
          {nameError && <p className="mt-1 text-xs text-accent-red font-mono">{nameError}</p>}
        </div>

        {/* Description */}
        <Input
          label="Description (optional)"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="A short description of your repository"
        />

        {/* Visibility */}
        <div>
          <label className="block text-xs text-secondary mb-3 font-mono">Visibility</label>
          <div className="space-y-2">
            {[
              { value: false, label: 'Public', desc: 'Anyone can see this repository' },
              { value: true, label: 'Private', desc: 'Only you can see this repository' },
            ].map((opt) => (
              <label
                key={opt.label}
                className={`flex items-start gap-3 p-3 border cursor-pointer transition-colors ${
                  isPrivate === opt.value ? 'border-accent-blue bg-elevated' : 'border-line hover:border-[#3a3d45]'
                }`}
              >
                <input
                  type="radio"
                  checked={isPrivate === opt.value}
                  onChange={() => setIsPrivate(opt.value)}
                  className="mt-0.5 accent-[#58a6ff]"
                />
                <div>
                  <p className="text-sm text-primary font-medium">{opt.label}</p>
                  <p className="text-xs text-secondary">{opt.desc}</p>
                </div>
              </label>
            ))}
          </div>
        </div>

        {/* Initialize */}
        <div className="border-t border-line pt-6">
          <label className="flex items-start gap-3 cursor-pointer">
            <input
              type="checkbox"
              checked={initRepo}
              onChange={(e) => setInitRepo(e.target.checked)}
              className="mt-0.5 accent-[#58a6ff]"
            />
            <div>
              <p className="text-sm text-primary">Add a README file</p>
              <p className="text-xs text-secondary">
                This will create an initial commit with a README.md
              </p>
            </div>
          </label>
        </div>

        {/* Default branch */}
        <Input
          label="Default branch"
          value={defaultBranch}
          onChange={(e) => setDefaultBranch(e.target.value)}
          monospace
        />

        <div className="border-t border-line pt-6">
          <Button
            type="submit"
            variant="primary"
            size="lg"
            loading={loading}
            disabled={!nameValid}
            className="font-mono"
          >
            Create repository
          </Button>
        </div>
      </form>
    </div>
  )
}
