import { forwardRef } from 'react'
import { cn } from '../../lib/utils'

interface InputProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'inputPrefix'> {
  label?: string
  error?: string
  inputPrefix?: React.ReactNode
  monospace?: boolean
}

export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ label, error, inputPrefix, monospace, className, id, ...props }, ref) => {
    const inputId = id ?? label?.toLowerCase().replace(/\s+/g, '-')
    return (
      <div className="flex flex-col gap-1.5">
        {label && (
          <label htmlFor={inputId} className="text-xs text-secondary font-sans font-medium">
            {label}
          </label>
        )}
        <div className="relative flex items-center">
          {inputPrefix && (
            <span className="absolute left-3 text-muted pointer-events-none text-sm">
              {inputPrefix}
            </span>
          )}
          <input
            ref={ref}
            id={inputId}
            className={cn(
              'w-full bg-base border border-line text-primary placeholder:text-muted',
              'h-8 px-3 text-sm focus:outline-none focus:ring-0',
              'transition-colors duration-100',
              'disabled:opacity-50 disabled:cursor-not-allowed',
              monospace && 'font-mono',
              inputPrefix && 'pl-8',
              error && 'border-accent-red',
              className
            )}
            {...props}
          />
        </div>
        {error && (
          <p className="text-xs text-accent-red">{error}</p>
        )}
      </div>
    )
  }
)
Input.displayName = 'Input'

interface TextareaProps extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {
  label?: string
  error?: string
}

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ label, error, className, id, ...props }, ref) => {
    const inputId = id ?? label?.toLowerCase().replace(/\s+/g, '-')
    return (
      <div className="flex flex-col gap-1.5">
        {label && (
          <label htmlFor={inputId} className="text-xs text-secondary font-sans font-medium">
            {label}
          </label>
        )}
        <textarea
          ref={ref}
          id={inputId}
          className={cn(
            'w-full bg-base border border-line text-primary placeholder:text-muted',
            'px-3 py-2 text-sm focus:outline-none focus:ring-0 resize-none',
            'transition-colors duration-100',
            error && 'border-accent-red',
            className
          )}
          {...props}
        />
        {error && <p className="text-xs text-accent-red">{error}</p>}
      </div>
    )
  }
)
Textarea.displayName = 'Textarea'
