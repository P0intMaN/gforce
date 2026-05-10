import { cn } from '../../lib/utils'
import { initials, avatarColor } from '../../lib/utils'

interface AvatarProps {
  user?: { username: string; avatar_url?: string; display_name?: string } | null
  username?: string
  size?: 'xs' | 'sm' | 'md' | 'lg'
  className?: string
}

const sizeMap = {
  xs: 'w-5 h-5 text-2xs',
  sm: 'w-7 h-7 text-xs',
  md: 'w-9 h-9 text-sm',
  lg: 'w-12 h-12 text-base',
}

export function Avatar({ user, username, size = 'sm', className }: AvatarProps) {
  const name = user?.display_name || user?.username || username || '?'
  const uname = user?.username || username || '?'
  const avatarUrl = user?.avatar_url

  const sz = sizeMap[size]
  const bgColor = avatarColor(uname)

  if (avatarUrl) {
    return (
      <img
        src={avatarUrl}
        alt={name}
        className={cn(sz, 'rounded-none object-cover flex-shrink-0', className)}
      />
    )
  }

  return (
    <div
      className={cn(
        sz,
        'flex items-center justify-center flex-shrink-0 font-mono font-semibold text-base select-none',
        className
      )}
      style={{ backgroundColor: bgColor + '33', color: bgColor, border: `1px solid ${bgColor}55` }}
      title={name}
    >
      {initials(name)}
    </div>
  )
}
