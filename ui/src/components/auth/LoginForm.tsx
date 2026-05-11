import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { login, getCurrentUser } from '../../api/auth'
import { useAuthStore } from '../../store/auth'
import { Button } from '../ui/Button'

export function LoginForm() {
  const navigate = useNavigate()
  const { login: storeLogin } = useAuthStore()
  const [loginVal, setLoginVal] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      const tokenData = await login(loginVal, password)
      const user = await getCurrentUser()
      storeLogin(tokenData.token, user)
      navigate('/', { replace: true })
    } catch {
      setError('Invalid credentials. Check your username/email and password.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="terminal-box max-w-sm w-full">
      {/* Title bar */}
      <div className="terminal-title-bar">
        <span className="terminal-dot" style={{ background: '#f85149' }} />
        <span className="terminal-dot" style={{ background: '#d29922' }} />
        <span className="terminal-dot" style={{ background: '#3fb950' }} />
        <span className="ml-2 text-xs font-mono text-muted">gforce login</span>
      </div>

      {/* Form body */}
      <div className="p-6">
        <p className="font-mono text-accent-green text-sm mb-6">
          $ gforce auth login
        </p>

        {error && (
          <div className="mb-4 p-3 border border-accent-red bg-[rgba(248,81,73,0.1)] text-accent-red text-xs font-mono">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <div>
            <label className="block text-xs text-secondary font-mono mb-1.5">
              username or email
            </label>
            <div className="flex items-center border border-line focus-within:border-accent-blue transition-colors">
              <span className="px-2 text-accent-green font-mono text-sm select-none">›</span>
              <input
                type="text"
                value={loginVal}
                onChange={(e) => setLoginVal(e.target.value)}
                className="flex-1 bg-transparent text-primary font-mono text-sm py-2 pr-3 outline-none placeholder:text-muted"
                placeholder="_"
                required
                autoFocus
              />
            </div>
          </div>

          <div>
            <label className="block text-xs text-secondary font-mono mb-1.5">
              password
            </label>
            <div className="flex items-center border border-line focus-within:border-accent-blue transition-colors">
              <span className="px-2 text-accent-green font-mono text-sm select-none">›</span>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="flex-1 bg-transparent text-primary font-mono text-sm py-2 pr-3 outline-none placeholder:text-muted"
                placeholder="••••••••"
                required
              />
            </div>
          </div>

          <Button
            type="submit"
            variant="primary"
            size="md"
            loading={loading}
            className="mt-2 font-mono"
          >
            authenticate
          </Button>
        </form>

        <p className="mt-4 text-xs text-muted font-mono text-center">
          no account?{' '}
          <Link to="/register" className="text-accent-blue hover:underline">
            register →
          </Link>
        </p>
      </div>
    </div>
  )
}
