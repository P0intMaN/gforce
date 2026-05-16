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

/**
 * Fetches the current user profile.
 * Pass `token` explicitly when calling immediately after login —
 * at that point the token is not yet in the Zustand store, so the
 * request interceptor would send no Authorization header.
 */
export async function getCurrentUser(token?: string): Promise<User> {
  return unwrap(
    await apiClient.get<{ data: User }>('/user', {
      headers: token ? { Authorization: `Bearer ${token}` } : undefined,
    })
  )
}

export async function updateProfile(params: {
  display_name?: string
  bio?: string
  avatar_url?: string
}): Promise<User> {
  return unwrap(await apiClient.patch<{ data: User }>('/user', params))
}
