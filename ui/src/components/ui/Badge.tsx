import { cn } from '../../lib/utils'

interface BadgeProps {
  variant?: 'default' | 'success' | 'warning' | 'error' | 'purple' | 'muted'
  children: React.ReactNode
  className?: string
}

export function Badge({ variant = 'default', children, className }: BadgeProps) {
  const variants = {
    default: 'bg-elevated border-line text-secondary',
    success: 'bg-[rgba(63,185,80,0.15)] border-[rgba(63,185,80,0.4)] text-accent-green',
    warning: 'bg-[rgba(210,153,34,0.15)] border-[rgba(210,153,34,0.4)] text-accent-orange',
    error: 'bg-[rgba(248,81,73,0.15)] border-[rgba(248,81,73,0.4)] text-accent-red',
    purple: 'bg-[rgba(188,140,255,0.15)] border-[rgba(188,140,255,0.4)] text-accent-purple',
    muted: 'bg-transparent border-line-muted text-muted',
  }

  return (
    <span
      className={cn(
        'inline-flex items-center border px-1.5 py-0.5 font-mono text-2xs leading-none',
        variants[variant],
        className
      )}
    >
      {children}
    </span>
  )
}
