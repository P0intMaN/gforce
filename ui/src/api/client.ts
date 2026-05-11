import axios from 'axios'
import { useAuthStore } from '../store/auth'

export const apiClient = axios.create({
  baseURL: '/api/v1',
  headers: { 'Content-Type': 'application/json' },
  timeout: 15_000,
})

// Read the token fresh on every request — never capture it at interceptor setup time.
apiClient.interceptors.request.use((config) => {
  const token = useAuthStore.getState().token
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// On 401:
// - If this was the /user rehydration probe → don't redirect (caller handles it).
// - Any other 401 → clear session and go to login.
apiClient.interceptors.response.use(
  (res) => res,
  (error) => {
    if (axios.isAxiosError(error) && error.response?.status === 401) {
      const url: string = error.config?.url ?? ''
      const isUserProbe = url === '/user' || url.endsWith('/user')
      if (!isUserProbe && !window.location.pathname.startsWith('/login')) {
        useAuthStore.getState().logout()
        window.location.href = '/login'
      }
    }
    return Promise.reject(error)
  }
)

/** Unwraps the GForce API envelope `{ data: T }` and returns T. */
export function unwrap<T>(response: { data: { data: T } }): T {
  return response.data.data
}
