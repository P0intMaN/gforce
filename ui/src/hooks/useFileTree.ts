import { useQuery } from '@tanstack/react-query'
import { getTree, getBlob } from '../api/git'

export function useFileTree(owner: string, repo: string, ref: string, path?: string) {
  return useQuery({
    queryKey: ['tree', owner, repo, ref, path ?? ''],
    queryFn: () => getTree(owner, repo, ref, path),
    enabled: !!ref,
    staleTime: 30_000,
  })
}

export function useBlob(owner: string, repo: string, ref: string, path: string) {
  return useQuery({
    queryKey: ['blob', owner, repo, ref, path],
    queryFn: () => getBlob(owner, repo, ref, path),
    enabled: !!path,
    staleTime: 60_000,
  })
}
