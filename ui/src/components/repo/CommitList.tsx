import { format } from 'date-fns'
import { CommitRow } from './CommitRow'
import { Spinner } from '../ui/Spinner'
import { EmptyState } from '../ui/EmptyState'
import { GitCommit } from 'lucide-react'
import type { CommitResponse } from '../../types/api'

interface CommitListProps {
  commits: CommitResponse[]
  loading?: boolean
  error?: string | null
}

function groupByDate(commits: CommitResponse[]): Map<string, CommitResponse[]> {
  const groups = new Map<string, CommitResponse[]>()
  for (const commit of commits) {
    const key = format(new Date(commit.author.date), 'MMMM d, yyyy')
    const group = groups.get(key) ?? []
    group.push(commit)
    groups.set(key, group)
  }
  return groups
}

export function CommitList({ commits, loading, error }: CommitListProps) {
  if (loading) return <Spinner size="md" className="text-secondary mx-auto my-12" />
  if (error) return (
    <div className="p-4 border border-accent-red text-accent-red text-sm">{error}</div>
  )
  if (commits.length === 0) return (
    <EmptyState
      icon={<GitCommit size={32} />}
      title="No commits yet"
      description="This repository has no commit history."
    />
  )

  const groups = groupByDate(commits)

  return (
    <div>
      {Array.from(groups.entries()).map(([date, groupCommits]) => (
        <div key={date}>
          <div className="px-4 py-2 bg-elevated border-y border-line-muted">
            <p className="text-xs font-mono text-secondary">
              Commits on <span className="text-primary">{date}</span>
            </p>
          </div>
          <div className="border border-t-0 border-line">
            {groupCommits.map((commit) => (
              <CommitRow key={commit.sha} commit={commit} />
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}
