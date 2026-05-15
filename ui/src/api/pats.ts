import { apiClient, unwrap } from './client'
import type { PersonalAccessToken, CreatePATResponse } from '../types/api'

interface CreatePATInput {
  name: string
  scopes: string[]
  expires_at?: string | null
}

export async function createPAT(input: CreatePATInput): Promise<CreatePATResponse> {
  return unwrap(
    await apiClient.post<{ data: CreatePATResponse }>('/user/tokens', input)
  )
}

export async function listPATs(): Promise<PersonalAccessToken[]> {
  return unwrap(
    await apiClient.get<{ data: PersonalAccessToken[] }>('/user/tokens')
  ) ?? []
}

export async function revokePAT(id: string): Promise<void> {
  await apiClient.delete(`/user/tokens/${id}`)
}
