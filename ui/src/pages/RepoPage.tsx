import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { Star, GitFork, Lock, Globe, GitBranch } from 'lucide-react'
import ReactMarkdown from 'react-markdown'
import { useQuery } from '@tanstack/react-query'
import { useRepo } from '../hooks/useRepo'
import { useFileTree } from '../hooks/useFileTree'
import { getBlob, getCommits } from '../api/git'
import { FileTree } from '../components/repo/FileTree'
import { BranchSelector } from '../components/repo/BranchSelector'
import { CloneButton } from '../components/repo/CloneButton'
import { Badge } from '../components/ui/Badge'
import { Spinner } from '../components/ui/Spinner'
import { CommitRow } from '../components/repo/CommitRow'
import type { CommitResponse, BlobResponse } from '../types/api'

export function RepoPage() {
  // '*' is populated when navigating via /:owner/:repo/tree/:ref/* (subfolder clicks)
  const { owner = '', repo: repoName = '', ref: routeRef, '*': treePath = '' } = useParams<{
    owner: string
    repo: string
    ref?: string
    '*'?: string
  }>()
  const [branch, setBranch] = useState<string | null>(null)

  const { data: repo, isLoading: repoLoading, error: repoError } = useRepo(owner, repoName)

  const currentBranch = branch ?? routeRef ?? repo?.default_branch ?? 'main'

  // useFileTree and commit queries — errors are expected for empty repos
  // Pass treePath so subfolder navigation loads the correct directory listing
  const { data: tree, isLoading: treeLoading } = useFileTree(
    owner, repoName, currentBranch, treePath || undefined
  )

  const { data: commits } = useQuery<CommitResponse[]>({
    queryKey: ['commits', owner, repoName, currentBranch, 1],
    queryFn: () => getCommits(owner, repoName, currentBranch, 1, 0),
    enabled: !!repo,
    retry: false,           // empty repos return 404 — don't retry
    throwOnError: false,
  })

  const readmeEntry = tree?.entries.find(
    (e) => e.type === 'blob' && /^readme(\.(md|txt|rst))?$/i.test(e.name)
  )
  const { data: readme } = useQuery<BlobResponse>({
    queryKey: ['readme', owner, repoName, currentBranch],
    queryFn: () => getBlob(owner, repoName, currentBranch, readmeEntry!.path),
    enabled: !!readmeEntry,
    retry: false,
    throwOnError: false,
  })

  if (repoLoading) {
    return (
      <div className="flex items-center justify-center min-h-[300px]">
        <Spinner size="md" className="text-secondary" />
      </div>
    )
  }

  if (repoError || !repo) {
    return (
      <div className="max-w-4xl mx-auto px-4 py-8">
        <div className="border border-accent-red p-4 text-accent-red text-sm font-mono">
          Repository not found or access denied.
        </div>
      </div>
    )
  }

  const lastCommit = commits?.[0]
  const isEmpty = !treeLoading && (!tree?.entries || tree.entries.length === 0)

  return (
    <div className="max-w-6xl mx-auto px-4 py-4">
      {/* Header */}
      <div className="mb-4">
        <div className="flex items-center gap-2 mb-1">
          <Link to={`/${owner}`} className="text-sm text-accent-blue hover:underline no-underline">
            {owner}
          </Link>
          <span className="text-secondary">/</span>
          <span className="text-sm text-primary font-semibold">{repoName}</span>
          <Badge variant={repo.is_private ? 'muted' : 'default'}>
            {repo.is_private ? (
              <span className="flex items-center gap-1"><Lock size={9} /> Private</span>
            ) : (
              <span className="flex items-center gap-1"><Globe size={9} /> Public</span>
            )}
          </Badge>
        </div>
        {repo.description && (
          <p className="text-sm text-secondary">{repo.description}</p>
        )}
        <div className="flex items-center gap-4 mt-2 text-xs text-muted">
          <span className="flex items-center gap-1"><Star size={12} /> {repo.star_count}</span>
          <span className="flex items-center gap-1"><GitFork size={12} /> {repo.fork_count}</span>
          <span className="flex items-center gap-1"><GitBranch size={12} /> {repo.default_branch}</span>
        </div>
      </div>

      <div className="border-b border-line mb-4" />

      {/* Toolbar */}
      <div className="flex items-center justify-between mb-3 flex-wrap gap-2">
        <BranchSelector
          owner={owner}
          repo={repoName}
          currentBranch={currentBranch}
          onSelect={setBranch}
        />
        <div className="flex items-center gap-2">
          <Link
            to={`/${owner}/${repoName}/commits/${currentBranch}`}
            className="flex items-center gap-1.5 h-7 px-3 text-xs border border-line text-secondary hover:text-primary hover:border-[#3a3d45] transition-colors no-underline"
          >
            <GitBranch size={12} />
            {commits?.length ?? 0} commit{commits?.length !== 1 ? 's' : ''}
          </Link>
          <CloneButton cloneUrl={repo.clone_url} />
        </div>
      </div>

      {/* Empty repo state */}
      {isEmpty && (
        <div className="border border-dashed border-line p-10 text-center mt-4">
          <p className="font-mono text-sm text-secondary mb-1">This repository is empty.</p>
          <p className="text-xs text-muted mb-4">
            Push an existing repository, or clone it and add files.
          </p>
          <div className="terminal-box max-w-md mx-auto text-left">
            <div className="terminal-title-bar">
              <span className="terminal-dot" style={{ background: '#f85149' }} />
              <span className="terminal-dot" style={{ background: '#d29922' }} />
              <span className="terminal-dot" style={{ background: '#3fb950' }} />
            </div>
            <div className="p-4 space-y-2 text-xs font-mono">
              <p><span className="text-accent-green">$</span> <span className="text-secondary">git clone {repo.clone_url}</span></p>
              <p><span className="text-accent-green">$</span> <span className="text-secondary">cd {repoName}</span></p>
              <p><span className="text-accent-green">$</span> <span className="text-secondary">echo "# {repoName}" &gt; README.md</span></p>
              <p><span className="text-accent-green">$</span> <span className="text-secondary">git add . && git commit -m "Initial commit"</span></p>
              <p><span className="text-accent-green">$</span> <span className="text-secondary">git push origin {repo.default_branch}</span></p>
            </div>
          </div>
        </div>
      )}

      {/* File tree + README */}
      {!isEmpty && (
        <div className="grid grid-cols-1 lg:grid-cols-5 gap-4">
          <div className="lg:col-span-2 border border-line">
            {lastCommit && (
              <div className="border-b border-line bg-elevated">
                <CommitRow commit={lastCommit} />
              </div>
            )}
            {treeLoading ? (
              <div className="flex items-center justify-center py-8">
                <Spinner size="sm" className="text-secondary" />
              </div>
            ) : (
              <FileTree
                owner={owner}
                repo={repoName}
                gitRef={currentBranch}
                entries={tree?.entries ?? []}
              />
            )}
          </div>

          <div className="lg:col-span-3">
            {readme ? (
              <div className="border border-line">
                <div className="px-4 py-2.5 border-b border-line bg-elevated">
                  <span className="text-xs font-mono text-secondary">README.md</span>
                </div>
                <div className="p-5 markdown-body">
                  <ReactMarkdown>{atob(readme.content)}</ReactMarkdown>
                </div>
              </div>
            ) : (
              <div className="border border-dashed border-line p-8 text-center">
                <p className="text-sm text-secondary font-mono">No README.md found</p>
                <p className="text-xs text-muted mt-1">
                  Add a README.md to help people understand your project.
                </p>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
