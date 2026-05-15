import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { format, formatDistanceToNow, addDays, addYears } from 'date-fns'
import { AlertTriangle, Copy, Check, Key, Trash2 } from 'lucide-react'
import { createPAT, listPATs, revokePAT } from '../api/pats'
import { useAuth } from '../hooks/useAuth'
import { Button } from '../components/ui/Button'
import { Input } from '../components/ui/Input'
import { Badge } from '../components/ui/Badge'
import { Spinner } from '../components/ui/Spinner'
import type { PersonalAccessToken, CreatePATResponse } from '../types/api'

const SCOPES = ['repo:read', 'repo:write'] as const

function expiresAtFromOption(option: string, custom: string): string | null {
  const now = new Date()
  switch (option) {
    case '30d':  return addDays(now, 30).toISOString()
    case '90d':  return addDays(now, 90).toISOString()
    case '1y':   return addYears(now, 1).toISOString()
    case 'custom': return custom ? new Date(custom).toISOString() : null
    default:     return null // never expires
  }
}

/** One-time token display — shown immediately after creation. */
function TokenReveal({ result, username }: { result: CreatePATResponse; username: string }) {
  const [copied, setCopied] = useState(false)

  async function handleCopy() {
    await navigator.clipboard.writeText(result.token)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="border-2 border-accent-orange p-4 mb-6 space-y-3">
      <div className="flex items-start gap-2">
        <AlertTriangle size={15} className="text-accent-orange flex-shrink-0 mt-0.5" />
        <p className="text-sm text-accent-orange font-medium">{result.message}</p>
      </div>

      <div className="flex items-center gap-2 bg-base border border-line px-3 py-2">
        <code className="flex-1 font-mono text-sm text-accent-green break-all">{result.token}</code>
        <button
          onClick={handleCopy}
          className="flex items-center gap-1 text-xs text-secondary hover:text-primary transition-colors flex-shrink-0"
        >
          {copied ? <><Check size={13} className="text-accent-green" /> Copied!</> : <><Copy size={13} /> Copy</>}
        </button>
      </div>

      <div className="text-xs font-mono text-secondary space-y-1 pt-1 border-t border-line-muted">
        <p className="text-muted">Use as git password:</p>
        <p><span className="text-muted">Username:</span> {username}</p>
        <p><span className="text-muted">Password:</span> {result.token}</p>
      </div>
    </div>
  )
}

