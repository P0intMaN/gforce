import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { register, login, getCurrentUser } from '../../api/auth'
import { useAuthStore } from '../../store/auth'
import { Button } from '../ui/Button'

const SLUG_RE = /^[a-z0-9]([a-z0-9-]*[a-z0-9])?$/

export function RegisterForm() {
  const navigate = useNavigate()
  const { login: storeLogin } = useAuthStore()
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const usernameError =
    username && !SLUG_RE.test(username)
      ? 'Only lowercase letters, numbers, and hyphens'
      : null

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (usernameError) return
    setError(null)
    setLoading(true)
    try {
      await register(username, email, password)
      const tokenData = await login(username, password)
      const user = await getCurrentUser()
      storeLogin(tokenData.token, user)
      navigate('/', { replace: true })
    } catch (err: unknown) {
      if (err && typeof err === 'object' && 'response' in err) {
        const axErr = err as { response?: { data?: { error?: string } } }
        setError(axErr.response?.data?.error ?? 'Registration failed')
      } else {
        setError('Registration failed. Try a different username or email.')
      }
    } finally {
      setLoading(false)
    }
  }

  const Field = ({
    label,
    type = 'text',
    value,
    onChange,
    fieldError,
  }: {
    label: string
    type?: string
    value: string
    onChange: (v: string) => void
    fieldError?: string | null
  }) => (
    <div>
      <label className="block text-xs text-secondary font-mono mb-1.5">{label}</label>
      <div
        className={`flex items-center border transition-colors focus-within:border-accent-blue ${
          fieldError ? 'border-accent-red' : 'border-line'
        }`}
      >
        <span className="px-2 text-accent-green font-mono text-sm select-none">›</span>
        <input
          type={type}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className="flex-1 bg-transparent text-primary font-mono text-sm py-2 pr-3 outline-none placeholder:text-muted"
          required
        />
      </div>
      {fieldError && <p className="text-xs text-accent-red mt-1 font-mono">{fieldError}</p>}
    </div>
  )

  return (
    <div className="terminal-box max-w-sm w-full">
      <div className="terminal-title-bar">
        <span className="terminal-dot" style={{ background: '#f85149' }} />
        <span className="terminal-dot" style={{ background: '#d29922' }} />
        <span className="terminal-dot" style={{ background: '#3fb950' }} />
        <span className="ml-2 text-xs font-mono text-muted">gforce register</span>
      </div>

      <div className="p-6">
        <p className="font-mono text-accent-green text-sm mb-6">
          $ gforce auth create-account
        </p>

        {error && (
          <div className="mb-4 p-3 border border-accent-red bg-[rgba(248,81,73,0.1)] text-accent-red text-xs font-mono">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <Field
            label="username"
            value={username}
            onChange={setUsername}
            fieldError={usernameError}
          />
          <Field label="email" type="email" value={email} onChange={setEmail} />
          <Field
            label="password (min 8 chars)"
            type="password"
            value={password}
            onChange={setPassword}
            fieldError={password && password.length < 8 ? 'At least 8 characters required' : null}
          />

          <Button
            type="submit"
            variant="primary"
            size="md"
            loading={loading}
            disabled={!!usernameError || password.length < 8}
            className="mt-2 font-mono"
          >
            create account
          </Button>
        </form>

        <p className="mt-4 text-xs text-muted font-mono text-center">
          have an account?{' '}
          <Link to="/login" className="text-accent-blue hover:underline">
            sign in →
          </Link>
        </p>
      </div>
    </div>
  )
}
