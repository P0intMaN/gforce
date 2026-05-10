import { useState, useRef, useEffect } from 'react'
import { Download } from 'lucide-react'
import { CopyButton } from '../ui/CopyButton'

interface CloneButtonProps {
  cloneUrl: string
}

export function CloneButton({ cloneUrl }: CloneButtonProps) {
  const [open, setOpen] = useState(false)
  const [tab, setTab] = useState<'https' | 'ssh'>('https')
  const ref = useRef<HTMLDivElement>(null)

  // Derive SSH URL from HTTPS URL
  const sshUrl = cloneUrl
    .replace(/^https?:\/\//, 'git@')
    .replace(/\/([^/]+)\//, ':$1/')

  const displayUrl = tab === 'https' ? cloneUrl : sshUrl

  useEffect(() => {
    function handler(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [])

  return (
    <div ref={ref} className="relative">
      <button
        onClick={() => setOpen(!open)}
        className="flex items-center gap-1.5 h-7 px-3 text-sm bg-accent-green text-base hover:bg-[#56d364] font-medium transition-colors"
      >
        <Download size={13} />
        <span>Code</span>
        <span className="border-l border-[rgba(0,0,0,0.2)] pl-1.5 ml-0.5">▾</span>
      </button>

      {open && (
        <div
          className="absolute right-0 top-full mt-1 w-80 bg-overlay border border-line z-50 animate-slide-down"
          style={{ boxShadow: '0 8px 24px rgba(0,0,0,0.5)' }}
        >
          <div className="p-3">
            <p className="text-xs font-mono text-secondary mb-3 font-semibold uppercase tracking-wide">
              Clone
            </p>

            {/* Protocol tabs */}
            <div className="flex border border-line mb-3">
              {(['https', 'ssh'] as const).map((t) => (
                <button
                  key={t}
                  onClick={() => setTab(t)}
                  className={`flex-1 py-1 text-xs font-mono transition-colors ${
                    tab === t
                      ? 'bg-elevated text-primary'
                      : 'text-secondary hover:text-primary'
                  }`}
                >
                  {t.toUpperCase()}
                </button>
              ))}
            </div>

            {/* URL field */}
            <div className="flex items-center gap-2 border border-line bg-base px-3 py-2">
              <input
                readOnly
                value={displayUrl}
                className="flex-1 bg-transparent text-xs font-mono text-secondary outline-none min-w-0"
              />
              <CopyButton text={displayUrl} iconOnly />
            </div>

            {/* CLI hint */}
            <div className="mt-3 p-2 bg-base border border-line-muted">
              <p className="text-2xs font-mono text-muted">
                <span className="text-accent-green">$</span>{' '}
                <span className="text-secondary">git clone {displayUrl}</span>
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
