import { apiClient, unwrap } from './client'
import type { User, TokenResponse } from '../types/api'

export async function login(login: string, password: string): Promise<TokenResponse> {
  return unwrap(await apiClient.post<{ data: TokenResponse }>('/auth/login', { login, password }))
}

export async function register(
  username: string,
  email: string,
  password: string
): Promise<User> {
  return unwrap(await apiClient.post<{ data: User }>('/auth/register', { username, email, password }))
}

export async function getCurrentUser(): Promise<User> {
  return unwrap(await apiClient.get<{ data: User }>('/user'))
}

export async function updateProfile(params: {
  display_name?: string
  bio?: string
  avatar_url?: string
}): Promise<User> {
  return unwrap(await apiClient.patch<{ data: User }>('/user', params))
}
