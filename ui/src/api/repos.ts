import { apiClient, unwrap } from './client'
import type { Repository, CreateRepoInput } from '../types/api'

export async function getMyRepos(limit = 30, offset = 0): Promise<Repository[]> {
  return unwrap(
    await apiClient.get<{ data: Repository[] }>('/user/repos', {
      params: { limit, offset },
    })
  ) ?? []
}

export async function getReposByUser(
  username: string,
  limit = 30,
  offset = 0
): Promise<Repository[]> {
  return unwrap(
    await apiClient.get<{ data: Repository[] }>(`/users/${username}/repos`, {
      params: { limit, offset },
    })
  ) ?? []
}

export async function getRepo(owner: string, repo: string): Promise<Repository> {
  return unwrap(
    await apiClient.get<{ data: Repository }>(`/repos/${owner}/${repo}`)
  )
}

export async function createRepo(input: CreateRepoInput): Promise<Repository> {
  return unwrap(
    await apiClient.post<{ data: Repository }>('/user/repos', input)
  )
}

export async function deleteRepo(owner: string, repo: string): Promise<void> {
  await apiClient.delete(`/repos/${owner}/${repo}`)
}

export async function getPublicRepos(limit = 30, offset = 0): Promise<Repository[]> {
  return unwrap(
    await apiClient.get<{ data: Repository[] }>('/explore/repos', {
      params: { limit, offset },
    })
  ) ?? []
}
