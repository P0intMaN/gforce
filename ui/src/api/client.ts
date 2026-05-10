import axios from 'axios'
import { useAuthStore } from '../store/auth'

export const apiClient = axios.create({
  baseURL: '/api/v1',
  headers: { 'Content-Type': 'application/json' },
  timeout: 15_000,
})

// Attach Bearer token from Zustand store on every request.
apiClient.interceptors.request.use((config) => {
  const token = useAuthStore.getState().token
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// On 401, clear auth state and redirect to login.
apiClient.interceptors.response.use(
  (res) => res,
  (error) => {
    if (axios.isAxiosError(error) && error.response?.status === 401) {
      useAuthStore.getState().logout()
      // Hard redirect — React Router may not be available here.
      if (!window.location.pathname.startsWith('/login')) {
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
