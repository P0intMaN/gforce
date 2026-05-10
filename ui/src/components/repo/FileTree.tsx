import { useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { Folder, FolderOpen, FileText, ChevronRight } from 'lucide-react'
import { Spinner } from '../ui/Spinner'
import { getTree } from '../../api/git'
import type { TreeEntry } from '../../types/api'

interface FileTreeProps {
  owner: string
  repo: string
  ref: string
  entries: TreeEntry[]
  basePath?: string
}

interface FileNodeProps {
  entry: TreeEntry
  owner: string
  repo: string
  ref: string
  depth: number
}

function FileNode({ entry, owner, repo, ref, depth }: FileNodeProps) {
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const [children, setChildren] = useState<TreeEntry[] | null>(null)
  const [loading, setLoading] = useState(false)

  const handleClick = useCallback(async () => {
    if (entry.type === 'blob') {
      navigate(`/${owner}/${repo}/blob/${ref}/${entry.path}`)
      return
    }
    if (!open && children === null) {
      setLoading(true)
      try {
        const tree = await getTree(owner, repo, ref, entry.path)
        setChildren(tree.entries ?? [])
      } finally {
        setLoading(false)
      }
    }
    setOpen(!open)
  }, [entry, open, children, owner, repo, ref, navigate])

  const isDir = entry.type === 'tree'
  const indent = depth * 12

  return (
    <div>
      <button
        onClick={handleClick}
        className="w-full flex items-center gap-2 px-3 py-1 text-sm hover:bg-elevated transition-colors text-left group"
        style={{ paddingLeft: `${12 + indent}px` }}
      >
        {isDir ? (
          <>
            <ChevronRight
              size={12}
              className={`text-muted transition-transform duration-150 ${open ? 'rotate-90' : ''}`}
            />
            {open ? (
              <FolderOpen size={14} className="text-accent-orange flex-shrink-0" />
            ) : (
              <Folder size={14} className="text-accent-orange flex-shrink-0" />
            )}
          </>
        ) : (
          <>
            <span className="w-3" />
            <FileText size={14} className="text-secondary flex-shrink-0" />
          </>
        )}
        <span className={isDir ? 'text-primary' : 'text-primary'}>
          {entry.name}
        </span>
        {loading && <Spinner size="xs" className="ml-auto text-secondary" />}
      </button>

      {isDir && open && children && (
        <div>
          {children.map((child) => (
            <FileNode
              key={child.path}
              entry={child}
              owner={owner}
              repo={repo}
              ref={ref}
              depth={depth + 1}
            />
          ))}
          {children.length === 0 && (
            <p className="text-xs text-muted px-4 py-1" style={{ paddingLeft: `${24 + indent}px` }}>
              Empty directory
            </p>
          )}
        </div>
      )}
    </div>
  )
}

export function FileTree({ owner, repo, ref, entries }: FileTreeProps) {
  if (entries.length === 0) {
    return (
      <div className="py-8 text-center text-sm text-muted">
        No files yet
      </div>
    )
  }

  // Sort: directories first, then files
  const sorted = [...entries].sort((a, b) => {
    if (a.type === b.type) return a.name.localeCompare(b.name)
    return a.type === 'tree' ? -1 : 1
  })

  return (
    <div className="text-sm">
      {sorted.map((entry) => (
        <FileNode
          key={entry.path}
          entry={entry}
          owner={owner}
          repo={repo}
          ref={ref}
          depth={0}
        />
      ))}
    </div>
  )
}
