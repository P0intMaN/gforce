import { useEffect } from 'react'
import { useAuthStore, useIsRehydrated } from '../store/auth'
import { getCurrentUser } from '../api/auth'

/**
 * Validates the persisted token against the server on every app load.
 * Must be called at the root of the component tree.
 *
 * Flow:
 *  1. Zustand rehydrates token from localStorage (sync, near-instant).
 *  2. isRehydrated flips to true.
 *  3. If token present and user not yet loaded → probe GET /user.
 *  4. Success → populate user, isAuthenticated = true.
 *  5. Failure (401 / network) → logout() + redirect to /login.
 */
export function useAuthRehydration() {
  const { token, user, setUser, logout } = useAuthStore()
  const isRehydrated = useIsRehydrated()

  useEffect(() => {
    if (!isRehydrated) return        // wait for localStorage read to finish
    if (!token) return               // no token → nothing to validate
    if (user) return                 // user already populated (just logged in)

    getCurrentUser()
      .then(setUser)
      .catch(() => {
        logout()
        window.location.href = '/login'
      })
  }, [isRehydrated, token, user, setUser, logout])
}

export function useAuth() {
  return useAuthStore()
}

export { useIsRehydrated }
