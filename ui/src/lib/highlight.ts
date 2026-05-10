import { createHighlighter, type Highlighter } from 'shiki'

let _highlighter: Highlighter | null = null
let _loading: Promise<Highlighter> | null = null

const SUPPORTED_LANGS = [
  'go', 'typescript', 'tsx', 'javascript', 'jsx', 'python', 'rust',
  'sql', 'yaml', 'json', 'bash', 'sh', 'markdown', 'html', 'css',
  'toml', 'dockerfile', 'ruby', 'java', 'kotlin', 'cpp', 'c', 'csharp',
  'swift', 'protobuf', 'text',
] as const

/** Returns the shiki Highlighter singleton. Thread-safe (Promise-cached). */
export async function getHighlighter(): Promise<Highlighter> {
  if (_highlighter) return _highlighter
  if (_loading) return _loading

  _loading = createHighlighter({
    themes: ['github-dark'],
    langs: [...SUPPORTED_LANGS],
  }).then((h) => {
    _highlighter = h
    return h
  })

  return _loading
}

/** Maps a filename extension to a shiki language id. */
export function fileExtToLang(filename: string): string {
  const ext = filename.split('.').pop()?.toLowerCase() ?? ''
  const map: Record<string, string> = {
    go: 'go', ts: 'typescript', tsx: 'tsx', js: 'javascript',
    jsx: 'jsx', py: 'python', rs: 'rust', sql: 'sql',
    yaml: 'yaml', yml: 'yaml', json: 'json', sh: 'bash',
    bash: 'bash', md: 'markdown', html: 'html', css: 'css',
    toml: 'toml', dockerfile: 'dockerfile', rb: 'ruby',
    java: 'java', kt: 'kotlin', cpp: 'cpp', c: 'c',
    cs: 'csharp', swift: 'swift', proto: 'protobuf',
  }
  // Handle Dockerfile (no extension)
  if (filename.toLowerCase() === 'dockerfile') return 'dockerfile'
  return map[ext] ?? 'text'
}

/** Highlight source code to HTML. Falls back to escaped plain text on error. */
export async function highlight(code: string, filename: string): Promise<string> {
  try {
    const hl = await getHighlighter()
    const lang = fileExtToLang(filename)
    return hl.codeToHtml(code, { lang, theme: 'github-dark' })
  } catch {
    // Fallback: wrap in pre/code with HTML escaping
    const escaped = code
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
    return `<pre><code>${escaped}</code></pre>`
  }
}
