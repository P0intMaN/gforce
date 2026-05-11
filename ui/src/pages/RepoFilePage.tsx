import { useParams } from 'react-router-dom'
import { useBlob } from '../hooks/useFileTree'
import { FileViewer } from '../components/repo/FileViewer'
import { Breadcrumb } from '../components/repo/Breadcrumb'
import { Spinner } from '../components/ui/Spinner'

export function RepoFilePage() {
  const { owner = '', repo = '', ref = 'main', '*': filePath = '' } = useParams()

  const { data: blob, isLoading, error } = useBlob(owner, repo, ref, filePath)

  const pathParts = filePath.split('/').filter(Boolean)
  const filename = pathParts[pathParts.length - 1] ?? filePath

  return (
    <div className="max-w-6xl mx-auto px-4 py-4">
      <div className="mb-4">
        <Breadcrumb owner={owner} repo={repo} branch={ref} path={filePath} type="blob" />
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
          filename={filename}
          size={blob.size}
          sha={blob.sha}
          rawUrl={`/api/v1/repos/${owner}/${repo}/blob/${ref}/${filePath}?raw=true`}
        />
      )}
    </div>
  )
}
