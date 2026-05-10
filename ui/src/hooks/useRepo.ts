import { useQuery } from '@tanstack/react-query'
import { getRepo, getMyRepos, getReposByUser } from '../api/repos'
import { getCommits, getBranches } from '../api/git'

export function useRepo(owner: string, repo: string) {
  return useQuery({
    queryKey: ['repo', owner, repo],
    queryFn: () => getRepo(owner, repo),
    retry: 1,
  })
}

export function useMyRepos() {
  return useQuery({
    queryKey: ['my-repos'],
    queryFn: () => getMyRepos(),
  })
}

export function useUserRepos(username: string) {
  return useQuery({
    queryKey: ['user-repos', username],
    queryFn: () => getReposByUser(username),
    enabled: !!username,
  })
}

export function useCommits(owner: string, repo: string, ref: string) {
  return useQuery({
    queryKey: ['commits', owner, repo, ref],
    queryFn: () => getCommits(owner, repo, ref),
    enabled: !!ref,
  })
}

export function useBranches(owner: string, repo: string) {
  return useQuery({
    queryKey: ['branches', owner, repo],
    queryFn: () => getBranches(owner, repo),
  })
}
