import { useEffect, useState } from 'react'
import { highlight } from '../../lib/highlight'
import { Spinner } from '../ui/Spinner'
import { formatBytes } from '../../lib/utils'

interface FileViewerProps {
  content: string   // base64 encoded
  filename: string
  size: number
  sha: string
}

export function FileViewer({ content, filename, size, sha }: FileViewerProps) {
  const [html, setHtml] = useState<string | null>(null)
  const [lineCount, setLineCount] = useState(0)

  useEffect(() => {
    const decoded = atob(content)
    const lines = decoded.split('\n').length
    setLineCount(lines)

    highlight(decoded, filename).then(setHtml)
  }, [content, filename])

  return (
    <div className="border border-line">
      {/* File info bar */}
      <div className="flex items-center justify-between px-4 py-2 bg-elevated border-b border-line">
        <div className="flex items-center gap-4 text-xs text-secondary font-mono">
          <span>{lineCount} lines</span>
          <span>{formatBytes(size)}</span>
          <span className="text-muted">{sha.slice(0, 7)}</span>
        </div>
        <div className="flex items-center gap-3">
          <a
            href={`/api/v1/repos/raw/${sha}`}
            target="_blank"
            rel="noreferrer"
            className="text-xs text-secondary hover:text-primary transition-colors no-underline"
          >
            Raw
          </a>
        </div>
      </div>

      {/* Code */}
      <div className="overflow-x-auto bg-base">
        {html === null ? (
          <div className="flex items-center justify-center py-12">
            <Spinner size="md" className="text-secondary" />
          </div>
        ) : (
          <div
            className="shiki-wrapper text-xs"
            dangerouslySetInnerHTML={{ __html: html }}
          />
        )}
      </div>
    </div>
  )
}
