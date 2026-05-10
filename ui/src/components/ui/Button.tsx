import { forwardRef } from 'react'
import { cn } from '../../lib/utils'

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger' | 'outline'
  size?: 'sm' | 'md' | 'lg'
  loading?: boolean
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ variant = 'secondary', size = 'md', loading, className, children, disabled, ...props }, ref) => {
    const base =
      'inline-flex items-center justify-center gap-2 font-sans font-medium transition-colors duration-100 disabled:opacity-50 disabled:cursor-not-allowed focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-blue focus-visible:ring-offset-1 focus-visible:ring-offset-base'

    const variants = {
      primary:
        'bg-accent-blue text-base hover:bg-[#79c0ff] border border-transparent',
      secondary:
        'bg-elevated border border-line text-primary hover:border-[#3a3d45] hover:bg-[#1e2128]',
      ghost:
        'bg-transparent border border-transparent text-secondary hover:text-primary hover:bg-elevated',
      danger:
        'bg-accent-red text-white hover:bg-[#ff6b6b] border border-transparent',
      outline:
        'bg-transparent border border-accent-blue text-accent-blue hover:bg-accent-blue hover:text-base',
    }

    const sizes = {
      sm: 'h-7 px-3 text-xs',
      md: 'h-8 px-4 text-sm',
      lg: 'h-10 px-6 text-sm',
    }

    return (
      <button
        ref={ref}
        disabled={disabled || loading}
        className={cn(base, variants[variant], sizes[size], className)}
        {...props}
      >
        {loading && (
          <span className="w-3.5 h-3.5 border-2 border-current border-t-transparent rounded-full animate-spin" />
        )}
        {children}
      </button>
    )
  }
)
Button.displayName = 'Button'
