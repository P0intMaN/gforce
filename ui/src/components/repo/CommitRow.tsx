import { formatDistanceToNow, format } from 'date-fns'
import { GitCommit } from 'lucide-react'
import { Avatar } from '../ui/Avatar'
import { CopyButton } from '../ui/CopyButton'
import { shortSha, truncate } from '../../lib/utils'
import type { CommitResponse } from '../../types/api'

interface CommitRowProps {
  commit: CommitResponse
}

export function CommitRow({ commit }: CommitRowProps) {
  const relativeTime = formatDistanceToNow(new Date(commit.author.date), { addSuffix: true })
  const absoluteTime = format(new Date(commit.author.date), 'PPpp')
  const firstLine = commit.message.split('\n')[0]

  return (
    <div className="flex items-start gap-3 py-3 px-4 border-b border-line-muted last:border-b-0 hover:bg-elevated transition-colors group">
      <Avatar username={commit.author.name} size="xs" className="mt-0.5 flex-shrink-0" />

      <div className="flex-1 min-w-0">
        <p className="text-sm text-primary font-medium leading-snug mb-0.5">
          {truncate(firstLine, 100)}
        </p>
        <div className="flex items-center gap-2 text-xs text-muted">
          <span className="text-secondary">{commit.author.name}</span>
          <span>·</span>
          <span title={absoluteTime}>{relativeTime}</span>
        </div>
      </div>

      {/* SHA */}
      <div className="flex items-center gap-1.5 flex-shrink-0 opacity-0 group-hover:opacity-100 transition-opacity">
        <span className="font-mono text-xs text-secondary border border-line px-1.5 py-0.5">
          {shortSha(commit.sha)}
        </span>
        <CopyButton text={commit.sha} iconOnly />
      </div>

      <GitCommit size={14} className="text-muted flex-shrink-0 mt-0.5" />
    </div>
  )
}
