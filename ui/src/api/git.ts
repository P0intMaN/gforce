import { apiClient, unwrap } from './client'
import type {
  TreeResponse,
  BlobResponse,
  CommitResponse,
  BranchResponse,
} from '../types/api'

export async function getTree(
  owner: string,
  repo: string,
  ref: string,
  path?: string
): Promise<TreeResponse> {
  return unwrap(
    await apiClient.get<{ data: TreeResponse }>(
      `/repos/${owner}/${repo}/tree/${ref}`,
      path ? { params: { path } } : undefined
    )
  )
}

export async function getBlob(
  owner: string,
  repo: string,
  ref: string,
  path: string
): Promise<BlobResponse> {
  return unwrap(
    await apiClient.get<{ data: BlobResponse }>(
      `/repos/${owner}/${repo}/blob/${ref}/${path}`
    )
  )
}

export async function getCommits(
  owner: string,
  repo: string,
  ref: string,
  limit = 30,
  offset = 0
): Promise<CommitResponse[]> {
  return unwrap(
    await apiClient.get<{ data: CommitResponse[] }>(
      `/repos/${owner}/${repo}/commits/${ref}`,
      { params: { limit, offset } }
    )
  ) ?? []
}

export async function getBranches(
  owner: string,
  repo: string
): Promise<BranchResponse[]> {
  return unwrap(
    await apiClient.get<{ data: BranchResponse[] }>(
      `/repos/${owner}/${repo}/branches`
    )
  ) ?? []
}
