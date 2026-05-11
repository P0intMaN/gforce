import axios from 'axios'
import { useAuthStore } from '../store/auth'

export const apiClient = axios.create({
  baseURL: '/api/v1',
  headers: { 'Content-Type': 'application/json' },
  timeout: 15_000,
})

// Read the token fresh on every request — never capture it at setup time.
apiClient.interceptors.request.use((config) => {
  const token = useAuthStore.getState().token
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// On 401:
// - /user endpoint is the token-validation probe — caller handles it, no redirect.
// - All other 401s → clear session and hard-redirect to /login.
apiClient.interceptors.response.use(
  (res) => res,
  (error) => {
    if (axios.isAxiosError(error) && error.response?.status === 401) {
      // config.url is relative to baseURL, so it's "/user" not "/api/v1/user"
      const url: string = error.config?.url ?? ''
      const isUserEndpoint = url === '/user' || url.endsWith('/user')
      if (!isUserEndpoint && !window.location.pathname.startsWith('/login')) {
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
