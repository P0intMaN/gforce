import { useEffect, useState } from 'react'
import { useAuthStore } from '../store/auth'
import { getCurrentUser } from '../api/auth'

/**
 * Tracks whether Zustand persist has finished reading from localStorage.
 * Uses useAuthStore.persist.hasHydrated() — the official Zustand v4 API.
 */
export function useIsRehydrated(): boolean {
  const [hydrated, setHydrated] = useState<boolean>(
    () => useAuthStore.persist.hasHydrated()
  )
  useEffect(() => {
    const unsub = useAuthStore.persist.onFinishHydration(() => setHydrated(true))
    if (useAuthStore.persist.hasHydrated()) setHydrated(true)
    return unsub
  }, [])
  return hydrated
}

/** Validates the persisted token against the server on every app load. */
export function useAuthRehydration() {
  const { token, user, setUser, logout } = useAuthStore()
  const hydrated = useIsRehydrated()

  useEffect(() => {
    if (!hydrated || !token || user) return
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
