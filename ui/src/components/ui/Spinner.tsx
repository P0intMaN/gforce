import { cn } from '../../lib/utils'

interface SpinnerProps {
  size?: 'xs' | 'sm' | 'md' | 'lg'
  className?: string
}

const sizes = {
  xs: 'w-3 h-3 border',
  sm: 'w-4 h-4 border',
  md: 'w-6 h-6 border-2',
  lg: 'w-8 h-8 border-2',
}

export function Spinner({ size = 'md', className }: SpinnerProps) {
  return (
    <div
      className={cn(
        'rounded-full border-current border-t-transparent animate-spin',
        sizes[size],
        className
      )}
      aria-label="Loading"
    />
  )
}

export function FullPageSpinner() {
  return (
    <div className="flex items-center justify-center min-h-[200px]">
      <Spinner size="md" className="text-secondary" />
    </div>
  )
}
