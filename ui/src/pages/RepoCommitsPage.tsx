import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { GitCommit } from 'lucide-react'
import { useCommits } from '../hooks/useRepo'
import { CommitList } from '../components/repo/CommitList'
import { BranchSelector } from '../components/repo/BranchSelector'

export function RepoCommitsPage() {
  const { owner = '', repo = '', ref = 'main' } = useParams<{
    owner: string
    repo: string
    ref: string
  }>()
  const [branch, setBranch] = useState(ref)

  const { data: commits = [], isLoading, error } = useCommits(owner, repo, branch)

  return (
    <div className="max-w-4xl mx-auto px-4 py-4">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <Link to={`/${owner}/${repo}`} className="text-sm text-accent-blue hover:underline no-underline">
            {owner}/{repo}
          </Link>
          <span className="text-muted text-sm">/</span>
          <span className="flex items-center gap-1.5 text-sm text-primary">
            <GitCommit size={14} />
            Commits
          </span>
        </div>
        <BranchSelector
          owner={owner}
          repo={repo}
          currentBranch={branch}
          onSelect={setBranch}
        />
      </div>

      <CommitList
        commits={commits}
        loading={isLoading}
        error={error ? 'Failed to load commits. The repository may be empty.' : null}
      />
    </div>
  )
}
