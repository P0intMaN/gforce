import { useEffect, useState } from 'react'
import { useAuthStore } from '../store/auth'
import { getCurrentUser } from '../api/auth'

/**
 * Called ONCE at the app root (App.tsx). Blocks route rendering until
 * the persisted token has been validated with the server.
 *
 * Flow:
 *  1. Read localStorage via Zustand persist (sync or async depending on timing).
 *  2. Call validateToken() which probes GET /user.
 *  3. On success  → refresh user object in store, isReady = true.
 *  4. On failure  → clear store (logout), isReady = true.
 *  5. No token    → isReady = true immediately.
 */
export function useAuthRehydration(): boolean {
  const [isReady, setIsReady] = useState(
    () => useAuthStore.persist.hasHydrated()
  )

  useEffect(() => {
    if (useAuthStore.persist.hasHydrated()) {
      // Hydration already complete (synchronous localStorage) — validate now.
      validateToken().finally(() => setIsReady(true))
      return
    }

    // Wait for async hydration, then validate.
    const unsub = useAuthStore.persist.onFinishHydration(() => {
      validateToken().finally(() => setIsReady(true))
    })

    return unsub
  }, [])

  return isReady
}

async function validateToken(): Promise<void> {
  const { token, logout, login } = useAuthStore.getState()

  if (!token) return // no token — nothing to validate, stay logged out

  try {
    const user = await getCurrentUser()
    // Token is valid — refresh the user object in case profile changed.
    login(token, user)
  } catch {
    // Token invalid or expired — clear everything.
    logout()
  }
}
