export interface User {
  id: string
  username: string
  email: string
  display_name: string
  avatar_url: string
  bio: string
  is_admin: boolean
  created_at: string
}

export interface Repository {
  id: string
  name: string
  full_name: string
  description: string
  is_private: boolean
  default_branch: string
  clone_url: string
  star_count: number
  fork_count: number
  owner: User
  created_at: string
  updated_at: string
}

export interface TreeEntry {
  name: string
  path: string
  type: 'blob' | 'tree'
  size: number
  sha: string
  mode: string
}

export interface TreeResponse {
  sha: string
  path: string
  entries: TreeEntry[]
}

export interface BlobResponse {
  path: string
  content: string
  encoding: string
  size: number
  sha: string
}

export interface CommitAuthor {
  name: string
  email: string
  date: string
}

export interface CommitResponse {
  sha: string
  message: string
  author: CommitAuthor
  committer: CommitAuthor
  parents: string[]
  html_url: string
}

export interface BranchResponse {
  name: string
  commit_sha: string
  is_default: boolean
}

export interface SSHKey {
  id: string
  title: string
  fingerprint: string
  last_used_at: string | null
  created_at: string
}

export interface TokenResponse {
  token: string
  expires_at: string
}

export interface ApiResponse<T> {
  data: T
  error?: string
  meta?: {
    total: number
    limit: number
    offset: number
  }
}

export interface CreateRepoInput {
  name: string
  description: string
  is_private: boolean
  init: boolean
  default_branch: string
}

export interface ActivityEvent {
  id: string
  actor_id: string
  event_type: string
  repo_id?: string
  payload: Record<string, unknown>
  created_at: string
}
