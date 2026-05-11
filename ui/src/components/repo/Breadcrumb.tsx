import { Link } from 'react-router-dom'
import { ChevronRight } from 'lucide-react'

interface BreadcrumbProps {
  owner: string
  repo: string
  branch: string
  /** File or directory path relative to repo root, e.g. "src/internal/server.go" */
  path: string
  type: 'blob' | 'tree'
}

/**
 * Renders a clickable breadcrumb trail for a file or directory path inside a repo.
 *
 * Example for path="src/internal/server.go", type="blob":
 *   owner / repo / src / internal / server.go
 *   ↑link   ↑link  ↑link  ↑link      (plain — last segment)
 */
export function Breadcrumb({ owner, repo, branch, path, type }: BreadcrumbProps) {
  const parts = path.split('/').filter(Boolean)

  return (
    <nav className="flex items-center gap-1 text-sm flex-wrap" aria-label="breadcrumb">
      {/* owner */}
      <Link
        to={`/${owner}`}
        className="text-accent-blue hover:underline no-underline"
      >
        {owner}
      </Link>

      <ChevronRight size={12} className="text-muted flex-shrink-0" />

      {/* repo root */}
      <Link
        to={`/${owner}/${repo}`}
        className="text-accent-blue hover:underline no-underline"
      >
        {repo}
      </Link>

      {/* path segments */}
      {parts.map((segment, i) => {
        const isLast = i === parts.length - 1
        const segmentPath = parts.slice(0, i + 1).join('/')
        // Intermediate segments always link as tree (directory); last segment
        // uses the provided type (blob for files, tree for directories).
        const linkType = isLast ? type : 'tree'
        const to = `/${owner}/${repo}/${linkType}/${branch}/${segmentPath}`

        return (
          <span key={segmentPath} className="flex items-center gap-1">
            <ChevronRight size={12} className="text-muted flex-shrink-0" />
            {isLast ? (
              <span className="text-primary font-medium">{segment}</span>
            ) : (
              <Link to={to} className="text-accent-blue hover:underline no-underline">
                {segment}
              </Link>
            )}
          </span>
        )
      })}
    </nav>
  )
}
