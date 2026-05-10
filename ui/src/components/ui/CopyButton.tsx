import { useState } from 'react'
import { Copy, Check } from 'lucide-react'
import { cn } from '../../lib/utils'

interface CopyButtonProps {
  text: string
  className?: string
  iconOnly?: boolean
}

export function CopyButton({ text, className, iconOnly = false }: CopyButtonProps) {
  const [copied, setCopied] = useState(false)

  async function handleCopy() {
    await navigator.clipboard.writeText(text)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <button
      onClick={handleCopy}
      className={cn(
        'inline-flex items-center gap-1.5 text-xs transition-colors duration-150',
        'text-secondary hover:text-primary',
        'focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-blue',
        className
      )}
      title="Copy to clipboard"
    >
      {copied ? (
        <>
          <Check size={13} className="text-accent-green" />
          {!iconOnly && <span className="text-accent-green">Copied!</span>}
        </>
      ) : (
        <>
          <Copy size={13} />
          {!iconOnly && <span>Copy</span>}
        </>
      )}
    </button>
  )
}
