import { useEffect } from 'react'
import { useAuthStore } from '../store/auth'
import { getCurrentUser } from '../api/auth'

/** Rehydrates the user from the server if a token is present in localStorage. */
export function useAuthRehydration() {
  const { token, user, setUser, logout } = useAuthStore()

  useEffect(() => {
    if (token && !user) {
      getCurrentUser()
        .then(setUser)
        .catch(() => logout())
    }
  }, [token, user, setUser, logout])
}

export function useAuth() {
  return useAuthStore()
}
