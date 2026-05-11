import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Calendar, GitBranch } from 'lucide-react'
import { format } from 'date-fns'
import { Avatar } from '../components/ui/Avatar'
import { RepoCard } from '../components/repo/RepoCard'
import { Spinner } from '../components/ui/Spinner'
import { EmptyState } from '../components/ui/EmptyState'
import { getReposByUser } from '../api/repos'
import { apiClient, unwrap } from '../api/client'
import type { User, Repository } from '../types/api'

async function getUserProfile(username: string): Promise<User> {
  return unwrap(await apiClient.get<{ data: User }>(`/users/${username}`))
}

export function UserProfilePage() {
  const { username = '' } = useParams<{ username: string }>()
  const [filter, setFilter] = useState('')

  const { data: user, isLoading: userLoading, error: userError } = useQuery<User>({
    queryKey: ['user-profile', username],
    queryFn: () => getUserProfile(username),
    enabled: !!username,
    retry: 1,
  })

  const { data: repos = [], isLoading: reposLoading } = useQuery<Repository[]>({
    queryKey: ['user-repos', username],
    queryFn: () => getReposByUser(username),
    enabled: !!username,
  })

  const filtered = repos.filter(
    (r) =>
      r.name.toLowerCase().includes(filter.toLowerCase()) ||
      r.description?.toLowerCase().includes(filter.toLowerCase())
  )

  if (userLoading) {
    return (
      <div className="flex items-center justify-center min-h-[300px]">
        <Spinner size="md" className="text-secondary" />
      </div>
    )
  }

  if (userError || !user) {
    return (
      <div className="max-w-4xl mx-auto px-4 py-8">
        <div className="border border-accent-red p-4 text-accent-red text-sm font-mono">
          User not found.
        </div>
      </div>
    )
  }

  const joinDate = format(new Date(user.created_at), 'MMMM yyyy')

  return (
    <div className="max-w-6xl mx-auto px-4 py-6 flex flex-col md:flex-row gap-8">
      {/* Left — profile card */}
      <aside className="md:w-64 flex-shrink-0">
        <Avatar user={user} size="lg" className="w-full h-auto aspect-square mb-4" />

        <h1 className="text-xl font-semibold text-primary leading-tight">
          {user.display_name || user.username}
        </h1>
        <p className="font-mono text-secondary text-sm mt-0.5 mb-3">
          @{user.username}
        </p>

        {user.bio && (
          <p className="text-sm text-secondary leading-relaxed mb-4">{user.bio}</p>
        )}

        <div className="space-y-2 text-sm text-muted">
          <div className="flex items-center gap-2">
            <Calendar size={14} className="flex-shrink-0" />
            <span>Joined {joinDate}</span>
          </div>
          <div className="flex items-center gap-2">
            <GitBranch size={14} className="flex-shrink-0" />
            <span>{repos.length} {repos.length === 1 ? 'repository' : 'repositories'}</span>
          </div>
        </div>
      </aside>

      {/* Right — repositories */}
      <main className="flex-1 min-w-0">
        {/* Tab bar (single active tab for now) */}
        <div className="flex border-b border-line mb-4">
          <button className="flex items-center gap-1.5 px-4 py-2.5 text-sm border-b-2 border-accent-orange text-primary">
            <GitBranch size={14} />
            Repositories
            <span className="font-mono text-xs bg-elevated border border-line px-1.5 py-0.5">
              {repos.length}
            </span>
          </button>
        </div>

        {/* Filter */}
        <div className="mb-4">
          <input
            type="text"
            placeholder="Find a repository..."
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            className="w-full h-8 bg-base border border-line text-primary placeholder:text-muted font-mono text-xs px-3 outline-none focus:border-accent-blue transition-colors"
          />
        </div>

        {/* List */}
        {reposLoading ? (
          <div className="flex items-center justify-center py-12">
            <Spinner size="md" className="text-secondary" />
          </div>
        ) : filtered.length === 0 ? (
          <EmptyState
            icon={<GitBranch size={28} />}
            title={
              repos.length === 0
                ? `${user.username} has no public repositories yet.`
                : 'No repositories match your filter.'
            }
          />
        ) : (
          <div>
            {filtered.map((repo) => (
              <RepoCard key={repo.id} repo={repo} />
            ))}
          </div>
        )}
      </main>
    </div>
  )
}
