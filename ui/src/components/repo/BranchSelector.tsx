import { useState, useRef, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { GitBranch, ChevronDown, Check } from 'lucide-react'
import { useQuery } from '@tanstack/react-query'
import { getBranches } from '../../api/git'
import { Spinner } from '../ui/Spinner'

interface BranchSelectorProps {
  owner: string
  repo: string
  currentBranch: string
  onSelect?: (branch: string) => void
}

export function BranchSelector({ owner, repo, currentBranch, onSelect }: BranchSelectorProps) {
  const [open, setOpen] = useState(false)
  const [filter, setFilter] = useState('')
  const ref = useRef<HTMLDivElement>(null)
  const navigate = useNavigate()

  const { data: branches = [], isLoading } = useQuery({
    queryKey: ['branches', owner, repo],
    queryFn: () => getBranches(owner, repo),
    enabled: open,
  })

  useEffect(() => {
    function handler(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
        setFilter('')
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [])

  const filtered = branches.filter((b) =>
    b.name.toLowerCase().includes(filter.toLowerCase())
  )

  function handleSelect(name: string) {
    setOpen(false)
    setFilter('')
    if (onSelect) {
      onSelect(name)
    } else {
      navigate(`/${owner}/${repo}/tree/${name}`)
    }
  }

  return (
    <div ref={ref} className="relative">
      <button
        onClick={() => setOpen(!open)}
        className="flex items-center gap-1.5 h-7 px-3 text-sm bg-elevated border border-line hover:border-[#3a3d45] text-primary transition-colors"
      >
        <GitBranch size={13} className="text-secondary" />
        <span className="font-mono max-w-[120px] truncate">{currentBranch}</span>
        <ChevronDown size={12} className="text-secondary" />
      </button>

      {open && (
        <div
          className="absolute left-0 top-full mt-1 w-64 bg-overlay border border-line z-50 animate-slide-down"
          style={{ boxShadow: '0 8px 24px rgba(0,0,0,0.5)' }}
        >
          <div className="p-2 border-b border-line">
            <input
              autoFocus
              type="text"
              placeholder="Find a branch..."
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              className="w-full bg-base border border-line px-2 py-1 text-xs font-mono text-primary placeholder:text-muted outline-none focus:border-accent-blue"
            />
          </div>
          <div className="max-h-48 overflow-y-auto">
            {isLoading && (
              <div className="flex items-center justify-center py-4">
                <Spinner size="sm" className="text-secondary" />
              </div>
            )}
            {filtered.map((branch) => (
              <button
                key={branch.name}
                onClick={() => handleSelect(branch.name)}
                className="w-full flex items-center gap-2 px-3 py-1.5 text-sm text-primary hover:bg-elevated transition-colors text-left"
              >
                <Check
                  size={12}
                  className={branch.name === currentBranch ? 'text-accent-green' : 'opacity-0'}
                />
                <span className="font-mono text-xs">{branch.name}</span>
                {branch.is_default && (
                  <span className="ml-auto text-2xs text-muted border border-line-muted px-1">default</span>
                )}
              </button>
            ))}
            {!isLoading && filtered.length === 0 && (
              <p className="px-3 py-2 text-xs text-muted">No branches found</p>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
