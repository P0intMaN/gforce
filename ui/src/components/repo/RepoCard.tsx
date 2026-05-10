import { Link } from 'react-router-dom'
import { Star, GitFork, Lock } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { Badge } from '../ui/Badge'
import { detectLanguage, languageColor } from '../../lib/utils'
import type { Repository } from '../../types/api'

interface RepoCardProps {
  repo: Repository
}

export function RepoCard({ repo }: RepoCardProps) {
  const lang = detectLanguage(repo.name) // fallback; real detection needs tree
  const updatedAt = formatDistanceToNow(new Date(repo.updated_at), { addSuffix: true })

  return (
    <div className="py-4 border-b border-line-muted last:border-b-0">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0 flex-1">
          {/* Name + visibility */}
          <div className="flex items-center gap-2 mb-1 flex-wrap">
            <Link
              to={`/${repo.owner.username}/${repo.name}`}
              className="font-mono text-sm text-accent-blue hover:underline font-medium no-underline"
            >
              {repo.name}
            </Link>
            <Badge variant={repo.is_private ? 'muted' : 'default'}>
              {repo.is_private ? (
                <span className="flex items-center gap-1">
                  <Lock size={9} />
                  Private
                </span>
              ) : (
                'Public'
              )}
            </Badge>
          </div>

          {/* Description */}
          {repo.description && (
            <p className="text-sm text-secondary mb-2 leading-relaxed">{repo.description}</p>
          )}

          {/* Meta row */}
          <div className="flex items-center gap-4 text-xs text-muted">
            <span
              className="flex items-center gap-1.5"
              style={{ color: languageColor(lang) }}
            >
              <span
                className="w-2.5 h-2.5 rounded-full"
                style={{ backgroundColor: languageColor(lang) }}
              />
              {lang}
            </span>
            {repo.star_count > 0 && (
              <span className="flex items-center gap-1 text-muted">
                <Star size={11} />
                {repo.star_count}
              </span>
            )}
            {repo.fork_count > 0 && (
              <span className="flex items-center gap-1 text-muted">
                <GitFork size={11} />
                {repo.fork_count}
              </span>
            )}
            <span>Updated {updatedAt}</span>
          </div>
        </div>
      </div>
    </div>
  )
}
