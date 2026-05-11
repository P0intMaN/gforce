import { useEffect, useState } from 'react'
import { useAuthStore } from '../store/auth'
import { getCurrentUser } from '../api/auth'

/**
 * Tracks whether Zustand's persist middleware has finished reading
 * the auth token from localStorage.
 *
 * Uses the official Zustand v4 API:
 *   useAuthStore.persist.hasHydrated()     — synchronous boolean
 *   useAuthStore.persist.onFinishHydration — subscribe to completion
 *
 * This avoids the circular-reference trap of calling
 * useAuthStore.setState() from inside onRehydrateStorage while the
 * store variable is still being assigned.
 */
export function useIsRehydrated(): boolean {
  const [hydrated, setHydrated] = useState<boolean>(
    () => useAuthStore.persist.hasHydrated()
  )

  useEffect(() => {
    // Subscribe first, then check — avoids the race where hydration
    // finishes between the useState initializer and this effect running.
    const unsubscribe = useAuthStore.persist.onFinishHydration(() => {
      setHydrated(true)
    })
    if (useAuthStore.persist.hasHydrated()) {
      setHydrated(true)
    }
    return unsubscribe
  }, [])

  return hydrated
}

/**
 * Validates the persisted token against the server on every app load.
 * Must be called inside AuthGate so hydration is guaranteed before it runs.
 */
export function useAuthRehydration() {
  const { token, user, setUser, logout } = useAuthStore()
  const hydrated = useIsRehydrated()

  useEffect(() => {
    if (!hydrated) return
    if (!token) return
    if (user) return

    getCurrentUser()
      .then(setUser)
      .catch(() => {
        logout()
        window.location.href = '/login'
      })
  }, [hydrated, token, user, setUser, logout])
}

export function useAuth() {
  return useAuthStore()
}
