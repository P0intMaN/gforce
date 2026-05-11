import { useState, useEffect } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { updateProfile } from '../api/auth'
import { useAuth } from '../hooks/useAuth'
import { useAuthStore } from '../store/auth'
import { Button } from '../components/ui/Button'
import { Input, Textarea } from '../components/ui/Input'

export function SettingsPage() {
  const { isAuthenticated, user } = useAuth()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const { login, token } = useAuthStore()

  if (!isAuthenticated) {
    navigate('/login', { replace: true })
    return null
  }

  const [displayName, setDisplayName] = useState(user?.display_name ?? '')
  const [bio, setBio] = useState(user?.bio ?? '')
  const [avatarUrl, setAvatarUrl] = useState(user?.avatar_url ?? '')
  const [saveOk, setSaveOk] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)

  useEffect(() => {
    if (user) {
      setDisplayName(user.display_name ?? '')
      setBio(user.bio ?? '')
      setAvatarUrl(user.avatar_url ?? '')
    }
  }, [user])

  const saveMutation = useMutation({
    mutationFn: () => updateProfile({
      display_name: displayName || undefined,
      bio: bio || undefined,
      avatar_url: avatarUrl || undefined,
    }),
    onSuccess: (updated) => {
      if (token) login(token, updated)
      qc.invalidateQueries({ queryKey: ['current-user'] })
      setSaveOk(true)
      setSaveError(null)
      setTimeout(() => setSaveOk(false), 3000)
    },
    onError: (err: unknown) => {
      const e = err as { response?: { data?: { error?: string } } }
      setSaveError(e.response?.data?.error ?? 'Failed to save profile')
    },
  })

  const bioCount = bio.length

  return (
    <div className="max-w-3xl mx-auto px-4 py-6">
      <h1 className="font-mono text-base text-primary font-semibold mb-6">Settings</h1>

      {/* Profile section */}
      <div className="border border-line p-6 mb-6 space-y-4">
        <h2 className="font-mono text-xs text-secondary font-semibold uppercase tracking-wide">Profile</h2>

        {saveError && <div className="border border-accent-red p-3 text-accent-red text-sm font-mono">{saveError}</div>}
        {saveOk && <div className="border border-accent-green p-3 text-accent-green text-sm font-mono">Profile saved.</div>}

        <Input label="Display name" value={displayName} onChange={(e) => setDisplayName(e.target.value)} placeholder={user?.username} />

        <div>
          <label className="block text-xs text-secondary font-mono mb-1.5">Bio</label>
          <Textarea
            value={bio}
            onChange={(e) => setBio(e.target.value)}
            placeholder="Tell people a little about yourself"
            rows={4}
            maxLength={500}
          />
          <p className={`text-xs mt-1 text-right ${bioCount > 480 ? 'text-accent-orange' : 'text-muted'}`}>
            {bioCount}/500
          </p>
        </div>

        <Input label="Avatar URL" value={avatarUrl} onChange={(e) => setAvatarUrl(e.target.value)} placeholder="https://..." />

        <Button variant="primary" size="sm" loading={saveMutation.isPending} onClick={() => saveMutation.mutate()} className="font-mono">
          Save profile
        </Button>
      </div>

      {/* Account section */}
      <div className="border border-line p-6 space-y-4">
        <h2 className="font-mono text-xs text-secondary font-semibold uppercase tracking-wide">Account</h2>

        <div>
          <label className="block text-xs text-secondary font-mono mb-1.5">Username</label>
          <div className="flex items-center gap-2">
            <div className="flex-1 h-8 px-3 border border-line bg-elevated text-secondary font-mono text-sm flex items-center">
              {user?.username}
            </div>
          </div>
          <p className="text-xs text-muted mt-1">Username changes are not supported yet.</p>
        </div>

        <div>
          <label className="block text-xs text-secondary font-mono mb-1.5">Email</label>
          <div className="h-8 px-3 border border-line bg-elevated text-secondary font-mono text-sm flex items-center">
            {user?.email}
          </div>
          <p className="text-xs text-muted mt-1">Email changes are not supported yet.</p>
        </div>

        <div className="pt-2 border-t border-line-muted">
          <Link to="/settings/keys" className="text-sm text-accent-blue hover:underline no-underline">
            Manage SSH keys →
          </Link>
        </div>
      </div>
    </div>
  )
}
