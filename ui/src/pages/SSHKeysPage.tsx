import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Key, Trash2 } from 'lucide-react'
import { format } from 'date-fns'
import { apiClient, unwrap } from '../api/client'
import { useAuth } from '../hooks/useAuth'
import { Button } from '../components/ui/Button'
import { Input, Textarea } from '../components/ui/Input'
import { Spinner } from '../components/ui/Spinner'
import type { SSHKey } from '../types/api'

async function listKeys(): Promise<SSHKey[]> {
  return unwrap(await apiClient.get<{ data: SSHKey[] }>('/user/keys')) ?? []
}

async function addKey(title: string, public_key: string): Promise<SSHKey> {
  return unwrap(await apiClient.post<{ data: SSHKey }>('/user/keys', { title, public_key }))
}

async function deleteKey(id: string): Promise<void> {
  await apiClient.delete(`/user/keys/${id}`)
}

export function SSHKeysPage() {
  const { isAuthenticated } = useAuth()
  const navigate = useNavigate()
  const qc = useQueryClient()

  if (!isAuthenticated) {
    navigate('/login', { replace: true })
    return null
  }

  const { data: keys = [], isLoading } = useQuery<SSHKey[]>({
    queryKey: ['ssh-keys'],
    queryFn: listKeys,
  })

  const [title, setTitle] = useState('')
  const [publicKey, setPublicKey] = useState('')
  const [addError, setAddError] = useState<string | null>(null)

  const addMutation = useMutation({
    mutationFn: () => addKey(title, publicKey),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['ssh-keys'] })
      setTitle('')
      setPublicKey('')
      setAddError(null)
    },
    onError: (err: unknown) => {
      const e = err as { response?: { data?: { error?: string } } }
      setAddError(e.response?.data?.error ?? 'Failed to add key')
    },
  })

  const [deletingId, setDeletingId] = useState<string | null>(null)
  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteKey(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['ssh-keys'] })
      setDeletingId(null)
    },
  })

  return (
    <div className="max-w-3xl mx-auto px-4 py-6">
      <div className="mb-6">
        <h1 className="font-mono text-base text-primary font-semibold flex items-center gap-2">
          <Key size={16} /> SSH Keys
        </h1>
        <p className="text-sm text-secondary mt-1">
          SSH keys let you push to GForce using your git client without entering a token every time.
          Use your token as the git password when adding a key.
        </p>
      </div>

      {/* Add key form */}
      <div className="border border-line p-4 mb-6 space-y-3">
        <h2 className="font-mono text-xs text-secondary font-semibold uppercase tracking-wide">Add SSH Key</h2>
        {addError && <div className="border border-accent-red p-2 text-accent-red text-xs font-mono">{addError}</div>}
        <Input label="Title" value={title} onChange={(e) => setTitle(e.target.value)} placeholder="e.g. my laptop" />
        <Textarea
          label="Public key"
          value={publicKey}
          onChange={(e) => setPublicKey(e.target.value)}
          placeholder="ssh-ed25519 AAAA..."
          rows={4}
          className="font-mono text-xs"
        />
        <Button
          variant="primary"
          size="sm"
          loading={addMutation.isPending}
          disabled={!title.trim() || !publicKey.trim()}
          onClick={() => addMutation.mutate()}
          className="font-mono"
        >
          Add SSH Key
        </Button>
      </div>

      {/* Key list */}
      {isLoading ? (
        <div className="flex justify-center py-8"><Spinner size="md" className="text-secondary" /></div>
      ) : keys.length === 0 ? (
        <div className="border border-dashed border-line p-8 text-center">
          <p className="font-mono text-sm text-secondary">No SSH keys added yet.</p>
        </div>
      ) : (
        <div className="space-y-3">
          {keys.map((key) => (
            <div key={key.id} className="border border-line p-4 flex items-start justify-between gap-4">
              <div className="min-w-0">
                <p className="font-mono text-sm text-primary font-medium">{key.title}</p>
                <p className="font-mono text-xs text-secondary mt-0.5 truncate">{key.fingerprint}</p>
                <div className="flex items-center gap-4 mt-1 text-xs text-muted">
                  <span>Added {format(new Date(key.created_at), 'MMM d, yyyy')}</span>
                  <span>
                    Last used: {key.last_used_at
                      ? format(new Date(key.last_used_at), 'MMM d, yyyy')
                      : 'Never'}
                  </span>
                </div>
              </div>
              <Button
                variant="danger"
                size="sm"
                loading={deleteMutation.isPending && deletingId === key.id}
                onClick={() => { setDeletingId(key.id); deleteMutation.mutate(key.id) }}
                className="flex-shrink-0"
              >
                <Trash2 size={13} />
              </Button>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
