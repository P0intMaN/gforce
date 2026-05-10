import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { Plus, Terminal, GitBranch, Box } from 'lucide-react'
import { useQuery } from '@tanstack/react-query'
import { useAuth } from '../hooks/useAuth'
import { getMyRepos } from '../api/repos'
import { RepoCard } from '../components/repo/RepoCard'
import { Button } from '../components/ui/Button'
import { Spinner } from '../components/ui/Spinner'
import type { Repository } from '../types/api'

function HeroSection() {
  const navigate = useNavigate()
  return (
    <div className="min-h-[calc(100vh-48px)] flex flex-col items-center justify-center px-4 py-16">
      <div className="text-center mb-12 max-w-2xl">
        <h1 className="font-mono text-6xl font-bold text-primary mb-2 leading-none">
          G<span className="text-accent-blue">Force</span>
        </h1>
        <p className="font-mono text-lg text-secondary mt-4">
          The Git platform that speaks Kubernetes
        </p>
        <p className="text-sm text-muted mt-2">
          Every repository is a Custom Resource. Every operation is observable.
        </p>
        <div className="flex items-center justify-center gap-3 mt-8">
          <Button variant="primary" size="lg" onClick={() => navigate('/register')} className="font-mono">
            Get Started
          </Button>
          <Button variant="secondary" size="lg" onClick={() => navigate('/login')} className="font-mono">
            Sign in
          </Button>
        </div>
      </div>

      {/* Feature cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 max-w-3xl w-full">
        {[
          {
            icon: <GitBranch size={20} className="text-accent-blue" />,
            title: '$ git clone',
            desc: 'Full Git smart-HTTP server. Clone, push, fetch. Standard git tooling, no plugins.',
          },
          {
            icon: <Box size={20} className="text-accent-purple" />,
            title: 'kind: Repository',
            desc: 'Every repo is a Kubernetes CRD. Operator reconciles state. GitOps from day one.',
          },
          {
            icon: <Terminal size={20} className="text-accent-green" />,
            title: '/metrics',
            desc: 'Prometheus endpoint built in. Observe every operation. Structured logs with zap.',
          },
        ].map((card) => (
          <div key={card.title} className="terminal-box p-4">
            <div className="flex items-center gap-2 mb-3">
              {card.icon}
              <span className="font-mono text-sm text-primary">{card.title}</span>
            </div>
            <p className="text-sm text-secondary leading-relaxed">{card.desc}</p>
          </div>
        ))}
      </div>
    </div>
  )
}

function AuthenticatedHome() {
  const navigate = useNavigate()
  const [filter, setFilter] = useState('')

  const { data: repos = [], isLoading } = useQuery<Repository[]>({
    queryKey: ['my-repos'],
    queryFn: () => getMyRepos(),
  })

  const filtered = repos.filter(
    (r) =>
      r.name.toLowerCase().includes(filter.toLowerCase()) ||
      r.description?.toLowerCase().includes(filter.toLowerCase())
  )

  return (
    <div className="max-w-6xl mx-auto px-4 py-6 flex gap-6">
      {/* Left column — repos */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between mb-4">
          <h2 className="font-mono text-sm text-primary font-semibold">Repositories</h2>
          <Button
            variant="primary"
            size="sm"
            onClick={() => navigate('/new')}
            className="font-mono"
          >
            <Plus size={13} />
            New
          </Button>
        </div>

        <div className="mb-3">
          <input
            type="text"
            placeholder="Find a repository..."
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            className="w-full h-8 bg-base border border-line text-primary placeholder:text-muted font-mono text-xs px-3 outline-none focus:border-accent-blue transition-colors"
          />
        </div>

        {isLoading ? (
          <div className="flex items-center justify-center py-12">
            <Spinner size="md" className="text-secondary" />
          </div>
        ) : filtered.length === 0 ? (
          <div className="border border-dashed border-line p-8 text-center">
            <p className="font-mono text-sm text-secondary">
              {repos.length === 0 ? 'No repositories yet.' : 'No results.'}
            </p>
            {repos.length === 0 && (
              <Button
                variant="primary"
                size="sm"
                onClick={() => navigate('/new')}
                className="mt-4 font-mono"
              >
                Create your first repository
              </Button>
            )}
          </div>
        ) : (
          <div>
            {filtered.map((repo) => (
              <RepoCard key={repo.id} repo={repo} />
            ))}
          </div>
        )}
      </div>

      {/* Right column — terminal activity */}
      <div className="w-72 flex-shrink-0 hidden lg:block">
        <div className="terminal-box sticky top-20">
          <div className="terminal-title-bar">
            <span className="terminal-dot" style={{ background: '#f85149' }} />
            <span className="terminal-dot" style={{ background: '#d29922' }} />
            <span className="terminal-dot" style={{ background: '#3fb950' }} />
            <span className="ml-2 text-xs font-mono text-muted">activity</span>
          </div>
          <div className="p-3 space-y-3">
            {repos.slice(0, 4).map((repo) => (
              <div key={repo.id} className="text-xs font-mono">
                <p className="text-accent-green">$ repo.view</p>
                <p className="text-secondary pl-2">
                  →{' '}
                  <Link
                    to={`/${repo.owner.username}/${repo.name}`}
                    className="text-accent-blue hover:underline"
                  >
                    {repo.full_name}
                  </Link>
                </p>
              </div>
            ))}
            {repos.length === 0 && (
              <p className="text-xs font-mono text-muted">
                {'// no activity yet'}
              </p>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

export function HomePage() {
  const { isAuthenticated } = useAuth()
  return isAuthenticated ? <AuthenticatedHome /> : <HeroSection />
}
