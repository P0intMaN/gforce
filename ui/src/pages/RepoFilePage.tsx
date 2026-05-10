import { useParams, Link } from 'react-router-dom'
import { ChevronRight } from 'lucide-react'
import { useBlob } from '../hooks/useFileTree'
import { FileViewer } from '../components/repo/FileViewer'
import { Spinner } from '../components/ui/Spinner'

export function RepoFilePage() {
  const { owner = '', repo = '', ref = 'main', '*': filePath = '' } = useParams()

  const { data: blob, isLoading, error } = useBlob(owner, repo, ref, filePath)

  const pathParts = filePath.split('/').filter(Boolean)

  return (
    <div className="max-w-6xl mx-auto px-4 py-4">
      {/* Breadcrumb */}
      <div className="flex items-center gap-1 text-sm mb-4 flex-wrap">
        <Link to={`/${owner}/${repo}`} className="text-accent-blue hover:underline no-underline">
          {repo}
        </Link>
        {pathParts.map((part, i) => {
          const partPath = pathParts.slice(0, i + 1).join('/')
          const isLast = i === pathParts.length - 1
          return (
            <span key={partPath} className="flex items-center gap-1">
              <ChevronRight size={12} className="text-muted" />
              {isLast ? (
                <span className="text-primary font-medium">{part}</span>
              ) : (
                <Link
                  to={`/${owner}/${repo}/tree/${ref}/${partPath}`}
                  className="text-accent-blue hover:underline no-underline"
                >
                  {part}
                </Link>
              )}
            </span>
          )
        })}
      </div>

      {isLoading && (
        <div className="flex items-center justify-center py-12">
          <Spinner size="md" className="text-secondary" />
        </div>
      )}

      {error && (
        <div className="border border-accent-red p-4 text-accent-red text-sm font-mono">
          Failed to load file.
        </div>
      )}

      {blob && (
        <FileViewer
          content={blob.content}
          filename={pathParts[pathParts.length - 1] ?? filePath}
          size={blob.size}
          sha={blob.sha}
        />
      )}
    </div>
  )
}
