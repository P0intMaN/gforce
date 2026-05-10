import { cn } from '../../lib/utils'
import { Button } from './Button'

interface EmptyStateProps {
  icon?: React.ReactNode
  title: string
  description?: string
  action?: {
    label: string
    onClick: () => void
  }
  className?: string
}

export function EmptyState({ icon, title, description, action, className }: EmptyStateProps) {
  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center py-16 px-8 text-center',
        'border border-dashed border-line',
        className
      )}
    >
      {icon && (
        <div className="text-muted mb-4 opacity-60">{icon}</div>
      )}
      <h3 className="font-mono text-sm text-primary mb-2">{title}</h3>
      {description && (
        <p className="text-sm text-secondary max-w-xs mb-6">{description}</p>
      )}
      {action && (
        <Button variant="primary" size="sm" onClick={action.onClick}>
          {action.label}
        </Button>
      )}
    </div>
  )
}
