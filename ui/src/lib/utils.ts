import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs))
}

export function truncate(str: string, n: number): string {
  return str.length > n ? str.slice(0, n) + '…' : str
}

export function shortSha(sha: string): string {
  return sha.slice(0, 7)
}

/** Derive initials from a display name or username. */
export function initials(name: string): string {
  const parts = name.trim().split(/\s+/)
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase()
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
}

/** Deterministic color for an avatar based on the username. */
const AVATAR_COLORS = [
  '#3fb950', '#58a6ff', '#d29922', '#bc8cff', '#f85149',
  '#79c0ff', '#56d364', '#e3b341', '#ff7b72', '#cae8ff',
]

export function avatarColor(username: string): string {
  let hash = 0
  for (let i = 0; i < username.length; i++) {
    hash = username.charCodeAt(i) + ((hash << 5) - hash)
  }
  return AVATAR_COLORS[Math.abs(hash) % AVATAR_COLORS.length]
}

/** Format bytes to human-readable. */
export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

/** Detect programming language from file extension. */
export function detectLanguage(filename: string): string {
  const ext = filename.split('.').pop()?.toLowerCase() ?? ''
  const map: Record<string, string> = {
    go: 'Go', ts: 'TypeScript', tsx: 'TypeScript', js: 'JavaScript',
    jsx: 'JavaScript', py: 'Python', rs: 'Rust', sql: 'SQL',
    yaml: 'YAML', yml: 'YAML', json: 'JSON', sh: 'Shell',
    bash: 'Shell', md: 'Markdown', html: 'HTML', css: 'CSS',
    toml: 'TOML', dockerfile: 'Dockerfile', rb: 'Ruby',
    java: 'Java', kt: 'Kotlin', cpp: 'C++', c: 'C', cs: 'C#',
    swift: 'Swift', proto: 'Protobuf',
  }
  return map[ext] ?? 'Text'
}

/** Language dot color. */
export function languageColor(lang: string): string {
  const colors: Record<string, string> = {
    Go: '#00ADD8', TypeScript: '#3178C6', JavaScript: '#F7DF1E',
    Python: '#3572A5', Rust: '#DEA584', SQL: '#E38C00',
    YAML: '#CB171E', JSON: '#FFA500', Shell: '#89E051',
    Markdown: '#083fa1', HTML: '#E34C26', CSS: '#563D7C',
    TOML: '#9C4121', Dockerfile: '#384D54', Ruby: '#701516',
    Java: '#B07219', Kotlin: '#A97BFF', 'C++': '#f34b7d',
    C: '#555555', 'C#': '#178600', Swift: '#F05138',
  }
  return colors[lang] ?? '#8b949e'
}