export function TokensPage() {
  const { isAuthenticated, user } = useAuth()
  const navigate = useNavigate()
  const qc = useQueryClient()

  if (!isAuthenticated) {
    navigate('/login', { replace: true })
    return null
  }

  // Form state
  const [name, setName] = useState('')
  const [selectedScopes, setSelectedScopes] = useState<string[]>(['repo:read', 'repo:write'])
  const [expiration, setExpiration] = useState('never')
  const [customExpiry, setCustomExpiry] = useState('')
  const [newToken, setNewToken] = useState<CreatePATResponse | null>(null)
  const [createError, setCreateError] = useState<string | null>(null)

  const { data: tokens = [], isLoading } = useQuery<PersonalAccessToken[]>({
    queryKey: ['pats'],
    queryFn: listPATs,
  })

  const createMutation = useMutation({
    mutationFn: () => createPAT({
      name,
      scopes: selectedScopes,
      expires_at: expiresAtFromOption(expiration, customExpiry),
    }),
    onSuccess: (result) => {
      qc.invalidateQueries({ queryKey: ['pats'] })
      setNewToken(result)
      setName('')
      setSelectedScopes(['repo:read', 'repo:write'])
      setExpiration('never')
      setCustomExpiry('')
      setCreateError(null)
    },
    onError: (err: unknown) => {
      const e = err as { response?: { data?: { error?: string } } }
      setCreateError(e.response?.data?.error ?? 'Failed to create token')
    },
  })

  const [revokeConfirm, setRevokeConfirm] = useState<string | null>(null)
  const revokeMutation = useMutation({
    mutationFn: (id: string) => revokePAT(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['pats'] })
      setRevokeConfirm(null)
    },
  })

  function toggleScope(scope: string) {
    setSelectedScopes((prev) =>
      prev.includes(scope) ? prev.filter((s) => s !== scope) : [...prev, scope]
    )
  }

  return (
    <div className="max-w-3xl mx-auto px-4 py-6">
      <div className="mb-6">
        <h1 className="font-mono text-base text-primary font-semibold flex items-center gap-2">
          <Key size={16} /> Personal Access Tokens
        </h1>
        <p className="text-sm text-secondary mt-1">
          Tokens work like passwords for git operations — use your GForce username and a token as the password.
          Tokens start with <code className="font-mono text-accent-green bg-elevated px-1">gf_</code> so
          secret scanners can detect accidental exposure.
        </p>
      </div>

      {/* One-time token reveal */}
      {newToken && <TokenReveal result={newToken} username={user?.username ?? ''} />}

      {/* Create form */}
      <div className="border border-line p-4 mb-6 space-y-4">
        <h2 className="font-mono text-xs text-secondary font-semibold uppercase tracking-wide">Generate New Token</h2>

        {createError && (
          <div className="border border-accent-red p-2 text-accent-red text-xs font-mono">{createError}</div>
        )}

        <Input
          label="Token name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="e.g. my laptop, CI pipeline"
        />

        <div>
          <p className="text-xs text-secondary font-mono mb-2">Scopes</p>
          <div className="flex gap-4">
            {SCOPES.map((scope) => (
              <label key={scope} className="flex items-center gap-2 cursor-pointer text-sm text-primary">
                <input
                  type="checkbox"
                  checked={selectedScopes.includes(scope)}
                  onChange={() => toggleScope(scope)}
                  className="accent-[#3fb950]"
                />
                <code className="font-mono text-xs">{scope}</code>
              </label>
            ))}
          </div>
        </div>

        <div>
          <label className="block text-xs text-secondary font-mono mb-1.5">Expiration</label>
          <select
            value={expiration}
            onChange={(e) => setExpiration(e.target.value)}
            className="h-8 bg-base border border-line text-primary font-mono text-sm px-3 w-48"
          >
            <option value="never">No expiration</option>
            <option value="30d">30 days</option>
            <option value="90d">90 days</option>
            <option value="1y">1 year</option>
            <option value="custom">Custom date</option>
          </select>
          {expiration === 'custom' && (
            <input
              type="date"
              value={customExpiry}
              onChange={(e) => setCustomExpiry(e.target.value)}
              className="ml-2 h-8 bg-base border border-line text-primary font-mono text-sm px-3"
            />
          )}
        </div>

        <Button
          variant="primary"
          size="sm"
          loading={createMutation.isPending}
          disabled={!name.trim() || selectedScopes.length === 0}
          onClick={() => createMutation.mutate()}
          className="font-mono"
        >
          Generate token
        </Button>
      </div>

      {/* Token list */}
      {isLoading ? (
        <div className="flex justify-center py-8"><Spinner size="md" className="text-secondary" /></div>
      ) : tokens.length === 0 ? (
        <div className="border border-dashed border-line p-8 text-center">
          <p className="font-mono text-sm text-secondary">No tokens yet.</p>
          <p className="text-xs text-muted mt-1">
            Generate one to push to GForce without entering your password every time.
          </p>
        </div>
      ) : (
        <div className="space-y-3">
          {tokens.map((token) => (
            <div key={token.id} className="border border-line p-4">
              <div className="flex items-start justify-between gap-4">
                <div className="min-w-0">
                  <div className="flex items-center gap-2 flex-wrap mb-1">
                    <span className="font-mono text-sm text-primary font-medium">{token.name}</span>
                    {token.scopes.map((s) => (
                      <Badge key={s} variant="purple">
                        <code className="text-2xs">{s}</code>
                      </Badge>
                    ))}
                  </div>
                  <p className="font-mono text-xs text-secondary">{token.prefix}•••</p>
                  <div className="flex items-center gap-4 mt-1 text-xs text-muted">
                    <span>
                      Last used: {token.last_used_at
                        ? formatDistanceToNow(new Date(token.last_used_at), { addSuffix: true })
                        : 'Never used'}
                    </span>
                    <span>
                      Expires: {token.expires_at
                        ? format(new Date(token.expires_at), 'MMM d, yyyy')
                        : 'Never'}
                    </span>
                    <span>Created {format(new Date(token.created_at), 'MMM d, yyyy')}</span>
                  </div>
                </div>
                <Button
                  variant="danger"
                  size="sm"
                  onClick={() => setRevokeConfirm(token.id)}
                  className="flex-shrink-0"
                >
                  <Trash2 size={13} /> Revoke
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Revoke confirmation modal */}
      {revokeConfirm && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 px-4">
          <div className="bg-overlay border border-line w-full max-w-sm p-6">
            <h3 className="font-mono text-sm text-primary font-semibold mb-3 flex items-center gap-2">
              <AlertTriangle size={14} className="text-accent-orange" /> Revoke token?
            </h3>
            <p className="text-sm text-secondary mb-4">
              Any scripts or applications using this token will stop working. This cannot be undone.
            </p>
            <div className="flex justify-end gap-2">
              <Button variant="secondary" size="sm" onClick={() => setRevokeConfirm(null)}>Cancel</Button>
              <Button
                variant="danger"
                size="sm"
                loading={revokeMutation.isPending}
                onClick={() => revokeMutation.mutate(revokeConfirm)}
                className="font-mono"
              >
                Revoke token
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
