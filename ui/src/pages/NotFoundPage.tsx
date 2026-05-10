import { Link } from 'react-router-dom'

export function NotFoundPage() {
  return (
    <div className="min-h-[calc(100vh-48px)] flex items-center justify-center px-4">
      <div className="terminal-box max-w-sm w-full">
        <div className="terminal-title-bar">
          <span className="terminal-dot" style={{ background: '#f85149' }} />
          <span className="terminal-dot" style={{ background: '#d29922' }} />
          <span className="terminal-dot" style={{ background: '#3fb950' }} />
          <span className="ml-2 text-xs font-mono text-muted">404</span>
        </div>
        <div className="p-6">
          <p className="font-mono text-accent-red text-sm mb-2">$ resolve: not found</p>
          <p className="font-mono text-2xl text-primary font-bold mb-1">404</p>
          <p className="text-sm text-secondary mb-6">
            The page you're looking for doesn't exist.
          </p>
          <Link
            to="/"
            className="inline-flex items-center gap-2 text-sm text-accent-blue hover:underline no-underline font-mono"
          >
            ← Go home
          </Link>
        </div>
      </div>
    </div>
  )
}
